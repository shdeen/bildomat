//go:build !windows

package terminal

// Invariants tested:
//  1. Styling on a terminal: EnableStyling must report true for a terminal and false for a pipe
//     end and a regular file, agreeing with IsTerminal on every platform other than Windows.

import (
	"os"
	"testing"
)

// TestEnableStylingTerminal verifies invariant #1: Styling on a terminal.
//
// What makes it or breaks it:
// With a terminal, a pipe, and a temporary regular file, EnableStyling must report true for the
// terminal's file and false for the pipe's read end and the regular file, and each answer must
// equal IsTerminal's for the same stream.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// Kind: permanent.
func TestEnableStylingTerminal(t *testing.T) {
	terminalStreams := openTerminal(t)

	pipeReadEnd, pipeWriteEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 os.Pipe: %v", err)
	}

	t.Cleanup(func() { _ = pipeReadEnd.Close(); _ = pipeWriteEnd.Close() })

	regular, err := os.CreateTemp(t.TempDir(), "stream")
	if err != nil {
		t.Fatalf("💣 regular stream: %v", err)
	}

	t.Cleanup(func() { _ = regular.Close() })

	for _, streamCase := range []struct {
		label  string
		stream *os.File
		want   bool
	}{
		{label: "terminal", stream: terminalStreams.Output(), want: true},
		{label: "pipe", stream: pipeReadEnd, want: false},
		{label: "regular file", stream: regular, want: false},
	} {
		got := EnableStyling(streamCase.stream)
		if got != streamCase.want {
			t.Errorf("✗ %s: EnableStyling = %t, want %t", streamCase.label, got, streamCase.want)
		}

		if detected := IsTerminal(streamCase.stream); got != detected {
			t.Errorf("✗ %s: EnableStyling = %t but IsTerminal = %t", streamCase.label, got, detected)
		}
	}

	if !t.Failed() {
		t.Log("✓ styling is enabled exactly on a terminal")
	}
}
