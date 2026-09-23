// Package provider executes image and video requests defined by provider descriptions.
package provider

import (
	"context"
	"fmt"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// sharedProvider holds independent request settings for a descriptor-class provider.
//   - providerID: the provider identity used in errors
//   - imageAPI: optional image request settings
//   - videoAPI: optional video request settings
//   - stringParams: parameters that must be encoded as strings
type sharedProvider struct {
	providerID   string
	imageAPI     *catalog.ImageAPI
	videoAPI     *catalog.VideoAPI
	stringParams []params.FlagType
}

// NewProvider validates the API sections required by the models and returns a generator with
// independent copies of the request settings.
func NewProvider(providerDescription *catalog.Provider) (generation.Generator, error) {
	if providerDescription == nil || providerDescription.Config == nil {
		return nil, &errs.ConfigError{Cause: errs.ErrProvConfigInvalid}
	}

	providerSettings := providerDescription.Config.Clone()
	for modelIndex := range providerDescription.Models {
		model := &providerDescription.Models[modelIndex]
		if model.Media == media.Image && providerSettings.ImageAPI == nil {
			return nil, missingAPIError(providerDescription.ID, catalog.SectionImageAPI)
		}

		if model.Media == media.Video && providerSettings.VideoAPI == nil {
			return nil, missingAPIError(providerDescription.ID, catalog.SectionVideoAPI)
		}
	}

	if providerSettings.ImageAPI == nil && providerSettings.VideoAPI == nil {
		return nil, missingAPIError(providerDescription.ID, catalog.SectionImageAPI)
	}

	return &sharedProvider{providerID: providerDescription.ID, imageAPI: providerSettings.ImageAPI, videoAPI: providerSettings.VideoAPI, stringParams: providerSettings.StringParams}, nil
}

// AdapterSettings returns an owned copy of the selected provider's adapter settings. A missing
// description or adapter section returns a configuration error naming providerID.
func AdapterSettings(providerDescription *catalog.Provider, providerID string) (*catalog.AdapterAPI, error) {
	if providerDescription == nil || providerDescription.Config == nil || providerDescription.Config.AdapterAPI == nil {
		return nil, &errs.ConfigError{Provider: providerID, Cause: errs.ErrProvConfigNoAdapterAPI}
	}

	providerSettings := providerDescription.Config.Clone()

	return providerSettings.AdapterAPI, nil
}

// AdjustParams returns adjusted parameters, retained media, and change records. Video API settings
// determine frame handling and image resizing.
func (provider *sharedProvider) AdjustParams(model *catalog.Model, inputs params.FlagInputs, mediaInputs []media.Input, _ *metadata.Reuse) (generation.Preparation, error) {
	preparedGeneration, err := generation.AdjustGeneration(model, inputs, mediaInputs)
	if err != nil {
		return preparedGeneration, err
	}

	framesSupported := false
	if model.Media == media.Video && provider.videoAPI != nil {
		_, framesSupported = provider.videoAPI.FrameFields()
	}

	if framesSupported {
		changes, frameErr := generation.ResolveFrameAnchors(preparedGeneration.InputMedia, preparedGeneration.Params)

		preparedGeneration.Changes = append(preparedGeneration.Changes, changes...)
		if frameErr != nil {
			return preparedGeneration, frameErr
		}
	} else {
		preparedGeneration.Changes = append(preparedGeneration.Changes, generation.DropFramePrefixes(preparedGeneration.InputMedia)...)
	}

	if model.Media == media.Video && provider.videoAPI != nil && provider.videoAPI.InputMediaMustResize {
		changes, conformErr := generation.ConformInputMedia(model, preparedGeneration.Params, preparedGeneration.InputMedia)

		preparedGeneration.Changes = append(preparedGeneration.Changes, changes...)
		if conformErr != nil {
			return preparedGeneration, conformErr
		}
	}

	return preparedGeneration, nil
}

// Generate submits the selected image or video request and returns its artifacts. It copies the
// request preparation and returns any preparation completed before a failure.
func (provider *sharedProvider) Generate(ctx context.Context, run *generation.Generation) (generation.Result, error) {
	request := *run
	request.Preparation = run.Clone()
	result := generation.Result{Preparation: request.Preparation}

	var err error

	if run.Model.Media == media.Video {
		if provider.videoAPI == nil {
			return result, missingAPIError(provider.providerID, catalog.SectionVideoAPI)
		}

		result.Artifacts, err = submitVideo(ctx, provider.videoAPI, run.APIKey, &request, provider.stringParams...)
	} else {
		if provider.imageAPI == nil {
			return result, missingAPIError(provider.providerID, catalog.SectionImageAPI)
		}

		result.Artifacts, err = submitImage(ctx, provider.imageAPI, run.APIKey, &request, provider.stringParams...)
	}

	result.Preparation = request.Preparation

	return result, err
}

// missingAPIError classifies an absent API section and names its provider.
func missingAPIError(providerID, section string) error {
	return &errs.ConfigError{Provider: providerID, Problem: fmt.Sprintf(APISectionMissing, providerID, section), Cause: errs.ErrProvConfigInvalid}
}
