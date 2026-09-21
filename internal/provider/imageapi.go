package provider

import (
	"context"
	"fmt"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// submitImage sends the request described by api and returns the generated artifacts.
// It calls sendImageRequest to contact the provider.
func submitImage(ctx context.Context, api *catalog.ImageAPI, apiKey string, run *generation.Generation, stringParams ...params.FlagType) ([]artifact.Media, error) {
	if len(run.InputMedia) > 0 && api.InputMediaURL != "" && api.InputMediaPayloadType == catalog.InputMediaPayloadForm {
		downloadedInputs, err := httpapi.DownloadInputMedia(ctx, run.InputMedia)
		run.InputMedia = downloadedInputs

		if err != nil {
			return nil, err
		}
	}

	provModelLabel := run.Label()
	if run.Record != nil {
		run.Record.SetFields(nil, []metadata.BinaryField{{Path: []string{fieldData, "*", fieldB64JSON}, MIMEField: fieldMIMEType}})
	}

	status, body, err := sendImageRequest(ctx, httpapi.Bearer(apiKey), api, &run.Model, run.Prompt, run.Params, run.InputMedia, run.Record, stringParams...)
	if err != nil {
		return nil, fmt.Errorf("%q, %w", provModelLabel, err)
	}

	return createImageArtifacts(ctx, provModelLabel, api, status, body, run.Record)
}

// sendImageRequest sends an image request and returns its status code and response body.
// Input media routes the request to the media endpoint, as a multipart form when the API
// declares the form encoding; every other request posts JSON.
func sendImageRequest(ctx context.Context, credential httpapi.AuthCredential, api *catalog.ImageAPI, model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input, record *metadata.Record, stringParams ...params.FlagType) (status int, respBody []byte, err error) {
	mediaEndpoint := len(mediaInputs) > 0 && api.InputMediaURL != ""
	record.Prepared(mediaInputs)

	if mediaEndpoint && api.InputMediaPayloadType == catalog.InputMediaPayloadForm {
		ct, body, formErr := formBody(api, model, prompt, parameterValues, mediaInputs, stringParams...)
		if formErr != nil {
			return 0, nil, formErr
		}

		return httpapi.SendBody(ctx, api.InputMediaURL, credential, ct, body, record)
	}

	endpoint := api.GenURL
	if mediaEndpoint {
		endpoint = api.InputMediaURL
	}

	requestBody, bodyErr := jsonBody(api, model, prompt, parameterValues, mediaInputs, stringParams...)
	if bodyErr != nil {
		return 0, nil, bodyErr
	}

	return httpapi.PostJSON(ctx, endpoint, credential, requestBody, record)
}
