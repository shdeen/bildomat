package httpapi

// Invariants tested:
//  1. Download input media defers compatibility: Given AVIF and extensionless URL sources,
//     DownloadInputMedia must return two byte-backed records with the source URLs in Filepath and
//     empty URL fields.
//  2. Remote media identity: Given misleading URL suffixes, ResolveInputMediaTypes must use image
//     or video response headers, or detected content after HEAD fails.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/shdeen/bildomat/internal/media"
)

// TestDownloadInputMediaDefersCompatibility verifies invariant #1: Download input media defers
// compatibility.
//
// What is being tested:
// Given AVIF and extensionless URL sources, DownloadInputMedia must return two byte-backed records
// with the source URLs in Filepath and empty URL fields. The AVIF record must preserve its bytes
// and MIME; the extensionless record must contain bytes without a format rejection.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDownloadInputMediaDefersCompatibility(t *testing.T) {
	avifBytes := []byte("\x00\x00\x00\x18ftypavif\x00\x00\x00\x00avifmif1")
	unknownBytes := []byte("provider decides whether this is usable")

	mediaServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/reference.avif":
			_, _ = response.Write(avifBytes)
		case "/reference":
			_, _ = response.Write(unknownBytes)
		default:
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(mediaServer.Close)

	mediaInputs := []media.Input{
		{URL: mediaServer.URL + "/reference.avif", MIME: "image/avif"},
		{URL: mediaServer.URL + "/reference", MIME: "application/octet-stream"},
	}

	downloadedInputs, err := DownloadInputMedia(context.Background(), mediaInputs)
	if err != nil {
		t.Fatalf("💣 DownloadInputMedia: %v", err)
	}

	if len(downloadedInputs) != 2 {
		t.Fatalf("💣 downloaded input count = %d, want 2", len(downloadedInputs))
	}

	if downloadedInputs[0].URL != "" || downloadedInputs[0].Filepath != mediaInputs[0].URL || downloadedInputs[0].MIME != "image/avif" || !bytes.Equal(downloadedInputs[0].Bytes, avifBytes) {
		t.Errorf("✗ AVIF download = %+v, want unchanged bytes and declared MIME", downloadedInputs[0])
	}

	if downloadedInputs[1].URL != "" || downloadedInputs[1].Filepath != mediaInputs[1].URL || len(downloadedInputs[1].Bytes) == 0 {
		t.Errorf("✗ extensionless download = %+v, want bytes without a compatibility rejection", downloadedInputs[1])
	}

	if !t.Failed() {
		t.Log("✓ URL downloads preserve provider-decided formats instead of applying the local-file allowlist")
	}
}

// TestRemoteMediaIdentity verifies invariant #2: Remote media identity.
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
// What is being tested:
// Given misleading URL suffixes, ResolveInputMediaTypes must use image or video response headers,
// or detected content after HEAD fails. It must preserve URL order, explicit zero time, local
// bytes, and established MIME and anchors without changing caller records or sending provider
// credentials.
func TestRemoteMediaIdentity(t *testing.T) {
	var carriedCredential atomic.Bool

	mediaServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "" || request.Header.Get("X-Key") != "" || request.Header.Get("X-Goog-Api-Key") != "" {
			carriedCredential.Store(true)
		}

		switch request.URL.Path {
		case "/picture.mp4":
			response.Header().Set("Content-Type", "IMAGE/JPEG; charset=binary")
		case "/movie.png":
			response.Header().Set("Content-Type", "VIDEO/MP4; charset=binary")
		case "/sample.png":
			if request.Method == http.MethodHead {
				response.WriteHeader(http.StatusMethodNotAllowed)

				return
			}

			response.Header().Set("Content-Type", "application/octet-stream")
			_, _ = response.Write([]byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom"))
		case "/known":
			t.Error("✗ already-established MIME triggered a request")
			response.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(mediaServer.Close)

	for _, inputCase := range []struct{ sourcePath, mimeType string }{
		{"/picture.mp4?download=.mp4", "image/jpeg"},
		{"/movie.png?download=.png", "video/mp4"},
		{"/sample.png", "video/mp4"},
	} {
		mediaInputs := []media.Input{
			{URL: mediaServer.URL + inputCase.sourcePath, Time: new(0.0)},
			{Bytes: []byte("local bytes"), MIME: "image/png", Filepath: "local.png"},
			{URL: mediaServer.URL + "/known", MIME: "image/avif", FrameAnchor: media.FrameLast},
		}

		resolvedInputs, err := ResolveInputMediaTypes(t.Context(), mediaInputs)
		if err != nil || len(resolvedInputs) != len(mediaInputs) {
			t.Errorf("✗ resolution returned %d inputs and error %v", len(resolvedInputs), err)

			continue
		}

		if resolvedInputs[0].MIME != inputCase.mimeType || resolvedInputs[0].URL != mediaInputs[0].URL || resolvedInputs[0].Filepath != "" || len(resolvedInputs[0].Bytes) != 0 {
			t.Errorf("✗ resolved remote source = %+v", resolvedInputs[0])
		}

		if seconds, supplied := resolvedInputs[0].FrameTime(); !supplied || seconds != 0 {
			t.Errorf("✗ explicit opening timestamp became %v, %v", seconds, supplied)
		}

		if resolvedInputs[1].MIME != "image/png" || resolvedInputs[1].Filepath != "local.png" || !bytes.Equal(resolvedInputs[1].Bytes, mediaInputs[1].Bytes) {
			t.Errorf("✗ local input changed: %+v", resolvedInputs[1])
		}

		if resolvedInputs[2].MIME != "image/avif" || resolvedInputs[2].URL != mediaInputs[2].URL || resolvedInputs[2].FrameAnchor != media.FrameLast || mediaInputs[0].MIME != "" {
			t.Errorf("✗ established metadata or caller records changed: %+v", resolvedInputs)
		}
	}

	if carriedCredential.Load() {
		t.Error("✗ source inspection carried a provider credential")
	}

	if !t.Failed() {
		t.Log("✓ remote types follow metadata or content while source records remain intact")
	}
}
