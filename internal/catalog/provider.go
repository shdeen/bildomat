package catalog

import (
	"maps"
	"slices"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// Provider is one provider as its configuration document declares it: its
// identity, its models, and its request settings. The request settings never
// reach a user-facing document; a reduced copy leaves them zero, which the
// encoding omits.
//   - ID: the provider's catalog identifier
//   - DisplayName: the provider name shown to users
//   - APIKeyEnvVar: the environment variable that contains the provider's API key
//   - APIKeyConfigKey: the dotted user-config key path, included in public identity copies
//   - Aggregator: whether the provider is a gateway to many vendors' models, which
//     the details page summarizes instead of listing
//   - DefaultModel: the ID of the provider's model used when --model is omitted and the
//     provider is the first configured one in listing order; empty when the provider names none
//   - DocsURL: the address of the provider's own documentation, shown on its pages
//   - Models: the models the provider offers, in configuration order
//   - Config: the provider's request settings; a configuration document must
//     declare them, and a reduced copy carries none
type Provider struct {
	ID              string          `json:"id"`
	DisplayName     string          `json:"displayName"`
	APIKeyEnvVar    string          `json:"apiKeyEnvVar"`
	APIKeyConfigKey string          `json:"apiKeyConfigKey,omitempty"`
	Aggregator      bool            `json:"aggregator,omitempty"`
	DefaultModel    string          `json:"defaultModel,omitempty"`
	DocsURL         string          `json:"docsURL,omitempty"`
	Models          []Model         `json:"models,omitempty"`
	Config          *ProviderConfig `json:"config,omitempty"`
}

// Identity returns the provider's identity alone: no models, no request
// settings or resolved credentials.
func (p *Provider) Identity() Provider {
	return Provider{
		ID: p.ID, DisplayName: p.DisplayName, APIKeyEnvVar: p.APIKeyEnvVar,
		APIKeyConfigKey: "api-keys." + p.ID,
		Aggregator:      p.Aggregator, DefaultModel: p.DefaultModel, DocsURL: p.DocsURL,
	}
}

// clone returns a deep copy of the provider: its models and the slices and maps
// of its request settings.
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

// Model contains a model's identity, media kind, family, aliases, and accepted params.
//   - ID: the provider's model identifier
//   - Name: the name the provider publishes for the model
//   - Description: the description the provider publishes, or empty where it publishes none
//   - Media: the kind of output that the model produces
//   - Family: the provider's model family, for a provider whose endpoints or
//     request shapes differ by family; empty where the provider has one family
//   - Aliases: alternate model specifiers
//   - PromptIgnored: whether the model ignores a prompt, so a run needs none
//   - DocsURL: the address of the model's own documentation, shown on its card
//   - Params: the parameter configurations accepted by the model
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

// MediaSelected takes the media selections and reports whether the model's
// medium is among the selected media.
func (model *Model) MediaSelected(imageSelected, videoSelected bool) bool {
	return (model.Media == media.Image && imageSelected) || (model.Media == media.Video && videoSelected)
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

// Clone returns independent copies of the request settings and their nested collections.
func (config *ProviderConfig) Clone() ProviderConfig {
	settings := *config
	settings.StringParams = slices.Clone(config.StringParams)

	if config.ImageAPI != nil {
		imageAPI := *config.ImageAPI
		imageAPI.FixedProvFields = maps.Clone(imageAPI.FixedProvFields)
		settings.ImageAPI = &imageAPI
	}

	if config.VideoAPI != nil {
		videoAPI := *config.VideoAPI
		videoAPI.ProgressStatusText = slices.Clone(videoAPI.ProgressStatusText)
		videoAPI.FailedStatusText = slices.Clone(videoAPI.FailedStatusText)
		videoAPI.URLPathSeq = slices.Clone(videoAPI.URLPathSeq)
		settings.VideoAPI = &videoAPI
	}

	if config.AdapterAPI != nil {
		adapterAPI := *config.AdapterAPI
		adapterAPI.PendingStatusText = slices.Clone(adapterAPI.PendingStatusText)
		adapterAPI.FailedStatusText = slices.Clone(adapterAPI.FailedStatusText)
		settings.AdapterAPI = &adapterAPI
	}

	return settings
}
