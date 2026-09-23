package generation

// Invariants tested:
//  1. Independent prepared values: After Preparation.Clone, changing copied parameter values, a
//     repeated string, a media record, or a notice must leave the source unchanged.

import (
	"testing"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestPreparationOwnership verifies invariant #1: Independent prepared values.
//
// What is being tested:
// After Preparation.Clone, changing copied parameter values, a repeated string, a media record, or
// a notice must leave the source unchanged. Clone must preserve ReuseURI and provide a writable
// parameter map when the source map is nil.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestPreparationOwnership(t *testing.T) {
	timestamp := 0.0
	original := Preparation{
		Params:     params.Values{params.FlagTypeQuality: "high", params.FlagTypeInputMedia: []string{"first.png", "second.png"}},
		InputMedia: []media.Input{{Filepath: "first.png", Bytes: []byte("original"), Time: &timestamp}},
		Changes:    []params.Adjustment{{FlagID: params.FlagTypeQuality, Type: params.ChangeSnapped, WireVal: "high"}},
		ReuseURI:   "provider-reference",
	}
	copied := original.Clone()
	copied.Params[params.FlagTypeQuality] = "low"

	repeated, typeErr := params.Value[[]string](copied.Params, params.FlagTypeInputMedia)
	if typeErr != nil {
		t.Fatalf("💣 repeated-value setup failed: %v", typeErr)
	}

	repeated[0] = "changed.png"
	replacementTime := 4.0
	copied.InputMedia[0] = media.Input{Bytes: []byte("replacement"), Time: &replacementTime}
	copied.Changes[0].WireVal = "low"

	originals, typeErr := params.Value[[]string](original.Params, params.FlagTypeInputMedia)
	if typeErr != nil {
		t.Fatalf("💣 original-value inspection failed: %v", typeErr)
	}

	if original.Params[params.FlagTypeQuality] != "high" || originals[0] != "first.png" || original.InputMedia[0].Filepath != "first.png" || string(original.InputMedia[0].Bytes) != "original" || original.InputMedia[0].Time != &timestamp || timestamp != 0 || original.Changes[0].WireVal != "high" || copied.ReuseURI != "provider-reference" {
		t.Errorf("✗ copied preparation changed its source or lost reuse data: %+v / %+v", original, copied)
	}

	empty := Preparation{}
	writable := empty.Clone()

	writable.Params[params.FlagTypeQuality] = "high"
	if empty.Params != nil {
		t.Errorf("✗ empty source acquired mutable parameters: %v", empty.Params)
	}

	if !t.Failed() {
		t.Log("✓ copied preparation owns mutable values while retaining immutable input data")
	}
}
