// Package bfl provides image and video generation through the Black Forest Labs asynchronous API.
package bfl

import (
	"context"
	"fmt"

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

// ConfigJSON is the embedded Black Forest Labs provider configuration document.
//
//go:embed config/bfl.json
var ConfigJSON []byte

// ProviderID is the Black Forest Labs provider identifier.
const ProviderID = "bfl"

// Provider generates images and videos through the Black Forest Labs asynchronous API.
type Provider struct {
	// adapterAPI contains the endpoints and polling settings used by the provider.
	adapterAPI *catalog.AdapterAPI
}

// NewProvider returns a generator with owned adapter settings, or a missing-description error.
func NewProvider(providerDescription *catalog.Provider) (generation.Generator, error) {
	adapterSettings, err := provider.AdapterSettings(providerDescription, ProviderID)
	if err != nil {
		return nil, err
	}

	return &Provider{adapterAPI: adapterSettings}, nil
}

// AdjustParams returns model-compatible generation parameters and records describing each adjustment.
// Video models resolve the first and last frame anchors to keyframe times; image models drop
// frame prefixes with a record.
func (*Provider) AdjustParams(model *catalog.Model, inputs params.FlagInputs, mediaInputs []media.Input, _ *metadata.Reuse) (generation.Preparation, error) {
	preparedGeneration, adjustmentErr := generation.AdjustGeneration(model, inputs, mediaInputs)
	if adjustmentErr != nil {
		return preparedGeneration, adjustmentErr
	}

	if model.Media == media.Video {
		resolveFluxFrameAnchors(preparedGeneration.InputMedia, preparedGeneration.Params)
	} else {
		preparedGeneration.Changes = append(preparedGeneration.Changes, generation.DropFramePrefixes(preparedGeneration.InputMedia)...)
	}

	if _, err := requestBody(model, "", preparedGeneration.Params, preparedGeneration.InputMedia); err != nil {
		return preparedGeneration, err
	}

	return preparedGeneration, nil
}

// Generate starts an image or video job, waits for completion, and returns the downloaded artifact.
func (p *Provider) Generate(ctx context.Context, run *generation.Generation) (generation.Result, error) {
	apiKey := run.APIKey

	provModelLabel := run.Label()
	cred := httpapi.HeaderCred(keyHeader, apiKey)

	jobID, pollURL, err := startJob(ctx, p.adapterAPI.APIBase, cred, provModelLabel, &run.Model, run.Prompt, run.Params, run.InputMedia, run.Record)
	if err != nil {
		return generation.Result{Preparation: run.Clone()}, err
	}

	probe := &pollProbe{cred: cred, adapterAPI: p.adapterAPI, id: jobID, pollURL: pollURL, record: run.Record}

	err = httpapi.Poll(ctx, p.adapterAPI.PollInterval.Duration(), p.adapterAPI.PollTimeout.Duration(), probe)
	if err != nil {
		return generation.Result{Preparation: run.Clone()}, &errs.PollError{Model: provModelLabel, Resource: jobID, Cause: err}
	}

	fallbackExt := p.adapterAPI.ImageFallbackExt
	if run.Model.Media == media.Video {
		fallbackExt = p.adapterAPI.VideoFallbackExt
	}

	run.Record.ProviderFinished(nil)

	generatedMedia, err := httpapi.Fetch(ctx, probe.completed.Result.Sample, httpapi.AuthCredential{}, fallbackExt, run.Record)
	if err != nil {
		return generation.Result{Preparation: run.Clone()}, fmt.Errorf("%q, %w, %w", provModelLabel, errs.ErrTransport, err)
	}

	return generation.Result{Preparation: run.Clone(), Artifacts: []artifact.Media{generatedMedia}}, nil
}
