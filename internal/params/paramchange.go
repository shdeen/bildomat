package params

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
)

// Adjustment records one adjustment to a supplied generation parameter.
//   - FlagID: the parameter that is adjusted
//   - Type: the kind of adjustment
//   - InputVal: the value supplied by the user
//   - WireVal: the resulting value
//   - Comment: additional user-facing detail about the adjustment
type Adjustment struct {
	FlagID   FlagType `json:"flagID"`
	Type     Change   `json:"type"`
	InputVal string   `json:"inputVal"`
	WireVal  string   `json:"wireVal"`
	Comment  string   `json:"comment"`
}

// Values maps parameter names to values that are ready for a provider request.
type Values map[FlagType]any

// inputVal takes parameter values and a parameter name and returns the value with the
// requested type. It reports false when the parameter is absent or has another type.
func inputVal[T any](paramVals map[FlagType]any, param FlagType) (T, bool) {
	v, ok := paramVals[param].(T)

	return v, ok
}

// Value takes parameter values and a parameter name and returns the stored value with
// the requested type. An absent parameter reads as the zero value. A stored value of another
// type is a repository defect and returns ErrParamValueTypeMismatch naming the parameter.
func Value[T any](paramVals map[FlagType]any, param FlagType) (T, error) {
	var zero T

	stored, present := paramVals[param]
	if !present {
		return zero, nil
	}

	value, ok := stored.(T)
	if !ok {
		return zero, fmt.Errorf("%q, %w", param, errs.ErrParamValueTypeMismatch)
	}

	return value, nil
}

// adjustmentState owns the supplied values, request values, and notices for one adjustment.
type adjustmentState struct {
	supplied    FlagInputs
	adjusted    Values
	changes     []Adjustment
	definitions Definitions
	modelLabel  string
	failure     error
}

// Adjust applies parameter definitions and returns request values and their notices.
// modelLabel identifies the model in existing notices and conflicting-bound errors.
func Adjust(inputs FlagInputs, definitions Definitions, modelLabel string) (Values, []Adjustment, error) {
	paramAdjustment := adjustmentState{supplied: inputs, adjusted: Values{}, definitions: definitions, modelLabel: modelLabel}
	if !paramAdjustment.adjustSize() {
		paramAdjustment.adjustMode()
	}

	if paramAdjustment.failure != nil {
		return paramAdjustment.adjusted, paramAdjustment.changes, paramAdjustment.failure
	}

	paramAdjustment.applyParamLimits()

	return paramAdjustment.adjusted, paramAdjustment.changes, nil
}

// adjustSize applies an explicit, declared size and records superseded sizing inputs.
// It reports whether that size was consumed, including a conflicting-bound failure.
func (paramAdjustment *adjustmentState) adjustSize() bool {
	sizeInput, ok := inputVal[string](paramAdjustment.supplied, FlagTypeSize)

	sizeParam, okSize := paramAdjustment.definitions.Param(FlagTypeSize)
	if !ok || !okSize {
		return false
	}

	width, height, ok := ParseDimensions(sizeInput)
	if !ok {
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeSize, Type: ChangeDropped, InputVal: sizeInput})

		return false
	}

	switch {
	case sizeParam.CustomSize != nil:
		paramAdjustment.storeDimensions(FlagTypeSize, sizeInput, *sizeParam.CustomSize, width, height)
	case len(sizeParam.AllowedValues) > 0:
		normSize := fmtWH(width, height)

		if slices.Contains(sizeParam.AllowedValues, normSize) {
			paramAdjustment.adjusted[FlagTypeSize] = normSize
		} else {
			nearestAllowed := nearestSize(sizeParam.AllowedValues, float64(width)/float64(height))
			paramAdjustment.adjusted[FlagTypeSize] = nearestAllowed
			paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{
				FlagID: FlagTypeSize, Type: ChangeSnapped, InputVal: sizeInput, WireVal: nearestAllowed,
			})
		}
	default:
		paramAdjustment.adjusted[FlagTypeSize] = sizeInput
	}

	if paramAdjustment.failure == nil {
		paramAdjustment.supersede(sizeInput)
	}

	return true
}

// supersede records aspect and resolution values displaced by an explicit size.
func (paramAdjustment *adjustmentState) supersede(sizeInput string) {
	detail := fmt.Sprintf(ReasonSupersededBySize, sizeInput)

	for _, flagID := range []FlagType{FlagTypeAspect, FlagTypeResolution} {
		_, declared := paramAdjustment.definitions.Param(flagID)

		_, supplied := inputVal[string](paramAdjustment.supplied, flagID)
		if declared && supplied {
			paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: flagID, Type: ChangeIgnored, Comment: detail})
		}
	}
}

// adjustMode selects free-form dimensions, fixed dimensions, or independent aspect and resolution adjustment.
func (paramAdjustment *adjustmentState) adjustMode() {
	sizeCfg, _ := paramAdjustment.definitions.Param(FlagTypeSize)

	switch {
	case sizeCfg.CustomSize != nil:
		paramAdjustment.customSize(declaredSizingInputs(paramAdjustment.supplied, paramAdjustment.definitions), *sizeCfg.CustomSize)
	case len(sizeCfg.AllowedValues) > 0:
		paramAdjustment.selectFixedSize(declaredSizingInputs(paramAdjustment.supplied, paramAdjustment.definitions), sizeCfg.AllowedValues)
	default:
		paramAdjustment.splitAspect()
		paramAdjustment.adjustResolution()
	}
}

// splitAspect stores an accepted aspect ratio and records a snapped or unusable input.
func (paramAdjustment *adjustmentState) splitAspect() {
	aspectInput, ok := inputVal[string](paramAdjustment.supplied, FlagTypeAspect)

	aspectCfg, cfgOK := paramAdjustment.definitions.Param(FlagTypeAspect)
	if !ok || !cfgOK {
		return
	}

	if len(aspectCfg.AllowedValues) == 0 {
		paramAdjustment.adjusted[FlagTypeAspect] = aspectInput

		return
	}

	nearestAspect, isMember := aspectNearest(aspectInput, aspectCfg.AllowedValues)

	switch {
	case isMember:
		paramAdjustment.adjusted[FlagTypeAspect] = nearestAspect
	case nearestAspect != "":
		paramAdjustment.adjusted[FlagTypeAspect] = nearestAspect
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeAspect, Type: ChangeSnapped, InputVal: aspectInput, WireVal: nearestAspect})
	default:
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeAspect, Type: ChangeDropped, InputVal: aspectInput})
	}
}

// adjustResolution stores an exact declared label or its nearest numerical tier.
// Inputs without an interpretable declared match receive a dropped notice.
func (paramAdjustment *adjustmentState) adjustResolution() {
	resInput, ok := inputVal[string](paramAdjustment.supplied, FlagTypeResolution)

	resCfg, cfgOK := paramAdjustment.definitions.Param(FlagTypeResolution)
	if !ok || !cfgOK {
		return
	}

	if len(resCfg.AllowedValues) == 0 {
		paramAdjustment.adjusted[FlagTypeResolution] = resInput

		return
	}

	trimmedInput := strings.TrimSpace(resInput)
	if member, found := allowedMember(trimmedInput, resCfg.AllowedValues); found {
		paramAdjustment.adjusted[FlagTypeResolution] = member

		return
	}

	tierHeight, ok := repHeight(strings.ToLower(trimmedInput))
	if !ok {
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeResolution, Type: ChangeDropped, InputVal: resInput})

		return
	}

	nearestAllowed := nearestRes(tierHeight, resCfg.AllowedValues)
	if nearestAllowed == "" {
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeResolution, Type: ChangeDropped, InputVal: resInput})

		return
	}

	paramAdjustment.adjusted[FlagTypeResolution] = nearestAllowed
	paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeResolution, Type: ChangeSnapped, InputVal: resInput, WireVal: nearestAllowed})
}

// applyParamLimits adjusts supplied, declared parameters other than sizing and media inputs.
func (paramAdjustment *adjustmentState) applyParamLimits() {
	for i := range paramAdjustment.definitions {
		param := &paramAdjustment.definitions[i]
		switch param.FlagID {
		case FlagTypeSize, FlagTypeAspect, FlagTypeResolution, FlagTypeInputMedia:
			continue // the sizing modes own the trio, and input enforcement owns the reference images
		case FlagTypeQuality, FlagTypeThinkingLevel, FlagTypeThoughts,
			FlagTypeDuration, FlagTypeImageN, FlagTypeOutputFormat:
		}

		if paramVal, ok := paramAdjustment.supplied[param.FlagID]; ok {
			paramAdjustment.adjustSuppliedParam(param, paramVal)
		}
	}
}

// adjustSuppliedParam applies the declared set or range for the supplied value type.
func (paramAdjustment *adjustmentState) adjustSuppliedParam(param *Definition, paramVal any) {
	switch typedVal := paramVal.(type) {
	case string:
		paramAdjustment.checkAllowedValue(param.FlagID, typedVal, param.AllowedValues)
	case int:
		if param.FlagID == FlagTypeDuration && len(param.AllowedValues) > 0 {
			paramAdjustment.adjustDuration(typedVal, param)

			return
		}

		if len(param.AllowedValues) > 0 {
			paramAdjustment.checkTypedMembership(param, typedVal, DataInteger)

			return
		}

		adjusted, changeRecord := applyNumericRange(param, typedVal)
		paramAdjustment.storeRange(param.FlagID, adjusted, changeRecord)

		if param.FlagID == FlagTypeImageN {
			paramAdjustment.raiseCountFloor()
		}
	case float64:
		// The flag parser accepts nan/inf spellings, but a non-finite number
		// is not a usable value: JSON cannot even encode it, so it drops here
		// like any other unintelligible input rather than surfacing later as
		// a request-encoding failure.
		if math.IsNaN(typedVal) || math.IsInf(typedVal, 0) {
			paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{
				FlagID: param.FlagID, Type: ChangeDropped,
				InputVal: strconv.FormatFloat(typedVal, 'f', -1, 64),
			})

			return
		}

		if len(param.AllowedValues) > 0 {
			paramAdjustment.checkTypedMembership(param, typedVal, DataNumber)

			return
		}

		adjusted, changeRecord := applyNumericRange(param, typedVal)
		paramAdjustment.storeRange(param.FlagID, adjusted, changeRecord)
	case bool:
		if len(param.AllowedValues) > 0 {
			paramAdjustment.checkTypedMembership(param, typedVal, DataBoolean)

			return
		}

		paramAdjustment.adjusted[param.FlagID] = typedVal
	}
}

// checkTypedMembership stores a matching typed value or records its rejection.
func (paramAdjustment *adjustmentState) checkTypedMembership(paramCfg *Definition, paramVal any, dataType DataType) {
	for _, allowedVal := range paramCfg.AllowedValues {
		parsedVal, err := ParseValue(dataType, allowedVal)
		if err != nil {
			continue
		}

		if parsedVal == paramVal {
			paramAdjustment.adjusted[paramCfg.FlagID] = paramVal

			return
		}
	}

	paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{
		FlagID: paramCfg.FlagID, Type: ChangeRejected,
		InputVal: FormatValue(paramVal), Comment: strings.Join(paramCfg.AllowedValues, "|"),
	})
}

// raiseCountFloor raises a stored image count below one and records the change.
func (paramAdjustment *adjustmentState) raiseCountFloor() {
	adjustedCount, ok := paramAdjustment.adjusted[FlagTypeImageN].(int)
	if !ok || adjustedCount >= 1 {
		return
	}

	paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{
		FlagID: FlagTypeImageN, Type: ChangeRaised,
		InputVal: strconv.Itoa(adjustedCount), WireVal: "1", Comment: fmt.Sprintf(BoundMinForm, "1"),
	})
	paramAdjustment.adjusted[FlagTypeImageN] = 1
}

// checkAllowedValue stores the declared spelling of an accepted string or records its rejection.
func (paramAdjustment *adjustmentState) checkAllowedValue(paramFlag FlagType, flagInputVal string, allowedVals []string) {
	if len(allowedVals) == 0 {
		paramAdjustment.adjusted[paramFlag] = flagInputVal

		return
	}

	if member, found := allowedMember(strings.TrimSpace(flagInputVal), allowedVals); found {
		paramAdjustment.adjusted[paramFlag] = member

		return
	}

	paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{
		FlagID: paramFlag, Type: ChangeRejected,
		InputVal: flagInputVal, Comment: strings.Join(allowedVals, "|"),
	})
}

// adjustDuration stores the nearest declared duration and records a changed value.
func (paramAdjustment *adjustmentState) adjustDuration(secs int, param *Definition) {
	nearestSecs := snapInt(secs, allowedIntValues(param.AllowedValues))
	if nearestSecs != secs {
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{
			FlagID: FlagTypeDuration, Type: ChangeSnapped,
			InputVal: strconv.Itoa(secs), WireVal: strconv.Itoa(nearestSecs),
		})
	}

	paramAdjustment.adjusted[FlagTypeDuration] = nearestSecs
}

// allowedIntValues takes text values and returns those that parse as integers.
func allowedIntValues(allowedVals []string) []int {
	parsedInts := make([]int, 0, len(allowedVals))
	for _, allowedVal := range allowedVals {
		parsedVal, err := ParseValue(DataInteger, allowedVal)
		if err != nil {
			continue
		}

		parsedInt, ok := parsedVal.(int)
		if !ok {
			continue
		}

		parsedInts = append(parsedInts, parsedInt)
	}

	return parsedInts
}

// applyNumericRange returns a number constrained to its declared bounds and an optional change record.
func applyNumericRange[N int | float64](paramCfg *Definition, suppliedNum N) (N, *Adjustment) {
	if maxNum, ok := paramCfg.MaxValue.ValIf(); ok && float64(suppliedNum) > maxNum {
		return constrainToBound(paramCfg, suppliedNum, N(maxNum), ChangeCapped, BoundMaxForm)
	}

	if minNum, ok := paramCfg.MinValue.ValIf(); ok && float64(suppliedNum) < minNum {
		return constrainToBound(paramCfg, suppliedNum, N(minNum), ChangeRaised, BoundMinForm)
	}

	return suppliedNum, nil
}

// constrainToBound returns the constrained value and the notice identifying its bound.
func constrainToBound[N int | float64](param *Definition, suppliedNum, constrainedNum N, changeType Change, boundForm string) (N, *Adjustment) {
	return constrainedNum, &Adjustment{
		FlagID: param.FlagID, Type: changeType,
		InputVal: FormatValue(suppliedNum), WireVal: FormatValue(constrainedNum),
		Comment: fmt.Sprintf(boundForm, FormatValue(constrainedNum)),
	}
}

// storeRange stores a numeric result and its optional bound notice.
func (paramAdjustment *adjustmentState) storeRange(flag FlagType, value any, changeRecord *Adjustment) {
	paramAdjustment.adjusted[flag] = value
	if changeRecord != nil {
		paramAdjustment.changes = append(paramAdjustment.changes, *changeRecord)
	}
}

// storeDimensions applies bounds and records a change only for usable dimensions.
func (paramAdjustment *adjustmentState) storeDimensions(flag FlagType, supplied string, bounds SizeBounds, width, height int) {
	adjustedWidth, adjustedHeight := clampFree(bounds, width, height)
	if adjustedWidth == 0 || adjustedHeight == 0 {
		paramAdjustment.failBounds(bounds)

		return
	}

	adjustedSize := fmtWH(adjustedWidth, adjustedHeight)

	paramAdjustment.adjusted[FlagTypeSize] = adjustedSize
	if adjustedWidth != width || adjustedHeight != height {
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: flag, Type: ChangeDerived, InputVal: supplied, WireVal: adjustedSize, Comment: fmt.Sprintf(ReasonModelLimits, paramAdjustment.modelLabel)})
	}
}

// failBounds records a configuration failure with its model and conflicting bounds.
func (paramAdjustment *adjustmentState) failBounds(bounds SizeBounds) {
	paramAdjustment.failure = &errs.ConfigError{Problem: fmt.Sprintf("%s: %+v", paramAdjustment.modelLabel, bounds), Cause: errs.ErrProvConfigInvalid}
}

// sizeSources retains interpreted sizing values and their source flags for notices.
type sizeSources struct {
	parsedRatio         Nullable[float64]
	ratioParamName      FlagType
	userInputRatio      string
	resolutionLevel     Nullable[int]
	userInputResolution string
}

// declaredSizingInputs excludes sizing inputs that the parameter definitions do not consume.
func declaredSizingInputs(userInputs FlagInputs, definitions Definitions) FlagInputs {
	sizingInputs := make(FlagInputs, len(userInputs))

	for flagID, value := range userInputs {
		_, declared := definitions.Param(flagID)
		if (flagID == FlagTypeAspect || flagID == FlagTypeResolution || flagID == FlagTypeSize) && !declared {
			continue
		}

		sizingInputs[flagID] = value
	}

	return sizingInputs
}

// customSize derives free-form dimensions from the declared resolution or aspect input.
// It records unusable inputs and conflicting bounds.
func (paramAdjustment *adjustmentState) customSize(userInputs FlagInputs, bounds SizeBounds) {
	aspectInput, haveAspect := inputVal[string](userInputs, FlagTypeAspect)
	aspectParam := FlagTypeAspect

	if resInput, ok := inputVal[string](userInputs, FlagTypeResolution); ok {
		if width, height, parsed := ParseDimensions(resInput); parsed {
			paramAdjustment.storeDimensions(FlagTypeResolution, resInput, bounds, width, height)

			return
		}

		if _, ok := parseRatio(resInput); ok {
			aspectInput, haveAspect, aspectParam = resInput, true, FlagTypeResolution
		} else {
			paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeResolution, Type: ChangeDropped, InputVal: resInput})
		}
	}

	if !haveAspect {
		return
	}

	ratio, resolutionFound := parseRatio(aspectInput)
	if !resolutionFound {
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: aspectParam, Type: ChangeDropped, InputVal: aspectInput})

		return
	}

	derivedSize := deriveSize(bounds, ratio)

	if derivedSize == "" && bounds.LongEdge > 0 {
		paramAdjustment.failBounds(bounds)

		return
	}

	if derivedSize == "" {
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: aspectParam, Type: ChangeDropped, InputVal: aspectInput})

		return
	}

	paramAdjustment.adjusted[FlagTypeSize] = derivedSize
	paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: aspectParam, Type: ChangeDerived, InputVal: aspectInput, WireVal: derivedSize})
}

// interpretSources parses the sizing inputs and records unusable values.
// A ratio supplied as resolution takes precedence over the aspect input.
func (paramAdjustment *adjustmentState) interpretSources(userInputs FlagInputs) sizeSources {
	var sources sizeSources

	resolutionInput, suppliedResolution := inputVal[string](userInputs, FlagTypeResolution)

	sources.userInputResolution = resolutionInput
	if suppliedResolution {
		if ratio, valid := ratioFromSpec(resolutionInput); valid {
			sources.parsedRatio = GetSetIf(true, ratio)
			sources.ratioParamName, sources.userInputRatio = FlagTypeResolution, resolutionInput
		} else if height, valid := repHeight(strings.ToLower(strings.TrimSpace(resolutionInput))); valid {
			sources.resolutionLevel = GetSetIf(true, height)
		} else {
			paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeResolution, Type: ChangeDropped, InputVal: resolutionInput})
		}
	}

	_, hasRatio := sources.parsedRatio.ValIf()

	_, hasResolution := sources.resolutionLevel.ValIf()
	if aspectInput, supplied := inputVal[string](userInputs, FlagTypeAspect); supplied && !hasRatio {
		if ratio, valid := parseRatio(aspectInput); valid {
			sources.parsedRatio = GetSetIf(true, ratio)
			sources.ratioParamName, sources.userInputRatio = FlagTypeAspect, aspectInput
		} else if !hasResolution {
			paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeAspect, Type: ChangeDropped, InputVal: aspectInput})
		}
	}

	return sources
}

// selectFixedSize stores declared dimensions and records each contributing sizing input.
// Unchanged ratios receive derived notices; changed ratios receive snapped notices.
func (paramAdjustment *adjustmentState) selectFixedSize(userInputs FlagInputs, allowedSizes []string) {
	sources := paramAdjustment.interpretSources(userInputs)
	ratio, hasRatio := sources.parsedRatio.ValIf()
	resolution, hasResolution := sources.resolutionLevel.ValIf()

	if !hasRatio && !hasResolution {
		return
	}

	var selectedSize string

	if hasResolution {
		selectedSize = PickSize(allowedSizes, !hasRatio || ratio >= 1, resolution, sources.parsedRatio)
	} else {
		selectedSize = nearestSize(allowedSizes, ratio)
	}

	if selectedSize == "" {
		return
	}

	paramAdjustment.adjusted[FlagTypeSize] = selectedSize

	if hasRatio && sources.userInputRatio != selectedSize {
		changeKind := ChangeSnapped

		width, height, _ := ParseDimensions(selectedSize)
		if ratio == float64(width)/float64(height) {
			changeKind = ChangeDerived
		}

		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: sources.ratioParamName, Type: changeKind, InputVal: sources.userInputRatio, WireVal: selectedSize})
	}

	if hasResolution {
		paramAdjustment.changes = append(paramAdjustment.changes, Adjustment{FlagID: FlagTypeResolution, Type: ChangeDerived, InputVal: sources.userInputResolution, WireVal: selectedSize})
	}
}
