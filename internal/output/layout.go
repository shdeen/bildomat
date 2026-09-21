package output

// File: internal/output/layout.go
// The column primitives the info pages lay out with: measuring text in
// columns, padding text to a column, flowing separated phrases within a
// width, wrapping words within a width, and indenting a line to a column.

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// terminalSequenceExpression is the regular expression matching one terminal
// style sequence, which occupies no column.
const terminalSequenceExpression = "\x1b\\[[0-9;]*m"

// terminalSequence is terminalSequenceExpression compiled once at startup.
var terminalSequence = regexp.MustCompile(terminalSequenceExpression)

// textWidth takes text and returns the number of columns it occupies, one
// per rune, terminal style sequences occupying none.
func textWidth(text string) int {
	return utf8.RuneCountInString(terminalSequence.ReplaceAllString(text, ""))
}

// padText takes text and a column width and returns the text padded with
// spaces to the width, or followed by one space when it fills the width, so
// what follows always starts after a gap.
func padText(text string, width int) string {
	if textWidth(text) >= width {
		return text + " "
	}

	return text + strings.Repeat(" ", width-textWidth(text))
}

// indentTo takes a column and text and returns the text preceded by that
// many spaces.
func indentTo(column int, text string) string {
	return strings.Repeat(" ", column) + text
}

// flowPhrases takes phrases, the separator that joins them, and a width, and
// returns the phrases flowed greedily into lines within the width, breaking
// between phrases; a phrase wider than the width fills a line alone, wrapped
// on its own spaces, and a single term wider than the width overruns alone.
// No phrases return no lines.
func flowPhrases(phrases []string, separator string, width int) []string {
	var lines []string

	current := ""

	for _, phrase := range phrases {
		switch {
		case current == "":
			current = phrase
		case textWidth(current)+textWidth(separator)+textWidth(phrase) <= width:
			current += separator + phrase
		default:
			lines = append(lines, wrapWide(current, width)...)
			current = phrase
		}
	}

	if current != "" {
		lines = append(lines, wrapWide(current, width)...)
	}

	return lines
}

// wrapWide takes one flowed line and the width, and returns the line as it is
// when it fits or holds a single term, and otherwise wrapped on its spaces.
func wrapWide(flowedLine string, width int) []string {
	if textWidth(flowedLine) <= width || !strings.Contains(flowedLine, " ") {
		return []string{flowedLine}
	}

	return wrapWords(flowedLine, width)
}

// wrapWords takes text and a width and returns the text wrapped on its
// spaces within the width, every run of whitespace collapsed to one space; a
// word wider than the width overruns alone. Text holding no words returns no
// lines.
func wrapWords(text string, width int) []string {
	var lines []string

	current := ""

	for word := range strings.FieldsSeq(text) {
		switch {
		case current == "":
			current = word
		case textWidth(current)+1+textWidth(word) <= width:
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}
