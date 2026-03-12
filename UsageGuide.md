# Fabric CLI Usage Guide

This guide explains how Fabric stores AI provider settings, how to change them safely, how to choose models from the CLI, and what behavior has been verified in this branch.

It is written for the current executable-pipelines branch state as of March 12, 2026.

## What This Guide Covers

- Where provider credentials and defaults are stored
- How Bedrock and OpenRouter configuration works
- How to change provider credentials from the CLI
- How to change the default model from the CLI
- How to use one-off provider/model overrides without changing defaults
- How model discovery works
- What has been tested live versus only covered by automated tests

## Core Idea

Fabric has three separate concepts that are easy to confuse:

1. Configured providers
   A provider is configured when its credentials and required settings have been saved.

2. Default provider and default model
   These are what Fabric uses when you do not pass `-V` or `-m` on the command line.

3. One-off runtime overrides
   These are temporary. They affect only the current command and do not change saved defaults.

Example:

```bash
fabric --pattern summarize <<<"ping"
```

This uses the saved default provider and model.

```bash
fabric -V OpenRouter -m anthropic/claude-opus-4.6 --pattern summarize <<<"ping"
```

This uses OpenRouter only for that one command. It does not change the saved default provider or model.

## Where Fabric Stores Configuration

Fabric persists user configuration in:

```text
~/.config/fabric/.env
```

This file stores:

- default provider
- default model
- provider credentials
- provider-specific options
- pattern/strategy source settings

For the current branch, the important variables are:

### Shared defaults

```text
DEFAULT_VENDOR
DEFAULT_MODEL
```

### Bedrock

```text
BEDROCK_API_KEY
BEDROCK_AWS_REGION
```

### OpenRouter

```text
OPENROUTER_API_KEY
OPENROUTER_API_BASE_URL
OPENROUTER_PROVIDER_ORDER
OPENROUTER_ALLOW_FALLBACKS
```

## Current Local Default Behavior

If your `~/.config/fabric/.env` contains:

```text
DEFAULT_VENDOR=Bedrock
DEFAULT_MODEL=us.anthropic.claude-opus-4-6-v1
```

then this command:

```bash
fabric --pattern summarize <<<"ping"
```

will use Bedrock by default.

If you later set:

```text
DEFAULT_VENDOR=OpenRouter
DEFAULT_MODEL=anthropic/claude-opus-4.6
```

then the same command will use OpenRouter by default.

## How To Configure a Provider from the CLI

Fabric now supports configuring a single provider directly from CLI:

```bash
fabric --configure-provider Bedrock
fabric --configure-provider OpenRouter
```

This is intended for persistent configuration. It writes the settings to `~/.config/fabric/.env`.

### Bedrock CLI setup

Run:

```bash
fabric --configure-provider Bedrock
```

Fabric will walk through the Bedrock questions, including:

- authentication method
- API key or credential path depending on the selected auth mode
- AWS region
- default Bedrock model during setup

After setup, the settings are persisted to `~/.config/fabric/.env`.

### OpenRouter CLI setup

Run:

```bash
fabric --configure-provider OpenRouter
```

Fabric will prompt for:

- OpenRouter API key
- OpenRouter API base URL
- optional provider routing order
- whether OpenRouter may fall back to another provider

For your current Bedrock-via-OpenRouter flow, a useful setup is:

- provider order: `amazon-bedrock`
- allow fallbacks: `false`

That maps to:

```text
OPENROUTER_PROVIDER_ORDER=amazon-bedrock
OPENROUTER_ALLOW_FALLBACKS=false
```

## How To Change a Provider Key Later

There are two supported ways.

### Recommended: use the CLI again

Bedrock:

```bash
fabric --configure-provider Bedrock
```

OpenRouter:

```bash
fabric --configure-provider OpenRouter
```

This is the safest supported flow because Fabric validates the setup path and persists the result consistently.

### Manual: edit the env file

You can directly edit:

```text
~/.config/fabric/.env
```

Examples:

```text
BEDROCK_API_KEY=...
BEDROCK_AWS_REGION=us-east-1
```

```text
OPENROUTER_API_KEY=...
OPENROUTER_API_BASE_URL=https://openrouter.ai/api/v1
```

After editing, open a new shell or rerun Fabric. Fabric reads the env file during startup.

## Why There Are No Raw Secret Flags

Fabric intentionally does not use flags like these:

```bash
fabric --bedrock-api-key ...
fabric --openrouter-api-key ...
```

That is deliberate. Passing secrets directly on the command line is bad practice because secrets can leak through:

- shell history
- process lists
- terminal recording
- logs

The supported safe flows are:

- `fabric --configure-provider <Vendor>`
- direct manual edit of `~/.config/fabric/.env`

## How To Choose a Model from the CLI

Fabric now supports choosing and persisting the default model from the CLI.

### Interactive mode

```bash
fabric --configure-model
```

This fetches available models and then lets you choose one.

You can select by:

- numeric menu index
- exact model name
- `Vendor|Model` when the same model name exists across multiple vendors

Example:

```text
OpenRouter|anthropic/claude-opus-4.6
```

### Direct non-interactive mode

```bash
fabric --configure-model -V OpenRouter -m anthropic/claude-opus-4.6
fabric --configure-model -V Bedrock -m us.anthropic.claude-opus-4-6-v1
```

This persists both:

- `DEFAULT_VENDOR`
- `DEFAULT_MODEL`

## How Model Discovery Works

Fabric supports:

```bash
fabric --listvendors
fabric --listmodels
fabric --listmodels -V Bedrock
fabric --listmodels -V OpenRouter
```

### What these commands do

- `--listvendors`
  Prints all registered vendors known to the current Fabric build.

- `--listmodels`
  Lists models from configured vendors.

- `--listmodels -V <Vendor>`
  Filters model listing to the selected vendor.

### Important note

Model discovery is vendor-dependent:

- some vendors return live model lists
- some vendors may use fallback behavior when live listing fails

In this branch, OpenRouter live model listing was verified directly. Bedrock configuration and model-selection paths were verified, but live Bedrock model discovery can still depend on the auth state and provider response at the time of the call.

## One-Off Runtime Overrides

These do not touch `~/.config/fabric/.env`.

### Override provider and model for one command

```bash
fabric -V OpenRouter -m anthropic/claude-opus-4.6 --pattern summarize <<<"ping"
```

```bash
fabric -V Bedrock -m us.anthropic.claude-opus-4-6-v1 --pattern summarize <<<"ping"
```

### When to use this

Use one-off overrides when:

- you want to compare providers
- you want to try a different model without changing defaults
- you want scripts to pin an exact provider/model pair

## Brand-New Machine or Brand-New Home Directory

A fresh home directory has two separate setup concerns:

1. provider credentials and defaults
2. downloaded patterns and strategies

Configuring a provider does not automatically mean patterns are already installed.

There is also a current bootstrap nuance:

- some commands assume `~/.config/fabric/.env` already exists
- exporting provider variables alone in a completely blank home is not always enough
- the supported way to initialize a blank home is to let Fabric create its config first

In practice, for a brand-new machine or brand-new home, start with one of:

```bash
fabric --configure-provider Bedrock
fabric --configure-provider OpenRouter
fabric --setup
```

After Fabric has created `~/.config/fabric/.env`, provider/model configuration and pattern operations behave normally.

In a brand-new empty home, this can happen:

```bash
fabric --configure-provider OpenRouter
fabric --configure-model -V OpenRouter -m anthropic/claude-opus-4.6
fabric --pattern summarize <<<"ping"
```

The last command can still fail if patterns have not been downloaded yet.

To fully initialize a new home:

```bash
fabric --setup
```

or:

```bash
fabric -U
```

Use `--setup` when starting from scratch. Use `-U` when you mainly need patterns updated or downloaded.

## Recommended Practical Workflows

### Workflow A: Bedrock as your default

Configure Bedrock:

```bash
fabric --configure-provider Bedrock
```

Choose the default model:

```bash
fabric --configure-model -V Bedrock -m us.anthropic.claude-opus-4-6-v1
```

Run normally:

```bash
fabric --pattern summarize <<<"ping"
```

### Workflow B: Bedrock direct by default, OpenRouter only when needed

Keep defaults on Bedrock:

```text
DEFAULT_VENDOR=Bedrock
DEFAULT_MODEL=us.anthropic.claude-opus-4-6-v1
```

Use OpenRouter only as an override:

```bash
fabric -V OpenRouter -m anthropic/claude-opus-4.6 --pattern summarize <<<"ping"
```

### Workflow C: OpenRouter with Bedrock routing

Configure OpenRouter:

```bash
fabric --configure-provider OpenRouter
```

Recommended answers:

- API base URL: leave default
- provider order: `amazon-bedrock`
- allow fallbacks: `false`

Set default model:

```bash
fabric --configure-model -V OpenRouter -m anthropic/claude-opus-4.6
```

Run normally:

```bash
fabric --pattern summarize <<<"ping"
```

## Troubleshooting

### `model not found`

Use:

```bash
fabric --listmodels -V <Vendor>
```

Then choose the exact model name shown there.

### `provider is not configured or has no available models`

Usually means one of these:

- the provider was never configured
- the provider setup failed
- the provider credentials are invalid
- the vendor has no models available through the current config

Fix:

```bash
fabric --configure-provider <Vendor>
```

### `pattern 'summarize' not found`

Your provider config may be valid, but patterns are not installed in that home directory.

Fix:

```bash
fabric --setup
```

or:

```bash
fabric -U
```

### `error loading .env file: ... ~/.config/fabric/.env: no such file or directory`

This means you are in a completely blank home directory and Fabric has not created its config file yet.

Fix with one of:

```bash
fabric --configure-provider Bedrock
fabric --configure-provider OpenRouter
fabric --setup
```

That creates the config file and lets later provider/model commands persist correctly.

### Bedrock auth problems

Check:

- `BEDROCK_API_KEY`
- `BEDROCK_AWS_REGION`
- whether you are using the intended auth mode

This branch also contains a Bedrock fix so explicit Bedrock auth does not get silently overridden by `AWS_PROFILE` / `AWS_DEFAULT_PROFILE`.

### OpenRouter model slug problems

Do not use Bedrock-native model IDs with OpenRouter.

Correct examples:

- Bedrock direct: `us.anthropic.claude-opus-4-6-v1`
- OpenRouter: `anthropic/claude-opus-4.6`

## What Was Verified

This section is intentionally strict. It lists only behavior that was actually verified during this branch work.

### Automated verification

These test suites were run fresh:

```bash
go test ./internal/core ./internal/plugins/ai ./internal/cli -count=1
```

This passed after:

- tightening vendor setup validation
- fixing single-provider persistence into `~/.config/fabric/.env`
- adding regression coverage for incomplete provider setup

### Live runtime verification in the real working tree

These commands were run successfully:

```bash
./fabric --pattern summarize <<<"ping"
./fabric -V OpenRouter -m anthropic/claude-opus-4.6 --pattern summarize <<<"ping"
```

This verifies:

- default-provider runtime works in the real local environment
- one-off OpenRouter override works in the real local environment

### Live provider configuration verification in isolated temp homes

OpenRouter was verified in a blank temporary home with:

- `--configure-provider OpenRouter`
- persisted `OPENROUTER_API_KEY`
- persisted `OPENROUTER_API_BASE_URL`
- persisted `OPENROUTER_PROVIDER_ORDER`
- persisted `OPENROUTER_ALLOW_FALLBACKS`
- `--listmodels -V OpenRouter`
- `--configure-model -V OpenRouter -m anthropic/claude-opus-4.6`
- persisted `DEFAULT_VENDOR`
- persisted `DEFAULT_MODEL`

Bedrock was verified earlier in an isolated temp home for:

- interactive provider setup
- persisted Bedrock provider settings
- direct default-model persistence path

### Additional direct Bedrock verification

This direct Bedrock command was also rerun successfully in the working tree:

```bash
DEFAULT_VENDOR=Bedrock DEFAULT_MODEL=us.anthropic.claude-opus-4-6-v1 ./fabric --pattern summarize <<<"ping"
```

That gives fresh runtime evidence that the direct Bedrock path is working in the current branch.

## Verification Matrix

This is the strict status by area.

### Verified directly

- CLI flags exist and are wired:
  - `--configure-provider`
  - `--configure-model`
  - `--listmodels`
  - `--listvendors`
- single-provider persistence into `~/.config/fabric/.env`
- rejection of incomplete provider setup
- OpenRouter interactive provider setup in an isolated home
- OpenRouter live model listing
- OpenRouter default model persistence
- one-off OpenRouter runtime override
- direct Bedrock runtime in the real working tree
- default-provider runtime in the real working tree

### Verified by automated tests plus adjacent live runtime

- Bedrock setup/persistence regression paths
- vendor-manager behavior for incomplete setup
- registry persistence behavior for configured providers

### Implemented, but not exhaustively exercised in every permutation

- every possible interactive answer combination for every vendor
- every failure mode of provider APIs
- blank-home first-run bootstrap combined with every provider/model command sequence
- every provider other than the ones actively used here

So the honest statement is:

- yes, this provider/model CLI surface is materially tested
- no, not every theoretical permutation in Fabric has been exhaustively exercised

### Important boundary of verification

In a completely fresh blank home, provider configuration can succeed before patterns are installed. That means provider persistence and model persistence are verified independently from first-time pattern download.

This is expected behavior, not a failure of provider setup.

## Short Answers to the Original Questions

1. Where is the Bedrock API key saved
   In `~/.config/fabric/.env`.

2. How do I change it later
   Use `fabric --configure-provider Bedrock` or edit `~/.config/fabric/.env`.

3. Where is the model saved
   In `DEFAULT_MODEL` inside `~/.config/fabric/.env`.

4. Can I choose a different model from CLI
   Yes. Use `fabric --configure-model` or one-off `-m`.

5. Can Fabric fetch the model list
   Yes, via `--listmodels` and the interactive model chooser.

6. Can I switch to OpenRouter from CLI only
   Yes, via `fabric --configure-provider OpenRouter`.

7. Can I change Bedrock/OpenRouter keys from CLI
   Yes, through `--configure-provider <Vendor>`.

8. Can Fabric auto-fetch models when changing models
   Yes, for supported provider listings. OpenRouter live fetch was directly verified.

9. Is all of this checked in Fabric CLI
   Yes for the code paths documented above, with the verification boundaries noted in this guide.
