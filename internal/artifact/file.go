// Package artifact owns generated media, output paths, persistence, and cleanup.
package artifact

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/shdeen/bildomat/internal/errs"
)

// SavedFile describes a successfully closed output file.
//   - Path: the saved destination path
//   - Bytes: the number of bytes written
type SavedFile struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// claimFile exclusively creates the requested filename or its first free numbered variant. It
// returns the open file and its path; the extension remains unchanged.
func claimFile(dir, name, ext string) (*os.File, string, error) {
	for suffix := 1; ; suffix++ {
		candidate := name
		if suffix > 1 {
			candidate = suffixedName(name, suffix)
		}

		path := filepath.Join(dir, candidate+ext)

		file, err := openFile(path)
		if err == nil {
			return file, path, nil
		}

		if !errors.Is(err, os.ErrExist) {
			return nil, "", errs.FileError(errs.FileOpWrite, path, errs.ErrOutputFileWrite, err)
		}
	}
}

// openFile creates a new file without replacing any existing path.
func openFile(path string) (*os.File, error) {
	// #nosec G302 G304 -- output paths belong to the invoking user and follow the user's umask.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, errs.FileError(errs.FileOpCreate, path, errs.ErrOutputFileCreate, err)
	}

	return file, nil
}

// commitFile writes and closes a claimed destination, returning its path and byte count. A write or
// close failure removes the incomplete destination.
func commitFile(file *os.File, path string, data []byte) (SavedFile, error) {
	written, writeErr := file.Write(data)
	if writeErr == nil && written != len(data) {
		writeErr = io.ErrShortWrite
	}

	return closeOutput(file, path, int64(written), writeErr)
}

// copyFile copies a source into a claimed destination and closes both files. It preserves the
// source and removes an incomplete destination on failure.
func copyFile(file *os.File, path, source string) (SavedFile, error) {
	// #nosec G304 -- source is a generated temporary file or the user's selected input.
	input, err := os.Open(source)
	if err != nil {
		return closeOutput(file, path, 0, err)
	}

	written, copyErr := io.Copy(file, input)
	inputCloseErr := input.Close()

	saved, outputErr := closeOutput(file, path, written, copyErr)
	if inputCloseErr != nil {
		outputErr = errors.Join(outputErr, errs.FileError(errs.FileOpClose, source, errs.ErrOutputFileClose, inputCloseErr))
	}

	return saved, outputErr
}

// Write exclusively creates and writes one file under the first free name.
func Write(dir, name, ext string, data []byte) (SavedFile, error) {
	file, path, err := claimFile(dir, name, ext)
	if err != nil {
		return SavedFile{}, err
	}

	return commitFile(file, path, data)
}

// closeOutput closes the destination and removes only this attempt's incomplete file on failure.
// Both operation and cleanup errors remain discoverable.
func closeOutput(file *os.File, path string, written int64, writeErr error) (SavedFile, error) {
	closeErr := file.Close()
	if writeErr == nil && closeErr == nil {
		return SavedFile{Path: path, Bytes: written}, nil
	}

	var operationErr error

	cleanupClass := errs.ErrOutputFileClose

	if writeErr != nil {
		cleanupClass = errs.ErrOutputFileWrite
		operationErr = errs.FileError(errs.FileOpWrite, path, errs.ErrOutputFileWrite, writeErr)
	}

	if closeErr != nil {
		operationErr = errors.Join(operationErr, errs.FileError(errs.FileOpClose, path, errs.ErrOutputFileClose, closeErr))
	}

	return SavedFile{}, errors.Join(operationErr, removeFile(path, cleanupClass))
}

// removeFile removes an owned file, retaining the operation's classification and the original
// removal cause. An already absent file needs no more cleanup.
func removeFile(path string, classification error) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return errs.FileError(errs.FileOpCleanup, path, classification, err)
}
