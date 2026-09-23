package provider

// Invariants tested:
//  1. String-valued wire parameters: Given duration eight, image count two, and duration selected
//     for string conversion, WireParamValues must return seconds as string "8" and count as integer
//     2.
//  2. String-style input media payload: Given the string media style, addInputMedia must preserve
//     one URL, encode local PNG bytes in a decodable data URI, and preserve several inputs as
//     ordered strings.
//  3. Multipart fixed provider fields: Given response_format=b64_json in FixedProvFields, formBody
//     must include that exact multipart text value alongside model and prompt, with no other text
//     fields.
//  4. Dotted request paths: Given parameter IDs sharing dotted prefixes, WireParamValues must build
//     the expected response_format and generation_config.thinking objects, retain count as a
//     top-level integer, and return exactly three top-level fields.
//  5. Empty prompt omitted: Given an empty prompt, jsonBody and formBody must omit the prompt
//     field.
//  6. Configured video controls: Given the listed Hailuo, FLUX, and Avatar inputs, AdjustGeneration
//     must return no changes, and WireParamValues must produce exactly the expected nested request
//     fields, preserving false, zero, numeric controls, and arbitrary voice text.

import (
	"bytes"
	"encoding/base64"
	"errors"
	"mime"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
	providerconfig "github.com/shdeen/bildomat/internal/provider/config"
)

// TestWireParamValuesStringSelection verifies invariant #1: String-valued wire parameters.
//
// What is being tested:
// Given duration eight, image count two, and duration selected for string conversion,
// WireParamValues must return seconds as string "8" and count as integer 2.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWireParamValuesStringSelection(t *testing.T) {
	model := catalog.Model{Params: []params.Definition{
		{FlagID: params.FlagTypeDuration, ParamID: "seconds"},
		{FlagID: params.FlagTypeImageN, ParamID: "count"},
	}}
	parameterValues := params.Values{params.FlagTypeDuration: 8, params.FlagTypeImageN: 2}

	values := WireParamValues(&model, parameterValues, params.FlagTypeDuration)
	if values["seconds"] != "8" {
		t.Errorf("✗ seconds = %#v, want string %q", values["seconds"], "8")
	}

	if values["count"] != 2 {
		t.Errorf("✗ count = %#v, want integer 2", values["count"])
	}
}

// TestStringMediaStylePayload verifies invariant #2: String-style input media payload.
//
// What is being tested:
// Given the string media style, addInputMedia must preserve one URL, encode local PNG bytes in a
// decodable data URI, and preserve several inputs as ordered strings. A first-frame input must
// return ErrInputMediaTime and leave the empty body unchanged.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestStringMediaStylePayload(t *testing.T) {
	const providerField = "source_image"

	stringMediaStyle := catalog.InputMediaStyle("string")

	t.Run("one URL remains unchanged", func(t *testing.T) {
		checkStringMediaURL(t, stringMediaStyle, providerField)

		if !t.Failed() {
			t.Log("✓ one URL remains unchanged")
		}
	})

	t.Run("one local input becomes a decodable data URI", func(t *testing.T) {
		checkStringMediaDataURI(t, stringMediaStyle, providerField)

		if !t.Failed() {
			t.Log("✓ one local input becomes a decodable data URI")
		}
	})

	t.Run("several inputs remain ordered strings", func(t *testing.T) {
		checkStringMediaArray(t, stringMediaStyle, providerField)

		if !t.Failed() {
			t.Log("✓ several inputs remain ordered strings")
		}
	})

	t.Run("frame prefix is refused without changing the body", func(t *testing.T) {
		checkStringMediaFrameRefusal(t, stringMediaStyle, providerField)

		if !t.Failed() {
			t.Log("✓ a frame prefix is refused without changing the body")
		}
	})

	if !t.Failed() {
		t.Log("✓ the string media form carries bare media strings and keeps the shared frame refusal")
	}
}

// TestFormFixedFields verifies invariant #3: Multipart fixed provider fields.
//
// What is being tested:
// Given response_format=b64_json in FixedProvFields, formBody must include that exact multipart
// text value alongside model and prompt, with no other text fields.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFormFixedFields(t *testing.T) {
	api := catalog.ImageAPI{InputMediaProvParam: "image[]", FixedProvFields: map[string]string{"response_format": "b64_json"}}
	img := media.Input{Bytes: []byte("IMG"), MIME: "image/png", Filepath: "/in/a.png"}

	model := catalog.Model{ID: "m-1"}

	ct, body, err := formBody(&api, &model, "p", params.Values{}, []media.Input{img})
	if err != nil {
		t.Fatalf("💣 form assembly failed: %v", err)
	}

	form := parseForm(t, recReq{header: http.Header{"Content-Type": []string{ct}}, body: body})
	formKeys(t, form, "model", "prompt", "response_format")

	if v := form.Value["response_format"]; len(v) != 1 || v[0] != "b64_json" {
		t.Errorf("✗ response_format = %v, want the verbatim b64_json", v)
	}

	if !t.Failed() {
		t.Log("✓ FixedProvFields are sent in the multipart form verbatim")
	}
}

// TestWireParamValuesPlacesDottedPath verifies invariant #4: Dotted request paths.
//
// What is being tested:
// Given parameter IDs sharing dotted prefixes, WireParamValues must build the expected
// response_format and generation_config.thinking objects, retain count as a top-level integer, and
// return exactly three top-level fields.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWireParamValuesPlacesDottedPath(t *testing.T) {
	model := catalog.Model{Params: []params.Definition{
		{FlagID: params.FlagTypeAspect, ParamID: "response_format.aspect_ratio"},
		{FlagID: params.FlagTypeResolution, ParamID: "response_format.image_size"},
		{FlagID: params.FlagTypeThinkingLevel, ParamID: "generation_config.thinking.level"},
		{FlagID: params.FlagTypeImageN, ParamID: "count"},
	}}
	parameterValues := params.Values{params.FlagTypeAspect: "16:9", params.FlagTypeResolution: "1K", params.FlagTypeThinkingLevel: "high", params.FlagTypeImageN: 2}

	values := WireParamValues(&model, parameterValues)

	responseFormat, ok := values["response_format"].(map[string]any)
	if !ok || responseFormat["aspect_ratio"] != "16:9" || responseFormat["image_size"] != "1K" {
		t.Errorf("✗ response_format = %#v, want aspect_ratio 16:9 and image_size 1K under one object", values["response_format"])
	}

	generationConfig, ok := values["generation_config"].(map[string]any)
	thinking, nested := generationConfig["thinking"].(map[string]any)

	if !ok || !nested || thinking["level"] != "high" {
		t.Errorf("✗ generation_config = %#v, want thinking.level high two objects deep", values["generation_config"])
	}

	if values["count"] != 2 {
		t.Errorf("✗ count = %#v, want the top-level integer 2", values["count"])
	}

	if len(values) != 3 {
		t.Errorf("✗ %d top-level fields %v, want 3", len(values), values)
	}
}

// TestEmptyPromptOmitted verifies invariant #5: Empty prompt omitted.
//
// What is being tested:
// Given an empty prompt, jsonBody and formBody must omit the prompt field. Given a nonempty prompt,
// both must include it.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestEmptyPromptOmitted(t *testing.T) {
	api := &catalog.ImageAPI{InputMediaProvParam: "image", InputMediaStyle: catalog.InputMediaSingle}
	model := &catalog.Model{ID: "fixture-model"}

	for _, prompt := range []string{"", "a fixture prompt"} {
		body, err := jsonBody(api, model, prompt, nil, nil)
		if err != nil {
			t.Fatalf("💣 JSON body assembly failed: %v", err)
		}

		if _, present := body["prompt"]; present != (prompt != "") {
			t.Errorf("✗ JSON body with prompt %q: prompt field present = %t", prompt, present)
		}

		contentType, formBytes, err := formBody(api, model, prompt, nil, nil)
		if err != nil {
			t.Fatalf("💣 multipart body assembly failed: %v", err)
		}

		_, contentParams, err := mime.ParseMediaType(contentType)
		if err != nil {
			t.Fatalf("💣 multipart content type: %v", err)
		}

		form, err := multipart.NewReader(bytes.NewReader(formBytes), contentParams["boundary"]).ReadForm(32 << 20)
		if err != nil {
			t.Fatalf("💣 parse multipart body: %v", err)
		}

		if _, present := form.Value["prompt"]; present != (prompt != "") {
			t.Errorf("✗ multipart body with prompt %q: prompt part present = %t", prompt, present)
		}

		if err := form.RemoveAll(); err != nil {
			t.Fatalf("💣 remove multipart temp files: %v", err)
		}
	}

	if !t.Failed() {
		t.Log("✓ an empty prompt leaves the request without a prompt field")
	}
}

// TestConfiguredVideoControls verifies invariant #6: Configured video controls.
//
// What is being tested:
// Given the listed Hailuo, FLUX, and Avatar inputs, AdjustGeneration must return no changes, and
// WireParamValues must produce exactly the expected nested request fields, preserving false, zero,
// numeric controls, and arbitrary voice text.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestConfiguredVideoControls(t *testing.T) {
	providerConfig := loadTestProvider(t, providerconfig.IDOpenRouter, providerconfig.JSONOpenRouter)

	videoCases := []struct {
		modelID       string
		flagInputs    params.FlagInputs
		requestFields map[string]any
	}{
		{"minimax/hailuo-3-max", params.FlagInputs{"watermark": false, params.FlagTypeDuration: 5, params.FlagTypeResolution: "480p"}, map[string]any{"duration": 5, "resolution": "480p", "provider": map[string]any{"options": map[string]any{"minimax": map[string]any{"aigc_watermark": false}}}}},
		{"black-forest-labs/flux-video-edit", params.FlagInputs{"safety-tolerance": 2}, map[string]any{"provider": map[string]any{"options": map[string]any{"black-forest-labs": map[string]any{"safety_tolerance": 2}}}}},
		{"black-forest-labs/flux-video-upscale", params.FlagInputs{"upscale-factor": 1.5, "creativity": 0}, map[string]any{"upscale_factor": 1.5, "creativity": 0}},
		{"heygen/avatar-iv", params.FlagInputs{"voice-id": "provider-voice-identifier", "motion-prompt": "a small wave"}, map[string]any{"provider": map[string]any{"options": map[string]any{"heygen": map[string]any{"voice_id": "provider-voice-identifier", "motion_prompt": "a small wave"}}}}},
	}
	for _, videoCase := range videoCases {
		t.Run(videoCase.modelID, func(t *testing.T) {
			model := configuredModel(t, providerConfig, videoCase.modelID)

			preparedGeneration, adjustmentErr := generation.AdjustGeneration(&model, videoCase.flagInputs, nil)
			parameterValues, changes := preparedGeneration.Params, preparedGeneration.Changes

			if adjustmentErr != nil {
				t.Errorf("✗ supplied controls failed adjustment: %v", adjustmentErr)
			}

			requestFields := WireParamValues(&model, parameterValues)
			if len(changes) != 0 || !reflect.DeepEqual(requestFields, videoCase.requestFields) {
				t.Errorf("✗ supplied controls changed: fields=%#v changes=%+v", requestFields, changes)
			}

			if !t.Failed() {
				t.Log("✓ supplied scalar controls reach the provider fields unchanged")
			}
		})
	}
}

// recReq is one request a recording harness captured.
type recReq struct {
	reqMethod string
	reqPath   string
	header    http.Header
	body      []byte
}

// checkStringMediaURL verifies one URL remains unchanged in the configured field.
func checkStringMediaURL(t *testing.T, stringMediaStyle catalog.InputMediaStyle, providerField string) {
	t.Helper()

	mediaURL := "https://media.example/reference.png"
	requestBody := map[string]any{}

	if err := addInputMedia(requestBody, stringMediaStyle, providerField, "images", []media.Input{{URL: mediaURL, MIME: "image/png"}}); err != nil {
		t.Fatalf("💣 string media assembly failed: %v", err)
	}

	if requestBody[providerField] != mediaURL {
		t.Errorf("✗ configured field = %#v, want the unchanged media URL", requestBody[providerField])
	}
}

// checkStringMediaDataURI verifies local bytes round-trip through the assembled data URI.
func checkStringMediaDataURI(t *testing.T, stringMediaStyle catalog.InputMediaStyle, providerField string) {
	t.Helper()

	inputBytes := []byte("local-image-bytes")
	requestBody := map[string]any{}

	if err := addInputMedia(requestBody, stringMediaStyle, providerField, "images", []media.Input{{Bytes: inputBytes, MIME: "image/png"}}); err != nil {
		t.Fatalf("💣 string media assembly failed: %v", err)
	}

	mediaDataURI, ok := requestBody[providerField].(string)
	if !ok {
		t.Fatalf("💣 configured field = %#v, want a data URI string", requestBody[providerField])
	}

	encodedBytes, hasDataPrefix := strings.CutPrefix(mediaDataURI, "data:"+"image/png"+";base64,")
	if !hasDataPrefix {
		t.Fatalf("💣 configured field is not the expected PNG data URI: %q", mediaDataURI)
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(encodedBytes)
	if err != nil {
		t.Fatalf("💣 decode assembled data URI: %v", err)
	}

	if !bytes.Equal(decodedBytes, inputBytes) {
		t.Errorf("✗ decoded data URI bytes = %q, want %q", decodedBytes, inputBytes)
	}
}

// checkStringMediaArray verifies several inputs remain an ordered array of strings.
func checkStringMediaArray(t *testing.T, stringMediaStyle catalog.InputMediaStyle, providerField string) {
	t.Helper()

	mediaInputs := []media.Input{
		{URL: "https://media.example/first.png", MIME: "image/png"},
		{Bytes: []byte("second-image"), MIME: "image/webp"},
	}
	requestBody := map[string]any{}

	if err := addInputMedia(requestBody, stringMediaStyle, providerField, "images", mediaInputs); err != nil {
		t.Fatalf("💣 string media assembly failed: %v", err)
	}

	mediaStrings := []string{mediaInputs[0].DataURI(), mediaInputs[1].DataURI()}
	if !reflect.DeepEqual(requestBody[providerField], mediaStrings) {
		t.Errorf("✗ configured field = %#v, want ordered strings %#v", requestBody[providerField], mediaStrings)
	}
}

// checkStringMediaFrameRefusal verifies a frame prefix is classified before body mutation.
func checkStringMediaFrameRefusal(t *testing.T, stringMediaStyle catalog.InputMediaStyle, providerField string) {
	t.Helper()

	requestBody := map[string]any{}
	mediaInputs := []media.Input{{URL: "https://media.example/open.png", MIME: "image/png", FrameAnchor: media.FrameFirst}}

	err := addInputMedia(requestBody, stringMediaStyle, providerField, "images", mediaInputs)
	if !errors.Is(err, errs.ErrInputMediaTime) {
		t.Errorf("✗ frame-prefixed string media error = %v, want the input-media time classification", err)
	}

	if len(requestBody) != 0 {
		t.Errorf("✗ rejected frame input changed request body: %#v", requestBody)
	}
}

// parseForm parses a recorded multipart request into its form; fatal on a non-multipart recording
// since no field assertion could run.
func parseForm(t *testing.T, req recReq) *multipart.Form {
	t.Helper()

	mt, parameterValues, err := mime.ParseMediaType(req.header.Get("Content-Type"))
	if err != nil || mt != "multipart/form-data" {
		t.Fatalf("💣 content type = %q (%v), want multipart/form-data", req.header.Get("Content-Type"), err)
	}

	form, err := multipart.NewReader(bytes.NewReader(req.body), parameterValues["boundary"]).ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("💣 parse multipart body: %v", err)
	}

	t.Cleanup(func() { _ = form.RemoveAll() })

	return form
}

// formKeys asserts the form's text-field key set equals want exactly, both directions.
func formKeys(t *testing.T, form *multipart.Form, want ...string) {
	t.Helper()

	wantSet := make(map[string]bool, len(want))
	for _, k := range want {
		wantSet[k] = true
	}

	for k := range form.Value {
		if !wantSet[k] {
			t.Errorf("✗ undeclared form field %q", k)
		}
	}

	for k := range wantSet {
		if len(form.Value[k]) == 0 {
			t.Errorf("✗ expected form field %q absent", k)
		}
	}
}
