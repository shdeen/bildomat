package google

// Invariants tested:
//  1. Malformed file-state response: When a file probe returns malformed JSON, httpapi.Poll with
//     fileReadyProbe must stop after one request and return ErrResponseDecode naming the file
//     resource.
//  2. Strict file identifier parsing: Given a relative path with an embedded files segment, a
//     schemeless host path, two files segments, or a segment after the ID, canonicalFileID must
//     return ErrResponseDecode under ErrResponse and name the rejected reference.
//  3. Missing file state: When a file probe returns an empty object, httpapi.Poll with
//     fileReadyProbe must return ErrResponseUnknown and identify both the file resource and the
//     missing state.
//  4. File-state HTTP failure: When a file probe returns HTTP 500, httpapi.Poll with fileReadyProbe
//     must stop after one request and return ErrResponseStatus.
//  5. Observation recovery: After PROCESSING followed by HTTP 429, 502, 503, or 504, httpapi.Poll
//     with fileReadyProbe must issue a third GET and accept ACTIVE without an error.
//  6. Google file state parsing under arbitrary input: For arbitrary HTTP status and body values,
//     PollResponseError followed by checkFileReady must not panic or report completion with an
//     error. Every returned error must match ErrResponse.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/provider"
)

// TestFileStateBadJSON verifies invariant #1: Malformed file-state response.
//
// What is being tested:
// When a file probe returns malformed JSON, httpapi.Poll with fileReadyProbe must stop after one
// request and return ErrResponseDecode naming the file resource.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFileStateBadJSON(t *testing.T) {
	r := stateSrv(t, `{not json`)

	err := waitForFileReady(context.Background(), t, r.srv.URL, httpapi.HeaderCred(googleKeyHeader, "k1"), "files/fx")
	if !errors.Is(err, errs.ErrResponseDecode) {
		t.Errorf("✗ err = %v, want the decode chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "files/fx") {
		t.Errorf("✗ error does not name the resource: %v", err)
	}

	if len(r.paths()) != 1 {
		t.Errorf("✗ %d probes on malformed JSON, want 1 (terminal)", len(r.paths()))
	}

	if !t.Failed() {
		t.Log("✓ a malformed state body is a terminal classified decode error")
	}
}

// TestFileIDStrict verifies invariant #2: Strict file identifier parsing.
//
// What is being tested:
// Given a relative path with an embedded files segment, a schemeless host path, two files segments,
// or a segment after the ID, canonicalFileID must return ErrResponseDecode under ErrResponse and
// name the rejected reference.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFileIDStrict(t *testing.T) {
	bad := []struct{ name, ref string }{
		{"relative path with a files segment", "junk/files/file-1"},
		{"schemeless host path", "example.com/v1beta/files/f1:download"},
		{"two files segments", "https://generativelanguage.googleapis.com/v1beta/files/a/files/b"},
		{"segment after the id", "https://generativelanguage.googleapis.com/v1beta/files/a/extra"},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			checkFileBad(t, c.ref)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ nothing outside the two accepted forms reaches the probe")
	}
}

// TestFileStateMissing verifies invariant #3: Missing file state.
//
// What is being tested:
// When a file probe returns an empty object, httpapi.Poll with fileReadyProbe must return
// ErrResponseUnknown and identify both the file resource and the missing state.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFileStateMissing(t *testing.T) {
	r := stateSrv(t, `{}`)

	err := waitForFileReady(context.Background(), t, r.srv.URL, httpapi.HeaderCred(googleKeyHeader, "k1"), "files/f0")
	if !errors.Is(err, errs.ErrResponseUnknown) {
		t.Errorf("✗ err = %v, want the unexpected-status chain", err)
	}

	if err != nil && (!strings.Contains(err.Error(), "files/f0") || !strings.Contains(err.Error(), "missing")) {
		t.Errorf("✗ error does not name the resource and the missing state: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ an absent state classifies unknown, named as missing")
	}
}

// TestFileHTTP verifies invariant #4: File-state HTTP failure.
//
// What is being tested:
// When a file probe returns HTTP 500, httpapi.Poll with fileReadyProbe must stop after one request
// and return ErrResponseStatus.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFileHTTP(t *testing.T) {
	r := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "probe boom")
	})

	err := waitForFileReady(context.Background(), t, r.srv.URL, httpapi.HeaderCred(googleKeyHeader, "k1"), "files/f8")
	if !errors.Is(err, errs.ErrResponseStatus) {
		t.Errorf("✗ err = %v, want the error-status chain", err)
	}

	if len(r.paths()) != 1 {
		t.Errorf("✗ %d probes after the terminal status, want 1", len(r.paths()))
	}

	if !t.Failed() {
		t.Log("✓ a non-2xx probe response is terminal under the status chain")
	}
}

// TestGoogleFilePollRecovery verifies invariant #5: Observation recovery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// After PROCESSING followed by HTTP 429, 502, 503, or 504, httpapi.Poll with fileReadyProbe must
// issue a third GET and accept ACTIVE without an error.
func TestGoogleFilePollRecovery(t *testing.T) {
	for _, statusCode := range []int{429, 502, 503, 504} {
		t.Run(strconv.Itoa(statusCode), func(t *testing.T) {
			server, observations := pollRecoveryServer(t, statusCode, `{"state":"PROCESSING"}`, `{"state":"ACTIVE"}`)

			poller := &fileReadyProbe{apiBase: server.URL, fileResource: "files/file-identity"}

			err := httpapi.Poll(t.Context(), time.Millisecond, time.Second, poller)
			if err != nil || *observations != 3 {
				t.Errorf("✗ completion error=%v observations=%d; want successful third observation", err, *observations)
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

// FuzzFileState verifies invariant #6: Google file state parsing under arbitrary input.
//
// What is being tested:
// For arbitrary HTTP status and body values, PollResponseError followed by checkFileReady must not
// panic or report completion with an error. Every returned error must match ErrResponse.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzFileState(f *testing.F) {
	for _, s := range []string{`{"state":"PROCESSING"}`, `{"state":"ACTIVE"}`, `{"state":"FAILED"}`, `{}`, `not json`, ``} {
		f.Add(200, []byte(s))
	}

	f.Add(500, []byte(`{}`))
	f.Fuzz(func(t *testing.T, status int, body []byte) {
		done := false

		err := provider.PollResponseError("files/fz", status, body, nil)
		if err == nil {
			done, err = checkFileReady("files/fz", body)
		}

		if done && err != nil {
			t.Errorf("✗ checkFileReady returned done together with an error: %v", err)
		}

		if err != nil && !errors.Is(err, errs.ErrResponse) {
			t.Errorf("✗ checkFileReady error outside the response category: %v", err)
		}

		if !t.Failed() {
			t.Logf("✓ the state classification stayed exclusive and categorized")
		}
	})
}

// pollRecoveryServer serves pending, a selected HTTP failure, then completion.
func pollRecoveryServer(t *testing.T, statusCode int, pending, completed string) (*httptest.Server, *int) {
	t.Helper()

	observations := new(int)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("✗ observation used %s instead of GET", request.Method)
		}

		*observations++
		switch *observations {
		case 1:
			_, _ = writer.Write([]byte(pending))
		case 2:
			writer.WriteHeader(statusCode)
			_, _ = writer.Write([]byte(`{"message":"temporary observation failure"}`))
		default:
			_, _ = writer.Write([]byte(completed))
		}
	}))
	t.Cleanup(server.Close)

	return server, observations
}
