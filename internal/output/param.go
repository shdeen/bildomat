package output

import (
	"fmt"

	"github.com/shdeen/bildomat/internal/params"
)

// File: internal/output/param.go
// Parameter change notices use each classification's copy-catalog form, keyed
// to the parameter's user-facing FlagName. The command selects their destination.

// formatNotice takes the parameter-flag definitions and a parameter change and
// returns the change's notice: its classification's catalog form carrying the
// flag's user-facing name, with a string-typed used value in single quotes and
// a non-string one bare. The capped and raised limits render from the record's
// constrained value, which is the bound itself.
func formatNotice(paramFlags []params.Flag, paramChange *params.Adjustment, flagNames map[params.FlagType]string) string {
	flagName, usedValue := noticeFlagValues(paramFlags, paramChange, flagNames)

	switch paramChange.Type {
	case params.ChangeSnapped, params.ChangeDerived, params.ChangeConformed:
		return fmt.Sprintf(FlagAdjusted, flagName, usedValue)
	case params.ChangeForced:
		return fmt.Sprintf(FlagAdjustedReason, flagName, usedValue, paramChange.Comment)
	case params.ChangeDropped:
		return fmt.Sprintf(FlagNotSupported, flagName, paramChange.InputVal)
	case params.ChangeRejected:
		return fmt.Sprintf(FlagNotAllowed, flagName, paramChange.InputVal)
	case params.ChangeCapped:
		return fmt.Sprintf(FlagExceedsLimit, flagName, paramChange.InputVal, paramChange.WireVal, paramChange.WireVal)
	case params.ChangeRaised:
		return fmt.Sprintf(FlagBelowLimit, flagName, paramChange.InputVal, paramChange.WireVal, paramChange.WireVal)
	case params.ChangeIgnored:
		return fmt.Sprintf(FlagIgnored, flagName, paramChange.Comment)
	}
	// An unknown classification is a foreseeable repository defect: render it
	// visibly through the guard form rather than dropping the record.
	return fmt.Sprintf(FlagChangedGuard, flagName, string(paramChange.Type), paramChange.InputVal, paramChange.WireVal)
}

// noticeFlagValues takes the parameter-flag definitions and a parameter change
// and returns the change's user-facing flag name and its display-ready used
// value: single-quoted for a string-typed parameter and bare otherwise. A
// record outside the definitions — a run-surface flag — takes its display name
// from the run-surface table, or its raw flag ID as the visible floor, and its
// values render as strings.
func noticeFlagValues(paramFlags []params.Flag, paramChange *params.Adjustment, flagNames map[params.FlagType]string) (flagName, usedValue string) {
	usedValue = singleQuote + paramChange.WireVal + singleQuote

	for i := range paramFlags {
		paramFlag := &paramFlags[i]
		if paramFlag.FlagID != paramChange.FlagID {
			continue
		}

		if paramFlag.DataType != params.DataString {
			usedValue = paramChange.WireVal
		}

		return paramFlag.FlagName, usedValue
	}

	if surfaceName := flagNames[paramChange.FlagID]; surfaceName != "" {
		return surfaceName, usedValue
	}

	return string(paramChange.FlagID), usedValue
}
