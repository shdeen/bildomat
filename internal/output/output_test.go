package output

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Invariants tested:
//  1. Generation header model name: Given provider and model display names, PrintRunDetails must
//     include both names in their configured header messages on stdout.
//  2. Missing provider label: Given an empty provider name, PrintRunDetails must include
//     ProviderFallbackAsName in the provider header on stdout.

// TestRunHeaderModelName verifies invariant #1: Generation header model name.
//
// What is being tested:
// Given provider and model display names, PrintRunDetails must include both names in their
// configured header messages on stdout.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestRunHeaderModelName(t *testing.T) {
	block := captureStdout(t, func() {
		_ = PrintRunDetails(os.Stdout, "Fixture Provider", "Fixture Image Model")
	})
	for _, want := range []string{
		fmt.Sprintf(ProviderHeader, "Fixture Provider"),
		fmt.Sprintf(ModelHeader, "Fixture Image Model"),
	} {
		if !strings.Contains(block, want) {
			t.Errorf("✗ PrintRunDetails output missing %q in:\n%s", want, block)
		}
	}

	if !t.Failed() {
		t.Log("✓ PrintRunDetails renders the provider label and the published model name")
	}
}

// TestRenderFallbacks verifies invariant #2: Missing provider label.
//
// What is being tested:
// Given an empty provider name, PrintRunDetails must include ProviderFallbackAsName in the provider
// header on stdout.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestRenderFallbacks(t *testing.T) {
	block := captureStdout(t, func() {
		_ = PrintRunDetails(os.Stdout, "", "m")
	})
	if !strings.Contains(block, fmt.Sprintf(ProviderHeader, ProviderFallbackAsName)) {
		t.Errorf("✗ an empty label did not fall back to 'unknown provider':\n%s", block)
	}

	if !t.Failed() {
		t.Log("✓ the provider-label fallback holds")
	}
}
