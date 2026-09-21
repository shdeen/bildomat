// Package kling provides image and video generation through the Kling API.
package kling

import (
	"context"
	"fmt"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/provider"

	_ "embed"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// ConfigJSON is the embedded Kling provider configuration document.
//
//go:embed config/kling.json
var ConfigJSON []byte

// ProviderID is the Kling provider identifier.
const ProviderID = "kling"

// The audio parameter's flag and its request values.
//   - flagGenerateAudio: the flag ID the config declares for the audio switch
//   - audioOff: the value sent when audio is off
//   - audioNative: the value sent when audio is on
const (
	flagGenerateAudio params.FlagType = "generate-audio"

	audioOff    = "off"
	audioNative = "native"
)

// generator generates images and videos through the Kling API.
type generator struct {
	adapterAPI *catalog.AdapterAPI
}

// NewProvider returns a generator with owned adapter settings, or a missing-description error.
func NewProvider(providerDescription *catalog.Provider) (generation.Generator, error) {
	adapterSettings, err := provider.AdapterSettings(providerDescription, ProviderID)
	if err != nil {
		return nil, err
	}

	return &generator{adapterAPI: adapterSettings}, nil
}

// AdjustParams returns the Kling generation parameters and adjustment records.
func (*generator) AdjustParams(model *catalog.Model, inputs params.FlagInputs, mediaInputs []media.Input, _ *metadata.Reuse) (generation.Preparation, error) {
	preparedGeneration, adjustmentErr := generation.AdjustGeneration(model, inputs, mediaInputs)
	if adjustmentErr != nil {
		return preparedGeneration, adjustmentErr
	}

	for mediaIndex := range preparedGeneration.InputMedia {
		mediaInput := &preparedGeneration.InputMedia[mediaIndex]
		if mediaInput.Kind() == media.Video {
			detail := fmt.Sprintf(VideoInputRejected, mediaInput.Source())

			return preparedGeneration, &errs.MediaError{Problem: detail, Cause: errs.ErrInputMediaUnsendable}
		}
	}

	if model.Media == media.Video {
		frameChanges, err := generation.ResolveFrameAnchors(preparedGeneration.InputMedia, preparedGeneration.Params)

		preparedGeneration.Changes = append(preparedGeneration.Changes, frameChanges...)
		if err != nil {
			return preparedGeneration, err
		}
	} else {
		preparedGeneration.Changes = append(preparedGeneration.Changes, generation.DropFramePrefixes(preparedGeneration.InputMedia)...)
	}

	if audioEnabled, supplied := preparedGeneration.Params[flagGenerateAudio].(bool); supplied {
		audioValue := audioOff
		if audioEnabled {
			audioValue = audioNative
		}

		preparedGeneration.Params[flagGenerateAudio] = audioValue
		preparedGeneration.Changes = append(preparedGeneration.Changes, params.Adjustment{
			FlagID: flagGenerateAudio, Type: params.ChangeConformed,
			InputVal: params.FormatValue(audioEnabled), WireVal: audioValue,
		})
	}

	return preparedGeneration, nil
}

// Generate starts one Kling task, waits for completion, and downloads its artifacts.
func (prov *generator) Generate(ctx context.Context, generationRequest *generation.Generation) (generation.Result, error) {
	if generationRequest == nil {
		return generation.Result{}, fmt.Errorf("%q, %w", ProviderID, errs.ErrResponseNoData)
	}

	result, err := prov.generateMedia(ctx, generationRequest)
	result.Preparation = generationRequest.Clone()

	return result, err
}

// generateMedia starts the selected image or video task and retrieves its completed artifacts.
func (prov *generator) generateMedia(ctx context.Context, generationRequest *generation.Generation) (generation.Result, error) {
	apiKey := generationRequest.APIKey

	providerModelName := generationRequest.Label()
	credential := httpapi.Bearer(apiKey)
	creationPath, requestBody := buildRequestBody(generationRequest)
	apiBase := strings.TrimRight(prov.adapterAPI.APIBase, "/")

	generationRequest.Record.Prepared(generationRequest.InputMedia)

	inputDefinition, _ := generationRequest.Model.Param(params.FlagTypeInputMedia)
	generationRequest.Record.SetFields([]metadata.BinaryField{
		{Path: []string{inputDefinition.ParamID}},
		{Path: []string{inputDefinition.ParamID, "*", fieldImage}},
		{Path: []string{fieldContents, "*", fieldURL}},
	}, nil)

	taskID, err := createTask(ctx, apiBase+creationPath, credential, generationRequest, requestBody)
	if err != nil {
		return generation.Result{}, err
	}

	var completed generation.Result
	if generationRequest.Model.Media == media.Image {
		completed, err = prov.completeImageTask(ctx, &imageJobPoll{
			apiBase:           apiBase,
			credential:        credential,
			taskID:            taskID,
			providerModelName: providerModelName,
			queryPath:         creationPath + "/" + taskID,
			record:            generationRequest.Record,
		})
	} else {
		completed, err = prov.completeVideoTask(ctx, &videoJobPoll{
			apiBase:           apiBase,
			credential:        credential,
			taskID:            taskID,
			providerModelName: providerModelName,
			record:            generationRequest.Record,
		})
	}

	return completed, err
}

// completeImageTask polls one image task until it completes and downloads its
// result images in provider index order, returning them as the result.
func (prov *generator) completeImageTask(ctx context.Context, imagePoll *imageJobPoll) (generation.Result, error) {
	if err := httpapi.Poll(ctx, prov.adapterAPI.PollInterval.Duration(), prov.adapterAPI.PollTimeout.Duration(), imagePoll); err != nil {
		return generation.Result{}, &errs.PollError{Model: imagePoll.providerModelName, Resource: imagePoll.taskID, Cause: err}
	}

	imagePoll.record.ProviderFinished(nil)

	artifacts, err := downloadImages(ctx, imagePoll.providerModelName, imagePoll.resultURLs, prov.adapterAPI.ImageFallbackExt, imagePoll.record)
	if err != nil {
		return generation.Result{}, err
	}

	return generation.Result{Artifacts: artifacts}, nil
}

// completeVideoTask polls one video task until it completes and fetches its
// result video, returning it as the result.
func (prov *generator) completeVideoTask(ctx context.Context, videoPoll *videoJobPoll) (generation.Result, error) {
	if err := httpapi.Poll(ctx, prov.adapterAPI.PollInterval.Duration(), prov.adapterAPI.PollTimeout.Duration(), videoPoll); err != nil {
		return generation.Result{}, &errs.PollError{Model: videoPoll.providerModelName, Resource: videoPoll.taskID, Cause: err}
	}

	videoPoll.record.ProviderFinished(nil)

	generatedMedia, err := httpapi.Fetch(ctx, videoPoll.resultURL, httpapi.AuthCredential{}, prov.adapterAPI.VideoFallbackExt, videoPoll.record)
	if err != nil {
		return generation.Result{}, fmt.Errorf("%q: %w, %w", videoPoll.providerModelName, errs.ErrTransportDownload, err)
	}

	return generation.Result{Artifacts: []artifact.Media{generatedMedia}}, nil
}
