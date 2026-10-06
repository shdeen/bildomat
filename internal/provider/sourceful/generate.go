package sourceful

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/provider"
)

// fieldModel names the model ID supplied by the adapter.
const (
	fieldModel = "model"
)

// The Sourceful API's credential header, request routes, and request fields.
//   - keyHeader: the credential header
//   - riverflow2Family: the model family using version 2 of the API
//   - generationsRoute: the version 2.5 generation route
//   - generationsV2Route: the version 2 generation route
//   - textOperation: the creation operation for a prompt alone
//   - imageOperation: the creation operation for reference images
//   - fieldInstruction: the request field carrying the prompt
//   - fieldIdempotencyKey: the request field carrying the idempotency key
//   - fieldImageURLs: the request field carrying the reference images
const (
	keyHeader = "X-API-KEY"

	riverflow2Family   = "riverflow-2"
	generationsRoute   = "/v2.5/generations"
	generationsV2Route = "/v2/generations"
	textOperation      = "t2i"
	imageOperation     = "i2i"

	fieldInstruction    = "instruction"
	fieldIdempotencyKey = "idempotencyKey"
	fieldImageURLs      = "imageUrls"
)

// generationRoute selects the generation and polling route from the configured model family.
func generationRoute(modelFamily string) string {
	if modelFamily == riverflow2Family {
		return generationsV2Route
	}

	return generationsRoute
}

// buildRequestBody returns a Sourceful request with a fresh idempotency key.
func buildRequestBody(generationRequest *generation.Generation) map[string]any {
	requestBody := map[string]any{
		fieldInstruction:    generationRequest.Prompt,
		fieldIdempotencyKey: rand.Text(),
		fieldModel:          generationRequest.Model.ID,
	}

	// Validation keeps configured parameter paths separate from the required fields above. Each
	// configured object can therefore be copied without replacing a required value.
	maps.Copy(requestBody, provider.WireParamValues(&generationRequest.Model, generationRequest.Params))

	if len(generationRequest.InputMedia) > 0 {
		imageURLs := make([]string, 0, len(generationRequest.InputMedia))
		for mediaIndex := range generationRequest.InputMedia {
			imageURLs = append(imageURLs, generationRequest.InputMedia[mediaIndex].DataURI())
		}

		requestBody[fieldImageURLs] = imageURLs
	}

	return requestBody
}

// responseEnvelope retains the data object shared by creation and job responses.
type responseEnvelope struct {
	// Data contains the response payload.
	Data json.RawMessage `json:"data"`
}

// creationData decodes the required creation job identifier.
type creationData struct {
	// JobID identifies the created generation job.
	JobID string `json:"jobId"`
}

// creationJobID returns the required job ID from a creation response. Missing or incompatible
// identifiers retain missing-data classification and any decoding cause.
func creationJobID(responseBody []byte, providerModelName string) (string, error) {
	var response responseEnvelope
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("%q, %w, %w", providerModelName, errs.ErrResponseDecode, err)
	}

	var (
		data      creationData
		decodeErr error
	)
	if len(response.Data) != 0 {
		decodeErr = json.Unmarshal(response.Data, &data)
	}

	if decodeErr != nil || data.JobID == "" {
		return "", fmt.Errorf("%q, %w", fmt.Sprintf(JobIDMissing, providerModelName), errors.Join(errs.ErrResponseNoData, decodeErr))
	}

	return data.JobID, nil
}
