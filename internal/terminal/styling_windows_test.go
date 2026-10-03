//go:build windows

package terminal

// Invariants tested:
//  1. Console styling enabled: EnableStyling must report true for the console output device whose
//     virtual terminal processing is off, and leave that processing on.
//  2. No styling off a console: EnableStyling must report false for a pipe end and a regular file.

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

// TestEnableStylingWindowsConsole verifies invariant #1: Console styling enabled.
//
// What makes it or breaks it:
// The test turns virtual terminal processing off on the console output device. EnableStyling must
// then report true for that device, and the console output mode read afterwards must carry the
// processing flag.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// Kind: permanent.
func TestEnableStylingWindowsConsole(t *testing.T) {
	consoleOutput := openConsoleDevice(t, consoleOutputDevice)
	outputHandle := windows.Handle(consoleOutput.Fd())

	var originalMode uint32
	if err := windows.GetConsoleMode(outputHandle, &originalMode); err != nil {
		t.Fatalf("💣 read the console output mode: %v", err)
	}

	t.Cleanup(func() { _ = windows.SetConsoleMode(outputHandle, originalMode) })

	if err := windows.SetConsoleMode(outputHandle, originalMode&^windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		t.Fatalf("💣 turn virtual terminal processing off: %v", err)
	}

	if !EnableStyling(consoleOutput) {
		t.Error("✗ EnableStyling = false for the console output device, want true")
	}

	var enabledMode uint32
	if err := windows.GetConsoleMode(outputHandle, &enabledMode); err != nil {
		t.Fatalf("💣 read the console output mode after enabling: %v", err)
	}

	if enabledMode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
		t.Errorf("✗ virtual terminal processing is off after EnableStyling: mode %#x", enabledMode)
	}

	if !t.Failed() {
		t.Log("✓ the console renders styling after EnableStyling")
	}
}

// TestEnableStylingOffConsole verifies invariant #2: No styling off a console.
//
// What makes it or breaks it:
// EnableStyling must report false for the read end of a pipe and for a temporary regular file.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// Kind: permanent.
func TestEnableStylingOffConsole(t *testing.T) {
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
	}{
		{label: "pipe", stream: pipeReadEnd},
		{label: "regular file", stream: regular},
	} {
		if EnableStyling(streamCase.stream) {
			t.Errorf("✗ %s: EnableStyling = true, want false", streamCase.label)
		}
	}

	if !t.Failed() {
		t.Log("✓ a redirected stream gets no styling")
	}
}
