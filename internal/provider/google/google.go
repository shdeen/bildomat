// Package google provides image and video generation through Google media APIs.
package google

import (
	"context"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/provider"

	_ "embed"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// ConfigJSON is the embedded Google provider configuration document.
//
//go:embed config/google.json
var ConfigJSON []byte

// ProviderID is the Google provider identifier.
const ProviderID = "google"

// Provider generates media through the Google Veo and Interactions APIs.
type Provider struct {
	// adapterAPI contains the endpoints and polling settings used by the provider.
	adapterAPI *catalog.AdapterAPI
}

// NewProvider returns a generator with independent adapter settings. It rejects missing settings
// and configured paths that overwrite required Interactions fields.
func NewProvider(providerDescription *catalog.Provider) (generation.Generator, error) {
	adapterSettings, err := provider.AdapterSettings(providerDescription, ProviderID)
	if err != nil {
		return nil, err
	}

	for modelIndex := range providerDescription.Models {
		model := &providerDescription.Models[modelIndex]
		if veoFamily(model) {
			continue
		}

		if err := provider.CheckOwnedPaths(ProviderID, model,
			interactionFieldModel, wireKeyInput,
			wireKeyResponseFormat+"."+interactionFieldType, wireKeyResponseFormat+"."+wireKeyDelivery,
			wireKeyGenerationConfig+"."+wireKeyThinkingSummaries,
			wireKeyGenerationConfig+"."+wireKeyVideoConfig+"."+wireKeyTask); err != nil {
			return nil, err
		}
	}

	return &Provider{adapterAPI: adapterSettings}, nil
}

// AdjustParams returns model-compatible parameters, media, and change records. It validates reuse
// selections and Veo frame constraints, and removes frame prefixes from Interactions inputs.
func (*Provider) AdjustParams(model *catalog.Model, inputs params.FlagInputs, mediaInputs []media.Input, reuse *metadata.Reuse) (generation.Preparation, error) {
	preparedGeneration, adjustmentErr := generation.AdjustGeneration(model, inputs, mediaInputs)
	if adjustmentErr != nil {
		return preparedGeneration, adjustmentErr
	}

	if reuse != nil {
		reuseURI, reuseChanges, err := adjustReuse(model, preparedGeneration.Params, preparedGeneration.InputMedia, reuse)
		if err != nil {
			return preparedGeneration, err
		}

		preparedGeneration.Changes = append(preparedGeneration.Changes, reuseChanges...)
		preparedGeneration.ReuseURI = reuseURI

		return preparedGeneration, nil
	}

	if veoFamily(model) {
		var (
			forceRecords []params.Adjustment
			err          error
		)

		forceRecords, err = adjustVeoDuration(preparedGeneration.Params, preparedGeneration.InputMedia)

		preparedGeneration.Changes = append(preparedGeneration.Changes, forceRecords...)
		if err != nil {
			return preparedGeneration, err
		}

		frameRecords, err := generation.ResolveFrameAnchors(preparedGeneration.InputMedia, preparedGeneration.Params)

		preparedGeneration.Changes = append(preparedGeneration.Changes, frameRecords...)
		if err != nil {
			return preparedGeneration, err
		}

		if err := validateVeoInputMedia(preparedGeneration.InputMedia); err != nil {
			return preparedGeneration, err
		}
	} else {
		preparedGeneration.Changes = append(preparedGeneration.Changes, generation.DropFramePrefixes(preparedGeneration.InputMedia)...)
	}

	return preparedGeneration, nil
}

// validateVeoInputMedia rejects mixed images and videos, multiple videos, and a closing frame
// without an opening frame. The caller must resolve frame prefixes to anchors first.
func validateVeoInputMedia(mediaInputs []media.Input) error {
	imageInputs, videoInputs := media.SplitInputs(mediaInputs)
	if len(imageInputs) > 0 && len(videoInputs) > 0 {
		return &errs.MediaError{Problem: VeoMediaMixed, Cause: errs.ErrInputMedia}
	}

	if len(videoInputs) > 1 {
		return &errs.MediaError{Problem: VeoOneVideo, Cause: errs.ErrInputMedia}
	}

	openingFrames, closingFrames := 0, 0

	for inputIndex := range imageInputs {
		switch imageInputs[inputIndex].FrameAnchor {
		case media.FrameFirst:
			openingFrames++
		case media.FrameLast:
			closingFrames++
		}
	}

	if closingFrames == 1 && openingFrames == 0 {
		return &errs.MediaError{Problem: VeoClosingNeedsOpening, Cause: errs.ErrInputMediaTime}
	}

	return nil
}

// Generate submits to the model's API family and returns artifacts and thought summaries. It
// preserves the caller's request and returns completed preparation even on failure.
func (prov *Provider) Generate(ctx context.Context, run *generation.Generation) (generation.Result, error) {
	request := *run

	request.Preparation = run.Clone()

	session := apiSession{settings: prov.adapterAPI, credential: httpapi.HeaderCred(googleKeyHeader, run.APIKey), model: run.Label()}

	var (
		result generation.Result
		err    error
	)
	if veoFamily(&run.Model) {
		result, err = session.generateVeo(ctx, &request)
	} else {
		result, err = session.generateInteraction(ctx, &request)
	}

	result.Preparation = request.Preparation

	return result, err
}

// Google API selection and authentication values.
//   - familyVeo: the model family served by the Veo API
//   - googleKeyHeader: the request header carrying the API key
const (
	familyVeo       = "veo"
	googleKeyHeader = "x-goog-api-key"
)

// veoFamily reports whether a model belongs to the Veo API family.
func veoFamily(model *catalog.Model) bool {
	return model.Family == familyVeo
}
