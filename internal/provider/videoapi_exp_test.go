package provider

// Invariants tested:
//  1. Video start errors: Given a missing job ID, an HTTP rejection, or a closed server,
//     submitVideo must return ErrResponseNoJobID, ErrResponseStatus, or ErrTransportRequest
//     respectively.
//  2. Video polling HTTP failures: When video polling returns HTTP 500 with a provider message,
//     submitVideo must return ErrResponseStatus containing that message.
//  3. Video reference conformance failure: Given an undecodable image that requires resizing,
//     submitVideo must return ErrInputMediaDecode and ErrInputMedia with the provider/model label
//     before sending any request.
//  4. Empty multipart video reference: Given an empty image for a multipart video request,
//     submitVideo must return ErrInputMediaEmpty without sending a request.
//  5. Large video streaming: Given a completed video larger than 64 MiB, submitVideo must return
//     one temporary-file artifact whose SHA-256 matches the complete served byte pattern.
//  6. Video failure paths: Given failed or unknown job statuses, submitVideo must return
//     ErrResponseGen or ErrResponseUnknown with the job ID and diagnostic.
//  7. Observation recovery: Given pending, temporary HTTP failure, and completed responses,
//     videoJobProbe.Poll through httpapi.Poll must complete without error after exactly three GET
//     requests for each of HTTP 429, 502, 503, and 504.
//  8. Required completed video URL: Given a completed response without the configured required
//     result URL, videoJobProbe.Poll must return false and ErrResponseNoData containing the job ID.
//  9. Permanent status with read failure: Given HTTP 400, 401, 403, 404, or 500 with a truncated
//     body, videoJobProbe.Poll through httpapi.Poll must stop after one request and return both
//     ErrResponseStatus and ErrTransportRead, without ErrTransportTimeout.
//  10. Video poll deadline: Given a job that stays working and PollTimeout set to one second,
//      submitVideo must return ErrTransportTimeout after at least one second and before five
//      seconds.
//  11. URL path traversal under arbitrary response bodies: For arbitrary response bytes and path
//      segments, walkURL must not panic and must return an empty string whenever the body is
//      invalid JSON.

import (
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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestVidStartErrs verifies invariant #1: Video start errors.
//
// What is being tested:
// Given a missing job ID, an HTTP rejection, or a closed server, submitVideo must return
// ErrResponseNoJobID, ErrResponseStatus, or ErrTransportRequest respectively. The response errors
// must name the provider/model pair, and the HTTP rejection must retain the provider message.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVidStartErrs(t *testing.T) {
	d := fixtureProvider(t, contentEndpointVideo)
	m := fixtureModel(t, d)
	label := pairLabel(d, m)

	t.Run("missing id", func(t *testing.T) {
		_, srv := recSrv(t, textOK(t, `{}`))
		if srv == nil {
			t.Fatal("💣 failed to start server")
		}

		_, err := vidGo(t, d, m, vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
		if !errors.Is(err, errs.ErrResponseNoJobID) || err == nil || !strings.Contains(err.Error(), label) {
			t.Errorf("✗ err = %v, want ErrResponseNoJobID naming %s", err, label)
		}

		if !t.Failed() {
			t.Log("✓ missing id")
		}
	})
	t.Run("non-2xx start", func(t *testing.T) {
		_, srv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"message":"denied by policy"}}`)
		})

		_, err := vidGo(t, d, m, vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
		if !errors.Is(err, errs.ErrResponseStatus) {
			t.Errorf("✗ err = %v, want ErrResponseStatus", err)
		}

		for _, want := range []string{label, "denied by policy"} {
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("✗ error %v missing %q", err, want)
			}
		}

		if !t.Failed() {
			t.Log("✓ non-2xx start")
		}
	})
	t.Run("transport failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		srv.Close()

		_, err := vidGo(t, d, m, vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
		if !errors.Is(err, errs.ErrTransportRequest) {
			t.Errorf("✗ err = %v, want the propagated ErrTransportRequest", err)
		}

		if !t.Failed() {
			t.Log("✓ transport failure")
		}
	})

	if !t.Failed() {
		t.Log("✓ the builder's id read fails clearly on missing ids, error statuses, and transport failures")
	}
}

// TestVidPoll5xx verifies invariant #2: Video polling HTTP failures.
//
// What is being tested:
// When video polling returns HTTP 500 with a provider message, submitVideo must return
// ErrResponseStatus containing that message.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVidPoll5xx(t *testing.T) {
	d := fixtureProvider(t, contentEndpointVideo)
	_, srv := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			fmt.Fprint(w, `{"id":"vid-p5"}`)

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"message":"backend down"}}`)
	})

	_, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
	if err == nil || !errors.Is(err, errs.ErrResponseStatus) || !strings.Contains(err.Error(), "backend down") {
		t.Errorf("✗ err = %v, want the labeled ErrResponseStatus carrying the provider message", err)
	}

	if !t.Failed() {
		t.Log("✓ a non-2xx poll is terminal with the provider API error")
	}
}

// TestVidRefsFail verifies invariant #3: Video reference conformance failure.
//
// What is being tested:
// Given an undecodable image that requires resizing, submitVideo must return ErrInputMediaDecode
// and ErrInputMedia with the provider/model label before sending any request.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVidRefsFail(t *testing.T) {
	d := fixtureProvider(t, contentEndpointVideo)
	recorder, srv := vidSrv(t, "id", "vid-rf", `{"status":"completed"}`, nil)
	m := fixtureModel(t, d)
	gp := params.Values{params.FlagTypeSize: "1280x720"}

	_, err := vidGo(t, d, m, vidAPI(t, d, srv.URL+"/videos"), gp, []media.Input{badRef(t)})
	if err == nil || !errors.Is(err, errs.ErrInputMediaDecode) || !errors.Is(err, errs.ErrInputMedia) || !strings.Contains(err.Error(), pairLabel(d, m)) {
		t.Errorf("✗ err = %v, want the labeled decode failure from the conformance step", err)
	}

	if got := recorder.count(); got != 0 {
		t.Errorf("✗ %d request(s) reached the harness; a conformance failure must precede any network use", got)
	}

	if !t.Failed() {
		t.Log("✓ a conformance failure surfaces as the labeled decode error before any network use")
	}
}

// TestVidFormEmpty verifies invariant #4: Empty multipart video reference.
//
// What is being tested:
// Given an empty image for a multipart video request, submitVideo must return ErrInputMediaEmpty
// without sending a request.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVidFormEmpty(t *testing.T) {
	recorder, srv := recSrv(t, textOK(t, `{"id":"never"}`))
	api := boundsAPI(t, srv.URL+"/v", 60)
	api.InputMediaPayloadType, api.InputMediaStyle, api.InputMediaProvParam = "form", "parts", "ref"
	d := fixtureProvider(t, contentEndpointVideo)
	empty := []media.Input{{MIME: "image/png", Filepath: "/in/empty.png"}}

	_, err := vidGo(t, d, fixtureModel(t, d), api, params.Values{}, empty)
	if err == nil || !errors.Is(err, errs.ErrInputMediaEmpty) {
		t.Errorf("✗ err = %v, want the empty-image error before any request is sent", err)
	}

	if got := recorder.count(); got != 0 {
		t.Errorf("✗ %d request(s) reached the harness; the assembly failure must precede any request", got)
	}

	if !t.Failed() {
		t.Log("✓ an empty reference fails the form assembly cleanly before any request")
	}
}

// TestVidBigStream verifies invariant #5: Large video streaming.
//
// What is being tested:
// Given a completed video larger than 64 MiB, submitVideo must return one temporary-file artifact
// whose SHA-256 matches the complete served byte pattern.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVidBigStream(t *testing.T) {
	const n = apiBound + 4096

	d := fixtureProvider(t, contentEndpointVideo)
	_, srv := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			fmt.Fprint(w, `{"id":"vid-big"}`)
		case strings.HasSuffix(r.URL.Path, "/content"):
			if err := patWrite(t, w, n); err != nil {
				t.Errorf("✗ harness body write failed: %v", err)
			}
		default:
			fmt.Fprint(w, `{"status":"completed"}`)
		}
	})

	artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
	if err != nil {
		t.Fatalf("💣 submit failed: %v", err)
	}

	if len(artifacts) != 1 || artifacts[0].TmpPath == "" {
		t.Fatalf("💣 artifacts = %+v, want one file-backed artifact", artifacts)
	}
	defer rmArtifact(t, artifacts[0])

	f, err := os.Open(artifacts[0].TmpPath)
	if err != nil {
		t.Fatalf("💣 open the streamed file: %v", err)
	}

	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("💣 hash the streamed file: %v", err)
	}

	if got := hex.EncodeToString(h.Sum(nil)); got != patSum(t, n) {
		t.Errorf("✗ streamed video hash differs from the served body — the over-bound download must stream byte-identically")
	}

	if !t.Failed() {
		t.Log("✓ a video larger than the API read bound streams byte-identically to its temp file")
	}
}

// TestVidFailurePaths verifies invariant #6: Video failure paths.
//
// What is being tested:
// Given failed or unknown job statuses, submitVideo must return ErrResponseGen or
// ErrResponseUnknown with the job ID and diagnostic. If the video download loses its connection, it
// must return ErrTransportDownload and leave no new download file.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVidFailurePaths(t *testing.T) {
	vidFail(t, "vid-f1", `{"status":"failed","error":{"message":"moderation blocked"}}`,
		errs.ErrResponseGen, "moderation blocked", "vid-f1")
	vidFail(t, "vid-f2", `{"status":"weird"}`, errs.ErrResponseUnknown, "vid-f2", "weird")

	t.Run("mid-download drop leaves no temp file", func(t *testing.T) {
		t.Setenv("TMPDIR", t.TempDir())

		before := dlSet(t)
		d := fixtureProvider(t, contentEndpointVideo)
		_, srv := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				fmt.Fprint(w, `{"id":"vid-f3"}`)
			case strings.HasSuffix(r.URL.Path, "/content"):
				w.Header().Set("Content-Length", "4096")
				_, _ = w.Write([]byte("partial"))
				w.(http.Flusher).Flush()
				panic(http.ErrAbortHandler)
			default:
				fmt.Fprint(w, `{"status":"completed"}`)
			}
		})

		_, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
		if err == nil || !errors.Is(err, errs.ErrTransportDownload) {
			t.Errorf("✗ err = %v, want a wrapped ErrTransportDownload", err)
		}

		assertNoNewDl(t, before, "the dropped video download")

		if !t.Failed() {
			t.Log("✓ mid-download drop leaves no temp file")
		}
	})

	if !t.Failed() {
		t.Log("✓ builder failure paths name the job and diagnostic, and a dropped download cleans up")
	}
}

// TestSharedVideoPollRecovery verifies invariant #7: Observation recovery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given pending, temporary HTTP failure, and completed responses, videoJobProbe.Poll through
// httpapi.Poll must complete without error after exactly three GET requests for each of HTTP 429,
// 502, 503, and 504.
func TestSharedVideoPollRecovery(t *testing.T) {
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
					_, _ = writer.Write([]byte(`{"status":"pending"}`))
				case 2:
					writer.WriteHeader(statusCode)
					_, _ = writer.Write([]byte(`{"message":"temporary observation failure"}`))
				default:
					_, _ = writer.Write([]byte(`{"status":"done","video":{"url":"https://result.example/video.mp4"}}`))
				}
			}))
			defer server.Close()

			poller := &videoJobProbe{api: &catalog.VideoAPI{AsyncJobsURL: server.URL, ProgressStatusText: []string{"pending"}, CompletedStatusText: "done", URLPathSeq: []string{"video", "url"}, URLPathRequired: true}, jobID: "shared-job"}

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

// TestCompletedVideoRequiresURL verifies invariant #8: Required completed video URL.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given a completed response without the configured required result URL, videoJobProbe.Poll must
// return false and ErrResponseNoData containing the job ID.
func TestCompletedVideoRequiresURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"status":"done"}`))
	}))
	defer server.Close()

	poller := &videoJobProbe{api: &catalog.VideoAPI{AsyncJobsURL: server.URL, CompletedStatusText: "done", URLPathSeq: []string{"video", "url"}, URLPathRequired: true}, jobID: "empty-result-job"}

	complete, err := poller.Poll(t.Context())
	if complete || !errors.Is(err, errs.ErrResponseNoData) || !strings.Contains(err.Error(), "empty-result-job") {
		t.Errorf("✗ complete=%t error=%v; want missing-data failure naming the job", complete, err)
	}

	if !t.Failed() {
		t.Log("✓ completed status requires the declared video URL")
	}
}

// TestPermanentStatusReadFailure verifies invariant #9: Permanent status with read failure.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given HTTP 400, 401, 403, 404, or 500 with a truncated body, videoJobProbe.Poll through
// httpapi.Poll must stop after one request and return both ErrResponseStatus and ErrTransportRead,
// without ErrTransportTimeout.
func TestPermanentStatusReadFailure(t *testing.T) {
	for _, statusCode := range []int{400, 401, 403, 404, 500} {
		t.Run(strconv.Itoa(statusCode), func(t *testing.T) {
			observations := 0

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				observations++

				writer.Header().Set("Content-Length", "100")
				writer.WriteHeader(statusCode)
				_, _ = writer.Write([]byte(`{"error":`))
			}))
			defer server.Close()

			poller := &videoJobProbe{api: &catalog.VideoAPI{AsyncJobsURL: server.URL, CompletedStatusText: "done"}, jobID: "permanent-job"}

			err := httpapi.Poll(t.Context(), time.Millisecond, 25*time.Millisecond, poller)
			if !errors.Is(err, errs.ErrResponseStatus) || !errors.Is(err, errs.ErrTransportRead) || errors.Is(err, errs.ErrTransportTimeout) || observations != 1 {
				t.Errorf("✗ status=%d error=%v observations=%d; want one permanent observation preserving status and read causes", statusCode, err, observations)
			}

			if !t.Failed() {
				t.Log("✓ permanent HTTP status remains terminal when body reading fails")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ authentication and other permanent statuses are never retried")
	}
}

// TestVidPollDeadline verifies invariant #10: Video poll deadline.
//
// What is being tested:
// Given a job that stays working and PollTimeout set to one second, submitVideo must return
// ErrTransportTimeout after at least one second and before five seconds.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVidPollDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			fmt.Fprint(w, `{"id":"job-dead"}`)

			return
		}

		fmt.Fprint(w, `{"status":"working"}`)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	d := fixtureProvider(t, contentEndpointVideo)
	api := boundsAPI(t, srv.URL+"/v", 1)
	run := imgReq(t, d, fixtureModel(t, d), "a prompt", params.Values{}, nil)
	start := time.Now()
	_, err := submitVideo(ctx, &api, "k", &run)
	elapsed := time.Since(start)

	if err == nil || !errors.Is(err, errs.ErrTransportTimeout) {
		t.Errorf("✗ err = %v, want a wrapped ErrTransportTimeout at the 1s deadline", err)
	}

	if elapsed < time.Second || elapsed >= 5*time.Second {
		t.Errorf("✗ deadline observed at %v, want within [1s, 5s) for PollTimeout 1", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ PollTimeout 1 deadlines at an observed time within [1s, 5s)")
	}
}

// FuzzURLPath verifies invariant #11: URL path traversal under arbitrary response bodies.
//
// What is being tested:
// For arbitrary response bytes and path segments, walkURL must not panic and must return an empty
// string whenever the body is invalid JSON.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzURLPath(f *testing.F) {
	f.Add([]byte(`{"video":{"url":"u"}}`), "video/url")
	f.Add([]byte(`{"unsigned_urls":["a"]}`), "unsigned_urls/0")
	f.Add([]byte(`not json`), "a/b")
	f.Add([]byte(`{}`), "")
	f.Add([]byte(`[[["deep"]]]`), "0/0/0")
	f.Fuzz(func(t *testing.T, body []byte, path string) {
		got := walkURL(body, strings.Split(path, "/"))
		// Invalid JSON must not produce a URL.
		if !json.Valid(body) && got != "" {
			t.Errorf("✗ walkURL over an undecodable body = %q, want the empty miss", got)
		}

		if !t.Failed() {
			t.Logf("✓ the traversal classified without inventing a value")
		}
	})
}

// apiBound is the 64 MiB API-response bound the bounded in-memory reads must enforce.
const (
	apiBound = 64 << 20
	pngExt   = ".png"
)

// patWrite streams n deterministic pattern bytes to w.
func patWrite(test testing.TB, w io.Writer, n int) error {
	test.Helper()

	buf := make([]byte, 64<<10)
	for i := range buf {
		buf[i] = byte(i*7 + 13)
	}

	for n > 0 {
		chunk := min(n, len(buf))
		if _, err := w.Write(buf[:chunk]); err != nil {
			return err
		}

		n -= chunk
	}

	return nil
}

// patSum returns the SHA-256 of n pattern bytes without downloading them.
func patSum(test testing.TB, n int) string {
	test.Helper()

	h := sha256.New()
	_ = patWrite(test, h, n)

	return hex.EncodeToString(h.Sum(nil))
}

// vidFail drives one content-endpoint video submit against pollBody and asserts the classified
// sentinel with every diagnostic fragment present.
func vidFail(t *testing.T, id, pollBody string, sentinel error, wants ...string) {
	t.Helper()
	d := fixtureProvider(t, contentEndpointVideo)
	_, srv := vidSrv(t, "id", id, pollBody, nil)

	_, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
	if err == nil || !errors.Is(err, sentinel) {
		t.Errorf("✗ err = %v, want a wrapped %v", err, sentinel)
	}

	for _, want := range wants {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("✗ error %v missing %q", err, want)
		}
	}
}

// badRef returns undecodable bytes labeled as a PNG reference.
func badRef(test testing.TB) media.Input {
	test.Helper()

	return media.Input{Bytes: []byte("not an image"), MIME: "image/png", Filepath: "/in/bad.png"}
}
