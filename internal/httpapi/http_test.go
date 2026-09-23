package httpapi

// Invariants tested:
//  1. Credential headers: Given bearer or custom-header credentials, PostJSON, GetAuth, and
//     SendBody must send the expected method, credential, and applicable content type and body.
//  2. Retained HTTP transactions: Given a JSON submission followed by pending and error polls,
//     PostJSON and GetAuth must record three calls and responses in order, preserving the submitted
//     large integer and each complete response value.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/metadata"
)

// TestCredHeaders verifies invariant #1: Credential headers.
//
// What is being tested:
// Given bearer or custom-header credentials, PostJSON, GetAuth, and SendBody must send the expected
// method, credential, and applicable content type and body. Each call must return HTTP 200 and a
// response body containing ok.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCredHeaders(t *testing.T) {
	cases := []credCase{
		{
			name: "PostJSON bearer",
			call: func(ctx context.Context, url string) (int, []byte, error) {
				return PostJSON(ctx, url, Bearer("k"), map[string]string{"model": "m"}, nil)
			},
			reqMethod: http.MethodPost, header: "Authorization", value: "Bearer k",
			ctype: "application/json", body: `{"model":"m"}`,
		},
		{
			name: "GetAuth key header",
			call: func(ctx context.Context, url string) (int, []byte, error) {
				return GetAuth(ctx, url, HeaderCred("x-key", "k"), metadata.Asynchronous, nil)
			},
			reqMethod: http.MethodGet, header: "x-key", value: "k",
		},
		{
			name: "SendBody bearer",
			call: func(ctx context.Context, url string) (int, []byte, error) {
				return SendBody(ctx, url, Bearer("k"), "text/plain", []byte("payload"), nil)
			},
			reqMethod: http.MethodPost, header: "Authorization", value: "Bearer k",
			ctype: "text/plain", body: "payload",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recorder, srv := recSrv(t, textOK(t, `{"ok":true}`))
			status, body, err := c.call(t.Context(), srv.URL)
			checkCred(t, c, status, body, err, recorder)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every surface method sends its constructed credential and returns status+body")
	}
}

// TestHTTPRecord verifies invariant #2: Retained HTTP transactions.
//
// What is being tested:
// Given a JSON submission followed by pending and error polls, PostJSON and GetAuth must record
// three calls and responses in order, preserving the submitted large integer and each complete
// response value. Response roles, request indexes, and endpoints must agree; marshaling the record
// must omit the credential and Authorization header.
//
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
func TestHTTPRecord(t *testing.T) {
	errorBody := `{"error":{"message":"` + strings.Repeat("detail", 300) + `"},"unknown":null}`
	bodies := []string{`{"job":"boat","unknown":[9007199254740993,null]}`, `{"status":"pending"}`, errorBody}
	recorder, server := recSrv(t, func(writer http.ResponseWriter, request *http.Request) {
		index, err := strconv.Atoi(strings.TrimPrefix(request.URL.Path, "/"))
		if err != nil || index < 0 || index >= len(bodies) {
			t.Error("✗ invalid harness request")
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("Content-Type", "application/json")

		if index == 2 {
			writer.WriteHeader(http.StatusBadGateway)
		}

		if _, err := io.WriteString(writer, bodies[index]); err != nil {
			t.Errorf("✗ harness write: %v", err)
		}
	})
	record := metadata.New("provider", "model", "test", "boat", time.Now())
	payload := json.RawMessage(`{"prompt":"boat","number":9007199254740993}`)

	if _, _, err := PostJSON(t.Context(), server.URL+"/0", Bearer("private-key"), payload, record); err != nil {
		t.Errorf("✗ submission: %v", err)
	}

	for index := 1; index < len(bodies); index++ {
		if _, _, err := GetAuth(t.Context(), server.URL+"/"+strconv.Itoa(index), Bearer("private-key"), metadata.Asynchronous, record); err != nil {
			t.Errorf("✗ poll: %v", err)
		}
	}

	if recorder.count() != 3 || len(record.Request.Calls) != 3 || len(record.Responses) != 3 {
		t.Errorf("✗ transaction counts: sent=%d requests=%d responses=%d", recorder.count(), len(record.Request.Calls), len(record.Responses))
	} else {
		var submitted struct {
			Prompt string `json:"prompt"`
			Number int64  `json:"number"`
		}
		if err := json.Unmarshal(record.Request.Calls[0].Payload, &submitted); err != nil {
			t.Errorf("✗ retained payload: %v", err)
		}

		if submitted.Prompt != "boat" || submitted.Number != 9007199254740993 {
			t.Errorf("✗ submitted payload changed: %+v", submitted)
		}

		for index, response := range record.Responses {
			var originalValue, retainedValue any

			originalDecoder := json.NewDecoder(strings.NewReader(bodies[index]))
			originalDecoder.UseNumber()

			retainedDecoder := json.NewDecoder(bytes.NewReader(response.Body))
			retainedDecoder.UseNumber()

			if err := originalDecoder.Decode(&originalValue); err != nil {
				t.Fatal("💣 invalid response fixture")
			}

			if err := retainedDecoder.Decode(&retainedValue); err != nil {
				t.Errorf("✗ invalid retained response: %v", err)
			}

			if !reflect.DeepEqual(originalValue, retainedValue) {
				t.Errorf("✗ response %d lost data: %s", index, response.Body)
			}

			role := metadata.Asynchronous
			if index == 0 {
				role = metadata.Synchronous
			}

			if response.Type != role || response.RequestIndex != index || response.Endpoint != record.Request.Calls[index].Endpoint {
				t.Errorf("✗ response association %d: %+v", index, response)
			}
		}
	}

	encoded, err := json.Marshal(record)
	if err != nil || bytes.Contains(encoded, []byte("private-key")) || bytes.Contains(encoded, []byte("Authorization")) {
		t.Errorf("✗ credential included or invalid record: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ records retain complete ordered HTTP transactions and exclude credentials")
	}
}

// Response and artifact fixtures.
//   - apiBound: the 64 MiB limit expected for API responses
//   - pngExt: the expected extension for PNG artifacts
const (
	apiBound = 64 << 20
	pngExt   = ".png"
)

// recReq holds one captured HTTP request.
//   - reqMethod: the HTTP method
//   - reqPath: the request URL path
//   - header: the captured request headers
//   - body: the complete request body
type recReq struct {
	reqMethod string
	reqPath   string
	header    http.Header
	body      []byte
}

// capture retains HTTP requests received by the test server.
//   - test: the owning test
//   - mu: protects concurrent request access
//   - reqs: requests in receipt order
type capture struct {
	test testing.TB
	mu   sync.Mutex
	reqs []recReq
}

// credCase defines one authenticated request and its expected encoding.
//   - name: the table case name
//   - call: the request function to exercise
//   - reqMethod: the expected HTTP method
//   - header: the credential header name
//   - value: the credential header value
//   - ctype: the expected content type, or empty to omit that check
//   - body: the expected body, or empty to omit that check
type credCase struct {
	name      string
	call      func(ctx context.Context, url string) (int, []byte, error)
	reqMethod string
	header    string
	value     string
	ctype     string // expected request content type ("" = not asserted)
	body      string // expected recorded request body ("" = not asserted)
}

// add records one incoming request, body included.
func (c *capture) add(r *http.Request) {
	c.test.Helper()

	body, _ := io.ReadAll(r.Body)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.reqs = append(c.reqs, recReq{reqMethod: r.Method, reqPath: r.URL.Path, header: r.Header.Clone(), body: body})
}

// count returns how many requests the harness has received.
func (c *capture) count() int {
	c.test.Helper()

	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.reqs)
}

// last returns the most recently captured request; fatal when none arrived, since no request
// assertion after it could run.
func (c *capture) last(t *testing.T) recReq {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.reqs) == 0 {
		t.Fatalf("💣 no request reached the harness — nothing to inspect")
	}

	return c.reqs[len(c.reqs)-1]
}

// recSrv records each request before invoking respond and closes the server when the test ends.
func recSrv(t *testing.T, respond http.HandlerFunc) (*capture, *httptest.Server) {
	t.Helper()

	c := &capture{test: t}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.add(r)
		respond(w, r)
	}))
	t.Cleanup(srv.Close)

	return c, srv
}

// textOK returns a handler that responds with a fixed successful text body.
func textOK(test testing.TB, body string) http.HandlerFunc {
	test.Helper()

	return func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }
}

// checkCred compares a captured request with its credential and encoding expectations.
func checkCred(t *testing.T, c credCase, status int, body []byte, err error, recorder *capture) {
	t.Helper()

	if err != nil {
		t.Errorf("✗ unexpected error: %v", err)

		return
	}

	if status != http.StatusOK || !strings.Contains(string(body), "ok") {
		t.Errorf("✗ returned (%d, %q), want 200 and the response body", status, body)
	}

	req := recorder.last(t)
	if req.reqMethod != c.reqMethod {
		t.Errorf("✗ reqMethod = %s, want %s", req.reqMethod, c.reqMethod)
	}

	if got := req.header.Get(c.header); got != c.value {
		t.Errorf("✗ header %s = %q, want %q", c.header, got, c.value)
	}

	if c.ctype != "" && !strings.HasPrefix(req.header.Get("Content-Type"), c.ctype) {
		t.Errorf("✗ content type = %q, want %q", req.header.Get("Content-Type"), c.ctype)
	}

	if c.body != "" && string(req.body) != c.body {
		t.Errorf("✗ recorded body = %q, want %q", req.body, c.body)
	}
}
