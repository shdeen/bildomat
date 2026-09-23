package sourceful

// Invariants tested:
//  1. Creation identifier causes: Given a numeric jobId, creationJobID must return an empty ID and
//     an error matching ErrResponseNoData that retains the original json.UnmarshalTypeError.
//  2. Sourceful creation response: For arbitrary response bytes, creationJobID must return either a
//     nonempty ID without an error or an empty ID with ErrResponseDecode or ErrResponseNoData. It
//     must not panic.

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestCreationIDDecodeCause verifies invariant #1: Creation identifier causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given a numeric jobId, creationJobID must return an empty ID and an error matching
// ErrResponseNoData that retains the original json.UnmarshalTypeError.
func TestCreationIDDecodeCause(t *testing.T) {
	jobID, err := creationJobID([]byte(`{"data":{"jobId":17}}`), "sourceful/example")

	var typeFailure *json.UnmarshalTypeError
	if jobID != "" || !errors.Is(err, errs.ErrResponseNoData) || !errors.As(err, &typeFailure) {
		t.Errorf("✗ numeric jobId result=%q error=%v; want missing data with original JSON type cause", jobID, err)
	}

	if !t.Failed() {
		t.Log("✓ invalid identifier retains the decoder cause and classification")
	}
}

// FuzzSourcefulCreationResponse verifies invariant #2: Sourceful creation response.
//
// What is being tested:
// For arbitrary response bytes, creationJobID must return either a nonempty ID without an error or
// an empty ID with ErrResponseDecode or ErrResponseNoData. It must not panic.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSourcefulCreationResponse(f *testing.F) {
	f.Add([]byte(`{"data":{"jobId":17}}`))
	f.Add([]byte(`{"data":{"jobId":null}}`))
	f.Add(sourcefulJSONDocument(f, map[string]any{"data": map[string]any{"jobId": "sourceful-fuzz-job"}}))
	f.Add([]byte("{"))
	f.Add([]byte(`{"data":null}`))
	f.Add([]byte(`{"data":{"jobId":""}}`))

	f.Fuzz(func(t *testing.T, responseBody []byte) {
		jobID, creationErr := creationJobID(responseBody, ProviderID)
		if creationErr == nil && jobID == "" {
			t.Errorf("✗ creation response returned an empty job ID without an error")
		}

		if creationErr != nil && jobID != "" {
			t.Errorf("✗ creation response returned job ID %q with error %v", jobID, creationErr)
		}

		if creationErr != nil &&
			!errors.Is(creationErr, errs.ErrResponseDecode) &&
			!errors.Is(creationErr, errs.ErrResponseNoData) {
			t.Errorf("✗ creation body returned unclassified error: %v", creationErr)
		}

		if !t.Failed() {
			t.Log("✓ the Sourceful creation body returned a job ID or classified error")
		}
	})
}
