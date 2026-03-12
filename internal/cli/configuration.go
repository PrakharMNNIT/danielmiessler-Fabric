package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/danielmiessler/fabric/internal/core"
	"github.com/danielmiessler/fabric/internal/plugins/ai"
)

// handleConfigurationCommands handles configuration-related commands.
// Returns (handled, error) where handled indicates if a command was processed and should exit.
func handleConfigurationCommands(currentFlags *Flags, registry *core.PluginRegistry) (handled bool, err error) {
	if currentFlags.UpdatePatterns {
		if err = registry.PatternsLoader.PopulateDB(); err != nil {
			return true, err
		}
		// Save configuration in case any paths were migrated during pattern loading.
		err = registry.SaveEnvFile()
		return true, err
	}

	shouldConfigureModel := currentFlags.ConfigureModel || currentFlags.ChangeDefaultModel
	shouldConfigureProvider := currentFlags.ConfigureProvider != ""
	if !shouldConfigureProvider && !shouldConfigureModel {
		return false, nil
	}

	if registry == nil {
		return true, errors.New("fabric configuration is not initialized")
	}

	selectedVendor := strings.TrimSpace(currentFlags.Vendor)
	if shouldConfigureProvider {
		if err = registry.SetupVendor(currentFlags.ConfigureProvider); err != nil {
			return true, err
		}
		if selectedVendor == "" {
			selectedVendor = currentFlags.ConfigureProvider
		}
	}

	if shouldConfigureModel {
		if err = configureDefaultModel(registry, selectedVendor, currentFlags.Model); err != nil {
			return true, err
		}
	}

	return true, nil
}

func configureDefaultModel(registry *core.PluginRegistry, vendorFilter, requestedModel string) error {
	models, err := registry.GetModels()
	if err != nil {
		return err
	}

	vendorFilter = strings.TrimSpace(vendorFilter)
	if vendorFilter != "" {
		models = models.FilterByVendor(vendorFilter)
		if len(models.GroupsItems) == 0 {
			return fmt.Errorf("vendor %q is not configured or has no available models", vendorFilter)
		}
	}

	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		models.PrintWithVendor(false, registry.Defaults.Vendor.Value, registry.Defaults.Model.Value)

		prompt := "\nEnter model number or exact model name"
		if vendorFilter == "" {
			prompt += " (or Vendor|Model)"
		}
		fmt.Printf("%s: ", prompt)

		reader := bufio.NewReader(os.Stdin)
		selection, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		requestedModel = strings.TrimSpace(selection)
		if requestedModel == "" {
			return errors.New("no model selected")
		}
	}

	vendorName, modelName, err := resolveDefaultModelSelection(models, vendorFilter, requestedModel)
	if err != nil {
		return err
	}

	registry.Defaults.Vendor.Value = vendorName
	registry.Defaults.Model.Value = modelName
	if err = os.Setenv(registry.Defaults.Vendor.EnvVariable, vendorName); err != nil {
		return err
	}
	if err = os.Setenv(registry.Defaults.Model.EnvVariable, modelName); err != nil {
		return err
	}

	fmt.Printf("Default provider: %s\n", vendorName)
	fmt.Printf("Default model: %s\n", modelName)

	return registry.SaveEnvFile()
}

func resolveDefaultModelSelection(models *ai.VendorsModels, vendorFilter, selection string) (vendorName string, modelName string, err error) {
	selection = strings.TrimSpace(selection)
	if selection == "" {
		return "", "", errors.New("no model selected")
	}

	if index, parseErr := strconv.Atoi(selection); parseErr == nil {
		return models.GetGroupAndItemByItemNumber(index)
	}

	if vendor, model, ok := splitVendorModelSelection(selection); ok {
		if vendorFilter != "" && !strings.EqualFold(vendorFilter, vendor) {
			return "", "", fmt.Errorf("selection vendor %q does not match requested vendor %q", vendor, vendorFilter)
		}
		if !modelExistsForVendor(models, vendor, model) {
			return "", "", fmt.Errorf("model %q was not found for vendor %q", model, vendor)
		}
		return vendor, canonicalModelName(models, vendor, model), nil
	}

	vendors := models.FindGroupsByItem(selection)
	if len(vendors) == 0 {
		return "", "", fmt.Errorf("model %q was not found in available models", selection)
	}
	if len(vendors) > 1 {
		return "", "", fmt.Errorf("model %q is available from multiple vendors; use --vendor or Vendor|Model", selection)
	}

	return vendors[0], canonicalModelName(models, vendors[0], selection), nil
}

func splitVendorModelSelection(selection string) (vendor string, model string, ok bool) {
	parts := strings.SplitN(selection, "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	vendor = strings.TrimSpace(parts[0])
	model = strings.TrimSpace(parts[1])
	if vendor == "" || model == "" {
		return "", "", false
	}
	return vendor, model, true
}

func modelExistsForVendor(models *ai.VendorsModels, vendor, model string) bool {
	for _, groupItems := range models.GroupsItems {
		if !strings.EqualFold(groupItems.Group, vendor) {
			continue
		}
		for _, item := range groupItems.Items {
			if strings.EqualFold(item, model) {
				return true
			}
		}
	}
	return false
}

func canonicalModelName(models *ai.VendorsModels, vendor, model string) string {
	for _, groupItems := range models.GroupsItems {
		if !strings.EqualFold(groupItems.Group, vendor) {
			continue
		}
		for _, item := range groupItems.Items {
			if strings.EqualFold(item, model) {
				return item
			}
		}
	}
	return model
}
