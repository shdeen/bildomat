//go:build windows

package main

// Invariants tested:
// This file contains test support only and no test or fuzz function.

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The console devices, the screen buffer size the fixture sets so that no line of program output
// wraps across rows, and the key event values the fixture types with.
//   - keyEventType: the INPUT_RECORD type of a key event
//   - returnKeyCode: the virtual key of Enter
//   - endOfInputCharacter: Ctrl-Z, which ends console input the way a closed pipe does
const (
	consoleOutputDevice = "CONOUT$"
	consoleInputDevice  = "CONIN$"
	consoleColumns      = 1000
	consoleRows         = 3000
	keyEventType        = 1
	returnKeyCode       = 0x0D
	endOfInputCharacter = 0x1A
)

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

// terminalFixture is the console the program sees on its streams: the console input device as its
// input, and the console output device, a screen buffer the test reads back, as its output. Opening
// the fixture clears the screen buffer, so a read returns only what the program wrote afterwards.
//
// Test class: Core: Helper.
type terminalFixture struct {
	input, output     *os.File
	outputHandle      windows.Handle
	defaultAttributes uint16
}

// openTerminal opens the console devices, attaching a console to the process when it has none,
// and prepares the screen buffer and the input mode. The devices close when the test ends.
//
// Test class: Core: Helper.
func openTerminal(test testing.TB) *terminalFixture {
	test.Helper()

	output := openConsoleDevice(test, consoleOutputDevice)
	fixture := &terminalFixture{input: openConsoleDevice(test, consoleInputDevice), output: output, outputHandle: windows.Handle(output.Fd())}

	fixture.prepareScreen(test)
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

// ReadOutput returns everything the program wrote to the screen buffer, row by row, without the
// trailing blanks of each row.
func (fixture *terminalFixture) ReadOutput(test testing.TB) string {
	test.Helper()

	return strings.Join(fixture.screenRows(test), "\n")
}

// Styled reports whether the program wrote text with character attributes that differ from the
// screen buffer's default, which is how rendered styling shows on a console.
func (fixture *terminalFixture) Styled(test testing.TB, text string) bool {
	test.Helper()

	for rowIndex, row := range fixture.screenRows(test) {
		before, _, found := strings.Cut(row, text)
		if !found {
			continue
		}

		// #nosec G115 -- the screen buffer holds at most consoleColumns by consoleRows cells, within int16.
		column, rowNumber := int16(len(utf16.Encode([]rune(before)))), int16(rowIndex)
		for _, attributes := range fixture.readAttributes(test, column, rowNumber, len(utf16.Encode([]rune(text)))) {
			if attributes != fixture.defaultAttributes {
				return true
			}
		}

		return false
	}

	return false
}

// prepareScreen records the default attributes, widens the screen buffer so no line wraps, and
// clears it so that reads return only this test's output.
func (fixture *terminalFixture) prepareScreen(test testing.TB) {
	test.Helper()

	info := fixture.screenInfo(test)
	fixture.defaultAttributes = info.Attributes

	if info.Size.X < consoleColumns || info.Size.Y < consoleRows {
		if ok, _, err := kernel32Proc("SetConsoleScreenBufferSize").Call(uintptr(fixture.outputHandle), coordArgument(windows.Coord{X: consoleColumns, Y: consoleRows})); ok == 0 {
			test.Fatalf("💣 widen the screen buffer: %v", err)
		}

		info = fixture.screenInfo(test)
	}

	// #nosec G115 -- a screen buffer dimension is never negative.
	cells := uintptr(info.Size.X) * uintptr(info.Size.Y)

	var filled uint32

	// #nosec G103 -- the count stays addressable for the duration of each call.
	if ok, _, err := kernel32Proc("FillConsoleOutputCharacterW").Call(uintptr(fixture.outputHandle), ' ', cells, coordArgument(windows.Coord{}), uintptr(unsafe.Pointer(&filled))); ok == 0 {
		test.Fatalf("💣 clear the screen buffer: %v", err)
	}

	// #nosec G103 -- the count stays addressable for the duration of each call.
	if ok, _, err := kernel32Proc("FillConsoleOutputAttribute").Call(uintptr(fixture.outputHandle), uintptr(info.Attributes), cells, coordArgument(windows.Coord{}), uintptr(unsafe.Pointer(&filled))); ok == 0 {
		test.Fatalf("💣 reset the screen buffer attributes: %v", err)
	}

	if ok, _, err := kernel32Proc("SetConsoleCursorPosition").Call(uintptr(fixture.outputHandle), coordArgument(windows.Coord{})); ok == 0 {
		test.Fatalf("💣 home the cursor: %v", err)
	}
}

// prepareInput keeps line input, so a read returns at Enter, and turns echo off so typed replies
// do not reach the screen buffer. It leaves queued input alone.
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

// screenInfo returns the screen buffer's size, cursor position, and default attributes.
func (fixture *terminalFixture) screenInfo(test testing.TB) windows.ConsoleScreenBufferInfo {
	test.Helper()

	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(fixture.outputHandle, &info); err != nil {
		test.Fatalf("💣 read the screen buffer information: %v", err)
	}

	return info
}

// screenRows returns the rows from the top of the screen buffer through the cursor's row, each
// without its trailing blanks.
func (fixture *terminalFixture) screenRows(test testing.TB) []string {
	test.Helper()

	info := fixture.screenInfo(test)
	rows := make([]string, 0, info.CursorPosition.Y+1)

	for row := int16(0); row <= info.CursorPosition.Y; row++ {
		cells := make([]uint16, info.Size.X)

		var read uint32

		// #nosec G103 -- the cells and the count stay addressable for the duration of the call.
		if ok, _, err := kernel32Proc("ReadConsoleOutputCharacterW").Call(uintptr(fixture.outputHandle), uintptr(unsafe.Pointer(&cells[0])), uintptr(len(cells)), coordArgument(windows.Coord{Y: row}), uintptr(unsafe.Pointer(&read))); ok == 0 {
			test.Fatalf("💣 read screen buffer row %d: %v", row, err)
		}

		rows = append(rows, strings.TrimRight(string(utf16.Decode(cells[:read])), " "))
	}

	return rows
}

// readAttributes returns the character attributes of count cells starting at the given cell.
func (fixture *terminalFixture) readAttributes(test testing.TB, column, row int16, count int) []uint16 {
	test.Helper()

	attributes := make([]uint16, count)

	var read uint32

	// #nosec G103 -- the attributes and the count stay addressable for the duration of the call.
	if ok, _, err := kernel32Proc("ReadConsoleOutputAttribute").Call(uintptr(fixture.outputHandle), uintptr(unsafe.Pointer(&attributes[0])), uintptr(count), coordArgument(windows.Coord{X: column, Y: row}), uintptr(unsafe.Pointer(&read))); ok == 0 {
		test.Fatalf("💣 read screen buffer attributes at row %d: %v", row, err)
	}

	return attributes[:read]
}

// openConsoleDevice opens a console device of the process, attaching a console first when the
// process has none. The file closes when the test ends.
//
// Test class: Core: Helper.
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

// coordArgument packs a console coordinate the way the API passes a COORD by value: two signed
// 16-bit fields in one 32-bit word, each reinterpreted bit for bit.
func coordArgument(coordinate windows.Coord) uintptr {
	// #nosec G115 -- the reinterpretation of each signed field's bits is the packing itself.
	return uintptr(uint32(uint16(coordinate.X)) | uint32(uint16(coordinate.Y))<<16)
}
