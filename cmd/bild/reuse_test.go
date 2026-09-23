package main

// Invariants tested:
//  1. Reuse selection parsing: parseReuse must preserve the complete HTTPS URI or load the expected
//     model record from an absolute, relative, or home-relative path.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReuseSelection verifies invariant #1: Reuse selection parsing.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Given veo-extend with a Google HTTPS URI, parseReuse must return that entire URI, including its
// query, without an error. Given the fixture record through an absolute, relative, or home-relative
// path, it must load the Veo model from the record and leave URI empty.
func TestReuseSelection(t *testing.T) {
	videoURI := "https://generativelanguage.googleapis.com/v1beta/files/original:download?alt=media&token=a=b"

	selection, err := parseReuse("veo-extend="+videoURI, "google")
	if err != nil || selection == nil || selection.URI != videoURI {
		t.Errorf("✗ direct reference %#v, error %v", selection, err)
	}

	temporaryHome := t.TempDir()
	t.Setenv("HOME", temporaryHome)
	t.Chdir(temporaryHome)
	recordPath := filepath.Join(temporaryHome, "boat=source.bild.json")

	recordJSON := []byte(`{"schema-version":1,"provider":"google","model":"veo-3.1-generate-preview","request":{"provider-requests":[]},"provider-responses":[]}`)
	if err := os.WriteFile(recordPath, recordJSON, 0o600); err != nil {
		t.Fatalf("💣 fixture record: %v", err)
	}

	for _, pathValue := range []string{recordPath, "boat=source.bild.json", "~/boat=source.bild.json"} {
		selection, err := parseReuse("veo-extend="+pathValue, "google")
		if err != nil || selection == nil || selection.Record == nil || selection.Record.Model != "veo-3.1-generate-preview" || selection.URI != "" {
			t.Errorf("✗ record %q: %#v, error %v", pathValue, selection, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ reuse preserves URIs and reads expanded record paths")
	}
}
