package httpapi

// Invariants tested:
//  1. Remote media failures: Given an invalid URL, ResolveInputMediaTypes must return
//     ErrInputMediaRead and ErrTransportCreate.
//  2. Bounded media inspection: Given a server that sends a 512-byte PNG prefix then waits
//     indefinitely, ResolveInputMediaTypes must finish successfully within the one-second context
//     and return one record with the original URL, image/png, and no body bytes.
//  3. Partial download results: Given a successful URL followed by a failed or malformed URL,
//     DownloadInputMedia must return ErrInputMediaRead and all three input records in order.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// TestRemoteMediaFailures verifies invariant #1: Remote media failures.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given an unavailable or unrecognizable remote source, ResolveInputMediaTypes must return no
// inputs and an error naming the source with ErrInputMediaRead and the applicable status or MIME
// category. A canceled context must return no inputs and preserve both ErrCanceled and
// context.Canceled.
func TestRemoteMediaFailures(t *testing.T) {
	mediaServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/unavailable" {
			response.WriteHeader(http.StatusServiceUnavailable)

			return
		}

		response.Header().Set("Content-Type", "application/octet-stream")
		_, _ = response.Write([]byte("unrecognizable media"))
	}))
	t.Cleanup(mediaServer.Close)

	for _, failureCase := range []struct {
		sourcePath string
		category   error
	}{
		{"/unavailable", errs.ErrTransportStatus},
		{"/unrecognized.mp4", errs.ErrInputMediaMIME},
	} {
		source := mediaServer.URL + failureCase.sourcePath

		resolvedInputs, err := ResolveInputMediaTypes(t.Context(), []media.Input{{URL: source}})
		if !errors.Is(err, errs.ErrInputMediaRead) || !errors.Is(err, failureCase.category) || (err != nil && !strings.Contains(err.Error(), source)) || len(resolvedInputs) != 0 {
			t.Errorf("✗ %s returned inputs %+v and error %v", source, resolvedInputs, err)
		}
	}

	canceledContext, cancel := context.WithCancel(t.Context())
	cancel()

	resolvedInputs, err := ResolveInputMediaTypes(canceledContext, []media.Input{{URL: mediaServer.URL}})
	if !errors.Is(err, errs.ErrCanceled) || !errors.Is(err, context.Canceled) || len(resolvedInputs) != 0 {
		t.Errorf("✗ canceled resolution returned inputs %+v and error %v", resolvedInputs, err)
	}

	if !t.Failed() {
		t.Log("✓ unavailable and unidentified remote inputs preserve classification, source, and cancellation")
	}
}

// TestRemoteMediaReadCauses verifies invariant #1: Remote media failures.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given an invalid URL, ResolveInputMediaTypes must return ErrInputMediaRead and
// ErrTransportCreate. A truncated body must preserve ErrInputMediaRead, ErrTransportRead, and
// io.ErrUnexpectedEOF. Cancellation during the request must return no inputs and preserve
// ErrCanceled and context.Canceled.
// Kind: permanent.
func TestRemoteMediaReadCauses(t *testing.T) {
	_, err := ResolveInputMediaTypes(t.Context(), []media.Input{{URL: "://invalid"}})
	if !errors.Is(err, errs.ErrInputMediaRead) || !errors.Is(err, errs.ErrTransportCreate) {
		t.Errorf("✗ malformed URL error = %v", err)
	}

	cancelContext, cancel := context.WithCancel(t.Context())
	defer cancel()

	mediaServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/octet-stream")

		if request.Method == http.MethodHead {
			return
		}

		if request.URL.Path == "/cancel" {
			cancel()

			return
		}

		response.Header().Set("Content-Length", "512")
		_, _ = response.Write([]byte("short"))
	}))
	t.Cleanup(mediaServer.Close)

	_, err = ResolveInputMediaTypes(t.Context(), []media.Input{{URL: mediaServer.URL}})
	if !errors.Is(err, errs.ErrInputMediaRead) || !errors.Is(err, errs.ErrTransportRead) || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("✗ truncated body error = %v", err)
	}

	resolvedInputs, err := ResolveInputMediaTypes(cancelContext, []media.Input{{URL: mediaServer.URL + "/cancel"}})
	if len(resolvedInputs) != 0 || !errors.Is(err, errs.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Errorf("✗ interrupted source returned %+v and error %v", resolvedInputs, err)
	}

	if !t.Failed() {
		t.Log("✓ source request and read failures retain classification and original causes")
	}
}

// TestRemoteMediaInspectionBound verifies invariant #2: Bounded media inspection.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given a server that sends a 512-byte PNG prefix then waits indefinitely, ResolveInputMediaTypes
// must finish successfully within the one-second context and return one record with the original
// URL, image/png, and no body bytes.
func TestRemoteMediaInspectionBound(t *testing.T) {
	mediaServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/octet-stream")

		if request.Method == http.MethodHead {
			return
		}

		prefix := make([]byte, 512)
		copy(prefix, "\x89PNG\r\n\x1a\n")
		_, _ = response.Write(prefix)
		response.(http.Flusher).Flush()
		<-request.Context().Done()
	}))
	t.Cleanup(mediaServer.Close)

	inspectionContext, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	resolvedInputs, err := ResolveInputMediaTypes(inspectionContext, []media.Input{{URL: mediaServer.URL}})
	if err != nil || len(resolvedInputs) != 1 {
		t.Errorf("✗ bounded inspection returned inputs %+v and error %v", resolvedInputs, err)
	} else if resolvedInputs[0].MIME != "image/png" || len(resolvedInputs[0].Bytes) != 0 || resolvedInputs[0].URL != mediaServer.URL {
		t.Errorf("✗ bounded inspection changed the URL record: %+v", resolvedInputs[0])
	}

	if !t.Failed() {
		t.Log("✓ media inspection finishes after the readable prefix without retaining body bytes")
	}
}

// TestDownloadPartialResults verifies invariant #3: Partial download results.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Test kind: permanent.
// What is being tested:
// Given a successful URL followed by a failed or malformed URL, DownloadInputMedia must return
// ErrInputMediaRead and all three input records in order. It must retain the first download bytes
// and source path, leave later sources unchanged, and preserve the caller's first record.
func TestDownloadPartialResults(t *testing.T) {
	mediaServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/failed" {
			response.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = response.Write([]byte("downloaded image"))
	}))
	t.Cleanup(mediaServer.Close)

	for _, failedURL := range []string{mediaServer.URL + "/failed", "://invalid"} {
		mediaInputs := []media.Input{{URL: mediaServer.URL}, {URL: failedURL}, {Filepath: "local.png", Bytes: []byte("local image")}}

		downloadedInputs, err := DownloadInputMedia(t.Context(), mediaInputs)
		if !errors.Is(err, errs.ErrInputMediaRead) || len(downloadedInputs) != len(mediaInputs) {
			t.Errorf("✗ partial download lost its inputs or error: %+v, %v", downloadedInputs, err)

			continue
		}

		if string(downloadedInputs[0].Bytes) != "downloaded image" || downloadedInputs[0].URL != "" || downloadedInputs[0].Filepath != mediaServer.URL {
			t.Errorf("✗ completed download was lost: %+v", downloadedInputs[0])
		}

		if downloadedInputs[1].URL != failedURL || downloadedInputs[2].Filepath != "local.png" || string(downloadedInputs[2].Bytes) != "local image" {
			t.Errorf("✗ untouched sources changed: %+v", downloadedInputs[1:])
		}

		if mediaInputs[0].URL != mediaServer.URL || mediaInputs[0].Filepath != "" || len(mediaInputs[0].Bytes) != 0 {
			t.Errorf("✗ download modified caller inputs: %+v", mediaInputs)
		}
	}

	if !t.Failed() {
		t.Log("✓ partial downloads preserve completed bytes, remaining sources, and caller ownership")
	}
}
