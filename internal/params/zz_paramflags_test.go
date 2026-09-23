package params

// Invariants tested:
//  1. Parameter identifiers match the catalog: Flags must return a record for each FlagType
//     constant listed in this test.
//  2. Strength parameter flag record: Flags must include strength as DataNumber with a nonempty
//     TextHint and no aliases.
//  3. Generated records equal the document: Given the nonempty checked-in parameter flag document,
//     Flags must return the same number of records in the same order, with every record equal field
//     for field.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestParamNameConstantsCorrespond verifies invariant #1: Parameter identifiers match the catalog.
//
// What is being tested:
// Flags must return a record for each FlagType constant listed in this test.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestParamNameConstantsCorrespond(t *testing.T) {
	enumerated := map[FlagType]bool{}
	for _, paramFlag := range Flags() {
		enumerated[paramFlag.FlagID] = true
	}

	for _, constant := range []FlagType{FlagTypeAspect, FlagTypeResolution, FlagTypeQuality, FlagTypeThinkingLevel, FlagTypeThoughts, FlagTypeDuration, FlagTypeImageN, FlagTypeOutputFormat, FlagTypeInputMedia, FlagTypeSize} {
		if !enumerated[constant] {
			t.Errorf("✗ constant value %q names no enumerated flag ID", constant)
		}
	}

	if !t.Failed() {
		t.Log("✓ every ParamName constant names an enumerated flag ID")
	}
}

// TestStrengthParamFlagRecord verifies invariant #2: Strength parameter flag record.
//
// What is being tested:
// Flags must include strength as DataNumber with a nonempty TextHint and no aliases.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestStrengthParamFlagRecord(t *testing.T) {
	flags := Flags()

	strengthFlagID := FlagType("strength")
	strengthFlagPresent := false

	for _, paramFlag := range flags {
		if paramFlag.FlagID != strengthFlagID {
			continue
		}

		strengthFlagPresent = true

		if paramFlag.DataType != DataNumber {
			t.Errorf("✗ strength data type = %q, want %q", paramFlag.DataType, DataNumber)
		}

		if paramFlag.TextHint == "" {
			t.Error("✗ strength has no text hint")
		}

		if len(paramFlag.Aliases) != 0 {
			t.Errorf("✗ strength aliases = %v, want none", paramFlag.Aliases)
		}

		break
	}

	if !strengthFlagPresent {
		t.Errorf("✗ shipped parameter enumeration has no %q record", strengthFlagID)
	}

	if !t.Failed() {
		t.Log("✓ strength is a number-typed parameter with value guidance and no alias")
	}
}

// TestGeneratedRecordsEqualDocument verifies invariant #3: Generated records equal the document.
//
// What is being tested:
// Given the nonempty checked-in parameter flag document, Flags must return the same number of
// records in the same order, with every record equal field for field.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestGeneratedRecordsEqualDocument(t *testing.T) {
	// #nosec G304 -- the read is this repository's own parameter flag document.
	document, err := os.ReadFile(filepath.Join(paramsPackageDir, documentPath))
	if err != nil {
		t.Fatalf("💣 the parameter flag document is unreadable: %v", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()

	var declared []Flag
	if err := decoder.Decode(&declared); err != nil {
		t.Fatalf("💣 the parameter flag document does not decode: %v", err)
	}

	if len(declared) == 0 {
		t.Fatalf("💣 the parameter flag document declares no record, so the comparison proves nothing")
	}

	generated := Flags()
	if len(generated) != len(declared) {
		t.Fatalf("💣 params.Flags() returns %d records, the document declares %d; stopping ahead of an index panic", len(generated), len(declared))
	}

	for i := range declared {
		if !reflect.DeepEqual(generated[i], declared[i]) {
			t.Errorf("✗ record %d (%s) = %+v, the document declares %+v", i, declared[i].FlagID, generated[i], declared[i])
		}
	}

	if !t.Failed() {
		t.Logf("✓ all %d generated records equal the document's", len(declared))
	}
}

// Parameter fixture paths are relative to this package.
//   - paramsPackageDir: the package directory used by the document comparison
//   - documentPath: the parameter declaration document within the package
const (
	paramsPackageDir = "."
	documentPath     = "config/paramflags.json"
)
