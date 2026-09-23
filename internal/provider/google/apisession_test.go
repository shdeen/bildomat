package google

// Invariants tested:
//  1. Download credential origin: For a file on the API origin, downloadArtifact must GET
//     /files/f1:download?alt=media with x-goog-api-key and return the served bytes in a temporary
//     .mp4 file. For a URL on another origin, it must GET without that credential and return the
//     served bytes in a temporary file.

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/httpapi"
)

// TestDlOrigin verifies invariant #1: Download credential origin.
//
// What is being tested:
// For a file on the API origin, downloadArtifact must GET /files/f1:download?alt=media with
// x-goog-api-key and return the served bytes in a temporary .mp4 file. For a URL on another origin,
// it must GET without that credential and return the served bytes in a temporary file.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDlOrigin(t *testing.T) {
	t.Run("same-origin file download is credentialed", dlSameOrigin)
	t.Run("external-host download is bare", dlExternal)

	if !t.Failed() {
		t.Log("✓ the origin rule holds on both download paths")
	}
}

// dlSameOrigin checks authenticated download of a file on the API origin and removes its artifact.
func dlSameOrigin(t *testing.T) {
	t.Helper()
	r := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = io.WriteString(w, "VIDBYTES")
	})

	generatedMedia, err := (&apiSession{settings: &catalog.AdapterAPI{APIBase: r.srv.URL}, credential: httpapi.HeaderCred(googleKeyHeader, "sek")}).downloadArtifact(context.Background(), r.srv.URL+"/files/f1"+downloadQuerySuffix, ".mp4", nil)
	if err != nil {
		t.Errorf("✗ downloadFile failed: %v", err)

		return
	}

	h := r.first(t)
	if h.reqPath != "/files/f1:download?alt=media" {
		t.Errorf("✗ download path = %q, want /files/f1:download?alt=media", h.reqPath)
	}

	if h.reqMethod != http.MethodGet {
		t.Errorf("✗ download method = %s, want GET", h.reqMethod)
	}

	if h.apiKey != "sek" {
		t.Errorf("✗ same-origin download missing the x-goog-api-key credential (got %q)", h.apiKey)
	}

	fileBacked(t, generatedMedia, "VIDBYTES")

	if generatedMedia.FileExt != ".mp4" {
		t.Errorf("✗ FileExt = %q, want .mp4", generatedMedia.FileExt)
	}

	if !t.Failed() {
		t.Log("✓ the Files download GETs its own origin credentialed and yields a file-backed artifact")
	}
}

// dlExternal checks that a download outside the API origin receives no credential and removes its
// artifact.
func dlExternal(t *testing.T) {
	t.Helper()
	base := newRecServer(t, func(_ http.ResponseWriter, _ *http.Request) {})
	ext := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = io.WriteString(w, "EXTBYTES")
	})

	generatedMedia, err := (&apiSession{settings: &catalog.AdapterAPI{APIBase: base.srv.URL}, credential: httpapi.HeaderCred(googleKeyHeader, "sek")}).downloadArtifact(context.Background(), ext.srv.URL+"/media/m1", ".mp4", nil)
	if err != nil {
		t.Errorf("✗ downloadArtifact failed: %v", err)

		return
	}

	h := ext.first(t)
	if h.apiKey != "" {
		t.Errorf("✗ credential leaked to an external host: x-goog-api-key=%q", h.apiKey)
	}

	if h.reqMethod != http.MethodGet {
		t.Errorf("✗ external download method = %s, want GET", h.reqMethod)
	}

	fileBacked(t, generatedMedia, "EXTBYTES")

	if !t.Failed() {
		t.Log("✓ an external-host URI downloads bare — the credential never leaves the origin")
	}
}
