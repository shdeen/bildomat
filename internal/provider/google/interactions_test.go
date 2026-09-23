package google

// Invariants tested:
//  1. Interaction input media sources: Given local image bytes, a remote image URL, a remote video
//     URL, and a prompt, interactionInput must return input blocks in that order. The local image
//     must use base64 data, remote inputs must preserve their URIs and MIME types, and the last
//     block must contain the supplied text.
//  2. Declared parameter placement: Given aspect and output-format parameters declared under
//     response_format, interactionImageBody must place 16:9 and png at those paths and preserve the
//     fixed type=image field without an error.

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestInteractionInputMediaSources verifies invariant #1: Interaction input media sources.
//
// What is being tested:
// Given local image bytes, a remote image URL, a remote video URL, and a prompt, interactionInput
// must return input blocks in that order. The local image must use base64 data, remote inputs must
// preserve their URIs and MIME types, and the last block must contain the supplied text.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestInteractionInputMediaSources(t *testing.T) {
	mediaInputs := []media.Input{
		{Bytes: []byte("PNG"), MIME: "image/png", Filepath: "/tmp/local.png"},
		{URL: "https://media.example/reference.webp", MIME: "image/webp"},
		{URL: "https://media.example/source.mp4", MIME: "video/mp4"},
	}

	body := interactionInput("gemini-model", "make media", mediaInputs)

	blocks, ok := body["input"].([]map[string]any)
	if !ok {
		t.Fatalf("💣 input blocks = %#v, want []map[string]any", body["input"])
	}

	want := []map[string]any{
		{"type": "image", "mime_type": "image/png", "data": base64.StdEncoding.EncodeToString([]byte("PNG"))},
		{"type": "image", "mime_type": "image/webp", "uri": mediaInputs[1].URL},
		{"type": "video", "mime_type": "video/mp4", "uri": mediaInputs[2].URL},
		{"type": "text", "text": "make media"},
	}
	if !reflect.DeepEqual(blocks, want) {
		t.Errorf("✗ interaction input = %#v, want %#v", blocks, want)
	}
}

// TestInteractionBodyPlacesDeclaredPath verifies invariant #2: Declared parameter placement.
//
// What is being tested:
// Given aspect and output-format parameters declared under response_format, interactionImageBody
// must place 16:9 and png at those paths and preserve the fixed type=image field without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestInteractionBodyPlacesDeclaredPath(t *testing.T) {
	model := catalog.Model{ID: "synthetic-interactions-image", Media: media.Image, Params: []params.Definition{
		{FlagID: params.FlagTypeAspect, ParamID: "response_format.aspect_ratio"},
		{FlagID: params.FlagTypeOutputFormat, ParamID: "response_format.mime_type"},
	}}
	parameterValues := params.Values{params.FlagTypeAspect: "16:9", params.FlagTypeOutputFormat: "png"}

	body, err := interactionImageBody(&model, "a prompt", parameterValues, nil)
	if err != nil {
		t.Fatalf("💣 interactionImageBody: %v", err)
	}

	rf := sub(t, body, "response_format")
	wantStr(t, rf, "type", "image")
	wantStr(t, rf, "aspect_ratio", "16:9")
	wantStr(t, rf, "mime_type", "png")

	if !t.Failed() {
		t.Log("✓ a declared dotted path places its value beside the fixed response-format type")
	}
}
