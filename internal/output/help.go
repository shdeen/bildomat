package output

import (
	"strings"

	"github.com/shdeen/bildomat/internal/params"
)

// File: internal/output/help.go
// The help pages this package owns: the page measures every page lays out to,
// the indentation step, the dashed flag heading, the paragraph wrapping, and
// the value words for the flags the command layer declares directly. The help
// page copy itself lives in internal/templates.

// The layout measures of the rendered pages. Section labels sit flush left and
// each nested level steps in by one indent.
//   - PageWidth: the column at which page text wraps
//   - IndentWidth: the number of spaces in one indentation step
const (
	PageWidth   = 80
	IndentWidth = 4
)

// IndentText takes a number of indentation steps and a block of text, and
// returns that text with every line indented by the given number of steps.
func IndentText(steps int, text string) string {
	indentChars := strings.Repeat(" ", steps*IndentWidth)

	return indentChars + strings.ReplaceAll(text, "\n", "\n"+indentChars)
}

// DashedFlagNames takes a flag's names and the word that names its value, and
// returns them as one comma-separated heading. Every name carries a dash
// prefix, the aliases come first and the long name last, and the heading ends
// with the value word in angle brackets where the flag accepts a value.
func DashedFlagNames(flagNames []string, valueHint string) string {
	if len(flagNames) == 0 {
		return ""
	}
	// The library returns the long name first, then the aliases.
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

// dashedFlagName takes one of a flag's names and returns it with a dash prefix:
// a single dash for a one-character name, and a double dash for a longer name.
func dashedFlagName(flagName string) string {
	if len(flagName) == 1 {
		return "-" + flagName
	}

	return "--" + flagName
}

// WrapDetails takes a block of detail text and the number of indentation steps
// to lay it out at, and returns it wrapped to the page width. It preserves the
// blank line between paragraphs and drops any paragraph that holds no words.
func WrapDetails(steps int, detailText string) string {
	var wrappedParagraphs []string

	for paragraph := range strings.SplitSeq(detailText, "\n\n") {
		if wrapped := wrapParagraph(steps, paragraph); wrapped != "" {
			wrappedParagraphs = append(wrappedParagraphs, wrapped)
		}
	}

	return strings.Join(wrappedParagraphs, "\n\n")
}

// wrapParagraph takes one paragraph and the number of indentation steps to lay
// it out at, and returns it broken to the page width and indented to that
// level, with every run of whitespace collapsed to a single space. A paragraph
// holding no words returns an empty string, and a term wider than the margin
// overruns it rather than being split.
func wrapParagraph(steps int, paragraphText string) string {
	wrappedLines := wrapWords(paragraphText, PageWidth-steps*IndentWidth)
	if len(wrappedLines) == 0 {
		return ""
	}

	indentChars := strings.Repeat(" ", steps*IndentWidth)

	return indentChars + strings.Join(wrappedLines, "\n"+indentChars)
}

// FlagValueHint takes a flag's long name and the parameter enumeration, and
// returns the word naming that flag's value: the hint on the matching
// parameter record, then the hint declared for a directly declared flag, and
// otherwise the long name itself.
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

// FlagEntry takes a flag's names, the word naming its value, and its detail
// text, and returns that flag's help page entry: the dashed names and value
// word on the first line, with the wrapped details beneath.
func FlagEntry(flagNames []string, valueHint, detailText string) string {
	return IndentText(1, DashedFlagNames(flagNames, valueHint)) + "\n" + WrapDetails(2, detailText)
}
