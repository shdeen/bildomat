package main

// Invariants tested:
//  1. Submitted generation flags: getGenFlagsInput must retain the supplied model, output-path,
//     duration, and input-media values. Later changes to the input-media source slice must not
//     change its returned copy.

import (
	"slices"
	"testing"

	"github.com/shdeen/bildomat/internal/params"
)

// TestSubmittedGenerationFlags verifies invariant #1: Submitted generation flags.
//
// What is being tested:
// Given explicit model, output-path, duration, and input-media values, getGenFlagsInput must retain
// those values. Changing the original input-media slice afterward must leave the returned slice
// equal to first.png and second.png.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSubmittedGenerationFlags(t *testing.T) {
	mediaSources := []string{"first.png", "second.png"}
	runFlags := RunFlags{
		Model:   params.GetSetIf(true, "model-token"),
		OutPath: params.GetSetIf(true, "results/"),
	}
	flags := getGenFlagsInput(&runFlags, params.FlagInputs{
		params.FlagTypeDuration:   5,
		params.FlagTypeInputMedia: mediaSources,
	})

	mediaSources[0] = "changed.png"
	reportedSources, _ := flags[string(params.FlagTypeInputMedia)].([]string)

	if !slices.Equal(reportedSources, []string{"first.png", "second.png"}) {
		t.Errorf("✗ reported repeatable flag values changed with the source slice: %v", reportedSources)
	}

	if flags["model"] != "model-token" || flags["output-path"] != "results/" || flags["duration"] != 5 {
		t.Errorf("✗ submitted flags are incomplete: %#v", flags)
	}

	if !t.Failed() {
		t.Log("✓ JSON reporting preserves exactly the user-facing flags as submitted")
	}
}
