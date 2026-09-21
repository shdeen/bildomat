package main

// Invariants tested:
//  1. Generated files current: Every package's generated constants file matches the
//     checked-in catalog, so a catalog edit is incomplete without regeneration.
//
// This file is part of core tests as it verifies that generated files are current.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGeneratedFilesCurrent verifies invariant #1: Generated files current.
//
// What makes it or breaks it:
// This test passes only when this rule holds: Every package's generated constants file matches the
// checked-in catalog source, so a catalog edit is not complete without regeneration.
//
// Test class: Core.
func TestGeneratedFilesCurrent(t *testing.T) {
	repositoryRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("💣 resolve repository: %v", err)
	}

	sections, err := loadCatalog(filepath.Join(repositoryRoot, "internal", "templates", sourceName), repositoryRoot)
	if err != nil {
		t.Fatalf("💣 the catalog failed to load: %v", err)
	}

	for dir, entries := range sections {
		rendered, err := renderPackageSource(repositoryRoot, dir, entries)
		if err != nil {
			t.Fatalf("💣 rendering the %s section failed: %v", dir, err)
		}

		// #nosec G304 -- the read compares this repository's own generated file.
		onDisk, readErr := os.ReadFile(filepath.Join(repositoryRoot, dir, generatedName))
		if readErr != nil {
			t.Errorf("✗ the %s section has no generated file: %v", dir, readErr)

			continue
		}

		if string(onDisk) != rendered {
			t.Errorf("✗ %s/%s is stale — run go generate on internal/templates", dir, generatedName)
		}
	}

	if !t.Failed() {
		t.Log("✓ every package's generated constants file matches the catalog source")
	}
}
