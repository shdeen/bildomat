package google

// Invariants tested:
//  1. Google file availability wait: httpapi.Poll with fileReadyProbe must GET the canonical file
//     path with the Google credential twice for PROCESSING followed by ACTIVE, then succeed. FAILED
//     must return ErrResponseGen naming the resource. LIMBO must return ErrResponseUnknown naming
//     both the resource and status.
//  2. Canonical file identifiers: Given a Google download URL, a local server download URL, or a
//     bare files/<id> resource, canonicalFileID must return the exact files/<id> value. Missing
//     file segments, empty IDs, empty input, and unparseable text must return ErrResponseDecode
//     under ErrResponse, naming the supplied reference when nonempty.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
)

// TestWaitFile verifies invariant #1: Google file availability wait.
//
// What is being tested:
// httpapi.Poll with fileReadyProbe must GET the canonical file path with the Google credential
// twice for PROCESSING followed by ACTIVE, then succeed. FAILED must return ErrResponseGen naming
// the resource. LIMBO must return ErrResponseUnknown naming both the resource and status.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWaitFile(t *testing.T) {
	t.Run("processing then active", waitDone)
	t.Run("failed state errors", waitFailed)
	t.Run("unknown state errors verbatim", waitUnknown)

	if !t.Failed() {
		t.Log("✓ the file-state probe classifies every documented state")
	}
}

// TestFileID verifies invariant #2: Canonical file identifiers.
//
// What is being tested:
// Given a Google download URL, a local server download URL, or a bare files/<id> resource,
// canonicalFileID must return the exact files/<id> value. Missing file segments, empty IDs, empty
// input, and unparseable text must return ErrResponseDecode under ErrResponse, naming the supplied
// reference when nonempty.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFileID(t *testing.T) {
	ok := []struct{ name, ref, want string }{
		{"documented download URI", "https://generativelanguage.googleapis.com/v1beta/files/xk29a:download?alt=media", "files/xk29a"},
		{"harness-origin URI", "http://127.0.0.1:9999/v1beta/files/f7:download?alt=media", "files/f7"},
		{"bare resource", "files/xk29a", "files/xk29a"},
	}
	for _, c := range ok {
		t.Run(c.name, func(t *testing.T) {
			checkFileOk(t, c.ref, c.want)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	bad := []struct{ name, ref string }{
		{"no files segment", "https://example.com/nothing/here"},
		{"empty uri id", "https://generativelanguage.googleapis.com/v1beta/files/:download?alt=media"},
		{"bare empty id", "files/"},
		{"empty reference", ""},
		{"unparseable junk", "not a uri at all"},
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
		t.Log("✓ every normalization form resolves or classifies as expected")
	}
}

// waitForFileReady polls a file resource through the file-ready probe every five milliseconds for
// up to one second, discarding the result and returning the terminal error.
func waitForFileReady(ctx context.Context, test testing.TB, apiBase string, c httpapi.AuthCredential, fileResource string) error {
	test.Helper()

	err := httpapi.Poll(ctx, 5*time.Millisecond, time.Second, &fileReadyProbe{apiBase: apiBase, credential: c, fileResource: fileResource})

	return err
}

// stateSrv serves the scripted state bodies in order (the last repeats).
func stateSrv(t *testing.T, states ...string) *recServer {
	t.Helper()

	n := 0

	return newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		i := n
		if i >= len(states) {
			i = len(states) - 1
		}

		n++

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, states[i])
	})
}

// waitDone checks authenticated polling from a processing file to an active file.
func waitDone(t *testing.T) {
	t.Helper()
	r := stateSrv(t, `{"state":"PROCESSING"}`, `{"state":"ACTIVE"}`)

	err := waitForFileReady(context.Background(), t, r.srv.URL, httpapi.HeaderCred(googleKeyHeader, "k1"), "files/f1")
	if err != nil {
		t.Errorf("✗ waitForFileReady errored on PROCESSING→ACTIVE: %v", err)
	}

	want := []string{"/files/f1", "/files/f1"}
	if got := r.paths(); !slices.Equal(got, want) {
		t.Errorf("✗ probe paths = %v, want %v", got, want)
	}

	wantMethod(t, r, "/files/f1", http.MethodGet)

	if key, _ := r.keyOf("/files/f1"); key != "k1" {
		t.Errorf("✗ probe request missing the x-goog-api-key credential")
	}

	if !t.Failed() {
		t.Log("✓ PROCESSING keeps polling and ACTIVE finishes, GETting the canonical resource path")
	}
}

// waitFailed checks the failure classification and file identity for a failed file.
func waitFailed(t *testing.T) {
	t.Helper()
	r := stateSrv(t, `{"state":"FAILED","error":{"message":"encode broke"}}`)

	err := waitForFileReady(context.Background(), t, r.srv.URL, httpapi.HeaderCred(googleKeyHeader, "k1"), "files/f2")
	if err == nil {
		t.Errorf("✗ FAILED state returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseGen) {
		t.Errorf("✗ FAILED error outside the generation-failure chain: %v", err)
	}

	if !strings.Contains(err.Error(), "files/f2") {
		t.Errorf("✗ FAILED error does not name the resource: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ FAILED errors naming the resource")
	}
}

// waitUnknown checks that an unknown state retains the file identity and observed value.
func waitUnknown(t *testing.T) {
	t.Helper()
	r := stateSrv(t, `{"state":"LIMBO"}`)

	err := waitForFileReady(context.Background(), t, r.srv.URL, httpapi.HeaderCred(googleKeyHeader, "k1"), "files/f3")
	if err == nil {
		t.Errorf("✗ unknown state returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseUnknown) {
		t.Errorf("✗ unknown-state error outside the unexpected-status chain: %v", err)
	}

	if !strings.Contains(err.Error(), "files/f3") || !strings.Contains(err.Error(), "LIMBO") {
		t.Errorf("✗ unknown-state error does not name the resource and observed state: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ an unknown state errors naming the resource and the observed value")
	}
}

// checkFileOk asserts one extractable reference normalizes exactly.
func checkFileOk(t *testing.T, ref, want string) {
	t.Helper()

	got, err := canonicalFileID(ref)
	if err != nil {
		t.Errorf("✗ canonicalFileID(%q) error: %v", ref, err)
	}

	if got != want {
		t.Errorf("✗ canonicalFileID(%q) = %q, want %q", ref, got, want)
	}

	if !t.Failed() {
		t.Logf("✓ %q normalizes to %s", ref, want)
	}
}

// checkFileBad asserts one unextractable reference is a classified error.
func checkFileBad(t *testing.T, ref string) {
	t.Helper()

	got, err := canonicalFileID(ref)
	if err == nil {
		t.Errorf("✗ canonicalFileID(%q) = %q with no error, want a classified error", ref, got)

		return
	}

	if !errors.Is(err, errs.ErrResponseDecode) || !errors.Is(err, errs.ErrResponse) {
		t.Errorf("✗ error = %v, want the decode sentinel under the response root", err)
	}

	if ref != "" && !strings.Contains(err.Error(), ref) {
		t.Errorf("✗ error does not name the reference: %v", err)
	}

	if !t.Failed() {
		t.Logf("✓ %q is a classified error", ref)
	}
}
