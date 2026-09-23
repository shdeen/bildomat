package provider

// Invariants tested:
//  1. Failed poll message: Given a failed job response with an error message, classifyResponse must
//     return an error containing that message and the job ID, with both state flags false.
//  2. Poll response shapes: Given string or object errors and top-level details, classifyResponse
//     must return ErrResponseGen with the diagnostic and job ID, even when the status says pending
//     or done.
//  3. Poll response classification: Given the listed HTTP statuses and response bodies,
//     PollResponseError followed by classifyResponse must return the expected pending, completed,
//     generation-error, unknown-status, decode-error, or HTTP-error result.
//  4. Polling deadline: When classifyResponse keeps reporting a pending job, httpapi.Poll must
//     return ErrTransportTimeout after at least the configured 60 ms and within one second.
//  5. Transient request failures: When two requests lose their connections before a completed
//     response arrives, httpapi.Poll using classifyResponse must retry at least twice and return
//     success with the completion marker.
//  6. Transient response-read failures: When the first response body is shorter than its declared
//     length and the next reports completion, httpapi.Poll using classifyResponse must retry and
//     return success with the completion marker.
//  7. Last transient polling error: When every request loses its connection, httpapi.Poll using
//     classifyResponse must return within one second with an error wrapping both
//     ErrTransportTimeout and ErrTransportRequest.
//  8. Deadline during a polling wait: When classifyResponse reports a pending job and the interval
//     is 30 seconds, httpapi.Poll must interrupt the wait at its 100 ms deadline, returning
//     ErrTransportTimeout within one second.
//  9. Deadline during a polling request: When a polling request stalls, httpapi.Poll using the
//     classifyResponse probe must cancel the request at its 150 ms deadline and return
//     ErrTransportTimeout within one second.
//  10. Terminal response classification: When classifyResponse receives a failed status with a
//      provider message, httpapi.Poll must return ErrResponseGen containing that message after
//      exactly one request and within five seconds.
//  11. Canceled polling: Given cancellation during a minute-long wait or a stalled request,
//      httpapi.Poll using the classifyResponse probe must return ErrCanceled within two seconds.
//  12. Poll decoder causes: Given truncated JSON or a JSON array instead of an object,
//      classifyResponse must return ErrResponseDecode with both state flags false and retain the
//      original syntax or type error for errors.As.
//  13. Temporary polling statuses: Given pending, temporary HTTP failure, and completed responses
//      in sequence, httpapi.Poll using classifyResponse must return success with the completion
//      marker after exactly three requests for each of HTTP 429, 502, 503, and 504.
//  14. Timeout observation history: Given HTTP 503 followed by pending responses, httpapi.Poll
//      using classifyResponse must make at least two requests and return an error wrapping
//      ErrTransportTimeout and ErrResponseStatus while retaining the original provider message.
//  15. Poll response classification under arbitrary input: For arbitrary HTTP statuses and bodies,
//      PollResponseError followed by classifyResponse must leave both state flags false on error or
//      set exactly one of pending and completed on success, without panicking.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/metadata"
)

// TestClassifyFailMessage verifies invariant #1: Failed poll message.
//
// What is being tested:
// Given a failed job response with an error message, classifyResponse must return an error
// containing that message and the job ID, with both state flags false.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestClassifyFailMessage(t *testing.T) {
	keepPolling, jobDone, err := classifyResponse("job-9", []byte(`{"status":"failed","error":{"message":"moderation"}}`),
		[]string{"pending"}, "done", []string{"failed"})
	if keepPolling || jobDone || err == nil {
		t.Errorf("✗ Classify(failed+msg) = (%v,%v,%v), want a terminal error", keepPolling, jobDone, err)
	}

	if err != nil && (!strings.Contains(err.Error(), "moderation") || !strings.Contains(err.Error(), "job-9")) {
		t.Errorf("✗ Classify(failed+msg) error %q does not carry the message and id", err)
	}

	if !t.Failed() {
		t.Log("✓ Classify surfaces the provider failure message and the job id on a terminal status")
	}
}

// TestClassifyShapes verifies invariant #2: Poll response shapes.
//
// What is being tested:
// Given string or object errors and top-level details, classifyResponse must return ErrResponseGen
// with the diagnostic and job ID, even when the status says pending or done. A numeric status must
// produce ErrResponseUnknown containing that number; a null error with pending status must allow
// polling to continue.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestClassifyShapes(t *testing.T) {
	const id = "job-s"

	fail := func(tag, body, wantMsg string) {
		t.Helper()

		keepPolling, jobDone, err := classifyResponse(id, []byte(body), pollRunning, "done", pollFailed)
		if keepPolling || jobDone || err == nil {
			t.Errorf("✗ %s: Classify = (keep=%v,done=%v,err=%v), want a terminal error", tag, keepPolling, jobDone, err)

			return
		}

		if !errors.Is(err, errs.ErrResponseGen) {
			t.Errorf("✗ %s: error %v is not a wrapped ErrResponseGen", tag, err)
		}

		if !strings.Contains(err.Error(), wantMsg) || !strings.Contains(err.Error(), id) {
			t.Errorf("✗ %s: error %q does not carry the diagnostic %q and id", tag, err, wantMsg)
		}
	}
	fail("top-level string error", `{"status":"failed","error":"quota gone"}`, "quota gone")
	fail("top-level details", `{"status":"failed","details":"upstream refused"}`, "upstream refused")
	fail("error over running word", `{"status":"pending","error":{"message":"midway fault"}}`, "midway fault")
	fail("error over done word", `{"status":"done","error":"late failure"}`, "late failure")

	_, _, err := classifyResponse(id, []byte(`{"status":123}`), pollRunning, "done", pollFailed)
	if err == nil || !errors.Is(err, errs.ErrResponseUnknown) || !strings.Contains(err.Error(), "123") {
		t.Errorf("✗ numeric status: error %v must wrap ErrResponseUnknown and show the observed value 123", err)
	}

	keepPolling, jobDone, err := classifyResponse(id, []byte(`{"status":"pending","error":null}`), pollRunning, "done", pollFailed)
	if !keepPolling || jobDone || err != nil {
		t.Errorf("✗ null error field: Classify = (keep=%v,done=%v,err=%v), want keep-polling — a null error is no error", keepPolling, jobDone, err)
	}

	if !t.Failed() {
		t.Log("✓ Classify carries the diagnostic from every required form, fails error-over-status, and names a numeric status verbatim")
	}
}

// TestClassifyHostile verifies invariant #3: Poll response classification.
//
// What is being tested:
// Given the listed HTTP statuses and response bodies, PollResponseError followed by
// classifyResponse must return the expected pending, completed, generation-error, unknown-status,
// decode-error, or HTTP-error result. Every error must contain the job ID and wrap the expected
// sentinel.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestClassifyHostile(t *testing.T) {
	const id = "job-x"

	type want struct {
		keepPolling, jobDone, err bool
	}

	cases := []struct {
		status   int
		body     string
		want     want
		sentinel error // the required errors.Is target for err cases
	}{
		{200, `{"status":"pending"}`, want{keepPolling: true}, nil},
		{200, `{"status":"queued"}`, want{keepPolling: true}, nil},
		{200, `{"status":"done"}`, want{jobDone: true}, nil},
		{200, `{"status":"failed"}`, want{err: true}, errs.ErrResponseGen},
		{200, `{"status":"expired"}`, want{err: true}, errs.ErrResponseGen},
		{200, `{"status":"weird"}`, want{err: true}, errs.ErrResponseUnknown},
		{200, `not json`, want{err: true}, errs.ErrResponseDecode},
		{200, `{}`, want{err: true}, errs.ErrResponseUnknown},
		{200, `{"status":123}`, want{err: true}, errs.ErrResponseUnknown},
		{200, `{"status":null}`, want{err: true}, errs.ErrResponseUnknown},
		{200, ``, want{err: true}, errs.ErrResponseDecode},
		{500, `{"error":{"message":"boom"}}`, want{err: true}, errs.ErrResponseStatus},
		{418, `teapot`, want{err: true}, errs.ErrResponseStatus},
	}
	for _, c := range cases {
		keepPolling, isDone := false, false

		err := PollResponseError(id, c.status, []byte(c.body), nil)
		if err == nil {
			keepPolling, isDone, err = classifyResponse(id, []byte(c.body), pollRunning, "done", pollFailed)
		}

		if keepPolling != c.want.keepPolling || isDone != c.want.jobDone || (err != nil) != c.want.err {
			t.Errorf("✗ Classify(%d,%q) = (keep=%v,done=%v,err=%v), want %+v", c.status, c.body, keepPolling, isDone, err, c.want)
		}

		if err != nil && !strings.Contains(err.Error(), id) {
			t.Errorf("✗ Classify(%d,%q) error %q does not name the id %q", c.status, c.body, err, id)
		}

		if c.sentinel != nil && !errors.Is(err, c.sentinel) {
			t.Errorf("✗ Classify(%d,%q) error %v is not a wrapped %v", c.status, c.body, err, c.sentinel)
		}
	}

	if !t.Failed() {
		t.Log("✓ Classify classifies hostile/malformed bodies deterministically under the right sentinels; errors name the id, never panicking")
	}
}

// TestPollDeadline verifies invariant #4: Polling deadline.
//
// What is being tested:
// When classifyResponse keeps reporting a pending job, httpapi.Poll must return ErrTransportTimeout
// after at least the configured 60 ms and within one second.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollDeadline(t *testing.T) {
	_, srv := dropSrv(t, 0, `{"status":"pending"}`)
	start := time.Now()
	err := httpapi.Poll(t.Context(), 2*time.Millisecond, 60*time.Millisecond, classProbe(t, srv.URL, "id-dl"))
	elapsed := time.Since(start)

	if err == nil || !errors.Is(err, errs.ErrTransportTimeout) {
		t.Errorf("✗ httpapi.Poll(never completes) = %v, want a wrapped ErrTransportTimeout", err)
	}

	if elapsed < 60*time.Millisecond {
		t.Errorf("✗ Poll returned after %v, before the 60ms deadline", elapsed)
	}

	if elapsed > time.Second {
		t.Errorf("✗ Poll took %v, want a return proportionate to the 60ms deadline", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ a never-completing job deadlines at the configured timeout with the transport-timeout sentinel")
	}
}

// TestPollTransient verifies invariant #5: Transient request failures.
//
// What is being tested:
// When two requests lose their connections before a completed response arrives, httpapi.Poll using
// classifyResponse must retry at least twice and return success with the completion marker.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollTransient(t *testing.T) {
	polls, srv := dropSrv(t, 2, `{"status":"done"}`)

	poller := classProbe(t, srv.URL, "id-tr")
	err := httpapi.Poll(t.Context(), time.Millisecond, 5*time.Second, poller)

	res := poller.completed
	if err != nil || res != "RESULT" {
		t.Errorf("✗ httpapi.Poll(two drops then done) = (%q,%v), want RESULT with no error", res, err)
	}

	if got := polls.Load(); got < 3 {
		t.Errorf("✗ harness saw %d polls, want at least 3 (two tolerated drops and the completion)", got)
	}

	if !t.Failed() {
		t.Log("✓ transient transport drops are tolerated and polling continues to the result")
	}
}

// TestPollTransientRead verifies invariant #6: Transient response-read failures.
//
// What is being tested:
// When the first response body is shorter than its declared length and the next reports completion,
// httpapi.Poll using classifyResponse must retry and return success with the completion marker.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollTransientRead(t *testing.T) {
	var polls atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if polls.Add(1) == 1 {
			// A body shorter than its declared length: the client's read fails
			// mid-body, after a successful request execution.
			w.Header().Set("Content-Length", "64")
			_, _ = w.Write([]byte("short"))

			return
		}

		fmt.Fprint(w, `{"status":"done"}`)
	}))
	t.Cleanup(srv.Close)

	poller := classProbe(t, srv.URL, "id-rr")
	err := httpapi.Poll(t.Context(), time.Millisecond, 5*time.Second, poller)

	res := poller.completed
	if err != nil || res != "RESULT" {
		t.Errorf("✗ httpapi.Poll(read failure then done) = (%q,%v), want RESULT with no error", res, err)
	}

	if got := polls.Load(); got < 2 {
		t.Errorf("✗ harness saw %d polls, want at least 2 (the read failure retried)", got)
	}

	if !t.Failed() {
		t.Log("✓ a response-read failure is retried and polling continues to the result")
	}
}

// TestPollLastErr verifies invariant #7: Last transient polling error.
//
// What is being tested:
// When every request loses its connection, httpapi.Poll using classifyResponse must return within
// one second with an error wrapping both ErrTransportTimeout and ErrTransportRequest.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollLastErr(t *testing.T) {
	_, srv := dropSrv(t, 1<<30, "")
	start := time.Now()

	err := httpapi.Poll(t.Context(), 2*time.Millisecond, 50*time.Millisecond, classProbe(t, srv.URL, "id-le"))
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("✗ Poll took %v, want a return proportionate to the 50ms deadline", elapsed)
	}

	if err == nil || !errors.Is(err, errs.ErrTransportTimeout) {
		t.Errorf("✗ httpapi.Poll(all drops) = %v, want a wrapped ErrTransportTimeout", err)
	}

	if !errors.Is(err, errs.ErrTransportRequest) {
		t.Errorf("✗ the timeout error's chain %v does not carry the last transport error", err)
	}

	if !t.Failed() {
		t.Log("✓ the deadline error carries the last transient transport error in its chain")
	}
}

// TestPollWaitDeadline verifies invariant #8: Deadline during a polling wait.
//
// What is being tested:
// When classifyResponse reports a pending job and the interval is 30 seconds, httpapi.Poll must
// interrupt the wait at its 100 ms deadline, returning ErrTransportTimeout within one second.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollWaitDeadline(t *testing.T) {
	_, srv := dropSrv(t, 0, `{"status":"pending"}`)
	start := time.Now()
	err := httpapi.Poll(t.Context(), 30*time.Second, 100*time.Millisecond, classProbe(t, srv.URL, "id-wd"))
	elapsed := time.Since(start)

	if err == nil || !errors.Is(err, errs.ErrTransportTimeout) {
		t.Errorf("✗ httpapi.Poll(deadline mid-wait) = %v, want a wrapped ErrTransportTimeout", err)
	}

	if elapsed < 100*time.Millisecond || elapsed > time.Second {
		t.Errorf("✗ Poll returned after %v, want the deadline to interrupt the 30s wait near 100ms", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ the deadline interrupts a long wait and surfaces the timeout")
	}
}

// TestPollInFlight verifies invariant #9: Deadline during a polling request.
//
// What is being tested:
// When a polling request stalls, httpapi.Poll using the classifyResponse probe must cancel the
// request at its 150 ms deadline and return ErrTransportTimeout within one second.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollInFlight(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	start := time.Now()
	err := httpapi.Poll(t.Context(), 10*time.Millisecond, 150*time.Millisecond, classProbe(t, srv.URL, "id-if"))
	elapsed := time.Since(start)

	if err == nil || !errors.Is(err, errs.ErrTransportTimeout) {
		t.Errorf("✗ httpapi.Poll(stalled probe) = %v, want a wrapped ErrTransportTimeout", err)
	}

	if elapsed < 150*time.Millisecond {
		t.Errorf("✗ Poll returned after %v, before the 150ms deadline", elapsed)
	}

	if elapsed > time.Second {
		t.Errorf("✗ Poll took %v — the deadline did not bound the in-flight call", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ the deadline bounds an in-flight probe call and surfaces the transport-timeout sentinel")
	}
}

// TestPollTerminalClassify verifies invariant #10: Terminal response classification.
//
// What is being tested:
// When classifyResponse receives a failed status with a provider message, httpapi.Poll must return
// ErrResponseGen containing that message after exactly one request and within five seconds.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollTerminalClassify(t *testing.T) {
	polls, srv := dropSrv(t, 0, `{"status":"failed","error":{"message":"boom"}}`)
	start := time.Now()

	err := httpapi.Poll(t.Context(), time.Millisecond, 30*time.Second, classProbe(t, srv.URL, "id-tc"))
	if err == nil || !errors.Is(err, errs.ErrResponseGen) || !strings.Contains(err.Error(), "boom") {
		t.Errorf("✗ httpapi.Poll(classify failure) = %v, want the classifier's error with its diagnostic", err)
	}

	if got := polls.Load(); got != 1 {
		t.Errorf("✗ harness saw %d polls after a terminal classification, want exactly 1", got)
	}

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("✗ Poll took %v on a terminal classification, want an immediate return", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ a classifier failure is terminal on the first poll with no retries")
	}
}

// TestPollCanceled verifies invariant #11: Canceled polling.
//
// What is being tested:
// Given cancellation during a minute-long wait or a stalled request, httpapi.Poll using the
// classifyResponse probe must return ErrCanceled within two seconds.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollCanceled(t *testing.T) {
	t.Run("mid-wait", func(t *testing.T) {
		_, srv := dropSrv(t, 0, `{"status":"pending"}`)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		start := time.Now()

		err := httpapi.Poll(ctx, time.Minute, time.Hour, classProbe(t, srv.URL, "id-cw"))
		if err == nil || !errors.Is(err, errs.ErrCanceled) {
			t.Errorf("✗ httpapi.Poll(canceled mid-wait) = %v, want a wrapped ErrCanceled", err)
		}

		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("✗ Poll took %v to notice cancellation of a minute-long wait", elapsed)
		}

		if !t.Failed() {
			t.Log("✓ mid-wait")
		}
	})
	t.Run("mid-call", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		t.Cleanup(srv.Close)

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		start := time.Now()

		err := httpapi.Poll(ctx, time.Millisecond, time.Hour, classProbe(t, srv.URL, "id-cc"))
		if err == nil || !errors.Is(err, errs.ErrCanceled) {
			t.Errorf("✗ httpapi.Poll(canceled mid-call) = %v, want a wrapped ErrCanceled", err)
		}

		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("✗ Poll took %v to notice cancellation of an in-flight call", elapsed)
		}

		if !t.Failed() {
			t.Log("✓ mid-call")
		}
	})

	if !t.Failed() {
		t.Log("✓ cancellation interrupts the wait and the in-flight call promptly with ErrCanceled")
	}
}

// TestPollDecodeCauses verifies invariant #12: Poll decoder causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given truncated JSON or a JSON array instead of an object, classifyResponse must return
// ErrResponseDecode with both state flags false and retain the original syntax or type error for
// errors.As.
// Kind: permanent.
func TestPollDecodeCauses(t *testing.T) {
	for _, responseBody := range []string{`{"status":`, `["pending"]`} {
		keepPolling, completed, err := classifyResponse("job", []byte(responseBody), []string{"pending"}, "done", []string{"failed"})
		if keepPolling || completed || !errors.Is(err, errs.ErrResponseDecode) {
			t.Errorf("✗ invalid poll response accepted: %q, %v", responseBody, err)
		}

		var (
			syntaxError *json.SyntaxError
			typeError   *json.UnmarshalTypeError
		)

		if responseBody[0] == '{' && !errors.As(err, &syntaxError) {
			t.Errorf("✗ original syntax error lost: %v", err)
		}

		if responseBody[0] == '[' && !errors.As(err, &typeError) {
			t.Errorf("✗ original type error lost: %v", err)
		}
	}

	if !t.Failed() {
		t.Log("✓ poll failures preserve the original decoder errors")
	}
}

// TestPollTemporaryStatuses verifies invariant #13: Temporary polling statuses.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given pending, temporary HTTP failure, and completed responses in sequence, httpapi.Poll using
// classifyResponse must return success with the completion marker after exactly three requests for
// each of HTTP 429, 502, 503, and 504.
func TestPollTemporaryStatuses(t *testing.T) {
	for _, statusCode := range []int{429, 502, 503, 504} {
		t.Run(strconv.Itoa(statusCode), func(t *testing.T) {
			var observations atomic.Int64

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				switch observations.Add(1) {
				case 1:
					fmt.Fprint(w, `{"status":"pending"}`)
				case 2:
					w.WriteHeader(statusCode)
					fmt.Fprint(w, `{"error":{"message":"temporary observation failure"}}`)
				default:
					fmt.Fprint(w, `{"status":"done"}`)
				}
			}))
			t.Cleanup(server.Close)

			poller := classProbe(t, server.URL, "recovering-job")
			err := httpapi.Poll(t.Context(), time.Millisecond, time.Second, poller)

			result := poller.completed
			if err != nil || result != "RESULT" || observations.Load() != 3 {
				t.Errorf("✗ status %d: result=%q error=%v observations=%d; want completed observation after recovery", statusCode, result, err, observations.Load())
			}

			if !t.Failed() {
				t.Log("✓ a temporary observation status recovers to completion")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ all selected temporary statuses recover within the polling budget")
	}
}

// TestPollStatusThenPendingTimeout verifies invariant #14: Timeout observation history.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given HTTP 503 followed by pending responses, httpapi.Poll using classifyResponse must make at
// least two requests and return an error wrapping ErrTransportTimeout and ErrResponseStatus while
// retaining the original provider message.
func TestPollStatusThenPendingTimeout(t *testing.T) {
	var observations atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if observations.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"error":{"message":"observation-history-marker"}}`)

			return
		}

		fmt.Fprint(w, `{"status":"pending"}`)
	}))
	t.Cleanup(server.Close)

	err := httpapi.Poll(t.Context(), time.Millisecond, 50*time.Millisecond, classProbe(t, server.URL, "waiting-job"))
	if !errors.Is(err, errs.ErrTransportTimeout) || !errors.Is(err, errs.ErrResponseStatus) {
		t.Errorf("✗ expiry error=%v; want timeout and preserved status classifications", err)
	}

	if err == nil || !strings.Contains(err.Error(), "observation-history-marker") || observations.Load() < 2 {
		t.Errorf("✗ error=%v observations=%d; want the failed observation followed by pending observations", err, observations.Load())
	}

	if !t.Failed() {
		t.Log("✓ timeout preserves the failed observation across later pending replies")
	}
}

// FuzzPollState verifies invariant #15: Poll response classification under arbitrary input.
//
// What is being tested:
// For arbitrary HTTP statuses and bodies, PollResponseError followed by classifyResponse must leave
// both state flags false on error or set exactly one of pending and completed on success, without
// panicking.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzPollState(f *testing.F) {
	for _, s := range []string{
		`{"status":"pending"}`, `{"status":"done"}`, `{"status":"failed"}`,
		`{"status":"failed","error":"x"}`, `{"status":"pending","error":null}`, `not json`, `{}`, ``,
	} {
		f.Add(200, []byte(s))
	}

	f.Fuzz(func(t *testing.T, status int, body []byte) {
		keepPolling, isDone := false, false

		err := PollResponseError("job", status, body, nil)
		if err == nil {
			keepPolling, isDone, err = classifyResponse("job", body, pollRunning, "done", pollFailed)
		}

		switch {
		case err != nil && (keepPolling || isDone):
			t.Errorf("✗ Classify errored but returned keep=%v done=%v, want both false", keepPolling, isDone)
		case err == nil && keepPolling == isDone:
			t.Errorf("✗ Classify(status=%d) returned keep=%v done=%v with no error, want exactly one true", status, keepPolling, isDone)
		}

		if !t.Failed() {
			t.Logf("✓ the classification stayed exclusive")
		}
	})
}

// The polling fixtures share these pending and failed status words.
//
//nolint:gochecknoglobals // read-only test vocabulary.
var (
	pollRunning = []string{"pending", "queued"}
	pollFailed  = []string{"failed", "expired"}
)

// classCheck checks HTTP failures and the shared status vocabulary at a harness URL, retaining
// "RESULT" on completion.
type classCheck struct {
	test      testing.TB
	url, id   string
	completed string
}

// Poll retrieves and classifies one response, storing the completion marker on success.
func (c *classCheck) Poll(ctx context.Context) (bool, error) {
	c.test.Helper()

	status, body, err := httpapi.GetAuth(ctx, c.url, httpapi.AuthCredential{}, metadata.Asynchronous, nil)
	if err := PollResponseError(c.id, status, body, err); err != nil {
		return false, err
	}

	_, jobDone, cErr := classifyResponse(c.id, body, pollRunning, "done", pollFailed)
	if cErr != nil {
		return false, cErr
	}

	if jobDone {
		c.completed = "RESULT"

		return true, nil
	}

	return false, nil
}

// classProbe creates a probe that applies classifyResponse to responses from the test server.
func classProbe(test testing.TB, url, id string) *classCheck {
	test.Helper()

	return &classCheck{test: test, url: url, id: id}
}

// dropSrv closes the first n request connections without a response, then serves body. It returns a
// request counter and the test server.
func dropSrv(t *testing.T, n int, body string) (*atomic.Int64, *httptest.Server) {
	t.Helper()

	var polls atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if polls.Add(1) <= int64(n) {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Errorf("✗ harness lacks Hijacker — cannot simulate a dropped connection")

				return
			}

			conn, _, _ := hj.Hijack()
			_ = conn.Close()

			return
		}

		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	return &polls, srv
}
