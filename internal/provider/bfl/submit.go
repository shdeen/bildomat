package bfl

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
	"github.com/shdeen/bildomat/internal/provider"
)

// The BFL adapter's headers, request fields, modes, and error contexts.
//   - keyHeader: the credential header
//   - fieldMode: the request field selecting the generation mode
//   - fieldPrompt: the request field carrying the prompt
//   - fieldWidth: the request field carrying the width
//   - fieldHeight: the request field carrying the height
//   - fieldStartVideo: the request field carrying the continuation video
//   - fieldKeyframes: the request field carrying the keyframe array
//   - fieldInputImage: the base name of the indexed input-image fields
//   - fieldImages: the ordered image reference array
//   - fieldVideo: the video editing input
//   - fieldInputVideo: the video upscaling input
//   - modeVideoContinuation: the mode value continuing a video
//   - modeTextToVideo: the mode value generating from the prompt alone
//   - modeImageToVideo: the mode value generating from keyframe images
//   - submitResponseContext: names the submit response in a failed-decode context
const (
	keyHeader = "x-key"

	fieldMode       = "mode"
	fieldPrompt     = "prompt"
	fieldWidth      = "width"
	fieldHeight     = "height"
	fieldStartVideo = "start_video"
	fieldKeyframes  = "keyframes"
	fieldInputImage = "input_image"
	fieldImages     = "images"
	fieldVideo      = "video"
	fieldInputVideo = "input_video"

	modeVideoContinuation = "v2v"
	modeTextToVideo       = "t2v"
	modeImageToVideo      = "i2v"

	submitResponseContext = "submit response"
)

// submitAck contains the identifiers returned for an asynchronous generation job.
type submitAck struct {
	// ID is the generation job identifier.
	ID string `json:"id"`
	// PollURL is the endpoint used to inspect the job.
	PollURL string `json:"polling_url"`
}

// startJob sends an image or video job request and returns its identifier and polling URL.
func startJob(ctx context.Context, base string, cred httpapi.AuthCredential, provModelLabel string, model *catalog.Model, prompt string, gp params.Values, inputs []media.Input, record *metadata.Record) (jobID, pollURL string, err error) {
	reqBody, err := requestBody(model, prompt, gp, inputs)
	if err != nil {
		return "", "", fmt.Errorf("%q, %w", provModelLabel, err)
	}

	record.Prepared(inputs)

	if record != nil {
		record.SetFields(requestBinaryFields(model, inputs), nil)
	}

	status, body, err := httpapi.PostJSON(ctx, base+"/"+model.ID, cred, reqBody, record)
	if err != nil {
		return "", "", fmt.Errorf("%q, %w, %w", provModelLabel, errs.ErrTransport, err)
	}

	if status/100 != 2 {
		return "", "", provider.APIErr(provModelLabel, status, body)
	}

	var submitResp submitAck
	if uErr := json.Unmarshal(body, &submitResp); uErr != nil {
		return "", "", fmt.Errorf("%q, %w, %w", provModelLabel+": "+submitResponseContext, errs.ErrResponseDecode, uErr)
	}

	if submitResp.PollURL == "" {
		return "", "", fmt.Errorf("%q, %w", fmt.Sprintf(PollURLMissing, provModelLabel), errs.ErrResponseNoData)
	}

	id := submitResp.ID
	if id == "" {
		id = provModelLabel
	}

	return id, submitResp.PollURL, nil
}

// requestBody returns the request fields for an image or video job. It omits the prompt field when
// the supplied prompt is empty.
func requestBody(model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input) (map[string]any, error) {
	requestDocument := map[string]any{}
	if prompt != "" {
		requestDocument[fieldPrompt] = prompt
	}

	sizeValue, err := params.Value[string](parameterValues, params.FlagTypeSize)
	if err != nil {
		return nil, err
	}

	if sizeValue != "" {
		if width, height, ok := parseWidthHeight(sizeValue); ok {
			requestDocument[fieldWidth] = width
			requestDocument[fieldHeight] = height
		}
	}

	maps.Copy(requestDocument, provider.WireParamValues(model, parameterValues))

	inputDefinition, inputDeclared := model.Param(params.FlagTypeInputMedia)
	if inputDeclared && isVideoToolInput(inputDefinition.ParamID) {
		if err := addVideoToolInput(requestDocument, inputDefinition.ParamID, mediaInputs); err != nil {
			return nil, err
		}

		return requestDocument, nil
	}

	if len(mediaInputs) == 0 {
		if model.Media == media.Video {
			requestDocument[fieldMode] = modeTextToVideo
		}

		return requestDocument, nil
	}

	if model.Media == media.Video {
		err = addVideoInputs(requestDocument, mediaInputs, parameterValues)
	} else {
		err = addImageInputs(requestDocument, model, mediaInputs)
	}

	if err != nil {
		return nil, err
	}

	return requestDocument, nil
}

// requestBinaryFields describes the encoded media locations used by BFL's image and video request
// shapes. URL values at these locations remain URLs.
func requestBinaryFields(model *catalog.Model, mediaInputs []media.Input) []metadata.BinaryField {
	binaryFields := make([]metadata.BinaryField, 0, len(mediaInputs))
	// An undeclared input uses the zero-value ID, selecting the indexed image fields below.
	inputDefinition, _ := model.Param(params.FlagTypeInputMedia)

	for inputIndex, mediaInput := range mediaInputs {
		if isVideoToolInput(inputDefinition.ParamID) {
			binaryFields = append(binaryFields, metadata.BinaryField{Path: []string{inputDefinition.ParamID}, MIME: mediaInput.MIME})

			continue
		}

		if model.Media != media.Video {
			fieldName := inputDefinition.ParamID
			if fieldName == "" {
				fieldName = indexedInputMediaParam(inputIndex)
			}

			fieldPath := []string{fieldName}
			if fieldName == fieldImages {
				fieldPath = append(fieldPath, strconv.Itoa(inputIndex))
			}

			binaryFields = append(binaryFields, metadata.BinaryField{Path: fieldPath, MIME: mediaInput.MIME})

			continue
		}

		if mediaInput.Kind() == media.Video {
			binaryFields = append(binaryFields, metadata.BinaryField{Path: []string{fieldStartVideo}, MIME: mediaInput.MIME})

			continue
		}

		// BFL accepts an image string or a pair containing time and image.
		binaryFields = append(binaryFields,
			metadata.BinaryField{Path: []string{fieldKeyframes, strconv.Itoa(inputIndex)}, MIME: mediaInput.MIME},
			metadata.BinaryField{Path: []string{fieldKeyframes, strconv.Itoa(inputIndex), "1"}, MIME: mediaInput.MIME})
	}

	return binaryFields
}

// addVideoInputs writes continuation media or image keyframes and their mode into the body. It
// rejects mixed media, multiple videos, and timed continuation videos.
func addVideoInputs(requestDocument map[string]any, mediaInputs []media.Input, parameterValues params.Values) error {
	imageInputs, videoInputs := media.SplitInputs(mediaInputs)

	switch {
	case len(imageInputs) > 0 && len(videoInputs) > 0:
		return &errs.MediaError{Problem: VideoMediaMixed, Cause: errs.ErrInputMedia}
	case len(videoInputs) > 1:
		return &errs.MediaError{Problem: VideoOneContinuation, Cause: errs.ErrInputMedia}
	case len(videoInputs) == 1:
		if _, timed := videoInputs[0].FrameTime(); timed {
			return &errs.MediaError{Problem: VideoContinuationTimed, Cause: errs.ErrInputMediaTime}
		}

		requestDocument[fieldMode] = modeVideoContinuation
		requestDocument[fieldStartVideo] = videoInputs[0].URLOrBase64()

		return nil
	}

	keyframes, err := buildKeyframes(imageInputs, parameterValues)
	if err != nil {
		return err
	}

	requestDocument[fieldMode] = modeImageToVideo
	requestDocument[fieldKeyframes] = keyframes

	return nil
}

// addImageInputs writes an image array, a single declared field, or indexed fields. It rejects
// videos, timed inputs, and multiple inputs for a single field.
func addImageInputs(requestDocument map[string]any, model *catalog.Model, mediaInputs []media.Input) error {
	for _, mediaInput := range mediaInputs {
		if mediaInput.Kind() == media.Video {
			return &errs.MediaError{Problem: ImageGotVideo, Cause: errs.ErrInputMedia}
		}

		if _, timed := mediaInput.FrameTime(); timed {
			return &errs.MediaError{Problem: ImageGotTimed, Cause: errs.ErrInputMediaTime}
		}
	}

	inputDefinition, _ := model.Param(params.FlagTypeInputMedia)
	if inputDefinition.ParamID == fieldImages {
		imageReferences := make([]string, 0, len(mediaInputs))
		for _, mediaInput := range mediaInputs {
			imageReferences = append(imageReferences, mediaInput.URLOrBase64())
		}

		requestDocument[fieldImages] = imageReferences

		return nil
	}

	if inputDefinition.ParamID != "" {
		if len(mediaInputs) != 1 {
			return &errs.MediaError{Problem: fmt.Sprintf(SingleInputOnly, inputDefinition.ParamID), Cause: errs.ErrInputMedia}
		}

		requestDocument[inputDefinition.ParamID] = mediaInputs[0].URLOrBase64()

		return nil
	}

	for inputIndex, mediaInput := range mediaInputs {
		requestDocument[indexedInputMediaParam(inputIndex)] = mediaInput.URLOrBase64()
	}

	return nil
}

// indexedInputMediaParam returns the request field name for a zero-based image index.
func indexedInputMediaParam(i int) string {
	baseName := fieldInputImage

	if i == 0 {
		return baseName
	}

	baseName = baseName + "_" + strconv.Itoa(i+1)

	return baseName
}

// parseWidthHeight returns the positive width and height represented by a size value.
func parseWidthHeight(size string) (width, height int, valid bool) {
	// BFL accepts lower-case x without surrounding component whitespace.
	widthText, heightText, separated := strings.Cut(size, "x")
	if !separated || strings.TrimSpace(widthText) != widthText || strings.TrimSpace(heightText) != heightText {
		return 0, 0, false
	}

	return params.ParseDimensions(size)
}

// isVideoToolInput identifies the declared fields that accept a whole video without a mode.
func isVideoToolInput(parameterID string) bool {
	return parameterID == fieldVideo || parameterID == fieldInputVideo
}

// addVideoToolInput writes one whole video at the model's declared input field.
func addVideoToolInput(requestDocument map[string]any, inputField string, mediaInputs []media.Input) error {
	if len(mediaInputs) == 0 {
		return &errs.MediaError{Problem: VideoInputRequired, Cause: errs.ErrInputMedia}
	}

	if len(mediaInputs) != 1 {
		return &errs.MediaError{Problem: fmt.Sprintf(SingleInputOnly, inputField), Cause: errs.ErrInputMedia}
	}

	referenceVideo := mediaInputs[0]
	// URLs remain unidentified until the command has credentials to inspect them.
	unresolvedURL := referenceVideo.URL != "" && referenceVideo.MIME == ""
	if !unresolvedURL && referenceVideo.Kind() != media.Video {
		return &errs.MediaError{Source: referenceVideo.Source(), Cause: errs.ErrInputMediaMIME}
	}

	if referenceVideo.HasFrame() {
		return &errs.MediaError{Source: referenceVideo.Source(), Cause: errs.ErrInputMediaTime}
	}

	requestDocument[inputField] = referenceVideo.URLOrBase64()

	return nil
}
