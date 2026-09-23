package main

// Invariants tested:
//  1. Omitted thoughts parameter: When adjusted parameters omit include-thoughts, writeRunResults
//     must save run.png without changing the existing run.md. Neither run-02.png nor run-02.md may
//     appear.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"
)

// TestWriteRunResultsSkipsIgnoredThoughtsSidecar verifies invariant #1: Omitted thoughts parameter.
//
// What is being tested:
// Given returned thoughts but no params.FlagTypeThoughts in adjusted parameters, writeRunResults
// must save IMG to run.png without an error and leave the existing run.md content unchanged.
// Neither run-02.png nor run-02.md may exist afterward.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWriteRunResultsSkipsIgnoredThoughtsSidecar(t *testing.T) {
	app := realApp(t)
	outDir := t.TempDir()
	existingSidecarPath := filepath.Join(outDir, "run.md")

	// #nosec G306 -- permissions are sufficient for this test-owned file.
	if err := os.WriteFile(existingSidecarPath, []byte("existing notes"), 0o644); err != nil {
		t.Fatalf("💣 seed existing Markdown file: %v", err)
	}

	result := generation.Result{
		Artifacts: []artifact.Media{{Data: []byte("IMG"), FileExt: ".png"}},
		Thoughts:  []string{"provider did not consume this"},
	}

	var writeErr error

	_ = captureBoth(t, func() {
		_, _, writeErr = writeRunResults(
			&result,
			outDir,
			"run",
			"",
			&generation.Generation{ProvModelPair: catalog.ProvModelPair{Model: catalog.Model{ID: "fixture-model"}}, Prompt: params.GetSetIf(true, "a prompt").ValOr(""), Preparation: generation.Preparation{Params: params.Values{}}},
			app.catalog.Flags,
			&output.GenerationOutcome{},
		)
	})
	if writeErr != nil {
		t.Errorf("✗ write result after ignored include-thoughts: %v", writeErr)
	}

	// #nosec G304 -- the expected artifact belongs to this test's temporary directory.
	artifactBytes, err := os.ReadFile(filepath.Join(outDir, "run.png"))
	if err != nil {
		t.Errorf("✗ artifact did not keep the requested filename: %v", err)
	} else if string(artifactBytes) != "IMG" {
		t.Errorf("✗ artifact content = %q, want IMG", artifactBytes)
	}

	// #nosec G304 -- the seeded file belongs to this test's temporary directory.
	existingSidecarBytes, err := os.ReadFile(existingSidecarPath)
	if err != nil {
		t.Errorf("✗ read seeded Markdown file: %v", err)
	} else if string(existingSidecarBytes) != "existing notes" {
		t.Errorf("✗ seeded Markdown file changed to %q", existingSidecarBytes)
	}

	for _, unwantedFilename := range []string{"run-02.png", "run-02.md"} {
		if _, err := os.Stat(filepath.Join(outDir, unwantedFilename)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("✗ ignored include-thoughts created %s", unwantedFilename)
		}
	}

	if !t.Failed() {
		t.Log("✓ ignored include-thoughts writes no sidecar and does not change the artifact filename")
	}
}
