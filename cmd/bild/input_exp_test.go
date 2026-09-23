package main

// Invariants tested:
//  1. Nonfinite submitted flags: Given NaN or infinity for strength, getGenFlagsInput must omit
//     that flag, retain the other supplied values, and return a map that json.Marshal can encode.

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/shdeen/bildomat/internal/params"
)

// TestSubmittedFlagsOmitNonfinite verifies invariant #1: Nonfinite submitted flags.
//
// What is being tested:
// Given NaN or either infinity as strength, getGenFlagsInput must omit strength and retain the
// supplied guidance-scale, size, and model values. json.Marshal must encode the returned map
// without an error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSubmittedFlagsOmitNonfinite(t *testing.T) {
	strengthFlag := params.FlagType("strength")
	guidanceFlag := params.FlagType("guidance-scale")

	for _, nonfinite := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		runFlags := RunFlags{Model: params.GetSetIf(true, "model-token")}
		flags := getGenFlagsInput(&runFlags, params.FlagInputs{
			strengthFlag:        nonfinite,
			guidanceFlag:        2.5,
			params.FlagTypeSize: "1024x1024",
		})

		if _, present := flags[string(strengthFlag)]; present {
			t.Errorf("✗ the nonfinite value %v stayed in the flags record: %#v", nonfinite, flags)
		}

		if flags[string(guidanceFlag)] != 2.5 || flags[string(params.FlagTypeSize)] != "1024x1024" || flags["model"] != "model-token" {
			t.Errorf("✗ a finite flag went missing beside the nonfinite value %v: %#v", nonfinite, flags)
		}

		if _, err := json.Marshal(flags); err != nil {
			t.Errorf("✗ the flags record does not encode beside the nonfinite value %v: %v", nonfinite, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ a nonfinite flag value is left out of the flags record and the record encodes")
	}
}
