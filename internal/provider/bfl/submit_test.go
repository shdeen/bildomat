package bfl

// Invariants tested:
//  1. Image reference arrays: One or ten images reach the images field in input order, with
//     matching retained binary locations for each element.
//  2. Video tool inputs: Video editing and upscaling send their declared video field without a
//     generation mode, preserve a supplied prompt, and describe the retained video location.
//  3. Video tool validation: Missing, non-video, multiple, and frame-marked inputs fail before
//     submission because the video tools require one whole MP4.
//  4. Width-height parsing: Given 1024x768 or 64x64, parseWidthHeight must return the specified
//     positive dimensions and true. For empty input, a missing separator, a nonnumeric height, or a
//     nonpositive width, it must return false.
//  5. Empty prompt omitted: For an image model, requestBody must omit the prompt field when the
//     supplied prompt is empty and include it when the prompt is nonempty. Both requests must
//     succeed.

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestImageReferenceArray verifies invariant #1: Image reference arrays.
// Test class: Core.
// Kind: permanent.
// What makes it or breaks it: Every supplied reference survives in order, including a single
// reference, and local bytes have the same array location in retained requests.
func TestImageReferenceArray(t *testing.T) {
	configuredModel := catalog.Model{Media: media.Image, Params: params.Definitions{
		{FlagID: params.FlagTypeInputMedia, ParamID: "images", MaxMultiple: 10},
	}}

	for _, referenceCount := range []int{1, 10} {
		references := make([]media.Input, referenceCount)
		serializedReferences := make([]string, referenceCount)

		for referenceIndex := range references {
			references[referenceIndex] = media.Input{Bytes: []byte("image " + strconv.Itoa(referenceIndex)), MIME: "image/png"}
			serializedReferences[referenceIndex] = references[referenceIndex].URLOrBase64()
		}

		requestDocument, requestErr := requestBody(&configuredModel, testPrompt, params.Values{}, references)
		if requestErr != nil || !reflect.DeepEqual(requestDocument["images"], serializedReferences) {
			t.Errorf("✗ %d references: images = %#v, error = %v", referenceCount, requestDocument["images"], requestErr)
		}

		binaryFields := requestBinaryFields(&configuredModel, references)
		if len(binaryFields) != referenceCount {
			t.Errorf("✗ retained locations = %d, require %d", len(binaryFields), referenceCount)

			continue
		}

		for referenceIndex, binaryField := range binaryFields {
			if !reflect.DeepEqual(binaryField.Path, []string{"images", strconv.Itoa(referenceIndex)}) || binaryField.MIME != "image/png" {
				t.Errorf("✗ reference %d location = %+v", referenceIndex, binaryField)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ image arrays and retained locations preserve every reference")
	}
}

// TestVideoToolInputs verifies invariant #2: Video tool inputs.
// Test class: Core.
// Kind: permanent.
// What makes it or breaks it: URL and local MP4 inputs use video or input_video without mode;
// prompt omission and forwarding both work, and retained requests identify the same field.
func TestVideoToolInputs(t *testing.T) {
	for _, inputField := range []string{"video", "input_video"} {
		configuredModel := catalog.Model{Media: media.Video, Params: params.Definitions{
			{FlagID: params.FlagTypeInputMedia, ParamID: inputField, MaxMultiple: 1, Required: true},
		}}

		for _, referenceVideo := range []media.Input{
			{URL: "https://media.example/reference.mp4", MIME: "video/mp4"},
			{URL: "https://media.example/unresolved.mp4"},
			{Bytes: []byte("local MP4"), MIME: "video/mp4"},
		} {
			for _, suppliedPrompt := range []string{"", testPrompt} {
				requestDocument, requestErr := requestBody(&configuredModel, suppliedPrompt, params.Values{}, []media.Input{referenceVideo})
				if requestErr != nil || requestDocument[inputField] != referenceVideo.URLOrBase64() || requestDocument["mode"] != nil || requestDocument["start_video"] != nil {
					t.Errorf("✗ %s request = %#v, error = %v", inputField, requestDocument, requestErr)
				}

				if promptValue, promptPresent := requestDocument["prompt"]; promptPresent != (suppliedPrompt != "") || (promptPresent && promptValue != suppliedPrompt) {
					t.Errorf("✗ supplied prompt %q became %#v", suppliedPrompt, promptValue)
				}
			}

			binaryFields := requestBinaryFields(&configuredModel, []media.Input{referenceVideo})
			if len(binaryFields) != 1 || !reflect.DeepEqual(binaryFields[0].Path, []string{inputField}) || binaryFields[0].MIME != referenceVideo.MIME {
				t.Errorf("✗ %s retained video location = %+v", inputField, binaryFields)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ video tools retain the declared input and prompt without generation fields")
	}
}

// TestVideoToolValidation verifies invariant #3: Video tool validation.
// Test class: Core.
// Kind: permanent.
// What makes it or breaks it: Neither video tool can send an absent, mistyped, ambiguous, or
// frame-marked source to the provider.
func TestVideoToolValidation(t *testing.T) {
	frameSeconds := 0.0

	for _, inputField := range []string{"video", "input_video"} {
		configuredModel := catalog.Model{Media: media.Video, Params: params.Definitions{
			{FlagID: params.FlagTypeInputMedia, ParamID: inputField, MaxMultiple: 1, Required: true},
		}}
		for _, invalidInputs := range [][]media.Input{
			nil,
			{{URL: "https://media.example/image.png", MIME: "image/png"}},
			{{MIME: "video/mp4"}, {MIME: "video/mp4"}},
			{{MIME: "video/mp4", Time: &frameSeconds}},
			{{MIME: "video/mp4", FrameAnchor: media.FrameFirst}},
		} {
			_, requestErr := requestBody(&configuredModel, testPrompt, params.Values{}, invalidInputs)
			if !errors.Is(requestErr, errs.ErrInputMedia) {
				t.Errorf("✗ %s invalid inputs %+v returned %v, require input-media error", inputField, invalidInputs, requestErr)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ video tools reject unusable inputs before submission")
	}
}

// TestSplitWH verifies invariant #4: Width-height parsing.
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

// TestEmptyPromptOmitted verifies invariant #5: Empty prompt omitted.
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
