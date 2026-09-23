package params

// Invariants tested:
//  1. Nullable JSON round trips under arbitrary values: For arbitrary Nullable[float64] values,
//     JSON marshaling and unmarshaling must preserve finite set values and presence.

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// FuzzNullableJSON verifies invariant #1: Nullable JSON round trips under arbitrary values.
//
// What is being tested:
// For arbitrary Nullable[float64] values, JSON marshaling and unmarshaling must preserve finite set
// values and presence. Unset values must encode as null and decode unset; set NaN or infinity must
// return ErrJSONEncode.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzNullableJSON(f *testing.F) {
	f.Add(0.0, true)
	f.Add(4.5, true)
	f.Add(-1.0, false)

	f.Fuzz(func(t *testing.T, value float64, set bool) {
		encoded, err := json.Marshal(GetSetIf(set, value))
		if set && (math.IsNaN(value) || math.IsInf(value, 0)) {
			if !errors.Is(err, errs.ErrJSONEncode) {
				t.Errorf("✗ a set bound of %v encoded as %s with %v, want the JSON encode sentinel", value, encoded, err)
			}

			if !t.Failed() {
				t.Logf("✓ the encoder refuses the non-finite bound %v", value)
			}

			return
		}

		if err != nil {
			t.Fatalf("💣 encoding %v (set %v): %v", value, set, err)
		}

		var decoded Nullable[float64]
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("💣 decoding %s: %v", encoded, err)
		}

		decodedValue, decodedSet := decoded.ValIf()
		if decodedSet != set || (set && decodedValue != value) {
			t.Errorf("✗ %v (set %v) round-tripped through %s as %v (set %v)", value, set, encoded, decodedValue, decodedSet)
		}

		if !set && string(encoded) != "null" {
			t.Errorf("✗ an unset bound encoded as %s, want %s", encoded, "null")
		}

		if !t.Failed() {
			t.Logf("✓ %v (set %v) round-trips through %s", value, set, encoded)
		}
	})
}
