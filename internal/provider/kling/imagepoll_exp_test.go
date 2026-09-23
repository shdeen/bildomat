package kling

// Invariants tested:
//  1. Kling image classification branches: imageJobPoll.classifyResponse must return ErrResponseGen
//     for a nonzero code, ErrResponseUnknown for absent or unknown status, and ErrResponseNoData
//     for missing results, empty images, or a missing URL. Malformed image records, fractional
//     indexes, and duplicate indexes must return ErrResponseDecode. None of these responses may
//     report completion.
//  2. Kling image poll transport branches: Given an invalid API URL, imageJobPoll.Poll must return
//     ErrTransportCreate without ErrTransportRequest. Given HTTP 429, it must return
//     ErrResponseStatus.
//  3. Permanent response limits: Given a 65 MiB response, both imageJobPoll.Poll and
//     videoJobPoll.Poll must return ErrTransportSize without ErrTransportRequest or
//     ErrTransportRead.
//  4. Observation recovery: After a processing response followed by HTTP 429, 502, 503, or 504,
//     httpapi.Poll with imageJobPoll must issue a third GET and accept the succeed response without
//     an error.
//  5. Kling image poll classification: For arbitrary response bytes, imageJobPoll.classifyResponse
//     must not panic or report completion with an error. Completion must retain at least one result
//     URL. Any error must match ErrResponseDecode, ErrResponseNoData, ErrResponseGen, or
//     ErrResponseUnknown.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
)

// TestKlingImageClassificationBranches verifies invariant #1: Kling image classification branches.
//
// What is being tested:
// imageJobPoll.classifyResponse must return ErrResponseGen for a nonzero code, ErrResponseUnknown
// for absent or unknown status, and ErrResponseNoData for missing results, empty images, or a
// missing URL. Malformed image records, fractional indexes, and duplicate indexes must return
// ErrResponseDecode. None of these responses may report completion.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingImageClassificationBranches(t *testing.T) {
	classificationCases := []struct {
		caseName      string
		responseBody  []byte
		expectedError error
	}{
		{
			caseName:      "nonzero envelope",
			responseBody:  klingFuzzJSON(t, map[string]any{"code": 1200, "message": "image envelope failure", "data": nil}),
			expectedError: errs.ErrResponseGen,
		},
		{
			caseName:      "missing status",
			responseBody:  klingFuzzJSON(t, map[string]any{"code": 0, "data": map[string]any{}}),
			expectedError: errs.ErrResponseUnknown,
		},
		{
			caseName: "missing task result",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": map[string]any{
				"task_status": imageStatusSucceeded,
			}}),
			expectedError: errs.ErrResponseNoData,
		},
		{
			caseName: "empty image array",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": map[string]any{
				"task_status": imageStatusSucceeded,
				"task_result": map[string]any{"images": []any{}},
			}}),
			expectedError: errs.ErrResponseNoData,
		},
		{
			caseName: "invalid image record",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": map[string]any{
				"task_status": imageStatusSucceeded,
				"task_result": map[string]any{"images": []any{"invalid"}},
			}}),
			expectedError: errs.ErrResponseDecode,
		},
		{
			caseName: "invalid image index",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": map[string]any{
				"task_status": imageStatusSucceeded,
				"task_result": map[string]any{"images": []any{map[string]any{"index": 0.5, "url": "https://media.example/image.png"}}},
			}}),
			expectedError: errs.ErrResponseDecode,
		},
		{
			caseName: "missing image URL",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": map[string]any{
				"task_status": imageStatusSucceeded,
				"task_result": map[string]any{"images": []any{map[string]any{"index": 0}}},
			}}),
			expectedError: errs.ErrResponseNoData,
		},
		{
			caseName: "duplicate image index",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": map[string]any{
				"task_status": imageStatusSucceeded,
				"task_result": map[string]any{"images": []any{
					map[string]any{"index": 0, "url": "https://media.example/first.png"},
					map[string]any{"index": 0, "url": "https://media.example/second.png"},
				}},
			}}),
			expectedError: errs.ErrResponseDecode,
		},
		{
			caseName:      "unknown status",
			responseBody:  klingFuzzJSON(t, map[string]any{"code": 0, "data": map[string]any{"task_status": "unexpected"}}),
			expectedError: errs.ErrResponseUnknown,
		},
	}

	for _, classificationCase := range classificationCases {
		verifyKlingClassificationCase(t, media.Image, classificationCase.caseName, classificationCase.responseBody, classificationCase.expectedError)
	}

	if !t.Failed() {
		t.Log("✓ malformed, missing, failed, and unknown Kling image responses return classified failures")
	}
}

// TestKlingImagePollTransportBranches verifies invariant #2: Kling image poll transport branches.
//
// What is being tested:
// Given an invalid API URL, imageJobPoll.Poll must return ErrTransportCreate without
// ErrTransportRequest. Given HTTP 429, it must return ErrResponseStatus.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingImagePollTransportBranches(t *testing.T) {
	verifyKlingPollBranches(t, media.Image)

	if !t.Failed() {
		t.Log("✓ image polling classifies transport and HTTP status failures")
	}
}

// TestPollResponseLimit verifies invariant #3: Permanent response limits.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given a 65 MiB response, both imageJobPoll.Poll and videoJobPoll.Poll must return
// ErrTransportSize without ErrTransportRequest or ErrTransportRead.
func TestPollResponseLimit(t *testing.T) {
	responseBytes := bytes.Repeat([]byte("x"), 65<<20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(responseBytes)
	}))
	t.Cleanup(server.Close)
	imagePoll := &imageJobPoll{apiBase: server.URL, queryPath: "/image-task", taskID: "image-task", providerModelName: "kling/image"}

	videoPoll := &videoJobPoll{apiBase: server.URL, taskID: "video-task", providerModelName: "kling/video"}
	for _, poller := range []klingTestPoller{imagePoll, videoPoll} {
		_, err := poller.Poll(t.Context())
		if !errors.Is(err, errs.ErrTransportSize) || errors.Is(err, errs.ErrTransportRequest) || errors.Is(err, errs.ErrTransportRead) {
			t.Errorf("✗ oversized response error=%v; want only permanent size classification", err)
		}
	}

	if !t.Failed() {
		t.Log("✓ neither Kling poller retries oversized responses")
	}
}

// TestKlingImagePollRecovery verifies invariant #4: Observation recovery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// After a processing response followed by HTTP 429, 502, 503, or 504, httpapi.Poll with
// imageJobPoll must issue a third GET and accept the succeed response without an error.
func TestKlingImagePollRecovery(t *testing.T) {
	for _, statusCode := range []int{429, 502, 503, 504} {
		t.Run(strconv.Itoa(statusCode), func(t *testing.T) {
			observations := 0

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet {
					t.Errorf("✗ observation used %s instead of GET", request.Method)
				}

				observations++
				switch observations {
				case 1:
					_, _ = writer.Write([]byte(`{"code":0,"data":{"task_status":"processing"}}`))
				case 2:
					writer.WriteHeader(statusCode)
					_, _ = writer.Write([]byte(`{"message":"temporary observation failure"}`))
				default:
					_, _ = writer.Write([]byte(`{"code":0,"data":{"task_status":"succeed","task_result":{"images":[{"index":0,"url":"https://result.example/image.jpg"}]}}}`))
				}
			}))
			defer server.Close()

			poller := &imageJobPoll{apiBase: server.URL, queryPath: "/image-job", taskID: "image-job", providerModelName: "kling/image"}

			err := httpapi.Poll(t.Context(), time.Millisecond, time.Second, poller)
			if err != nil || observations != 3 {
				t.Errorf("✗ completion error=%v observations=%d; want successful third observation", err, observations)
			}

			if !t.Failed() {
				t.Log("✓ temporary observation failure recovers without resubmission")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every selected HTTP status recovers to completion")
	}
}

// FuzzKlingImagePollClassification verifies invariant #5: Kling image poll classification.
//
// What is being tested:
// For arbitrary response bytes, imageJobPoll.classifyResponse must not panic or report completion
// with an error. Completion must retain at least one result URL. Any error must match
// ErrResponseDecode, ErrResponseNoData, ErrResponseGen, or ErrResponseUnknown.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzKlingImagePollClassification(f *testing.F) {
	f.Add([]byte(`{"code":9007199254740992.5}`))
	f.Add([]byte(`{"code":null}`))
	f.Add([]byte(`{"code":0.0,"data":{"task_status":"succeed","task_result":{"images":[{"index":0,"url":"https://media.example/a.png"},{"index":0,"url":"https://media.example/b.png"}]}}}`))
	f.Add(klingFuzzJSON(f, map[string]any{"code": 0, "data": map[string]any{"task_status": taskStatusSubmitted}}))
	f.Add(klingFuzzJSON(f, map[string]any{"code": 0, "data": map[string]any{
		"task_status": imageStatusSucceeded,
		"task_result": map[string]any{"images": []any{map[string]any{"index": 0, "url": "https://media.example/image.png"}}},
	}}))
	f.Add(klingFuzzJSON(f, map[string]any{"code": 0, "data": map[string]any{"task_status": taskStatusFailed, "task_status_msg": "fuzz failure"}}))
	f.Add([]byte("not JSON"))
	f.Add([]byte(`{"code":0,"data":null}`))

	f.Fuzz(func(t *testing.T, responseBody []byte) {
		imagePoll := imageJobPoll{taskID: "image-fuzz-task", providerModelName: ProviderID}

		var resultText string

		taskComplete, classificationErr := imagePoll.classifyResponse(responseBody)
		verifyKlingPollResult(t, "image", resultText, taskComplete, classificationErr)

		if taskComplete && len(imagePoll.resultURLs) == 0 {
			t.Errorf("✗ completed image classification retained no result URL")
		}

		if !t.Failed() {
			t.Log("✓ the image poll body returns a classified outcome")
		}
	})
}

// klingTestPoller is the polling surface shared by the image and video tests.
type klingTestPoller interface {
	Poll(ctx context.Context) (bool, error)
}

// verifyKlingPollTransportFailure checks that an invalid polling URL returns ErrTransportCreate
// without ErrTransportRequest.
func verifyKlingPollTransportFailure(test *testing.T, poller klingTestPoller) {
	test.Helper()

	_, pollErr := poller.Poll(test.Context())
	if errors.Is(pollErr, errs.ErrTransportRequest) || !errors.Is(pollErr, errs.ErrTransportCreate) {
		test.Errorf("✗ poll transport error = %v, want request-creation classification without a retryable request classification", pollErr)
	}
}

// newKlingRateLimitServer returns a local server carrying one top-level error message and registers
// its cleanup with the current test.
func newKlingRateLimitServer(test *testing.T, failureMessage string) *httptest.Server {
	test.Helper()

	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, _ *http.Request) {
		responseWriter.WriteHeader(http.StatusTooManyRequests)
		_, _ = responseWriter.Write(klingFuzzJSON(test, map[string]any{"message": failureMessage}))
	}))
	test.Cleanup(testServer.Close)

	return testServer
}

// verifyKlingPollStatusFailure verifies the shared non-success HTTP classification for one image or
// video poller.
func verifyKlingPollStatusFailure(test *testing.T, poller klingTestPoller) {
	test.Helper()

	_, pollErr := poller.Poll(test.Context())
	if !errors.Is(pollErr, errs.ErrResponseStatus) {
		test.Errorf("✗ poll status error = %v, want response-status classification", pollErr)
	}
}

// verifyKlingClassificationCase verifies one direct image or video poll response without reaching a
// network host.
func verifyKlingClassificationCase(test *testing.T, medium media.Kind, caseName string, responseBody []byte, expectedError error) {
	test.Helper()

	test.Run(caseName, func(test *testing.T) {
		var (
			resultText        string
			taskComplete      bool
			classificationErr error
		)

		if medium == media.Image {
			imagePoll := imageJobPoll{taskID: "image-coverage-task", providerModelName: ProviderID}
			taskComplete, classificationErr = imagePoll.classifyResponse(responseBody)
		} else {
			videoPoll := videoJobPoll{taskID: "video-coverage-task", providerModelName: ProviderID}
			taskComplete, classificationErr = videoPoll.classifyResponse(responseBody)
		}

		verifyKlingPollClassification(test, string(medium), resultText, taskComplete, classificationErr, expectedError)

		if !test.Failed() {
			test.Log("✓ the poll response returns its required classification")
		}
	})
}

// verifyKlingPollBranches verifies transport and non-success HTTP handling for one polling family.
func verifyKlingPollBranches(test *testing.T, medium media.Kind) {
	test.Helper()

	if medium == media.Image {
		imagePoll := imageJobPoll{apiBase: "://invalid", queryPath: "/image", providerModelName: ProviderID}
		verifyKlingPollTransportFailure(test, &imagePoll)

		testServer := newKlingRateLimitServer(test, "image rate limit")
		imagePoll = imageJobPoll{apiBase: testServer.URL, queryPath: "/image", providerModelName: ProviderID}
		verifyKlingPollStatusFailure(test, &imagePoll)

		return
	}

	videoPoll := videoJobPoll{apiBase: "://invalid", taskID: "video-task", providerModelName: ProviderID}
	verifyKlingPollTransportFailure(test, &videoPoll)

	testServer := newKlingRateLimitServer(test, "video rate limit")
	videoPoll = videoJobPoll{apiBase: testServer.URL, taskID: "video-task", providerModelName: ProviderID}
	verifyKlingPollStatusFailure(test, &videoPoll)
}

// verifyKlingPollClassification verifies one direct poll response's expected terminal fields and
// error classification.
func verifyKlingPollClassification(test *testing.T, familyName string, resultText string, taskComplete bool, classificationErr, expectedError error) {
	test.Helper()

	if !errors.Is(classificationErr, expectedError) || resultText != "" || taskComplete {
		test.Errorf("✗ %s classification = %q, %v, %v; want empty, false, %v", familyName, resultText, taskComplete, classificationErr, expectedError)
	}
}

// verifyKlingPollResult checks that an error prevents completion and belongs to an accepted
// response category. It also requires empty result text.
func verifyKlingPollResult(test *testing.T, familyName string, resultText string, taskComplete bool, classificationErr error) {
	test.Helper()

	if resultText != "" {
		test.Errorf("✗ %s classifier returned unexpected result text %q", familyName, resultText)
	}

	if classificationErr != nil && taskComplete {
		test.Errorf("✗ %s classifier returned complete with error %v", familyName, classificationErr)
	}

	if classificationErr != nil &&
		!errors.Is(classificationErr, errs.ErrResponseDecode) &&
		!errors.Is(classificationErr, errs.ErrResponseNoData) &&
		!errors.Is(classificationErr, errs.ErrResponseGen) &&
		!errors.Is(classificationErr, errs.ErrResponseUnknown) {
		test.Errorf("✗ %s classifier returned unclassified error %v", familyName, classificationErr)
	}
}

// klingFuzzJSON returns a JSON seed document for a Kling fuzz target.
func klingFuzzJSON(test testing.TB, responseDocument any) []byte {
	test.Helper()

	responseBody, err := json.Marshal(responseDocument)
	if err != nil {
		panic(err)
	}

	return responseBody
}
