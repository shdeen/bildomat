package errs

import (
	"errors"
	"os"
)

// File operation labels identify the action reported by a filesystem failure.
// Cleanup and expansion describe application operations; the remaining labels
// match the corresponding filesystem operations.
//   - FileOpCleanup: remove an owned temporary or incomplete file
//   - FileOpClose: close a file after writing
//   - FileOpCreate: exclusively create a destination
//   - FileOpExpand: resolve a home-relative path
//   - FileOpMkdir: create an output directory
//   - FileOpWrite: write destination bytes
const (
	FileOpCleanup = "cleanup"
	FileOpClose   = "close"
	FileOpCreate  = "create"
	FileOpExpand  = "expand"
	FileOpMkdir   = "mkdir"
	FileOpWrite   = "write"
)

// FileError retains the affected path, operation classification, and original
// cause in a standard filesystem error that callers can inspect with errors.As.
func FileError(operation, path string, classification, cause error) *os.PathError {
	return &os.PathError{Op: operation, Path: path, Err: errors.Join(classification, cause)}
}
