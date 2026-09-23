package httpapi

// Invariants tested:
//  1. Transport error classification: Given an unsupported JSON value, PostJSON must return
//     ErrTransportMarshal.
//  2. Redirect credential scope: When GetAuth or Fetch follows redirects, it must send credentials
//     at the original origin and omit them at a different origin.
//  3. Mid-request cancellation: When the caller cancels an in-flight PostJSON request, PostJSON
//     must return within two seconds with context.Canceled and ErrCanceled, without ErrTransport.
//  4. Pre-canceled transport: Given a context canceled before the request, PostJSON must return
//     within one second with context.Canceled and ErrCanceled.
//  5. Transport deadline classification: Given a 50-millisecond deadline and a server that waits
//     two seconds, PostJSON must return ErrTransportRequest without ErrCanceled.
//  6. API response size bound: Given a response 1024 bytes larger than the 64 MiB API bound,
//     PostJSON and GetAuth must each return ErrTransportSize.
//  7. Incomplete response capture: Given a truncated JSON response, GetAuth must return
//     ErrTransportRead and a nil body while recording the received prefix as incomplete with
//     capture and decode errors.
//  8. Bounded read behavior: Given bodies below or exactly at a four-byte limit, readFirstBytes
//     must return their complete bytes without an error.
//  9. Partial response reads: Given a reader returning abcdef with io.ErrUnexpectedEOF and a
//     four-byte limit, readFirstBytes must return abcd with ErrTransportRead and the original
//     cause.
//  10. Bounded reads with nonpositive limits: Given a zero limit, readFirstBytes must accept an
//      empty body and reject a nonempty body with ErrTransportSize.
//  11. Bounded reads under arbitrary input: For arbitrary bytes and unsigned 16-bit limits,
//      readFirstBytes must return all original bytes without an error when the body fits, and
//      return ErrTransportSize when it exceeds the limit.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/metadata"
)

// TestTransportErrs verifies invariant #1: Transport error classification.
//
// What is being tested:
// Given an unsupported JSON value, PostJSON must return ErrTransportMarshal. Given malformed URLs,
// SendBody and GetAuth must return ErrTransportCreate. Given an unreachable endpoint, PostJSON must
// return ErrTransportRequest.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestTransportErrs(t *testing.T) {
	_, srv := recSrv(t, textOK(t, `{"ok":true}`))
	if _, _, err := PostJSON(t.Context(), srv.URL, Bearer("k"), make(chan int), nil); !errors.Is(err, errs.ErrTransportMarshal) {
		t.Errorf("✗ PostJSON(unmarshalable) = %v, want ErrTransportMarshal", err)
	}

	bad := "http://\x7f bad"
	if _, _, err := SendBody(t.Context(), bad, Bearer("k"), "application/json", []byte("{}"), nil); !errors.Is(err, errs.ErrTransportCreate) {
		t.Errorf("✗ SendBody(bad endpoint) = %v, want ErrTransportCreate", err)
	}

	if _, _, err := GetAuth(t.Context(), bad, Bearer("k"), metadata.Asynchronous, nil); !errors.Is(err, errs.ErrTransportCreate) {
		t.Errorf("✗ GetAuth(bad url) = %v, want ErrTransportCreate", err)
	}

	if _, _, err := PostJSON(t.Context(), "http://127.0.0.1:1/x", Bearer("k"), map[string]string{}, nil); !errors.Is(err, errs.ErrTransportRequest) {
		t.Errorf("✗ PostJSON(unreachable) = %v, want ErrTransportRequest", err)
	}

	if !t.Failed() {
		t.Log("✓ marshal, create, and request failures wrap their transport sentinels")
	}
}

// TestRedirectCreds verifies invariant #2: Redirect credential scope.
//
// What is being tested:
// When GetAuth or Fetch follows redirects, it must send credentials at the original origin and omit
// them at a different origin. For GetAuth chains that return to the original origin, it must
// restore the credential there while leaving the foreign hop uncredentialed.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRedirectCreds(t *testing.T) {
	// The main server redirects /hop/same/<tag> within its own origin and /hop/other/<tag> to
	// the second origin; each case's final destination is its own tagged path, so one shared
	// capture never conflates cases.
	var mainURL string

	otherRec, other := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
		if redirectTag, ok := strings.CutPrefix(r.URL.Path, "/bounce/"); ok {
			// #nosec G710 -- the harness redirects to the redirect target the case itself requested; no untrusted client exists.
			http.Redirect(w, r, mainURL+"/land/"+redirectTag, http.StatusFound)

			return
		}

		w.Header().Set("Content-Type", "image/png")
		fmt.Fprint(w, "DATA")
	})
	mainRec, main := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/hop/same/"):
			// #nosec G710 -- the harness redirects to the redirect target the case itself requested; no untrusted client exists.
			http.Redirect(w, r, mainURL+"/land/"+strings.TrimPrefix(r.URL.Path, "/hop/same/"), http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/hop/other/"):
			// #nosec G710 -- the harness redirects to the redirect target the case itself requested; no untrusted client exists.
			http.Redirect(w, r, other.URL+"/land/"+strings.TrimPrefix(r.URL.Path, "/hop/other/"), http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/hop/back/"):
			// #nosec G710 -- the harness redirects to the redirect target the case itself requested; no untrusted client exists.
			http.Redirect(w, r, other.URL+"/bounce/"+strings.TrimPrefix(r.URL.Path, "/hop/back/"), http.StatusFound)
		default:
			w.Header().Set("Content-Type", "image/png")
			fmt.Fprint(w, "DATA")
		}
	})
	mainURL = main.URL

	creds := []struct {
		name       string
		credential AuthCredential
		header     string
		value      string
	}{
		{"bearer", Bearer("k"), "Authorization", "Bearer k"},
		{"keyhdr", HeaderCred("x-key", "secret"), "x-key", "secret"},
	}
	for _, c := range creds {
		t.Run(c.name+" same-origin keeps", func(t *testing.T) {
			checkHop(t, main.URL+"/hop/same/"+c.name, c.credential, mainRec, "/land/"+c.name, c.header, c.value)

			if !t.Failed() {
				t.Logf("✓ %s", c.name+" same-origin keeps")
			}
		})
		t.Run(c.name+" cross-origin strips", func(t *testing.T) {
			checkHop(t, main.URL+"/hop/other/"+c.name, c.credential, otherRec, "/land/"+c.name, c.header, "")

			if !t.Failed() {
				t.Logf("✓ %s", c.name+" cross-origin strips")
			}
		})
	}

	t.Run("fetch cross-origin strips", func(t *testing.T) {
		checkFetchHop(t, main.URL+"/hop/other/fetch", otherRec, "/land/fetch", "")

		if !t.Failed() {
			t.Log("✓ fetch cross-origin strips")
		}
	})
	t.Run("fetch same-origin keeps", func(t *testing.T) {
		checkFetchHop(t, main.URL+"/hop/same/fetch", mainRec, "/land/fetch", "fetch-secret")

		if !t.Failed() {
			t.Log("✓ fetch same-origin keeps")
		}
	})
	// The strip is per hop against the original origin, never sticky: a chain leaving the
	// origin and returning to it carries no credential at the foreign hop but carries its own
	// again at the final same-origin destination.
	for _, c := range creds {
		t.Run(c.name+" returning to the origin re-carries", func(t *testing.T) {
			tag := "back-" + c.name
			checkHop(t, main.URL+"/hop/back/"+tag, c.credential, mainRec, "/land/"+tag, c.header, c.value)

			if got := otherRec.byPath(t, "/bounce/"+tag).header.Get(c.header); got != "" {
				t.Errorf("✗ the foreign bounce hop carried %s = %q, want absent", c.header, got)
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name+" returning to the origin re-carries")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ credentials survive same-origin redirects, never cross an origin change, and re-carry on a return to the origin")
	}
}

// TestTransportCancelMid verifies invariant #3: Mid-request cancellation.
//
// What is being tested:
// When the caller cancels an in-flight PostJSON request, PostJSON must return within two seconds
// with context.Canceled and ErrCanceled, without ErrTransport.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestTransportCancelMid(t *testing.T) {
	_, srv := recSrv(t, func(_ http.ResponseWriter, r *http.Request) {
		// Hold the response until the canceled client disconnects, so the call is genuinely
		// in flight; the request context unblocks the handler and lets the harness close
		// cleanly.
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()

	_, _, err := PostJSON(ctx, srv.URL, Bearer("k"), map[string]string{"a": "b"}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("✗ err = %v, want context.Canceled through the chain", err)
	}

	if !errors.Is(err, errs.ErrCanceled) {
		t.Errorf("✗ err = %v, want the cancellation marker, not a transport failure", err)
	}

	if errors.Is(err, errs.ErrTransport) {
		t.Errorf("✗ err = %v, an interrupted run must not read as a transport failure", err)
	}

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("✗ in-flight call survived %v after cancellation, want prompt interruption", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ a mid-request cancellation interrupts the transport promptly and is marked canceled")
	}
}

// TestTransportCanceled verifies invariant #4: Pre-canceled transport.
//
// What is being tested:
// Given a context canceled before the request, PostJSON must return within one second with
// context.Canceled and ErrCanceled.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestTransportCanceled(t *testing.T) {
	_, srv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)

		_, _ = w.Write([]byte("{}"))
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()

	_, _, err := PostJSON(ctx, srv.URL, Bearer("k"), map[string]string{"a": "b"}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("✗ err = %v, want context.Canceled through the transport chain", err)
	}

	if !errors.Is(err, errs.ErrCanceled) {
		t.Errorf("✗ err = %v, want the cancellation marker", err)
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("✗ blocked %v on a pre-canceled context, want prompt failure", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ a pre-canceled context fails the transport promptly and is marked canceled")
	}
}

// TestTransportDeadlineIsNotCancellation verifies invariant #5: Transport deadline classification.
//
// What is being tested:
// Given a 50-millisecond deadline and a server that waits two seconds, PostJSON must return
// ErrTransportRequest without ErrCanceled.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestTransportDeadlineIsNotCancellation(t *testing.T) {
	_, srv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)

		_, _ = w.Write([]byte("{}"))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := PostJSON(ctx, srv.URL, Bearer("k"), map[string]string{"a": "b"}, nil)
	if !errors.Is(err, errs.ErrTransportRequest) {
		t.Errorf("✗ err = %v, want a wrapped ErrTransportRequest", err)
	}

	if errors.Is(err, errs.ErrCanceled) {
		t.Errorf("✗ err = %v, a deadline must not be marked as a cancellation", err)
	}

	if !t.Failed() {
		t.Log("✓ a deadline stays a transport failure, distinct from an interrupt")
	}
}

// TestRespBound verifies invariant #6: API response size bound.
//
// What is being tested:
// Given a response 1024 bytes larger than the 64 MiB API bound, PostJSON and GetAuth must each
// return ErrTransportSize.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRespBound(t *testing.T) {
	over := func(w http.ResponseWriter, _ *http.Request) {
		t.Helper()

		_ = patWrite(t, w, apiBound+1024)
	}

	cases := []struct {
		name string
		call func(ctx context.Context, url string) (int, []byte, error)
	}{
		{"PostJSON", func(ctx context.Context, url string) (int, []byte, error) {
			return PostJSON(ctx, url, Bearer("k"), map[string]string{"a": "b"}, nil)
		}},
		{"GetAuth", func(ctx context.Context, url string) (int, []byte, error) {
			return GetAuth(ctx, url, Bearer("k"), metadata.Asynchronous, nil)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, srv := recSrv(t, over)

			_, _, err := c.call(t.Context(), srv.URL)
			if !errors.Is(err, errs.ErrTransportSize) {
				t.Errorf("✗ over-bound body err = %v, want ErrTransportSize", err)
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ over-bound API bodies fail with the size sentinel, never truncate")
	}
}

// TestHTTPCaptureFailure verifies invariant #7: Incomplete response capture.
//
// What is being tested:
// Given a truncated JSON response, GetAuth must return ErrTransportRead and a nil body while
// recording the received prefix as incomplete with capture and decode errors. A subsequent canceled
// PostJSON must record its request error and ErrCanceled without adding a response.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestHTTPCaptureFailure(t *testing.T) {
	const partialBody = `{"unfinished":`

	_, server := recSrv(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Length", strconv.Itoa(len(partialBody)+100))
		writer.Header().Set("Content-Type", "application/json")

		if _, err := io.WriteString(writer, partialBody); err != nil {
			t.Errorf("✗ harness write: %v", err)
		}
	})
	record := metadata.New("provider", "model", "test", "boat", time.Now())

	_, body, err := GetAuth(t.Context(), server.URL, AuthCredential{}, metadata.Asynchronous, record)
	if !errors.Is(err, errs.ErrTransportRead) || body != nil {
		t.Errorf("✗ read failure outcome: %q %v", body, err)
	}

	if len(record.Responses) != 1 {
		t.Errorf("✗ missing incomplete response: %+v", record.Responses)
	} else {
		response := record.Responses[0]

		var text string

		if err := json.Unmarshal(response.Body, &text); err != nil {
			t.Errorf("✗ partial text decode: %v", err)
		}

		if text != partialBody || !response.Incomplete || response.CaptureError == "" || response.DecodeError == "" {
			t.Errorf("✗ inaccurate partial response: %+v", response)
		}
	}

	canceled, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err = PostJSON(canceled, server.URL, AuthCredential{}, map[string]string{"prompt": "boat"}, record)
	if !errors.Is(err, errs.ErrCanceled) {
		t.Errorf("✗ canceled request: %v", err)
	}

	if len(record.Request.Calls) != 2 || len(record.Responses) != 1 {
		t.Errorf("✗ request without response misrecorded: %+v", record)
	} else if record.Request.Calls[1].Error == "" {
		t.Error("✗ canceled request lacks its error")
	}

	if !t.Failed() {
		t.Log("✓ partial responses and response-free failures retain truthful capture details")
	}
}

// TestReadLimited verifies invariant #8: Bounded read behavior.
//
// What is being tested:
// Given bodies below or exactly at a four-byte limit, readFirstBytes must return their complete
// bytes without an error. An oversized body must return ErrTransportSize; a failing reader must
// return ErrTransportRead under ErrTransport. Both errors must mention the requested limit.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestReadLimited(t *testing.T) {
	_, err := readFirstBytes(errReader{test: t}, 1024)
	if !errors.Is(err, errs.ErrTransportRead) || !errors.Is(err, errs.ErrTransport) {
		t.Errorf("✗ ReadLimited(failing reader) = %v, want errs.ErrTransportRead under errs.ErrTransport", err)
	}

	if err != nil && !strings.Contains(err.Error(), "1024") {
		t.Errorf("✗ ReadLimited(failing reader) err %q does not carry the byte limit", err)
	}

	if b, err := readFirstBytes(strings.NewReader("abcd"), 4); err != nil || string(b) != "abcd" {
		t.Errorf("✗ at-limit = (%q, %v), want the whole body", b, err)
	}

	if b, err := readFirstBytes(strings.NewReader("abc"), 4); err != nil || string(b) != "abc" {
		t.Errorf("✗ under-limit = (%q, %v), want abc", b, err)
	}

	_, err = readFirstBytes(strings.NewReader("0123456789"), 4)
	if !errors.Is(err, errs.ErrTransportSize) {
		t.Errorf("✗ over-limit body = %v, want errs.ErrTransportSize, not truncation", err)
	}

	if err != nil && !strings.Contains(err.Error(), "4") {
		t.Errorf("✗ over-limit err %q does not carry the byte limit", err)
	}

	if !t.Failed() {
		t.Log("✓ ReadLimited wraps a reader error with the limit context, reads at-limit whole, and errors on overflow")
	}
}

// TestReadPrefixFailure verifies invariant #9: Partial response reads.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a reader returning abcdef with io.ErrUnexpectedEOF and a four-byte limit, readFirstBytes
// must return abcd with ErrTransportRead and the original cause. With math.MaxInt64 as the limit,
// it must read abcd completely without an error.
// Kind: permanent.
func TestReadPrefixFailure(t *testing.T) {
	prefix, err := readFirstBytes(prefixFaultReader{test: t}, 4)
	if string(prefix) != "abcd" || !errors.Is(err, io.ErrUnexpectedEOF) || !errors.Is(err, errs.ErrTransportRead) {
		t.Errorf("✗ failed read returned prefix %q, error %v", prefix, err)
	}

	prefix, err = readFirstBytes(strings.NewReader("abcd"), math.MaxInt64)
	if string(prefix) != "abcd" || err != nil {
		t.Errorf("✗ maximum limit returned prefix %q, error %v", prefix, err)
	}

	if !t.Failed() {
		t.Log("✓ failed reads retain only bounded bytes and limit arithmetic does not overflow")
	}
}

// TestReadLimitedDegenerate verifies invariant #10: Bounded reads with nonpositive limits.
//
// What is being tested:
// Given a zero limit, readFirstBytes must accept an empty body and reject a nonempty body with
// ErrTransportSize. A negative limit must return ErrTransportSize even for an empty body.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestReadLimitedDegenerate(t *testing.T) {
	if b, err := readFirstBytes(strings.NewReader(""), 0); err != nil || len(b) != 0 {
		t.Errorf("✗ zero-limit empty source = (%q, %v), want an empty read", b, err)
	}

	if _, err := readFirstBytes(strings.NewReader("x"), 0); !errors.Is(err, errs.ErrTransportSize) {
		t.Errorf("✗ zero-limit non-empty source = %v, want errs.ErrTransportSize", err)
	}

	if _, err := readFirstBytes(strings.NewReader(""), -1); !errors.Is(err, errs.ErrTransportSize) {
		t.Errorf("✗ negative limit = %v, want the fail-closed errs.ErrTransportSize", err)
	}

	if !t.Failed() {
		t.Log("✓ the zero and negative limits fail closed, never truncating")
	}
}

// FuzzReadLimited verifies invariant #11: Bounded reads under arbitrary input.
//
// What is being tested:
// For arbitrary bytes and unsigned 16-bit limits, readFirstBytes must return all original bytes
// without an error when the body fits, and return ErrTransportSize when it exceeds the limit.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzReadLimited(f *testing.F) {
	f.Add([]byte{}, uint16(0))
	f.Add([]byte("abcd"), uint16(4))
	f.Add([]byte("abcde"), uint16(4))
	f.Add([]byte("x"), uint16(0))
	f.Fuzz(func(t *testing.T, data []byte, limit uint16) {
		got, err := readFirstBytes(bytes.NewReader(data), int64(limit))
		if int64(len(data)) <= int64(limit) {
			if err != nil || !bytes.Equal(got, data) {
				t.Errorf("✗ within-limit read = (%d bytes, %v), want the whole %d-byte body", len(got), err, len(data))
			}
		} else if !errors.Is(err, errs.ErrTransportSize) {
			t.Errorf("✗ over-limit read = (%d bytes, %v), want errs.ErrTransportSize, never truncation", len(got), err)
		}

		if !t.Failed() {
			t.Logf("✓ the bound held without truncation")
		}
	})
}

// byPath returns the first captured request with the given path; fatal when none arrived, since no
// request assertion after it could run.
func (c *capture) byPath(t *testing.T, reqPath string) recReq {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.reqs {
		if r.reqPath == reqPath {
			return r
		}
	}

	t.Fatalf("💣 no request for %s reached the harness — nothing to inspect", reqPath)

	return recReq{}
}

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

// checkHop drives one credentialed GetAuth through a redirect and asserts the credential header
// that arrived at the target ("" = must be absent).
func checkHop(t *testing.T, url string, credential AuthCredential, recorder *capture, land, header, want string) {
	t.Helper()

	status, _, err := GetAuth(t.Context(), url, credential, metadata.Asynchronous, nil)
	if err != nil || status != http.StatusOK {
		t.Errorf("✗ redirect not followed: (%d, %v)", status, err)

		return
	}

	if got := recorder.byPath(t, land).header.Get(header); got != want {
		t.Errorf("✗ landing %s = %q, want %q", header, got, want)
	}
}

// checkFetchHop drives one credentialed Fetch through a redirect and asserts the x-key header that
// arrived at the target ("" = must be absent).
func checkFetchHop(t *testing.T, url string, recorder *capture, land, want string) {
	t.Helper()

	credential := HeaderCred("x-key", "fetch-secret")
	generatedMedia, err := Fetch(t.Context(), url, credential, ".bin", nil)
	rmArtifact(t, generatedMedia)

	if err != nil {
		t.Errorf("✗ redirected fetch failed: %v", err)

		return
	}

	if got := recorder.byPath(t, land).header.Get("X-Key"); got != want {
		t.Errorf("✗ landing x-key = %q, want %q", got, want)
	}
}

// errReader exercises a response read failure without returning bytes.
//   - test: the owning test
type errReader struct{ test testing.TB }

// Read always fails before returning any bytes.
func (reader errReader) Read([]byte) (int, error) {
	reader.test.Helper()

	return 0, errors.New("read boom")
}

// prefixFaultReader returns bytes alongside an interrupted-body error.
//   - test: the owning test
type prefixFaultReader struct{ test testing.TB }

// Read returns the fixture prefix and its configured read error.
func (reader prefixFaultReader) Read(destination []byte) (int, error) {
	reader.test.Helper()

	return copy(destination, "abcdef"), io.ErrUnexpectedEOF
}
