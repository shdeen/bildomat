package bfl

// Invariants tested:
//  1. BFL transport: When the submission host is unreachable, Generate must return
//     ErrTransportRequest under ErrTransport and name the provider and model. When the first
//     polling connection drops, Generate must retry polling and return one artifact without an
//     error.
//  2. BFL acknowledgment without a job identifier: When a creation response omits the job ID and a
//     subsequent Ready response omits the sample, Generate must return ErrResponseNoSample and
//     include bfl (flux-2-pro) in the error.
//  3. BFL API failure paths: Generate must return ErrResponseStatus for submission HTTP 429 or 503
//     and polling HTTP 500, ErrResponseDecode for malformed submission or polling JSON,
//     ErrResponseNoData when submission omits polling_url, and ErrTransport when the artifact
//     download fails.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/params"
)

// TestBFLTransport verifies invariant #1: BFL transport.
//
// What is being tested:
// When the submission host is unreachable, Generate must return ErrTransportRequest under
// ErrTransport and name the provider and model. When the first polling connection drops, Generate
// must retry polling and return one artifact without an error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestBFLTransport(t *testing.T) {
	t.Run("an unreachable submit host fails labeled", transportSubmitDown)
	t.Run("a dropped poll retries to the result", transportPollDrop)

	if !t.Failed() {
		t.Log("✓ submit transport failures are labeled and classified; dropped polls are transient")
	}
}

// TestBFLNoIDAck verifies invariant #2: BFL acknowledgment without a job identifier.
//
// What is being tested:
// When a creation response omits the job ID and a subsequent Ready response omits the sample,
// Generate must return ErrResponseNoSample and include bfl (flux-2-pro) in the error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestBFLNoIDAck(t *testing.T) {
	h := newHarness(t)
	sub := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"polling_url":"`+h.pollURL()+`"}`)
	})
	h.pollBodies = []string{`{"status":"Ready"}`} // sampleless: the no-data failure

	t.Setenv("BFL_API_KEY", testKey)

	g := harnessProvider(t, sub.srv.URL)

	_, err := g.Generate(context.Background(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: bflProvider(t), Model: fixtureFlux2(t)}, Prompt: params.GetSetIf(true, testPrompt).ValOr(""), APIKey: os.Getenv((catalog.ProvModelPair{Provider: bflProvider(t), Model: fixtureFlux2(t)}).Provider.APIKeyEnvVar)})
	if err == nil || !errors.Is(err, errs.ErrResponseNoSample) || !strings.Contains(err.Error(), "bfl (flux-2-pro)") {
		t.Errorf("✗ err = %v, want the no-sample failure naming the provider label in the id's place", err)
	}

	if !t.Failed() {
		t.Log("✓ an id-less ack falls back to the provider label for error naming")
	}
}

// TestBFLErrPaths verifies invariant #3: BFL API failure paths.
//
// What is being tested:
// Generate must return ErrResponseStatus for submission HTTP 429 or 503 and polling HTTP 500,
// ErrResponseDecode for malformed submission or polling JSON, ErrResponseNoData when submission
// omits polling_url, and ErrTransport when the artifact download fails.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestBFLErrPaths(t *testing.T) {
	cases := []errCase{
		{name: "429 submit", subStatus: 429, subBody: `{"error":{"message":"rate limited"}}`, want: errs.ErrResponseStatus},
		{name: "503 submit", subStatus: 503, subBody: `service unavailable`, want: errs.ErrResponseStatus},
		{name: "malformed submit body", subStatus: 200, subBody: `{not json`, want: errs.ErrResponseDecode},
		{name: "submit without polling_url", subStatus: 200, subBody: `{"id":"job-1"}`, want: errs.ErrResponseNoData},
		{name: "failed sample download", dlStatus: 500, want: errs.ErrTransport},
		{name: "malformed poll body", pollBody: `{"status": not json`, want: errs.ErrResponseDecode},
		{name: "non-2xx poll", pollStatus: 500, want: errs.ErrResponseStatus},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkErrPath(t, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every submit/poll/download error path classifies to its sentinel")
	}
}

// errCase specifies the submission, polling, or download response to alter and the expected error
// classification.
type errCase struct {
	name       string
	subStatus  int // non-zero: the submit host answers this status with subBody
	subBody    string
	pollStatus int    // non-zero: the poll host answers this status
	pollBody   string // non-empty: the (2xx) poll host serves this body once
	dlStatus   int    // non-zero: the download host answers this status
	want       error
}

// transportSubmitDown checks that an unreachable submission retains transport and model context.
func transportSubmitDown(t *testing.T) {
	t.Helper()
	t.Setenv("BFL_API_KEY", testKey)

	g := harnessProvider(t, "http://127.0.0.1:1")

	_, err := g.Generate(context.Background(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: bflProvider(t), Model: fixtureFlux2(t)}, Prompt: params.GetSetIf(true, testPrompt).ValOr(""), APIKey: os.Getenv((catalog.ProvModelPair{Provider: bflProvider(t), Model: fixtureFlux2(t)}).Provider.APIKeyEnvVar)})
	if err == nil || !errors.Is(err, errs.ErrTransportRequest) || !errors.Is(err, errs.ErrTransport) {
		t.Errorf("✗ err = %v, want the request-failure sentinel under the transport root", err)
	}

	if err != nil && !strings.Contains(err.Error(), "bfl (flux-2-pro)") {
		t.Errorf("✗ err %q does not carry the provider label", err)
	}

	if !t.Failed() {
		t.Log("✓ an unreachable submit host fails labeled")
	}
}

// dropOnceSrv serves a hijack-dropped connection on the first request and body on every later one,
// reporting how many requests arrived.
func dropOnceSrv(t *testing.T, body func() string) (*httptest.Server, func() int) {
	t.Helper()

	var mu sync.Mutex

	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		polls++
		first := polls == 1
		mu.Unlock()

		if first {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Errorf("✗ harness lacks Hijacker — cannot simulate a dropped connection")

				return
			}

			conn, _, _ := hj.Hijack()
			_ = conn.Close()

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body())
	}))
	t.Cleanup(srv.Close)

	return srv, func() int {
		mu.Lock()
		defer mu.Unlock()

		return polls
	}
}

// transportPollDrop checks recovery after a dropped polling connection and registers artifact
// cleanup.
func transportPollDrop(t *testing.T) {
	t.Helper()
	h := newHarness(t)
	drop, polls := dropOnceSrv(t, h.ready)
	sub := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"job-1","polling_url":"`+drop.URL+`/v1/get_result?id=job-1"}`)
	})
	t.Setenv("BFL_API_KEY", testKey)

	g := harnessProvider(t, sub.srv.URL)

	res, err := g.Generate(context.Background(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: bflProvider(t), Model: fixtureFlux2(t)}, Prompt: params.GetSetIf(true, testPrompt).ValOr(""), APIKey: os.Getenv((catalog.ProvModelPair{Provider: bflProvider(t), Model: fixtureFlux2(t)}).Provider.APIKeyEnvVar)})
	for _, a := range res.Artifacts {
		if a.TmpPath != "" {
			path := a.TmpPath

			t.Cleanup(func() { _ = os.Remove(path) })
		}
	}

	if err != nil || len(res.Artifacts) != 1 {
		t.Errorf("✗ run = (%d artifacts, %v), want the download past the dropped poll", len(res.Artifacts), err)
	}

	if got := polls(); got < 2 {
		t.Errorf("✗ the poll host saw %d request(s), want the drop retried", got)
	}

	if !t.Failed() {
		t.Log("✓ a dropped poll retries to the result")
	}
}

// checkErrPath creates a harness with the case's fault injected and asserts the run's error carries
// the expected sentinel.
func checkErrPath(t *testing.T, c errCase) {
	t.Helper()

	h := newHarness(t)
	if c.subStatus != 0 {
		h.sub = errServer(t, c.subStatus, c.subBody)
	}

	if c.pollStatus != 0 {
		h.poll = errServer(t, c.pollStatus, "boom")
	}

	if c.dlStatus != 0 {
		h.dl = errServer(t, c.dlStatus, "gone")
	}

	if c.pollBody != "" {
		h.pollBodies = []string{c.pollBody}
	} else {
		h.pollBodies = []string{h.ready()}
	}

	_, err := run(t, h, fixtureFlux2(t), params.FlagInputs{}, nil)
	if !errors.Is(err, c.want) {
		t.Errorf("✗ %s: error = %v, want sentinel %v", c.name, err, c.want)
	}

	if !t.Failed() {
		t.Logf("✓ %s classifies to its sentinel", c.name)
	}
}

// errServer returns a recording server that responds with the supplied HTTP status and body.
func errServer(t *testing.T, status int, body string) *recServer {
	t.Helper()

	return newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})
}
