package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
	"github.com/shdeen/bildomat/internal/provider"
)

// Veo instance fields identify media and prompt inputs.
//   - veoFieldMIMEType: the image MIME type
//   - veoFieldImage: an image reference or opening frame
//   - veoFieldVideo: a video input
//   - veoFieldPrompt: the generation prompt
const (
	veoFieldMIMEType = "mimeType"
	veoFieldImage    = "image"
	veoFieldVideo    = "video"
	veoFieldPrompt   = "prompt"
)

// The forced-duration rule: the resolutions that force the duration, and the duration Veo requires
// under them or with reference images.
//   - resolution1080p: full HD output
//   - resolution4K: 4K output
//   - forcedDurationSeconds: the required duration, in seconds
const (
	resolution1080p = "1080p"
	resolution4K    = "4k"

	forcedDurationSeconds = 8
)

// startResponseContext names the operation start response in a failed-decode context.
const startResponseContext = "start response"

// adjustVeoDuration forces eight seconds for any media input or high-resolution output. It updates
// the supplied parameter map and returns a change record, or an error for an incompatible
// resolution or duration value.
func adjustVeoDuration(parameterValues params.Values, inputs []media.Input) ([]params.Adjustment, error) {
	adjustedRes, err := params.Value[string](parameterValues, params.FlagTypeResolution)
	if err != nil {
		return nil, err
	}

	adjustmentTrigger := ""

	switch {
	case len(inputs) > 0:
		adjustmentTrigger = ReasonReferenceImages
	case adjustedRes == resolution1080p || adjustedRes == resolution4K:
		adjustmentTrigger = adjustedRes
	}

	if adjustmentTrigger == "" {
		return nil, nil
	}

	paramInput := ""

	if durationVal, durationSet := parameterValues[params.FlagTypeDuration]; durationSet {
		secs, err := params.Value[int](parameterValues, params.FlagTypeDuration)
		if err != nil {
			return nil, err
		}

		if secs == forcedDurationSeconds {
			return nil, nil
		}

		paramInput = params.FormatValue(durationVal)
	}

	parameterValues[params.FlagTypeDuration] = forcedDurationSeconds

	return []params.Adjustment{{
		FlagID: params.FlagTypeDuration, Type: params.ChangeForced,
		InputVal: paramInput, WireVal: params.FormatValue(forcedDurationSeconds), Comment: adjustmentTrigger,
	}}, nil
}

// operation contains the state and result of a Veo operation.
type operation struct {
	// Name is the operation resource name.
	Name string `json:"name"`
	// Done reports whether the operation has finished.
	Done bool `json:"done"`
	// Error contains a terminal operation failure.
	Error *opError `json:"error"`
	// Response contains the completed operation result.
	Response *result `json:"response"`
}

// opError contains a terminal operation failure.
type opError struct {
	// Code is the provider's error code.
	Code any `json:"code"`
	// Message is the provider's error message.
	Message string `json:"message"`
}

// result contains a completed operation response.
type result struct {
	// GenerateVideoResponse contains generated videos and filter reasons.
	GenerateVideoResponse *videos `json:"generateVideoResponse"`
}

// videos contains generated samples and safety filter reasons.
type videos struct {
	// GeneratedSamples contains the generated video samples.
	GeneratedSamples []sample `json:"generatedSamples"`
	// RaiMediaFilteredReasons contains explanations for filtered media.
	RaiMediaFilteredReasons []string `json:"raiMediaFilteredReasons"`
}

// sample contains one generated video sample.
type sample struct {
	// Video contains the sample's video data or location.
	Video *video `json:"video"`
}

// video contains an inline video or a video download location.
type video struct {
	// URI is the video download location.
	URI string `json:"uri"`
	// EncodedVideo contains base64-encoded video data.
	EncodedVideo string `json:"encodedVideo"`
	// Encoding identifies the video's MIME type.
	Encoding string `json:"encoding"`
}

// The Veo API's endpoint and request field tokens.
//   - modelsPathPrefix: the start-request path before the model ID
//   - predictActionSuffix: the start-request action after the model ID
//   - wireKeyLastFrame: the instance field carrying the closing frame image
//   - wireKeyReferenceType: the reference field naming a reference's kind
//   - wireKeyReferenceImages: the instance field carrying the reference images
//   - wireKeyInstances: the request field carrying the instances
//   - wireKeyParameters: the request field carrying the generation parameters
//   - wireKeyGcsURI: the image object's remote location field
//   - wireKeyBytesBase64: the image object's inline data field
//   - wireKeyEncodedVideo: the video object's inline data field
//   - wireKeyEncoding: the video object's MIME type field
//   - wireValueAsset: the reference kind of an asset image
const (
	modelsPathPrefix    = "/models/"
	predictActionSuffix = ":predictLongRunning"

	wireKeyLastFrame       = "lastFrame"
	wireKeyReferenceType   = "referenceType"
	wireKeyReferenceImages = "referenceImages"
	wireKeyInstances       = "instances"
	wireKeyParameters      = "parameters"
	wireKeyGcsURI          = "gcsUri"
	wireKeyBytesBase64     = "bytesBase64Encoded"
	wireKeyEncodedVideo    = "encodedVideo"
	wireKeyEncoding        = "encoding"

	wireValueAsset = "asset"
)

// requestBody returns the request fields for a Veo operation.
func requestBody(model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input, reuseURI string) map[string]any {
	inst := map[string]any{veoFieldPrompt: prompt}
	if reuseURI != "" {
		// Reuse preserves Google's original reference. The video has already been
		// generated; it needs neither input-media discovery nor encoding.
		inst[veoFieldVideo] = map[string]any{wireKeyURI: reuseURI}
	} else if len(mediaInputs) > 0 {
		placeInstanceMedia(inst, model, mediaInputs)
	}

	body := map[string]any{wireKeyInstances: []any{inst}}

	wireParams := provider.WireParamValues(model, parameterValues)

	if len(wireParams) > 0 {
		body[wireKeyParameters] = wireParams
	}

	return body
}

// placeInstanceMedia writes video or image inputs into the supplied request instance.
func placeInstanceMedia(inst map[string]any, model *catalog.Model, mediaInputs []media.Input) {
	imageInputs, videoInputs := media.SplitInputs(mediaInputs)
	if len(videoInputs) == 1 {
		inst[veoFieldVideo] = inputVideoObject(&videoInputs[0])

		return
	}

	placeOrdinaryImages(inst, model, placeFrameImages(inst, imageInputs))
}

// placeFrameImages writes opening and closing frame images into the supplied instance and returns
// unanchored images in input order.
func placeFrameImages(inst map[string]any, imageInputs []media.Input) []media.Input {
	ordinaryImages := make([]media.Input, 0, len(imageInputs))
	for imageIndex := range imageInputs {
		imageInput := &imageInputs[imageIndex]

		switch imageInput.FrameAnchor {
		case media.FrameFirst:
			inst[veoFieldImage] = inputMediaObject(imageInput)
		case media.FrameLast:
			inst[wireKeyLastFrame] = inputMediaObject(imageInput)
		default:
			ordinaryImages = append(ordinaryImages, *imageInput)
		}
	}

	return ordinaryImages
}

// placeOrdinaryImages writes unanchored images into the supplied request instance. A single image
// uses the image field only for single-input models with no opening frame; otherwise the images
// become asset references.
func placeOrdinaryImages(inst map[string]any, model *catalog.Model, ordinaryImages []media.Input) {
	if len(ordinaryImages) == 0 {
		return
	}

	inputMediaCfg, _ := model.Param(params.FlagTypeInputMedia)
	if len(ordinaryImages) == 1 && inputMediaCfg.MaxMultiple == 1 && inst[veoFieldImage] == nil {
		inst[veoFieldImage] = inputMediaObject(&ordinaryImages[0])

		return
	}

	refs := make([]map[string]any, 0, len(ordinaryImages))
	for imageIndex := range ordinaryImages {
		refs = append(refs, map[string]any{veoFieldImage: inputMediaObject(&ordinaryImages[imageIndex]), wireKeyReferenceType: wireValueAsset})
	}

	inst[wireKeyReferenceImages] = refs
}

// inputVideoObject returns a video URI or base64 bytes with the input MIME type.
func inputVideoObject(mediaInput *media.Input) map[string]any {
	videoObject := map[string]any{wireKeyEncoding: mediaInput.MIME}
	if mediaInput.URL != "" {
		videoObject[wireKeyURI] = mediaInput.URL
	} else {
		videoObject[wireKeyEncodedVideo] = base64.StdEncoding.EncodeToString(mediaInput.Bytes)
	}

	return videoObject
}

// inputMediaObject returns an image's local bytes or remote URI encoded for a Veo request.
func inputMediaObject(mediaInput *media.Input) map[string]any {
	mediaObject := map[string]any{veoFieldMIMEType: mediaInput.MIME}
	if mediaInput.URL != "" {
		mediaObject[wireKeyGcsURI] = mediaInput.URL
	} else {
		mediaObject[wireKeyBytesBase64] = base64.StdEncoding.EncodeToString(mediaInput.Bytes)
	}

	return mediaObject
}

// operationProbe holds the values and completed result for a Veo operation.
//   - apiBase: the base endpoint for operation status requests
//   - credential: the credential sent with each status request
//   - name: the operation resource name
//   - completed: the completed operation response
//   - record: optional retention of every operation response
type operationProbe struct {
	apiBase    string
	credential httpapi.AuthCredential
	name       string
	completed  operation
	record     *metadata.Record
}

// Poll checks the Veo operation and stores its completed response.
func (probe *operationProbe) Poll(ctx context.Context) (complete bool, err error) {
	status, body, err := httpapi.GetAuth(ctx, probe.apiBase+"/"+probe.name, probe.credential, metadata.Asynchronous, probe.record)
	if err := provider.PollResponseError(probe.name, status, body, err); err != nil {
		return false, err
	}

	var response operation
	if uErr := json.Unmarshal(body, &response); uErr != nil {
		return false, fmt.Errorf("%q, %w, %w", probe.name+": "+provider.PollResponseContext, errs.ErrResponseDecode, uErr)
	}

	if response.Error != nil {
		msg := fmt.Sprintf(ErrorCodeForm, probe.name, response.Error.Code, response.Error.Message)

		return false, &errs.ProviderError{Message: msg, Cause: errs.ErrResponseGen}
	}

	if !response.Done {
		return false, nil
	}

	if _, err := completedSamples(response, probe.name); err != nil {
		return false, err
	}

	probe.completed = response

	return true, nil
}

// completedSamples validates required output while preserving sample-specific errors and
// safety-filter explanations from a completed operation.
func completedSamples(completed operation, identity string) ([]sample, error) {
	var (
		samples       []sample
		filterReasons []string
	)

	if completed.Response != nil && completed.Response.GenerateVideoResponse != nil {
		samples = completed.Response.GenerateVideoResponse.GeneratedSamples
		filterReasons = completed.Response.GenerateVideoResponse.RaiMediaFilteredReasons
	}

	videoCount := 0

	for sampleIndex, sample := range samples {
		if sample.Video == nil {
			continue
		}

		if sample.Video.EncodedVideo == "" && sample.Video.URI == "" {
			return nil, fmt.Errorf("%q, %w", fmt.Sprintf(VeoSampleEmpty, identity, sampleIndex), errs.ErrResponseNoData)
		}

		videoCount++
	}

	if videoCount == 0 {
		message := fmt.Sprintf(VeoVideoMissing, identity)
		if len(filterReasons) > 0 {
			message += ": " + strings.Join(filterReasons, "; ")
		}

		return nil, fmt.Errorf("%q, %w", message, errs.ErrResponseNoData)
	}

	return samples, nil
}
