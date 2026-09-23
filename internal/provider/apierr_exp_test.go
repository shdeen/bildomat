package provider

// Invariants tested:
//  1. Provider API error classification: Given a JSON error message, APIErr must return
//     ErrResponseStatus and include the message, HTTP status, and provider/model label.
//  2. API error classification under arbitrary bodies: For arbitrary response bytes, APIErr must
//     return ErrResponseStatus.

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestAPIErr verifies invariant #1: Provider API error classification.
//
// What is being tested:
// Given a JSON error message, APIErr must return ErrResponseStatus and include the message, HTTP
// status, and provider/model label. Plain bodies must retain their status and content; a 500-byte
// body must produce shorter error text while retaining its status and a body prefix.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestAPIErr(t *testing.T) {
	err := APIErr("fixture-provider (m)", 400, []byte(`{"error":{"message":"nope"}}`))
	if err == nil || !errors.Is(err, errs.ErrResponseStatus) {
		t.Fatalf("💣 APIErr = %v, want an ErrResponseStatus error to inspect", err)
	}

	for _, want := range []string{"nope", "400", "fixture-provider (m)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("✗ APIErr text %q missing %q", err.Error(), want)
		}
	}

	raw := APIErr("x (m)", 500, []byte("plain body"))
	if raw == nil || !strings.Contains(raw.Error(), "500") || !strings.Contains(raw.Error(), "plain body") {
		t.Errorf("✗ APIErr(raw body) = %v, want the status and body text", raw)
	}

	long := APIErr("x (m)", 500, []byte(strings.Repeat("x", 500)))
	switch {
	case long == nil:
		t.Errorf("✗ APIErr(long body) = nil, want a truncated snippet naming the status")
	case len(long.Error()) >= 500 || !strings.Contains(long.Error(), "500"):
		t.Errorf("✗ APIErr(long body) renders %d chars, want a truncated snippet naming the status", errLen(t, long))
	case !strings.Contains(long.Error(), strings.Repeat("x", 50)):
		t.Errorf("✗ APIErr(long body) dropped the body content — a bounded snippet prefix must remain")
	}

	if !t.Failed() {
		t.Log("✓ APIErr renders message or snippet with status and label under the response-status sentinel")
	}
}

// FuzzAPIErrBody verifies invariant #2: API error classification under arbitrary bodies.
//
// What is being tested:
// For arbitrary response bytes, APIErr must return ErrResponseStatus. If it also returns
// ErrResponseServer, its text must start with quoted context; otherwise its diagnostic text must
// stay within the tested excerpt bound.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzAPIErrBody(f *testing.F) {
	const diagnosticExcerptBytes = 200

	f.Add([]byte(`{"error":{"message":"nope"}}`))
	f.Add([]byte(`{"detail":"over capacity"}`))
	f.Add([]byte(`{"detail":[{"msg":"a"},{"msg":"b"}]}`))
	f.Add([]byte(`{"detail":[{"msg":1},{"loc":[]}]}`))
	f.Add([]byte(`{"error":"flat"}`))
	f.Add([]byte("<html>boom</html>"))
	f.Add([]byte(""))
	f.Add([]byte("\x00\xff\x9b"))

	f.Fuzz(func(t *testing.T, body []byte) {
		err := APIErr("prov (m)", 500, body)
		if err == nil || !errors.Is(err, errs.ErrResponseStatus) {
			t.Errorf("✗ APIErr(%q) = %v, want a response-status classification", body, err)

			return
		}

		if errors.Is(err, errs.ErrResponseServer) {
			if _, quoteErr := strconv.QuotedPrefix(err.Error()); quoteErr != nil {
				t.Errorf("✗ a server-message mark without a quoted context: %q", err.Error())
			}
		} else if len(err.Error()) > diagnosticExcerptBytes*4+100 {
			// Response excerpts are limited to 200 bytes before quoting; %q escaping
			// expands a byte to at most four characters, and the label, status, and
			// sentinel text ride alongside.
			t.Errorf("✗ an unextracted body rendered %d chars, want a bounded excerpt", len(err.Error()))
		}
	})
}

// errLen returns the error text length, or zero for nil.
func errLen(test testing.TB, err error) int {
	test.Helper()

	if err == nil {
		return 0
	}

	return len(err.Error())
}
