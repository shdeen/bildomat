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

// The model field is supplied by the adapter.
const (
	fieldModel = "model"
)

// The Sourceful API's credential header, request routes, and request and response fields.
//   - keyHeader: the credential header
//   - generationsRoute: the route every generation request starts from
//   - textRoute: the creation route for a prompt alone
//   - imageRoute: the creation route for a prompt with reference images
//   - pollRoute: the job query route, before the job ID
//   - fieldInstruction: the request field carrying the prompt
//   - fieldIdempotencyKey: the request field carrying the idempotency key
//   - fieldImageURLs: the request field carrying the reference images
const (
	keyHeader = "X-API-KEY"

	generationsRoute = "/v2.5/generations"
	textRoute        = generationsRoute + "/t2i"
	imageRoute       = generationsRoute + "/i2i"
	pollRoute        = generationsRoute + "/"

	fieldInstruction    = "instruction"
	fieldIdempotencyKey = "idempotencyKey"
	fieldImageURLs      = "imageUrls"
)

// buildRequestBody returns the request fields for one Sourceful job.
func buildRequestBody(generationRequest *generation.Generation) map[string]any {
	requestBody := map[string]any{
		fieldInstruction:    generationRequest.Prompt,
		fieldIdempotencyKey: rand.Text(),
		fieldModel:          generationRequest.Model.ID,
	}

	// Validated configured fields own their complete nested objects; the
	// adapter-owned fields above are disjoint scalar assignments.
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
	Data json.RawMessage `json:"data"`
}

// creationData decodes the required creation job identifier.
type creationData struct {
	JobID string `json:"jobId"`
}

// creationJobID retains missing-data classification and the original
// decoding cause when a required creation field has an incompatible type.
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
