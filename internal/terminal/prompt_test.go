package terminal

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// Invariants tested:
// 1. Case-independent cancellation: For a one-word prompt on a terminal, ConfirmOneWordPrompt must
//    display a prompt and cancel on n or no in any capitalization.

// TestConfirmationCancellationCase verifies invariant #1: Case-independent cancellation.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// For a one-word prompt on a terminal, ConfirmOneWordPrompt must display a prompt and cancel on n
// or no in any capitalization. Other replies must continue; every case must leave the second input
// line unread.
//
// Kind: permanent.
func TestConfirmationCancellationCase(t *testing.T) {
	for _, reply := range []string{"n", "N", "no", "No", "nO", "NO", "yes", "nope", ""} {
		t.Run("reply="+reply, func(t *testing.T) {
			input, writer, err := os.Pipe()
			if err != nil {
				t.Fatalf("💣 reply pipe: %v", err)
			}

			t.Cleanup(func() { _ = input.Close() })

			if _, err := io.WriteString(writer, reply+"\nremaining reply\n"); err != nil {
				t.Fatalf("💣 write reply: %v", err)
			}

			if err := writer.Close(); err != nil {
				t.Fatalf("💣 close reply writer: %v", err)
			}

			var diagnostics bytes.Buffer

			canceled, err := ConfirmOneWordPrompt(input, &diagnostics, "boat", true, false)

			cancelExpected := strings.EqualFold(reply, "n") || strings.EqualFold(reply, "no")
			if err != nil || canceled != cancelExpected || diagnostics.Len() == 0 {
				t.Errorf("✗ reply %q: canceled %t, error %v, prompt %q", reply, canceled, err, diagnostics.String())
			}

			remainingReply, err := io.ReadAll(input)
			if err != nil || string(remainingReply) != "remaining reply\n" {
				t.Errorf("✗ confirmation consumed another reply: %q, %v", remainingReply, err)
			}

			if !t.Failed() {
				t.Log("✓ cancellation is case-independent and leaves the next reply intact")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ all cancellation spellings and continuation controls hold")
	}
}
