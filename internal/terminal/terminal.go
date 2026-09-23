// Package terminal owns interactive input, terminal capabilities, and animation.
package terminal

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether the supplied stream is an actual terminal file. Non-file readers and
// writers, nil files, pipes, and ordinary devices return false.
func IsTerminal(stream any) bool {
	file, isFile := stream.(*os.File)

	return isFile && file != nil && term.IsTerminal(int(file.Fd()))
}
