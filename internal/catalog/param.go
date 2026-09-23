package catalog

import (
	"fmt"

	"github.com/shdeen/bildomat/internal/params"
)

// checkParamCfg returns an error when a parameter configuration does not match the parameter flag
// definitions.
func checkParamCfg(cfgName, modelID string, paramCfg *params.Definition, flagsByID map[params.FlagType]params.Flag) error {
	paramFlag, enumerated := flagsByID[paramCfg.FlagID]
	if !enumerated {
		return createInvalidCfgError(cfgName, fmt.Sprintf(ParamsFlagUnknown, modelID, string(paramCfg.FlagID)))
	}

	if fault := params.ConstraintFault(paramCfg); fault != "" {
		return createInvalidCfgError(cfgName, fmt.Sprintf(ModelFlagFault, modelID, string(paramCfg.FlagID), fault))
	}

	if params.GuidanceMissing(paramCfg, &paramFlag) {
		return createInvalidCfgError(cfgName, fmt.Sprintf(ParamGuidanceMissing, modelID, string(paramCfg.FlagID)))
	}

	for _, allowedVal := range paramCfg.AllowedValues {
		parsedVal, err := params.ParseValue(paramFlag.DataType, allowedVal)
		if err != nil {
			return createInvalidCfgError(cfgName, fmt.Sprintf(AllowedValueFault, modelID, string(paramCfg.FlagID), err.Error()))
		}
		// The count floor is universal: an image count below one is never a valid value, so
		// a declared member under one is incoherent config — membership could otherwise
		// transmit it, and floor-raising a miss would transmit a value outside the declared
		// set.
		if memberCount, ok := parsedVal.(int); paramCfg.FlagID == params.FlagTypeImageN && ok && memberCount < 1 {
			return createInvalidCfgError(cfgName, fmt.Sprintf(AllowedValueBelowCount, modelID, string(paramCfg.FlagID), allowedVal))
		}
	}

	return nil
}
