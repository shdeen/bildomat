package output

import (
	"errors"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// Invariants tested:
//  1. Unknown page classification: Given an unknown page name, renderPage must return an error
//     matching both ErrOutputPageUnknown and ErrOutputPage.

// TestRenderPageUnknown verifies invariant #1: Unknown page classification.
//
// What is being tested:
// Given an unknown page name, renderPage must return an error matching both ErrOutputPageUnknown
// and ErrOutputPage.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRenderPageUnknown(t *testing.T) {
	const missingPage pageName = "missing-page"

	_, err := renderPage(missingPage, nil)
	if !errors.Is(err, errs.ErrOutputPageUnknown) {
		t.Errorf("✗ renderPage(%q) error = %v, want the unknown-page sentinel", missingPage, err)
	}

	if !errors.Is(err, errs.ErrOutputPage) {
		t.Errorf("✗ renderPage(%q) error = %v, want the output-page root", missingPage, err)
	}

	if !t.Failed() {
		t.Log("✓ an unregistered page name returns the precise output-page sentinel")
	}
}
