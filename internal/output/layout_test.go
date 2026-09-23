package output

import (
	"strings"
	"testing"
)

// Invariants tested:
//  1. Flow phrase wrapping widths: Given phrases and a line width, flowPhrases must return the
//     expected lines: join phrases that fit, wrap long phrases at spaces, and leave an indivisible
//     URL whole on its own line. With no phrases, it must return no text.

// TestFlowPhrasesWidths verifies invariant #1: Flow phrase wrapping widths.
//
// What is being tested:
// Given phrases and a line width, flowPhrases must return the expected lines: join phrases that
// fit, wrap long phrases at spaces, and leave an indivisible URL whole on its own line. With no
// phrases, it must return no text.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFlowPhrasesWidths(t *testing.T) {
	cases := []struct {
		name      string
		phrases   []string
		separator string
		width     int
		want      []string
	}{
		{"fits on one line", []string{"1:1", "2:3", "3:2"}, "  ", 20, []string{"1:1  2:3  3:2"}},
		{"breaks between phrases", []string{"1:1", "2:3", "3:2"}, "  ", 8, []string{"1:1  2:3", "3:2"}},
		{"wraps a wide phrase on its spaces", []string{"a phrase of many small words"}, " · ", 10, []string{"a phrase", "of many", "small", "words"}},
		{"overruns a wide term", []string{"https://example.com/reference.mp4", "x"}, " · ", 10, []string{"https://example.com/reference.mp4", "x"}},
		{"no phrases", nil, " · ", 10, nil},
	}
	for _, c := range cases {
		got := flowPhrases(c.phrases, c.separator, c.width)
		if strings.Join(got, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("✗ %s: flowPhrases = %q, want %q", c.name, got, c.want)
		}
	}

	if !t.Failed() {
		t.Log("✓ phrases flow within the width, wide phrases wrap, and lone wide terms overrun")
	}
}
