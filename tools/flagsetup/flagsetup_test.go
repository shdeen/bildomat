package main

// Invariants tested:
//  1. Generated flag records current: Given the checked-in parameter flag document, generate must
//     write exactly the bytes of internal/params/zz_paramflags.go without an error.
//  2. Generated source compiles into the declared records: Given the three fixture flag records,
//     generate must write Go source beginning with the generated-code marker. A program using that
//     source must compile, run, and return records equal to readRecords for the same document. The
//     returned strings must preserve quotes and backslashes, and empty lists must remain distinct
//     from omitted lists.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestGeneratedRecordsCurrent verifies invariant #1: Generated flag records current.
//
// What is being tested:
// Given the checked-in parameter flag document, generate must write exactly the bytes of
// internal/params/zz_paramflags.go without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestGeneratedRecordsCurrent(t *testing.T) {
	generatedPath := filepath.Join(t.TempDir(), generatedName)

	if err := generate(filepath.Join(paramsPackageDir, documentPath), generatedPath); err != nil {
		t.Fatalf("💣 generating from the checked-in document failed: %v", err)
	}

	// #nosec G304 -- the read is the file this test just generated in its own scratch directory.
	generated, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("💣 the generator reported success and wrote no file: %v", err)
	}

	// #nosec G304 -- the read compares this repository's own generated file.
	checkedIn, err := os.ReadFile(filepath.Join(paramsPackageDir, generatedName))
	if err != nil {
		t.Fatalf("💣 the checked-in generated file is unreadable: %v", err)
	}

	if string(generated) != string(checkedIn) {
		t.Errorf("✗ %s is stale — run go generate on internal/params", generatedName)
	}

	if !t.Failed() {
		t.Log("✓ the checked-in generated records file matches the parameter flag document")
	}
}

// TestGeneratedSourceCompiles verifies invariant #2: Generated source compiles into the declared
// records.
//
// What is being tested:
// Given the three fixture flag records, generate must write Go source beginning with the
// generated-code marker. A program using that source must compile, run, and return records equal to
// readRecords for the same document. The returned strings must preserve quotes and backslashes, and
// empty lists must remain distinct from omitted lists.
// Test class: Expanded.
// Test layer: Coverage.
func TestGeneratedSourceCompiles(t *testing.T) {
	programDir := t.TempDir()
	packageDir := filepath.Join(programDir, "params")

	if err := os.Mkdir(packageDir, 0o750); err != nil {
		t.Fatalf("💣 creating the scratch package directory failed: %v", err)
	}

	document := []byte(`[
		{"flagID": "input-media", "dataType": "string", "aliases": ["i"], "flagName": "Input media",
		 "description": "A \"quoted\" path with a back\\slash.", "exampleValues": ["a.png"], "comment": "Repeatable.",
		 "textHint": "media-file", "allowMultiple": true},
		{"flagID": "steps", "dataType": "integer", "aliases": [], "flagName": "Sampling steps", "description": "Step count."},
		{"flagID": "seed", "dataType": "integer", "flagName": "Seed", "description": "Sampling seed."}]`)
	sourcePath := filepath.Join(programDir, "paramflags.json")
	generatedPath := filepath.Join(packageDir, generatedName)

	scratchFiles := map[string]string{
		sourcePath:                                string(document),
		filepath.Join(programDir, "go.mod"):       scratchModuleFile,
		filepath.Join(programDir, "main.go"):      scratchProgramFile,
		filepath.Join(packageDir, "paramflag.go"): scratchRecordTypeFile,
	}
	for scratchPath, content := range scratchFiles {
		if err := os.WriteFile(scratchPath, []byte(content), 0o600); err != nil {
			t.Fatalf("💣 writing %s failed: %v", scratchPath, err)
		}
	}

	if err := generate(sourcePath, generatedPath); err != nil {
		t.Fatalf("💣 generating from a valid document failed: %v", err)
	}

	// #nosec G304 -- the read is the file this test just generated in its own scratch directory.
	generated, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("💣 the generated file is unreadable: %v", err)
	}

	openingLine, _, _ := strings.Cut(string(generated), "\n")
	if !strings.HasPrefix(openingLine, "// Code generated ") || !strings.HasSuffix(openingLine, "DO NOT EDIT.") {
		t.Errorf("✗ the generated file does not open with the generated-code marker: %q", openingLine)
	}

	// #nosec G204 -- the command is the Go toolchain over this test's own scratch program.
	programRun := exec.CommandContext(t.Context(), "go", "run", ".")
	programRun.Dir = programDir

	printedRecords, err := programRun.Output()
	if err != nil {
		t.Fatalf("💣 the scratch program over the generated file did not compile and run: %v", err)
	}

	var returned []flagRecord
	if err := json.Unmarshal(printedRecords, &returned); err != nil {
		t.Fatalf("💣 the scratch program's output does not decode: %v", err)
	}

	declared, err := readRecords(sourcePath, document)
	if err != nil {
		t.Fatalf("💣 the fixture document was rejected: %v", err)
	}

	if !reflect.DeepEqual(returned, declared) {
		t.Errorf("✗ the generated function returns %+v, the document declares %+v", returned, declared)
	}

	if !t.Failed() {
		t.Log("✓ the generated file compiles into exactly the records the document declares")
	}
}

// paramsPackageDir locates the parameter data and generated records for comparison.
const paramsPackageDir = "../../internal/params"

// The scratch program of TestGeneratedSourceCompiles: a module holding the generated file beside a
// copy of the record type, and a main package that prints what the generated function returns. The
// copy's JSON keys carry no omitempty, so a nil list prints as null and an empty one as [].
const (
	scratchModuleFile = "module scratch\n\ngo 1.24\n"

	scratchProgramFile = `package main

import (
	"encoding/json"
	"os"

	"scratch/params"
)

func main() {
	if err := json.NewEncoder(os.Stdout).Encode(params.Flags()); err != nil {
		os.Exit(1)
	}
}
`

	scratchRecordTypeFile = `package params

type Flag struct {
	FlagID        string   ` + "`json:\"flagID\"`" + `
	FlagName      string   ` + "`json:\"flagName\"`" + `
	DataType      string   ` + "`json:\"dataType\"`" + `
	Aliases       []string ` + "`json:\"aliases\"`" + `
	Description   string   ` + "`json:\"description\"`" + `
	ExampleValues []string ` + "`json:\"exampleValues\"`" + `
	Comment       string   ` + "`json:\"comment\"`" + `
	TextHint      string   ` + "`json:\"textHint\"`" + `
	AllowMultiple bool     ` + "`json:\"allowMultiple\"`" + `
}
`
)
