package sourceful

// Invariants tested:
//  1. Declared parameter paths: Given nested parameter declarations, buildRequestBody must place
//     aspect, resolution, format, and background mode at their declared paths without displacing
//     sibling values. It must preserve model and instruction, leave the unrequested thinking level
//     unset, and produce distinct nonempty idempotency keys on two calls.

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/params"
)

// TestDeclaredParameterPaths verifies invariant #1: Declared parameter paths.
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
// What is being tested:
// Given nested parameter declarations, buildRequestBody must place aspect, resolution, format, and
// background mode at their declared paths without displacing sibling values. It must preserve model
// and instruction, leave the unrequested thinking level unset, and produce distinct nonempty
// idempotency keys on two calls.
func TestDeclaredParameterPaths(t *testing.T) {
	model := catalog.Model{ID: "declared-model", Params: params.Definitions{
		{FlagID: params.FlagTypeAspect, ParamID: "output.aspectRatio"},
		{FlagID: params.FlagTypeResolution, ParamID: "output.resolution"},
		{FlagID: params.FlagTypeOutputFormat, ParamID: "output.format"},
		{FlagID: params.FlagTypeQuality, ParamID: "output.background.mode"},
		{FlagID: params.FlagTypeThinkingLevel, ParamID: "thinkingLevel"},
	}}
	request := &generation.Generation{ProvModelPair: catalog.ProvModelPair{Model: model}, Prompt: "preserved prompt", Preparation: generation.Preparation{Params: params.Values{
		params.FlagTypeAspect: "3:2", params.FlagTypeResolution: "2K",
		params.FlagTypeOutputFormat: "png", params.FlagTypeQuality: "transparent",
	}}}
	body := buildRequestBody(request)
	outputFields, outputPresent := body["output"].(map[string]any)

	backgroundFields, backgroundPresent := outputFields["background"].(map[string]any)
	if !outputPresent || !backgroundPresent || backgroundFields["mode"] != "transparent" || outputFields["aspectRatio"] != "3:2" || outputFields["resolution"] != "2K" || outputFields["format"] != "png" {
		t.Errorf("✗ nested declared parameters were lost: %#v", body)
	}

	if body["model"] != model.ID || body["instruction"] != request.Prompt || body["thinkingLevel"] != nil {
		t.Errorf("✗ required or omitted fields changed: %#v", body)
	}

	secondBody := buildRequestBody(request)
	firstID, _ := body["idempotencyKey"].(string)

	secondID, _ := secondBody["idempotencyKey"].(string)
	if firstID == "" || secondID == "" || firstID == secondID {
		t.Errorf("✗ creation identifiers are absent or repeated")
	}

	if !t.Failed() {
		t.Log("✓ nested declarations compose without displacing required fields")
	}
}

// ServeHTTP records a request and returns the answer for its Sourceful route.
func (fixture *sourcefulHTTPFixture) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	fixture.test.Helper()

	requestBytes, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		http.Error(responseWriter, readErr.Error(), http.StatusInternalServerError)

		return
	}

	fixture.mutex.Lock()
	fixture.recordedRequests = append(fixture.recordedRequests, sourcefulRecordedRequest{
		method:       request.Method,
		path:         request.URL.Path,
		credential:   request.Header.Get("X-Api-Key"),
		requestBytes: requestBytes,
	})
	fixture.mutex.Unlock()

	if request.Method == http.MethodPost && (request.URL.Path == "/v2.5/generations/t2i" || request.URL.Path == "/v2.5/generations/i2i") {
		writeSourcefulTestAnswer(fixture.test, responseWriter, fixture.creationAnswer)

		return
	}

	if request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v2.5/generations/") {
		writeSourcefulTestAnswer(fixture.test, responseWriter, fixture.nextPollAnswer())

		return
	}

	if request.Method == http.MethodGet && request.URL.Path == "/result" {
		if fixture.downloadRedirectTo != "" {
			http.Redirect(responseWriter, request, fixture.downloadRedirectTo, http.StatusFound)

			return
		}

		writeSourcefulTestAnswer(fixture.test, responseWriter, fixture.downloadAnswer)

		return
	}

	http.NotFound(responseWriter, request)
}

// nextPollAnswer returns the next configured poll answer, or a completed answer thereafter.
func (fixture *sourcefulHTTPFixture) nextPollAnswer() sourcefulTestAnswer {
	fixture.test.Helper()

	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()

	if fixture.nextPollingAnswer < len(fixture.pollingAnswers) {
		pollAnswer := fixture.pollingAnswers[fixture.nextPollingAnswer]
		fixture.nextPollingAnswer++

		return pollAnswer
	}

	return sourcefulTestAnswer{
		statusCode: http.StatusOK,
		answerBytes: sourcefulCompletedPollDocument(fixture.test,
			fixture.server.URL+"/result",
			"application/octet-stream",
			fixture.completedStatus,
		),
	}
}

// writeSourcefulTestAnswer renders one fixture answer.
func writeSourcefulTestAnswer(test testing.TB, responseWriter http.ResponseWriter, answer sourcefulTestAnswer) {
	test.Helper()

	if answer.contentType != "" {
		responseWriter.Header().Set("Content-Type", answer.contentType)
	}

	if answer.location != "" {
		responseWriter.Header().Set("Location", answer.location)
	}

	if answer.byteCount > 0 {
		responseWriter.Header().Set("Content-Length", strconv.FormatInt(answer.byteCount, 10))
	}

	statusCode := answer.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	responseWriter.WriteHeader(statusCode)
	_, _ = responseWriter.Write(answer.answerBytes)
}
