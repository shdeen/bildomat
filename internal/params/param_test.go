package params_test

// Invariants tested:
//  1. Parameter encoding: Given a Definition with only FlagID, JSON marshaling must emit only
//     flagID.

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shdeen/bildomat/internal/params"
)

// TestParamEncoding verifies invariant #1: Parameter encoding.
//
// What is being tested:
// Given a Definition with only FlagID, JSON marshaling must emit only flagID. With all tested
// fields declared, it must emit exactly their values, including required true, a zero minimum,
// maximum four, repeat limit two, and the allowed-value list.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestParamEncoding(t *testing.T) {
	bare := encodedKeys(t, params.Definition{FlagID: params.FlagTypeAspect})
	if !reflect.DeepEqual(bare, map[string]any{"flagID": string(params.FlagTypeAspect)}) {
		t.Errorf("✗ a bare param encodes %v, want only its flagID", bare)
	}

	declared := encodedKeys(t, params.Definition{ParamID: "n", FlagID: params.FlagTypeImageN, Required: true, MinValue: params.GetSetIf(true, 0.0), MaxValue: params.GetSetIf(true, 4.0), MaxMultiple: 2, AllowedValues: []string{"1"}})
	want := map[string]any{"paramID": "n", "flagID": string(params.FlagTypeImageN), "required": true, "minValue": 0.0, "maxValue": 4.0, "maxMultiple": 2.0, "allowedValues": []any{"1"}}

	if !reflect.DeepEqual(declared, want) {
		t.Errorf("✗ a declared param encodes %v, want %v", declared, want)
	}

	if !t.Failed() {
		t.Log("✓ a param encodes exactly its declared values")
	}
}

// encodedKeys decodes JSON into an object map for field-presence assertions.
func encodedKeys(t *testing.T, value any) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("💣 encoding: %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("💣 decoding the encoding: %v", err)
	}

	return document
}
