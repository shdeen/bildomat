// Package templates holds the copy catalog source (copy.toml, generated
// into per-package constants by tools/copygen) and the embedded page-scale
// copy in the .tmpl files, so every wording is edited without touching
// rendering code.
package templates

//go:generate go run ../../tools/copygen

import _ "embed"

// CompactUsageText is the compact usage template rendered for invalid command-line input.
//
//go:embed usage.tmpl
var CompactUsageText string

// HelpMainText is the general help page: the usage forms, the commands, the
// flags, and the tips.
//
//go:embed help-main.tmpl
var HelpMainText string

// HelpTipsText is the tips section of the general help page: the fully
// qualified model form and the info and search examples.
//
//go:embed help-tips.tmpl
var HelpTipsText string

// HelpCommandText is one command's help page: its usage form and its flags.
//
//go:embed help-command.tmpl
var HelpCommandText string

// ProviderInfoText is the standard provider page: the provider's identity and
// catalog counts, then per medium its model roster and options.
//
//go:embed provider-info.tmpl
var ProviderInfoText string

// ProviderSummaryText is the aggregator summary: the provider's identity and
// catalog counts, its vendors and shared flags per medium, and the next commands.
//
//go:embed provider-summary.tmpl
var ProviderSummaryText string

// ModelInfoText is the model card: the model's identity block and one option
// per declared parameter.
//
//go:embed model-info.tmpl
var ModelInfoText string

// APIKeyText defines the credential guidance shared by provider and model pages.
//
//go:embed api-key.tmpl
var APIKeyText string

// ListNestedText is the default listing: each provider with its models by medium.
//
//go:embed list-nested.tmpl
var ListNestedText string

// ListProvidersText is the providers listing: one line per provider.
//
//go:embed list-providers.tmpl
var ListProvidersText string

// ListModelsText is the flat model directory: one fully qualified key per line.
//
//go:embed list-models.tmpl
var ListModelsText string
