package kling

// Invariants tested:
//  1. Kling out of order image indexes: Given image results in index order 2, 0, 1, Generate must
//     return three artifacts whose downloaded contents appear in index order 0, 1, 2.
//  2. Kling image job states: After submitted or processing followed by succeed, Generate must
//     return one image artifact and make three GET requests for the two polls and download. A
//     failed response must return ErrResponseGen containing task_status_msg.
//  3. Kling configuration: Loading the embedded Kling configuration must return the package's
//     provider ID, nonempty provider and model names, an API key environment variable, and both
//     image and video models. The polling interval must be positive and no longer than the timeout.
//     catalog.LoadCatalog and NewProvider must accept the configuration.
//  4. Kling parameter adjustments: For the discrete-duration video model, AdjustParams must change
//     duration 7 to 5 and audio true to native, with matching Snapped and Conformed records. For
//     the range-duration model, it must cap duration 20 to 15 and change audio false to off, with
//     matching Capped and Conformed records. Both calls must succeed.
//  5. Kling omni resolution adjustments: Given resolution 4k, AdjustParams must preserve it without
//     a resolution adjustment for the model that declares 4k. For the model without 4k, it must
//     return a different resolution value and exactly one resolution adjustment. Both calls must
//     succeed.
//  6. Kling standard image requests: For remote and local image inputs, Generate must POST to
//     /v1/images/generations with the configured bearer credential and selected model. It must
//     preserve the remote URL, encode local bytes as raw base64, and send n=2,
//     negative_prompt=blur, aspect_ratio=16:9, and resolution=1k.
//  7. Kling omni image requests: Given a remote image followed by local image bytes and resolution
//     4k, Generate must POST once to /v1/images/omni-image with the selected model. Its image_list
//     must contain the unchanged URL followed by raw base64 image objects, and resolution must
//     remain 4k.
//  8. Kling video requests: AdjustParams and Generate must select text-to-video without images,
//     image-to-video for standard frame inputs, and omni-video for reference images. The request
//     must contain the supplied aspect and ordered prompt, frame, or reference record types. Audio
//     true and false must become native and off; unrequested audio and options must be absent. A
//     text request must preserve its top-level prompt and omit contents.
//  9. Kling accepted input requests: AdjustParams and Generate must accept WebP bytes without an
//     input-media adjustment and send them as raw base64 in the image field. They must also accept
//     a lone closing-frame URL without adjustment and send it in a last_frame content record after
//     the prompt.
//  10. Kling video job states: After submitted or processing followed by succeeded, Generate must
//      return one video artifact and make three GET requests for polling and download. The first
//      poll must request /tasks?task_ids=video-task. A failed response must return ErrResponseGen
//      containing the provider's message.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestKlingOutOfOrderImageIndexes verifies invariant #1: Kling out of order image indexes.
//
// What is being tested:
// Given image results in index order 2, 0, 1, Generate must return three artifacts whose downloaded
// contents appear in index order 0, 1, 2.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingOutOfOrderImageIndexes(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	configuredModel := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	fixture.imagePollingAnswers = []klingHTTPAnswer{fixture.completedImageAnswer([]map[string]any{
		{"index": 2, "url": fixture.server.URL + "/result/two"},
		{"index": 0, "url": fixture.server.URL + "/result/zero"},
		{"index": 1, "url": fixture.server.URL + "/result/one"},
	})}
	fixture.downloadAnswersByPath = map[string]klingHTTPAnswer{
		"/result/zero": {statusCode: http.StatusOK, body: []byte("zero")},
		"/result/one":  {statusCode: http.StatusOK, body: []byte("one")},
		"/result/two":  {statusCode: http.StatusOK, body: []byte("two")},
	}

	result, _, generationErr := generateKlingTestMedia(t, &providerConfig, configuredModel, params.FlagInputs{params.FlagTypeImageN: 3}, nil)
	if generationErr != nil || len(result.Artifacts) != 3 {
		t.Fatalf("💣 out-of-order result = %+v, %v; want three artifacts", result, generationErr)
	}
	defer removeKlingArtifacts(t, result.Artifacts)

	artifactContents := make([]string, 0, len(result.Artifacts))
	for _, generatedMedia := range result.Artifacts {
		artifactBytes, readErr := os.ReadFile(generatedMedia.TmpPath)
		if readErr != nil {
			t.Errorf("✗ could not read artifact %s: %v", generatedMedia.TmpPath, readErr)

			continue
		}

		artifactContents = append(artifactContents, string(artifactBytes))
	}

	if !slices.Equal(artifactContents, []string{"zero", "one", "two"}) {
		t.Errorf("✗ artifact contents = %v, want provider index order", artifactContents)
	}

	if !t.Failed() {
		t.Log("✓ out-of-order Kling image records download in ascending provider index order")
	}
}

// TestKlingImageJobStates verifies invariant #2: Kling image job states.
//
// What is being tested:
// After submitted or processing followed by succeed, Generate must return one image artifact and
// make three GET requests for the two polls and download. A failed response must return
// ErrResponseGen containing task_status_msg.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingImageJobStates(t *testing.T) {
	for _, pendingStatus := range []string{"submitted", "processing"} {
		t.Run(pendingStatus, func(t *testing.T) {
			providerConfig := loadKlingTestProvider(t)
			fixture := newKlingHTTPFixture(t, &providerConfig)
			model := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
			fixture.imagePollingAnswers = []klingHTTPAnswer{
				klingJSONAnswer(t, http.StatusOK, map[string]any{"code": 0, "data": map[string]any{"task_status": pendingStatus}}),
				fixture.completedImageAnswer([]map[string]any{{"index": 0, "url": fixture.server.URL + "/result/image"}}),
			}

			result, _, generationErr := generateKlingTestMedia(t, &providerConfig, model, nil, nil)
			if generationErr != nil || len(result.Artifacts) != 1 {
				t.Errorf("✗ %s result = %+v, %v; want a later completed artifact", pendingStatus, result, generationErr)
			}

			removeKlingArtifacts(t, result.Artifacts)

			if len(fixture.requestsByMethod(http.MethodGet)) != 3 {
				t.Errorf("✗ %s did not cause two polls and one download", pendingStatus)
			}

			if !t.Failed() {
				t.Log("✓ the pending image status continues to a completed download")
			}
		})
	}

	t.Run("failed", func(t *testing.T) {
		providerConfig := loadKlingTestProvider(t)
		fixture := newKlingHTTPFixture(t, &providerConfig)
		model := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
		failureMessage := "image task failed"
		fixture.imagePollingAnswers = []klingHTTPAnswer{klingJSONAnswer(t, http.StatusOK, map[string]any{
			"code": 0,
			"data": map[string]any{"task_status": "failed", "task_status_msg": failureMessage},
		})}

		_, _, generationErr := generateKlingTestMedia(t, &providerConfig, model, nil, nil)
		if !errors.Is(generationErr, errs.ErrResponseGen) || generationErr == nil || !strings.Contains(generationErr.Error(), failureMessage) {
			t.Errorf("✗ failed image status error = %v, want generation failure carrying message", generationErr)
		}

		if !t.Failed() {
			t.Log("✓ the failed image status preserves its provider message")
		}
	})

	if !t.Failed() {
		t.Log("✓ submitted and processing image jobs continue to succeed, while failed preserves its message")
	}
}

// TestKlingConfiguration verifies invariant #3: Kling configuration.
//
// What is being tested:
// Loading the embedded Kling configuration must return the package's provider ID, nonempty provider
// and model names, an API key environment variable, and both image and video models. The polling
// interval must be positive and no longer than the timeout. catalog.LoadCatalog and NewProvider
// must accept the configuration.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingConfiguration(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)

	if providerConfig.ID != ProviderID {
		t.Errorf("✗ provider ID = %q, want package provider ID %q", providerConfig.ID, ProviderID)
	}

	if providerConfig.DisplayName == "" || providerConfig.APIKeyEnvVar == "" {
		t.Errorf("✗ provider identity is incomplete: %+v", providerConfig.Identity())
	}

	if providerConfig.Config == nil || providerConfig.Config.AdapterAPI == nil {
		t.Fatalf("💣 configuration has no adapter settings to inspect")
	}

	if providerConfig.Config.AdapterAPI.PollInterval <= 0 || providerConfig.Config.AdapterAPI.PollTimeout < providerConfig.Config.AdapterAPI.PollInterval {
		t.Errorf("✗ polling settings = %d/%d, want a positive interval within the timeout", providerConfig.Config.AdapterAPI.PollInterval, providerConfig.Config.AdapterAPI.PollTimeout)
	}

	verifyKlingModelConfiguration(t, &providerConfig)
	verifyKlingCatalogConstruction(t)

	if !t.Failed() {
		t.Log("✓ the Kling configuration validates, declares image and video models, and constructs its generator")
	}
}

// TestKlingParameterAdjustments verifies invariant #4: Kling parameter adjustments.
//
// What is being tested:
// For the discrete-duration video model, AdjustParams must change duration 7 to 5 and audio true to
// native, with matching Snapped and Conformed records. For the range-duration model, it must cap
// duration 20 to 15 and change audio false to off, with matching Capped and Conformed records. Both
// calls must succeed.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingParameterAdjustments(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	discreteModel := klingTestModel(t, &providerConfig, media.Video, 2, true, true)
	rangeModel := klingTestModel(t, &providerConfig, media.Video, 2, true, false)
	generator := constructedTestProvider(t, &providerConfig)

	preparedGeneration2, err := generator.AdjustParams(&discreteModel, params.FlagInputs{
		params.FlagTypeDuration: 7, params.FlagType("generate-audio"): true,
	}, nil, nil)
	discreteParams, discreteChanges := preparedGeneration2.Params, preparedGeneration2.Changes

	if err != nil {
		t.Errorf("✗ discrete duration adjustment failed: %v", err)
	}

	if discreteParams[params.FlagTypeDuration] != 5 || discreteParams[params.FlagType("generate-audio")] != "native" {
		t.Errorf("✗ discrete params = %+v, want duration 5 and native audio", discreteParams)
	}

	if !hasChange(t, discreteChanges, params.FlagTypeDuration, params.ChangeSnapped, "7", "5") {
		t.Errorf("✗ duration 7 lacks a snapped-to-5 record: %+v", discreteChanges)
	}

	if !hasChange(t, discreteChanges, params.FlagType("generate-audio"), params.ChangeConformed, "true", "native") {
		t.Errorf("✗ enabled audio lacks a conformed-to-native record: %+v", discreteChanges)
	}

	preparedGeneration3, err := generator.AdjustParams(&rangeModel, params.FlagInputs{
		params.FlagTypeDuration: 20, params.FlagType("generate-audio"): false,
	}, nil, nil)
	rangeParams, rangeChanges := preparedGeneration3.Params, preparedGeneration3.Changes

	if err != nil {
		t.Errorf("✗ range duration adjustment failed: %v", err)
	}

	if rangeParams[params.FlagTypeDuration] != 15 || rangeParams[params.FlagType("generate-audio")] != "off" {
		t.Errorf("✗ range params = %+v, want duration 15 and off audio", rangeParams)
	}

	if !hasChange(t, rangeChanges, params.FlagTypeDuration, params.ChangeCapped, "20", "15") {
		t.Errorf("✗ duration 20 lacks a capped-to-15 record: %+v", rangeChanges)
	}

	if !hasChange(t, rangeChanges, params.FlagType("generate-audio"), params.ChangeConformed, "false", "off") {
		t.Errorf("✗ disabled audio lacks a conformed-to-off record: %+v", rangeChanges)
	}

	if !t.Failed() {
		t.Log("✓ Kling duration values move or cap through configured constraints and audio booleans map to provider words")
	}
}

// TestKlingOmniResolutionAdjustments verifies invariant #5: Kling omni resolution adjustments.
//
// What is being tested:
// Given resolution 4k, AdjustParams must preserve it without a resolution adjustment for the model
// that declares 4k. For the model without 4k, it must return a different resolution value and
// exactly one resolution adjustment. Both calls must succeed.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingOmniResolutionAdjustments(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	modelWith4K, modelWithout4K := klingOmniImageModels(t, &providerConfig)
	generator := constructedTestProvider(t, &providerConfig)

	preparedGeneration4, retainedErr := generator.AdjustParams(&modelWith4K, params.FlagInputs{params.FlagTypeResolution: "4k"}, nil, nil)
	retainedParams, retainedChanges := preparedGeneration4.Params, preparedGeneration4.Changes

	if retainedErr != nil {
		t.Errorf("✗ supported resolution adjustment failed: %v", retainedErr)
	}

	if retainedParams[params.FlagTypeResolution] != "4k" || changeCount(t, retainedChanges, params.FlagTypeResolution) != 0 {
		t.Errorf("✗ supported resolution changed: params=%+v changes=%+v", retainedParams, retainedChanges)
	}

	preparedGeneration5, adjustedErr := generator.AdjustParams(&modelWithout4K, params.FlagInputs{params.FlagTypeResolution: "4k"}, nil, nil)
	adjustedParams, adjustedChanges := preparedGeneration5.Params, preparedGeneration5.Changes

	if adjustedErr != nil {
		t.Errorf("✗ unsupported resolution adjustment failed: %v", adjustedErr)
	}

	if adjustedParams[params.FlagTypeResolution] == "4k" || changeCount(t, adjustedChanges, params.FlagTypeResolution) != 1 {
		t.Errorf("✗ unsupported resolution remained unchanged or lacked its record: params=%+v changes=%+v", adjustedParams, adjustedChanges)
	}

	if !t.Failed() {
		t.Log("✓ omni image resolution adjustment follows each model's configured values")
	}
}

// TestKlingStandardImageRequests verifies invariant #6: Kling standard image requests.
//
// What is being tested:
// For remote and local image inputs, Generate must POST to /v1/images/generations with the
// configured bearer credential and selected model. It must preserve the remote URL, encode local
// bytes as raw base64, and send n=2, negative_prompt=blur, aspect_ratio=16:9, and resolution=1k.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingStandardImageRequests(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	model := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	localImageBytes := []byte("local Kling image bytes")
	inputCases := []struct {
		name            string
		mediaInput      media.Input
		expectedPayload string
	}{
		{
			name:            "URL input",
			mediaInput:      media.Input{URL: "https://media.example/kling-reference.png", MIME: "image/png"},
			expectedPayload: "https://media.example/kling-reference.png",
		},
		{
			name:            "local input",
			mediaInput:      media.Input{Bytes: localImageBytes, MIME: "image/png", Filepath: "reference.png"},
			expectedPayload: base64.StdEncoding.EncodeToString(localImageBytes),
		},
	}

	for _, inputCase := range inputCases {
		result, _, generationErr := generateKlingTestMedia(t, &providerConfig, model, params.FlagInputs{
			params.FlagTypeAspect:              "16:9",
			params.FlagTypeResolution:          "1k",
			params.FlagTypeImageN:              2,
			params.FlagType("negative-prompt"): "blur",
		}, []media.Input{inputCase.mediaInput})
		if generationErr != nil {
			t.Errorf("✗ %s generation failed: %v", inputCase.name, generationErr)
		}

		removeKlingArtifacts(t, result.Artifacts)
	}

	creationRequests := fixture.requestsByMethod(http.MethodPost)
	if len(creationRequests) != len(inputCases) {
		t.Fatalf("💣 creation request count = %d, want %d", len(creationRequests), len(inputCases))
	}

	for requestIndex, request := range creationRequests {
		if request.path != "/v1/images/generations" {
			t.Errorf("✗ %s path = %q, want standard image path", inputCases[requestIndex].name, request.path)
		}

		if request.authorization != "Bearer "+klingTestAPIKey {
			t.Errorf("✗ %s authorization = %q, want bearer test key", inputCases[requestIndex].name, request.authorization)
		}

		if request.body["model_name"] != model.ID || request.body["image"] != inputCases[requestIndex].expectedPayload {
			t.Errorf("✗ %s identity or image payload = %#v", inputCases[requestIndex].name, request.body)
		}

		if strings.HasPrefix(fmt.Sprint(request.body["image"]), "data:") {
			t.Errorf("✗ %s image carries a data-URI prefix: %v", inputCases[requestIndex].name, request.body["image"])
		}

		for fieldName, expectedValue := range map[string]any{
			"n": float64(2), "negative_prompt": "blur", "aspect_ratio": "16:9", "resolution": "1k",
		} {
			if request.body[fieldName] != expectedValue {
				t.Errorf("✗ %s field %s = %#v, want %#v", inputCases[requestIndex].name, fieldName, request.body[fieldName], expectedValue)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ standard image requests preserve URLs, encode local bytes as raw base64, and carry every supplied field")
	}
}

// TestKlingOmniImageRequests verifies invariant #7: Kling omni image requests.
//
// What is being tested:
// Given a remote image followed by local image bytes and resolution 4k, Generate must POST once to
// /v1/images/omni-image with the selected model. Its image_list must contain the unchanged URL
// followed by raw base64 image objects, and resolution must remain 4k.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingOmniImageRequests(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	modelWith4K, _ := klingOmniImageModels(t, &providerConfig)
	mediaInputs := []media.Input{
		{URL: "https://media.example/first.png", MIME: "image/png"},
		{Bytes: []byte("second image"), MIME: "image/jpeg", Filepath: "second.jpg"},
	}

	result, _, generationErr := generateKlingTestMedia(t, &providerConfig, modelWith4K, params.FlagInputs{
		params.FlagTypeResolution: "4k",
		params.FlagTypeImageN:     1,
	}, mediaInputs)
	if generationErr != nil {
		t.Errorf("✗ 4k omni generation failed: %v", generationErr)
	}

	removeKlingArtifacts(t, result.Artifacts)

	verifyKlingOmniImageBody(t, fixture, &modelWith4K, mediaInputs)

	if !t.Failed() {
		t.Log("✓ omni image requests preserve ordered image objects and the accepted resolution")
	}
}

// TestKlingVideoRequests verifies invariant #8: Kling video requests.
//
// What is being tested:
// AdjustParams and Generate must select text-to-video without images, image-to-video for standard
// frame inputs, and omni-video for reference images. The request must contain the supplied aspect
// and ordered prompt, frame, or reference record types. Audio true and false must become native and
// off; unrequested audio and options must be absent. A text request must preserve its top-level
// prompt and omit contents.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingVideoRequests(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	standardModel, turboModel, omniModel := klingVideoModels(t, &providerConfig)

	testCases := []struct {
		name              string
		model             catalog.Model
		flagInputs        params.FlagInputs
		mediaInputs       []media.Input
		expectedPath      string
		expectedTypes     []string
		expectedAudio     any
		expectAudioAbsent bool
	}{
		{
			name:          "text route with native audio",
			model:         standardModel,
			flagInputs:    params.FlagInputs{params.FlagTypeAspect: "16:9", params.FlagType("generate-audio"): true},
			expectedPath:  "/text-to-video/" + standardModel.ID,
			expectedAudio: "native",
		},
		{
			name:  "anchored image route with disabled audio",
			model: standardModel,
			flagInputs: params.FlagInputs{
				params.FlagTypeAspect: "9:16", params.FlagType("generate-audio"): false,
			},
			mediaInputs: []media.Input{
				{URL: "https://media.example/open.png", MIME: "image/png", FrameAnchor: media.FrameFirst},
				{URL: "https://media.example/close.png", MIME: "image/png", FrameAnchor: media.FrameLast},
			},
			expectedPath:  "/image-to-video/" + standardModel.ID,
			expectedTypes: []string{"prompt", "first_frame", "last_frame"},
			expectedAudio: "off",
		},
		{
			name:              "one unanchored image becomes first frame",
			model:             turboModel,
			flagInputs:        params.FlagInputs{params.FlagTypeAspect: "1:1"},
			mediaInputs:       []media.Input{{URL: "https://media.example/only.png", MIME: "image/png"}},
			expectedPath:      "/image-to-video/" + turboModel.ID,
			expectedTypes:     []string{"prompt", "first_frame"},
			expectAudioAbsent: true,
		},
		{
			name:       "two unanchored images become first and last",
			model:      standardModel,
			flagInputs: params.FlagInputs{params.FlagTypeAspect: "16:9"},
			mediaInputs: []media.Input{
				{URL: "https://media.example/a.png", MIME: "image/png"},
				{URL: "https://media.example/b.png", MIME: "image/png"},
			},
			expectedPath:      "/image-to-video/" + standardModel.ID,
			expectedTypes:     []string{"prompt", "first_frame", "last_frame"},
			expectAudioAbsent: true,
		},
		{
			name:              "omni reference image",
			model:             omniModel,
			flagInputs:        params.FlagInputs{params.FlagTypeAspect: "16:9"},
			mediaInputs:       []media.Input{{URL: "https://media.example/reference.png", MIME: "image/png"}},
			expectedPath:      "/omni-video/" + omniModel.ID,
			expectedTypes:     []string{"prompt", "refer_image"},
			expectAudioAbsent: true,
		},
	}

	for _, testCase := range testCases {
		result, _, generationErr := generateKlingTestMedia(t, &providerConfig, testCase.model, testCase.flagInputs, testCase.mediaInputs)
		if generationErr != nil {
			t.Errorf("✗ %s generation failed: %v", testCase.name, generationErr)
		}

		removeKlingArtifacts(t, result.Artifacts)
	}

	creationRequests := fixture.requestsByMethod(http.MethodPost)
	if len(creationRequests) != len(testCases) {
		t.Fatalf("💣 video creation request count = %d, want %d", len(creationRequests), len(testCases))
	}

	for requestIndex, request := range creationRequests {
		testCase := testCases[requestIndex]
		verifyKlingVideoRequest(
			t,
			request,
			testCase.name,
			testCase.flagInputs,
			testCase.expectedPath,
			testCase.expectedTypes,
			testCase.expectedAudio,
			testCase.expectAudioAbsent,
		)
	}

	if !t.Failed() {
		t.Log("✓ video requests select all three route kinds, encode frames and references, map audio, and omit options")
	}
}

// TestKlingAcceptedInputRequests verifies invariant #9: Kling accepted input requests.
//
// What is being tested:
// AdjustParams and Generate must accept WebP bytes without an input-media adjustment and send them
// as raw base64 in the image field. They must also accept a lone closing-frame URL without
// adjustment and send it in a last_frame content record after the prompt.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingAcceptedInputRequests(t *testing.T) {
	providerConfig := loadKlingTestProvider(t)
	fixture := newKlingHTTPFixture(t, &providerConfig)
	imageModel := klingTestModel(t, &providerConfig, media.Image, 1, false, false)
	videoModel := klingTestModel(t, &providerConfig, media.Video, 2, true, false)

	webPBytes := []byte("RIFF____WEBP")

	result, changes, generationErr := generateKlingTestMedia(t, &providerConfig, imageModel, nil, []media.Input{{
		Bytes: webPBytes, MIME: "image/webp", Filepath: "reference.webp",
	}})
	if generationErr != nil {
		t.Errorf("✗ WebP input did not reach Kling: %v", generationErr)
	}

	removeKlingArtifacts(t, result.Artifacts)

	if changeCount(t, changes, params.FlagTypeInputMedia) != 0 {
		t.Errorf("✗ WebP input was adjusted: %+v", changes)
	}

	closingURL := "https://media.example/closing.png"

	result, changes, generationErr = generateKlingTestMedia(t, &providerConfig, videoModel, nil, []media.Input{{
		URL: closingURL, MIME: "image/png", FrameAnchor: media.FrameLast,
	}})
	if generationErr != nil {
		t.Errorf("✗ closing frame without opening frame did not reach Kling: %v", generationErr)
	}

	removeKlingArtifacts(t, result.Artifacts)

	if changeCount(t, changes, params.FlagTypeInputMedia) != 0 {
		t.Errorf("✗ closing frame without opening frame was adjusted: %+v", changes)
	}

	verifyKlingAcceptedInputRequests(t, fixture, webPBytes, closingURL)

	if !t.Failed() {
		t.Log("✓ WebP and a lone closing frame reach Kling unchanged")
	}
}

// TestKlingVideoJobStates verifies invariant #10: Kling video job states.
//
// What is being tested:
// After submitted or processing followed by succeeded, Generate must return one video artifact and
// make three GET requests for polling and download. The first poll must request
// /tasks?task_ids=video-task. A failed response must return ErrResponseGen containing the
// provider's message.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingVideoJobStates(t *testing.T) {
	for _, pendingStatus := range []string{"submitted", "processing"} {
		t.Run(pendingStatus, func(t *testing.T) {
			providerConfig := loadKlingTestProvider(t)
			fixture := newKlingHTTPFixture(t, &providerConfig)
			model := klingTestModel(t, &providerConfig, media.Video, 2, true, false)
			fixture.videoPollingAnswers = []klingHTTPAnswer{
				klingJSONAnswer(t, http.StatusOK, map[string]any{"code": 0, "data": []any{map[string]any{"status": pendingStatus}}}),
				fixture.completedVideoAnswer(fixture.server.URL + "/result/video"),
			}

			result, _, generationErr := generateKlingTestMedia(t, &providerConfig, model, nil, nil)
			if generationErr != nil || len(result.Artifacts) != 1 {
				t.Errorf("✗ %s result = %+v, %v; want a later completed artifact", pendingStatus, result, generationErr)
			}

			removeKlingArtifacts(t, result.Artifacts)

			pollRequests := fixture.requestsByMethod(http.MethodGet)
			if len(pollRequests) != 3 || pollRequests[0].path != "/tasks" || pollRequests[0].rawQuery != "task_ids=video-task" {
				t.Errorf("✗ %s query flow = %+v, want two task queries and one download", pendingStatus, pollRequests)
			}

			if !t.Failed() {
				t.Log("✓ the pending video status continues to a completed download")
			}
		})
	}

	t.Run("failed", func(t *testing.T) {
		providerConfig := loadKlingTestProvider(t)
		fixture := newKlingHTTPFixture(t, &providerConfig)
		model := klingTestModel(t, &providerConfig, media.Video, 2, true, false)
		failureMessage := "video task failed"
		fixture.videoPollingAnswers = []klingHTTPAnswer{klingJSONAnswer(t, http.StatusOK, map[string]any{
			"code": 0,
			"data": []any{map[string]any{"status": "failed", "message": failureMessage}},
		})}

		_, _, generationErr := generateKlingTestMedia(t, &providerConfig, model, nil, nil)
		if !errors.Is(generationErr, errs.ErrResponseGen) || generationErr == nil || !strings.Contains(generationErr.Error(), failureMessage) {
			t.Errorf("✗ failed video status error = %v, want generation failure carrying message", generationErr)
		}

		if !t.Failed() {
			t.Log("✓ the failed video status preserves its provider message")
		}
	})

	if !t.Failed() {
		t.Log("✓ submitted and processing video jobs continue to completion, while failed preserves its message")
	}
}

// klingTestAPIKey is the credential recorded by local Kling request fixtures.
const klingTestAPIKey = "kling-test-key"

// klingRecordedRequest contains one request captured by the Kling HTTP fixture.
type klingRecordedRequest struct {
	method        string
	path          string
	rawQuery      string
	authorization string
	body          map[string]any
}

// klingHTTPAnswer contains one scripted HTTP response.
type klingHTTPAnswer struct {
	statusCode  int
	body        []byte
	contentType string
	byteCount   int
}

// klingHTTPFixture records Kling requests and serves scripted creation, polling, and download
// responses.
type klingHTTPFixture struct {
	test                  *testing.T
	server                *httptest.Server
	mutex                 sync.Mutex
	recordedRequests      []klingRecordedRequest
	creationAnswers       []klingHTTPAnswer
	imagePollingAnswers   []klingHTTPAnswer
	videoPollingAnswers   []klingHTTPAnswer
	downloadAnswersByPath map[string]klingHTTPAnswer
	temporaryDirectory    string
}

// newKlingHTTPFixture starts a recording server and points the decoded provider at it.
func newKlingHTTPFixture(test *testing.T, providerConfig *catalog.Provider) *klingHTTPFixture {
	test.Helper()

	fixture := &klingHTTPFixture{
		test:                  test,
		downloadAnswersByPath: map[string]klingHTTPAnswer{},
	}
	fixture.server = httptest.NewServer(fixture)
	test.Cleanup(fixture.server.Close)

	providerConfig.Config.AdapterAPI.APIBase = fixture.server.URL
	providerConfig.Config.AdapterAPI.PollInterval = 0
	providerConfig.Config.AdapterAPI.PollTimeout = 1
	test.Setenv(providerConfig.APIKeyEnvVar, klingTestAPIKey)
	fixture.temporaryDirectory = test.TempDir()
	test.Setenv("TMPDIR", fixture.temporaryDirectory)

	return fixture
}

// ServeHTTP records one request and writes its scripted or default response.
func (fixture *klingHTTPFixture) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	fixture.test.Helper()

	requestBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		fixture.test.Errorf("✗ fixture could not read request body: %v", readErr)
	}

	decodedBody := map[string]any{}
	if len(requestBody) > 0 {
		if decodeErr := json.Unmarshal(requestBody, &decodedBody); decodeErr != nil {
			fixture.test.Errorf("✗ request body is not a JSON object: %v; body %q", decodeErr, requestBody)
		}
	}

	fixture.mutex.Lock()
	fixture.recordedRequests = append(fixture.recordedRequests, klingRecordedRequest{
		method:        request.Method,
		path:          request.URL.Path,
		rawQuery:      request.URL.RawQuery,
		authorization: request.Header.Get("Authorization"),
		body:          decodedBody,
	})
	answer := fixture.answerForRequest(request)
	fixture.mutex.Unlock()

	if answer.statusCode == 0 {
		answer.statusCode = http.StatusOK
	}

	if answer.contentType != "" {
		responseWriter.Header().Set("Content-Type", answer.contentType)
	}

	if answer.byteCount > 0 {
		responseWriter.Header().Set("Content-Length", strconv.Itoa(answer.byteCount))
	}

	responseWriter.WriteHeader(answer.statusCode)
	_, _ = responseWriter.Write(answer.body)
}

// answerForRequest returns and consumes the response selected by a request. The caller holds
// fixture.mutex.
func (fixture *klingHTTPFixture) answerForRequest(request *http.Request) klingHTTPAnswer {
	fixture.test.Helper()

	switch {
	case request.Method == http.MethodPost:
		if answer, present := firstKlingAnswer(fixture.test, &fixture.creationAnswers); present {
			return answer
		}

		if strings.HasPrefix(request.URL.Path, "/v1/images/") {
			return klingJSONAnswer(fixture.test, http.StatusOK, map[string]any{
				"code": 0,
				"data": map[string]any{"task_id": "image-task"},
			})
		}

		return klingJSONAnswer(fixture.test, http.StatusOK, map[string]any{
			"code": 0,
			"data": map[string]any{"id": "video-task"},
		})
	case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v1/images/"):
		if answer, present := firstKlingAnswer(fixture.test, &fixture.imagePollingAnswers); present {
			return answer
		}

		return fixture.completedImageAnswer([]map[string]any{{"index": 0, "url": fixture.server.URL + "/result/image-0"}})
	case request.Method == http.MethodGet && request.URL.Path == "/tasks":
		if answer, present := firstKlingAnswer(fixture.test, &fixture.videoPollingAnswers); present {
			return answer
		}

		return fixture.completedVideoAnswer(fixture.server.URL + "/result/video-0")
	case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/result/"):
		if answer, present := fixture.downloadAnswersByPath[request.URL.Path]; present {
			return answer
		}

		return klingHTTPAnswer{statusCode: http.StatusOK, body: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, contentType: "image/png"}
	}

	return klingJSONAnswer(fixture.test, http.StatusNotFound, map[string]any{"message": "fixture route not found"})
}

// completedImageAnswer returns a successful image polling envelope.
func (fixture *klingHTTPFixture) completedImageAnswer(images []map[string]any) klingHTTPAnswer {
	fixture.test.Helper()

	return klingJSONAnswer(fixture.test, http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]any{
			"task_status": "succeed",
			"task_result": map[string]any{"images": images},
		},
	})
}

// completedVideoAnswer returns a successful video polling envelope.
func (fixture *klingHTTPFixture) completedVideoAnswer(resultURL string) klingHTTPAnswer {
	fixture.test.Helper()

	return klingJSONAnswer(fixture.test, http.StatusOK, map[string]any{
		"code": 0,
		"data": []any{map[string]any{
			"status":  "succeeded",
			"outputs": []any{map[string]any{"type": "video", "url": resultURL}},
		}},
	})
}

// requestsByMethod returns recorded requests with the named method.
func (fixture *klingHTTPFixture) requestsByMethod(method string) []klingRecordedRequest {
	fixture.test.Helper()

	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()

	var matchingRequests []klingRecordedRequest

	for _, recordedRequest := range fixture.recordedRequests {
		if recordedRequest.method == method {
			matchingRequests = append(matchingRequests, recordedRequest)
		}
	}

	return matchingRequests
}

// firstKlingAnswer removes and returns the first scripted response.
func firstKlingAnswer(test testing.TB, answers *[]klingHTTPAnswer) (klingHTTPAnswer, bool) {
	test.Helper()

	if len(*answers) == 0 {
		return klingHTTPAnswer{}, false
	}

	answer := (*answers)[0]
	*answers = (*answers)[1:]

	return answer, true
}

// klingJSONAnswer encodes one response document.
func klingJSONAnswer(test testing.TB, statusCode int, document any) klingHTTPAnswer {
	test.Helper()

	responseBody, marshalErr := json.Marshal(document)
	if marshalErr != nil {
		panic(marshalErr)
	}

	return klingHTTPAnswer{statusCode: statusCode, body: responseBody, contentType: "application/json"}
}

// loadKlingTestProvider decodes the embedded configuration with the built-in flag records.
func loadKlingTestProvider(test testing.TB) catalog.Provider {
	test.Helper()

	parameterFlags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(parameterFlags, catalog.Source{ProviderID: ProviderID, ConfigBytes: ConfigJSON})
	if err != nil {
		test.Fatalf("💣 embedded Kling configuration failed to load: %v", err)
	}

	providerConfig, loaded := loadedCatalog.Provider(ProviderID)
	if !loaded {
		test.Fatalf("💣 embedded Kling provider failed to load: %v", loadedCatalog.ConfigError(ProviderID))
	}

	return providerConfig
}

// klingTestModel returns the first configured model matching the requested medium and parameter
// properties.
func klingTestModel(test *testing.T, providerConfig *catalog.Provider, medium media.Kind, inputLimit int, audioDeclared bool, discreteDuration bool) catalog.Model {
	test.Helper()

	for modelIndex := range providerConfig.Models {
		model := providerConfig.Models[modelIndex]
		if model.Media != medium {
			continue
		}

		inputParameter, acceptsInput := model.Param(params.FlagTypeInputMedia)
		if !acceptsInput || inputParameter.MaxMultiple != inputLimit {
			continue
		}

		_, hasAudio := model.Param(params.FlagType("generate-audio"))
		if hasAudio != audioDeclared {
			continue
		}

		durationParameter, hasDuration := model.Param(params.FlagTypeDuration)
		if medium == media.Video && (!hasDuration || (len(durationParameter.AllowedValues) > 0) != discreteDuration) {
			continue
		}

		return model
	}

	test.Fatalf("💣 no configured %s model matches input limit %d, audio %t, discrete duration %t", medium, inputLimit, audioDeclared, discreteDuration)

	return catalog.Model{}
}

// klingOmniImageModels returns the configured omni image models separated by 4k support.
func klingOmniImageModels(test *testing.T, providerConfig *catalog.Provider) (with4K, without4K catalog.Model) {
	test.Helper()

	for modelIndex := range providerConfig.Models {
		model := providerConfig.Models[modelIndex]

		inputParameter, acceptsInput := model.Param(params.FlagTypeInputMedia)
		if model.Media != media.Image || !acceptsInput || inputParameter.MaxMultiple <= 1 {
			continue
		}

		resolutionParameter, acceptsResolution := model.Param(params.FlagTypeResolution)
		if !acceptsResolution {
			continue
		}

		if slices.Contains(resolutionParameter.AllowedValues, "4k") {
			with4K = model
		} else {
			without4K = model
		}
	}

	if with4K.ID == "" || without4K.ID == "" {
		test.Fatalf("💣 configured omni image models do not distinguish 4k support: with=%+v without=%+v", with4K, without4K)
	}

	return with4K, without4K
}

// klingVideoModels returns configured video models by their request-shaping properties.
func klingVideoModels(test *testing.T, providerConfig *catalog.Provider) (standard, turbo, omni catalog.Model) {
	test.Helper()

	standard = klingTestModel(test, providerConfig, media.Video, 2, true, false)
	turbo = klingTestModel(test, providerConfig, media.Video, 1, false, false)
	omni = klingTestModel(test, providerConfig, media.Video, 7, true, false)

	return standard, turbo, omni
}

// generateKlingTestMedia adjusts, caps, and sends one generation through the production generator
// in the same order as the command path.
func generateKlingTestMedia(test *testing.T, providerConfig *catalog.Provider, model catalog.Model, flagInputs params.FlagInputs, mediaInputs []media.Input) (generation.Result, []params.Adjustment, error) {
	test.Helper()

	generator := constructedTestProvider(test, providerConfig)

	preparedGeneration, err := generator.AdjustParams(&model, flagInputs, mediaInputs, nil)

	changeRecords := preparedGeneration.Changes
	if err != nil {
		return generation.Result{}, changeRecords, err
	}

	run := generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: providerConfig.Identity(), Model: model}, Prompt: params.GetSetIf(true, "two-word prompt").ValOr(""), Preparation: preparedGeneration, APIKey: os.Getenv((catalog.ProvModelPair{Provider: providerConfig.Identity(), Model: model}).Provider.APIKeyEnvVar)}
	result, err := generator.Generate(test.Context(), &run)

	return result, changeRecords, err
}

// removeKlingArtifacts removes temporary artifact files created by a successful test run.
func removeKlingArtifacts(test *testing.T, artifacts []artifact.Media) {
	test.Helper()

	for _, generatedMedia := range artifacts {
		if generatedMedia.TmpPath != "" {
			if removeErr := os.Remove(generatedMedia.TmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
				test.Errorf("✗ could not remove test artifact %s: %v", generatedMedia.TmpPath, removeErr)
			}
		}
	}
}

// verifyKlingModelConfiguration verifies every configured model identity and confirms that both
// supported media families are present.
func verifyKlingModelConfiguration(test *testing.T, providerConfig *catalog.Provider) {
	test.Helper()

	configuredMedia := map[media.Kind]bool{}

	for modelIndex := range providerConfig.Models {
		model := &providerConfig.Models[modelIndex]
		if model.ID == "" || model.Name == "" {
			test.Errorf("✗ configured model has incomplete identity: %+v", *model)
		}

		switch model.Media {
		case media.Image:
			configuredMedia[media.Image] = true
		case media.Video:
			configuredMedia[media.Video] = true
		default:
			test.Errorf("✗ configured model has unsupported medium: %+v", *model)
		}
	}

	if !configuredMedia[media.Image] || !configuredMedia[media.Video] {
		test.Errorf("✗ configured media = %v, want image and video models", configuredMedia)
	}
}

// verifyKlingCatalogConstruction verifies that the configuration loads into a catalog and
// constructs its registered generator.
func verifyKlingCatalogConstruction(test *testing.T) {
	test.Helper()

	parameterFlags := params.Flags()

	catalog, err := catalog.LoadCatalog(parameterFlags, catalog.Source{ProviderID: ProviderID, ConfigBytes: ConfigJSON})
	if err != nil {
		test.Fatalf("💣 Kling configuration failed catalog construction: %v", err)
	}

	description, loaded := catalog.Provider(ProviderID)
	if !loaded {
		test.Fatal("💣 catalog omitted the Kling description")
	}

	if _, err := NewProvider(&description); err != nil {
		test.Errorf("✗ catalog could not construct the Kling generator: %v", err)
	}
}

// mapValue returns a map value or fails the current test because further field checks would be
// meaningless.
func mapValue(t *testing.T, value any, label string) map[string]any {
	t.Helper()

	mappedValue, valid := value.(map[string]any)
	if !valid {
		t.Fatalf("💣 %s = %#v, want an object", label, value)
	}

	return mappedValue
}

// changeCount returns the number of adjustment records for one flag.
func changeCount(test testing.TB, changeRecords []params.Adjustment, flagID params.FlagType) int {
	test.Helper()

	matchingCount := 0

	for _, changeRecord := range changeRecords {
		if changeRecord.FlagID == flagID {
			matchingCount++
		}
	}

	return matchingCount
}

// hasChange reports whether a record matches the specified flag, change type, input value, and wire
// value.
func hasChange(test testing.TB, changeRecords []params.Adjustment, flagID params.FlagType, changeType params.Change, inputValue, wireValue string) bool {
	test.Helper()

	for _, changeRecord := range changeRecords {
		if changeRecord.FlagID == flagID && changeRecord.Type == changeType && changeRecord.InputVal == inputValue && changeRecord.WireVal == wireValue {
			return true
		}
	}

	return false
}

// constructedTestProvider requires a valid configured generator for subsequent behavior checks.
func constructedTestProvider(test testing.TB, description *catalog.Provider) generation.Generator {
	test.Helper()

	generator, constructionErr := NewProvider(description)
	if constructionErr != nil || generator == nil {
		test.Fatalf("💣 configured generator construction failed: %v", constructionErr)
	}

	return generator
}

// verifyKlingOmniImageBody verifies the captured omni route, model, ordered image list, and
// accepted 4k resolution.
func verifyKlingOmniImageBody(test *testing.T, fixture *klingHTTPFixture, configuredModel *catalog.Model, mediaInputs []media.Input) {
	test.Helper()

	creationRequests := fixture.requestsByMethod(http.MethodPost)
	if len(creationRequests) != 1 {
		test.Fatalf("💣 omni image creation request count = %d, want 1", len(creationRequests))
	}

	request := creationRequests[0]
	if request.path != "/v1/images/omni-image" || request.body["model_name"] != configuredModel.ID {
		test.Errorf("✗ omni route or model is wrong: path=%q body=%#v", request.path, request.body)
	}

	imageList, imageListValid := request.body["image_list"].([]any)
	if !imageListValid || len(imageList) != 2 {
		test.Fatalf("💣 image_list = %#v, want two ordered objects", request.body["image_list"])
	}

	firstImage := mapValue(test, imageList[0], "first image_list item")
	secondImage := mapValue(test, imageList[1], "second image_list item")

	if firstImage["image"] != mediaInputs[0].URL || secondImage["image"] != base64.StdEncoding.EncodeToString(mediaInputs[1].Bytes) {
		test.Errorf("✗ image_list order or encoding differs: %#v", imageList)
	}

	if request.body["resolution"] != "4k" {
		test.Errorf("✗ omni request resolution = %#v, want 4k", request.body["resolution"])
	}
}

// verifyKlingVideoRequest verifies one captured route, its permanent field omission, settings, and
// ordered contents.
func verifyKlingVideoRequest(test *testing.T, request klingRecordedRequest, caseName string, flagInputs params.FlagInputs, expectedPath string, expectedTypes []string, expectedAudio any, expectAudioAbsent bool) {
	test.Helper()

	if request.path != expectedPath {
		test.Errorf("✗ %s path = %q, want %q", caseName, request.path, expectedPath)
	}

	if _, optionsPresent := request.body["options"]; optionsPresent {
		test.Errorf("✗ %s request carries forbidden options: %#v", caseName, request.body["options"])
	}

	settings := mapValue(test, request.body["settings"], caseName+" settings")
	verifyKlingVideoSettings(test, caseName, settings, flagInputs, expectedAudio, expectAudioAbsent)
	verifyKlingVideoContents(test, caseName, request.body, expectedTypes)
}

// verifyKlingVideoSettings verifies the supplied aspect ratio and audio field.
func verifyKlingVideoSettings(test *testing.T, caseName string, settings map[string]any, flagInputs params.FlagInputs, expectedAudio any, expectAudioAbsent bool) {
	test.Helper()

	if settings["aspect_ratio"] != flagInputs[params.FlagTypeAspect] {
		test.Errorf("✗ %s aspect_ratio = %#v, want supplied value %#v", caseName, settings["aspect_ratio"], flagInputs[params.FlagTypeAspect])
	}

	if expectAudioAbsent {
		if _, audioPresent := settings["audio"]; audioPresent {
			test.Errorf("✗ %s settings unexpectedly carry audio: %#v", caseName, settings)
		}

		return
	}

	if settings["audio"] != expectedAudio {
		test.Errorf("✗ %s audio = %#v, want %#v", caseName, settings["audio"], expectedAudio)
	}
}

// verifyKlingVideoContents verifies a top-level text prompt or the ordered content record types
// expected by an image or omni route.
func verifyKlingVideoContents(test *testing.T, caseName string, requestBody map[string]any, expectedTypes []string) {
	test.Helper()

	if len(expectedTypes) == 0 {
		if requestBody["prompt"] != "two-word prompt" {
			test.Errorf("✗ %s prompt = %#v, want top-level prompt", caseName, requestBody["prompt"])
		}

		if _, contentsPresent := requestBody["contents"]; contentsPresent {
			test.Errorf("✗ %s text request unexpectedly carries contents", caseName)
		}

		return
	}

	contents, contentsValid := requestBody["contents"].([]any)
	if !contentsValid || len(contents) != len(expectedTypes) {
		test.Errorf("✗ %s contents = %#v, want types %v", caseName, requestBody["contents"], expectedTypes)

		return
	}

	for contentIndex, expectedType := range expectedTypes {
		contentItem := mapValue(test, contents[contentIndex], caseName+" content")
		if contentItem["type"] != expectedType {
			test.Errorf("✗ %s content %d type = %#v, want %q", caseName, contentIndex, contentItem["type"], expectedType)
		}
	}
}

// verifyKlingAcceptedInputRequests verifies that WebP bytes and the lone closing-frame role reach
// their selected request bodies unchanged.
func verifyKlingAcceptedInputRequests(test *testing.T, fixture *klingHTTPFixture, webPBytes []byte, closingURL string) {
	test.Helper()

	creationRequests := fixture.requestsByMethod(http.MethodPost)
	if len(creationRequests) != 2 {
		test.Fatalf("💣 accepted-input request count = %d, want 2", len(creationRequests))
	}

	if creationRequests[0].body["image"] != base64.StdEncoding.EncodeToString(webPBytes) {
		test.Errorf("✗ WebP bytes were changed or prefixed: %#v", creationRequests[0].body["image"])
	}

	contents, _ := creationRequests[1].body["contents"].([]any)
	if len(contents) != 2 || mapValue(test, contents[1], "last-frame content")["type"] != "last_frame" || mapValue(test, contents[1], "last-frame content")["url"] != closingURL {
		test.Errorf("✗ closing frame was not sent unchanged: %#v", creationRequests[1].body["contents"])
	}
}
