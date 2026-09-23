package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

// Invariants tested:
// 1. Sidecar file contents: Write must create run-stem.md containing exactly the supplied bytes.

// TestWriteSidecarFile verifies invariant #1: Sidecar file contents.
//
// What is being tested:
// Write must create run-stem.md containing exactly the supplied bytes.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWriteSidecarFile(t *testing.T) {
	dir := t.TempDir()

	content := []byte("---\nprompt: \"x\"\n---\n")
	if _, err := Write(dir, "run-stem", SidecarExt, content); err != nil {
		t.Fatalf("💣 WriteSidecar: %v", err)
	}

	// #nosec G304 -- dir is a test-owned temporary directory.
	b, err := os.ReadFile(filepath.Join(dir, "run-stem.md"))
	if err != nil {
		t.Errorf("✗ sidecar file missing: %v", err)
	} else if string(b) != string(content) {
		t.Errorf("✗ sidecar content = %q, want exactly the given bytes", b)
	}

	if !t.Failed() {
		t.Log("✓ WriteSidecar writes <stem>.md with exactly the given content")
	}
}
