package output

import (
	"strings"

	"github.com/shdeen/bildomat/internal/params"
)

// The layout measures of the rendered pages. Section labels sit flush left and each nested level
// steps in by one indent.
//   - pageWidth: the column at which page text wraps
//   - indentWidth: the number of spaces in one indentation step
const (
	pageWidth   = 80
	indentWidth = 4
)

// IndentText indents every line by the requested number of indentation steps.
func IndentText(steps int, text string) string {
	indentChars := strings.Repeat(" ", steps*indentWidth)

	return indentChars + strings.ReplaceAll(text, "\n", "\n"+indentChars)
}

// dashedFlagNames formats aliases before the long flag name, with their dash prefixes. A nonempty
// value hint follows the names in angle brackets.
func dashedFlagNames(flagNames []string, valueHint string) string {
	if len(flagNames) == 0 {
		return ""
	}
	// The input lists the long name first, followed by aliases.
	dashedNames := make([]string, 0, len(flagNames))
	for _, alias := range flagNames[1:] {
		if alias != "" {
			dashedNames = append(dashedNames, dashedFlagName(alias))
		}
	}

	dashedNames = append(dashedNames, dashedFlagName(flagNames[0]))
	if valueHint == "" {
		return strings.Join(dashedNames, ", ")
	}

	return strings.Join(dashedNames, ", ") + " <" + valueHint + ">"
}

// dashedFlagName prefixes a one-byte name with one dash and longer names with two.
func dashedFlagName(flagName string) string {
	if len(flagName) == 1 {
		return "-" + flagName
	}

	return "--" + flagName
}

// WrapDetails wraps and indents paragraphs to the page width, separated by blank lines.
func WrapDetails(steps int, detailText string) string {
	var wrappedParagraphs []string

	for paragraph := range strings.SplitSeq(detailText, "\n\n") {
		if wrapped := wrapParagraph(steps, paragraph); wrapped != "" {
			wrappedParagraphs = append(wrappedParagraphs, wrapped)
		}
	}

	return strings.Join(wrappedParagraphs, "\n\n")
}

// wrapParagraph wraps and indents one paragraph, collapsing whitespace. Indivisible terms may
// exceed the available width; empty paragraphs return no text.
func wrapParagraph(steps int, paragraphText string) string {
	wrappedLines := wrapWords(paragraphText, pageWidth-steps*indentWidth)
	if len(wrappedLines) == 0 {
		return ""
	}

	indentChars := strings.Repeat(" ", steps*indentWidth)

	return indentChars + strings.Join(wrappedLines, "\n"+indentChars)
}

// FlagValueHint returns the parameter hint when present. Otherwise, it returns the command flag
// hint, or the long flag name if neither hint exists.
func FlagValueHint(longName string, paramFlags []params.Flag, flagHints map[string]string) string {
	for paramFlagIndex := range paramFlags {
		paramFlag := &paramFlags[paramFlagIndex]
		if string(paramFlag.FlagID) == longName && paramFlag.TextHint != "" {
			return paramFlag.TextHint
		}
	}

	if hint := flagHints[longName]; hint != "" {
		return hint
	}

	return longName
}

// FlagEntry formats a flag heading followed by its wrapped, indented description.
func FlagEntry(flagNames []string, valueHint, detailText string) string {
	return IndentText(1, dashedFlagNames(flagNames, valueHint)) + "\n" + WrapDetails(2, detailText)
}
