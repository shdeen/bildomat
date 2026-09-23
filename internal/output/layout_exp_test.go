package output

import (
	"slices"
	"strings"
	"testing"
)

// Invariants tested:
//  1. Flow phrase wrapping under arbitrary widths: For widths from 1 through 200 and the supported
//     separators, flowPhrases must preserve the input words in order. Every output line that joins
//     phrases with a separator must fit within the requested width.

// FuzzFlowPhrases verifies invariant #1: Flow phrase wrapping under arbitrary widths.
//
// What is being tested:
// For widths from 1 through 200 and the supported separators, flowPhrases must preserve the input
// words in order. Every output line that joins phrases with a separator must fit within the
// requested width.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzFlowPhrases(f *testing.F) {
	f.Add("1:1 2:3 3:2 16:9", "  ", 12)
	f.Add("min edge 64|max pixels 4194304|long edge 1024", " · ", 38)
	f.Add("a phrase of many small words", " ", 5)
	f.Add("", " ", 1)

	f.Fuzz(func(t *testing.T, joined, separator string, width int) {
		// The separators are the pages' own; an arbitrary separator whose runes recur
		// inside the phrases would make the word comparison below ambiguous.
		if width < 1 || width > 200 || !slices.Contains([]string{" ", "  ", " · ", ", "}, separator) || strings.Contains(joined, separator) {
			return
		}

		phrases := strings.Split(joined, "|")
		lines := flowPhrases(phrases, separator, width)

		for _, flowedLine := range lines {
			if strings.Contains(flowedLine, separator) && textWidth(flowedLine) > width {
				t.Errorf("✗ the flowed line %q holds several phrases yet exceeds the width %d", flowedLine, width)
			}
		}

		gotWords := strings.Fields(strings.ReplaceAll(strings.Join(lines, " "), separator, " "))
		wantWords := strings.Fields(strings.ReplaceAll(strings.Join(phrases, " "), separator, " "))

		if strings.Join(gotWords, " ") != strings.Join(wantWords, " ") {
			t.Errorf("✗ flowing %q with %q at %d lost or reordered words: %q", joined, separator, width, lines)
		}

		if !t.Failed() {
			t.Logf("✓ %q flowed with %q at %d within the width and intact", joined, separator, width)
		}
	})
}
