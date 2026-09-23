package params

// Invariants tested:
//  1. Parameter value error identities: Given invalid number, integer, or Boolean text, ParseValue
//     must return the corresponding conversion sentinel under ErrParamValue.
//  2. Parameter value parsing under arbitrary input: For arbitrary data-type names and values, any
//     successful ParseValue result must have the declared Go type.

import (
	"errors"
	"math"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestParseValueSentinels verifies invariant #1: Parameter value error identities.
//
// What is being tested:
// Given invalid number, integer, or Boolean text, ParseValue must return the corresponding
// conversion sentinel under ErrParamValue. Given an unknown data type, it must return
// ErrParamValueDataType under the same root.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestParseValueSentinels(t *testing.T) {
	cases := []struct {
		name     string
		dataType DataType
		rawValue string
		want     error
	}{
		{"number", DataNumber, "not-a-number", errs.ErrParamValueNumber},
		{"integer", DataInteger, "1.5", errs.ErrParamValueInteger},
		{"boolean", DataBoolean, "sometimes", errs.ErrParamValueBoolean},
		{"data type", DataType("bogus"), "value", errs.ErrParamValueDataType},
	}

	for _, testCase := range cases {
		_, err := ParseValue(testCase.dataType, testCase.rawValue)
		if !errors.Is(err, testCase.want) {
			t.Errorf("✗ %s parse error = %v, want %v", testCase.name, err, testCase.want)
		}

		if !errors.Is(err, errs.ErrParamValue) {
			t.Errorf("✗ %s parse error = %v, want the parameter-value root", testCase.name, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ rejected parameter values wrap their precise sentinel and category root")
	}
}

// FuzzParseValue verifies invariant #2: Parameter value parsing under arbitrary input.
//
// What is being tested:
// For arbitrary data-type names and values, any successful ParseValue result must have the declared
// Go type. Formatting and reparsing it must preserve its value, including NaN as NaN.
//
// Test class: Expanded.
// Test layer: Fuzzing.
// Kind: permanent.
func FuzzParseValue(f *testing.F) {
	for _, seedPair := range [][2]string{
		{"string", "auto"},
		{"number", "1.5"},
		{"integer", "8"},
		{"integer", "2.5"},
		{"boolean", "true"},
		{"number", "nan"},
		{"integer", "0x10"},
		{"bogus", "x"},
	} {
		f.Add(seedPair[0], seedPair[1])
	}

	f.Fuzz(func(t *testing.T, dataTypeText, rawValue string) {
		parsedVal, err := ParseValue(DataType(dataTypeText), rawValue)
		if err != nil {
			return
		}

		typeMatch := false

		switch DataType(dataTypeText) {
		case DataString:
			_, typeMatch = parsedVal.(string)
		case DataNumber:
			_, typeMatch = parsedVal.(float64)
		case DataInteger:
			_, typeMatch = parsedVal.(int)
		case DataBoolean:
			_, typeMatch = parsedVal.(bool)
		}

		if !typeMatch {
			t.Errorf("✗ ParseValue(%q, %q) accepted a value of mismatched dynamic type %T", dataTypeText, rawValue, parsedVal)
		}

		reparsedVal, reErr := ParseValue(DataType(dataTypeText), FormatValue(parsedVal))
		// NaN is the one accepted value that never equals itself; a NaN reparsing as NaN IS
		// the stable round trip (the adjustment layer drops non-finite values before any
		// transmission).
		parsedFloat, parsedIsFloat := parsedVal.(float64)
		reparsedFloat, reparsedIsFloat := reparsedVal.(float64)

		bothNaN := parsedIsFloat && reparsedIsFloat && math.IsNaN(parsedFloat) && math.IsNaN(reparsedFloat)
		if reErr != nil || (reparsedVal != parsedVal && !bothNaN) {
			t.Errorf("✗ round trip of (%q, %q): parsed %v, reparsed (%v, %v)", dataTypeText, rawValue, parsedVal, reparsedVal, reErr)
		}

		if !t.Failed() {
			t.Logf("✓ accepted values type-match and round-trip through formatting")
		}
	})
}
