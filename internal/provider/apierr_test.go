package provider

// Invariants tested:
//  1. Provider server messages: Given the listed provider error shapes, APIErr must return
//     ErrResponseStatus and ErrResponseServer with the exact extracted message quoted in its text,
//     preferring a nonempty nested message.

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestAPIErrServerMessage verifies invariant #1: Provider server messages.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// What is being tested:
// Given the listed provider error shapes, APIErr must return ErrResponseStatus and
// ErrResponseServer with the exact extracted message quoted in its text, preferring a nonempty
// nested message. For an HTML body, it must retain ErrResponseStatus without ErrResponseServer.
func TestAPIErrServerMessage(t *testing.T) {
	extractedCases := []struct {
		name string
		body string
		want string
	}{
		{
			"an error object carrying error.message",
			`{"error":{"message":"rate limited","code":429}}`,
			"rate limited",
		},
		{
			"a top-level detail string",
			`{"detail":"/v1/fixture-video is over capacity. Please retry shortly."}`,
			"/v1/fixture-video is over capacity. Please retry shortly.",
		},
		{
			"a validation failure carrying a detail array of msg objects",
			`{"detail":[{"loc":["body","width"],"msg":"value is not a multiple of 32"},{"msg":"second fault"}]}`,
			"value is not a multiple of 32; second fault",
		},
		{
			"a top-level message",
			`{"code":1201,"message":"request parameter error"}`,
			"request parameter error",
		},
		{
			"an error document wrapped in a one-element array",
			`[{"error":{"code":400,"message":"API key not valid. Please pass a valid API key.","status":"INVALID_ARGUMENT"}}]`,
			"API key not valid. Please pass a valid API key.",
		},
		{
			"nested error message precedes a top-level message",
			`{"error":{"message":"nested message"},"message":"top-level message"}`,
			"nested message",
		},
		{
			"an empty nested message allows the top-level message",
			`{"error":{"message":""},"message":"top-level message"}`,
			"top-level message",
		},
	}
	for _, c := range extractedCases {
		t.Run(c.name, func(t *testing.T) {
			err := APIErr("prov (m)", 400, []byte(c.body))
			if err == nil || !errors.Is(err, errs.ErrResponseStatus) {
				t.Fatalf("💣 APIErr = %v, want an ErrResponseStatus error to inspect", err)
			}

			if !errors.Is(err, errs.ErrResponseServer) {
				t.Errorf("✗ the extracted message is not marked with the server-message sentinel: %v", err)
			}

			if !strings.Contains(err.Error(), strconv.Quote(c.want)) {
				t.Errorf("✗ the chain %q does not carry the extracted text %q as a quoted context", err.Error(), c.want)
			}

			if !t.Failed() {
				t.Log("✓ APIErr preserves the documented server message")
			}
		})
	}

	unextracted := APIErr("prov (m)", 500, []byte("<html>gateway timeout</html>"))
	if errors.Is(unextracted, errs.ErrResponseServer) {
		t.Errorf("✗ a body with no documented shape must not carry the server-message mark: %v", unextracted)
	}

	if !errors.Is(unextracted, errs.ErrResponseStatus) {
		t.Errorf("✗ the unextracted failure lost its response-status classification: %v", unextracted)
	}

	if !t.Failed() {
		t.Log("✓ APIErr extracts and marks the documented server-message shapes and leaves undocumented bodies unmarked")
	}
}
