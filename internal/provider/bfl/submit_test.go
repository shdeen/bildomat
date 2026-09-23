package bfl

// Invariants tested:
//  1. Width-height parsing: Given 1024x768 or 64x64, parseWidthHeight must return the specified
//     positive dimensions and true. For empty input, a missing separator, a nonnumeric height, or a
//     nonpositive width, it must return false.
//  2. Empty prompt omitted: For an image model, requestBody must omit the prompt field when the
//     supplied prompt is empty and include it when the prompt is nonempty. Both requests must
//     succeed.

import (
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestSplitWH verifies invariant #1: Width-height parsing.
//
// What is being tested:
// Given 1024x768 or 64x64, parseWidthHeight must return the specified positive dimensions and true.
// For empty input, a missing separator, a nonnumeric height, or a nonpositive width, it must return
// false.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSplitWH(t *testing.T) {
	cases := []struct {
		in     string
		w, h   int
		wantOK bool
	}{
		{"1024x768", 1024, 768, true},
		{"64x64", 64, 64, true},
		{"1024", 0, 0, false},
		{"1024xABC", 0, 0, false},
		{"0x100", 0, 0, false},
		{"-8x8", 0, 0, false},
		{"", 0, 0, false},
	}
	for _, c := range cases {
		w, h, ok := parseWidthHeight(c.in)
		if ok != c.wantOK || (ok && (w != c.w || h != c.h)) {
			t.Errorf("✗ parseWidthHeight(%q) = %d,%d,%v; want %d,%d,%v", c.in, w, h, ok, c.w, c.h, c.wantOK)
		}
	}

	if !t.Failed() {
		t.Log("✓ parseWidthHeight parses valid WxH and rejects malformed, zero, and negative edges")
	}
}

// TestEmptyPromptOmitted verifies invariant #2: Empty prompt omitted.
//
// What is being tested:
// For an image model, requestBody must omit the prompt field when the supplied prompt is empty and
// include it when the prompt is nonempty. Both requests must succeed.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestEmptyPromptOmitted(t *testing.T) {
	model := catalog.Model{ID: "fixture-model", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeInputMedia, ParamID: "image", MaxMultiple: 1}}}

	for _, prompt := range []string{"", testPrompt} {
		body, err := requestBody(&model, prompt, params.Values{}, nil)
		if err != nil {
			t.Fatalf("💣 requestBody failed: %v", err)
		}

		if _, present := body["prompt"]; present != (prompt != "") {
			t.Errorf("✗ body with prompt %q: prompt field present = %t", prompt, present)
		}
	}

	if !t.Failed() {
		t.Log("✓ an empty prompt leaves the job body without a prompt field")
	}
}

// testPrompt is the fixture prompt; the body-assembly cases assert it is sent in every submit body.
const testPrompt = "a plain gray circle"
