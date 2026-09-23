package httpapi

// Invariants tested:
//  1. Fetch credentials: Given no credential, Fetch must succeed without Authorization or X-Key
//     headers.
//  2. Case-insensitive media types: Given mixed-case declared PNG or SVG types, artifact.New must
//     choose .png or .svg.

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/shdeen/bildomat/internal/artifact"
)

// TestFetchCred verifies invariant #1: Fetch credentials.
//
// What is being tested:
// Given no credential, Fetch must succeed without Authorization or X-Key headers. Given an X-Key
// credential, it must succeed and send the supplied key.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFetchCred(t *testing.T) {
	recorder, srv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		fmt.Fprint(w, "DATA")
	})

	generatedMedia, err := Fetch(t.Context(), srv.URL, AuthCredential{}, ".bin", nil)
	rmArtifact(t, generatedMedia)

	if err != nil {
		t.Errorf("✗ bare fetch failed: %v", err)
	} else {
		req := recorder.last(t)
		if v := req.header.Get("Authorization"); v != "" {
			t.Errorf("✗ bare fetch sent Authorization %q, want no auth header", v)
		}

		if v := req.header.Get("X-Key"); v != "" {
			t.Errorf("✗ bare fetch sent x-key %q, want no auth header", v)
		}
	}

	credential := HeaderCred("x-key", "k")
	art2, err := Fetch(t.Context(), srv.URL, credential, ".bin", nil)
	rmArtifact(t, art2)

	if err != nil {
		t.Errorf("✗ credentialed fetch failed: %v", err)
	} else if v := recorder.last(t).header.Get("X-Key"); v != "k" {
		t.Errorf("✗ credentialed fetch x-key = %q, want k", v)
	}

	if !t.Failed() {
		t.Log("✓ Fetch sends nothing bare and the credential header otherwise")
	}
}

// TestExtChainCase verifies invariant #2: Case-insensitive media types.
//
// What is being tested:
// Given mixed-case declared PNG or SVG types, artifact.New must choose .png or .svg. Fetch must
// choose .png for a parameterized mixed-case PNG header even when the response bytes have a JPEG
// signature.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestExtChainCase(t *testing.T) {
	generatedMedia, err := artifact.New(jpegStub(t), "IMAGE/PNG", ".webp")
	if err != nil || generatedMedia.FileExt != pngExt {
		t.Errorf("✗ artifact.New(IMAGE/PNG over JPEG magic) = (%q, %v), want .png", generatedMedia.FileExt, err)
	}

	svg, err := artifact.New(svgStub(t), "Image/SVG+XML", pngExt)
	if err != nil || svg.FileExt != ".svg" {
		t.Errorf("✗ artifact.New(Image/SVG+XML) = (%q, %v), want .svg", svg.FileExt, err)
	}

	_, srv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "IMAGE/PNG; Charset=Binary")
		_, _ = w.Write(jpegStub(t))
	})
	fart, err := Fetch(t.Context(), srv.URL, AuthCredential{}, ".webp", nil)
	rmArtifact(t, fart)

	if err != nil || fart.FileExt != pngExt {
		t.Errorf("✗ Fetch(IMAGE/PNG with parameters, JPEG magic) = (%q, %v), want .png", fart.FileExt, err)
	}

	if !t.Failed() {
		t.Log("✓ mixed-case declared media types win over conflicting magic bytes")
	}
}

// rmArtifact removes a downloaded artifact's temporary file.
func rmArtifact(t *testing.T, generatedMedia artifact.Media) {
	t.Helper()

	if generatedMedia.TmpPath != "" {
		_ = os.Remove(generatedMedia.TmpPath)
	}
}

// jpegStub is a minimal JPEG header (SOI + JFIF APP0 marker) — enough magic for content-type
// detection; the extension contract does not require a decodable image.
func jpegStub(test testing.TB) []byte {
	test.Helper()

	return []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00}
}

// svgStub is a minimal SVG document; Go byte-sniffing cannot classify it, so only a declared MIME
// can yield its truthful extension.
func svgStub(test testing.TB) []byte {
	test.Helper()

	return []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"/>`)
}
