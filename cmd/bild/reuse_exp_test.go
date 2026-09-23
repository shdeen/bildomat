package main

// Invariants tested:
//  1. Invalid reuse selections: parseReuse must return nil with the appropriate syntax, provider,
//     record-read, or record-invalid classification for each rejected selection.

import (
	"errors"
	"os"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestReuseSelectionErrors verifies invariant #1: Invalid reuse selections.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// parseReuse must return nil with errs.ErrReuseSyntax for missing separators or values,
// errs.ErrReuseProvider for unsupported identifiers or providers, and errs.ErrRecordRead for a
// missing file. Invalid JSON, schema version 2, or a schema-version-only record must return nil
// with errs.ErrRecordInvalid.
func TestReuseSelectionErrors(t *testing.T) {
	t.Chdir(t.TempDir())

	for _, testCase := range []struct {
		selection, provider string
		failure             error
	}{
		{"", "google", errs.ErrReuseSyntax},
		{"veo-extend", "google", errs.ErrReuseSyntax},
		{"=value", "google", errs.ErrReuseSyntax},
		{"veo-extend=", "google", errs.ErrReuseSyntax},
		{"unknown=https://example.com/video", "google", errs.ErrReuseProvider},
		{"veo-extend=https://example.com/video", "openai", errs.ErrReuseProvider},
		{"veo-extend=https://example.com/video", "unknown", errs.ErrReuseProvider},
		{"veo-extend=missing.bild.json", "google", errs.ErrRecordRead},
	} {
		selection, err := parseReuse(testCase.selection, testCase.provider)
		if !errors.Is(err, testCase.failure) || selection != nil {
			t.Errorf("✗ selection %q/%q: %#v, error %v, expected %v", testCase.provider, testCase.selection, selection, err, testCase.failure)
		}
	}

	for _, recordJSON := range []string{`{`, `{"schema-version":2}`, `{"schema-version":1}`} {
		if err := os.WriteFile("invalid.bild.json", []byte(recordJSON), 0o600); err != nil {
			t.Fatalf("💣 fixture record: %v", err)
		}

		selection, err := parseReuse("veo-extend=invalid.bild.json", "google")
		if !errors.Is(err, errs.ErrRecordInvalid) || selection != nil {
			t.Errorf("✗ invalid record accepted: %#v, error %v", selection, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ invalid reuse selections preserve classified failures")
	}
}
