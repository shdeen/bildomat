package config

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// Invariants tested:
// 1. User configuration loading matrix: Load must decode valid settings and return the
//    configuration path under HOME.
// 2. Unknown provider fault: Given a configuration path and unknown provider ID,
//    UnknownProviderFault must return an error matching ErrUserConfig and
//    ErrUserConfigUnknownProvider whose text includes both supplied values.

// TestLoad verifies invariant #1: User configuration loading matrix.
//
// What is being tested:
// Load must decode valid settings and return the configuration path under HOME. Missing files and
// empty values must yield unset settings without faults; unreadable or invalid YAML files must
// yield unset settings and one ErrUserConfig fault. An unknown key must produce a fault naming that
// key while preserving valid settings.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLoad(t *testing.T) {
	cases := []struct {
		name          string
		fileContent   string
		hasFile       bool
		unreadable    bool
		wantModel     string
		wantOutDir    string
		wantKeys      map[string]string
		wantFault     bool
		wantFaultName string
	}{
		{
			name: "complete file decodes",
			fileContent: "default-model: gamma/fixture-model\noutput-dir: ~/Pictures/bild\n" +
				"api-keys:\n  alpha: alpha-test-value\n  beta: beta-test-value\n",
			hasFile:    true,
			wantModel:  "gamma/fixture-model",
			wantOutDir: "~/Pictures/bild",
			wantKeys:   map[string]string{"alpha": "alpha-test-value", "beta": "beta-test-value"},
		},
		{
			name: "missing config directory is silent",
		},
		{
			name:        "invalid yaml warns and yields nothing",
			fileContent: "default-model: [unclosed\n",
			hasFile:     true,
			wantFault:   true,
		},
		{
			name:        "unreadable file warns and yields nothing",
			fileContent: "default-model: gamma/fixture-model\n",
			hasFile:     true,
			unreadable:  true,
			wantFault:   true,
		},
		{
			name:          "unknown setting warns and the rest applies",
			fileContent:   "default-model: gamma/fixture-model\ndefault-modle: mistyped-value\n",
			hasFile:       true,
			wantModel:     "gamma/fixture-model",
			wantFault:     true,
			wantFaultName: "default-modle",
		},
		{
			name:        "empty values load as unset",
			fileContent: "default-model: \"\"\noutput-dir: \"\"\napi-keys:\n  alpha: \"\"\n",
			hasFile:     true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			wantPath := filepath.Join(home, ".bildomat", "config.yml")
			if c.hasFile {
				writeConfigFile(t, home, c.fileContent)
			}

			if c.unreadable {
				if err := os.Chmod(wantPath, 0o000); err != nil {
					t.Fatalf("💣 chmod failed: %v", err)
				}
			}

			settings, configPath, faults := Load()

			if configPath != wantPath {
				t.Errorf("✗ path = %q, want %q", configPath, wantPath)
			}

			if settings.DefaultModel != c.wantModel {
				t.Errorf("✗ DefaultModel = %q, want %q", settings.DefaultModel, c.wantModel)
			}

			if settings.DefaultOutputDir != c.wantOutDir {
				t.Errorf("✗ DefaultOutputDir = %q, want %q", settings.DefaultOutputDir, c.wantOutDir)
			}

			if !maps.Equal(settings.APIKeys, c.wantKeys) {
				t.Errorf("✗ APIKeys = %v, want %v", settings.APIKeys, c.wantKeys)
			}

			if c.wantFault {
				expectClassifiedFault(t, faults, c.wantFaultName)
			} else if len(faults) != 0 {
				t.Errorf("✗ faults = %v, want none", faults)
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ loading decodes complete settings, stays silent on a missing file, classifies every fault, and drops empty values")
	}
}

// TestUnknownProviderFault verifies invariant #2: Unknown provider fault.
//
// What is being tested:
// Given a configuration path and unknown provider ID, UnknownProviderFault must return an error
// matching ErrUserConfig and ErrUserConfigUnknownProvider whose text includes both supplied values.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestUnknownProviderFault(t *testing.T) {
	fault := UnknownProviderFault("/scratch-home/.bildomat/config.yml", "not-a-provider")

	if !errors.Is(fault, errs.ErrUserConfig) || !errors.Is(fault, errs.ErrUserConfigUnknownProvider) {
		t.Errorf("✗ fault %v does not classify as the unknown-provider user-config fault", fault)
	}

	for _, wantValue := range []string{"/scratch-home/.bildomat/config.yml", "not-a-provider"} {
		if !strings.Contains(fault.Error(), wantValue) {
			t.Errorf("✗ fault %q does not name %q", fault, wantValue)
		}
	}

	if !t.Failed() {
		t.Log("✓ the unknown-provider fault classifies correctly and names the path and the provider ID")
	}
}

// writeConfigFile writes a config file under the given home directory's .bildomat directory.
func writeConfigFile(t *testing.T, home, content string) {
	t.Helper()

	configDir := filepath.Join(home, ".bildomat")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("💣 config directory creation failed: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("💣 config file write failed: %v", err)
	}
}

// expectClassifiedFault asserts exactly one fault matching errs.ErrUserConfig whose text names
// wantFaultName; an empty wantFaultName requires no name.
func expectClassifiedFault(t *testing.T, faults []error, wantFaultName string) {
	t.Helper()

	if len(faults) != 1 || !errors.Is(faults[0], errs.ErrUserConfig) {
		t.Errorf("✗ faults = %v, want one fault under errs.ErrUserConfig", faults)

		return
	}

	// Contains holds trivially when no name is required.
	if !strings.Contains(faults[0].Error(), wantFaultName) {
		t.Errorf("✗ fault %q does not name %q", faults[0], wantFaultName)
	}
}
