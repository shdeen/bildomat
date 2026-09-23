package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"unicode/utf8"

	"github.com/shdeen/bildomat/internal/errs"
)

// Invariants tested:
// 1. User configuration shape attacks: Given settings with incorrect YAML types, a null document,
//    or a directory at the configuration file path, Load must return unset settings and exactly one
//    ErrUserConfig fault.
// 2. Home directory lookup failure: When HOME is empty, Load must return unset settings, an empty
//    configuration path, and exactly one ErrUserConfigLocate fault.
// 3. User configuration decoding under arbitrary input: For arbitrary content, decode must return
//    only ErrUserConfig faults and discard empty API keys.

// TestLoadShapeAttacks verifies invariant #1: User configuration shape attacks.
//
// What is being tested:
// Given settings with incorrect YAML types, a null document, or a directory at the configuration
// file path, Load must return unset settings and exactly one ErrUserConfig fault.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestLoadShapeAttacks(t *testing.T) {
	cases := []struct {
		name        string
		fileContent string
		isDirectory bool
	}{
		{name: "scalar where the api-keys map belongs", fileContent: "api-keys: just-text\n"},
		{name: "sequence where a string belongs", fileContent: "default-model:\n  - a\n  - b\n"},
		{name: "mapping where a string belongs", fileContent: "output-dir:\n  nested: x\n"},
		{name: "sequence as an api-keys value", fileContent: "api-keys:\n  alpha: [1, 2]\n"},
		{name: "mapping as an api-keys value", fileContent: "api-keys:\n  alpha:\n    nested: x\n"},
		{name: "null document", fileContent: "null\n"},
		{name: "directory at the config path", isDirectory: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			if c.isDirectory {
				if err := os.MkdirAll(filepath.Join(home, ".bildomat", "config.yml"), 0o750); err != nil {
					t.Fatalf("💣 directory creation failed: %v", err)
				}
			} else {
				writeConfigFile(t, home, c.fileContent)
			}

			settings, _, faults := Load()

			if settings.DefaultModel != "" || settings.DefaultOutputDir != "" || len(settings.APIKeys) != 0 {
				t.Errorf("✗ settings = %+v, want fully unset", settings)
			}

			expectClassifiedFault(t, faults, "")

			if !t.Failed() {
				t.Logf("✓ %s yields zero settings and one classified fault", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every shape attack yields zero settings and one classified fault, and the loader returns normally")
	}
}

// TestLoadHomeLookupFailure verifies invariant #2: Home directory lookup failure.
//
// What is being tested:
// When HOME is empty, Load must return unset settings, an empty configuration path, and exactly one
// ErrUserConfigLocate fault.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestLoadHomeLookupFailure(t *testing.T) {
	t.Setenv("HOME", "")

	settings, configPath, faults := Load()

	if settings.DefaultModel != "" || settings.DefaultOutputDir != "" || len(settings.APIKeys) != 0 {
		t.Errorf("✗ settings = %+v, want fully unset", settings)
	}

	if configPath != "" {
		t.Errorf("✗ path = %q, want empty for an unresolvable location", configPath)
	}

	if len(faults) != 1 || !errors.Is(faults[0], errs.ErrUserConfigLocate) {
		t.Errorf("✗ faults = %v, want one fault under errs.ErrUserConfigLocate", faults)
	}

	if !t.Failed() {
		t.Log("✓ a failed home lookup yields zero settings, no path, and the location fault")
	}
}

// FuzzDecode verifies invariant #3: User configuration decoding under arbitrary input.
//
// What is being tested:
// For arbitrary content, decode must return only ErrUserConfig faults and discard empty API keys.
// For valid UTF-8 without YAML quoting, escapes, tags, aliases, block scalars, carriage returns, or
// indented lines, nonempty decoded DefaultModel and DefaultOutputDir values must occur verbatim
// unless numeric or Boolean.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzDecode(f *testing.F) {
	seedContents := []string{
		"default-model: gamma/fixture-model\noutput-dir: ~/Pictures/bild\napi-keys:\n  alpha: alpha-test-value\n  beta: beta-test-value\n",
		"",
		"default-model: [unclosed\n",
		"default-model: gamma/fixture-model\ndefault-modle: mistyped-value\n",
		"default-model: \"\"\noutput-dir: \"\"\napi-keys:\n  alpha: \"\"\n",
		"api-keys: just-text\n",
		"default-model:\n  - a\n  - b\n",
		"output-dir:\n  nested: x\n",
		"api-keys:\n  alpha: [1, 2]\n",
		"null\n",
	}
	for _, seedContent := range seedContents {
		f.Add([]byte(seedContent))
	}

	f.Fuzz(func(t *testing.T, content []byte) {
		settings, faults := decode("/fuzz-home/.bildomat/config.yml", content)

		for _, fault := range faults {
			if !errors.Is(fault, errs.ErrUserConfig) {
				t.Errorf("✗ fault %v does not match errs.ErrUserConfig", fault)
			}
		}

		for providerID, apiKey := range settings.APIKeys {
			if apiKey == "" {
				t.Errorf("✗ an empty api-keys value survived under %q", providerID)
			}
		}

		if !plainShapeContent(t, content) {
			if !t.Failed() {
				t.Log("✓ every fault classifies and no empty api-keys value survives")
			}

			return
		}

		for _, decodedValue := range []string{settings.DefaultModel, settings.DefaultOutputDir} {
			if decodedValue != "" && !coercibleScalar(t, decodedValue) && !bytes.Contains(content, []byte(decodedValue)) {
				t.Errorf("✗ decoded value %q does not appear in the input %q", decodedValue, content)
			}
		}

		if !t.Failed() {
			t.Log("✓ every fault classifies, no empty api-keys value survives, and plain-shape settings come from the input")
		}
	})
}

// plainShapeContent excludes YAML syntax and invalid UTF-8 that can rewrite string values. Numeric
// and Boolean coercion is checked separately.
func plainShapeContent(test testing.TB, content []byte) bool {
	test.Helper()

	if !utf8.Valid(content) || bytes.ContainsAny(content, "\\\"'&*>|!\r") {
		return false
	}

	for contentLine := range bytes.SplitSeq(content, []byte("\n")) {
		if len(contentLine) > 0 && (contentLine[0] == ' ' || contentLine[0] == '\t') {
			return false
		}
	}

	return true
}

// coercibleScalar reports whether YAML could have converted a number or Boolean into this string.
func coercibleScalar(test testing.TB, value string) bool {
	test.Helper()

	if value == "true" || value == "false" {
		return true
	}

	_, err := strconv.ParseFloat(value, 64)

	return err == nil
}
