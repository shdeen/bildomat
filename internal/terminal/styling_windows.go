//go:build windows

package terminal

import (
	"os"

	"golang.org/x/sys/windows"
)

// EnableStyling reports whether the supplied stream is a terminal that renders styling escape
// sequences, enabling the console's virtual terminal processing where it is off. It reports false
// for a stream that is not a terminal, and for a console whose processing cannot be enabled, so
// that the caller falls back to plain output.
func EnableStyling(stream any) bool {
	if !IsTerminal(stream) {
		return false
	}

	file, isFile := stream.(*os.File)
	if !isFile {
		return false
	}

	console := windows.Handle(file.Fd())

	var mode uint32
	if err := windows.GetConsoleMode(console, &mode); err != nil {
		return false
	}

	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}

	return windows.SetConsoleMode(console, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
