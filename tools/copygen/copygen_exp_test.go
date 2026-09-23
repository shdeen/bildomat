package main

// Invariants tested:
//  1. Package clauses: packageNameOf must read the actual package clause despite misleading
//     comments and strings, skip tests and generated targets, and retain parser causes and source
//     paths on failure.
//  2. Complete rendering: If the final package cannot render, generate must preserve the existing
//     first target and leave the second target absent.
//  3. Generator failure causes: loadCatalog and packageNameOf must retain filesystem causes and
//     absolute missing paths. A missing catalog destination must also match errCatalogEntry.

import (
	"errors"
	"go/scanner"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackageClauses verifies invariant #1: Package clauses.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given misleading package text in comments or raw strings, packageNameOf must return actual from
// the real package clause. It must ignore the test file and generated target. Given package 123, it
// must return an error containing the source path and a scanner.ErrorList cause.
func TestPackageClauses(t *testing.T) {
	for _, source := range []string{
		"/*\npackage misleading\n*/\npackage actual\n",
		"package\tactual\nconst example = `\npackage misleading\n`\n",
	} {
		directory := t.TempDir()
		writeGeneratorInput(t, filepath.Join(directory, "source.go"), source)
		writeGeneratorInput(t, filepath.Join(directory, "aaa_test.go"), "package ignored\n")
		writeGeneratorInput(t, filepath.Join(directory, generatedName), "package ignored\n")

		packageName, err := packageNameOf(directory)
		if err != nil || packageName != "actual" {
			t.Errorf("✗ package = %q, error = %v", packageName, err)
		}
	}

	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "invalid.go")
	writeGeneratorInput(t, sourcePath, "package 123\n")

	_, err := packageNameOf(directory)

	var parseErrors scanner.ErrorList
	if !errors.As(err, &parseErrors) || !strings.Contains(err.Error(), sourcePath) {
		t.Errorf("✗ parser failure lacks source and cause: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ package clauses use Go syntax and preserve parsing causes")
	}
}

// TestRenderBeforeWriting verifies invariant #2: Complete rendering.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When the last package contains invalid Go source, generate must return an error, preserve the
// existing first target byte for byte, and leave the second package's target absent.
func TestRenderBeforeWriting(t *testing.T) {
	repository := t.TempDir()

	catalogDirectory := filepath.Join(repository, "internal", "templates")
	if err := os.MkdirAll(catalogDirectory, 0o700); err != nil {
		t.Fatalf("💣 catalog directory: %v", err)
	}

	for _, packageName := range []string{"aaa", "bbb", "zzz"} {
		directory := filepath.Join(repository, packageName)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("💣 package directory: %v", err)
		}

		source := "package " + packageName + "\n"
		if packageName == "zzz" {
			source = "not Go source\n"
		}

		writeGeneratorInput(t, filepath.Join(directory, "source.go"), source)
	}

	original := "package aaa\nconst Existing = `preserved`\n"
	target := filepath.Join(repository, "aaa", generatedName)
	writeGeneratorInput(t, target, original)
	writeGeneratorInput(t, filepath.Join(catalogDirectory, sourceName), "[aaa]\nAdded = 'one'\n[bbb]\nAdded = 'two'\n[zzz]\nAdded = 'three'\n")
	t.Chdir(catalogDirectory)

	if err := generate(); err == nil {
		t.Error("✗ invalid final source rendered successfully")
	}
	// #nosec G304 -- target belongs to this test's isolated temporary repository.
	preserved, err := os.ReadFile(target)
	if err != nil || string(preserved) != original {
		t.Errorf("✗ early target changed: %q, %v", preserved, err)
	}

	_, err = os.Stat(filepath.Join(repository, "bbb", generatedName))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("✗ new target appeared before all packages rendered: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a late rendering failure leaves every destination untouched")
	}
}

// TestCatalogReadCauses verifies invariant #3: Generator failure causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// For a missing catalog or destination directory, loadCatalog must return os.ErrNotExist and name
// the absolute missing path. A missing destination must also return errCatalogEntry. For a missing
// package directory, packageNameOf must retain os.ErrNotExist and its absolute path.
func TestCatalogReadCauses(t *testing.T) {
	root := t.TempDir()
	missingCatalog := filepath.Join(root, sourceName)

	_, err := loadCatalog(missingCatalog, root)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), missingCatalog) {
		t.Errorf("✗ catalog read cause or path lost: %v", err)
	}

	writeGeneratorInput(t, missingCatalog, "[absent]\nLabel = 'value'\n")

	_, err = loadCatalog(missingCatalog, root)
	if !errors.Is(err, os.ErrNotExist) || !errors.Is(err, errCatalogEntry) || !strings.Contains(err.Error(), filepath.Join(root, "absent")) {
		t.Errorf("✗ destination read cause or path lost: %v", err)
	}

	missingPackage := filepath.Join(root, "missing-package")

	_, err = packageNameOf(missingPackage)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), missingPackage) {
		t.Errorf("✗ package read cause or path lost: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ generator read failures retain filesystem causes and absolute paths")
	}
}
