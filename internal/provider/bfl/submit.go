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
//   - wireKeyMode: the request field selecting the generation mode
//   - wireKeyPrompt: the request field carrying the prompt
//   - wireKeyWidth: the request field carrying the width
//   - wireKeyHeight: the request field carrying the height
//   - wireKeyStartVideo: the request field carrying the continuation video
//   - wireKeyKeyframes: the request field carrying the keyframe array
//   - wireKeyInputImage: the base name of the indexed input-image fields
//   - modeVideoContinuation: the mode value continuing a video
//   - modeTextToVideo: the mode value generating from the prompt alone
//   - modeImageToVideo: the mode value generating from keyframe images
//   - submitResponseContext: names the submit response in a failed-decode context
const (
	keyHeader = "x-key"

	wireKeyMode       = "mode"
	wireKeyPrompt     = "prompt"
	wireKeyWidth      = "width"
	wireKeyHeight     = "height"
	wireKeyStartVideo = "start_video"
	wireKeyKeyframes  = "keyframes"
	wireKeyInputImage = "input_image"

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
func requestBody(model *catalog.Model, prompt string, parameterValues params.Values, inputs []media.Input) (map[string]any, error) {
	body := map[string]any{}
	if prompt != "" {
		body[wireKeyPrompt] = prompt
	}

	sizeVal, err := params.Value[string](parameterValues, params.FlagTypeSize)
	if err != nil {
		return nil, err
	}

	if sizeVal != "" {
		if width, h, ok := parseWidthHeight(sizeVal); ok {
			body[wireKeyWidth] = width
			body[wireKeyHeight] = h
		}
	}

	maps.Copy(body, provider.WireParamValues(model, parameterValues))

	if len(inputs) == 0 {
		if model.Media == media.Video {
			body[wireKeyMode] = modeTextToVideo
		}

		return body, nil
	}

	if model.Media == media.Video {
		err = addVideoInputs(body, inputs, parameterValues)
	} else {
		err = addImageInputs(body, model, inputs)
	}

	if err != nil {
		return nil, err
	}

	return body, nil
}

// requestBinaryFields describes the encoded media locations used by BFL's image and video request
// shapes. URL values at these locations remain URLs.
func requestBinaryFields(model *catalog.Model, inputs []media.Input) []metadata.BinaryField {
	fields := make([]metadata.BinaryField, 0, len(inputs))
	inputConfig, _ := model.Param(params.FlagTypeInputMedia)

	for index, input := range inputs {
		if model.Media != media.Video {
			name := inputConfig.ParamID
			if name == "" {
				name = indexedInputMediaParam(index)
			}

			fields = append(fields, metadata.BinaryField{Path: []string{name}, MIME: input.MIME})

			continue
		}

		if input.Kind() == media.Video {
			fields = append(fields, metadata.BinaryField{Path: []string{wireKeyStartVideo}, MIME: input.MIME})

			continue
		}

		// BFL accepts an image string or a pair containing time and image.
		fields = append(fields,
			metadata.BinaryField{Path: []string{wireKeyKeyframes, strconv.Itoa(index)}, MIME: input.MIME},
			metadata.BinaryField{Path: []string{wireKeyKeyframes, strconv.Itoa(index), "1"}, MIME: input.MIME})
	}

	return fields
}

// addVideoInputs writes continuation media or image keyframes and their mode into the body. It
// rejects mixed media, multiple videos, and timed continuation videos.
func addVideoInputs(body map[string]any, inputs []media.Input, parameterValues params.Values) error {
	imageInputs, videoInputs := media.SplitInputs(inputs)

	switch {
	case len(imageInputs) > 0 && len(videoInputs) > 0:
		return &errs.MediaError{Problem: VideoMediaMixed, Cause: errs.ErrInputMedia}
	case len(videoInputs) > 1:
		return &errs.MediaError{Problem: VideoOneContinuation, Cause: errs.ErrInputMedia}
	case len(videoInputs) == 1:
		if _, timed := videoInputs[0].FrameTime(); timed {
			return &errs.MediaError{Problem: VideoContinuationTimed, Cause: errs.ErrInputMediaTime}
		}

		body[wireKeyMode] = modeVideoContinuation
		body[wireKeyStartVideo] = videoInputs[0].URLOrBase64()

		return nil
	}

	keyframes, err := buildKeyframes(imageInputs, parameterValues)
	if err != nil {
		return err
	}

	body[wireKeyMode] = modeImageToVideo
	body[wireKeyKeyframes] = keyframes

	return nil
}

// addImageInputs writes images into the model's single declared field or indexed fields. It rejects
// videos, timed inputs, and multiple inputs for a single field.
func addImageInputs(body map[string]any, model *catalog.Model, inputs []media.Input) error {
	for _, mediaInput := range inputs {
		if mediaInput.Kind() == media.Video {
			return &errs.MediaError{Problem: ImageGotVideo, Cause: errs.ErrInputMedia}
		}

		if _, timed := mediaInput.FrameTime(); timed {
			return &errs.MediaError{Problem: ImageGotTimed, Cause: errs.ErrInputMediaTime}
		}
	}

	inputCfg, _ := model.Param(params.FlagTypeInputMedia)
	if inputCfg.ParamID != "" {
		if len(inputs) != 1 {
			return &errs.MediaError{Problem: fmt.Sprintf(SingleInputOnly, inputCfg.ParamID), Cause: errs.ErrInputMedia}
		}

		body[inputCfg.ParamID] = inputs[0].URLOrBase64()

		return nil
	}

	for inputIndex, mediaInput := range inputs {
		body[indexedInputMediaParam(inputIndex)] = mediaInput.URLOrBase64()
	}

	return nil
}

// indexedInputMediaParam returns the request field name for a zero-based image index.
func indexedInputMediaParam(i int) string {
	baseName := wireKeyInputImage

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
