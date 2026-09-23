// Package params owns generation parameter definitions, values, and adjustment.
package params

import (
	"fmt"
	"strconv"

	"github.com/shdeen/bildomat/internal/errs"
)

// FlagType identifies a generation parameter by its command flag name.
//
//go:generate go run ../../tools/flagsetup
type FlagType string

// Parameter names identify the generation values that models may accept.
//   - FlagTypeAspect: the requested aspect ratio
//   - FlagTypeResolution: the requested resolution
//   - FlagTypeQuality: the requested quality
//   - FlagTypeThinkingLevel: the requested thinking level
//   - FlagTypeThoughts: whether provider thoughts are requested
//   - FlagTypeDuration: the requested video duration
//   - FlagTypeImageN: the requested number of images
//   - FlagTypeOutputFormat: the requested image format
//   - FlagTypeInputMedia: an input-media path
//   - FlagTypeSize: the requested width and height
const (
	FlagTypeAspect        FlagType = "aspect-ratio"
	FlagTypeResolution    FlagType = "resolution"
	FlagTypeQuality       FlagType = "quality"
	FlagTypeThinkingLevel FlagType = "thinking-level"
	FlagTypeThoughts      FlagType = "include-thoughts"
	FlagTypeDuration      FlagType = "duration"
	FlagTypeImageN        FlagType = "num-images"
	FlagTypeOutputFormat  FlagType = "output-format"
	FlagTypeInputMedia    FlagType = "input-media"
	FlagTypeSize          FlagType = "size"
)

// DataType identifies a parameter value's data type.
type DataType string

// Parameter value data types.
//   - DataString: text values
//   - DataNumber: floating-point values
//   - DataInteger: integer values
//   - DataBoolean: boolean values
const (
	DataString  DataType = "string"
	DataNumber  DataType = "number"
	DataInteger DataType = "integer"
	DataBoolean DataType = "boolean"
)

// Flag describes a generation parameter's identifier, accepted names, type, and help text.
//   - FlagID: the parameter's canonical flag name
//   - FlagName: the parameter name shown to users ("Duration", "Aspect ratio")
//   - DataType: the parameter's value type
//   - Aliases: alternate flag names
//   - Description: the parameter description shown in help output
//   - ExampleValues: example values shown in help output
//   - Comment: additional parameter guidance shown in help output
//   - TextHint: the value placeholder shown in help output
//   - AllowMultiple: whether the parameter accepts multiple values
//
// The encoding carries only what a record declares: an empty alias list, example list, comment, or
// hint is absent.
type Flag struct {
	FlagID        FlagType `json:"flagID"`
	FlagName      string   `json:"flagName"`
	DataType      DataType `json:"dataType"`
	Aliases       []string `json:"aliases,omitempty"`
	Description   string   `json:"description"`
	ExampleValues []string `json:"exampleValues,omitempty"`
	Comment       string   `json:"comment,omitempty"`
	TextHint      string   `json:"textHint,omitempty"`
	AllowMultiple bool     `json:"allowMultiple,omitempty"`
}

// ParseValue parses parameter text according to the requested data type and returns its typed
// value.
func ParseValue(dataType DataType, rawValue string) (any, error) {
	switch dataType {
	case DataString:
		return rawValue, nil
	case DataNumber:
		parsedFloat, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return nil, fmt.Errorf("%q: %w, %w", rawValue, errs.ErrParamValueNumber, err)
		}

		return parsedFloat, nil
	case DataInteger:
		parsedInt, err := strconv.Atoi(rawValue)
		if err != nil {
			return nil, fmt.Errorf("%q: %w, %w", rawValue, errs.ErrParamValueInteger, err)
		}

		return parsedInt, nil
	case DataBoolean:
		parsedBool, err := strconv.ParseBool(rawValue)
		if err != nil {
			return nil, fmt.Errorf("%q: %w, %w", rawValue, errs.ErrParamValueBoolean, err)
		}

		return parsedBool, nil
	}

	return nil, fmt.Errorf("%q: %w", dataType, errs.ErrParamValueDataType)
}

// CheckFlagInputTypes rejects the first stored input that differs from its declared Go type.
// Repeatable flags require a string slice; inputs without a flag definition are not checked.
func CheckFlagInputTypes(inputs FlagInputs, paramFlags []Flag) error {
	for i := range paramFlags {
		paramFlag := &paramFlags[i]

		stored, present := inputs[paramFlag.FlagID]
		if !present {
			continue
		}

		if !holdsDeclaredType(stored, paramFlag) {
			return fmt.Errorf("%q, %w", paramFlag.FlagID, errs.ErrParamValueTypeMismatch)
		}
	}

	return nil
}

// holdsDeclaredType checks a parsed value against its flag's scalar or repeatable type.
func holdsDeclaredType(stored any, paramFlag *Flag) bool {
	if paramFlag.AllowMultiple {
		_, ok := stored.([]string)

		return ok
	}

	switch paramFlag.DataType {
	case DataString:
		_, ok := stored.(string)

		return ok
	case DataNumber:
		_, ok := stored.(float64)

		return ok
	case DataInteger:
		_, ok := stored.(int)

		return ok
	case DataBoolean:
		_, ok := stored.(bool)

		return ok
	}

	return false
}

// FormatValue returns the textual representation of a parameter value.
func FormatValue(paramVal any) string {
	switch typedVal := paramVal.(type) {
	case string:
		return typedVal
	case float64:
		return strconv.FormatFloat(typedVal, 'f', -1, 64)
	case int:
		return strconv.Itoa(typedVal)
	case bool:
		return strconv.FormatBool(typedVal)
	}

	return fmt.Sprintf("%v", paramVal)
}
