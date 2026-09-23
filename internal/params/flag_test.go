package params

// Invariants tested:
//  1. Flag input type check: Given parsed values for every built-in flag and string slices for
//     repeatable flags, CheckFlagInputTypes must succeed, including with no inputs.

import (
	"errors"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestCheckFlagInputTypes verifies invariant #1: Flag input type check.
//
// What is being tested:
// Given parsed values for every built-in flag and string slices for repeatable flags,
// CheckFlagInputTypes must succeed, including with no inputs. A Boolean stored as text or a
// repeatable flag stored as one string must return ErrParamValueTypeMismatch and name the flag.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCheckFlagInputTypes(t *testing.T) {
	paramFlags := Flags()

	samples := map[DataType]string{DataString: "x", DataNumber: "1.5", DataInteger: "2", DataBoolean: "true"}
	parsedInputs := FlagInputs{}

	var booleanFlag, repeatableFlag FlagType

	for i := range paramFlags {
		paramFlag := &paramFlags[i]
		if paramFlag.AllowMultiple {
			parsedInputs[paramFlag.FlagID] = []string{"one", "two"}
			repeatableFlag = paramFlag.FlagID

			continue
		}

		parsed, parseErr := ParseValue(paramFlag.DataType, samples[paramFlag.DataType])
		if parseErr != nil {
			t.Fatalf("💣 parse the %s sample for %s: %v", paramFlag.DataType, paramFlag.FlagID, parseErr)
		}

		parsedInputs[paramFlag.FlagID] = parsed

		if paramFlag.DataType == DataBoolean {
			booleanFlag = paramFlag.FlagID
		}
	}

	if booleanFlag == "" || repeatableFlag == "" {
		t.Fatalf("💣 the shipped enumeration declares no boolean flag or no repeatable flag")
	}

	if err := CheckFlagInputTypes(parsedInputs, paramFlags); err != nil {
		t.Errorf("✗ a parsed value of every shipped flag was refused: %v", err)
	}

	if err := CheckFlagInputTypes(FlagInputs{}, paramFlags); err != nil {
		t.Errorf("✗ no inputs were refused: %v", err)
	}

	err := CheckFlagInputTypes(FlagInputs{booleanFlag: "yes"}, paramFlags)
	if err == nil || !errors.Is(err, errs.ErrParamValueTypeMismatch) || !strings.Contains(err.Error(), string(booleanFlag)) {
		t.Errorf("✗ a boolean flag stored as text returned %v, want the type mismatch sentinel naming %s", err, booleanFlag)
	}

	err = CheckFlagInputTypes(FlagInputs{repeatableFlag: "one.png"}, paramFlags)
	if err == nil || !errors.Is(err, errs.ErrParamValueTypeMismatch) || !strings.Contains(err.Error(), string(repeatableFlag)) {
		t.Errorf("✗ a repeatable flag stored as one string returned %v, want the type mismatch sentinel naming %s", err, repeatableFlag)
	}

	if !t.Failed() {
		t.Log("✓ the type check accepts every parsed shipped value and names the flag of a mistyped one")
	}
}
