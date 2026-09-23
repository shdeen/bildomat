package main

// Invariants tested:
//  1. Generated files current: renderPackageSource must produce exactly each package's checked-in
//     generated constants from the checked-in catalog.
//  2. Generator destination consistency: In each temporary repository, generate must write the
//     declared constants under the local target package in alphabetical order. Repeating generation
//     must preserve the exact bytes.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"
)

// TestGeneratedFilesCurrent verifies invariant #1: Generated files current.
//
// What is being tested:
// For every section in the checked-in copy catalog, renderPackageSource must succeed and return
// exactly the bytes of that package's checked-in generated constants file.
//
// Test class: Expanded.
// Test layer: Coverage.
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

// TestGenerationRoots verifies invariant #2: Generator destination consistency.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Run from each of two temporary catalog directories, generate must create the constants file under
// that repository's target package. The file must parse as Go, declare package target, and define
// First = first and Second = second in that order. A second generate call must succeed and leave
// the file's bytes unchanged.
func TestGenerationRoots(t *testing.T) {
	for range 2 {
		root := t.TempDir()
		catalogDirectory := filepath.Join(root, "internal", "templates")

		destinationDirectory := filepath.Join(root, "target")
		for _, directory := range []string{catalogDirectory, destinationDirectory} {
			if err := os.MkdirAll(directory, 0o700); err != nil {
				t.Fatalf("💣 create isolated repository: %v", err)
			}
		}

		writeGeneratorInput(t, filepath.Join(destinationDirectory, "source.go"), "package target\n")
		writeGeneratorInput(t, filepath.Join(catalogDirectory, sourceName), "[target]\nSecond = 'second'\nFirst = 'first'\n")
		t.Chdir(catalogDirectory)

		if err := generate(); err != nil {
			t.Errorf("✗ generation failed: %v", err)

			continue
		}

		target := filepath.Join(destinationDirectory, generatedName)
		// #nosec G304 -- target is in the test-owned temporary repository.
		original, err := os.ReadFile(target)
		if err != nil {
			t.Errorf("✗ generated destination unavailable: %v", err)

			continue
		}

		checkGeneratedDefinitions(t, original)

		if err := generate(); err != nil {
			t.Errorf("✗ repeated generation failed: %v", err)
		}
		// #nosec G304 -- target is in the test-owned temporary repository.
		repeated, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(original, repeated) {
			t.Errorf("✗ repeated generation changed bytes: %v", err)
		}
	}

	if !t.Failed() {
		t.Log("✓ catalog location governs every destination and repeated output is identical")
	}
}

// checkGeneratedDefinitions checks the generated Go declarations against the supplied catalog
// values.
func checkGeneratedDefinitions(t *testing.T, source []byte) {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), generatedName, source, 0)
	if err != nil {
		t.Errorf("✗ generated Go source is invalid: %v", err)

		return
	}

	if parsed.Name.Name != "target" {
		t.Errorf("✗ generated package = %q, want target", parsed.Name.Name)
	}

	var names []string

	values := map[string]string{}

	for _, declaration := range parsed.Decls {
		constants, ok := declaration.(*ast.GenDecl)
		if !ok || constants.Tok != token.CONST {
			t.Errorf("✗ generated declaration is not a constant group: %T", declaration)

			continue
		}

		for _, specification := range constants.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
				t.Errorf("✗ generated declaration does not define one value: %T", specification)

				continue
			}

			literal, ok := value.Values[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Errorf("✗ generated value is not a string: %T", value.Values[0])

				continue
			}

			decoded, decodeErr := strconv.Unquote(literal.Value)
			if decodeErr != nil {
				t.Errorf("✗ generated string is invalid: %v", decodeErr)
			}

			names = append(names, value.Names[0].Name)
			values[value.Names[0].Name] = decoded
		}
	}

	if !slices.Equal(names, []string{"First", "Second"}) || !reflect.DeepEqual(values, map[string]string{"First": "first", "Second": "second"}) {
		t.Errorf("✗ generated definitions or declaration order changed: %v, %v", names, values)
	}
}

// writeGeneratorInput writes isolated Go or catalog input for a generator operation.
func writeGeneratorInput(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("💣 write generator input: %v", err)
	}
}
