package output

import (
	"strings"
	"testing"
)

// Invariants tested:
//  1. Paragraph wrapping beyond the page width: Given a 100-character word followed by another
//     word, wrapParagraph must leave the long word whole, place the next word on a new line, and
//     indent both lines by eight spaces. A lone long word must remain on one indented line.

// TestWrapParagraphOverrun verifies invariant #1: Paragraph wrapping beyond the page width.
//
// What is being tested:
// Given a 100-character word followed by another word, wrapParagraph must leave the long word
// whole, place the next word on a new line, and indent both lines by eight spaces. A lone long word
// must remain on one indented line.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWrapParagraphOverrun(t *testing.T) {
	wideTerm := strings.Repeat("x", 100)
	got := wrapParagraph(2, wideTerm+" tail")

	wantLines := []string{"        " + wideTerm, "        tail"}
	if got != strings.Join(wantLines, "\n") {
		t.Errorf("✗ over-wide term wrap = %q, want the overrun line then the tail", got)
	}

	if got := wrapParagraph(2, wideTerm); got != "        "+wideTerm {
		t.Errorf("✗ a lone over-wide term = %q, want the single overrun line", got)
	}

	if !t.Failed() {
		t.Log("✓ an over-wide term overruns whole and wrapping resumes after it")
	}
}
