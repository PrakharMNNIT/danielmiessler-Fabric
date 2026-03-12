package core

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/danielmiessler/fabric/internal/chat"
	"github.com/danielmiessler/fabric/internal/domain"
	debuglog "github.com/danielmiessler/fabric/internal/log"
	"github.com/danielmiessler/fabric/internal/plugins"
	"github.com/danielmiessler/fabric/internal/plugins/ai"
	"github.com/danielmiessler/fabric/internal/plugins/db/fsdb"
	"github.com/danielmiessler/fabric/internal/tools"
)

func TestSaveEnvFile(t *testing.T) {
	db := fsdb.NewDb(os.TempDir())
	registry, err := NewPluginRegistry(db)
	if err != nil {
		t.Fatalf("NewPluginRegistry() error = %v", err)
	}

	err = registry.SaveEnvFile()
	if err != nil {
		t.Fatalf("SaveEnvFile() error = %v", err)
	}
}

// testVendor implements ai.Vendor for testing purposes
type testVendor struct {
	name       string
	models     []string
	configured bool
	envLine    string
}

func (m *testVendor) GetName() string             { return m.name }
func (m *testVendor) GetSetupDescription() string { return m.name }
func (m *testVendor) IsConfigured() bool          { return m.configured }
func (m *testVendor) Configure() error            { return nil }
func (m *testVendor) Setup() error                { return nil }
func (m *testVendor) SetupFillEnvFileContent(buf *bytes.Buffer) {
	if m.envLine == "" {
		return
	}
	buf.WriteString(m.envLine)
	buf.WriteString("\n")
}
func (m *testVendor) ListModels() ([]string, error) { return m.models, nil }
func (m *testVendor) SendStream([]*chat.ChatCompletionMessage, *domain.ChatOptions, chan domain.StreamUpdate) error {
	return nil
}
func (m *testVendor) Send(context.Context, []*chat.ChatCompletionMessage, *domain.ChatOptions) (string, error) {
	return "", nil
}
func (m *testVendor) NeedsRawMode(string) bool { return false }

func TestGetChatter_WarnsOnAmbiguousModel(t *testing.T) {
	tempDir := t.TempDir()
	db := fsdb.NewDb(tempDir)

	vendorA := &testVendor{name: "VendorA", models: []string{"shared-model"}, configured: true}
	vendorB := &testVendor{name: "VendorB", models: []string{"shared-model"}, configured: true}

	vm := ai.NewVendorsManager()
	vm.AddVendors(vendorA, vendorB)

	defaults := &tools.Defaults{
		PluginBase:         &plugins.PluginBase{},
		Vendor:             &plugins.Setting{Value: "VendorA"},
		Model:              &plugins.SetupQuestion{Setting: &plugins.Setting{Value: "shared-model"}},
		ModelContextLength: &plugins.SetupQuestion{Setting: &plugins.Setting{Value: "0"}},
	}

	registry := &PluginRegistry{Db: db, VendorManager: vm, Defaults: defaults}

	r, w, _ := os.Pipe()
	oldStderr := os.Stderr
	os.Stderr = w
	// Redirect log output to our pipe to capture unconditional log messages
	debuglog.SetOutput(w)
	defer func() {
		os.Stderr = oldStderr
		debuglog.SetOutput(oldStderr)
	}()

	chatter, err := registry.GetChatter("shared-model", 0, "", "", false, false)
	w.Close()
	warning, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("GetChatter() error = %v", err)
	}
	// Verify that one of the valid vendors was selected (don't care which one due to map iteration randomness)
	vendorName := chatter.vendor.GetName()
	if vendorName != "VendorA" && vendorName != "VendorB" {
		t.Fatalf("expected vendor VendorA or VendorB, got %s", vendorName)
	}
	if !strings.Contains(string(warning), "multiple vendors provide model shared-model") {
		t.Fatalf("expected warning about multiple vendors, got %q", string(warning))
	}
}

func TestSetupVendorPersistsSingleConfiguredVendor(t *testing.T) {
	tempDir := t.TempDir()
	db := fsdb.NewDb(tempDir)

	registry, err := NewPluginRegistry(db)
	if err != nil {
		t.Fatalf("NewPluginRegistry() error = %v", err)
	}

	registry.VendorManager = ai.NewVendorsManager()
	registry.VendorsAll = ai.NewVendorsManager()

	vendor := &testVendor{
		name:       "Bedrock",
		models:     []string{"us.anthropic.claude-opus-4-6-v1"},
		configured: true,
		envLine:    "BEDROCK_API_KEY=test-token",
	}
	registry.VendorsAll.AddVendors(vendor)

	if err = registry.SetupVendor("Bedrock"); err != nil {
		t.Fatalf("SetupVendor() error = %v", err)
	}

	if got := registry.VendorManager.FindByName("Bedrock"); got == nil {
		t.Fatalf("expected configured vendor to be registered in VendorManager")
	}
	if len(registry.VendorManager.Vendors) != 1 {
		t.Fatalf("expected one configured vendor, got %d", len(registry.VendorManager.Vendors))
	}

	envContent, readErr := os.ReadFile(registry.Db.EnvFilePath)
	if readErr != nil {
		t.Fatalf("reading env file: %v", readErr)
	}
	if !strings.Contains(string(envContent), "BEDROCK_API_KEY=test-token") {
		t.Fatalf("expected env file to persist configured vendor settings, got %q", string(envContent))
	}
}
