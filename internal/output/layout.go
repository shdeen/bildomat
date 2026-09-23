package output

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// terminalSequenceExpression is the regular expression matching one terminal style sequence, which
// occupies no column.
const terminalSequenceExpression = "\x1b\\[[0-9;]*m"

// terminalSequence is terminalSequenceExpression compiled once at startup.
var terminalSequence = regexp.MustCompile(terminalSequenceExpression)

// textWidth counts runes after removing terminal style sequences, treating each remaining rune as
// one column.
func textWidth(text string) int {
	return utf8.RuneCountInString(terminalSequence.ReplaceAllString(text, ""))
}

// padText pads text to the requested width, leaving at least one trailing space.
func padText(text string, width int) string {
	if textWidth(text) >= width {
		return text + " "
	}

	return text + strings.Repeat(" ", width-textWidth(text))
}

// indentTo prefixes text with the requested number of spaces.
func indentTo(column int, text string) string {
	return strings.Repeat(" ", column) + text
}

// flowPhrases joins phrases within the requested width, wrapping long phrases at spaces. An
// indivisible term may exceed the width; empty input returns no lines.
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

// wrapWide wraps text at spaces only when it exceeds the requested width.
func wrapWide(flowedLine string, width int) []string {
	if textWidth(flowedLine) <= width || !strings.Contains(flowedLine, " ") {
		return []string{flowedLine}
	}

	return wrapWords(flowedLine, width)
}

// wrapWords collapses whitespace and wraps words within the requested width. An indivisible word
// may exceed the width; whitespace-only input returns no lines.
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
