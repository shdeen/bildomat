package output

import (
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// CatalogPage contains the providers, models, and parameter descriptions selected for a listing or
// information document. Its values are independent of the catalog. Empty listings retain an empty
// providers array; unneeded flag records are omitted.
//   - Providers: the selected provider and model descriptions
//   - Flags: the flag definitions referenced by information pages
type CatalogPage struct {
	Providers []providerRecord `json:"providers"`
	Flags     []params.Flag    `json:"flags,omitempty"`
}

// providerRecord contains public identity and selected models, without request settings.
//   - ID, DisplayName: the catalog identifier and published provider name
//   - APIKeyEnvVar: the environment variable that supplies the credential
//   - APIKeyConfigKey: the credential key in the user configuration
//   - Aggregator: whether the provider serves models from multiple vendors
//   - DefaultModel: the provider default model identifier
//   - DocsURL: the provider documentation URL
//   - Models: the selected model descriptions
type providerRecord struct {
	ID              string        `json:"id"`
	DisplayName     string        `json:"displayName"`
	APIKeyEnvVar    string        `json:"apiKeyEnvVar"`
	APIKeyConfigKey string        `json:"apiKeyConfigKey,omitempty"`
	Aggregator      bool          `json:"aggregator,omitempty"`
	DefaultModel    string        `json:"defaultModel,omitempty"`
	DocsURL         string        `json:"docsURL,omitempty"`
	Models          []modelRecord `json:"models,omitempty"`
}

// modelRecord contains a displayed model's identity and optional parameter descriptions.
//   - ID, Name: the catalog identifier and published model name
//   - Description: the catalog description
//   - Media: the generated medium
//   - Family: the related model family
//   - Aliases: the included model selection aliases
//   - PromptIgnored: whether the model disregards the prompt
//   - DocsURL: the model documentation URL
//   - Params: the accepted parameter descriptions
type modelRecord struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Media         media.Kind        `json:"media"`
	Family        string            `json:"family,omitempty"`
	Aliases       []string          `json:"aliases,omitempty"`
	PromptIgnored bool              `json:"promptIgnored,omitempty"`
	DocsURL       string            `json:"docsURL,omitempty"`
	Params        []parameterRecord `json:"params,omitempty"`
}

// parameterRecord describes an accepted parameter without its provider request key.
//   - FlagID: the permanent parameter flag identifier
//   - Required: whether the parameter is required
//   - AllowedValues: the accepted values, when constrained to a set
//   - MinValue, MaxValue: the optional inclusive numeric limits
//   - MaxMultiple: the maximum number of repeated values
//   - CustomSize: the constraints on custom image dimensions
//   - RuleDescription: the explanation of a conditional parameter rule
//   - ModelInfoComment: the extra parameter guidance on the model information page
type parameterRecord struct {
	FlagID           params.FlagType          `json:"flagID"`
	Required         bool                     `json:"required,omitempty"`
	AllowedValues    []string                 `json:"allowedValues,omitempty"`
	MinValue         params.Nullable[float64] `json:"minValue,omitzero"`
	MaxValue         params.Nullable[float64] `json:"maxValue,omitzero"`
	MaxMultiple      int                      `json:"maxMultiple,omitempty"`
	CustomSize       *params.SizeBounds       `json:"customSize,omitempty"`
	RuleDescription  string                   `json:"ruleDescription,omitempty"`
	ModelInfoComment string                   `json:"modelInfoComment,omitempty"`
}

// ListingPage describes already selected provider/model pairs. Providers sort by display name,
// ignoring case, with aggregators last; models sort by identifier. Models and aliases appear only
// when requested. Listings never include parameters.
func ListingPage(pairs []catalog.ProvModelPair, withModels, includeAliases bool) CatalogPage {
	page := CatalogPage{Providers: []providerRecord{}}
	providers := listedProviders(pairs)
	slices.SortFunc(providers, catalog.CompareProviders)

	for index := range providers {
		provider := providerIdentity(&providers[index])

		if withModels {
			for pairIndex := range pairs {
				if pairs[pairIndex].Provider.ID == provider.ID {
					provider.Models = append(provider.Models, modelIdentity(&pairs[pairIndex].Model, includeAliases))
				}
			}

			slices.SortFunc(provider.Models, compareModelIDs)
		}

		page.Providers = append(page.Providers, provider)
	}

	return page
}

// compareModelIDs orders displayed model identifiers alphabetically.
//
//nolint:gocritic // slices.SortFunc requires model values; sorting must not add pointer ownership to page records.
func compareModelIDs(firstModel, secondModel modelRecord) int {
	return strings.Compare(firstModel.ID, secondModel.ID)
}

// ProviderInfoPage describes a provider and its already selected models in input order, followed by
// the referenced parameter flags in their declaration order.
func ProviderInfoPage(prov *catalog.Provider, pairs []catalog.ProvModelPair, paramFlags []params.Flag) CatalogPage {
	provider := providerIdentity(prov)
	for index := range pairs {
		provider.Models = append(provider.Models, detailedModel(&pairs[index].Model))
	}

	return CatalogPage{Providers: []providerRecord{provider}, Flags: referencedFlags(provider.Models, paramFlags)}
}

// ModelInfoPage describes one provider/model pair and its referenced parameter flags.
func ModelInfoPage(pair *catalog.ProvModelPair, paramFlags []params.Flag) CatalogPage {
	provider := providerIdentity(&pair.Provider)
	provider.Models = []modelRecord{detailedModel(&pair.Model)}

	return CatalogPage{Providers: []providerRecord{provider}, Flags: referencedFlags(provider.Models, paramFlags)}
}

// providerIdentity prepares public provider identity, including its user-config key.
func providerIdentity(prov *catalog.Provider) providerRecord {
	return providerRecord{
		ID: prov.ID, DisplayName: prov.DisplayName, APIKeyEnvVar: prov.APIKeyEnvVar,
		APIKeyConfigKey: "api-keys." + prov.ID, Aggregator: prov.Aggregator,
		DefaultModel: prov.DefaultModel, DocsURL: prov.DocsURL,
	}
}

// modelIdentity prepares the displayed model fields and an independent alias list.
func modelIdentity(model *catalog.Model, includeAliases bool) modelRecord {
	record := modelRecord{
		ID: model.ID, Name: model.Name, Description: model.Description,
		Media: model.Media, Family: model.Family, PromptIgnored: model.PromptIgnored,
		DocsURL: model.DocsURL,
	}
	if includeAliases {
		record.Aliases = slices.Clone(model.Aliases)
	}

	return record
}

// detailedModel adds independent parameter descriptions to the displayed identity.
func detailedModel(model *catalog.Model) modelRecord {
	record := modelIdentity(model, true)
	for index := range model.Params {
		definition := &model.Params[index]

		parameter := parameterRecord{
			FlagID: definition.FlagID, Required: definition.Required,
			AllowedValues: slices.Clone(definition.AllowedValues),
			MinValue:      definition.MinValue, MaxValue: definition.MaxValue,
			MaxMultiple: definition.MaxMultiple, RuleDescription: definition.RuleDescription,
			ModelInfoComment: definition.ModelInfoComment,
		}
		if definition.CustomSize != nil {
			bounds := *definition.CustomSize
			parameter.CustomSize = &bounds
		}

		record.Params = append(record.Params, parameter)
	}

	return record
}

// referencedFlags copies referenced flag records in their original order.
func referencedFlags(models []modelRecord, paramFlags []params.Flag) []params.Flag {
	referenced := map[params.FlagType]bool{}

	for index := range models {
		for parameterIndex := range models[index].Params {
			referenced[models[index].Params[parameterIndex].FlagID] = true
		}
	}

	var flags []params.Flag

	for index := range paramFlags {
		if referenced[paramFlags[index].FlagID] {
			flag := paramFlags[index]
			flag.Aliases = slices.Clone(flag.Aliases)
			flag.ExampleValues = slices.Clone(flag.ExampleValues)
			flags = append(flags, flag)
		}
	}

	return flags
}

// ModelsForMedia returns the listed models of the requested medium in their existing order.
func (provider *providerRecord) ModelsForMedia(mediaKind media.Kind) []modelRecord {
	var models []modelRecord

	for index := range provider.Models {
		if provider.Models[index].Media == mediaKind {
			models = append(models, provider.Models[index])
		}
	}

	return models
}
