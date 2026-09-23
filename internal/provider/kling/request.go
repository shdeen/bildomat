package kling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
	"github.com/shdeen/bildomat/internal/provider"
)

// Kling content fields identify prompts and media.
//   - fieldPrompt: the generation prompt
//   - fieldImage: an input image reference
//   - fieldType: the content kind
//   - fieldText: prompt content text
//   - fieldURL: the media location
//   - contentPrompt: the prompt content kind
const (
	fieldPrompt   = "prompt"
	fieldImage    = "image"
	fieldType     = "type"
	fieldText     = "text"
	fieldURL      = "url"
	contentPrompt = "prompt" //nolint:goconst // The content type and prompt field name are independent protocol values.
)

// familyOmni identifies models that use the omni request routes.
const familyOmni = "omni"

// The creation paths, one per Kling product. A video path takes the escaped model ID as its final
// segment.
//   - imagePath: standard image generation
//   - omniImagePath: omni image generation
//   - omniVideoPath: omni video generation
//   - textToVideoPath: video generation from the prompt alone
//   - imageToVideoPath: video generation from frame images
const (
	imagePath        = "/v1/images/generations"
	omniImagePath    = "/v1/images/omni-image"
	omniVideoPath    = "/omni-video/"
	textToVideoPath  = "/text-to-video/"
	imageToVideoPath = "/image-to-video/"
)

// The Kling request fields, content types, and task identifiers.
//   - fieldModelName: the image request field naming the model
//   - fieldSettings: the video request's parameter object
//   - fieldContents: the video request's content list
//   - contentReferImage: the content type of an unanchored reference image
//   - contentFirstFrame: the content type of the opening frame
//   - contentLastFrame: the content type of the closing frame
const (
	fieldModelName = "model_name"
	fieldSettings  = "settings"
	fieldContents  = "contents"

	contentReferImage = "refer_image"
	contentFirstFrame = "first_frame"
	contentLastFrame  = "last_frame"
)

// buildRequestBody returns the creation path and JSON body for one generation.
func buildRequestBody(generationRequest *generation.Generation) (creationPath string, requestBody map[string]any) {
	omniRequest := generationRequest.Model.Family == familyOmni

	creationPath = selectCreationPath(&generationRequest.Model, len(generationRequest.InputMedia))
	if generationRequest.Model.Media == media.Image {
		return creationPath, imageRequestBody(generationRequest, omniRequest)
	}

	return creationPath, videoRequestBody(generationRequest, omniRequest)
}

// selectCreationPath selects a documented creation path from a model's medium and family and its
// retained input count. It returns an empty path for an unsupported medium.
func selectCreationPath(model *catalog.Model, mediaCount int) string {
	if model == nil {
		return ""
	}

	omniModel := model.Family == familyOmni

	switch model.Media {
	case media.Image:
		if omniModel {
			return omniImagePath
		}

		return imagePath
	case media.Video:
		modelPath := url.PathEscape(model.ID)
		if omniModel {
			return omniVideoPath + modelPath
		}

		if mediaCount == 0 {
			return textToVideoPath + modelPath
		}

		return imageToVideoPath + modelPath
	default:
		return ""
	}
}

// imageRequestBody returns the standard or omni image request fields.
func imageRequestBody(generationRequest *generation.Generation, omniRequest bool) map[string]any {
	requestBody := map[string]any{
		fieldModelName: generationRequest.Model.ID,
		fieldPrompt:    generationRequest.Prompt,
	}
	maps.Copy(requestBody, provider.WireParamValues(&generationRequest.Model, generationRequest.Params))

	if len(generationRequest.InputMedia) == 0 {
		return requestBody
	}

	inputDefinition, _ := generationRequest.Model.Param(params.FlagTypeInputMedia)
	if !omniRequest {
		requestBody[inputDefinition.ParamID] = generationRequest.InputMedia[0].URLOrBase64()

		return requestBody
	}

	imageObjects := make([]any, 0, len(generationRequest.InputMedia))
	for mediaIndex := range generationRequest.InputMedia {
		imageObjects = append(imageObjects, map[string]any{
			fieldImage: generationRequest.InputMedia[mediaIndex].URLOrBase64(),
		})
	}

	requestBody[inputDefinition.ParamID] = imageObjects

	return requestBody
}

// videoRequestBody returns a text, image, or omni video request body.
func videoRequestBody(generationRequest *generation.Generation, omniRequest bool) map[string]any {
	settings := provider.WireParamValues(&generationRequest.Model, generationRequest.Params)
	if len(generationRequest.InputMedia) == 0 && !omniRequest {
		return map[string]any{
			fieldPrompt:   generationRequest.Prompt,
			fieldSettings: settings,
		}
	}

	contents := []any{map[string]any{
		fieldType: contentPrompt,
		fieldText: generationRequest.Prompt,
	}}
	if omniRequest {
		contents = append(contents, omniContents(generationRequest.InputMedia)...)
	} else {
		contents = append(contents, frameContents(generationRequest.InputMedia)...)
	}

	return map[string]any{
		fieldContents: contents,
		fieldSettings: settings,
	}
}

// omniContents returns anchored frames and unanchored reference images in their retained order.
func omniContents(mediaInputs []media.Input) []any {
	contents := make([]any, 0, len(mediaInputs))
	for mediaIndex := range mediaInputs {
		mediaInput := &mediaInputs[mediaIndex]
		contentType := contentReferImage

		switch mediaInput.FrameAnchor {
		case media.FrameFirst:
			contentType = contentFirstFrame
		case media.FrameLast:
			contentType = contentLastFrame
		}

		contents = append(contents, map[string]any{
			fieldType: contentType,
			fieldURL:  mediaInput.URLOrBase64(),
		})
	}

	return contents
}

// frameContents maps at most two images onto opening and closing frames. Explicit anchors keep
// their roles; unanchored images fill the free roles in retained order.
func frameContents(mediaInputs []media.Input) []any {
	firstFrameAssigned, lastFrameAssigned := false, false

	for mediaIndex := range mediaInputs {
		switch mediaInputs[mediaIndex].FrameAnchor {
		case media.FrameFirst:
			firstFrameAssigned = true
		case media.FrameLast:
			lastFrameAssigned = true
		}
	}

	contents := make([]any, 0, len(mediaInputs))
	for mediaIndex := range mediaInputs {
		mediaInput := &mediaInputs[mediaIndex]
		contentType := ""

		switch mediaInput.FrameAnchor {
		case media.FrameFirst:
			contentType = contentFirstFrame
		case media.FrameLast:
			contentType = contentLastFrame
		case "":
			if !firstFrameAssigned {
				contentType = contentFirstFrame
				firstFrameAssigned = true
			} else if !lastFrameAssigned {
				contentType = contentLastFrame
				lastFrameAssigned = true
			}
		}

		if contentType == "" {
			continue
		}

		contents = append(contents, map[string]any{
			fieldType: contentType,
			fieldURL:  mediaInput.URLOrBase64(),
		})
	}

	return contents
}

// createTask submits one request and returns the task identifier.
func createTask(ctx context.Context, endpoint string, credential httpapi.AuthCredential, generationRequest *generation.Generation, requestBody map[string]any) (string, error) {
	providerModelName := generationRequest.Label()

	statusCode, responseBody, err := httpapi.PostJSON(ctx, endpoint, credential, requestBody, generationRequest.Record)
	if err != nil {
		return "", fmt.Errorf("%q: %w, %w", providerModelName, errs.ErrTransportRequest, err)
	}

	if statusCode/100 != 2 {
		return "", provider.APIErr(providerModelName, statusCode, responseBody)
	}

	responseDocument, err := decodeEnvelope(responseBody, providerModelName+": "+provider.CreationResponseContext)
	if err != nil {
		return "", err
	}

	if err := envelopeFailure(responseDocument, providerModelName); err != nil {
		return "", err
	}

	var (
		taskID    string
		decodeErr error
	)

	if generationRequest.Model.Media == media.Image {
		var response struct {
			TaskID string `json:"task_id"`
		}

		decodeErr = json.Unmarshal(responseDocument.Data, &response)
		taskID = response.TaskID
	} else {
		var response struct {
			ID string `json:"id"`
		}

		decodeErr = json.Unmarshal(responseDocument.Data, &response)
		taskID = response.ID
	}

	if decodeErr != nil || taskID == "" {
		return "", fmt.Errorf("%q, %w", fmt.Sprintf(TaskIDMissing, providerModelName), errors.Join(errs.ErrResponseNoData, decodeErr))
	}

	return taskID, nil
}
