package params

import (
	"fmt"
)

// GuidanceMissing reports whether a parameter lacks declared constraints and flag value guidance.
// Boolean flags need no value guidance.
func GuidanceMissing(paramCfg *Definition, paramFlag *Flag) bool {
	if paramFlag.DataType == DataBoolean {
		return false
	}

	_, hasMin := paramCfg.MinValue.ValIf()

	_, hasMax := paramCfg.MaxValue.ValIf()
	if len(paramCfg.AllowedValues) > 0 || hasMin || hasMax || paramCfg.CustomSize != nil {
		return false
	}

	return paramFlag.TextHint == "" && len(paramFlag.ExampleValues) == 0
}

// ConstraintFault returns the first constraint error in a parameter configuration, or an empty
// string when it is valid.
func ConstraintFault(param *Definition) string {
	minVal, hasMin := param.MinValue.ValIf()

	maxVal, hasMax := param.MaxValue.ValIf()
	switch {
	case len(param.AllowedValues) > 0 && (hasMin || hasMax):
		return ValuesAndRangeConflict
	case param.CustomSize != nil && len(param.AllowedValues) > 0:
		return BoundsAndValuesConflict
	case param.MaxMultiple < 0:
		return fmt.Sprintf(NegativeMaxMultipleForm, param.MaxMultiple)
	case (hasMin && minVal < 0) || (hasMax && maxVal < 0):
		return RangeBoundNegative
	case hasMin && hasMax && minVal > maxVal:
		return fmt.Sprintf(RangeInverted, FormatValue(minVal), FormatValue(maxVal))
	}

	return ""
}
