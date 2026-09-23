package bfl

// Invariants tested:
//  1. BFL caller media ownership: Given closing and opening frame markers in that order,
//     AdjustParams must succeed and leave the caller's media order, URLs, markers, and nil
//     timestamp pointers unchanged.
//  2. BFL configuration: Loading the embedded BFL configuration must return ID bfl, display name
//     Black Forest Labs, environment variable BFL_API_KEY, and at least one model. NewProvider must
//     construct a nonnil generator without an error.
//  3. BFL catalog valid: Given the embedded BFL configuration and the declared flags,
//     catalog.LoadCatalog must return no error.
//  4. BFL parameter adjustment: For FLUX.2 Pro, AdjustParams must derive 64x64 from 10x10, preserve
//     valid explicit sizes, and report superseded sizing flags and unsupported flags as ignored. It
//     must report capping nine images to eight, reject bmp, and normalize WEBP to webp. Each call
//     must return exactly the expected parameter values and adjustment records without an error,
//     including when several adjustments occur together.
//  5. BFL polling state transitions: After Pending then Ready, Generate must return one artifact
//     after two polls. Moderated, Error, Failed, and Task not found responses must produce
//     ErrResponseGen naming the job and preserving any supplied details. An Enhancing response must
//     produce ErrResponseUnknown naming both the job and status.
//  6. BFL built-in model paths: For outpainting with a source URL and size 1024x768, Generate must
//     submit to /flux-tools/outpainting-v1 with that URL in input_image and the specified
//     dimensions. For FLUX.3 Video with an unidentifiable download format, Generate must return one
//     artifact with the configured video fallback extension.
//  7. BFL request fields: Generate must POST to the selected model's route and preserve the prompt.
//     FLUX.2 must send explicit or derived width and height, the stepped model must send explicit
//     dimensions without aspect_ratio, and Ultra and Kontext must send the supplied aspect without
//     dimensions. Unrequested dimensions, aspect_ratio, and output_format must be absent; a
//     requested webp format must be present.
//  8. BFL lifecycle: Generate must send x-key on submission and GET the returned polling URL with
//     its path, query, and credential intact. It must download the sample without x-key and return
//     the exact bytes in a temporary .jpg file. A Ready response without a sample must return
//     ErrResponseNoSample naming the job and make no download request.
//  9. BFL image references: Given local image inputs, AdjustParams and Generate must send raw
//     base64 in input_image and subsequent indexed fields in input order. They must cap FLUX.2 from
//     nine inputs to the first eight and Kontext from six to the first four, omit fields beyond
//     each cap, and return the corresponding Capped adjustment record.
//  10. BFL size bounds: After AdjustParams and Generate, FLUX.2 must send 2048x2048 unchanged,
//      constrain 5000x5000 to at most 4,194,304 pixels with edges of at least 64, and raise both
//      edges of 10x10 to at least 64. The stepped model must convert 1000x1000 to dimensions
//      divisible by 32 and within 256 to 1440.
//  11. Configured BFL image operations: For expansion, fine-tuned fill and Ultra, erasing, and both
//      try-on models, Generate must submit to the model's route and return one artifact. It must
//      encode source bytes in image, image_prompt, or person as configured, and preserve the
//      supplied margins, fine-tune controls, mask, mask dilation, strength, and garment URL in
//      their request fields.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestBFLPreparationOwnsMedia verifies invariant #1: BFL caller media ownership.
//
// What is being tested:
// Given closing and opening frame markers in that order, AdjustParams must succeed and leave the
// caller's media order, URLs, markers, and nil timestamp pointers unchanged.
//
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
func TestBFLPreparationOwnsMedia(t *testing.T) {
	providerConfig := shippedBFLCfg(t)

	model := catalog.Model{ID: "flux-3-video", Media: media.Video, Params: params.Definitions{{FlagID: params.FlagTypeInputMedia, MaxMultiple: 2}}}
	callerMedia := []media.Input{
		{URL: "https://example.test/closing.png", MIME: "image/png", FrameAnchor: media.FrameLast},
		{URL: "https://example.test/opening.png", MIME: "image/png", FrameAnchor: media.FrameFirst},
	}

	_, err := constructedTestProvider(t, &providerConfig).AdjustParams(&model, nil, callerMedia, nil)
	if err != nil {
		t.Errorf("✗ supported keyframes failed: %v", err)
	}

	if callerMedia[0].URL != "https://example.test/closing.png" || callerMedia[0].FrameAnchor != media.FrameLast || callerMedia[0].Time != nil || callerMedia[1].URL != "https://example.test/opening.png" || callerMedia[1].FrameAnchor != media.FrameFirst || callerMedia[1].Time != nil {
		t.Errorf("✗ keyframe preparation mutated caller media: %+v", callerMedia)
	}

	if !t.Failed() {
		t.Log("✓ BFL keyframe policy preserves caller media records")
	}
}

// TestBFLCfg verifies invariant #2: BFL configuration.
//
// What is being tested:
// Loading the embedded BFL configuration must return ID bfl, display name Black Forest Labs,
// environment variable BFL_API_KEY, and at least one model. NewProvider must construct a nonnil
// generator without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBFLCfg(t *testing.T) {
	provCfg := shippedBFLCfg(t)

	if provCfg.ID != "bfl" || provCfg.DisplayName != "Black Forest Labs" || provCfg.APIKeyEnvVar != "BFL_API_KEY" {
		t.Errorf("✗ identity = %+v, want bfl/Black Forest Labs/BFL_API_KEY", provCfg.Identity())
	}

	if constructedTestProvider(t, &provCfg) == nil {
		t.Errorf("✗ the constructor composed no generator over the decoded entry")
	}

	if len(provCfg.Models) == 0 {
		t.Errorf("✗ the embedded BFL config declares no models")
	}

	if !t.Failed() {
		t.Log("✓ the embedded BFL config loads with its identity, models, and generator")
	}
}

// TestBFLCatalogValid verifies invariant #3: BFL catalog valid.
//
// What is being tested:
// Given the embedded BFL configuration and the declared flags, catalog.LoadCatalog must return no
// error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBFLCatalogValid(t *testing.T) {
	flags := params.Flags()

	if _, err := catalog.LoadCatalog(flags, catalog.Source{ProviderID: ProviderID, ConfigBytes: ConfigJSON}); err != nil {
		t.Errorf("✗ the BFL config does not validate into a catalog: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ the BFL config validates into a catalog (no duplicate pair or alias defect)")
	}
}

// TestBFLAdjustParams verifies invariant #4: BFL parameter adjustment.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// What is being tested:
// For FLUX.2 Pro, AdjustParams must derive 64x64 from 10x10, preserve valid explicit sizes, and
// report superseded sizing flags and unsupported flags as ignored. It must report capping nine
// images to eight, reject bmp, and normalize WEBP to webp. Each call must return exactly the
// expected parameter values and adjustment records without an error, including when several
// adjustments occur together.
func TestBFLAdjustParams(t *testing.T) {
	provCfg := shippedBFLCfg(t)
	p := constructedTestProvider(t, &provCfg)
	cases := []adjustContractCase{
		{
			name:   "an out-of-bounds size derives into the declared bounds",
			inputs: params.FlagInputs{params.FlagTypeSize: "10x10"},
			gp:     params.Values{params.FlagTypeSize: "64x64"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeSize, Type: params.ChangeDerived, InputVal: "10x10", WireVal: "64x64", Comment: "flux-2-pro limits"},
			},
		},
		{
			name:   "a valid explicit size supersedes the sizing pair",
			inputs: params.FlagInputs{params.FlagTypeSize: "800x600", params.FlagTypeAspect: "3:2", params.FlagTypeResolution: "720p"},
			gp:     params.Values{params.FlagTypeSize: "800x600"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeAspect, Type: params.ChangeIgnored, Comment: fmt.Sprintf(params.ReasonSupersededBySize, "800x600")},
				{FlagID: params.FlagTypeResolution, Type: params.ChangeIgnored, Comment: fmt.Sprintf(params.ReasonSupersededBySize, "800x600")},
			},
		},
		{
			name:   "unconsumed supplied flags warn as ignored",
			inputs: params.FlagInputs{params.FlagTypeImageN: 3, params.FlagTypeDuration: 5},
			gp:     params.Values{},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeDuration, Type: params.ChangeIgnored, Comment: fmt.Sprintf(generation.ReasonNotConsumed, "flux-2-pro")},
				{FlagID: params.FlagTypeImageN, Type: params.ChangeIgnored, Comment: fmt.Sprintf(generation.ReasonNotConsumed, "flux-2-pro")},
			},
		},
		{
			name:   "over-cap input images carry the exact cap decision",
			inputs: params.FlagInputs{},
			images: 9,
			gp:     params.Values{},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped, InputVal: "9", WireVal: "8", Comment: "max 8"},
			},
		},
		{
			name:   "an output-format outside the declared set rejects naming it",
			inputs: params.FlagInputs{params.FlagTypeOutputFormat: "bmp"},
			gp:     params.Values{},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeOutputFormat, Type: params.ChangeRejected, InputVal: "bmp", Comment: "jpeg|png|webp"},
			},
		},
		{
			name:   "an output-format member passes as the declared spelling",
			inputs: params.FlagInputs{params.FlagTypeOutputFormat: "WEBP"},
			gp:     params.Values{params.FlagTypeOutputFormat: "webp"},
		},
		{
			name:   "the complete record set returns from the one call",
			inputs: params.FlagInputs{params.FlagTypeSize: "10x10", params.FlagTypeAspect: "3:2", params.FlagTypeImageN: 2},
			images: 9,
			gp:     params.Values{params.FlagTypeSize: "64x64"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeImageN, Type: params.ChangeIgnored, Comment: fmt.Sprintf(generation.ReasonNotConsumed, "flux-2-pro")},
				{FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped, InputVal: "9", WireVal: "8", Comment: "max 8"},
				{FlagID: params.FlagTypeSize, Type: params.ChangeDerived, InputVal: "10x10", WireVal: "64x64", Comment: "flux-2-pro limits"},
				{FlagID: params.FlagTypeAspect, Type: params.ChangeIgnored, Comment: fmt.Sprintf(params.ReasonSupersededBySize, "10x10")},
			},
		},
	}

	fluxPro := shippedBFLModel(t, provCfg, "flux-2-pro")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkAdjustContract(t, p, fluxPro, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the one AdjustParams call returns the adjusted values and the complete record set with a nil error")
	}
}

// TestBFLStatus verifies invariant #5: BFL polling state transitions.
//
// What is being tested:
// After Pending then Ready, Generate must return one artifact after two polls. Moderated, Error,
// Failed, and Task not found responses must produce ErrResponseGen naming the job and preserving
// any supplied details. An Enhancing response must produce ErrResponseUnknown naming both the job
// and status.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBFLStatus(t *testing.T) {
	t.Run("Pending keeps polling to Ready", statusPending)

	failWords := []struct {
		name    string
		body    string
		details string // text that must reach the error
	}{
		{
			"Request Moderated carries object details",
			`{"id":"job-1","status":"Request Moderated","result":null,"details":{"Moderation Reasons":["Violence"]}}`,
			"Violence",
		},
		{
			"Content Moderated carries object details",
			`{"id":"job-1","status":"Content Moderated","result":null,"details":{"Moderation Reasons":["Sexual"]}}`,
			"Sexual",
		},
		{
			"Error carries string details",
			`{"id":"job-1","status":"Error","details":"backend exploded"}`,
			"backend exploded",
		},
		{
			"Failed carries string details",
			`{"id":"job-1","status":"Failed","details":"out of credits"}`,
			"out of credits",
		},
		{
			"Task not found fails naming the id",
			`{"id":"job-1","status":"Task not found"}`,
			"",
		},
	}
	for _, c := range failWords {
		t.Run(c.name, func(t *testing.T) {
			err := statusErr(t, c.body)
			if !errors.Is(err, errs.ErrResponseGen) {
				t.Errorf("✗ error is not the generation failure: %v", err)
			}

			wantInErr(t, err, "job-1")

			if c.details != "" {
				wantInErr(t, err, c.details)
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	t.Run("an unknown word errors naming the id and value", statusUnknown)

	if !t.Failed() {
		t.Log("✓ every status word in the vocabulary classifies as expected and details reach the failure errors")
	}
}

// TestBFLShippedModelPaths verifies invariant #6: BFL built-in model paths.
//
// What is being tested:
// For outpainting with a source URL and size 1024x768, Generate must submit to
// /flux-tools/outpainting-v1 with that URL in input_image and the specified dimensions. For FLUX.3
// Video with an unidentifiable download format, Generate must return one artifact with the
// configured video fallback extension.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBFLShippedModelPaths(t *testing.T) {
	provCfg := shippedBFLCfg(t)

	t.Run("outpainting request field and path", func(t *testing.T) {
		h := newHarness(t)
		h.pollBodies = []string{h.ready()}
		model := shippedBFLModel(t, provCfg, "flux-tools/outpainting-v1")
		inputMedia := []media.Input{{URL: "https://media.example/source.png", MIME: "image/png"}}
		parameterValues := params.FlagInputs{params.FlagTypeSize: "1024x768"}

		if _, err := run(t, h, model, parameterValues, inputMedia); err != nil {
			t.Fatalf("💣 outpainting run: %v", err)
		}

		submitRequest := h.sub.at(t, 0)
		if submitRequest.uri != "/flux-tools/outpainting-v1" {
			t.Errorf("✗ submit path = %q, want /flux-tools/outpainting-v1", submitRequest.uri)
		}

		body := jmap(t, submitRequest.body)
		if body["input_image"] != inputMedia[0].URL {
			t.Errorf("✗ input_image = %#v, want unchanged URL %q", body["input_image"], inputMedia[0].URL)
		}

		if body["width"] != float64(1024) || body["height"] != float64(768) {
			t.Errorf("✗ outpainting dimensions = (%#v, %#v), want (1024, 768)", body["width"], body["height"])
		}
	})

	t.Run("video fallback extension", func(t *testing.T) {
		h := newHarness(t)
		h.pollBodies = []string{h.ready()}
		h.dlContentType = "application/octet-stream"
		h.dlData = []byte("not-signature-detectable")
		model := shippedBFLModel(t, provCfg, "flux-3-video")

		result, err := run(t, h, model, nil, nil)
		if err != nil {
			t.Fatalf("💣 video run: %v", err)
		}

		if len(result.Artifacts) != 1 || result.Artifacts[0].FileExt != provCfg.Config.AdapterAPI.VideoFallbackExt {
			t.Errorf("✗ video artifacts = %+v, want fallback extension %q", result.Artifacts, provCfg.Config.AdapterAPI.VideoFallbackExt)
		}
	})
}

// TestBFLBody verifies invariant #7: BFL request fields.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// What is being tested:
// Generate must POST to the selected model's route and preserve the prompt. FLUX.2 must send
// explicit or derived width and height, the stepped model must send explicit dimensions without
// aspect_ratio, and Ultra and Kontext must send the supplied aspect without dimensions. Unrequested
// dimensions, aspect_ratio, and output_format must be absent; a requested webp format must be
// present.
func TestBFLBody(t *testing.T) {
	cases := []bodyCase{
		{name: "flux2 bare run sends prompt only", model: fixtureFlux2(t), wantURI: "/flux-2-pro"},
		{
			name: "flux2 explicit size sends width+height", model: fixtureFlux2(t), wantURI: "/flux-2-pro",
			cfg:    params.FlagInputs{params.FlagTypeSize: "1024x1024"},
			wantWH: [2]float64{1024, 1024},
		},
		{
			name: "flux2 aspect derives width+height at long edge 1024", model: fixtureFlux2(t), wantURI: "/flux-2-pro",
			cfg:    params.FlagInputs{params.FlagTypeAspect: "16:9"},
			wantWH: [2]float64{1024, 576},
		},
		{
			name: "stepped model size sends width+height, never aspect_ratio", model: fixtureStepped(t), wantURI: "/flux-pro-1.1",
			cfg:    params.FlagInputs{params.FlagTypeSize: "1024x640", params.FlagTypeAspect: "16:9"},
			wantWH: [2]float64{1024, 640},
		},
		{
			name: "ultra forwards the aspect raw with no size fields", model: fixtureUltra(t), wantURI: "/flux-pro-1.1-ultra",
			cfg:    params.FlagInputs{params.FlagTypeAspect: "2:7", params.FlagTypeSize: "999x999"},
			aspect: "2:7",
		},
		{
			name: "kontext forwards the aspect raw", model: fixtureKontext(t), wantURI: "/flux-kontext-pro",
			cfg:    params.FlagInputs{params.FlagTypeAspect: "3:4"},
			aspect: "3:4",
		},
		{name: "kontext aspect is nullable — absent when not asked", model: fixtureKontext(t), wantURI: "/flux-kontext-pro"},
		{
			name: "output_format is sent only when set", model: fixtureFlux2(t), wantURI: "/flux-2-pro",
			cfg:    params.FlagInputs{params.FlagTypeOutputFormat: "webp"},
			format: "webp",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkBody(t, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the four assembly branches put exactly the requested-or-derived request fields, prompt always")
	}
}

// TestBFLLifecycle verifies invariant #8: BFL lifecycle.
//
// What is being tested:
// Generate must send x-key on submission and GET the returned polling URL with its path, query, and
// credential intact. It must download the sample without x-key and return the exact bytes in a
// temporary .jpg file. A Ready response without a sample must return ErrResponseNoSample naming the
// job and make no download request.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBFLLifecycle(t *testing.T) {
	t.Run("polling_url is consumed verbatim under x-key", lifeVerbatim)
	t.Run("the sample downloads without a credential and carries the artifact's bytes", lifeBareDownload)
	t.Run("a Ready body without a sample is a no-data error", lifeNoSample)

	if !t.Failed() {
		t.Log("✓ the returned-URL lifecycle and its credential split hold")
	}
}

// TestBFLRefs verifies invariant #9: BFL image references.
//
// What is being tested:
// Given local image inputs, AdjustParams and Generate must send raw base64 in input_image and
// subsequent indexed fields in input order. They must cap FLUX.2 from nine inputs to the first
// eight and Kontext from six to the first four, omit fields beyond each cap, and return the
// corresponding Capped adjustment record.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBFLRefs(t *testing.T) {
	t.Run("one input image is sent in input_image", func(t *testing.T) {
		body := inputMediaBody(t, fixtureFlux2(t), []media.Input{fixtureInputMedia(t, "a")})
		wantRef(t, body, "input_image", "REF-a-BYTES")

		if _, ok := body["input_image_2"]; ok {
			t.Errorf("✗ a lone input image produced an input_image_2 field")
		}

		if !t.Failed() {
			t.Log("✓ a single input image is sent in input_image as raw base64")
		}
	})
	t.Run("three input images are sent in the indexed fields in order", func(t *testing.T) {
		body := inputMediaBody(t, fixtureFlux2(t), []media.Input{fixtureInputMedia(t, "a"), fixtureInputMedia(t, "b"), fixtureInputMedia(t, "c")})
		wantRef(t, body, "input_image", "REF-a-BYTES")
		wantRef(t, body, "input_image_2", "REF-b-BYTES")
		wantRef(t, body, "input_image_3", "REF-c-BYTES")

		if _, ok := body["input_image_4"]; ok {
			t.Errorf("✗ three input images produced an input_image_4 field")
		}

		if !t.Failed() {
			t.Log("✓ three input images are sent in input_image, _2, _3 as raw base64 in order")
		}
	})
	t.Run("nine input images cap to the eight-image family limit", func(t *testing.T) {
		checkCap(t, fixtureFlux2(t), 9, 8)

		if !t.Failed() {
			t.Log("✓ nine input images cap to the eight-image family limit")
		}
	})
	t.Run("six input images cap to the four-image family limit", func(t *testing.T) {
		checkCap(t, fixtureKontext(t), 6, 4)

		if !t.Failed() {
			t.Log("✓ six input images cap to the four-image family limit")
		}
	})

	if !t.Failed() {
		t.Log("✓ input images encode indexed and cap at each family's MaxRefs")
	}
}

// TestBFLSizeBounds verifies invariant #10: BFL size bounds.
//
// What is being tested:
// After AdjustParams and Generate, FLUX.2 must send 2048x2048 unchanged, constrain 5000x5000 to at
// most 4,194,304 pixels with edges of at least 64, and raise both edges of 10x10 to at least 64.
// The stepped model must convert 1000x1000 to dimensions divisible by 32 and within 256 to 1440.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBFLSizeBounds(t *testing.T) {
	t.Run("flux2 4 MP exactly passes unchanged", boundExact4MP)
	t.Run("flux2 over 4 MP is constrained within bounds", boundOverClamps)
	t.Run("flux2 tiny request rises to the 64px floor", boundTinyFloors)
	t.Run("stepped edges settle on the 32px step within range", boundStepped)

	if !t.Failed() {
		t.Log("✓ the declared size envelopes hold in the request across the flux2 and stepped families")
	}
}

// TestConfiguredBFLImageOperations verifies invariant #11: Configured BFL image operations.
//
// What is being tested:
// For expansion, fine-tuned fill and Ultra, erasing, and both try-on models, Generate must submit
// to the model's route and return one artifact. It must encode source bytes in image, image_prompt,
// or person as configured, and preserve the supplied margins, fine-tune controls, mask, mask
// dilation, strength, and garment URL in their request fields.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestConfiguredBFLImageOperations(t *testing.T) {
	providerConfig := shippedBFLCfg(t)

	operationCases := []struct {
		modelID       string
		imageField    string
		flagInputs    params.FlagInputs
		requestFields map[string]any
	}{
		{"flux-pro-1.0-expand", "image", params.FlagInputs{"expand-top": 64, "expand-bottom": 32, "expand-left": 16, "expand-right": 8}, map[string]any{"top": float64(64), "bottom": float64(32), "left": float64(16), "right": float64(8)}},
		{"flux-pro-1.0-fill-finetuned", "image", params.FlagInputs{"finetune-id": "organization/custom-lora", "finetune-strength": 0.8, "mask": "bWFzaw=="}, map[string]any{"finetune_id": "organization/custom-lora", "finetune_strength": 0.8, "mask": "bWFzaw=="}},
		{"flux-pro-1.1-ultra-finetuned", "image_prompt", params.FlagInputs{"finetune-id": "organization/custom-lora", "finetune-strength": 0.8, "strength": 0.4}, map[string]any{"finetune_id": "organization/custom-lora", "finetune_strength": 0.8, "image_prompt_strength": 0.4}},
		{"flux-tools/erase-v1", "image", params.FlagInputs{"mask": "https://media.example/mask.png", "mask-dilation": 0}, map[string]any{"mask": "https://media.example/mask.png", "dilate_pixels": float64(0)}},
		{"flux-tools/vto-v1", "person", params.FlagInputs{"garment-url": "https://media.example/garment.png"}, map[string]any{"garment": "https://media.example/garment.png"}},
		{"flux-tools/vto-v2", "person", params.FlagInputs{"garment-url": "https://media.example/garment.png"}, map[string]any{"garment": "https://media.example/garment.png"}},
	}
	for _, operationCase := range operationCases {
		t.Run(operationCase.modelID, func(t *testing.T) {
			model := shippedBFLModel(t, providerConfig, operationCase.modelID)
			harness := newHarness(t)
			harness.pollBodies = []string{harness.ready()}
			mediaInput := media.Input{Bytes: []byte("source image"), MIME: "image/png"}

			result, err := run(t, harness, model, operationCase.flagInputs, []media.Input{mediaInput})
			if err != nil || len(result.Artifacts) != 1 {
				t.Fatalf("💣 image operation returned %d artifacts, error %v", len(result.Artifacts), err)
			}

			submission := harness.sub.at(t, 0)

			requestFields := jmap(t, submission.body)
			if submission.uri != "/"+operationCase.modelID || requestFields[operationCase.imageField] != base64.StdEncoding.EncodeToString(mediaInput.Bytes) {
				t.Errorf("✗ incorrect operation or source image: %s, %#v", submission.uri, requestFields)
			}

			for fieldName, fieldValue := range operationCase.requestFields {
				if requestFields[fieldName] != fieldValue {
					t.Errorf("✗ %s = %#v, required %#v", fieldName, requestFields[fieldName], fieldValue)
				}
			}

			if !t.Failed() {
				t.Log("✓ the configured image operation submits its inputs and returns an artifact")
			}
		})
	}
}

// adjustContractCase contains inputs, an image count, and the exact parameter values and adjustment
// records expected from AdjustParams. Record order is not checked.
type adjustContractCase struct {
	name    string
	inputs  params.FlagInputs
	images  int
	gp      params.Values
	records []params.Adjustment
}

// testKey is the credential every harness run carries via BFL_API_KEY.
const testKey = "test-bfl-key-123"

// recordedReq is one recorded provider request.
type recordedReq struct {
	reqMethod string
	uri       string // full request URI, query included — verbatim-URL assertions need it
	apiKey    string // the x-key header value ("" = absent)
	body      []byte
}

// recServer is a recording test server.
type recServer struct {
	test         testing.TB
	mu           sync.Mutex
	recordedReqs []recordedReq
	srv          *httptest.Server
}

// harness serves submission, polling, and downloads on separate hosts so tests can check request
// URLs and credentials independently. It serves poll bodies in order and repeats the last one.
type harness struct {
	test          testing.TB
	sub, poll, dl *recServer
	pollBodies    []string
	pollN         int
	dlData        []byte
	dlContentType string
	mu            sync.Mutex
}

// bodyCase is one body-assembly case.
type bodyCase struct {
	name    string
	model   catalog.Model
	cfg     params.FlagInputs
	wantURI string     // the endpoint path the submit must POST
	wantWH  [2]float64 // zero = width/height must be absent
	aspect  string     // "" = aspect_ratio must be absent
	format  string     // "" = output_format must be absent
}

// shippedBFLCfg decodes the embedded config through the strict shared path.
func shippedBFLCfg(t *testing.T) catalog.Provider {
	t.Helper()

	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags, catalog.Source{ProviderID: ProviderID, ConfigBytes: ConfigJSON})
	if err != nil {
		t.Fatalf("💣 the embedded config failed to decode: %v", err)
	}

	provCfg, loaded := loadedCatalog.Provider(ProviderID)
	if !loaded {
		t.Fatalf("💣 embedded BFL provider failed to load: %v", loadedCatalog.ConfigError(ProviderID))
	}

	return provCfg
}

// shippedBFLModel returns the decoded config's model by id.
func shippedBFLModel(t *testing.T, provCfg catalog.Provider, id string) catalog.Model {
	t.Helper()

	for _, model := range provCfg.Models {
		if model.ID == id {
			return model
		}
	}

	t.Fatalf("💣 no model %q in the decoded config", id)

	return catalog.Model{}
}

// recordCount counts the records equal to want exactly.
func recordCount(test testing.TB, records []params.Adjustment, want params.Adjustment) int {
	test.Helper()

	n := 0

	for _, record := range records {
		if record == want {
			n++
		}
	}

	return n
}

// checkAdjustContract drives one AdjustParams contract case and asserts the adjusted values and the
// record multiset exactly.
func checkAdjustContract(t *testing.T, p generation.Generator, model catalog.Model, c adjustContractCase) {
	t.Helper()

	images := make([]media.Input, 0, c.images)
	for i := range c.images {
		images = append(images, fixtureInputMedia(t, string(rune('a'+i))))
	}

	preparedGeneration, err := p.AdjustParams(&model, c.inputs, images, nil)
	gp, records := preparedGeneration.Params, preparedGeneration.Changes

	if err != nil {
		t.Errorf("✗ AdjustParams error = %v, want nil (BFL has no fallible adjustment)", err)
	}

	if len(gp) != len(c.gp) {
		t.Errorf("✗ adjusted values = %+v, want %+v", gp, c.gp)
	}

	for param, wantVal := range c.gp {
		if gp[param] != wantVal {
			t.Errorf("✗ adjusted %s = %v, want %v", param, gp[param], wantVal)
		}
	}

	if len(records) != len(c.records) {
		t.Errorf("✗ %d record(s) %+v, want %d: %+v", len(records), records, len(c.records), c.records)
	}

	for _, want := range c.records {
		if got, wantN := recordCount(t, records, want), recordCount(t, c.records, want); got != wantN {
			t.Errorf("✗ record %+v appears %d time(s), want %d", want, got, wantN)
		}
	}
}

// constructedTestProvider requires a valid configured generator for subsequent behavior checks.
func constructedTestProvider(test testing.TB, description *catalog.Provider) generation.Generator {
	test.Helper()

	generator, constructionErr := NewProvider(description)
	if constructionErr != nil || generator == nil {
		test.Fatalf("💣 configured generator construction failed: %v", constructionErr)
	}

	return generator
}

// statusErr runs a generation whose first poll body is the given fixture and returns the expected
// error.
func statusErr(t *testing.T, body string) error {
	t.Helper()
	h := newHarness(t)
	h.pollBodies = []string{body}

	_, err := run(t, h, fixtureFlux2(t), params.FlagInputs{}, nil)
	if err == nil {
		// Fatal: the callers classify the returned error, and a nil would make every
		// downstream assertion meaningless.
		t.Fatalf("💣 generation succeeded on a failure poll body")
	}

	return err
}

// statusPending scripts Pending then Ready: the run succeeds and the poll server was probed twice —
// Pending kept the loop alive.
func statusPending(t *testing.T) {
	t.Helper()
	h := newHarness(t)
	h.pollBodies = []string{`{"id":"job-1","status":"Pending","result":null}`, h.ready()}

	res, err := run(t, h, fixtureFlux2(t), params.FlagInputs{}, nil)
	if err != nil {
		t.Errorf("✗ run failed: %v", err)

		return
	}

	if h.poll.count() != 2 {
		t.Errorf("✗ poll count = %d, want 2 (Pending must keep polling)", h.poll.count())
	}

	if len(res.Artifacts) != 1 {
		t.Errorf("✗ artifacts = %d, want 1", len(res.Artifacts))
	}

	if !t.Failed() {
		t.Log("✓ Pending keeps polling and Ready completes the run")
	}
}

// statusUnknown verifies that the Enhancing status produces ErrResponseUnknown naming the job and
// status.
func statusUnknown(t *testing.T) {
	t.Helper()

	err := statusErr(t, `{"id":"job-1","status":"Enhancing"}`)
	if !errors.Is(err, errs.ErrResponseUnknown) {
		t.Errorf("✗ error is not the unknown-status error: %v", err)
	}

	wantInErr(t, err, "job-1")
	wantInErr(t, err, "Enhancing")

	if !t.Failed() {
		t.Log("✓ an unrecognized status word errors under the unknown-status sentinel naming the id and value")
	}
}

// harnessAPI returns the BFL adapter settings used by the provider test harness. It mirrors the
// built-in config over the harness base URL, including the declared lifecycle words and fallback
// extension.
func harnessAPI(test testing.TB, base string) catalog.AdapterAPI {
	test.Helper()

	return catalog.AdapterAPI{
		APIBase:           base,
		PendingStatusText: []string{"Pending", "Reasoning", "Generating"},
		ReadyStatusText:   "Ready",
		FailedStatusText:  []string{"Request Moderated", "Content Moderated", "Error", "Failed", "Task not found"},
		ImageFallbackExt:  ".jpg",
		VideoFallbackExt:  ".mp4",
	}
}

// bflProvider returns the provider identity under which the BFL test harness requests run.
func bflProvider(test testing.TB) catalog.Provider {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	return catalog.Provider{ID: "bfl", DisplayName: "Black Forest Labs", APIKeyEnvVar: "BFL_API_KEY"}
}

// fixtureFlux2 returns the FLUX.2 fixture with custom size bounds and input images.
func fixtureFlux2(test testing.TB) catalog.Model {
	test.Helper()

	return catalog.Model{
		ID: "flux-2-pro", Media: media.Image,
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution},
			{FlagID: params.FlagTypeSize, CustomSize: &params.SizeBounds{MinEdge: 64, MaxPx: 4194304, LongEdge: 1024}},
			{FlagID: params.FlagTypeOutputFormat, ParamID: "output_format", AllowedValues: []string{"jpeg", "png", "webp"}},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 8},
		},
	}
}

// fixtureStepped returns the stepped FLUX 1.1 fixture with custom size bounds.
func fixtureStepped(test testing.TB) catalog.Model {
	test.Helper()

	return catalog.Model{
		ID: "flux-pro-1.1", Media: media.Image,
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution},
			{FlagID: params.FlagTypeSize, CustomSize: &params.SizeBounds{MinEdge: 256, MaxEdge: 1440, EdgeIncrem: 32, LongEdge: 1024}},
			{FlagID: params.FlagTypeOutputFormat, ParamID: "output_format", AllowedValues: []string{"jpeg", "png", "webp"}},
		},
	}
}

// fixtureUltra returns the FLUX Ultra fixture with a raw aspect and no request size fields.
func fixtureUltra(test testing.TB) catalog.Model {
	test.Helper()

	return catalog.Model{
		ID: "flux-pro-1.1-ultra", Media: media.Image,
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeOutputFormat, ParamID: "output_format", AllowedValues: []string{"jpeg", "png", "webp"}},
		},
	}
}

// fixtureKontext returns the FLUX Kontext fixture with a raw aspect and input images.
func fixtureKontext(test testing.TB) catalog.Model {
	test.Helper()

	return catalog.Model{
		ID: "flux-kontext-pro", Media: media.Image,
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeOutputFormat, ParamID: "output_format", AllowedValues: []string{"jpeg", "png", "webp"}},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 4},
		},
	}
}

// newRecServer starts a recording server around handle and closes it at cleanup.
func newRecServer(t *testing.T, handle http.HandlerFunc) *recServer {
	t.Helper()

	r := &recServer{test: t}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body) // a short fixture body; a read failure only empties the record

		r.mu.Lock()
		r.recordedReqs = append(r.recordedReqs, recordedReq{
			reqMethod: req.Method,
			uri:       req.URL.RequestURI(),
			apiKey:    req.Header.Get("X-Key"),
			body:      body,
		})
		r.mu.Unlock()
		handle(w, req)
	}))
	t.Cleanup(r.srv.Close)

	return r
}

// count returns how many requests the server recorded.
func (r *recServer) count() int {
	r.test.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.recordedReqs)
}

// at returns the i-th recorded request, failing the test when it never happened.
func (r *recServer) at(t *testing.T, i int) recordedReq {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()

	if i >= len(r.recordedReqs) {
		t.Fatalf("💣 only %d requests recorded, need index %d", len(r.recordedReqs), i)
	}

	return r.recordedReqs[i]
}

// newHarness starts separate submission, polling, and download servers. Its submission response
// names a job and supplies the poll server URL with the get_result path and job query.
func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{test: t, dlData: []byte("BFLIMGDATA"), dlContentType: "image/jpeg"}
	h.poll = newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		h.mu.Lock()

		i := h.pollN
		if i >= len(h.pollBodies) {
			i = len(h.pollBodies) - 1
		}

		h.pollN++
		h.mu.Unlock()

		if i < 0 {
			http.NotFound(w, nil)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, h.pollBodies[i])
	})
	h.dl = newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", h.dlContentType)
		_, _ = w.Write(h.dlData)
	})
	h.sub = newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"job-1","polling_url":"`+h.pollURL()+`"}`)
	})

	return h
}

// pollURL returns the poll server URL with the path and query used in the submission response.
func (h *harness) pollURL() string {
	h.test.Helper()

	return h.poll.srv.URL + "/v1/get_result?id=job-1"
}

// sampleURL is the signed result URL on the download server's own host.
func (h *harness) sampleURL() string {
	h.test.Helper()

	return h.dl.srv.URL + "/signed/sample"
}

// ready creates a Ready poll body whose result.sample is the harness's download URL.
func (h *harness) ready() string {
	h.test.Helper()

	return `{"id":"job-1","status":"Ready","result":{"sample":"` + h.sampleURL() + `"}}`
}

// harnessProvider returns a provider using the local server with one-second polling and a
// two-second timeout.
func harnessProvider(test testing.TB, base string) *Provider {
	test.Helper()
	adapterAPI := harnessAPI(test, base)

	adapterAPI.PollInterval, adapterAPI.PollTimeout = 1, 2

	return &Provider{adapterAPI: &adapterAPI}
}

// run adjusts inputs and generates media against the local server. It returns the generation result
// and registers cleanup for downloaded artifacts.
func run(t *testing.T, h *harness, model catalog.Model, cfg params.FlagInputs, inputs []media.Input) (generation.Result, error) {
	t.Helper()
	t.Setenv("BFL_API_KEY", testKey)

	p := harnessProvider(t, h.sub.srv.URL)

	preparedGeneration, err := p.AdjustParams(&model, cfg, inputs, nil)
	if err != nil {
		t.Fatalf("💣 AdjustParams failed: %v", err)
	}

	res, err := p.Generate(context.Background(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: bflProvider(t), Model: model}, Prompt: params.GetSetIf(true, testPrompt).ValOr(""), Preparation: preparedGeneration, APIKey: os.Getenv((catalog.ProvModelPair{Provider: bflProvider(t), Model: model}).Provider.APIKeyEnvVar)})
	for _, a := range res.Artifacts {
		if a.TmpPath != "" {
			path := a.TmpPath

			t.Cleanup(func() { _ = os.Remove(path) })
		}
	}

	return res, err
}

// jmap decodes a recorded JSON object and fails the test if decoding fails.
func jmap(t *testing.T, body []byte) map[string]any {
	t.Helper()

	m := map[string]any{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("💣 recorded body is not JSON: %v (%s)", err, body)
	}

	return m
}

// checkBody runs one harness generation and asserts the recorded submit body and endpoint.
func checkBody(t *testing.T, c bodyCase) {
	t.Helper()
	h := newHarness(t)

	h.pollBodies = []string{h.ready()}
	if _, err := run(t, h, c.model, c.cfg, nil); err != nil {
		t.Errorf("✗ harness run failed: %v", err)

		return
	}

	sub := h.sub.at(t, 0)
	if sub.reqMethod != http.MethodPost || sub.uri != c.wantURI {
		t.Errorf("✗ submit = %s %s, want POST %s", sub.reqMethod, sub.uri, c.wantURI)
	}

	body := jmap(t, sub.body)
	if body["prompt"] != testPrompt {
		t.Errorf("✗ prompt = %v, want %q", body["prompt"], testPrompt)
	}

	checkWH(t, body, c.wantWH)
	checkOpt(t, body, "aspect_ratio", c.aspect)
	checkOpt(t, body, "output_format", c.format)

	if !t.Failed() {
		t.Logf("✓ %s", c.name)
	}
}

// checkWH asserts width/height carry want, or are absent when want is zero.
func checkWH(t *testing.T, body map[string]any, want [2]float64) {
	t.Helper()

	w, wOK := body["width"]

	hv, hOK := body["height"]
	if want == [2]float64{} {
		if wOK || hOK {
			t.Errorf("✗ width/height present on an unsized run: %v/%v", w, hv)
		}

		return
	}

	if w != want[0] || hv != want[1] {
		t.Errorf("✗ width/height = %v/%v, want %v/%v", w, hv, want[0], want[1])
	}
}

// checkOpt asserts an optional string request field carries want, or is absent when want is empty.
func checkOpt(t *testing.T, body map[string]any, field, want string) {
	t.Helper()

	got, ok := body[field]
	if want == "" {
		if ok {
			t.Errorf("✗ %s present unrequested: %v", field, got)
		}

		return
	}

	if got != want {
		t.Errorf("✗ %s = %v, want %q", field, got, want)
	}
}

// wantInErr asserts the error text carries needle.
func wantInErr(t *testing.T, err error, needle string) {
	t.Helper()

	if !strings.Contains(err.Error(), needle) {
		t.Errorf("✗ error %q does not carry %q", err, needle)
	}
}

// lifeVerbatim asserts the submit carries x-key and every poll arrived at the poll server's own
// host at the exact returned path and query.
func lifeVerbatim(t *testing.T) {
	t.Helper()
	h := newHarness(t)

	h.pollBodies = []string{`{"id":"job-1","status":"Pending","result":null}`, h.ready()}
	if _, err := run(t, h, fixtureFlux2(t), params.FlagInputs{}, nil); err != nil {
		t.Errorf("✗ run failed: %v", err)

		return
	}

	if k := h.sub.at(t, 0).apiKey; k != testKey {
		t.Errorf("✗ submit x-key = %q, want %q", k, testKey)
	}

	if h.poll.count() < 1 {
		t.Errorf("✗ the returned polling_url host was never reached — the URL was not consumed verbatim")

		return
	}

	for i := range h.poll.count() {
		p := h.poll.at(t, i)
		if p.reqMethod != http.MethodGet || p.uri != "/v1/get_result?id=job-1" {
			t.Errorf("✗ poll %d = %s %s, want GET /v1/get_result?id=job-1 verbatim", i, p.reqMethod, p.uri)
		}

		if p.apiKey != testKey {
			t.Errorf("✗ poll %d x-key = %q, want %q", i, p.apiKey, testKey)
		}
	}

	if !t.Failed() {
		t.Log("✓ polls reach the returned URL verbatim on its own host, credentialed")
	}
}

// lifeBareDownload asserts the sample download requests the signed URL with NO x-key and the
// artifact is file-backed with the served bytes under the truthful .jpg extension.
func lifeBareDownload(t *testing.T) {
	t.Helper()
	h := newHarness(t)
	h.pollBodies = []string{h.ready()}

	res, err := run(t, h, fixtureFlux2(t), params.FlagInputs{}, nil)
	if err != nil {
		t.Errorf("✗ run failed: %v", err)

		return
	}

	if h.dl.count() != 1 {
		t.Errorf("✗ download count = %d, want 1", h.dl.count())

		return
	}

	d := h.dl.at(t, 0)
	if d.uri != "/signed/sample" {
		t.Errorf("✗ download uri = %q, want /signed/sample", d.uri)
	}

	if d.apiKey != "" {
		t.Errorf("✗ the sample download carried x-key %q, want it sent with no credential", d.apiKey)
	}

	if len(res.Artifacts) != 1 {
		t.Errorf("✗ artifacts = %d, want 1", len(res.Artifacts))

		return
	}

	generatedMedia := res.Artifacts[0]
	if generatedMedia.TmpPath == "" {
		t.Errorf("✗ artifact is not file-backed")

		return
	}

	got, rerr := os.ReadFile(generatedMedia.TmpPath)
	if rerr != nil {
		t.Errorf("✗ artifact file unreadable: %v", rerr)
	} else if string(got) != string(h.dlData) {
		t.Errorf("✗ artifact bytes differ from the served sample")
	}

	if generatedMedia.FileExt != ".jpg" {
		t.Errorf("✗ artifact ext = %q, want .jpg (declared image/jpeg)", generatedMedia.FileExt)
	}

	if !t.Failed() {
		t.Log("✓ the signed sample streams with no credential to a file-backed .jpg artifact")
	}
}

// lifeNoSample asserts a Ready body with no result.sample is a classified no-sample error naming
// the id.
func lifeNoSample(t *testing.T) {
	t.Helper()
	h := newHarness(t)
	h.pollBodies = []string{`{"id":"job-1","status":"Ready","result":{}}`}

	_, err := run(t, h, fixtureFlux2(t), params.FlagInputs{}, nil)
	if err == nil {
		t.Errorf("✗ a sampleless Ready body produced no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseNoSample) {
		t.Errorf("✗ error is not the no-sample error: %v", err)
	}

	wantInErr(t, err, "job-1")

	if h.dl.count() != 0 {
		t.Errorf("✗ a download ran despite the missing sample")
	}

	if !t.Failed() {
		t.Log("✓ a Ready body without a sample errors as no-sample naming the id")
	}
}

// fixtureInputMedia creates a fixture input image whose bytes are distinct per marker.
func fixtureInputMedia(test testing.TB, marker string) media.Input {
	test.Helper()

	return media.Input{Bytes: []byte("REF-" + marker + "-BYTES"), MIME: "image/png", Filepath: "/tmp/" + marker + ".png"}
}

// inputMediaBody runs one input-media generation and returns the recorded submit body.
func inputMediaBody(t *testing.T, m catalog.Model, inputs []media.Input) map[string]any {
	t.Helper()
	h := newHarness(t)

	h.pollBodies = []string{h.ready()}
	if _, err := run(t, h, m, params.FlagInputs{}, inputs); err != nil {
		t.Fatalf("💣 input-media run failed: %v", err)
	}

	return jmap(t, h.sub.at(t, 0).body)
}

// wantRef asserts a request body field carries the raw base64 of want (no data: prefix) and
// round-trips back to want.
func wantRef(t *testing.T, body map[string]any, field, want string) {
	t.Helper()

	got, ok := body[field].(string)
	if !ok {
		t.Errorf("✗ %s missing or not a string", field)

		return
	}

	if strings.HasPrefix(got, "data:") {
		t.Errorf("✗ %s carries a data: URI prefix, want raw base64", field)
	}

	dec, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Errorf("✗ %s is not valid base64: %v", field, err)

		return
	}

	if string(dec) != want {
		t.Errorf("✗ %s decodes to %q, want %q", field, dec, want)
	}
}

// checkCap verifies that AdjustParams caps the image count with the specified Capped record and
// that Generate sends the first maxInputCount images in order, with no field beyond the cap.
func checkCap(t *testing.T, m catalog.Model, nInputs, maxInputCount int) {
	t.Helper()
	t.Setenv("BFL_API_KEY", testKey)
	h := newHarness(t)
	h.pollBodies = []string{h.ready()}

	inputs := make([]media.Input, 0, nInputs)
	for i := range nInputs {
		inputs = append(inputs, fixtureInputMedia(t, string(rune('a'+i))))
	}

	p := harnessProvider(t, h.sub.srv.URL)

	preparedGeneration, err := p.AdjustParams(&m, params.FlagInputs{}, inputs, nil)
	pre := preparedGeneration.Changes

	if err != nil {
		t.Fatalf("💣 AdjustParams failed: %v", err)
	}

	res, err := p.Generate(context.Background(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: bflProvider(t), Model: m}, Prompt: params.GetSetIf(true, testPrompt).ValOr(""), Preparation: preparedGeneration, APIKey: os.Getenv((catalog.ProvModelPair{Provider: bflProvider(t), Model: m}).Provider.APIKeyEnvVar)})
	for _, a := range res.Artifacts {
		if a.TmpPath != "" {
			path := a.TmpPath

			t.Cleanup(func() { _ = os.Remove(path) })
		}
	}

	if err != nil {
		t.Fatalf("💣 capped run failed: %v", err)
	}

	body := jmap(t, h.sub.at(t, 0).body)
	// The kept set is the FIRST maxInputCount image inputs, byte-identical — not merely the
	// right count of request fields.
	for i := range maxInputCount {
		got, _ := body[indexedInputMediaParam(i)].(string)
		if want := base64.StdEncoding.EncodeToString(inputs[i].Bytes); got != want {
			t.Errorf("✗ %s = %q, want the first-set input image %d's bytes %q", indexedInputMediaParam(i), got, i+1, want)
		}
	}

	if _, ok := body[indexedInputMediaParam(maxInputCount)]; ok {
		t.Errorf("✗ %s was sent past the max-%d cap", indexedInputMediaParam(maxInputCount), maxInputCount)
	}

	want := params.Adjustment{
		FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped,
		InputVal: strconv.Itoa(nInputs), WireVal: strconv.Itoa(maxInputCount), Comment: "max " + strconv.Itoa(maxInputCount),
	}
	capped := false

	for _, c := range pre {
		if c == want {
			capped = true
		}
	}

	if !capped {
		t.Errorf("✗ no Capped %d→%d record reported for the over-limit input-media set: %+v", nInputs, maxInputCount, pre)
	}

	if !t.Failed() {
		t.Logf("✓ %d image inputs cap to %d indexed request fields with the Capped record", nInputs, maxInputCount)
	}
}

// wh reads width/height from a recorded submit body as ints.
func wh(t *testing.T, body map[string]any) (int, int, bool) {
	t.Helper()

	w, wOK := body["width"].(float64)

	h, hOK := body["height"].(float64)
	if !wOK || !hOK {
		return 0, 0, false
	}

	return int(w), int(h), true
}

// bodyFor runs one sized generation and returns the recorded submit body.
func bodyFor(t *testing.T, m catalog.Model, size string) map[string]any {
	t.Helper()
	h := newHarness(t)

	h.pollBodies = []string{h.ready()}
	if _, err := run(t, h, m, params.FlagInputs{params.FlagTypeSize: size}, nil); err != nil {
		t.Fatalf("💣 harness run failed: %v", err)
	}

	return jmap(t, h.sub.at(t, 0).body)
}

// boundExact4MP verifies that an exact 4 MP size is sent unchanged.
func boundExact4MP(t *testing.T) {
	t.Helper()

	w, h, ok := wh(t, bodyFor(t, fixtureFlux2(t), "2048x2048"))
	if !ok || w != 2048 || h != 2048 {
		t.Errorf("✗ 2048x2048 (exactly 4 MP) = %dx%d, want 2048x2048 unchanged", w, h)
	}

	if !t.Failed() {
		t.Log("✓ 2048x2048 is sent unchanged")
	}
}

// boundOverClamps verifies that an over-4 MP request is constrained to the declared 4 MP and
// 64-pixel minimum-edge bounds.
func boundOverClamps(t *testing.T) {
	t.Helper()

	w, h, ok := wh(t, bodyFor(t, fixtureFlux2(t), "5000x5000"))
	if !ok {
		t.Fatalf("💣 no width/height on the constrained run")
	}

	if w*h > 4194304 || w < 64 || h < 64 {
		t.Errorf("✗ 5000x5000 constrained to %dx%d (%d px) is outside the 4 MP / 64px envelope", w, h, w*h)
	}

	if !t.Failed() {
		t.Logf("✓ 5000x5000 constrained to %dx%d, within the 4 MP / 64px envelope", w, h)
	}
}

// boundTinyFloors verifies that a sub-floor request rises to the 64-pixel minimum edge.
func boundTinyFloors(t *testing.T) {
	t.Helper()

	w, h, ok := wh(t, bodyFor(t, fixtureFlux2(t), "10x10"))
	if !ok || w < 64 || h < 64 {
		t.Errorf("✗ 10x10 = %dx%d, want both edges ≥ 64", w, h)
	}

	if !t.Failed() {
		t.Logf("✓ 10x10 rose to %dx%d", w, h)
	}
}

// boundStepped verifies that adjusted edges are multiples of 32 within the declared 256-to-1440
// range.
func boundStepped(t *testing.T) {
	t.Helper()

	w, h, ok := wh(t, bodyFor(t, fixtureStepped(t), "1000x1000"))
	if !ok {
		t.Fatalf("💣 no width/height on the stepped run")
	}

	if w%32 != 0 || h%32 != 0 || w < 256 || w > 1440 || h < 256 || h > 1440 {
		t.Errorf("✗ stepped edges %dx%d are not 32px-multiples within [256,1440]", w, h)
	}

	if !t.Failed() {
		t.Logf("✓ 1000x1000 stepped to %dx%d", w, h)
	}
}
