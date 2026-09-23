// Package sourceful provides image generation through the Sourceful Design API.
package sourceful

import (
	"context"
	"fmt"
	"strings"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/provider"

	_ "embed"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// ConfigJSON is the embedded Sourceful provider configuration document.
//
//go:embed config/sourceful.json
var ConfigJSON []byte

// ProviderID is the Sourceful provider identifier.
const ProviderID = "sourceful"

// generator holds the Sourceful request settings.
//   - adapterAPI: endpoints, polling limits, and artifact defaults
type generator struct {
	adapterAPI *catalog.AdapterAPI
}

// NewProvider returns a generator with independent adapter settings. It rejects missing settings
// and configured paths that overwrite required request fields.
func NewProvider(providerDescription *catalog.Provider) (generation.Generator, error) {
	adapterSettings, err := provider.AdapterSettings(providerDescription, ProviderID)
	if err != nil {
		return nil, err
	}

	for modelIndex := range providerDescription.Models {
		if err := provider.CheckOwnedPaths(ProviderID, &providerDescription.Models[modelIndex], fieldModel, fieldInstruction, fieldIdempotencyKey, fieldImageURLs); err != nil {
			return nil, err
		}
	}

	return &generator{adapterAPI: adapterSettings}, nil
}

// AdjustParams returns the Sourceful generation parameters and adjustment records.
func (*generator) AdjustParams(model *catalog.Model, flagInputs params.FlagInputs, mediaInputs []media.Input, _ *metadata.Reuse) (generation.Preparation, error) {
	preparedGeneration, adjustmentErr := generation.AdjustGeneration(model, flagInputs, mediaInputs)
	if adjustmentErr != nil {
		return preparedGeneration, adjustmentErr
	}

	preparedGeneration.Changes = append(preparedGeneration.Changes, generation.DropFramePrefixes(preparedGeneration.InputMedia)...)

	return preparedGeneration, nil
}

// Generate creates a Sourceful job, waits for completion, and downloads its image.
func (sourceful *generator) Generate(ctx context.Context, generationRequest *generation.Generation) (generation.Result, error) {
	apiKey := generationRequest.APIKey

	providerModelName := generationRequest.Label()
	apiCredential := httpapi.HeaderCred(keyHeader, apiKey)

	generationRequest.Record.Prepared(generationRequest.InputMedia)

	jobID, err := sourceful.createJob(ctx, apiCredential, providerModelName, generationRequest)
	if err != nil {
		return generation.Result{Preparation: generationRequest.Clone()}, err
	}

	generatedMedia, err := sourceful.fetchResult(ctx, apiCredential, providerModelName, jobID, generationRequest.Record)
	if err != nil {
		return generation.Result{Preparation: generationRequest.Clone()}, err
	}

	return generation.Result{Preparation: generationRequest.Clone(), Artifacts: []artifact.Media{generatedMedia}}, nil
}

// createJob posts a text or image generation request and returns its job ID.
func (sourceful *generator) createJob(ctx context.Context, apiCredential httpapi.AuthCredential, providerModelName string, generationRequest *generation.Generation) (string, error) {
	creationRoute := textRoute
	if len(generationRequest.InputMedia) > 0 {
		creationRoute = imageRoute
	}

	creationStatus, creationBody, err := httpapi.PostJSON(
		ctx,
		strings.TrimRight(sourceful.adapterAPI.APIBase, "/")+creationRoute,
		apiCredential,
		buildRequestBody(generationRequest), generationRequest.Record,
	)
	if err != nil {
		return "", fmt.Errorf("%q, %w", providerModelName, err)
	}

	if creationStatus/100 != 2 {
		return "", provider.APIErr(providerModelName, creationStatus, creationBody)
	}

	return creationJobID(creationBody, providerModelName)
}

// fetchResult waits for the job to complete and downloads its image artifact. It records provider
// completion and the artifact response.
func (sourceful *generator) fetchResult(ctx context.Context, apiCredential httpapi.AuthCredential, providerModelName, jobID string, record *metadata.Record) (artifact.Media, error) {
	jobPoll := &jobPoll{
		adapterAPI:        sourceful.adapterAPI,
		apiCredential:     apiCredential,
		jobID:             jobID,
		providerModelName: providerModelName,
		record:            record,
	}

	err := httpapi.Poll(ctx, sourceful.adapterAPI.PollInterval.Duration(), sourceful.adapterAPI.PollTimeout.Duration(), jobPoll)
	if err != nil {
		return artifact.Media{}, &errs.PollError{Model: providerModelName, Resource: jobID, Cause: err}
	}

	fallbackExtension := media.ExtForMimeOr(jobPoll.resultMIME, sourceful.adapterAPI.ImageFallbackExt)

	record.ProviderFinished(nil)

	generatedMedia, err := httpapi.Fetch(ctx, jobPoll.resultURL, httpapi.AuthCredential{}, fallbackExtension, record)
	if err != nil {
		return artifact.Media{}, fmt.Errorf("%q, %w", providerModelName, err)
	}

	return generatedMedia, nil
}
