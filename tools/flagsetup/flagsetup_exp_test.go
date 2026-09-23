package main

// Invariants tested:
//  1. Rejected documents: generate must reject malformed documents, unknown fields, trailing
//     content, duplicate IDs or aliases, and unsupported data types with their specified
//     classifications, without creating a target file.
//  2. Unreadable document and unwritable target: Given a missing document or target directory,
//     generate must return errParamFlags with fs.ErrNotExist and leave the target absent.
//  3. Document reading under arbitrary input: For arbitrary input, readRecords must not panic.
//     Rejections must return nil records with errParamFlags; accepted records must have unique IDs
//     and aliases and supported data types.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestGenerateRejectsBrokenDocument verifies invariant #1: Rejected documents.
//
// What is being tested:
// Given malformed JSON, unknown fields, or invalid trailing text, generate must return
// errParamFlagsDecode. A second JSON document must return errParamFlagsTrailing and
// errParamFlagsDecode. Duplicate flag IDs, duplicate aliases, and unsupported data types must
// return errParamFlagsInvalid. Every case must also match errParamFlags and leave the target file
// absent.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestGenerateRejectsBrokenDocument(t *testing.T) {
	cases := map[string]struct {
		document       string
		classification error
	}{
		"malformed document": {
			`[{"flagID": "quality", "dataType": "string", "flagName": "Quality"`, errParamFlagsDecode,
		},
		"unknown field": {
			`[{"flagID": "quality", "dataType": "string", "flagName": "Quality", "surplus": true}]`, errParamFlagsDecode,
		},
		"trailing content": {
			`[{"flagID": "quality", "dataType": "string", "flagName": "Quality"}] []`, errParamFlagsTrailing,
		},
		"trailing text that is no document": {
			`[{"flagID": "quality", "dataType": "string", "flagName": "Quality"}] surplus`, errParamFlagsDecode,
		},
		"duplicate flag ID": {
			`[{"flagID": "quality", "dataType": "string", "aliases": ["q"], "flagName": "Quality"},
			  {"flagID": "quality", "dataType": "string", "aliases": ["u"], "flagName": "Quality"}]`, errParamFlagsInvalid,
		},
		"duplicate alias": {
			`[{"flagID": "quality", "dataType": "string", "aliases": ["q"], "flagName": "Quality"},
			  {"flagID": "seed", "dataType": "integer", "aliases": ["q"], "flagName": "Seed"}]`, errParamFlagsInvalid,
		},
		"unsupported data type": {
			`[{"flagID": "quality", "dataType": "text", "flagName": "Quality"}]`, errParamFlagsInvalid,
		},
	}

	for name, brokenCase := range cases {
		t.Run(name, func(t *testing.T) {
			scratchDir := t.TempDir()
			sourcePath := filepath.Join(scratchDir, "paramflags.json")
			generatedPath := filepath.Join(scratchDir, generatedName)

			if err := os.WriteFile(sourcePath, []byte(brokenCase.document), 0o600); err != nil {
				t.Fatalf("💣 writing the broken document failed: %v", err)
			}

			err := generate(sourcePath, generatedPath)
			if !errors.Is(err, brokenCase.classification) {
				t.Errorf("✗ generate() error = %v, want the classification %v", err, brokenCase.classification)
			}

			if !errors.Is(err, errParamFlags) {
				t.Errorf("✗ generate() error = %v, want it within the parameter flag category", err)
			}

			if name == "trailing content" && !errors.Is(err, errParamFlagsDecode) {
				t.Errorf("✗ generate() error = %v, want the trailing content classified as a decode failure", err)
			}

			if _, statErr := os.Stat(generatedPath); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("✗ a rejected document left a file at the target path (stat error: %v)", statErr)
			}

			if !t.Failed() {
				t.Log("✓ the broken document is rejected under its classification and nothing is written")
			}
		})
	}
}

// TestGenerateReadAndWriteFailures verifies invariant #2: Unreadable document and unwritable
// target.
//
// What is being tested:
// Given a missing source document or a valid document with a target in a missing directory,
// generate must return an error matching both errParamFlags and fs.ErrNotExist. Neither target file
// may exist afterward.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestGenerateReadAndWriteFailures(t *testing.T) {
	scratchDir := t.TempDir()

	missingSourceTarget := filepath.Join(scratchDir, generatedName)

	err := generate(filepath.Join(scratchDir, "absent.json"), missingSourceTarget)
	if !errors.Is(err, errParamFlags) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("✗ generate() over a missing document = %v, want the category error wrapping the not-exist cause", err)
	}

	sourcePath := filepath.Join(scratchDir, "paramflags.json")
	if writeErr := os.WriteFile(sourcePath, []byte(`[{"flagID": "quality", "dataType": "string", "flagName": "Quality"}]`), 0o600); writeErr != nil {
		t.Fatalf("💣 writing the document failed: %v", writeErr)
	}

	missingDirTarget := filepath.Join(scratchDir, "absent-directory", generatedName)

	err = generate(sourcePath, missingDirTarget)
	if !errors.Is(err, errParamFlags) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("✗ generate() into a missing directory = %v, want the category error wrapping the not-exist cause", err)
	}

	for _, targetPath := range []string{missingSourceTarget, missingDirTarget} {
		if _, statErr := os.Stat(targetPath); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("✗ a failed generation left a file at %s (stat error: %v)", targetPath, statErr)
		}
	}

	if !t.Failed() {
		t.Log("✓ read and write failures carry the category and their cause, and leave no file")
	}
}

// FuzzReadRecords verifies invariant #3: Document reading under arbitrary input.
//
// What is being tested:
// For arbitrary document bytes, readRecords must not panic. A rejected document must return nil
// records and an error matching errParamFlags. Accepted records must have unique flag IDs, unique
// aliases, and only string, number, integer, or boolean data types.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzReadRecords(f *testing.F) {
	// #nosec G304 -- the read is this repository's own parameter flag document.
	shipped, err := os.ReadFile(filepath.Join(paramsPackageDir, documentPath))
	if err != nil {
		f.Fatalf("💣 the parameter flag document is unreadable: %v", err)
	}

	f.Add(shipped)
	f.Add([]byte(`[]`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[{"flagID": "a", "dataType": "string"}, {"flagID": "a", "dataType": "string"}]`))
	f.Add([]byte(`[{"flagID": "a", "dataType": "string", "aliases": ["x", "x"]}]`))
	f.Add([]byte(`[{"flagID": "a", "dataType": "text"}]`))
	f.Add([]byte(`[] []`))
	f.Add([]byte(`{`))

	f.Fuzz(func(t *testing.T, document []byte) {
		records, err := readRecords("fuzzed.json", document)
		if err != nil {
			if !errors.Is(err, errParamFlags) {
				t.Errorf("✗ readRecords() error = %v, want it within the parameter flag category", err)
			}

			if records != nil {
				t.Errorf("✗ readRecords() returned records beside the error %v", err)
			}

			return
		}

		acceptedIDs := map[string]bool{}
		acceptedAliases := map[string]bool{}

		for i := range records {
			record := &records[i]
			if acceptedIDs[record.FlagID] {
				t.Errorf("✗ an accepted document declares the flag ID %q twice", record.FlagID)
			}

			acceptedIDs[record.FlagID] = true

			if !slices.Contains([]string{dataString, dataNumber, dataInteger, dataBoolean}, record.DataType) {
				t.Errorf("✗ an accepted document declares the data type %q", record.DataType)
			}

			for _, alias := range record.Aliases {
				if acceptedAliases[alias] {
					t.Errorf("✗ an accepted document declares the alias %q twice", alias)
				}

				acceptedAliases[alias] = true
			}
		}

		if !t.Failed() {
			t.Logf("✓ the reader stayed within its contract")
		}
	})
}
