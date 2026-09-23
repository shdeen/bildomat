package httpapi

// Invariants tested:
//  1. Large artifact streaming: Given a response larger than the API response bound, Fetch must
//     stream every byte to a temporary file, choose .png from Content-Type, and return no inline
//     payload.
//  2. Redirect limit: Given a redirect loop, Fetch must return both ErrTransportRequest and
//     ErrTransportRedirectLimit and leave no new bild-dl-* file.
//  3. Failed artifact fetch cleanup: Given HTTP 403, an interrupted body, or an unreachable
//     endpoint, Fetch must return the corresponding status, download, or request error.
//  4. Download setup failures: Given an invalid URL, Fetch must return ErrTransportCreate.
//  5. Fetched artifact extension selection: For the listed responses, Fetch must choose the
//     extension from a usable declared media type, then detected bytes, then the supplied fallback.
//  6. Retained downloads: Given a complete download, Fetch and Record.Save must preserve its bytes
//     by referencing the matching artifact file, retain the provider URL, and avoid a duplicate
//     body file.
//  7. Empty download rejection: Given an empty HTTP 200 body, Fetch must return ErrResponseNoData
//     without artifact bytes, a temporary path, or a remaining download file.
//  8. Failed download response causes: Given HTTP 503 with a truncated body, Fetch must return an
//     error matching both ErrTransportStatus and io.ErrUnexpectedEOF and return no temporary
//     artifact path.
//  9. Failed response excerpts: Given HTTP 503 with a 1024-byte body, Fetch must return
//     ErrTransportStatus without ErrTransportSize and return no temporary artifact path.
//  10. Download-body cancellation: When the caller cancels after Fetch creates its download file,
//      Fetch must return within five seconds with ErrCanceled and context.Canceled.
//  11. Partial download byte accounting: Given a reader that returns bytes with
//      io.ErrUnexpectedEOF, copyDownload must write and count every received byte, including after
//      an 8192-byte prefix, and preserve the read error.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/metadata"
)

// TestFetchStreams verifies invariant #1: Large artifact streaming.
//
// What is being tested:
// Given a response larger than the API response bound, Fetch must stream every byte to a temporary
// file, choose .png from Content-Type, and return no inline payload. The file length and SHA-256
// must match the served pattern.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFetchStreams(t *testing.T) {
	const size = apiBound + 64<<10

	_, srv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_ = patWrite(t, w, size)
	})
	generatedMedia, err := Fetch(t.Context(), srv.URL, AuthCredential{}, ".bin", nil)
	t.Cleanup(func() { rmArtifact(t, generatedMedia) })

	if err != nil {
		t.Fatalf("💣 fetch failed, nothing to inspect: %v", err)
	}

	checkCarriage(t, "streamed", generatedMedia)

	if generatedMedia.TmpPath == "" {
		t.Fatalf("💣 no SrcPath on the streamed artifact — nothing to inspect")
	}

	if generatedMedia.FileExt != pngExt {
		t.Errorf("✗ FileExt = %q, want .png from the served Content-Type", generatedMedia.FileExt)
	}

	fi, statErr := os.Stat(generatedMedia.TmpPath)
	switch {
	case statErr != nil:
		t.Errorf("✗ stat temp file: %v", statErr)
	case fi.Size() != int64(size):
		t.Errorf("✗ temp file size = %d, want the full %d served bytes", fi.Size(), size)
	}

	f, err := os.Open(generatedMedia.TmpPath)
	if err != nil {
		t.Fatalf("💣 open temp file: %v", err)
	}

	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("💣 hash temp file: %v", err)
	}

	if got := hex.EncodeToString(h.Sum(nil)); got != patSum(t, size) {
		t.Errorf("✗ streamed bytes differ from the served body (hash mismatch)")
	}

	if !t.Failed() {
		t.Log("✓ an over-API-bound body streams intact to a temp file with the served extension")
	}
}

// TestRedirectCap verifies invariant #2: Redirect limit.
//
// What is being tested:
// Given a redirect loop, Fetch must return both ErrTransportRequest and ErrTransportRedirectLimit
// and leave no new bild-dl-* file.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRedirectCap(t *testing.T) {
	before := dlSet(t)
	_, srv := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	})

	_, err := Fetch(t.Context(), srv.URL+"/loop", AuthCredential{}, ".bin", nil)
	if err == nil || !errors.Is(err, errs.ErrTransportRequest) {
		t.Errorf("✗ Fetch(redirect loop) = %v, want a classified ErrTransportRequest", err)
	}

	if !errors.Is(err, errs.ErrTransportRedirectLimit) {
		t.Errorf("✗ Fetch(redirect loop) = %v, want the redirect-limit sentinel", err)
	}

	assertNoNewDl(t, before, "the redirect loop")

	if !t.Failed() {
		t.Log("✓ a redirect loop terminates as a classified transport failure with nothing left behind")
	}
}

// TestFetchFailClean verifies invariant #3: Failed artifact fetch cleanup.
//
// What is being tested:
// Given HTTP 403, an interrupted body, or an unreachable endpoint, Fetch must return the
// corresponding status, download, or request error. It must return no artifact bytes or temporary
// path and leave no new bild-dl-* file.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFetchFailClean(t *testing.T) {
	_, deny := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "denied")
	})
	_, drop := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		fmt.Fprint(w, "partial")
		// Flush the headers and partial body so the client is mid-stream — not pre-response
		// — when the connection aborts.
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	})

	cases := []struct {
		name string
		url  string
		want error
	}{
		{"non-2xx", deny.URL, errs.ErrTransportStatus},
		{"mid-body drop", drop.URL, errs.ErrTransportDownload},
		{"unreachable host", "http://127.0.0.1:1/x", errs.ErrTransportRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := dlSet(t)

			generatedMedia, err := Fetch(t.Context(), c.url, AuthCredential{}, ".bin", nil)
			if !errors.Is(err, c.want) {
				t.Errorf("✗ err = %v, want %v", err, c.want)
			}

			if generatedMedia.TmpPath != "" || len(generatedMedia.Data) > 0 {
				t.Errorf("✗ failed fetch returned an artifact: %+v", generatedMedia)
			}

			assertNoNewDl(t, before, c.name)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ failed fetches classify under their sentinels and leave no bild-dl-* files")
	}
}

// TestFetchSetupFails verifies invariant #4: Download setup failures.
//
// What is being tested:
// Given an invalid URL, Fetch must return ErrTransportCreate. Given an unavailable temporary
// directory, it must return ErrTransportDownload. Neither attempt may leave a new download file in
// the inspected temporary directory.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFetchSetupFails(t *testing.T) {
	before := dlSet(t)
	if _, err := Fetch(t.Context(), "http://\x7f bad", AuthCredential{}, ".bin", nil); !errors.Is(err, errs.ErrTransportCreate) {
		t.Errorf("✗ Fetch(bad url) = %v, want ErrTransportCreate", err)
	}

	_, srv := recSrv(t, textOK(t, "DATA"))
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing-subdir"))

	if _, err := Fetch(t.Context(), srv.URL, AuthCredential{}, ".bin", nil); !errors.Is(err, errs.ErrTransportDownload) {
		t.Errorf("✗ Fetch(no temp dir) = %v, want ErrTransportDownload", err)
	}

	assertNoNewDl(t, before, "setup failures")

	if !t.Failed() {
		t.Log("✓ create and temp-file failures classify and leave nothing behind")
	}
}

// TestFetchExtChain verifies invariant #5: Fetched artifact extension selection.
//
// What is being tested:
// For the listed responses, Fetch must choose the extension from a usable declared media type, then
// detected bytes, then the supplied fallback. It must normalize MIME parameters, map SVG and video
// types, and discard extra slash segments. Each result must have exactly one payload form.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFetchExtChain(t *testing.T) {
	cases := []struct {
		name  string
		ctype string // "" = no Content-Type header served
		body  []byte
		fall  string
		want  string
	}{
		{"declared wins over magic", "image/png", jpegStub(t), ".bin", pngExt},
		{"declared with parameters", "image/jpeg; charset=binary", jpegStub(t), ".bin", ".jpg"},
		{"absent falls to the sniff", "", jpegStub(t), ".bin", ".jpg"},
		{"unusable falls to the sniff", "application/octet-stream", jpegStub(t), ".bin", ".jpg"},
		{"unidentifiable falls to the declared fallback", "", []byte("plain text payload"), ".webp", ".webp"},
		{"svg maps to .svg", "image/svg+xml", svgStub(t), pngExt, ".svg"},
		{"a declared video type wins", "video/webm", []byte("plain text payload"), ".mp4", ".webm"},
		{"a parameterized video type normalizes", "video/mp4; codecs=avc1", []byte("plain text payload"), ".bin", ".mp4"},
		{"a slash-smuggling subtype takes the leading token", "image/a/b", jpegStub(t), ".bin", ".a"},
		{"a video slash-smuggling subtype takes the leading token", "video/a/b", []byte("plain text payload"), ".mp4", ".a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, srv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
				if c.ctype == "" {
					w.Header()["Content-Type"] = nil
				} else {
					w.Header().Set("Content-Type", c.ctype)
				}

				_, _ = w.Write(c.body)
			})
			generatedMedia, err := Fetch(t.Context(), srv.URL, AuthCredential{}, c.fall, nil)
			rmArtifact(t, generatedMedia)

			if err != nil {
				t.Errorf("✗ fetch failed: %v", err)

				return
			}

			if generatedMedia.FileExt != c.want {
				t.Errorf("✗ FileExt = %q, want %q", generatedMedia.FileExt, c.want)
			}

			checkCarriage(t, c.name, generatedMedia)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ Fetch extensions follow declared → sniff → declared fallback, .svg included")
	}
}

// TestFetchRecord verifies invariant #6: Retained downloads.
//
// What is being tested:
// Given a complete download, Fetch and Record.Save must preserve its bytes by referencing the
// matching artifact file, retain the provider URL, and avoid a duplicate body file. Given an
// interrupted download, they must return ErrTransportDownload with no artifact path, save the
// received bytes separately, and mark the response incomplete.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestFetchRecord(t *testing.T) {
	content := pngStub(t)
	for _, interrupted := range []bool{false, true} {
		t.Run(strconv.FormatBool(interrupted), func(t *testing.T) {
			_, server := recSrv(t, func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "image/png")

				if interrupted {
					writer.Header().Set("Content-Length", strconv.Itoa(len(content)+100))
				}

				if _, err := writer.Write(content); err != nil {
					t.Errorf("✗ harness write: %v", err)
				}
			})
			record := metadata.New("provider", "model", "test", "boat", time.Now())
			downloaded, downloadErr := Fetch(t.Context(), server.URL, AuthCredential{}, ".png", record)

			var saved []artifact.SavedFile

			if interrupted {
				if !errors.Is(downloadErr, errs.ErrTransportDownload) || downloaded.TmpPath != "" {
					t.Errorf("✗ failed download outcome: %+v %v", downloaded, downloadErr)
				}
			} else {
				if downloadErr != nil {
					t.Errorf("✗ download: %v", downloadErr)
				}

				t.Cleanup(func() { rmArtifact(t, downloaded) })

				saved = []artifact.SavedFile{{Path: downloaded.TmpPath, Bytes: int64(len(content))}}
			}

			directory := t.TempDir()

			_, saveErr := record.Save(directory, "boat", saved, downloadErr)
			if saveErr != nil {
				t.Errorf("✗ saving captured body: %v", saveErr)
			}

			if len(record.Responses) != 1 {
				t.Errorf("✗ missing downloaded response: %+v", record.Responses)

				return
			}

			response := record.Responses[0]

			var retained struct {
				File string `json:"file"`
			}
			if err := json.Unmarshal(response.Body, &retained); err != nil {
				t.Errorf("✗ binary envelope: %v", err)
			}

			data, readErr := os.ReadFile(retained.File)
			if readErr != nil || !bytes.Equal(data, content) {
				t.Errorf("✗ retained body: %x %v", data, readErr)
			}

			if response.Incomplete != interrupted || response.Type != metadata.Synchronous || response.ContentType != "image/png" {
				t.Errorf("✗ binary response details: %+v", response)
			}

			if !interrupted && retained.File != downloaded.TmpPath {
				t.Error("✗ identical artifact was duplicated")
			}

			if !interrupted {
				if len(record.Artifacts) != 1 || len(record.Artifacts[0].References) != 1 || record.Artifacts[0].References[0] != server.URL {
					t.Errorf("✗ artifact lacks its provider reference: %+v", record.Artifacts)
				}
			}

			files, readErr := os.ReadDir(directory)

			count := 1
			if interrupted {
				count = 2
			}

			if readErr != nil || len(files) != count {
				t.Errorf("✗ retained file count: %d %v", len(files), readErr)
			}

			if !t.Failed() {
				t.Log("✓ downloaded data survives with truthful references and no duplicate artifact")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ complete and interrupted download records preserve received bytes")
	}
}

// TestEmptyDownloadRejected verifies invariant #7: Empty download rejection.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given an empty HTTP 200 body, Fetch must return ErrResponseNoData without artifact bytes, a
// temporary path, or a remaining download file. With recording enabled, it must also retain the
// HTTP 200 response.
// Kind: permanent.
func TestEmptyDownloadRejected(t *testing.T) {
	for _, persist := range []bool{false, true} {
		t.Run(strconv.FormatBool(persist), func(t *testing.T) {
			temporaryDirectory := t.TempDir()
			t.Setenv("TMPDIR", temporaryDirectory)

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.Header().Set("Content-Type", "image/png")
				response.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)

			var record *metadata.Record
			if persist {
				record = metadata.New("provider", "model", "test", "boat", time.Now())
			}

			generated, err := Fetch(context.Background(), server.URL, AuthCredential{}, ".png", record)
			if !errors.Is(err, errs.ErrResponseNoData) || generated.TmpPath != "" || len(generated.Data) != 0 {
				t.Errorf("✗ empty download produced artifact %+v or lost no-data classification: %v", generated, err)
			}

			temporaryFiles, globErr := filepath.Glob(filepath.Join(temporaryDirectory, "bild-dl-*"))
			if globErr != nil || len(temporaryFiles) != 0 {
				t.Errorf("✗ empty download retained temporary files %v: %v", temporaryFiles, globErr)
			}

			if record != nil && (len(record.Responses) != 1 || record.Responses[0].Status != http.StatusOK) {
				t.Errorf("✗ empty response missing from transaction: %+v", record.Responses)
			}

			if !t.Failed() {
				t.Log("✓ empty download rejected without an owned file")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ empty downloads fail independently of persistence")
	}
}

// TestDownloadStatusReadCause verifies invariant #8: Failed download response causes.
//
// What is being tested:
// Given HTTP 503 with a truncated body, Fetch must return an error matching both ErrTransportStatus
// and io.ErrUnexpectedEOF and return no temporary artifact path.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestDownloadStatusReadCause(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Length", "100")
		response.WriteHeader(http.StatusServiceUnavailable)

		if _, err := response.Write([]byte("partial response")); err != nil {
			t.Errorf("✗ response fixture write: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	generated, err := Fetch(context.Background(), server.URL, AuthCredential{}, ".png", nil)
	if !errors.Is(err, errs.ErrTransportStatus) || !errors.Is(err, io.ErrUnexpectedEOF) || generated.TmpPath != "" {
		t.Errorf("✗ status/read causes or absent-artifact result lost: %+v, %v", generated, err)
	}

	if !t.Failed() {
		t.Log("✓ download status and read causes remain discoverable together")
	}
}

// TestDownloadStatusExcerpt verifies invariant #9: Failed response excerpts.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given HTTP 503 with a 1024-byte body, Fetch must return ErrTransportStatus without
// ErrTransportSize and return no temporary artifact path.
// Kind: permanent.
func TestDownloadStatusExcerpt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)

		if _, err := response.Write(bytes.Repeat([]byte("x"), 1024)); err != nil {
			t.Errorf("✗ response fixture write: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	generated, err := Fetch(t.Context(), server.URL, AuthCredential{}, ".png", nil)
	if !errors.Is(err, errs.ErrTransportStatus) || errors.Is(err, errs.ErrTransportSize) || generated.TmpPath != "" {
		t.Errorf("✗ diagnostic excerpt changed the failure classification: %+v, %v", generated, err)
	}

	if !t.Failed() {
		t.Log("✓ diagnostic excerpt length does not classify an ordinary failed response as oversized")
	}
}

// TestDownloadBodyCancellation verifies invariant #10: Download-body cancellation.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When the caller cancels after Fetch creates its download file, Fetch must return within five
// seconds with ErrCanceled and context.Canceled. It must return no artifact bytes or path and
// remove the download file.
// Kind: permanent.
func TestDownloadBodyCancellation(t *testing.T) {
	temporaryDirectory := t.TempDir()
	t.Setenv("TMPDIR", temporaryDirectory)

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Length", "100")

		if _, err := response.Write([]byte("x")); err != nil {
			t.Errorf("✗ response fixture write: %v", err)
		}

		response.(http.Flusher).Flush()
		<-request.Context().Done()
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	failures := make(chan error, 1)
	go finishCanceledDownload(t, ctx, server.URL, failures)

	deadline := time.Now().Add(5 * time.Second)

	for {
		temporaryFiles, err := filepath.Glob(filepath.Join(temporaryDirectory, "bild-dl-*"))
		if err != nil {
			t.Fatalf("💣 inspect download destination: %v", err)
		}

		if len(temporaryFiles) == 1 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("💣 HTTP fixture did not reach streaming persistence")
		}

		time.Sleep(time.Millisecond)
	}

	cancel()

	select {
	case err := <-failures:
		if !errors.Is(err, errs.ErrCanceled) || !errors.Is(err, context.Canceled) {
			t.Errorf("✗ in-body cancellation lost its classification or cause: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("💣 canceled download did not return")
	}

	temporaryFiles, globErr := filepath.Glob(filepath.Join(temporaryDirectory, "bild-dl-*"))
	if globErr != nil || len(temporaryFiles) != 0 {
		t.Errorf("✗ canceled download retained temporary files %v: %v", temporaryFiles, globErr)
	}

	if !t.Failed() {
		t.Log("✓ cancellation during streaming retains its cause and cleans the destination")
	}
}

// TestCopyDownloadFailures verifies invariant #11: Partial download byte accounting.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a reader that returns bytes with io.ErrUnexpectedEOF, copyDownload must write and count
// every received byte, including after an 8192-byte prefix, and preserve the read error. With a
// closed destination, it must report zero bytes written and preserve both io.ErrUnexpectedEOF and
// os.ErrClosed.
// Kind: permanent.
func TestCopyDownloadFailures(t *testing.T) {
	for _, prefixLength := range []int{0, 8192} {
		t.Run(strconv.Itoa(prefixLength), func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "download-")
			if err != nil {
				t.Fatalf("💣 download destination: %v", err)
			}

			prefix := bytes.Repeat([]byte("x"), prefixLength)

			_, written, copyErr := copyDownload(file, io.MultiReader(bytes.NewReader(prefix), prefixFaultReader{test: t}))
			if err := file.Close(); err != nil {
				t.Fatalf("💣 close test destination: %v", err)
			}

			expectedBytes := append(prefix, []byte("abcdef")...)
			// #nosec G304 -- this file belongs to the test temporary directory.
			persisted, readErr := os.ReadFile(file.Name())
			if readErr != nil {
				t.Fatalf("💣 inspect test destination: %v", readErr)
			}

			if !errors.Is(copyErr, io.ErrUnexpectedEOF) || written != int64(len(expectedBytes)) || !bytes.Equal(persisted, expectedBytes) {
				t.Errorf("✗ partial download: written=%d persisted=%d error=%v", written, len(persisted), copyErr)
			}

			if !t.Failed() {
				t.Log("✓ interrupted reads preserve exact received bytes, count, and cause")
			}
		})
	}

	file, err := os.CreateTemp(t.TempDir(), "closed-download-")
	if err != nil {
		t.Fatalf("💣 closed destination: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("💣 close test destination: %v", err)
	}

	_, written, copyErr := copyDownload(file, prefixFaultReader{test: t})
	if written != 0 || !errors.Is(copyErr, io.ErrUnexpectedEOF) || !errors.Is(copyErr, os.ErrClosed) {
		t.Errorf("✗ combined read/write causes: bytes=%d error=%v", written, copyErr)
	}

	if !t.Failed() {
		t.Log("✓ read and write failures preserve byte accounting and both original causes")
	}
}

// checkCarriage checks that an artifact uses exactly one payload form and has an extension.
func checkCarriage(t *testing.T, tag string, generatedMedia artifact.Media) {
	t.Helper()

	if (len(generatedMedia.Data) > 0) == (generatedMedia.TmpPath != "") {
		t.Errorf("✗ %s: artifact carriage not exclusive: %d inline bytes, SrcPath %q", tag, len(generatedMedia.Data), generatedMedia.TmpPath)
	}

	if generatedMedia.FileExt == "" {
		t.Errorf("✗ %s: artifact has no extension", tag)
	}
}

// patSum is the SHA-256 of n pattern bytes, computed independently of any download.
func patSum(test testing.TB, n int) string {
	test.Helper()

	h := sha256.New()
	_ = patWrite(test, h, n)

	return hex.EncodeToString(h.Sum(nil))
}

// pngStub is the 8-byte PNG signature — enough magic for content-type detection.
func pngStub(test testing.TB) []byte {
	test.Helper()

	return []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
}

// dlSet snapshots the bild-dl-* temp files currently present.
func dlSet(t *testing.T) map[string]bool {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "bild-dl-*"))
	if err != nil {
		t.Fatalf("💣 temp-dir glob failed: %v", err)
	}

	set := make(map[string]bool, len(matches))
	for _, m := range matches {
		set[m] = true
	}

	return set
}

// assertNoNewDl asserts no bild-dl-* temp file appeared since the before snapshot.
func assertNoNewDl(t *testing.T, before map[string]bool, label string) {
	t.Helper()

	for f := range dlSet(t) {
		if !before[f] {
			t.Errorf("✗ %s left a temp file behind: %s", label, f)
		}
	}
}

// finishCanceledDownload observes the result of the independently canceled request.
func finishCanceledDownload(t *testing.T, ctx context.Context, endpoint string, failures chan<- error) {
	t.Helper()

	generated, err := Fetch(ctx, endpoint, AuthCredential{}, ".png", nil)
	if generated.TmpPath != "" || len(generated.Data) != 0 {
		t.Errorf("✗ canceled download transferred an artifact: %+v", generated)
	}

	failures <- err
}
