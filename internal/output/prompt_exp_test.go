package output

import (
	"errors"
	"io"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// Invariants tested:
//  1. Prompt delivery causes: When the destination writer returns io.ErrClosedPipe,
//     PrintSingleWordPromptConfirmation and PrintReprompt must return errors matching both that
//     cause and ErrOutputFileWrite.

// TestPromptFormattingFailure verifies invariant #1: Prompt delivery causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When the destination writer returns io.ErrClosedPipe, PrintSingleWordPromptConfirmation and
// PrintReprompt must return errors matching both that cause and ErrOutputFileWrite.
// Kind: permanent.
func TestPromptFormattingFailure(t *testing.T) {
	confirmationErr := PrintSingleWordPromptConfirmation(failingJSONWriter{test: t}, "boat", false)

	repromptErr := PrintReprompt(failingJSONWriter{test: t}, false)
	for _, err := range []error{confirmationErr, repromptErr} {
		if !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, errs.ErrOutputFileWrite) {
			t.Errorf("✗ prompt formatting lost delivery causes: %v", err)
		}
	}

	if !t.Failed() {
		t.Log("✓ prompt formatters preserve output failures")
	}
}
