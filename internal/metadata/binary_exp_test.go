package metadata

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
)

// Invariants tested:
// 1. Owned media read failure: For a missing media file, binaryStore.copyFile and
//    binaryStore.matchArtifacts must return os.ErrNotExist through an os.PathError whose operation
//    is read and whose path is the supplied filename.

// TestOwnedMediaReadFailure verifies invariant #1: Owned media read failure.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// For a missing media file, binaryStore.copyFile and binaryStore.matchArtifacts must return
// os.ErrNotExist through an os.PathError whose operation is read and whose path is the supplied
// filename. Neither error may match ErrInputMedia.
//
// Kind: permanent.
func TestOwnedMediaReadFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.png")
	store := binaryStore{}
	_, downloadErr := store.copyFile(path, "image/png")

	_, artifactErr := store.matchArtifacts([]artifact.SavedFile{{Path: path}})
	for _, readErr := range []error{downloadErr, artifactErr} {
		var fileErr *os.PathError
		if !errors.Is(readErr, os.ErrNotExist) || errors.Is(readErr, errs.ErrInputMedia) || !errors.As(readErr, &fileErr) {
			t.Errorf("✗ owned media read lost its filesystem classification: %v", readErr)

			continue
		}

		if fileErr.Path != path || fileErr.Op != "read" {
			t.Errorf("✗ owned media read context: %q, %q", fileErr.Op, fileErr.Path)
		}
	}

	if !t.Failed() {
		t.Log("✓ owned media failures preserve path and cause without blaming input")
	}
}
