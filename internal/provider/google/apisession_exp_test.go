package google

// Invariants tested:
//  1. File timeout identity: When a video file remains PROCESSING until its timeout,
//     createBlockArtifact must return ErrTransportTimeout naming files/file-identity and
//     model-identity, with neither artifact bytes nor a temporary path.
//  2. Operation timeout identity: When an operation remains unfinished until its timeout,
//     pollOperation must return ErrTransportTimeout and name operations/operation-identity in the
//     error.
//  3. Interaction step traversal under arbitrary input: For arbitrary interaction response bytes
//     and either requested medium, extractInteractionMedia must not panic. It must return either an
//     artifact containing bytes or a temporary path, but not both, or an error matching
//     ErrResponse, ErrTransport, or ErrCanceled with no artifact data or path.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
)

// TestFileTimeoutIdentity verifies invariant #1: File timeout identity.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// When a video file remains PROCESSING until its timeout, createBlockArtifact must return
// ErrTransportTimeout naming files/file-identity and model-identity, with neither artifact bytes
// nor a temporary path.
func TestFileTimeoutIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"state":"PROCESSING"}`))
	}))
	defer server.Close()

	generated, err := (&apiSession{settings: &catalog.AdapterAPI{APIBase: server.URL, FilePollInterval: 3600, FilePollTimeout: 1}, credential: httpapi.HeaderCred(googleKeyHeader, "test-key"), model: "google/model-identity"}).createBlockArtifact(t.Context(),
		&interactionBlock{URI: "files/file-identity", Type: "video"}, ".mp4", nil)
	if !errors.Is(err, errs.ErrTransportTimeout) || !strings.Contains(err.Error(), "files/file-identity") || !strings.Contains(err.Error(), "model-identity") {
		t.Errorf("✗ file timeout loses identity or classification: %v", err)
	}

	if generated.TmpPath != "" || len(generated.Data) != 0 {
		t.Errorf("✗ incomplete file returned artifact %q", generated.TmpPath)
	}

	if !t.Failed() {
		t.Log("✓ timeout identifies the file operation and model")
	}
}

// TestOperationTimeoutIdentity verifies invariant #2: Operation timeout identity.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// When an operation remains unfinished until its timeout, pollOperation must return
// ErrTransportTimeout and name operations/operation-identity in the error.
func TestOperationTimeoutIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"done":false}`))
	}))
	defer server.Close()

	_, err := (&apiSession{settings: &catalog.AdapterAPI{APIBase: server.URL, PollInterval: 3600, PollTimeout: 1}, credential: httpapi.HeaderCred(googleKeyHeader, "test-key")}).pollOperation(t.Context(), "operations/operation-identity", nil)
	if !errors.Is(err, errs.ErrTransportTimeout) || !strings.Contains(err.Error(), "operations/operation-identity") {
		t.Errorf("✗ operation timeout loses identity or classification: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ timeout identifies the operation resource")
	}
}

// FuzzSteps verifies invariant #3: Interaction step traversal under arbitrary input.
//
// What is being tested:
// For arbitrary interaction response bytes and either requested medium, extractInteractionMedia
// must not panic. It must return either an artifact containing bytes or a temporary path, but not
// both, or an error matching ErrResponse, ErrTransport, or ErrCanceled with no artifact data or
// path.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSteps(f *testing.F) {
	seeds := []string{
		`not json`,
		`{"steps":[]}`,
		`{"steps":[{"type":"model_output","content":[{"type":"image","mime_type":"image/png","data":"aGk="}]}]}`,
		`{"steps":[{"type":"model_output","content":[{"type":"image","uri":"::bad::"}]}]}`,
		`{"steps":[{"type":"model_output","content":[{"type":"image","uri":"files/x"}]}]}`,
		`{"steps":[{"type":"model_output","status":"error","error":{"code":1,"message":"m"}}]}`,
		`{"steps":[{"type":"thought","summary":[{"type":"text","text":"t"}]},{"type":"model_output","content":[{"type":"text","text":"s"}]}]}`,
		`{"steps":[{"type":"model_output","content":[{"type":"video","mime_type":"video/mp4","data":"aGk="}]}]}`,
		`{"steps":[{"type":"model_output","content":[{"type":"video","uri":"::bad::"}]}]}`,
		`{"steps":[{"type":"model_output","content":[{"type":"video","uri":"files/v"}]}]}`,
		`{"steps":[{"type":"model_output","content":[{"type":"video","data":"!!!"},{"type":"video"}]}]}`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	media := []struct{ want, fallback string }{
		{"image", ".png"},
		{"video", ".mp4"},
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		for _, m := range media {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Millisecond)
			generatedMedia, _, err := (&apiSession{settings: &catalog.AdapterAPI{APIBase: "http://127.0.0.1:1", FilePollInterval: 1, FilePollTimeout: 1}, credential: httpapi.HeaderCred(googleKeyHeader, "k"), model: "fuzz-model"}).extractInteractionMedia(ctx, body, m.want, m.fallback, nil)

			cancel()
			checkWalkOutcome(t, m.want, generatedMedia, err)
		}

		if !t.Failed() {
			t.Log("✓ neither traversal panics nor yields an unclassified outcome")
		}
	})
}

// checkWalkOutcome asserts the fuzz invariant on one traversal outcome: a classified error carrying
// no artifact, or one artifact carrying either bytes or a file, never both.
func checkWalkOutcome(t *testing.T, media string, generatedMedia artifact.Media, err error) {
	t.Helper()

	if err != nil {
		if !classified(t, err) {
			t.Errorf("✗ unclassified %s-traversal error: %v", media, err)
		}

		if generatedMedia.TmpPath != "" || len(generatedMedia.Data) > 0 {
			t.Errorf("✗ artifact delivered beside a %s-traversal error: %+v", media, generatedMedia)
		}

		return
	}

	if (len(generatedMedia.Data) > 0) == (generatedMedia.TmpPath != "") {
		t.Errorf("✗ the %s-traversal artifact carries both bytes and a file, or neither: %+v", media, generatedMedia)
	}

	rmArtifact(t, generatedMedia)
}

// classified reports whether an error carries one of the traversal's reachable category roots.
func classified(test testing.TB, err error) bool {
	test.Helper()

	return errors.Is(err, errs.ErrResponse) || errors.Is(err, errs.ErrTransport) || errors.Is(err, errs.ErrCanceled)
}
