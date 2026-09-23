package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"mime/multipart"
	"net/textproto"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// Shared request and response fields identify model, prompt, and media values.
//   - fieldMIMEType: the media MIME type
//   - fieldModel: the requested model ID
//   - fieldPrompt: the generation prompt
//   - fieldType: the media reference kind
//   - fieldURL: the media location
//   - fieldData: the response artifact list
//   - fieldImageURL: an image reference
//   - fieldVideoURL: a video reference
const (
	fieldMIMEType = "mime_type"
	fieldModel    = "model"
	fieldPrompt   = "prompt"
	fieldType     = "type"
	fieldURL      = "url"
	fieldData     = "data"
	fieldImageURL = "image_url"
	fieldVideoURL = "video_url"
)

// Multipart headers describe the submitted file and its MIME type.
//   - headerContentDisposition: the form field name and filename
//   - headerContentType: the file MIME type
const (
	headerContentDisposition = "Content-Disposition"
	headerContentType        = "Content-Type"
)

// The image response fields for base64 data and media type.
//   - fieldB64JSON: the field carrying an image's base64 data
//   - fieldMediaType: the field some providers carry a MIME type under
const (
	fieldB64JSON   = "b64_json"
	fieldMediaType = "media_type"
)

// partFilenameForm is the multipart file part's content-disposition value, completed by the field
// name, the part's index, and its extension.
const partFilenameForm = `form-data; name=%q; filename="image-%d%s"`

// jsonBody returns the JSON fields for an image generation request; an empty prompt sends no prompt
// field.
func jsonBody(api *catalog.ImageAPI, model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input, stringParams ...params.FlagType) (map[string]any, error) {
	body := map[string]any{fieldModel: model.ID}
	if prompt != "" {
		body[fieldPrompt] = prompt
	}

	maps.Copy(body, WireParamValues(model, parameterValues, stringParams...))

	for k, v := range api.FixedProvFields {
		body[k] = v
	}

	if err := addInputMedia(body, api.InputMediaStyle, api.InputMediaProvParam, api.InputMediaListProvParam, mediaInputs); err != nil {
		return nil, err
	}

	return body, nil
}

// WireParamValues returns supplied parameters under their configured request paths. It omits
// parameters without paths and formats the selected parameters as strings. Dotted paths create
// nested objects shared by parameters with the same parent.
func WireParamValues(model *catalog.Model, gp params.Values, stringParams ...params.FlagType) map[string]any {
	vals := map[string]any{}

	for param, paramVal := range gp {
		paramCfg, ok := model.Param(param)
		if !ok || paramCfg.ParamID == "" {
			continue
		}

		if slices.Contains(stringParams, param) {
			paramVal = params.FormatValue(paramVal)
		}

		placeWireValue(vals, paramCfg.ParamID, paramVal)
	}

	return vals
}

// placeWireValue writes a value into the supplied map at a dotted request path. It creates
// intermediate objects, replacing any non-object values along the path.
func placeWireValue(fields map[string]any, path string, value any) {
	segments := strings.Split(path, ".")
	holder := fields

	for _, segment := range segments[:len(segments)-1] {
		nested, ok := holder[segment].(map[string]any)
		if !ok {
			nested = map[string]any{}
			holder[segment] = nested
		}

		holder = nested
	}

	holder[segments[len(segments)-1]] = value
}

// addInputMedia writes media references into the supplied request body in the configured format. It
// rejects frame prefixes, unsupported video references, and multipart-only styles.
func addInputMedia(body map[string]any, style catalog.InputMediaStyle, inputMediaProvParam, inputMediaListProvParam string, mediaInputs []media.Input) error {
	if len(mediaInputs) == 0 {
		return nil
	}

	for _, mediaInput := range mediaInputs {
		if mediaInput.HasFrame() {
			return &errs.MediaError{Problem: FrameFieldUnavailable, Cause: errs.ErrInputMediaTime}
		}

		if style == catalog.InputMediaSingle && mediaInput.Kind() == media.Video {
			return &errs.MediaError{Problem: SingleMediaNoVideo, Cause: errs.ErrInputMediaSource}
		}
	}

	switch style {
	case catalog.InputMediaString:
		addStringInputMedia(body, inputMediaProvParam, mediaInputs)
	case catalog.InputMediaSingle:
		if len(mediaInputs) == 1 {
			body[inputMediaProvParam] = inputMediaURLObject(&mediaInputs[0])

			return nil
		}

		arr := make([]any, 0, len(mediaInputs))
		for mediaIndex := range mediaInputs {
			arr = append(arr, inputMediaURLObject(&mediaInputs[mediaIndex]))
		}

		body[inputMediaListProvParam] = arr
	case catalog.InputMediaNested:
		arr := make([]any, 0, len(mediaInputs))
		for mediaIndex := range mediaInputs {
			mediaType := mediaURLType(&mediaInputs[mediaIndex])
			arr = append(arr, map[string]any{fieldType: mediaType, mediaType: inputMediaURLNested(&mediaInputs[mediaIndex])})
		}

		body[inputMediaProvParam] = arr
	case catalog.InputMediaParts:
		return &errs.ConfigError{Problem: JSONMediaPartsUnsupported, Cause: errs.ErrProvConfigInvalid}
	}

	return nil
}

// addStringInputMedia writes one media string or an ordered list into the supplied body.
func addStringInputMedia(body map[string]any, inputMediaProvParam string, mediaInputs []media.Input) {
	if len(mediaInputs) == 1 {
		body[inputMediaProvParam] = mediaInputs[0].DataURI()

		return
	}

	mediaStrings := make([]string, 0, len(mediaInputs))
	for mediaIndex := range mediaInputs {
		mediaStrings = append(mediaStrings, mediaInputs[mediaIndex].DataURI())
	}

	body[inputMediaProvParam] = mediaStrings
}

// inputMediaURLObject returns the request object for one media reference.
func inputMediaURLObject(mediaInput *media.Input) map[string]string {
	return map[string]string{fieldType: mediaURLType(mediaInput), fieldURL: mediaInput.DataURI()}
}

// mediaURLType returns the provider-neutral nested reference type for one media input.
func mediaURLType(mediaInput *media.Input) string {
	if mediaInput.Kind() == media.Video {
		return fieldVideoURL
	}

	return fieldImageURL
}

// inputMediaURLNested returns the nested URL value for one media reference.
func inputMediaURLNested(mediaInput *media.Input) map[string]string {
	return map[string]string{fieldURL: mediaInput.DataURI()}
}

// formBody returns the content type and multipart body for an image generation request; an empty
// prompt sends no prompt part.
func formBody(api *catalog.ImageAPI, model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input, stringParams ...params.FlagType) (contentType string, body []byte, err error) {
	fields := map[string]string{fieldModel: model.ID}
	if prompt != "" {
		fields[fieldPrompt] = prompt
	}

	for wireKey, paramVal := range WireParamValues(model, parameterValues, stringParams...) {
		fields[wireKey] = params.FormatValue(paramVal)
	}

	maps.Copy(fields, api.FixedProvFields)

	return multipartBody(fields, api.InputMediaProvParam, mediaInputs)
}

// multipartBody returns multipart form data containing fields and local image parts.
func multipartBody(fields map[string]string, inputMediaFormField string, mediaInputs []media.Input) (contentType string, body []byte, err error) {
	var buf bytes.Buffer

	multipartWriter := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := multipartWriter.WriteField(k, v); err != nil {
			return "", nil, fmt.Errorf("%q, %w, %w", fmt.Sprintf(FieldContextForm, k), errs.ErrTransportMultipart, err)
		}
	}

	for mediaIndex := range mediaInputs {
		if err := writeMediaPart(multipartWriter, inputMediaFormField, mediaIndex, &mediaInputs[mediaIndex]); err != nil {
			return "", nil, err
		}
	}

	if err := multipartWriter.Close(); err != nil {
		return "", nil, fmt.Errorf("%q, %w, %w", inputMediaFormField, errs.ErrTransportMultipart, err)
	}

	return multipartWriter.FormDataContentType(), buf.Bytes(), nil
}

// writeMediaPart writes one local image as a multipart file part.
func writeMediaPart(multipartWriter *multipart.Writer, field string, mediaIndex int, mediaInput *media.Input) error {
	if mediaInput.URL != "" || mediaInput.Kind() == media.Video {
		return &errs.MediaError{Problem: fmt.Sprintf(media.IndexForm, mediaIndex+1), Cause: errs.ErrInputMediaSource}
	}

	if _, timed := mediaInput.FrameTime(); timed {
		return &errs.MediaError{Problem: fmt.Sprintf(media.IndexForm, mediaIndex+1), Cause: errs.ErrInputMediaTime}
	}

	if len(mediaInput.Bytes) == 0 {
		return &errs.MediaError{Problem: fmt.Sprintf(media.ImageIndexForm, mediaIndex+1), Cause: errs.ErrInputMediaEmpty}
	}

	h := make(textproto.MIMEHeader)
	h.Set(headerContentDisposition, fmt.Sprintf(partFilenameForm, field, mediaIndex+1, media.ExtForMimeOr(mediaInput.MIME, "."+media.FormatPNG)))
	h.Set(headerContentType, mediaInput.MIME)

	p, err := multipartWriter.CreatePart(h)
	if err != nil {
		return fmt.Errorf("%q, %w, %w", fmt.Sprintf(media.ImageIndexForm, mediaIndex+1), errs.ErrTransportMultipart, err)
	}

	if _, err := p.Write(mediaInput.Bytes); err != nil {
		return fmt.Errorf("%q, %w, %w", fmt.Sprintf(media.ImageIndexForm, mediaIndex+1), errs.ErrTransportMultipart, err)
	}

	return nil
}

// imageDataItem contains one normalized image response item.
//   - base64: the optional base64-encoded image data
//   - url: the optional image download URL
//   - mime: the optional declared MIME type
type imageDataItem struct {
	base64, url, mime params.Nullable[string]
}

// parseRespImages decodes image response data while preserving absent and empty fields. It rejects
// null image records and prefers mime_type over media_type when both are strings.
func parseRespImages(body []byte) ([]imageDataItem, error) {
	var response struct {
		Data []*struct {
			Base64    any `json:"b64_json"` //nolint:tagliatelle // Fixed provider response field.
			URL       any `json:"url"`
			MIME      any `json:"mime_type"`  //nolint:tagliatelle // Fixed provider response field.
			MediaType any `json:"media_type"` //nolint:tagliatelle // Fixed provider response field.
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w, %w", errs.ErrResponseDecode, err)
	}

	respImages := make([]imageDataItem, 0, len(response.Data))
	for imageIndex, responseImage := range response.Data {
		if responseImage == nil {
			return nil, fmt.Errorf("%q, %w", fmt.Sprintf(RespEntryForm, fieldData, imageIndex+1), errs.ErrResponseDecode)
		}

		var imageData imageDataItem
		if value, present := responseImage.Base64.(string); present {
			imageData.base64 = params.GetSetIf(true, value)
		}

		if value, present := responseImage.URL.(string); present {
			imageData.url = params.GetSetIf(true, value)
		}

		if value, present := responseImage.MIME.(string); present {
			imageData.mime = params.GetSetIf(true, value)
		} else if value, present := responseImage.MediaType.(string); present {
			imageData.mime = params.GetSetIf(true, value)
		}

		respImages = append(respImages, imageData)
	}

	return respImages, nil
}

// createImageArtifacts decodes or downloads the images in a successful response. If an item fails,
// it cleans up earlier downloads and returns any cleanup error with the failure.
func createImageArtifacts(ctx context.Context, provModelLabel string, api *catalog.ImageAPI, status int, body []byte, record *metadata.Record) ([]artifact.Media, error) {
	if status/100 != 2 {
		return nil, APIErr(provModelLabel, status, body)
	}

	respImages, err := parseRespImages(body)
	if err != nil {
		return nil, fmt.Errorf("%q, %w", provModelLabel, err)
	}

	if len(respImages) == 0 {
		return nil, fmt.Errorf("%q, %w", provModelLabel, errs.ErrResponseNoData)
	}

	artifacts := make([]artifact.Media, 0, len(respImages))

	record.ProviderFinished(nil)

	for i, imageData := range respImages {
		generatedMedia, err := createRespArtifact(ctx, api, imageData, record)
		if err != nil {
			cleanupErr := artifact.Cleanup(artifacts)

			return nil, fmt.Errorf("%q, %w", fmt.Sprintf(RespEntryForm, provModelLabel, i+1), errors.Join(err, cleanupErr))
		}

		artifacts = append(artifacts, generatedMedia)
	}

	return artifacts, nil
}

// createRespArtifact returns the artifact represented by imageData. It may download a URL-backed
// image into a temporary file.
func createRespArtifact(ctx context.Context, api *catalog.ImageAPI, imageData imageDataItem, record *metadata.Record) (artifact.Media, error) {
	if b, ok := imageData.base64.ValIf(); ok {
		data, err := base64.StdEncoding.DecodeString(b)
		if err != nil {
			return artifact.Media{}, fmt.Errorf("%w, %w", errs.ErrResponseDecode, err)
		}

		return artifact.New(data, imageData.mime.ValOr(""), api.FallbackExt)
	}

	if u, ok := imageData.url.ValIf(); ok && api.RespImageURL {
		return httpapi.Fetch(ctx, u, httpapi.AuthCredential{}, api.FallbackExt, record)
	}

	return artifact.Media{}, errs.ErrResponseNoData
}
