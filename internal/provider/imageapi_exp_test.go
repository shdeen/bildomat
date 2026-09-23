package provider

// Invariants tested:
//  1. URL response normalization requirements: Given RespImageURL enabled, submitImage must
//     download the result URL without Authorization and return one file-backed .png.
//  2. Atomic response normalization after partial failure: Given a successful first URL followed by
//     HTTP 403 or an interrupted body, submitImage must return the expected transport category
//     under ErrTransport and name the provider/model and entry 2.
//  3. Empty multipart image: Given a multipart reference with MIME but no bytes, submitImage must
//     return ErrInputMediaEmpty.

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestNormURLGate verifies invariant #1: URL response normalization requirements.
//
// What is being tested:
// Given RespImageURL enabled, submitImage must download the result URL without Authorization and
// return one file-backed .png. With the option absent, it must avoid the download, return
// ErrResponseNoData naming the provider/model label, and return no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestNormURLGate(t *testing.T) {
	dlRec, dlSrv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		fmt.Fprint(w, "URLBYTES")
	})

	t.Run("RespImageURL declared fetches bare", func(t *testing.T) {
		body := fmt.Sprintf(`{"data":[{"url":%q}]}`, dlSrv.URL+"/img")

		out := driveSub(t, singleOrArrayImage, nil, params.Values{}, textOK(t, body))
		for _, a := range out.artifacts {
			rmArtifact(t, a)
		}

		if out.err != nil {
			t.Errorf("✗ submit failed: %v", out.err)

			return
		}

		if len(out.artifacts) != 1 || out.artifacts[0].TmpPath == "" || out.artifacts[0].FileExt != ".png" {
			t.Errorf("✗ artifacts = %+v, want one file-backed .png", out.artifacts)

			return
		}

		checkCarriage(t, "url fetch", out.artifacts[0])

		if v := dlRec.last(t).header.Get("Authorization"); v != "" {
			t.Errorf("✗ the pre-signed fetch sent Authorization %q, want bare", v)
		}

		if !t.Failed() {
			t.Log("✓ RespImageURL declared fetches bare")
		}
	})

	t.Run("RespImageURL absent never touches the URL", func(t *testing.T) {
		before := dlRec.count()
		body := fmt.Sprintf(`{"data":[{"url":%q}]}`, dlSrv.URL+"/img2")
		out := driveSub(t, multipartImage, nil, params.Values{}, textOK(t, body))
		checkNormFail(t, out.label, out.artifacts, out.err, errs.ErrResponseNoData)

		if dlRec.count() != before {
			t.Errorf("✗ the URL was fetched despite RespImageURL being absent")
		}

		if !t.Failed() {
			t.Log("✓ RespImageURL absent never touches the URL")
		}
	})

	if !t.Failed() {
		t.Log("✓ URL entries fetch bare under RespImageURL and are the no-data error without it")
	}
}

// TestNormPartialClean verifies invariant #2: Atomic response normalization after partial failure.
//
// What is being tested:
// Given a successful first URL followed by HTTP 403 or an interrupted body, submitImage must return
// the expected transport category under ErrTransport and name the provider/model and entry 2. It
// must return no artifacts and leave no new download files.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestNormPartialClean(t *testing.T) {
	_, good := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		fmt.Fprint(w, "GOODBYTES")
	})
	_, deny := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	_, drop := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "4096")
		fmt.Fprint(w, "part")
		// Flush the headers and partial body so the client is mid-stream — not pre-response
		// — when the connection aborts.
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	})

	cases := []struct {
		name, second string
		sentinel     error
	}{
		{"second entry non-2xx", deny.URL + "/bad", errs.ErrTransportStatus},
		{"second entry mid-body drop", drop.URL + "/bad", errs.ErrTransportDownload},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("TMPDIR", t.TempDir())

			body := fmt.Sprintf(`{"data":[{"url":%q},{"url":%q}]}`, good.URL+"/one", c.second)
			before := dlSet(t)
			out := driveSub(t, singleOrArrayImage, nil, params.Values{}, textOK(t, body))
			checkPartial(t, c.name, out, before, c.sentinel)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ a later-entry failure is atomic: zero artifacts, zero temp files, provider and entry named")
	}
}

// TestFormEmptyImg verifies invariant #3: Empty multipart image.
//
// What is being tested:
// Given a multipart reference with MIME but no bytes, submitImage must return ErrInputMediaEmpty.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFormEmptyImg(t *testing.T) {
	out := driveSub(t, multipartImage, []media.Input{{MIME: "image/png"}}, params.Values{}, artOK(t))
	if !errors.Is(out.err, errs.ErrInputMediaEmpty) {
		t.Errorf("✗ empty image part err = %v, want ErrInputMediaEmpty", out.err)
	}

	if !t.Failed() {
		t.Log("✓ an empty reference image fails the multipart assembly with its sentinel")
	}
}

// checkPartial asserts one partial-failure outcome: the failing leg's precise transport sentinel
// naming the provider and entry, zero artifacts, zero new temp files.
func checkPartial(t *testing.T, name string, out subOut, before map[string]bool, sentinel error) {
	t.Helper()

	if out.err == nil || !errors.Is(out.err, sentinel) || !errors.Is(out.err, errs.ErrTransport) {
		t.Errorf("✗ err = %v, want the precise %v under the transport root", out.err, sentinel)
	}

	if out.err != nil {
		for _, want := range []string{out.label, "entry 2"} {
			if !strings.Contains(out.err.Error(), want) {
				t.Errorf("✗ error %q does not name %q", out.err.Error(), want)
			}
		}
	}

	if len(out.artifacts) != 0 {
		t.Errorf("✗ a partial failure returned %d artifacts, want none", len(out.artifacts))
	}

	assertNoNewDl(t, before, name)
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
