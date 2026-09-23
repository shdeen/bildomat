package output

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"golang.org/x/term"
)

// Invariants tested:
//  1. Single-word prompt confirmation: With terminal styling disabled,
//     PrintSingleWordPromptConfirmation must write the configured confirmation text for hellp plus
//     one trailing space to stderr, and write nothing to stdout.
//  2. Prompt styling plain off terminal: With stdin connected to the null device and output
//     captured through a pipe, PrintSingleWordPromptConfirmation, PrintAmbiguity, and PrintReprompt
//     must write no ANSI escape characters to stderr.

// TestPrintSingleWordPromptConfirmation verifies invariant #1: Single-word prompt confirmation.
//
// What is being tested:
// With terminal styling disabled, PrintSingleWordPromptConfirmation must write the configured
// confirmation text for hellp plus one trailing space to stderr, and write nothing to stdout.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintSingleWordPromptConfirmation(t *testing.T) {
	originalInput := os.Stdin

	emptyInput, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("💣 empty prompt input: %v", err)
	}

	os.Stdin = emptyInput

	t.Cleanup(func() { os.Stdin = originalInput; _ = emptyInput.Close() })

	var stderr string

	stdout := captureStdout(t, func() {
		stderr = captureStderr(t, func() {
			_ = PrintSingleWordPromptConfirmation(os.Stderr, "hellp", term.IsTerminal(int(os.Stderr.Fd())))
		})
	})

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty", stdout)
	}

	want := fmt.Sprintf(SingleWordConfirmation, "", "", "", "hellp") + " "
	if stderr != want {
		t.Errorf("✗ confirmation = %q, want %q", stderr, want)
	}

	if !t.Failed() {
		t.Log("✓ the embedded one-word confirmation copy renders only on stderr")
	}
}

// TestPromptStylingPlainOffTerminal verifies invariant #2: Prompt styling plain off terminal.
//
// What is being tested:
// With stdin connected to the null device and output captured through a pipe,
// PrintSingleWordPromptConfirmation, PrintAmbiguity, and PrintReprompt must write no ANSI escape
// characters to stderr.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPromptStylingPlainOffTerminal(t *testing.T) {
	originalInput := os.Stdin

	emptyInput, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("💣 empty prompt input: %v", err)
	}

	os.Stdin = emptyInput

	t.Cleanup(func() { os.Stdin = originalInput; _ = emptyInput.Close() })
	stderr := captureStderr(t, func() {
		_ = PrintSingleWordPromptConfirmation(os.Stderr, "hellp", term.IsTerminal(int(os.Stderr.Fd())))
		_ = PrintAmbiguity(os.Stderr, "dup-model", nil, term.IsTerminal(int(os.Stderr.Fd())))
		_ = PrintReprompt(os.Stderr, term.IsTerminal(int(os.Stderr.Fd())))
	})

	if strings.Contains(stderr, "\x1b") {
		t.Errorf("✗ a prompt carried an ANSI escape off a terminal: %q", stderr)
	}

	if !t.Failed() {
		t.Log("✓ the interactive prompts render plain when standard error is not a terminal")
	}
}
