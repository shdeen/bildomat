package params

// Change identifies how a generation parameter was altered.
type Change string

// Change values identify the supported parameter adjustments.
//   - ChangeSnapped: replaced by the nearest supported value
//   - ChangeDropped: omitted because the supplied value could not be used
//   - ChangeRejected: omitted because the supplied value was not allowed
//   - ChangeCapped: reduced to a maximum value
//   - ChangeRaised: increased to a minimum value
//   - ChangeForced: replaced by a provider-required value
//   - ChangeDerived: calculated from another supplied value
//   - ChangeConformed: altered to match a model requirement
//   - ChangeIgnored: omitted because another value superseded it or the model does not use it
const (
	ChangeSnapped   Change = "snapped"
	ChangeDropped   Change = "dropped"
	ChangeRejected  Change = "rejected"
	ChangeCapped    Change = "capped"
	ChangeRaised    Change = "raised"
	ChangeForced    Change = "forced"
	ChangeDerived   Change = "derived"
	ChangeConformed Change = "conformed"
	ChangeIgnored   Change = "ignored"
)

// Definition contains a model's provider mapping and constraints for one parameter.
//   - ParamID: the provider request key, or empty when request code handles the parameter
//   - FlagID: the command flag that supplies the parameter
//   - AllowedValues: the accepted parameter values
//   - MinValue: the optional minimum numeric value
//   - MaxValue: the optional maximum numeric value
//   - MaxMultiple: the maximum number of values accepted for a repeatable parameter
//   - CustomSize: the optional constraints for free-form dimensions
//   - RuleDescription: the user-facing description of a parameter adjustment rule
//   - ModelInfoComment: expanded model-specific guidance appended on the model's details page
//   - Required: whether the provider's documentation names the parameter as required;
//     a record declaring nothing is optional
//
// The encoding carries only what a record declares: the request key and every
// constraint are absent when empty, and an unset bound is absent rather than null.
type Definition struct {
	ParamID          string            `json:"paramID,omitempty"`
	FlagID           FlagType          `json:"flagID"`
	Required         bool              `json:"required,omitempty"`
	AllowedValues    []string          `json:"allowedValues,omitempty"`
	MinValue         Nullable[float64] `json:"minValue,omitzero"`
	MaxValue         Nullable[float64] `json:"maxValue,omitzero"`
	MaxMultiple      int               `json:"maxMultiple,omitempty"`
	CustomSize       *SizeBounds       `json:"customSize,omitempty"`
	RuleDescription  string            `json:"ruleDescription,omitempty"`
	ModelInfoComment string            `json:"modelInfoComment,omitempty"`
}

// Definitions contains the declared parameter configurations for a model.
type Definitions []Definition

// Param returns a flag's declared configuration and whether it exists.
func (definitions Definitions) Param(flag FlagType) (Definition, bool) {
	for index := range definitions {
		if definitions[index].FlagID == flag {
			return definitions[index], true
		}
	}

	return Definition{}, false
}
