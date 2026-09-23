package params

// Invariants tested:
//  1. Nullable value presence: Given unset or conditionally unset Nullable values, ValOr must
//     return the fallback and ValIf must return zero with false.
//  2. Nullable unmarshal JSON: Given omitted or null fields, Nullable.UnmarshalJSON must leave them
//     unset.

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestNullable verifies invariant #1: Nullable value presence.
//
// What is being tested:
// Given unset or conditionally unset Nullable values, ValOr must return the fallback and ValIf must
// return zero with false. Given set values, including an empty string, these methods must return
// the stored value and report presence where checked.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestNullable(t *testing.T) {
	if got := (Nullable[string]{}).ValOr("def"); got != "def" {
		t.Errorf("✗ unset ValOr = %q, want default %q", got, "def")
	}

	if got := GetSetIf(true, "x").ValOr("def"); got != "x" {
		t.Errorf("✗ set ValOr = %q, want %q", got, "x")
	}
	// zero value explicitly set must read as set, not as default
	if got := GetSetIf(true, "").ValOr("def"); got != "" {
		t.Errorf("✗ explicit empty ValOr = %q, want %q", got, "")
	}

	if v, ok := GetSetIf(true, 7).ValIf(); !ok || v != 7 {
		t.Errorf("✗ set ValIf = (%d,%v), want (7,true)", v, ok)
	}

	if v, ok := (Nullable[int]{}).ValIf(); ok || v != 0 {
		t.Errorf("✗ unset ValIf = (%d,%v), want (0,false)", v, ok)
	}

	if got := GetSetIf(false, "x").ValOr("def"); got != "def" {
		t.Errorf("✗ SetIf(false) ValOr = %q, want default", got)
	}

	if !t.Failed() {
		t.Log("✓ Nullable distinguishes set (incl. zero) from unset")
	}
}

// TestNullableUnmarshalJSON verifies invariant #2: Nullable unmarshal JSON.
//
// What is being tested:
// Given omitted or null fields, Nullable.UnmarshalJSON must leave them unset. Given zero, 1.5, or
// an empty string, it must retain the value as present. Decoding text into Nullable[float64] must
// return ErrJSONDecode under ErrJSON.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestNullableUnmarshalJSON(t *testing.T) {
	type doc struct {
		A Nullable[float64] `json:"a"`
		B Nullable[string]  `json:"b"`
	}

	var zeroPresent doc
	if err := json.Unmarshal([]byte(`{"a": 0}`), &zeroPresent); err != nil {
		t.Fatalf("✗ decode failed: %v", err)
	}

	if v, ok := zeroPresent.A.ValIf(); !ok || v != 0 {
		t.Errorf("✗ present zero decoded as (%v,%v), want (0,true)", v, ok)
	}

	if _, ok := zeroPresent.B.ValIf(); ok {
		t.Error("✗ omitted key decoded as set")
	}

	var bothPresent doc
	if err := json.Unmarshal([]byte(`{"a": 1.5, "b": ""}`), &bothPresent); err != nil {
		t.Fatalf("✗ decode failed: %v", err)
	}

	if v, ok := bothPresent.A.ValIf(); !ok || v != 1.5 {
		t.Errorf("✗ present value decoded as (%v,%v), want (1.5,true)", v, ok)
	}

	if v, ok := bothPresent.B.ValIf(); !ok || v != "" {
		t.Errorf("✗ present empty string decoded as (%q,%v), want (\"\",true)", v, ok)
	}

	var nullPresent doc
	if err := json.Unmarshal([]byte(`{"a": null}`), &nullPresent); err != nil {
		t.Fatalf("✗ decode failed: %v", err)
	}

	if _, ok := nullPresent.A.ValIf(); ok {
		t.Error("✗ a JSON null decoded as set")
	}

	var mistyped doc
	if err := json.Unmarshal([]byte(`{"a": "text"}`), &mistyped); !errors.Is(err, errs.ErrJSONDecode) || !errors.Is(err, errs.ErrJSON) {
		t.Errorf("✗ a mistyped value decoded with %v, want the JSON decode sentinel under its root", err)
	}

	if !t.Failed() {
		t.Log("✓ Nullable decodes omitted-unset, present-set, null-unset, and classifies a mistyped value")
	}
}
