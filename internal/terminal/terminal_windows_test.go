//go:build windows

package terminal

// Invariants tested:
//  1. Windows console detection: IsTerminal must report true for the console input and output
//     devices and false for a pipe end and a regular file.

import (
	"os"
	"syscall"
	"testing"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The console devices and the key event values the fixture types with.
//   - keyEventType: the INPUT_RECORD type of a key event
//   - returnKeyCode: the virtual key of Enter
//   - endOfInputCharacter: Ctrl-Z, which ends console input the way a closed pipe does
const (
	consoleOutputDevice = "CONOUT$"
	consoleInputDevice  = "CONIN$"
	keyEventType        = 1
	returnKeyCode       = 0x0D
	endOfInputCharacter = 0x1A
)

// TestIsTerminalWindowsConsole verifies invariant #1: Windows console detection.
//
// What makes it or breaks it:
// The process has a console attached, either already or because the test attaches one, and the
// console output and input devices open. IsTerminal must return true for both device files and
// false for the read end of a pipe and for a temporary regular file.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestIsTerminalWindowsConsole(t *testing.T) {
	console := openTerminal(t)
	consoleOutput := console.Output()
	consoleInput := console.Input()

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
		{label: "console output", stream: consoleOutput, want: true},
		{label: "console input", stream: consoleInput, want: true},
		{label: "pipe", stream: pipeReadEnd, want: false},
		{label: "regular file", stream: regular, want: false},
	} {
		if got := IsTerminal(streamCase.stream); got != streamCase.want {
			t.Errorf("✗ %s: IsTerminal = %t, want %t", streamCase.label, got, streamCase.want)
		}
	}

	if !t.Failed() {
		t.Log("✓ the Windows console devices are terminals and redirected streams are not")
	}
}

// keyInputRecord is the INPUT_RECORD of one key event, laid out as the console API expects.
type keyInputRecord struct {
	eventType       uint16
	_               uint16
	keyDown         int32
	repeatCount     uint16
	virtualKeyCode  uint16
	virtualScanCode uint16
	unicodeChar     uint16
	controlKeyState uint32
}

// terminalFixture is the console the code under test sees on its streams: the console input
// device as its input and the console output device as its output.
type terminalFixture struct {
	input, output *os.File
}

// openTerminal opens the console devices, attaching a console to the process when it has none,
// and prepares the input mode. The devices close when the test ends.
func openTerminal(test testing.TB) *terminalFixture {
	test.Helper()

	fixture := &terminalFixture{input: openConsoleDevice(test, consoleInputDevice), output: openConsoleDevice(test, consoleOutputDevice)}
	fixture.prepareInput(test)

	return fixture
}

// Input returns the file the program reads as a terminal.
func (fixture *terminalFixture) Input() *os.File { return fixture.input }

// Output returns the file the program writes as a terminal.
func (fixture *terminalFixture) Output() *os.File { return fixture.output }

// TypeReply discards any pending input events, then supplies text as typed key events followed by
// the end of input, so that a read past the text sees EOF as it would on a closed pipe. The
// discard happens here rather than when a fixture opens, because a later fixture on the same
// console must not delete a reply already queued.
func (fixture *terminalFixture) TypeReply(test testing.TB, text string) {
	test.Helper()

	if err := windows.FlushConsoleInputBuffer(windows.Handle(fixture.input.Fd())); err != nil {
		test.Fatalf("💣 discard pending console input: %v", err)
	}

	typed := utf16.Encode([]rune(text + string(rune(endOfInputCharacter)) + "\n"))
	records := make([]keyInputRecord, 0, 2*len(typed))

	for _, unit := range typed {
		record := keyInputRecord{eventType: keyEventType, keyDown: 1, repeatCount: 1, unicodeChar: unit}
		if unit == '\n' {
			record.unicodeChar = '\r'
			record.virtualKeyCode = returnKeyCode
		}

		records = append(records, record)
		record.keyDown = 0
		records = append(records, record)
	}

	var written uint32

	// #nosec G103 -- the records and the count stay addressable for the duration of the call.
	ok, _, err := kernel32Proc("WriteConsoleInputW").Call(fixture.input.Fd(), uintptr(unsafe.Pointer(&records[0])), uintptr(len(records)), uintptr(unsafe.Pointer(&written)))
	if ok == 0 || int(written) != len(records) {
		test.Fatalf("💣 type the reply into the console: %d of %d events written: %v", written, len(records), err)
	}
}

// prepareInput keeps line input, so a read returns at Enter, and turns echo off. It leaves queued
// input alone.
func (fixture *terminalFixture) prepareInput(test testing.TB) {
	test.Helper()

	inputHandle := windows.Handle(fixture.input.Fd())

	var mode uint32
	if err := windows.GetConsoleMode(inputHandle, &mode); err != nil {
		test.Fatalf("💣 read the console input mode: %v", err)
	}

	if err := windows.SetConsoleMode(inputHandle, mode&^windows.ENABLE_ECHO_INPUT); err != nil {
		test.Fatalf("💣 turn console echo off: %v", err)
	}
}

// openConsoleDevice opens a console device of the process, attaching a console first when the
// process has none. The file closes when the test ends.
func openConsoleDevice(test testing.TB, device string) *os.File {
	test.Helper()

	// #nosec G304 -- device is one of the two console device names.
	file, openErr := os.OpenFile(device, os.O_RDWR, 0)
	if openErr != nil {
		if ok, _, callErr := kernel32Proc("AllocConsole").Call(); ok == 0 {
			test.Fatalf("💣 open %s: %v; attaching a console also failed: %v", device, openErr, callErr)
		}

		// #nosec G304 -- device is one of the two console device names.
		file, openErr = os.OpenFile(device, os.O_RDWR, 0)
		if openErr != nil {
			test.Fatalf("💣 open %s after attaching a console: %v", device, openErr)
		}
	}

	test.Cleanup(func() { _ = file.Close() })

	return file
}

// kernel32Proc returns a console API entry point of kernel32 that x/sys/windows does not wrap.
func kernel32Proc(name string) *syscall.LazyProc {
	return syscall.NewLazyDLL("kernel32.dll").NewProc(name)
}
