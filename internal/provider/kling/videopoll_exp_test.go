package kling

// Invariants tested:
//  1. Kling video classification branches: videoJobPoll.classifyResponse must return ErrResponseGen
//     for a nonzero code, ErrResponseDecode for malformed task or output records,
//     ErrResponseUnknown for absent or unknown status, and ErrResponseNoData for missing outputs or
//     a missing video URL. None of these responses may report completion.
//  2. Kling video poll transport branches: Given an invalid API URL, videoJobPoll.Poll must return
//     ErrTransportCreate without ErrTransportRequest. Given HTTP 429, it must return
//     ErrResponseStatus.
//  3. Observation recovery: After a processing response followed by HTTP 429, 502, 503, or 504,
//     httpapi.Poll with videoJobPoll must issue a third GET and accept the succeeded response
//     without an error.
//  4. Kling video poll classification: For arbitrary response bytes, videoJobPoll.classifyResponse
//     must not panic or report completion with an error. Completion must retain a nonempty result
//     URL. Any error must match ErrResponseDecode, ErrResponseNoData, ErrResponseGen, or
//     ErrResponseUnknown.

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
)

// TestKlingVideoClassificationBranches verifies invariant #1: Kling video classification branches.
//
// What is being tested:
// videoJobPoll.classifyResponse must return ErrResponseGen for a nonzero code, ErrResponseDecode
// for malformed task or output records, ErrResponseUnknown for absent or unknown status, and
// ErrResponseNoData for missing outputs or a missing video URL. None of these responses may report
// completion.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingVideoClassificationBranches(t *testing.T) {
	classificationCases := []struct {
		caseName      string
		responseBody  []byte
		expectedError error
	}{
		{
			caseName:      "nonzero envelope",
			responseBody:  klingFuzzJSON(t, map[string]any{"code": 1200, "message": "video envelope failure", "data": nil}),
			expectedError: errs.ErrResponseGen,
		},
		{
			caseName:      "invalid task record",
			responseBody:  klingFuzzJSON(t, map[string]any{"code": 0, "data": []any{"invalid"}}),
			expectedError: errs.ErrResponseDecode,
		},
		{
			caseName:      "missing status",
			responseBody:  klingFuzzJSON(t, map[string]any{"code": 0, "data": []any{map[string]any{}}}),
			expectedError: errs.ErrResponseUnknown,
		},
		{
			caseName: "missing outputs",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": []any{map[string]any{
				"status": videoStatusSucceeded,
			}}}),
			expectedError: errs.ErrResponseNoData,
		},
		{
			caseName: "invalid output record",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": []any{map[string]any{
				"status":  videoStatusSucceeded,
				"outputs": []any{"invalid"},
			}}}),
			expectedError: errs.ErrResponseDecode,
		},
		{
			caseName: "missing video URL",
			responseBody: klingFuzzJSON(t, map[string]any{"code": 0, "data": []any{map[string]any{
				"status":  videoStatusSucceeded,
				"outputs": []any{map[string]any{}},
			}}}),
			expectedError: errs.ErrResponseNoData,
		},
		{
			caseName:      "unknown status",
			responseBody:  klingFuzzJSON(t, map[string]any{"code": 0, "data": []any{map[string]any{"status": "unexpected"}}}),
			expectedError: errs.ErrResponseUnknown,
		},
	}

	for _, classificationCase := range classificationCases {
		verifyKlingClassificationCase(t, media.Video, classificationCase.caseName, classificationCase.responseBody, classificationCase.expectedError)
	}

	if !t.Failed() {
		t.Log("✓ malformed, missing, failed, and unknown Kling video responses return classified failures")
	}
}

// TestKlingVideoPollTransportBranches verifies invariant #2: Kling video poll transport branches.
//
// What is being tested:
// Given an invalid API URL, videoJobPoll.Poll must return ErrTransportCreate without
// ErrTransportRequest. Given HTTP 429, it must return ErrResponseStatus.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingVideoPollTransportBranches(t *testing.T) {
	verifyKlingPollBranches(t, media.Video)

	if !t.Failed() {
		t.Log("✓ video polling classifies transport and HTTP status failures")
	}
}

// TestKlingVideoPollRecovery verifies invariant #3: Observation recovery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// After a processing response followed by HTTP 429, 502, 503, or 504, httpapi.Poll with
// videoJobPoll must issue a third GET and accept the succeeded response without an error.
func TestKlingVideoPollRecovery(t *testing.T) {
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
					_, _ = writer.Write([]byte(`{"code":0,"data":[{"status":"processing"}]}`))
				case 2:
					writer.WriteHeader(statusCode)
					_, _ = writer.Write([]byte(`{"message":"temporary observation failure"}`))
				default:
					_, _ = writer.Write([]byte(`{"code":0,"data":[{"status":"succeeded","outputs":[{"url":"https://result.example/video.mp4"}]}]}`))
				}
			}))
			defer server.Close()

			poller := &videoJobPoll{apiBase: server.URL + "/", taskID: "video-job", providerModelName: "kling/video"}

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

// FuzzKlingVideoPollClassification verifies invariant #4: Kling video poll classification.
//
// What is being tested:
// For arbitrary response bytes, videoJobPoll.classifyResponse must not panic or report completion
// with an error. Completion must retain a nonempty result URL. Any error must match
// ErrResponseDecode, ErrResponseNoData, ErrResponseGen, or ErrResponseUnknown.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzKlingVideoPollClassification(f *testing.F) {
	f.Add(klingFuzzJSON(f, map[string]any{"code": 0, "data": []any{map[string]any{"status": taskStatusSubmitted}}}))
	f.Add(klingFuzzJSON(f, map[string]any{"code": 0, "data": []any{map[string]any{
		"status":  videoStatusSucceeded,
		"outputs": []any{map[string]any{"url": "https://media.example/video.mp4"}},
	}}}))
	f.Add(klingFuzzJSON(f, map[string]any{"code": 0, "data": []any{map[string]any{"status": taskStatusFailed, "message": "fuzz failure"}}}))
	f.Add([]byte("not JSON"))
	f.Add([]byte(`{"code":0,"data":null}`))

	f.Fuzz(func(t *testing.T, responseBody []byte) {
		videoPoll := videoJobPoll{taskID: "video-fuzz-task", providerModelName: ProviderID}

		var resultText string

		taskComplete, classificationErr := videoPoll.classifyResponse(responseBody)
		verifyKlingPollResult(t, "video", resultText, taskComplete, classificationErr)

		if taskComplete && videoPoll.resultURL == "" {
			t.Errorf("✗ completed video classification retained no result URL")
		}

		if !t.Failed() {
			t.Log("✓ the video poll body returns a classified outcome")
		}
	})
}
