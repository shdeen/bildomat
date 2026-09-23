package sourceful

// Invariants tested:
//  1. Sourceful malformed responses: When the creation or polling response contains malformed JSON,
//     Generate must return ErrResponseDecode and no artifacts.
//  2. Sourceful null data: When a successful creation or polling response contains null data,
//     Generate must return ErrResponseNoData.
//  3. Sourceful oversized poll response: When a polling response contains 65 MiB of data, Generate
//     must return ErrTransportSize and no artifacts.
//  4. Sourceful creation errors: For creation HTTP 401, 402, 403, or 429 with a nested
//     error.message, Generate must return an error matching both ErrResponseStatus and
//     ErrResponseServer and include the provider's message.
//  5. Sourceful missing response data: When creation omits jobId or a completed response omits
//     output.url, Generate must return ErrResponseNoData and make no artifact download request.
//  6. Sourceful failed download cleanup: When an artifact download ends before its declared
//     Content-Length, Generate must return an error and no artifacts, and leave the selected
//     temporary directory empty.
//  7. Required request ownership: When a parameter maps to model, instruction, idempotencyKey,
//     imageUrls, or a descendant of any of those fields, NewProvider must return an error matching
//     ErrProvConfig.

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// TestSourcefulMalformedResponses verifies invariant #1: Sourceful malformed responses.
//
// What is being tested:
// When the creation or polling response contains malformed JSON, Generate must return
// ErrResponseDecode and no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSourcefulMalformedResponses(t *testing.T) {
	t.Run("creation response", func(t *testing.T) {
		providerConfig, configLoaded := loadSourcefulTestProvider(t)
		if !configLoaded {
			return
		}

		configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
		if !modelLoaded {
			return
		}

		t.Setenv("TMPDIR", t.TempDir())
		fixture := newSourcefulHTTPFixture(t, &providerConfig)
		fixture.creationAnswer = sourcefulTestAnswer{statusCode: http.StatusCreated, answerBytes: []byte("{")}

		result, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
		if !errors.Is(generationErr, errs.ErrResponseDecode) || len(result.Artifacts) != 0 {
			t.Errorf("✗ malformed creation result = %+v, %v; want a decode error and no artifact", result, generationErr)
		}

		if !t.Failed() {
			t.Log("✓ malformed Sourceful creation JSON returns a decode error without an artifact")
		}
	})

	t.Run("poll response", func(t *testing.T) {
		providerConfig, configLoaded := loadSourcefulTestProvider(t)
		if !configLoaded {
			return
		}

		configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
		if !modelLoaded {
			return
		}

		t.Setenv("TMPDIR", t.TempDir())
		fixture := newSourcefulHTTPFixture(t, &providerConfig)
		fixture.pollingAnswers = []sourcefulTestAnswer{{statusCode: http.StatusOK, answerBytes: []byte("{")}}

		result, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
		if !errors.Is(generationErr, errs.ErrResponseDecode) || len(result.Artifacts) != 0 {
			t.Errorf("✗ malformed poll result = %+v, %v; want a decode error and no artifact", result, generationErr)
		}

		if !t.Failed() {
			t.Log("✓ malformed Sourceful poll JSON returns a decode error without an artifact")
		}
	})

	if !t.Failed() {
		t.Log("✓ malformed Sourceful creation and polling documents fail under the decode classification")
	}
}

// TestSourcefulNullData verifies invariant #2: Sourceful null data.
//
// What is being tested:
// When a successful creation or polling response contains null data, Generate must return
// ErrResponseNoData.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSourcefulNullData(t *testing.T) {
	t.Run("creation response", func(t *testing.T) {
		providerConfig, configLoaded := loadSourcefulTestProvider(t)
		if !configLoaded {
			return
		}

		configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
		if !modelLoaded {
			return
		}

		t.Setenv("TMPDIR", t.TempDir())
		fixture := newSourcefulHTTPFixture(t, &providerConfig)
		fixture.creationAnswer = sourcefulTestAnswer{
			statusCode:  http.StatusOK,
			answerBytes: sourcefulJSONDocument(t, map[string]any{"data": nil}),
		}

		_, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
		if !errors.Is(generationErr, errs.ErrResponseNoData) {
			t.Errorf("✗ null creation data error = %v, want no-data classification", generationErr)
		}

		if !t.Failed() {
			t.Log("✓ null creation data returns a no-data error")
		}
	})

	t.Run("poll response", func(t *testing.T) {
		providerConfig, configLoaded := loadSourcefulTestProvider(t)
		if !configLoaded {
			return
		}

		configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
		if !modelLoaded {
			return
		}

		t.Setenv("TMPDIR", t.TempDir())
		fixture := newSourcefulHTTPFixture(t, &providerConfig)
		fixture.pollingAnswers = []sourcefulTestAnswer{{
			statusCode:  http.StatusOK,
			answerBytes: sourcefulJSONDocument(t, map[string]any{"data": nil}),
		}}

		_, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
		if !errors.Is(generationErr, errs.ErrResponseNoData) {
			t.Errorf("✗ null poll data error = %v, want no-data classification", generationErr)
		}

		if !t.Failed() {
			t.Log("✓ null poll data returns a no-data error")
		}
	})

	if !t.Failed() {
		t.Log("✓ null data in successful Sourceful responses returns a classified no-data error")
	}
}

// TestSourcefulOversizedPollResponse verifies invariant #3: Sourceful oversized poll response.
//
// What is being tested:
// When a polling response contains 65 MiB of data, Generate must return ErrTransportSize and no
// artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSourcefulOversizedPollResponse(t *testing.T) {
	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &providerConfig)
	fixture.pollingAnswers = []sourcefulTestAnswer{{
		statusCode:  http.StatusOK,
		answerBytes: bytes.Repeat([]byte("x"), 65<<20),
	}}

	result, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
	if !errors.Is(generationErr, errs.ErrTransportSize) || len(result.Artifacts) != 0 {
		t.Errorf("✗ oversized poll result = %+v, %v; want a transport-size error and no artifact", result, generationErr)
	}

	if !t.Failed() {
		t.Log("✓ an oversized Sourceful poll response returns a transport-size error without an artifact")
	}
}

// TestSourcefulCreationErrors verifies invariant #4: Sourceful creation errors.
//
// What is being tested:
// For creation HTTP 401, 402, 403, or 429 with a nested error.message, Generate must return an
// error matching both ErrResponseStatus and ErrResponseServer and include the provider's message.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSourcefulCreationErrors(t *testing.T) {
	for _, statusCode := range []int{http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden, http.StatusTooManyRequests} {
		t.Run(fmt.Sprintf("status %d", statusCode), func(t *testing.T) {
			providerConfig, configLoaded := loadSourcefulTestProvider(t)
			if !configLoaded {
				return
			}

			configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
			if !modelLoaded {
				return
			}

			t.Setenv("TMPDIR", t.TempDir())
			fixture := newSourcefulHTTPFixture(t, &providerConfig)
			serverMessage := fmt.Sprintf("Sourceful test status %d", statusCode)
			fixture.creationAnswer = sourcefulTestAnswer{
				statusCode: statusCode,
				answerBytes: sourcefulJSONDocument(t, map[string]any{
					"error": map[string]any{"message": serverMessage},
				}),
			}

			_, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
			if !errors.Is(generationErr, errs.ErrResponseStatus) || !errors.Is(generationErr, errs.ErrResponseServer) {
				t.Errorf("✗ error = %v, want response-status and server-message classifications", generationErr)
			}

			if generationErr == nil || !strings.Contains(generationErr.Error(), serverMessage) {
				t.Errorf("✗ error = %v, want provider message %q", generationErr, serverMessage)
			}

			if !t.Failed() {
				t.Logf("✓ Sourceful creation status %d preserves its provider message and classifications", statusCode)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every documented Sourceful creation failure is classified with its provider message")
	}
}

// TestSourcefulMissingResponseData verifies invariant #5: Sourceful missing response data.
//
// What is being tested:
// When creation omits jobId or a completed response omits output.url, Generate must return
// ErrResponseNoData and make no artifact download request.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSourcefulMissingResponseData(t *testing.T) {
	t.Run("creation without job ID", verifySourcefulCreationWithoutJobID)
	t.Run("completed job without output URL", verifySourcefulCompletedWithoutOutputURL)

	if !t.Failed() {
		t.Log("✓ missing Sourceful creation and result data fail before artifact download")
	}
}

// TestSourcefulFailedDownloadCleanup verifies invariant #6: Sourceful failed download cleanup.
//
// What is being tested:
// When an artifact download ends before its declared Content-Length, Generate must return an error
// and no artifacts, and leave the selected temporary directory empty.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSourcefulFailedDownloadCleanup(t *testing.T) {
	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	temporaryDirectory := t.TempDir()
	t.Setenv("TMPDIR", temporaryDirectory)
	fixture := newSourcefulHTTPFixture(t, &providerConfig)
	fixture.downloadAnswer = sourcefulTestAnswer{
		statusCode:  http.StatusOK,
		byteCount:   100,
		answerBytes: []byte("short"),
	}

	result, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
	if generationErr == nil || len(result.Artifacts) != 0 {
		t.Errorf("✗ truncated download result = %+v, %v; want an error and no artifact", result, generationErr)
	}

	temporaryMatches, matchErr := filepath.Glob(filepath.Join(temporaryDirectory, "bild-dl-*"))
	if matchErr != nil {
		t.Errorf("✗ temporary-file glob failed: %v", matchErr)
	} else if len(temporaryMatches) != 0 {
		t.Errorf("✗ failed download left temporary files: %v", temporaryMatches)
	}

	directoryEntries, readErr := os.ReadDir(temporaryDirectory)
	if readErr != nil {
		t.Errorf("✗ temporary directory read failed: %v", readErr)
	} else if len(directoryEntries) != 0 {
		t.Errorf("✗ failed download left directory entries: %v", directoryEntries)
	}

	if !t.Failed() {
		t.Log("✓ a failed Sourceful result download leaves no temporary artifact")
	}
}

// TestRequiredRequestPaths verifies invariant #7: Required request ownership.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// When a parameter maps to model, instruction, idempotencyKey, imageUrls, or a descendant of any of
// those fields, NewProvider must return an error matching ErrProvConfig.
func TestRequiredRequestPaths(t *testing.T) {
	for _, requestPath := range []string{"model", "model.name", "instruction", "instruction.text", "idempotencyKey", "idempotencyKey.value", "imageUrls", "imageUrls.first"} {
		t.Run(requestPath, func(t *testing.T) {
			providerDescription, loaded := loadSourcefulTestProvider(t)
			if !loaded {
				t.Fatalf("💣 configuration setup failed")
			}

			providerDescription.Models[0].Params[0].ParamID = requestPath

			_, err := NewProvider(&providerDescription)
			if !errors.Is(err, errs.ErrProvConfig) {
				t.Errorf("✗ constructor accepts required field conflict %q: %v", requestPath, err)
			}

			if !t.Failed() {
				t.Log("✓ conflicting mapping fails before generation")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ required Sourceful fields cannot be displaced")
	}
}

// verifySourcefulCreationWithoutJobID checks that creation data without jobId stops early.
func verifySourcefulCreationWithoutJobID(t *testing.T) {
	t.Helper()

	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &providerConfig)
	fixture.creationAnswer = sourcefulTestAnswer{
		statusCode:  http.StatusOK,
		answerBytes: sourcefulJSONDocument(t, map[string]any{"data": map[string]any{"status": "accepted"}}),
	}

	_, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
	if !errors.Is(generationErr, errs.ErrResponseNoData) {
		t.Errorf("✗ missing-job-ID error = %v, want no-data classification", generationErr)
	}

	if downloadCount := fixture.requestCountByPrefix(http.MethodGet, "/result"); downloadCount != 0 {
		t.Errorf("✗ downloads after missing job ID = %d, want 0", downloadCount)
	}

	if !t.Failed() {
		t.Log("✓ a creation response without jobId fails before polling or download")
	}
}

// verifySourcefulCompletedWithoutOutputURL checks that completed data without a URL stops early.
func verifySourcefulCompletedWithoutOutputURL(t *testing.T) {
	t.Helper()

	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &providerConfig)
	fixture.pollingAnswers = []sourcefulTestAnswer{{
		statusCode: http.StatusOK,
		answerBytes: sourcefulJSONDocument(t, map[string]any{
			"data": map[string]any{
				"job": map[string]any{
					"status": providerConfig.Config.AdapterAPI.ReadyStatusText,
					"result": map[string]any{"output": map[string]any{"mimeType": "image/png"}},
				},
			},
		}),
	}}

	_, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
	if !errors.Is(generationErr, errs.ErrResponseNoData) {
		t.Errorf("✗ missing-output-URL error = %v, want no-data classification", generationErr)
	}

	if downloadCount := fixture.requestCountByPrefix(http.MethodGet, "/result"); downloadCount != 0 {
		t.Errorf("✗ downloads after missing output URL = %d, want 0", downloadCount)
	}

	if !t.Failed() {
		t.Log("✓ a completed response without output.url fails before download")
	}
}
