package bfl

// Invariants tested:
//  1. Observation recovery: After a Pending response followed by HTTP 429, 502, 503, or 504,
//     httpapi.Poll with the BFL pollProbe must issue a third GET and accept the Ready response
//     without an error.
//  2. BFL poll status classification under arbitrary input: For arbitrary JSON that decodes into
//     pollResp, classifyPollStatus must return either completion with a nonempty sample and no
//     error, an error without completion, or continued polling for a configured pending status. It
//     must not panic.
//  3. BFL result extraction under arbitrary input: For arbitrary result JSON that decodes into a
//     Ready response, classifyPollStatus must either complete with a nonempty sample and no error
//     or return an error without completion. It must not panic or continue polling a Ready
//     response.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/httpapi"
)

// TestBFLPollRecovery verifies invariant #1: Observation recovery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// After a Pending response followed by HTTP 429, 502, 503, or 504, httpapi.Poll with the BFL
// pollProbe must issue a third GET and accept the Ready response without an error.
func TestBFLPollRecovery(t *testing.T) {
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
					_, _ = writer.Write([]byte(`{"status":"Pending"}`))
				case 2:
					writer.WriteHeader(statusCode)
					_, _ = writer.Write([]byte(`{"message":"temporary observation failure"}`))
				default:
					_, _ = writer.Write([]byte(`{"status":"Ready","result":{"sample":"https://result.example/image.jpg"}}`))
				}
			}))
			defer server.Close()

			poller := &pollProbe{pollURL: server.URL, id: "bfl-job", adapterAPI: &catalog.AdapterAPI{PendingStatusText: []string{"Pending"}, ReadyStatusText: "Ready"}}

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

// FuzzBFLStatus verifies invariant #2: BFL poll status classification under arbitrary input.
//
// What is being tested:
// For arbitrary JSON that decodes into pollResp, classifyPollStatus must return either completion
// with a nonempty sample and no error, an error without completion, or continued polling for a
// configured pending status. It must not panic.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzBFLStatus(f *testing.F) {
	f.Add([]byte(`{"id":"j","status":"Pending"}`))
	f.Add([]byte(`{"id":"j","status":"Ready","result":{"sample":"http://x/y.jpg"}}`))
	f.Add([]byte(`{"status":"Request Moderated","details":{"Moderation Reasons":["Violence"]}}`))
	f.Add([]byte(`{"status":"Error","details":"boom"}`))
	f.Add([]byte(`{"status":"Weird"}`))
	f.Add([]byte(`{}`))

	adapterAPI := harnessAPI(f, "")

	f.Fuzz(func(t *testing.T, data []byte) {
		var p pollResp
		if json.Unmarshal(data, &p) != nil {
			return
		}

		done, err := classifyPollStatus(&adapterAPI, "job", p)
		if done {
			if p.Result == nil || p.Result.Sample == "" {
				t.Errorf("✗ done with an empty sample for status %q", p.Status)
			}

			if err != nil {
				t.Errorf("✗ done and errored at once for status %q: %v", p.Status, err)
			}
		}

		if !done && err == nil && !slices.Contains(harnessAPI(t, "").PendingStatusText, p.Status) {
			t.Errorf("✗ status %q yielded neither completion nor error nor keep-polling", p.Status)
		}

		if !t.Failed() {
			t.Logf("✓ the status classified exclusively")
		}
	})
}

// FuzzBFLResult verifies invariant #3: BFL result extraction under arbitrary input.
//
// What is being tested:
// For arbitrary result JSON that decodes into a Ready response, classifyPollStatus must either
// complete with a nonempty sample and no error or return an error without completion. It must not
// panic or continue polling a Ready response.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzBFLResult(f *testing.F) {
	f.Add(`{"sample":"http://x/y.jpg"}`)
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(`{"sample":""}`)

	adapterAPI := harnessAPI(f, "")

	f.Fuzz(func(t *testing.T, resultJSON string) {
		body := []byte(`{"id":"job","status":"Ready","result":` + resultJSON + `}`)

		var p pollResp
		if json.Unmarshal(body, &p) != nil {
			return
		}

		if p.Status != adapterAPI.ReadyStatusText {
			// A duplicate-key injection legally rewrote the status through
			// last-key-wins decoding; the body no longer exercises the Ready completion
			// behavior checked by this fuzz test.
			return
		}

		done, err := classifyPollStatus(&adapterAPI, "job", p)
		if done && (p.Result == nil || p.Result.Sample == "" || err != nil) {
			t.Errorf("✗ Ready-done broke its invariant: result=%+v err=%v", p.Result, err)
		}

		if !done && err == nil {
			t.Errorf("✗ a Ready body neither completed nor errored (result=%q)", resultJSON)
		}

		if !t.Failed() {
			t.Logf("✓ the Ready classification held")
		}
	})
}
