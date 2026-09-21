package output

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// File: internal/output/constraints.go
// The sentences the info pages render for a parameter's declared
// constraints, and the display words of the media.

// mediaLabels maps each medium to the word the details pages show for it.
//
//nolint:gochecknoglobals // a read-only table, written only at package load; a map cannot be a constant.
var mediaLabels = map[media.Kind]string{
	media.Image: MediaLabelImages,
	media.Video: MediaLabelVideo,
}

// constraintPhrases takes a parameter definition, its configuration, and
// the page style, and returns one phrase per declared constraint in the
// catalog's forms, each value in the style's value accent (none for the
// plain style).
func constraintPhrases(paramFlag *params.Flag, paramCfg *params.Definition, style pageStyle) []string {
	var phrases []string

	if len(paramCfg.AllowedValues) > 0 {
		phrases = append(phrases, fmt.Sprintf(AllowedValues, accentValues(paramCfg.AllowedValues, ", ", style)))
	}

	minVal, hasMin := paramCfg.MinValue.ValIf()
	maxVal, hasMax := paramCfg.MaxValue.ValIf()

	if bound := boundPhrase(AllowedRange, MaximumValue, MinimumValue, accentValue(rangeBound(paramFlag.DataType, minVal), style), accentValue(rangeBound(paramFlag.DataType, maxVal), style), hasMin, hasMax); bound != "" {
		phrases = append(phrases, bound)
	}

	if paramCfg.MaxMultiple > 0 {
		phrases = append(phrases, fmt.Sprintf(RepeatMaximum, accentValue(strconv.Itoa(paramCfg.MaxMultiple), style)))
	}

	if paramCfg.CustomSize != nil {
		phrases = append(phrases, sizeBoundsNotice(*paramCfg.CustomSize))
	}

	if paramCfg.RuleDescription != "" {
		phrases = append(phrases, fmt.Sprintf(RuleDescription, paramCfg.RuleDescription))
	}

	return phrases
}

// rangeBound takes a data type and numeric bound and returns the bound in the data type's display form.
func rangeBound(dataType params.DataType, bound float64) string {
	if dataType == params.DataInteger {
		return strconv.Itoa(int(bound))
	}

	return params.FormatValue(bound)
}

// boundPhrase takes the forms for a two-bound range, a lone maximum, and a
// lone minimum, the two bounds, and whether each is declared, and returns the
// phrase stating the declared bounds, or an empty string when neither is.
func boundPhrase(rangeForm, maximumForm, minimumForm string, minVal, maxVal any, hasMin, hasMax bool) string {
	switch {
	case hasMin && hasMax:
		return fmt.Sprintf(rangeForm, minVal, maxVal)
	case hasMax:
		return fmt.Sprintf(maximumForm, maxVal)
	case hasMin:
		return fmt.Sprintf(minimumForm, minVal)
	}

	return ""
}

// sizeBoundsNotice takes size constraints and returns a notice containing each positive bound.
func sizeBoundsNotice(bounds params.SizeBounds) string {
	var constraints []string

	if bounds.MaxRatio > 0 {
		constraints = append(constraints, fmt.Sprintf(RatioBetween, params.FormatValue(bounds.MaxRatio)))
	}

	if edge := boundPhrase(EdgeRange, MaximumEdge, MinimumEdge, bounds.MinEdge, bounds.MaxEdge, bounds.MinEdge > 0, bounds.MaxEdge > 0); edge != "" {
		constraints = append(constraints, edge)
	}

	if pixels := boundPhrase(PixelRange, MaximumPixels, MinimumPixels, bounds.MinPx, bounds.MaxPx, bounds.MinPx > 0, bounds.MaxPx > 0); pixels != "" {
		constraints = append(constraints, pixels)
	}

	if bounds.EdgeIncrem > 0 {
		constraints = append(constraints, fmt.Sprintf(EdgeIncrements, bounds.EdgeIncrem))
	}

	if bounds.LongEdge > 0 {
		constraints = append(constraints, fmt.Sprintf(LongEdge, bounds.LongEdge))
	}

	return fmt.Sprintf(SizeRequirements, strings.Join(constraints, ", "))
}
