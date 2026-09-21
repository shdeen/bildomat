package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
	"github.com/shdeen/bildomat/internal/provider"
)

// apiSession owns one generation's resolved Google credential, provider
// settings, and model identity. Contexts belong to individual operations.
type apiSession struct {
	settings   *catalog.AdapterAPI
	credential httpapi.AuthCredential
	model      string
}

// generateInteraction sends an Interactions request and returns its media artifact and thought summaries.
func (session *apiSession) generateInteraction(ctx context.Context, run *generation.Generation) (generation.Result, error) {
	prompt := run.Prompt

	var (
		body map[string]any
		err  error
	)

	mediaBlockType, fallbackExt := string(media.Image), session.settings.ImageFallbackExt

	if run.Model.Media == media.Video {
		body, err = interactionVideoBody(&run.Model, prompt, run.Params, run.InputMedia)
		mediaBlockType, fallbackExt = string(media.Video), session.settings.VideoFallbackExt
	} else {
		body, err = interactionImageBody(&run.Model, prompt, run.Params, run.InputMedia)
	}

	if err != nil {
		return generation.Result{}, err
	}

	run.Record.Prepared(run.InputMedia)

	run.Record.SetFields([]metadata.BinaryField{
		{Path: []string{wireKeyInput, "*", interactionFieldData}, MIMEField: interactionFieldMIMEType},
	}, []metadata.BinaryField{
		{Path: []string{"steps", "*", "content", "*", interactionFieldData}, MIMEField: interactionFieldMIMEType}, //nolint:goconst // This is the provider's fixed JSON path, not shared product copy.
		{Path: []string{"steps", "*", "summary", "*", interactionFieldData}, MIMEField: interactionFieldMIMEType}, //nolint:goconst // This is the provider's fixed JSON path, not shared product copy.
	})

	status, resp, err := httpapi.PostJSON(ctx, session.settings.APIBase+interactionsPathSuffix, session.credential, body, run.Record)
	if err != nil {
		return generation.Result{}, err
	}

	if status/100 != 2 {
		return generation.Result{}, provider.APIErr(run.Model.ID, status, resp)
	}

	generatedMedia, thoughts, err := session.extractInteractionMedia(ctx, resp, mediaBlockType, fallbackExt, run.Record)
	if err != nil {
		return generation.Result{}, err
	}

	return generation.Result{
		Artifacts: []artifact.Media{generatedMedia},
		Thoughts:  thoughts,
	}, nil
}

// generateVeo submits a Veo operation, retains its references, waits for completion,
// and returns its downloaded artifacts.
func (session *apiSession) generateVeo(ctx context.Context, run *generation.Generation) (generation.Result, error) {
	downloadedInputs, err := httpapi.DownloadInputMedia(ctx, run.InputMedia)

	run.InputMedia = downloadedInputs
	if err != nil {
		return generation.Result{}, fmt.Errorf("%q, %w", run.Model.ID, err)
	}

	run.Record.Prepared(run.InputMedia)

	name, err := session.startOperation(ctx, &run.Model, run.Prompt, run.Params, run.InputMedia, run.ReuseURI, run.Record)
	if err != nil {
		return generation.Result{}, err
	}

	if err := retainOperation(run.Record, run.Model.ID, operation{Name: name}); err != nil {
		return generation.Result{}, err
	}

	completedOperation, err := session.pollOperation(ctx, name, run.Record)
	if err != nil {
		return generation.Result{}, err
	}

	if completedOperation.Name == "" {
		completedOperation.Name = name
	}

	if err := retainOperation(run.Record, run.Model.ID, completedOperation); err != nil {
		return generation.Result{}, err
	}

	run.Record.ProviderFinished(nil)

	artifacts, err := session.createArtifacts(ctx, completedOperation, session.settings.VideoFallbackExt, run.Record)
	if err != nil {
		return generation.Result{}, err
	}

	return generation.Result{Artifacts: artifacts}, nil
}

// startOperation sends a Veo generation request and returns its operation name.
func (session *apiSession) startOperation(ctx context.Context, model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input, reuseURI string, record *metadata.Record) (string, error) {
	url := session.settings.APIBase + modelsPathPrefix + model.ID + predictActionSuffix

	if record != nil {
		record.SetFields([]metadata.BinaryField{
			{Path: []string{wireKeyInstances, "*", veoFieldImage, wireKeyBytesBase64}, MIMEField: veoFieldMIMEType},
			{Path: []string{wireKeyInstances, "*", wireKeyLastFrame, wireKeyBytesBase64}, MIMEField: veoFieldMIMEType},
			{Path: []string{wireKeyInstances, "*", wireKeyReferenceImages, "*", veoFieldImage, wireKeyBytesBase64}, MIMEField: veoFieldMIMEType},
			{Path: []string{wireKeyInstances, "*", veoFieldVideo, wireKeyEncodedVideo}, MIMEField: wireKeyEncoding},
		}, []metadata.BinaryField{
			{Path: []string{"response", "generateVideoResponse", "generatedSamples", "*", veoFieldVideo, wireKeyEncodedVideo}, MIMEField: wireKeyEncoding}, //nolint:goconst // This is the provider's fixed JSON path, not shared product copy.
		})
	}

	status, body, err := httpapi.PostJSON(ctx, url, session.credential, requestBody(model, prompt, parameterValues, mediaInputs, reuseURI), record)
	if err != nil {
		return "", err
	}

	if status/100 != 2 {
		return "", provider.APIErr(model.ID, status, body)
	}

	var response operation
	if uErr := json.Unmarshal(body, &response); uErr != nil {
		return "", fmt.Errorf("%q, %w, %w", model.ID+": "+startResponseContext, errs.ErrResponseDecode, uErr)
	}

	if response.Name == "" {
		msg := fmt.Sprintf(VeoOperationNameMissing, model.ID)

		return "", fmt.Errorf("%q, %w", msg, errs.ErrResponseNoData)
	}

	return response.Name, nil
}

// pollOperation waits for a Veo operation to complete and returns its final response.
func (session *apiSession) pollOperation(ctx context.Context, name string, record *metadata.Record) (operation, error) {
	probe := &operationProbe{apiBase: session.settings.APIBase, credential: session.credential, name: name, record: record}
	if err := httpapi.Poll(ctx, session.settings.PollInterval.Duration(), session.settings.PollTimeout.Duration(), probe); err != nil {
		return operation{}, &errs.PollError{Model: session.model, Resource: name, Cause: err}
	}

	return probe.completed, nil
}

// createArtifacts returns the video artifacts in a completed operation.
// It removes any temporary artifact files when a sample fails.
func (session *apiSession) createArtifacts(ctx context.Context, completedOperation operation, fallbackExt string, record *metadata.Record) ([]artifact.Media, error) {
	samples, err := completedSamples(completedOperation, session.model)
	if err != nil {
		return nil, err
	}

	var artifacts []artifact.Media

	for i, s := range samples {
		if s.Video == nil {
			continue
		}

		generatedMedia, err := session.createArtifact(ctx, i, *s.Video, fallbackExt, record)
		if err != nil {
			return nil, errors.Join(err, artifact.Cleanup(artifacts))
		}

		artifacts = append(artifacts, generatedMedia)
	}

	return artifacts, nil
}

// createArtifact returns the artifact represented by a video response.
// It may download the video URI into a temporary file.
func (session *apiSession) createArtifact(ctx context.Context, i int, sampleVideo video, fallbackExt string, record *metadata.Record) (artifact.Media, error) {
	switch {
	case sampleVideo.EncodedVideo != "":
		data, err := base64.StdEncoding.DecodeString(sampleVideo.EncodedVideo)
		if err != nil {
			msg := fmt.Sprintf(VeoSampleBytesInvalid, session.model, i)

			return artifact.Media{}, fmt.Errorf("%q, %w, %w", msg, errs.ErrResponseDecode, err)
		}

		return artifact.New(data, sampleVideo.Encoding, fallbackExt)
	case sampleVideo.URI != "":
		return session.downloadArtifact(ctx, sampleVideo.URI, media.ExtForMimeOr(sampleVideo.Encoding, fallbackExt), record)
	}

	msg := fmt.Sprintf(VeoSampleEmpty, session.model, i)

	return artifact.Media{}, fmt.Errorf("%q, %w", msg, errs.ErrResponseNoData)
}

// extractInteractionMedia returns the first requested media artifact and each thought summary in a response body.
// It may download URI-backed media into a temporary file.
func (session *apiSession) extractInteractionMedia(ctx context.Context, body []byte, mediaBlockType, fallback string, record *metadata.Record) (artifact.Media, []string, error) {
	var response interactionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return artifact.Media{}, nil, fmt.Errorf("%q, %w, %w", session.model+": "+interactionResponseContext, errs.ErrResponseDecode, err)
	}

	var (
		thoughts   []string
		mediaBlock *interactionBlock
	)

	firstTextBlock := ""

	for si := range response.InteractionSteps {
		step := &response.InteractionSteps[si]
		switch {
		case step.Type == wireValueThought:
			for _, block := range step.Summary {
				if block.Type == interactionTextType && block.Text != "" {
					thoughts = append(thoughts, block.Text)
				}
			}
		case step.Type == wireValueModelOutput && step.Status == interactionStatusError:
			return artifact.Media{}, nil, newInteractionStepError(session.model, step.Error)
		case step.Type == wireValueModelOutput:
			mediaBlock, firstTextBlock = findMediaBlock(step.Content, mediaBlockType, mediaBlock, firstTextBlock)
		}
	}

	if mediaBlock == nil {
		msg := fmt.Sprintf(InteractionMediaMissing, session.model, mediaBlockType)
		if firstTextBlock != "" {
			msg += ": " + firstTextBlock
		}

		return artifact.Media{}, nil, fmt.Errorf("%q, %w", msg, errs.ErrResponseNoData)
	}

	record.ProviderFinished(nil)

	generatedMedia, err := session.createBlockArtifact(ctx, mediaBlock, fallback, record)
	if err != nil {
		return artifact.Media{}, nil, err
	}

	return generatedMedia, thoughts, nil
}

// createBlockArtifact returns the artifact represented by a media block.
// It may wait for and download a file-backed media resource.
func (session *apiSession) createBlockArtifact(ctx context.Context, block *interactionBlock, fallback string, record *metadata.Record) (artifact.Media, error) {
	if block.Data != "" {
		data, err := base64.StdEncoding.DecodeString(block.Data)
		if err != nil {
			msg := session.model + ": " + block.Type + " " + blockDataContext

			return artifact.Media{}, fmt.Errorf("%q, %w, %w", msg, errs.ErrResponseDecode, err)
		}

		return artifact.New(data, block.MimeType, fallback)
	}

	fileResource, err := canonicalFileID(block.URI)
	if err != nil {
		return artifact.Media{}, err
	}

	if err := httpapi.Poll(ctx, session.settings.FilePollInterval.Duration(), session.settings.FilePollTimeout.Duration(), &fileReadyProbe{apiBase: session.settings.APIBase, credential: session.credential, fileResource: fileResource, record: record}); err != nil {
		return artifact.Media{}, &errs.PollError{Model: session.model, Resource: fileResource, Cause: err}
	}

	return session.downloadArtifact(ctx, session.settings.APIBase+"/"+fileResource+downloadQuerySuffix, media.ExtForMimeOr(block.MimeType, fallback), record)
}

// downloadArtifact downloads a URL and returns a file-backed artifact.
// It sends the credential only when the URL and API base have the same origin.
func (session *apiSession) downloadArtifact(ctx context.Context, endpoint, fallback string, record *metadata.Record) (artifact.Media, error) {
	return httpapi.Fetch(ctx, endpoint, httpapi.CredentialForURL(endpoint, session.settings.APIBase, session.credential), fallback, record)
}
