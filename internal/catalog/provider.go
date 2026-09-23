package catalog

import (
	"slices"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// Provider contains the identity, models, and request settings declared by one provider
// configuration.
//   - ID: the provider's catalog identifier
//   - DisplayName: the provider name shown to users
//   - APIKeyEnvVar: the environment variable that supplies the provider's API key
//   - Aggregator: whether the provider offers models from multiple vendors
//   - DefaultModel: the model ID used for default selection, or empty when undeclared
//   - DocsURL: the provider's documentation address
//   - Models: the models in configuration order
//   - Config: the provider's request settings, absent in identity-only copies
type Provider struct {
	ID           string          `json:"id"`
	DisplayName  string          `json:"displayName"`
	APIKeyEnvVar string          `json:"apiKeyEnvVar"`
	Aggregator   bool            `json:"aggregator,omitempty"`
	DefaultModel string          `json:"defaultModel,omitempty"`
	DocsURL      string          `json:"docsURL,omitempty"`
	Models       []Model         `json:"models,omitempty"`
	Config       *ProviderConfig `json:"config,omitempty"`
}

// Identity returns the provider's identity without its models or request settings.
func (p *Provider) Identity() Provider {
	return Provider{
		ID: p.ID, DisplayName: p.DisplayName, APIKeyEnvVar: p.APIKeyEnvVar,
		Aggregator: p.Aggregator, DefaultModel: p.DefaultModel, DocsURL: p.DocsURL,
	}
}

// clone returns a deep copy of the provider: its models and the slices and maps of its request
// settings.
func (p *Provider) clone() Provider {
	cloned := *p

	cloned.Models = make([]Model, len(p.Models))
	for i := range p.Models {
		cloned.Models[i] = p.Models[i].clone()
	}

	if p.Config == nil {
		return cloned
	}

	settings := p.Config.Clone()

	cloned.Config = &settings

	return cloned
}

// Model contains a model's identity, media kind, family, aliases, and accepted parameters.
//   - ID: the provider's model identifier
//   - Name: the model's configured display name
//   - Description: the optional model description
//   - Media: the kind of output the model produces
//   - Family: the model family used when a provider varies its request format by family
//   - Aliases: alternate model specifiers
//   - PromptIgnored: whether the model ignores the prompt and therefore requires none
//   - DocsURL: the model's documentation address
//   - Params: the parameter definitions accepted by the model
type Model struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	Media         media.Kind         `json:"media"`
	Family        string             `json:"family,omitempty"`
	Aliases       []string           `json:"aliases,omitempty"`
	PromptIgnored bool               `json:"promptIgnored,omitempty"`
	DocsURL       string             `json:"docsURL,omitempty"`
	Params        params.Definitions `json:"params,omitempty"`
}

// SupportsParam takes a parameter name and reports whether the model accepts it.
func (model *Model) SupportsParam(flagName params.FlagType) bool {
	_, ok := model.Param(flagName)

	return ok
}

// Param takes a flag name and returns its configuration and whether it is declared.
func (model *Model) Param(flagName params.FlagType) (params.Definition, bool) {
	return model.Params.Param(flagName)
}

// mediaSelected takes the media selections and reports whether the model's medium is among the
// selected media.
func (model *Model) mediaSelected(imageSelected, videoSelected bool) bool {
	return (model.Media == media.Image && imageSelected) || (model.Media == media.Video && videoSelected)
}

// clone returns a deep copy of the model, including parameter slices and size constraints.
func (model *Model) clone() Model {
	clonedModel := *model
	clonedModel.Aliases = slices.Clone(model.Aliases)

	paramCfgs := slices.Clone(model.Params)
	for i := range paramCfgs {
		paramCfg := &paramCfgs[i]

		paramCfg.AllowedValues = slices.Clone(paramCfg.AllowedValues)
		if paramCfg.CustomSize != nil {
			bounds := *paramCfg.CustomSize
			paramCfg.CustomSize = &bounds
		}
	}

	clonedModel.Params = paramCfgs

	return clonedModel
}

// findModel returns the provider identity and a copy of the model with the requested ID, if
// present.
func (p *Provider) findModel(modelID string) (ProvModelPair, bool) {
	for i := range p.Models {
		if p.Models[i].ID == modelID {
			return ProvModelPair{Provider: p.Identity(), Model: p.Models[i].clone()}, true
		}
	}

	return ProvModelPair{}, false
}

// findAlias returns the provider identity and a copy of the model declaring the alias, if present.
func (p *Provider) findAlias(modelAlias string) (ProvModelPair, bool) {
	for i := range p.Models {
		if slices.Contains(p.Models[i].Aliases, modelAlias) {
			return ProvModelPair{Provider: p.Identity(), Model: p.Models[i].clone()}, true
		}
	}

	return ProvModelPair{}, false
}
