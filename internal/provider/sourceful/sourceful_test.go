package sourceful

// Invariants tested:
//  1. Sourceful redirected result: When the result URL redirects to another server, Generate must
//     return one artifact without an error and omit X-Api-Key from both download requests.
//  2. Sourceful request and response branches: Given output under data.result, classifyJobResponse
//     must complete with the supplied URL and MIME type. NewProvider must return
//     ErrProvConfigNoAdapterAPI for absent adapter settings. Generate must classify a closed
//     creation endpoint as ErrTransportRequest, polling HTTP 500 with a message as
//     ErrResponseStatus and ErrResponseServer, and missing job status as ErrResponseNoData.
//     jobPoll.Poll must return ErrTransportRequest for a closed polling endpoint.
//  3. Sourceful text request: Without input media, Generate must POST to /v2.5/generations/t2i with
//     the selected model, supplied instruction, and a distinct nonempty idempotency key for each
//     call. Supplied output and thinking controls must appear at their configured paths.
//     Unrequested output, thinkingLevel, and enhancePrompt fields, and the listed unsupported
//     fields, must be absent.
//  4. Sourceful image request: Given a remote image followed by local image bytes, Generate must
//     POST once to /v2.5/generations/i2i and preserve that order in imageUrls. The first value must
//     be the unchanged URL and the second must encode the original local bytes as a data URI.
//  5. Sourceful job states: For every configured pending status followed by completion, Generate
//     must make two polls and return one artifact. A ready response must return one artifact. Each
//     configured failure must return ErrResponseGen containing lastErrorMessage, and an
//     unconfigured status must return ErrResponseUnknown.
//  6. Sourceful download: Given PNG or WebP output MIME types, Generate must return one artifact
//     with the matching extension; an unusable MIME type must select the configured fallback
//     extension. API requests must carry X-Api-Key, and the single artifact download must omit it.
//  7. Sourceful configuration: catalog.LoadCatalog must accept the embedded Sourceful configuration
//     and return the package's provider ID, nonempty display name and API key environment variable,
//     and two named image models with nonempty IDs. NewProvider must construct a nonnil generator
//     without an error.
//  8. Sourceful parameter adjustment: Given one more image than the model permits, a first-frame
//     marker, and an unsupported duration, AdjustParams must retain only the permitted image count,
//     remove the retained image's frame marker, and omit duration. It must preserve the caller's
//     marker and return one Capped, one Conformed, and one Ignored record for those changes without
//     an error.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestSourcefulRedirectedResult verifies invariant #1: Sourceful redirected result.
//
// What is being tested:
// When the result URL redirects to another server, Generate must return one artifact without an
// error and omit X-Api-Key from both download requests.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSourcefulRedirectedResult(t *testing.T) {
	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	redirectTargetCredential := ""
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		redirectTargetCredential = request.Header.Get("X-Api-Key")
		_, _ = responseWriter.Write([]byte("redirected image bytes"))
	}))
	t.Cleanup(redirectTarget.Close)

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &providerConfig)
	fixture.downloadRedirectTo = redirectTarget.URL + "/image"

	result, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
	if generationErr != nil || len(result.Artifacts) != 1 {
		t.Errorf("✗ redirected result = %+v, %v; want one artifact", result, generationErr)
	}

	initialDownloads := fixture.requestsMatching(http.MethodGet, "/result")
	if len(initialDownloads) != 1 || initialDownloads[0].credential != "" || redirectTargetCredential != "" {
		t.Errorf("✗ redirect credentials = initial %#v, target %q; want neither request credentialed", initialDownloads, redirectTargetCredential)
	}

	if !t.Failed() {
		t.Log("✓ a redirected Sourceful result lands one artifact without carrying the Sourceful credential")
	}
}

// TestSourcefulCoverageBranches verifies invariant #2: Sourceful request and response branches.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// What is being tested:
// Given output under data.result, classifyJobResponse must complete with the supplied URL and MIME
// type. NewProvider must return ErrProvConfigNoAdapterAPI for absent adapter settings. Generate
// must classify a closed creation endpoint as ErrTransportRequest, polling HTTP 500 with a message
// as ErrResponseStatus and ErrResponseServer, and missing job status as ErrResponseNoData.
// jobPoll.Poll must return ErrTransportRequest for a closed polling endpoint.
func TestSourcefulCoverageBranches(t *testing.T) {
	t.Run("version 1.96.0 completed response", verifySourcefulVersion1960Response)
	t.Run("missing adapter API", verifySourcefulMissingAdapterAPI)
	t.Run("creation transport", verifySourcefulCreationTransportFailure)
	t.Run("poll error status", verifySourcefulPollErrorStatus)
	t.Run("missing job status", verifySourcefulMissingJobStatus)
	t.Run("poll transport", verifySourcefulPollTransportFailure)

	if !t.Failed() {
		t.Log("✓ Sourceful configuration, credential, transport, status, and missing-data branches are covered")
	}
}

// TestSourcefulTextRequest verifies invariant #3: Sourceful text request.
//
// What is being tested:
// Without input media, Generate must POST to /v2.5/generations/t2i with the selected model,
// supplied instruction, and a distinct nonempty idempotency key for each call. Supplied output and
// thinking controls must appear at their configured paths. Unrequested output, thinkingLevel, and
// enhancePrompt fields, and the listed unsupported fields, must be absent.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSourcefulTextRequest(t *testing.T) {
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
	completedStatus := providerConfig.Config.AdapterAPI.ReadyStatusText
	fixture.pollingAnswers = []sourcefulTestAnswer{
		{statusCode: http.StatusOK, answerBytes: sourcefulCompletedPollDocument(t, fixture.server.URL+"/result", "image/png", completedStatus)},
		{statusCode: http.StatusOK, answerBytes: sourcefulCompletedPollDocument(t, fixture.server.URL+"/result", "image/png", completedStatus)},
	}

	parameterValues := sourcefulParameterValues(t, &configuredModel)
	if _, err := generateSourcefulTestImage(t, &providerConfig, configuredModel, parameterValues, nil); err != nil {
		t.Errorf("✗ configured-parameter generation failed: %v", err)
	}

	if _, err := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil); err != nil {
		t.Errorf("✗ no-parameter generation failed: %v", err)
	}

	creationRequests := fixture.requestsMatching(http.MethodPost, "/v2.5/generations/t2i")
	if len(creationRequests) != 2 {
		t.Errorf("✗ text creation requests = %d, want 2", len(creationRequests))

		return
	}

	configuredRequest := decodeSourcefulRequest(t, creationRequests[0])
	emptyParameterRequest := decodeSourcefulRequest(t, creationRequests[1])
	verifySourcefulTextRequestFields(t, configuredRequest, emptyParameterRequest, configuredModel)
	verifySourcefulParameterFields(t, configuredRequest, parameterValues)
	verifySourcefulOmittedFields(t, configuredRequest, emptyParameterRequest)

	if !t.Failed() {
		t.Log("✓ Sourceful text requests carry only the required and supplied fields with a fresh idempotency key")
	}
}

// TestSourcefulImageRequest verifies invariant #4: Sourceful image request.
//
// What is being tested:
// Given a remote image followed by local image bytes, Generate must POST once to
// /v2.5/generations/i2i and preserve that order in imageUrls. The first value must be the unchanged
// URL and the second must encode the original local bytes as a data URI.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSourcefulImageRequest(t *testing.T) {
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
	remoteURL := "https://media.example/sourceful-reference.png"
	localBytes := []byte("local reference bytes")
	mediaInputs := []media.Input{
		{URL: remoteURL, MIME: "image/png"},
		{Bytes: localBytes, MIME: "image/png", Filepath: "reference.png"},
	}

	if _, err := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, mediaInputs); err != nil {
		t.Errorf("✗ image-to-image generation failed: %v", err)
	}

	creationRequests := fixture.requestsMatching(http.MethodPost, "/v2.5/generations/i2i")
	if len(creationRequests) != 1 {
		t.Errorf("✗ image creation requests = %d, want 1", len(creationRequests))

		return
	}

	requestDocument := decodeSourcefulRequest(t, creationRequests[0])
	imageURLs, imageURLsValid := requestDocument["imageUrls"].([]any)

	if !imageURLsValid || len(imageURLs) != len(mediaInputs) {
		t.Errorf("✗ imageUrls = %#v, want %d ordered inputs", requestDocument["imageUrls"], len(mediaInputs))

		return
	}

	if imageURLs[0] != remoteURL {
		t.Errorf("✗ first image URL = %v, want unchanged URL %q", imageURLs[0], remoteURL)
	}

	dataURI, dataURIValid := imageURLs[1].(string)
	commaIndex := strings.IndexByte(dataURI, ',')

	if !dataURIValid || commaIndex < 0 {
		t.Errorf("✗ second image URL = %v, want a data URI", imageURLs[1])
	} else {
		decodedBytes, decodeErr := base64.StdEncoding.DecodeString(dataURI[commaIndex+1:])
		if decodeErr != nil || !slices.Equal(decodedBytes, localBytes) {
			t.Errorf("✗ data URI decode = %q, %v; want local bytes %q", decodedBytes, decodeErr, localBytes)
		}
	}

	if !t.Failed() {
		t.Log("✓ Sourceful image requests preserve remote URLs and encode local bytes in input order")
	}
}

// TestSourcefulJobStates verifies invariant #5: Sourceful job states.
//
// What is being tested:
// For every configured pending status followed by completion, Generate must make two polls and
// return one artifact. A ready response must return one artifact. Each configured failure must
// return ErrResponseGen containing lastErrorMessage, and an unconfigured status must return
// ErrResponseUnknown.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSourcefulJobStates(t *testing.T) {
	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	adapterAPI := providerConfig.Config.AdapterAPI

	for _, pendingStatus := range adapterAPI.PendingStatusText {
		t.Run(pendingStatus, func(t *testing.T) {
			verifySourcefulPendingStatus(t, pendingStatus)
		})
	}

	t.Run(adapterAPI.ReadyStatusText, verifySourcefulCompletedStatus)

	for _, failedStatus := range adapterAPI.FailedStatusText {
		t.Run(failedStatus, func(t *testing.T) {
			verifySourcefulFailedStatus(t, failedStatus)
		})
	}

	t.Run("undocumented status", verifySourcefulUnknownStatus)

	if !t.Failed() {
		t.Log("✓ all configured Sourceful job states and an undocumented state have terminal classifications")
	}
}

// TestSourcefulDownload verifies invariant #6: Sourceful download.
//
// What is being tested:
// Given PNG or WebP output MIME types, Generate must return one artifact with the matching
// extension; an unusable MIME type must select the configured fallback extension. API requests must
// carry X-Api-Key, and the single artifact download must omit it.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSourcefulDownload(t *testing.T) {
	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	extensionCases := []struct {
		mimeType          string
		expectedExtension string
	}{
		{mimeType: "image/png", expectedExtension: "." + media.FormatPNG},
		{mimeType: "image/webp", expectedExtension: "." + media.FormatWebP},
		{mimeType: "unusable", expectedExtension: providerConfig.Config.AdapterAPI.ImageFallbackExt},
	}

	for _, extensionCase := range extensionCases {
		t.Run(extensionCase.expectedExtension+" from "+extensionCase.mimeType, func(t *testing.T) {
			verifySourcefulDownloadExtension(t, extensionCase.mimeType, extensionCase.expectedExtension)
		})
	}

	if !t.Failed() {
		t.Log("✓ Sourceful downloads isolate credentials and use returned MIME types before the configured fallback")
	}
}

// TestSourcefulConfiguration verifies invariant #7: Sourceful configuration.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// What is being tested:
// catalog.LoadCatalog must accept the embedded Sourceful configuration and return the package's
// provider ID, nonempty display name and API key environment variable, and two named image models
// with nonempty IDs. NewProvider must construct a nonnil generator without an error.
func TestSourcefulConfiguration(t *testing.T) {
	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	if providerConfig.ID != ProviderID {
		t.Errorf("✗ provider ID = %q, want package provider ID %q", providerConfig.ID, ProviderID)
	}

	if providerConfig.DisplayName == "" || providerConfig.APIKeyEnvVar == "" {
		t.Errorf("✗ provider identity is incomplete: %+v", providerConfig.Identity())
	}

	if len(providerConfig.Models) != 2 {
		t.Errorf("✗ configured model count = %d, want 2", len(providerConfig.Models))
	}

	for modelIndex := range providerConfig.Models {
		configuredModel := &providerConfig.Models[modelIndex]
		if configuredModel.ID == "" || configuredModel.Name == "" || configuredModel.Media != media.Image {
			t.Errorf("✗ configured model is incomplete: %+v", *configuredModel)
		}
	}

	if _, err := catalog.LoadCatalog(params.Flags(), catalog.Source{ProviderID: ProviderID, ConfigBytes: ConfigJSON}); err != nil {
		t.Errorf("✗ Sourceful config does not validate into a catalog: %v", err)
	}

	if constructedTestProvider(t, &providerConfig) == nil {
		t.Errorf("✗ constructedTestProvider(t, ) returned no generator")
	}

	if !t.Failed() {
		t.Log("✓ the embedded Sourceful config validates and composes a generator over both image models")
	}
}

// TestSourcefulAdjustParams verifies invariant #8: Sourceful parameter adjustment.
//
// What is being tested:
// Given one more image than the model permits, a first-frame marker, and an unsupported duration,
// AdjustParams must retain only the permitted image count, remove the retained image's frame
// marker, and omit duration. It must preserve the caller's marker and return one Capped, one
// Conformed, and one Ignored record for those changes without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSourcefulAdjustParams(t *testing.T) {
	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	inputParameter, acceptsInput := configuredModel.Param(params.FlagTypeInputMedia)
	if !acceptsInput || inputParameter.MaxMultiple <= 0 {
		t.Errorf("✗ model input-media declaration = %+v, want a positive cap", inputParameter)

		return
	}

	mediaInputs := make([]media.Input, inputParameter.MaxMultiple+1)
	for mediaIndex := range mediaInputs {
		mediaInputs[mediaIndex] = media.Input{
			URL:  "https://media.example/reference.png",
			MIME: "image/png",
		}
	}

	mediaInputs[0].FrameAnchor = media.FrameFirst

	undeclaredParameter := params.FlagTypeDuration
	flagInputs := params.FlagInputs{undeclaredParameter: 7}

	preparedGeneration, err := constructedTestProvider(t, &providerConfig).AdjustParams(&configuredModel, flagInputs, mediaInputs, nil)
	generationParams, changeRecords := preparedGeneration.Params, preparedGeneration.Changes

	if err != nil {
		t.Errorf("✗ AdjustParams() error = %v, want nil", err)
	}

	if _, included := generationParams[undeclaredParameter]; included {
		t.Errorf("✗ undeclared parameter %q reached adjusted values", undeclaredParameter)
	}

	if changeCount := countSourcefulChanges(t, changeRecords, params.FlagTypeInputMedia, params.ChangeCapped); changeCount != 1 {
		t.Errorf("✗ input cap records = %d, want 1: %+v", changeCount, changeRecords)
	}

	if changeCount := countSourcefulChanges(t, changeRecords, params.FlagTypeInputMedia, params.ChangeConformed); changeCount != 1 {
		t.Errorf("✗ frame-prefix conformance records = %d, want 1: %+v", changeCount, changeRecords)
	}

	if changeCount := countSourcefulChanges(t, changeRecords, undeclaredParameter, params.ChangeIgnored); changeCount != 1 {
		t.Errorf("✗ ignored undeclared-parameter records = %d, want 1: %+v", changeCount, changeRecords)
	}

	retainedMedia := preparedGeneration.InputMedia
	if len(retainedMedia) != inputParameter.MaxMultiple {
		t.Errorf("✗ retained input count = %d, want configured cap %d", len(retainedMedia), inputParameter.MaxMultiple)

		return
	}

	if mediaInputs[0].FrameAnchor != media.FrameFirst {
		t.Error("✗ adjustment changed the caller frame marker")
	}

	if retainedMedia[0].FrameAnchor != "" || retainedMedia[0].HasFrame() {
		t.Errorf("✗ retained first input still carries its frame prefix: %+v", retainedMedia[0])
	}

	if !t.Failed() {
		t.Log("✓ Sourceful adjustment retains the configured input cap and records cap, conformance, and ignored changes")
	}
}

// sourcefulTestHeaderValue is the credential recorded by local Sourceful request fixtures.
const sourcefulTestHeaderValue = "local-sourceful-fixture"

// sourcefulRecordedRequest is one HTTP request observed by the local Sourceful fixture.
type sourcefulRecordedRequest struct {
	method       string
	path         string
	credential   string
	requestBytes []byte
}

// sourcefulTestAnswer is one HTTP answer returned by the local Sourceful fixture.
type sourcefulTestAnswer struct {
	statusCode  int
	contentType string
	location    string
	byteCount   int64
	answerBytes []byte
}

// sourcefulHTTPFixture records Sourceful requests and returns configurable creation, polling, and
// download answers from a local HTTP server.
type sourcefulHTTPFixture struct {
	test               *testing.T
	mutex              sync.Mutex
	server             *httptest.Server
	recordedRequests   []sourcefulRecordedRequest
	creationAnswer     sourcefulTestAnswer
	pollingAnswers     []sourcefulTestAnswer
	nextPollingAnswer  int
	downloadAnswer     sourcefulTestAnswer
	downloadRedirectTo string
	completedStatus    string
}

// newSourcefulHTTPFixture starts a successful Sourceful server and closes it with the test.
func newSourcefulHTTPFixture(t *testing.T, providerConfig *catalog.Provider) *sourcefulHTTPFixture {
	t.Helper()

	fixture := &sourcefulHTTPFixture{
		test:           t,
		creationAnswer: sourcefulTestAnswer{statusCode: http.StatusCreated},
		downloadAnswer: sourcefulTestAnswer{statusCode: http.StatusOK, answerBytes: []byte("sourceful image bytes")},
	}
	fixture.server = httptest.NewServer(fixture)
	t.Cleanup(fixture.server.Close)

	fixture.creationAnswer.answerBytes = sourcefulJSONDocument(t, map[string]any{
		"data": map[string]any{"jobId": "sourceful-test-job"},
	})

	if providerConfig.Config == nil || providerConfig.Config.AdapterAPI == nil {
		t.Errorf("✗ Sourceful config has no adapter API")

		return fixture
	}

	providerConfig.Config.AdapterAPI.APIBase = fixture.server.URL
	providerConfig.Config.AdapterAPI.PollInterval = 0
	providerConfig.Config.AdapterAPI.PollTimeout = 1
	fixture.completedStatus = providerConfig.Config.AdapterAPI.ReadyStatusText

	return fixture
}

// requestsMatching returns recorded requests with the specified method and path.
func (fixture *sourcefulHTTPFixture) requestsMatching(method, path string) []sourcefulRecordedRequest {
	fixture.test.Helper()

	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()

	var matchingRequests []sourcefulRecordedRequest

	for _, recordedRequest := range fixture.recordedRequests {
		if recordedRequest.method == method && recordedRequest.path == path {
			matchingRequests = append(matchingRequests, recordedRequest)
		}
	}

	return matchingRequests
}

// requestCountByPrefix returns the number of recorded requests matching a method and path prefix.
func (fixture *sourcefulHTTPFixture) requestCountByPrefix(method, pathPrefix string) int {
	fixture.test.Helper()

	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()

	matchingRequests := 0

	for _, recordedRequest := range fixture.recordedRequests {
		if recordedRequest.method == method && strings.HasPrefix(recordedRequest.path, pathPrefix) {
			matchingRequests++
		}
	}

	return matchingRequests
}

// sourcefulJSONDocument encodes a fixture document and panics only on impossible map encoding.
func sourcefulJSONDocument(test testing.TB, document map[string]any) []byte {
	test.Helper()

	documentBytes, err := json.Marshal(document)
	if err != nil {
		panic(err)
	}

	return documentBytes
}

// sourcefulPollDocument returns a polling document for a job status and optional error text.
func sourcefulPollDocument(test testing.TB, statusWord, failureText string) []byte {
	test.Helper()

	jobDocument := map[string]any{"status": statusWord}
	if failureText != "" {
		jobDocument["lastErrorMessage"] = failureText
	}

	return sourcefulJSONDocument(test, map[string]any{
		"data": map[string]any{"job": jobDocument},
	})
}

// sourcefulCompletedPollDocument returns a completed polling document with one output.
func sourcefulCompletedPollDocument(test testing.TB, resultURL, mimeType, completedStatus string) []byte {
	test.Helper()

	return sourcefulJSONDocument(test, map[string]any{
		"data": map[string]any{
			"job": map[string]any{
				"status": completedStatus,
				"result": map[string]any{
					"output": map[string]any{"url": resultURL, "mimeType": mimeType},
				},
			},
		},
	})
}

// sourcefulTestRun creates one generation request from the decoded provider and model.
func sourcefulTestRun(test testing.TB, providerConfig catalog.Provider, configuredModel catalog.Model, prompt string, generationParams params.Values, mediaInputs []media.Input) generation.Generation {
	test.Helper()

	return generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: providerConfig.Identity(), Model: configuredModel}, Prompt: params.GetSetIf(true, prompt).ValOr(""), Preparation: generation.Preparation{Params: generationParams, InputMedia: mediaInputs}, APIKey: os.Getenv((catalog.ProvModelPair{Provider: providerConfig.Identity(), Model: configuredModel}).Provider.APIKeyEnvVar)}
}

// generateSourcefulTestImage invokes the composed generator with a local-server credential.
func generateSourcefulTestImage(t *testing.T, providerConfig *catalog.Provider, configuredModel catalog.Model, generationParams params.Values, mediaInputs []media.Input) (generation.Result, error) {
	t.Helper()
	t.Setenv(providerConfig.APIKeyEnvVar, sourcefulTestHeaderValue)

	generationRequest := sourcefulTestRun(t, *providerConfig, configuredModel, "two words", generationParams, mediaInputs)

	return constructedTestProvider(t, providerConfig).Generate(context.Background(), &generationRequest)
}

// sourcefulParameterValues returns one valid value for every configured body parameter.
func sourcefulParameterValues(t *testing.T, configuredModel *catalog.Model) params.Values {
	t.Helper()

	parameterValues := params.Values{}

	for _, parameterConfig := range configuredModel.Params {
		switch parameterConfig.FlagID {
		case params.FlagTypeInputMedia:
			continue
		case params.FlagType("prompt-upsampling"):
			parameterValues[parameterConfig.FlagID] = true
		default:
			if len(parameterConfig.AllowedValues) == 0 {
				t.Errorf("✗ configured Sourceful parameter %q has no testable value", parameterConfig.FlagID)

				continue
			}

			parameterValues[parameterConfig.FlagID] = parameterConfig.AllowedValues[0]
		}
	}

	return parameterValues
}

// decodeSourcefulRequest decodes a recorded request body into its JSON object.
func decodeSourcefulRequest(t *testing.T, recordedRequest sourcefulRecordedRequest) map[string]any {
	t.Helper()

	var requestDocument map[string]any
	if err := json.Unmarshal(recordedRequest.requestBytes, &requestDocument); err != nil {
		t.Errorf("✗ request JSON failed to decode: %v", err)

		return nil
	}

	return requestDocument
}

// sourcefulNestedValue returns a nested request value and whether the full path exists.
func sourcefulNestedValue(test testing.TB, requestDocument map[string]any, fieldPath ...string) (any, bool) {
	test.Helper()

	currentDocument := requestDocument
	for fieldIndex, fieldName := range fieldPath {
		fieldValue, fieldExists := currentDocument[fieldName]
		if !fieldExists {
			return nil, false
		}

		if fieldIndex == len(fieldPath)-1 {
			return fieldValue, true
		}

		nestedDocument, nestedExists := fieldValue.(map[string]any)
		if !nestedExists {
			return nil, false
		}

		currentDocument = nestedDocument
	}

	return nil, false
}

// sourcefulDocumentContainsKey reports whether a key occurs anywhere in a JSON object.
func sourcefulDocumentContainsKey(test testing.TB, document map[string]any, forbiddenKey string) bool {
	test.Helper()

	for fieldName, fieldValue := range document {
		if fieldName == forbiddenKey {
			return true
		}

		switch nestedValue := fieldValue.(type) {
		case map[string]any:
			if sourcefulDocumentContainsKey(test, nestedValue, forbiddenKey) {
				return true
			}
		case []any:
			for _, arrayValue := range nestedValue {
				if nestedDocument, isDocument := arrayValue.(map[string]any); isDocument && sourcefulDocumentContainsKey(test, nestedDocument, forbiddenKey) {
					return true
				}
			}
		}
	}

	return false
}

// verifySourcefulTextRequestFields checks required fields and idempotency keys across two requests.
func verifySourcefulTextRequestFields(t *testing.T, configuredRequest, emptyParameterRequest map[string]any, configuredModel catalog.Model) {
	t.Helper()

	if configuredRequest["instruction"] != "two words" || configuredRequest["model"] != configuredModel.ID {
		t.Errorf("✗ required creation fields = %+v, want prompt and configured model", configuredRequest)
	}

	firstIdempotencyKey, firstKeyValid := configuredRequest["idempotencyKey"].(string)
	secondIdempotencyKey, secondKeyValid := emptyParameterRequest["idempotencyKey"].(string)

	if !firstKeyValid || !secondKeyValid || firstIdempotencyKey == "" || secondIdempotencyKey == "" || firstIdempotencyKey == secondIdempotencyKey {
		t.Errorf("✗ idempotency keys = %q and %q, want distinct nonempty strings", firstIdempotencyKey, secondIdempotencyKey)
	}
}

// verifySourcefulParameterFields checks every configured value at its Sourceful request path.
func verifySourcefulParameterFields(t *testing.T, configuredRequest map[string]any, parameterValues params.Values) {
	t.Helper()

	requestFieldPaths := map[params.FlagType][]string{
		params.FlagTypeAspect:                {"output", "aspectRatio"},
		params.FlagTypeResolution:            {"output", "resolution"},
		params.FlagTypeOutputFormat:          {"output", "format"},
		params.FlagType("background"):        {"output", "background", "mode"},
		params.FlagTypeThinkingLevel:         {"thinkingLevel"},
		params.FlagType("prompt-upsampling"): {"enhancePrompt"},
	}
	for parameterName, parameterValue := range parameterValues {
		fieldPath, pathDeclared := requestFieldPaths[parameterName]
		if !pathDeclared {
			t.Errorf("✗ no request-field assertion for configured parameter %q", parameterName)

			continue
		}

		requestValue, fieldExists := sourcefulNestedValue(t, configuredRequest, fieldPath...)
		if !fieldExists || requestValue != parameterValue {
			t.Errorf("✗ request field %s = %v, want configured value %v", strings.Join(fieldPath, "."), requestValue, parameterValue)
		}
	}
}

// verifySourcefulOmittedFields checks absent parameters and fields Bild never sends.
func verifySourcefulOmittedFields(t *testing.T, configuredRequest, emptyParameterRequest map[string]any) {
	t.Helper()

	for _, absentField := range []string{"output", "thinkingLevel", "enhancePrompt"} {
		if _, fieldExists := emptyParameterRequest[absentField]; fieldExists {
			t.Errorf("✗ no-parameter request unexpectedly carries %q: %+v", absentField, emptyParameterRequest)
		}
	}

	for _, forbiddenField := range []string{"safetyChecker", "scoringPrompt", "scoringRubric", "fontInputs", "color", string(params.FlagTypeDuration)} {
		if sourcefulDocumentContainsKey(t, configuredRequest, forbiddenField) || sourcefulDocumentContainsKey(t, emptyParameterRequest, forbiddenField) {
			t.Errorf("✗ request carries forbidden field %q", forbiddenField)
		}
	}
}

// verifySourcefulPendingStatus checks that one pending status causes a second poll.
func verifySourcefulPendingStatus(t *testing.T, pendingStatus string) {
	t.Helper()

	pendingProvider, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &pendingProvider)
	if !modelLoaded {
		return
	}

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &pendingProvider)
	fixture.pollingAnswers = []sourcefulTestAnswer{
		{statusCode: http.StatusOK, answerBytes: sourcefulPollDocument(t, pendingStatus, "")},
		{statusCode: http.StatusOK, answerBytes: sourcefulCompletedPollDocument(t, fixture.server.URL+"/result", "image/png", pendingProvider.Config.AdapterAPI.ReadyStatusText)},
	}

	result, generationErr := generateSourcefulTestImage(t, &pendingProvider, configuredModel, params.Values{}, nil)
	if generationErr != nil || len(result.Artifacts) != 1 {
		t.Errorf("✗ %q state result = %+v, %v; want a later completed artifact", pendingStatus, result, generationErr)
	}

	if pollCount := fixture.requestCountByPrefix(http.MethodGet, "/v2.5/generations/"); pollCount != 2 {
		t.Errorf("✗ poll requests after %q = %d, want 2", pendingStatus, pollCount)
	}

	if !t.Failed() {
		t.Logf("✓ Sourceful state %q continues polling until completion", pendingStatus)
	}
}

// verifySourcefulCompletedStatus checks that the ready status downloads one artifact.
func verifySourcefulCompletedStatus(t *testing.T) {
	t.Helper()

	readyProvider, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &readyProvider)
	if !modelLoaded {
		return
	}

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &readyProvider)
	fixture.pollingAnswers = []sourcefulTestAnswer{{
		statusCode:  http.StatusOK,
		answerBytes: sourcefulCompletedPollDocument(t, fixture.server.URL+"/result", "image/png", readyProvider.Config.AdapterAPI.ReadyStatusText),
	}}

	result, generationErr := generateSourcefulTestImage(t, &readyProvider, configuredModel, params.Values{}, nil)
	if generationErr != nil || len(result.Artifacts) != 1 {
		t.Errorf("✗ completed state result = %+v, %v; want one artifact", result, generationErr)
	}

	if !t.Failed() {
		t.Logf("✓ Sourceful state %q downloads the completed artifact", readyProvider.Config.AdapterAPI.ReadyStatusText)
	}
}

// verifySourcefulFailedStatus checks one configured failure status and its provider message.
func verifySourcefulFailedStatus(t *testing.T, failedStatus string) {
	t.Helper()

	failedProvider, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &failedProvider)
	if !modelLoaded {
		return
	}

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &failedProvider)
	failureText := "Sourceful test job failed"
	fixture.pollingAnswers = []sourcefulTestAnswer{{
		statusCode:  http.StatusOK,
		answerBytes: sourcefulPollDocument(t, failedStatus, failureText),
	}}

	_, generationErr := generateSourcefulTestImage(t, &failedProvider, configuredModel, params.Values{}, nil)
	if !errors.Is(generationErr, errs.ErrResponseGen) || generationErr == nil || !strings.Contains(generationErr.Error(), failureText) {
		t.Errorf("✗ %q state error = %v, want generation classification and lastErrorMessage", failedStatus, generationErr)
	}

	if !t.Failed() {
		t.Logf("✓ Sourceful state %q preserves lastErrorMessage in a generation failure", failedStatus)
	}
}

// verifySourcefulUnknownStatus checks that an unconfigured status is classified as unknown.
func verifySourcefulUnknownStatus(t *testing.T) {
	t.Helper()

	unknownProvider, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &unknownProvider)
	if !modelLoaded {
		return
	}

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &unknownProvider)
	fixture.pollingAnswers = []sourcefulTestAnswer{{
		statusCode:  http.StatusOK,
		answerBytes: sourcefulPollDocument(t, "undocumented-sourceful-test-state", ""),
	}}

	_, generationErr := generateSourcefulTestImage(t, &unknownProvider, configuredModel, params.Values{}, nil)
	if !errors.Is(generationErr, errs.ErrResponseUnknown) {
		t.Errorf("✗ undocumented state error = %v, want unexpected-status classification", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ an undocumented Sourceful state returns an unexpected-status error")
	}
}

// verifySourcefulDownloadExtension checks one MIME-to-extension result and credential boundary.
func verifySourcefulDownloadExtension(t *testing.T, mimeType, expectedExtension string) {
	t.Helper()

	extensionProvider, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &extensionProvider)
	if !modelLoaded {
		return
	}

	t.Setenv("TMPDIR", t.TempDir())
	fixture := newSourcefulHTTPFixture(t, &extensionProvider)
	fixture.pollingAnswers = []sourcefulTestAnswer{{
		statusCode: http.StatusOK,
		answerBytes: sourcefulCompletedPollDocument(t,
			fixture.server.URL+"/result",
			mimeType,
			extensionProvider.Config.AdapterAPI.ReadyStatusText,
		),
	}}

	result, generationErr := generateSourcefulTestImage(t, &extensionProvider, configuredModel, params.Values{}, nil)
	if generationErr != nil || len(result.Artifacts) != 1 {
		t.Errorf("✗ download result = %+v, %v; want one artifact", result, generationErr)
	} else if result.Artifacts[0].FileExt != expectedExtension {
		t.Errorf("✗ artifact extension = %q, want %q", result.Artifacts[0].FileExt, expectedExtension)
	}

	downloadRequests := fixture.requestsMatching(http.MethodGet, "/result")
	if len(downloadRequests) != 1 {
		t.Errorf("✗ result downloads = %d, want 1", len(downloadRequests))
	} else if downloadRequests[0].credential != "" {
		t.Errorf("✗ result download carried Sourceful credential %q", downloadRequests[0].credential)
	}

	for _, recordedRequest := range fixture.recordedRequests {
		if strings.HasPrefix(recordedRequest.path, "/v2.5/generations/") && recordedRequest.credential != sourcefulTestHeaderValue {
			t.Errorf("✗ Sourceful API request credential = %q, want test key", recordedRequest.credential)
		}
	}

	if !t.Failed() {
		t.Logf("✓ Sourceful MIME %q lands %q and the result download carries no API credential", mimeType, expectedExtension)
	}
}

// verifySourcefulVersion1960Response checks the alternate result location for a completed Sourceful
// job.
func verifySourcefulVersion1960Response(t *testing.T) {
	t.Helper()

	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	pollRequest := &jobPoll{
		adapterAPI:        providerConfig.Config.AdapterAPI,
		jobID:             "sourceful-version-1.96.0-job",
		providerModelName: ProviderID,
	}
	resultAddress := "https://media.example/version-1.96.0.webp"
	responseBody := sourcefulJSONDocument(t, map[string]any{
		"data": map[string]any{
			"job": map[string]any{"status": pollRequest.adapterAPI.ReadyStatusText},
			"result": map[string]any{
				"output": map[string]any{"url": resultAddress, "mimeType": "image/webp"},
			},
		},
	})

	jobDone, classificationErr := pollRequest.classifyJobResponse(responseBody)

	classifiedAddress := pollRequest.resultURL
	if classificationErr != nil || !jobDone || classifiedAddress != resultAddress || pollRequest.resultMIME != "image/webp" {
		t.Errorf("✗ version 1.96.0 completed response = %q, %v, %q, %v; want %q, true, %q, nil", classifiedAddress, jobDone, pollRequest.resultMIME, classificationErr, resultAddress, "image/webp")
	}

	if !t.Failed() {
		t.Log("✓ the Sourceful response path published in version 1.96.0 remains readable")
	}
}

// verifySourcefulMissingAdapterAPI checks constructor classification for absent adapter settings.
func verifySourcefulMissingAdapterAPI(t *testing.T) {
	t.Helper()

	providerConfig := catalog.Provider{
		ID:     ProviderID,
		Config: &catalog.ProviderConfig{},
	}

	_, generationErr := NewProvider(&providerConfig)
	if !errors.Is(generationErr, errs.ErrProvConfigNoAdapterAPI) {
		t.Errorf("✗ missing-adapter error = %v, want provider-config classification", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ a missing Sourceful adapter API returns the provider-config classification")
	}
}

// verifySourcefulCreationTransportFailure checks a request to a closed creation endpoint.
func verifySourcefulCreationTransportFailure(t *testing.T) {
	t.Helper()

	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	closedServer := httptest.NewServer(http.NotFoundHandler())
	closedServerURL := closedServer.URL
	closedServer.Close()

	providerConfig.Config.AdapterAPI.APIBase = closedServerURL
	providerConfig.Config.AdapterAPI.PollInterval = 0
	providerConfig.Config.AdapterAPI.PollTimeout = 1
	t.Setenv(providerConfig.APIKeyEnvVar, sourcefulTestHeaderValue)
	generationRequest := sourcefulTestRun(t, providerConfig, configuredModel, "two words", params.Values{}, nil)

	_, generationErr := constructedTestProvider(t, &providerConfig).Generate(context.Background(), &generationRequest)
	if !errors.Is(generationErr, errs.ErrTransportRequest) {
		t.Errorf("✗ closed creation server error = %v, want transport-request classification", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ a closed creation server returns the transport-request classification")
	}
}

// verifySourcefulPollErrorStatus checks status and server-message classification for a failed poll.
func verifySourcefulPollErrorStatus(t *testing.T) {
	t.Helper()

	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	fixture := newSourcefulHTTPFixture(t, &providerConfig)
	fixture.pollingAnswers = []sourcefulTestAnswer{{
		statusCode: http.StatusInternalServerError,
		answerBytes: sourcefulJSONDocument(t, map[string]any{
			"error": map[string]any{"message": "Sourceful poll failure test"},
		}),
	}}

	_, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
	if !errors.Is(generationErr, errs.ErrResponseStatus) || !errors.Is(generationErr, errs.ErrResponseServer) {
		t.Errorf("✗ poll error status = %v, want response-status and server-message classifications", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ a Sourceful poll error status returns the response classifications")
	}
}

// verifySourcefulMissingJobStatus checks missing-data classification for a response without job
// status.
func verifySourcefulMissingJobStatus(t *testing.T) {
	t.Helper()

	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	configuredModel, modelLoaded := firstSourcefulTestModel(t, &providerConfig)
	if !modelLoaded {
		return
	}

	fixture := newSourcefulHTTPFixture(t, &providerConfig)
	fixture.pollingAnswers = []sourcefulTestAnswer{{
		statusCode:  http.StatusOK,
		answerBytes: sourcefulJSONDocument(t, map[string]any{"data": map[string]any{"job": map[string]any{}}}),
	}}

	_, generationErr := generateSourcefulTestImage(t, &providerConfig, configuredModel, params.Values{}, nil)
	if !errors.Is(generationErr, errs.ErrResponseNoData) {
		t.Errorf("✗ missing-status error = %v, want no-data classification", generationErr)
	}

	if !t.Failed() {
		t.Log("✓ a Sourceful poll response without job status returns a no-data error")
	}
}

// verifySourcefulPollTransportFailure checks transport classification for a closed polling
// endpoint.
func verifySourcefulPollTransportFailure(t *testing.T) {
	t.Helper()

	providerConfig, configLoaded := loadSourcefulTestProvider(t)
	if !configLoaded {
		return
	}

	closedServer := httptest.NewServer(http.NotFoundHandler())
	closedServerURL := closedServer.URL
	closedServer.Close()

	providerConfig.Config.AdapterAPI.APIBase = closedServerURL
	pollRequest := &jobPoll{
		adapterAPI:        providerConfig.Config.AdapterAPI,
		apiCredential:     httpapi.HeaderCred("X-API-KEY", sourcefulTestHeaderValue),
		jobID:             "sourceful-closed-poll-test",
		providerModelName: ProviderID,
	}

	_, pollErr := pollRequest.Poll(context.Background())
	if !errors.Is(pollErr, errs.ErrTransportRequest) {
		t.Errorf("✗ closed poll server error = %v, want transport-request classification", pollErr)
	}

	if !t.Failed() {
		t.Log("✓ a closed poll server returns the transport-request classification")
	}
}

// loadSourcefulTestProvider decodes the embedded configuration through the built-in flag records.
func loadSourcefulTestProvider(t testing.TB) (catalog.Provider, bool) {
	t.Helper()

	parameterFlags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(parameterFlags, catalog.Source{ProviderID: ProviderID, ConfigBytes: ConfigJSON})
	if err != nil {
		t.Errorf("✗ the embedded Sourceful config failed to decode: %v", err)

		return catalog.Provider{}, false
	}

	providerConfig, loaded := loadedCatalog.Provider(ProviderID)
	if !loaded {
		t.Errorf("✗ embedded Sourceful provider failed to load: %v", loadedCatalog.ConfigError(ProviderID))

		return catalog.Provider{}, false
	}

	return providerConfig, true
}

// firstSourcefulTestModel returns the first configured model and reports whether one exists.
func firstSourcefulTestModel(t *testing.T, providerConfig *catalog.Provider) (catalog.Model, bool) {
	t.Helper()

	if len(providerConfig.Models) == 0 {
		t.Errorf("✗ the embedded Sourceful config declares no models")

		return catalog.Model{}, false
	}

	return providerConfig.Models[0], true
}

// countSourcefulChanges returns the number of records matching a parameter and change type.
func countSourcefulChanges(test testing.TB, changeRecords []params.Adjustment, parameterName params.FlagType, changeType params.Change) int {
	test.Helper()

	matchingRecords := 0

	for _, changeRecord := range changeRecords {
		if changeRecord.FlagID == parameterName && changeRecord.Type == changeType {
			matchingRecords++
		}
	}

	return matchingRecords
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
