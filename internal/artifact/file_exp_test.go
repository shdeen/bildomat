package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
// 1. Sidecar write failure: Given a missing destination directory, Write must return
//    ErrOutputFileWrite.
// 2. Sidecar exclusive claim: When two artifact runs share a stem, Write must give their sidecars
//    distinct paths and preserve both sidecar contents.
// 3. Incomplete output files: Given a closed destination, commitFile must return
//    ErrOutputFileWrite, ErrOutputFileClose, and os.ErrClosed.
// 4. Close failure with denied removal: Given a closed file and directory permissions that prevent
//    removal, closeOutput must return ErrOutputFileClose, os.ErrClosed, and os.ErrPermission with
//    the destination path.

// TestWriteSidecarError verifies invariant #1: Sidecar write failure.
//
// What is being tested:
// Given a missing destination directory, Write must return ErrOutputFileWrite.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWriteSidecarError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir")

	_, err := Write(missing, "stem", SidecarExt, []byte("x"))
	if err == nil || !errors.Is(err, errs.ErrOutputFileWrite) {
		t.Errorf("✗ a sidecar write into a missing dir = %v, want ErrOutputFileWrite", err)
	}

	if !t.Failed() {
		t.Log("✓ a failed sidecar write wraps its sentinel")
	}
}

// TestSidecarNeverReplaces verifies invariant #2: Sidecar exclusive claim.
//
// What is being tested:
// When two artifact runs share a stem, Write must give their sidecars distinct paths and preserve
// both sidecar contents.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSidecarNeverReplaces(t *testing.T) {
	dir := t.TempDir()

	firstStem, _, err := WriteMedia(dir, "study", media.Image, []Media{{Data: []byte("PNG"), FileExt: ".png"}}, false)
	if err != nil {
		t.Fatalf("💣 first WriteMedia: %v", err)
	}

	secondStem, _, err := WriteMedia(dir, "study", media.Image, []Media{{Data: []byte("JPG"), FileExt: ".jpg"}}, false)
	if err != nil {
		t.Fatalf("💣 second WriteMedia: %v", err)
	}

	firstSidecar, err := Write(dir, firstStem, SidecarExt, []byte("first thoughts"))
	if err != nil {
		t.Fatalf("💣 first WriteSidecar: %v", err)
	}

	secondSidecar, err := Write(dir, secondStem, SidecarExt, []byte("second thoughts"))
	if err != nil {
		t.Fatalf("💣 second WriteSidecar: %v", err)
	}

	if firstSidecar.Path == secondSidecar.Path {
		t.Errorf("✗ both sidecars report the same path %q", firstSidecar.Path)
	}

	for _, sidecar := range []struct {
		path, content string
	}{{firstSidecar.Path, "first thoughts"}, {secondSidecar.Path, "second thoughts"}} {
		// #nosec G304 -- the path is the report of a write into this test's temporary directory.
		got, readErr := os.ReadFile(sidecar.path)
		if readErr != nil || string(got) != sidecar.content {
			t.Errorf("✗ %s = (%q, %v), want %q", sidecar.path, got, readErr, sidecar.content)
		}
	}

	if !t.Failed() {
		t.Log("✓ a second sidecar under a shared stem lands beside the first instead of replacing it")
	}
}

// TestIncompleteOutput verifies invariant #3: Incomplete output files.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a closed destination, commitFile must return ErrOutputFileWrite, ErrOutputFileClose, and
// os.ErrClosed. Given a missing source, copyFile must return os.ErrNotExist. Both must remove the
// incomplete destination and return no saved-file result.
//
// Kind: permanent.
func TestIncompleteOutput(t *testing.T) {
	for _, failure := range []string{"closed destination", "missing source"} {
		t.Run(failure, func(t *testing.T) {
			directory := t.TempDir()

			file, destination, err := claimFile(directory, "boat", ".bin")
			if err != nil {
				t.Fatalf("💣 exclusive destination fixture: %v", err)
			}

			var saved SavedFile

			if failure == "closed destination" {
				if err := file.Close(); err != nil {
					t.Fatalf("💣 close destination fixture: %v", err)
				}

				saved, err = commitFile(file, destination, []byte("generated media"))
				if !errors.Is(err, errs.ErrOutputFileWrite) || !errors.Is(err, errs.ErrOutputFileClose) || !errors.Is(err, os.ErrClosed) {
					t.Errorf("✗ write or close failure cause lost: %v", err)
				}
			} else {
				saved, err = copyFile(file, destination, filepath.Join(directory, "missing.bin"))
				if !errors.Is(err, os.ErrNotExist) {
					t.Errorf("✗ source-read cause lost: %v", err)
				}
			}

			if saved.Path != "" || saved.Bytes != 0 {
				t.Errorf("✗ incomplete output claimed as saved: %+v", saved)
			}

			if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("✗ incomplete destination remains: %v", err)
			}

			if !t.Failed() {
				t.Log("✓ incomplete destination removed and operation causes preserved")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ failed writes, closes, and source reads never claim saved output")
	}
}

// TestFailedCloseRemovalClassification verifies invariant #4: Close failure with denied removal.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a closed file and directory permissions that prevent removal, closeOutput must return
// ErrOutputFileClose, os.ErrClosed, and os.ErrPermission with the destination path. It must return
// no saved-file result or ErrOutputFileWrite classification, and leave the unremovable file in
// place.
//
// Kind: permanent.
func TestFailedCloseRemovalClassification(t *testing.T) {
	directory := t.TempDir()

	file, destination, err := claimFile(directory, "boat", ".bin")
	if err != nil {
		t.Fatalf("💣 destination fixture: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("💣 closed destination fixture: %v", err)
	}

	// #nosec G302 -- owner directory permissions establish and restore the test failure.
	if err := os.Chmod(directory, 0o500); err != nil {
		t.Fatalf("💣 directory permissions: %v", err)
	}

	t.Cleanup(func() {
		// #nosec G302 -- owner directory permissions establish and restore the test failure.
		if err := os.Chmod(directory, 0o700); err != nil {
			t.Errorf("✗ restore directory permissions: %v", err)
		}
	})

	if err := os.Remove(destination); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("💣 fixture did not establish removal denial: %v", err)
	}

	savedFile, closeErr := closeOutput(file, destination, 0, nil)
	if !errors.Is(closeErr, errs.ErrOutputFileClose) || !errors.Is(closeErr, os.ErrClosed) || !errors.Is(closeErr, os.ErrPermission) {
		t.Errorf("✗ failed close or cleanup cause lost: %v", closeErr)
	}

	if errors.Is(closeErr, errs.ErrOutputFileWrite) {
		t.Errorf("✗ close/removal failure mislabeled as failed writing: %v", closeErr)
	}

	if savedFile.Path != "" || savedFile.Bytes != 0 {
		t.Errorf("✗ unsuccessfully closed output reported as saved: %+v", savedFile)
	}

	var pathError *os.PathError
	if !errors.As(closeErr, &pathError) || pathError.Path != destination {
		t.Errorf("✗ partial destination path lost: %v", closeErr)
	}

	if _, statErr := os.Stat(destination); statErr != nil {
		t.Errorf("✗ permission-denied partial destination unexpectedly absent: %v", statErr)
	}

	if !t.Failed() {
		t.Log("✓ failed close retains its operation, causes, and partial destination")
	}
}
