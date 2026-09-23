package output

import (
	"fmt"

	"github.com/shdeen/bildomat/internal/params"
)

// formatNotice renders an adjustment notice using the user-facing flag name. Adjusted numeric
// values remain unquoted; string values are quoted where the message permits.
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
	// Include the unknown adjustment type and values in a warning.
	return fmt.Sprintf(FlagChangedGuard, flagName, string(paramChange.Type), paramChange.InputVal, paramChange.WireVal)
}

// noticeFlagValues returns a flag's display name and formatted used value. Flags absent from
// paramFlags use the supplied name map or their identifier, with quoted values.
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
