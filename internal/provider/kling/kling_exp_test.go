package kling

// Invariants tested:
//  1. Kling video download failure: When a video download ends before its declared Content-Length,
//     Generate must return an error and no artifacts.
//  2. Kling multiple image results: Given three indexed image results, Generate must return their
//     downloaded bytes in index order. If the second download is truncated, Generate must return an
//     error and no artifacts, remove the earlier temporary download, and skip the third download.
//  3. Kling malformed image poll body: When an image poll returns non-JSON text, Generate must
//     return ErrResponseDecode and no artifacts.
//  4. Kling null image poll data: When an image poll returns code zero with null data, Generate
//     must return ErrResponseNoData and no artifacts.
//  5. Kling video input rejection: Given an MP4 input for either the selected image model or video
//     model, AdjustParams must return ErrInputMediaUnsendable naming clip.mp4. The attempted
//     generation must send no HTTP request.
//  6. Kling nil generator input: Given a nil generation request, generator.Generate must return
//     ErrResponseNoData without panicking.
//  7. Kling frame resolution conflict: Given two images marked as the opening frame for a video
//     model, AdjustParams must return an error.
//  8. Retained input validation and caller ownership: For a model limited to one image,
//     AdjustParams must accept an image followed by a surplus video and return Capped then
//     Conformed records. It must preserve the caller's frame marker and timestamp pointer and
//     value. Given a retained video and unsupported size, it must return ErrInputMediaUnsendable
//     while retaining the Ignored size record.
//  9. Kling missing adapter: When the provider configuration omits AdapterAPI, NewProvider must
//     return ErrProvConfigNoAdapterAPI.
//  10. Kling ten omni image references: Given ten image URLs for the omni image model, AdjustParams
//      and Generate must succeed without input-media adjustments and submit exactly one request.
//      Its image_list must contain all ten URLs as image objects in the original order.
//  11. Kling null creation data: When image creation returns code zero with null data, Generate
//      must return ErrResponseNoData and no artifacts.
//  12. Kling creation transport failure: Given the invalid API base ://invalid, Generate must
//      return an error matching both ErrTransportRequest and ErrTransportCreate.
//  13. Kling malformed creation response: When image creation returns HTTP 200 with non-JSON text,
//      Generate must return ErrResponseDecode.
//  14. Kling empty creation task ID: When image creation returns code zero with an empty task_id,
//      Generate must return ErrResponseNoData.
//  15. Kling envelope failures: For HTTP 400 with code 1201 or HTTP 429 with code 1303, Generate
//      must return both ErrResponseStatus and ErrResponseServer. For HTTP 200 with code 1200, it
//      must return ErrResponseGen. Every error must include the supplied provider message.
//  16. Exact envelope codes: When creation returns fractional codes beyond float64 precision or
//      integers outside int64 bounds, Generate must return ErrResponseCodeInvalid and no artifacts.
//  17. Envelope presence: When creation omits code, Generate must return ErrResponseCodeMissing; a
//      null code must return ErrResponseCodeInvalid. Codes 0.0 and 0e9 must allow artifacts without
//      an error despite unknown fields or a non-string optional message. Code 12 with an object
//      message must return ErrResponseGen. Each failing case must return no artifacts.
//  18. Kling empty video task array: When a video poll returns code zero and an empty data array,
//      Generate must return ErrResponseNoData and no artifacts.
//  19. Kling malformed video poll body: When a video poll returns non-JSON text, Generate must
//      return ErrResponseDecode and no artifacts.
//  20. Kling null video poll data: When a video poll returns code zero with null data, Generate
//      must return ErrResponseNoData and no artifacts.

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestKlingVideoDownloadFailure verifies invariant #1: Kling video download failure.
//
// What is being tested:
// When a video download ends before its declared Content-Length, Generate must return an error and
// no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingVideoDownloadFailure(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	configuredModel := klingTestModel(t, &providerConfig, media.Video, 2, true, false)
	fixture.downloadAnswersByPath["/result/video-0"] = klingHTTPAnswer{
		statusCode: http.StatusOK,
		body:       []byte("short"),
		byteCount:  100,
	}

	result, _, generationErr := generateKlingTestMedia(t, &providerConfig, configuredModel, nil, nil)
	if generationErr == nil || len(result.Artifacts) != 0 {
		t.Errorf("✗ failed video download result = %+v, %v; want error and no artifact", result, generationErr)
	}

	if !t.Failed() {
		t.Log("✓ a failed video download returns no artifact")
	}
}

// TestKlingMultipleImageResults verifies invariant #2: Kling multiple image results.
//
// What is being tested:
// Given three indexed image results, Generate must return their downloaded bytes in index order. If
// the second download is truncated, Generate must return an error and no artifacts, remove the
// earlier temporary download, and skip the third download.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingMultipleImageResults(t *testing.T) {
	t.Run("three artifacts in index order", verifyKlingMultipleImageOrder)
	t.Run("failed second download removes earlier files", verifyKlingFailedImageDownloadCleanup)

	if !t.Failed() {
		t.Log("✓ image results land in index order and a later download failure removes earlier temporary files")
	}
}

// TestKlingMalformedImagePollBody verifies invariant #3: Kling malformed image poll body.
//
// What is being tested:
// When an image poll returns non-JSON text, Generate must return ErrResponseDecode and no
// artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingMalformedImagePollBody(t *testing.T) {
	verifyKlingPollBodyFailure(t, media.Image, []byte("not JSON"), errs.ErrResponseDecode, "malformed image poll")

	if !t.Failed() {
		t.Log("✓ a malformed image poll body returns a decode failure without an artifact")
	}
}

// TestKlingNullImagePollData verifies invariant #4: Kling null image poll data.
//
// What is being tested:
// When an image poll returns code zero with null data, Generate must return ErrResponseNoData and
// no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingNullImagePollData(t *testing.T) {
	nullResponse := klingFuzzJSON(t, map[string]any{"code": 0, "data": nil})
	verifyKlingPollBodyFailure(t, media.Image, nullResponse, errs.ErrResponseNoData, "null image poll data")

	if !t.Failed() {
		t.Log("✓ null image poll data returns a no-data failure without an artifact")
	}
}

// TestKlingVideoInputRejection verifies invariant #5: Kling video input rejection.
//
// What is being tested:
// Given an MP4 input for either the selected image model or video model, AdjustParams must return
// ErrInputMediaUnsendable naming clip.mp4. The attempted generation must send no HTTP request.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingVideoInputRejection(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	configuredModels := []catalog.Model{
		klingTestModel(t, &providerConfig, media.Image, 1, false, false),
		klingTestModel(t, &providerConfig, media.Video, 2, true, false),
	}
	videoInput := media.Input{Bytes: []byte("video"), MIME: "video/mp4", Filepath: "clip.mp4"}

	for _, configuredModel := range configuredModels {
		requestCountBefore := fixture.requestCount()

		_, _, generationErr := generateKlingTestMedia(t, &providerConfig, configuredModel, nil, []media.Input{videoInput})
		if !errors.Is(generationErr, errs.ErrInputMediaUnsendable) || generationErr == nil || !strings.Contains(generationErr.Error(), videoInput.Filepath) {
			t.Errorf("✗ %s video-input error = %v, want unsendable classification naming %q", configuredModel.Media, generationErr, videoInput.Filepath)
		}

		if fixture.requestCount() != requestCountBefore {
			t.Errorf("✗ %s video-input rejection reached the HTTP fixture", configuredModel.Media)
		}
	}

	if !t.Failed() {
		t.Log("✓ Kling rejects video input before an image or video request reaches the provider")
	}
}

// TestKlingNilGeneratorInput verifies invariant #6: Kling nil generator input.
//
// What is being tested:
// Given a nil generation request, generator.Generate must return ErrResponseNoData without
// panicking.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingNilGeneratorInput(t *testing.T) {
	generator := &generator{}
	if _, generationErr := generator.Generate(t.Context(), nil); !errors.Is(generationErr, errs.ErrResponseNoData) {
		t.Errorf("✗ nil generation error = %v, want no-data classification", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ a nil provider and request return a no-data failure")
	}
}

// TestKlingFrameResolutionConflict verifies invariant #7: Kling frame resolution conflict.
//
// What is being tested:
// Given two images marked as the opening frame for a video model, AdjustParams must return an
// error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingFrameResolutionConflict(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	configuredModel := klingTestModel(t, &providerConfig, media.Video, 2, true, false)
	mediaInputs := []media.Input{
		{URL: "https://media.example/first-a.png", MIME: "image/png", FrameAnchor: media.FrameFirst},
		{URL: "https://media.example/first-b.png", MIME: "image/png", FrameAnchor: media.FrameFirst},
	}

	_, adjustmentErr := constructedTestProvider(t, &providerConfig).AdjustParams(&configuredModel, nil, mediaInputs, nil)
	if adjustmentErr == nil {
		t.Errorf("✗ duplicate opening frames returned no adjustment error")
	}

	if !t.Failed() {
		t.Log("✓ duplicate opening frames return an adjustment failure")
	}
}

// TestKlingRetainedMediaOnly verifies invariant #8: Retained input validation and caller ownership.
//
// What is being tested:
// For a model limited to one image, AdjustParams must accept an image followed by a surplus video
// and return Capped then Conformed records. It must preserve the caller's frame marker and
// timestamp pointer and value. Given a retained video and unsupported size, it must return
// ErrInputMediaUnsendable while retaining the Ignored size record.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestKlingRetainedMediaOnly(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	generator := constructedTestProvider(t, &providerConfig)
	model := catalog.Model{ID: "retained-image", Media: media.Image, Params: params.Definitions{{FlagID: params.FlagTypeInputMedia, MaxMultiple: 1}}}
	frameTime := 0.0
	imageInput := media.Input{Filepath: "valid.png", MIME: "image/png", Bytes: []byte("immutable image bytes"), FrameAnchor: media.FrameFirst, Time: &frameTime}
	videoInput := media.Input{Filepath: "surplus.mp4", MIME: "video/mp4"}
	callerInputs := []media.Input{imageInput, videoInput}
	preparedGeneration6, err := generator.AdjustParams(&model, nil, callerInputs, nil)
	changes := preparedGeneration6.Changes

	if err != nil {
		t.Errorf("✗ discarded video invalidated retained image: %v", err)
	}

	if callerInputs[0].FrameAnchor != media.FrameFirst || callerInputs[0].Time != &frameTime || frameTime != 0 {
		t.Errorf("✗ preparation mutated caller media: %+v", callerInputs[0])
	}

	if len(changes) != 2 || changes[0].Type != params.ChangeCapped || changes[1].Type != params.ChangeConformed {
		t.Errorf("✗ retained media adjustments: %+v", changes)
	}

	preparedGeneration7, err := generator.AdjustParams(&model, params.FlagInputs{params.FlagTypeSize: "1024x1024"}, []media.Input{videoInput}, nil)
	changes = preparedGeneration7.Changes

	if !errors.Is(err, errs.ErrInputMediaUnsendable) {
		t.Errorf("✗ retained video lacked classified rejection: %v", err)
	}

	if len(changes) != 1 || changes[0].Type != params.ChangeIgnored || changes[0].FlagID != params.FlagTypeSize {
		t.Errorf("✗ retained-input failure lost prior adjustments: %+v", changes)
	}

	if !t.Failed() {
		t.Log("✓ only retained inputs are validated and caller records remain intact")
	}
}

// TestKlingMissingAdapter verifies invariant #9: Kling missing adapter.
//
// What is being tested:
// When the provider configuration omits AdapterAPI, NewProvider must return
// ErrProvConfigNoAdapterAPI.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingMissingAdapter(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	providerConfig.Config.AdapterAPI = nil

	if _, generationErr := NewProvider(&providerConfig); !errors.Is(generationErr, errs.ErrProvConfigNoAdapterAPI) {
		t.Errorf("✗ missing adapter error = %v, want provider-config classification", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ missing adapter settings return a provider-configuration failure")
	}
}

// TestKlingTenOmniImageReferences verifies invariant #10: Kling ten omni image references.
//
// What is being tested:
// Given ten image URLs for the omni image model, AdjustParams and Generate must succeed without
// input-media adjustments and submit exactly one request. Its image_list must contain all ten URLs
// as image objects in the original order.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingTenOmniImageReferences(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	configuredModel, _ := klingOmniImageModels(t, &providerConfig)

	mediaInputs := make([]media.Input, 10)
	for mediaIndex := range mediaInputs {
		mediaInputs[mediaIndex] = media.Input{
			URL:  fmt.Sprintf("https://media.example/reference-%d.png", mediaIndex),
			MIME: "image/png",
		}
	}

	result, changeRecords, generationErr := generateKlingTestMedia(t, &providerConfig, configuredModel, nil, mediaInputs)
	if generationErr != nil {
		t.Fatalf("💣 ten-reference omni generation failed: %v", generationErr)
	}
	defer removeKlingArtifacts(t, result.Artifacts)

	if changeCount(t, changeRecords, params.FlagTypeInputMedia) != 0 {
		t.Errorf("✗ ten supported references were adjusted: %+v", changeRecords)
	}

	creationRequests := fixture.requestsByMethod(http.MethodPost)
	if len(creationRequests) != 1 {
		t.Fatalf("💣 creation request count = %d, want one", len(creationRequests))
	}

	imageRecords, recordsPresent := creationRequests[0].body["image_list"].([]any)
	if !recordsPresent || len(imageRecords) != len(mediaInputs) {
		t.Fatalf("💣 image_list = %#v, want ten records", creationRequests[0].body["image_list"])
	}

	for mediaIndex := range mediaInputs {
		imageRecord := mapValue(t, imageRecords[mediaIndex], "omni image record")
		if imageRecord["image"] != mediaInputs[mediaIndex].URL {
			t.Errorf("✗ image_list record %d = %#v, want %q", mediaIndex, imageRecord, mediaInputs[mediaIndex].URL)
		}
	}

	if !t.Failed() {
		t.Log("✓ ten Kling omni references reach the request in retained order")
	}
}

// TestKlingNullCreationData verifies invariant #11: Kling null creation data.
//
// What is being tested:
// When image creation returns code zero with null data, Generate must return ErrResponseNoData and
// no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingNullCreationData(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	configuredModel := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	fixture.creationAnswers = []klingHTTPAnswer{klingJSONAnswer(t, http.StatusOK, map[string]any{"code": 0, "data": nil})}

	result, _, generationErr := generateKlingTestMedia(t, &providerConfig, configuredModel, nil, nil)
	verifyKlingNoArtifactFailure(t, "null creation data", result, generationErr, errs.ErrResponseNoData)

	if !t.Failed() {
		t.Log("✓ null creation data returns a no-data failure without an artifact")
	}
}

// TestKlingCreationTransportFailure verifies invariant #12: Kling creation transport failure.
//
// What is being tested:
// Given the invalid API base ://invalid, Generate must return an error matching both
// ErrTransportRequest and ErrTransportCreate.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingCreationTransportFailure(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	configuredModel := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	providerConfig.Config.AdapterAPI.APIBase = "://invalid"
	t.Setenv(providerConfig.APIKeyEnvVar, klingTestAPIKey)
	generationRequest := generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: providerConfig.Identity(), Model: configuredModel}, Prompt: params.GetSetIf(true, "two words").ValOr(""), APIKey: os.Getenv((catalog.ProvModelPair{Provider: providerConfig.Identity(), Model: configuredModel}).Provider.APIKeyEnvVar)}

	_, generationErr := constructedTestProvider(t, &providerConfig).Generate(t.Context(), &generationRequest)
	if !errors.Is(generationErr, errs.ErrTransportRequest) || !errors.Is(generationErr, errs.ErrTransportCreate) {
		t.Errorf("✗ creation transport error = %v, want Kling request and request-creation classifications", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ creation transport failure preserves both operation classifications")
	}
}

// TestKlingMalformedCreationResponse verifies invariant #13: Kling malformed creation response.
//
// What is being tested:
// When image creation returns HTTP 200 with non-JSON text, Generate must return ErrResponseDecode.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingMalformedCreationResponse(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	configuredModel := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	fixture.creationAnswers = []klingHTTPAnswer{{statusCode: http.StatusOK, body: []byte("not JSON")}}

	_, _, generationErr := generateKlingTestMedia(t, &providerConfig, configuredModel, nil, nil)
	if !errors.Is(generationErr, errs.ErrResponseDecode) {
		t.Errorf("✗ malformed creation error = %v, want decode classification", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ malformed creation JSON returns a decode failure")
	}
}

// TestKlingEmptyCreationTaskID verifies invariant #14: Kling empty creation task ID.
//
// What is being tested:
// When image creation returns code zero with an empty task_id, Generate must return
// ErrResponseNoData.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingEmptyCreationTaskID(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	configuredModel := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	fixture.creationAnswers = []klingHTTPAnswer{klingJSONAnswer(t, http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]any{"task_id": ""},
	})}

	_, _, generationErr := generateKlingTestMedia(t, &providerConfig, configuredModel, nil, nil)
	if !errors.Is(generationErr, errs.ErrResponseNoData) {
		t.Errorf("✗ empty creation task error = %v, want no-data classification", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ an empty creation task identifier returns a no-data failure")
	}
}

// TestKlingEnvelopeFailures verifies invariant #15: Kling envelope failures.
//
// What is being tested:
// For HTTP 400 with code 1201 or HTTP 429 with code 1303, Generate must return both
// ErrResponseStatus and ErrResponseServer. For HTTP 200 with code 1200, it must return
// ErrResponseGen. Every error must include the supplied provider message.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingEnvelopeFailures(t *testing.T) {
	failureCases := []struct {
		name           string
		statusCode     int
		responseCode   int
		message        string
		expectedErrors []error
	}{
		{
			name: "request parameter status", statusCode: http.StatusBadRequest, responseCode: 1201,
			message: "request parameter error", expectedErrors: []error{errs.ErrResponseStatus, errs.ErrResponseServer},
		},
		{
			name: "concurrency limit", statusCode: http.StatusTooManyRequests, responseCode: 1303,
			message: "concurrency limit exceeded", expectedErrors: []error{errs.ErrResponseStatus, errs.ErrResponseServer},
		},
		{
			name: "non-zero code on HTTP 200", statusCode: http.StatusOK, responseCode: 1200,
			message: "business request failed", expectedErrors: []error{errs.ErrResponseGen},
		},
	}

	for _, failureCase := range failureCases {
		t.Run(failureCase.name, func(t *testing.T) {
			providerConfig := loadKlingTestProvider(t)
			fixture := newKlingHTTPFixture(t, &providerConfig)
			model := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
			fixture.creationAnswers = []klingHTTPAnswer{klingJSONAnswer(t, failureCase.statusCode, map[string]any{
				"code": failureCase.responseCode, "message": failureCase.message, "data": nil,
			})}

			_, _, generationErr := generateKlingTestMedia(t, &providerConfig, model, nil, nil)
			for _, expectedError := range failureCase.expectedErrors {
				if !errors.Is(generationErr, expectedError) {
					t.Errorf("✗ error = %v, want classification %v", generationErr, expectedError)
				}
			}

			if generationErr == nil || !strings.Contains(generationErr.Error(), failureCase.message) {
				t.Errorf("✗ error = %v, want provider message %q", generationErr, failureCase.message)
			}

			if !t.Failed() {
				t.Log("✓ the Kling envelope preserves its required error classifications and message")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ Kling error envelopes preserve status, generation, and server-message classifications")
	}
}

// TestExactEnvelopeCodes verifies invariant #16: Exact envelope codes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// When creation returns fractional codes beyond float64 precision or integers outside int64 bounds,
// Generate must return ErrResponseCodeInvalid and no artifacts.
func TestExactEnvelopeCodes(t *testing.T) {
	for _, codeJSON := range []string{"9007199254740992.5", "-9223372036854775809", "9223372036854775808", "1.00000000000000000001"} {
		t.Run(codeJSON, func(t *testing.T) {
			providerDescription := loadKlingTestProvider(t)
			fixture := newKlingHTTPFixture(t, &providerDescription)
			fixture.creationAnswers = []klingHTTPAnswer{{statusCode: http.StatusOK, body: []byte(`{"code":` + codeJSON + `,"data":{"task_id":"unused"}}`)}}
			model := klingTestModel(t, &providerDescription, media.Image, 1, false, false)

			result, _, err := generateKlingTestMedia(t, &providerDescription, model, nil, nil)
			if !errors.Is(err, errs.ErrResponseCodeInvalid) || len(result.Artifacts) != 0 {
				t.Errorf("✗ code=%s error=%v artifacts=%d; want invalid-code classification", codeJSON, err, len(result.Artifacts))
			}

			if !t.Failed() {
				t.Log("✓ invalid numerical code is rejected without conversion loss")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ envelope codes retain exact numerical meaning")
	}
}

// TestEnvelopePresence verifies invariant #17: Envelope presence.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When creation omits code, Generate must return ErrResponseCodeMissing; a null code must return
// ErrResponseCodeInvalid. Codes 0.0 and 0e9 must allow artifacts without an error despite unknown
// fields or a non-string optional message. Code 12 with an object message must return
// ErrResponseGen. Each failing case must return no artifacts.
// Kind: permanent.
func TestEnvelopePresence(t *testing.T) {
	for _, responseCase := range []struct {
		name, document string
		classification error
	}{
		{"missing", `{"data":{"task_id":"image-task"}}`, errs.ErrResponseCodeMissing},
		{"null", `{"code":null,"data":{"task_id":"image-task"}}`, errs.ErrResponseCodeInvalid},
		{"decimal zero", `{"code":0.0,"message":17,"extra":{"value":true},"data":{"task_id":"image-task"}}`, nil},
		{"exponent zero", `{"code":0e9,"data":{"task_id":"image-task"}}`, nil},
		{"optional message", `{"code":12,"message":{},"data":null}`, errs.ErrResponseGen},
	} {
		t.Run(responseCase.name, func(t *testing.T) {
			configuration := loadKlingTestProvider(t)
			fixture := newKlingHTTPFixture(t, &configuration)
			fixture.creationAnswers = []klingHTTPAnswer{{statusCode: http.StatusOK, body: []byte(responseCase.document)}}
			model := klingTestModel(t, &configuration, media.Image, 1, false, false)

			result, _, err := generateKlingTestMedia(t, &configuration, model, nil, nil)
			defer removeKlingArtifacts(t, result.Artifacts)

			if !errors.Is(err, responseCase.classification) || (len(result.Artifacts) > 0) != (responseCase.classification == nil) {
				t.Errorf("✗ result artifacts=%d error=%v; want classification %v", len(result.Artifacts), err, responseCase.classification)
			}

			if !t.Failed() {
				t.Log("✓ envelope presence and optional values retain their classifications")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ required presence and tolerated optional envelope fields are preserved")
	}
}

// TestKlingEmptyVideoTaskArray verifies invariant #18: Kling empty video task array.
//
// What is being tested:
// When a video poll returns code zero and an empty data array, Generate must return
// ErrResponseNoData and no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingEmptyVideoTaskArray(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	configuredModel := klingTestModel(t, &providerConfig, media.Video, 2, true, false)
	fixture.videoPollingAnswers = []klingHTTPAnswer{klingJSONAnswer(t, http.StatusOK, map[string]any{
		"code": 0,
		"data": []any{},
	})}

	result, _, generationErr := generateKlingTestMedia(t, &providerConfig, configuredModel, nil, nil)
	if !errors.Is(generationErr, errs.ErrResponseNoData) || len(result.Artifacts) != 0 {
		t.Errorf("✗ empty video task array result = %+v, %v; want no-data failure and no artifact", result, generationErr)
	}

	if !t.Failed() {
		t.Log("✓ an empty Kling video task array returns a no-data failure without an artifact")
	}
}

// TestKlingMalformedVideoPollBody verifies invariant #19: Kling malformed video poll body.
//
// What is being tested:
// When a video poll returns non-JSON text, Generate must return ErrResponseDecode and no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingMalformedVideoPollBody(t *testing.T) {
	verifyKlingPollBodyFailure(t, media.Video, []byte("not JSON"), errs.ErrResponseDecode, "malformed video poll")

	if !t.Failed() {
		t.Log("✓ a malformed video poll body returns a decode failure without an artifact")
	}
}

// TestKlingNullVideoPollData verifies invariant #20: Kling null video poll data.
//
// What is being tested:
// When a video poll returns code zero with null data, Generate must return ErrResponseNoData and no
// artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingNullVideoPollData(t *testing.T) {
	nullResponse := klingFuzzJSON(t, map[string]any{"code": 0, "data": nil})
	verifyKlingPollBodyFailure(t, media.Video, nullResponse, errs.ErrResponseNoData, "null video poll data")

	if !t.Failed() {
		t.Log("✓ null video poll data returns a no-data failure without an artifact")
	}
}

// verifyKlingMultipleImageOrder verifies three downloaded image contents in provider index order.
func verifyKlingMultipleImageOrder(t *testing.T) {
	t.Helper()
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	model := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	indexedImages := []map[string]any{
		{"index": 0, "url": fixture.server.URL + "/result/zero"},
		{"index": 1, "url": fixture.server.URL + "/result/one"},
		{"index": 2, "url": fixture.server.URL + "/result/two"},
	}
	fixture.imagePollingAnswers = []klingHTTPAnswer{fixture.completedImageAnswer(indexedImages)}
	fixture.downloadAnswersByPath = map[string]klingHTTPAnswer{
		"/result/zero": {statusCode: http.StatusOK, body: []byte("zero")},
		"/result/one":  {statusCode: http.StatusOK, body: []byte("one")},
		"/result/two":  {statusCode: http.StatusOK, body: []byte("two")},
	}

	result, _, generationErr := generateKlingTestMedia(t, &providerConfig, model, params.FlagInputs{params.FlagTypeImageN: 3}, nil)
	if generationErr != nil || len(result.Artifacts) != 3 {
		t.Fatalf("💣 multi-image result = %+v, %v; want three artifacts", result, generationErr)
	}
	defer removeKlingArtifacts(t, result.Artifacts)

	artifactContents := make([]string, 0, len(result.Artifacts))

	for _, generatedMedia := range result.Artifacts {
		artifactBytes, readErr := os.ReadFile(generatedMedia.TmpPath)
		if readErr != nil {
			t.Errorf("✗ could not read artifact %s: %v", generatedMedia.TmpPath, readErr)
		}

		artifactContents = append(artifactContents, string(artifactBytes))
	}

	if !slices.Equal(artifactContents, []string{"zero", "one", "two"}) {
		t.Errorf("✗ artifact contents = %v, want index order", artifactContents)
	}

	if !t.Failed() {
		t.Log("✓ three Kling images download in provider index order")
	}
}

// verifyKlingFailedImageDownloadCleanup verifies that a failed second download removes the first
// file and prevents the third request.
func verifyKlingFailedImageDownloadCleanup(t *testing.T) {
	t.Helper()
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	model := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	fixture.imagePollingAnswers = []klingHTTPAnswer{fixture.completedImageAnswer([]map[string]any{
		{"index": 0, "url": fixture.server.URL + "/result/good"},
		{"index": 1, "url": fixture.server.URL + "/result/truncated"},
		{"index": 2, "url": fixture.server.URL + "/result/unreached"},
	})}
	fixture.downloadAnswersByPath = map[string]klingHTTPAnswer{
		"/result/good":      {statusCode: http.StatusOK, body: []byte("complete")},
		"/result/truncated": {statusCode: http.StatusOK, body: []byte("short"), byteCount: 100},
		"/result/unreached": {statusCode: http.StatusOK, body: []byte("must not download")},
	}

	result, _, generationErr := generateKlingTestMedia(t, &providerConfig, model, nil, nil)
	if generationErr == nil || len(result.Artifacts) != 0 {
		t.Errorf("✗ failed multi-image result = %+v, %v; want an error and no artifacts", result, generationErr)
	}

	temporaryFiles, globErr := filepath.Glob(filepath.Join(fixture.temporaryDirectory, "bild-dl-*"))
	if globErr != nil {
		t.Errorf("✗ temporary file glob failed: %v", globErr)
	} else if len(temporaryFiles) != 0 {
		t.Errorf("✗ failed second download left temporary files: %v", temporaryFiles)
	}

	for _, request := range fixture.recordedRequests {
		if request.path == "/result/unreached" {
			t.Errorf("✗ third download ran after the second failed")
		}
	}

	if !t.Failed() {
		t.Log("✓ a failed later download removes earlier temporary artifacts and stops the sequence")
	}
}

// verifyKlingPollBodyFailure verifies one image or video poll body that must return a classified
// failure without an artifact.
func verifyKlingPollBodyFailure(test *testing.T, medium media.Kind, responseBody []byte, expectedError error, diagnosticContext string) {
	test.Helper()

	providerConfig := loadKlingTestProvider(test)
	fixture := newKlingHTTPFixture(test, &providerConfig)
	inputLimit, audioDeclared := 1, false

	if medium == media.Image {
		fixture.imagePollingAnswers = []klingHTTPAnswer{{statusCode: http.StatusOK, body: responseBody}}
	} else {
		inputLimit, audioDeclared = 2, true
		fixture.videoPollingAnswers = []klingHTTPAnswer{{statusCode: http.StatusOK, body: responseBody}}
	}

	configuredModel := klingTestModel(test, &providerConfig, medium, inputLimit, audioDeclared, false)
	result, _, generationErr := generateKlingTestMedia(test, &providerConfig, configuredModel, nil, nil)
	verifyKlingNoArtifactFailure(test, diagnosticContext, result, generationErr, expectedError)
}

// verifyKlingNoArtifactFailure verifies one classified generation failure that must not return an
// artifact.
func verifyKlingNoArtifactFailure(test *testing.T, diagnosticContext string, result generation.Result, generationErr, expectedError error) {
	test.Helper()

	if !errors.Is(generationErr, expectedError) || len(result.Artifacts) != 0 {
		test.Errorf("✗ %s result = %+v, %v; want %v and no artifact", diagnosticContext, result, generationErr, expectedError)
	}
}

// requestCount returns the number of recorded requests.
func (fixture *klingHTTPFixture) requestCount() int {
	fixture.test.Helper()

	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()

	return len(fixture.recordedRequests)
}
