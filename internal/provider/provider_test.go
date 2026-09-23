package provider

// Invariants tested:
//  1. Size-declaring configuration: Given image models with fixed or custom sizes, catalog loading
//     must preserve the fixture provider identity and size support on every model, and NewProvider
//     must return a generator.
//  2. Size-free configuration: Given image and video models without size parameters, catalog
//     loading must preserve the fixture provider identity and leave size unsupported on every
//     model, and NewProvider must return a generator.
//  3. Aggregator configuration: Given an aggregator configuration with image and video APIs,
//     catalog loading must preserve the fixture identity and Aggregator flag, and NewProvider must
//     return a generator.
//  4. URL-response image configuration: Given the URL-response image fixture, catalog loading must
//     preserve the provider identity, model IDs in document order, and RespImageURL flag.
//  5. Image generate requests: Given text inputs, the descriptor generator's Generate must use the
//     configured generation route with exactly the expected model, prompt, count, seed, size, and
//     supported negative-prompt fields.
//  6. Image parameter adjustment: Given excessive strength, the descriptor generator's AdjustParams
//     must cap it at the declared maximum and return a Capped record.
//  7. Image response handling: Given an SVG result URL, the descriptor generator's Generate must
//     return one file-backed .svg with exact bytes and send the adjusted vector aspect under its
//     provider field.
//  8. Descriptor generation: Given image and video models, the descriptor generator's Generate must
//     return the decoded image bytes or a file containing the exact video bytes.
//  9. Descriptor adjust params: Given the fixture constraints, the descriptor generator's
//     AdjustParams must derive or snap sizes, normalize or reject quality, bound counts and
//     durations, and record undeclared flags as Ignored.
//  10. Descriptor adjust resize decision: Given resizing video settings, the descriptor generator's
//      AdjustParams must adopt landscape or portrait dimensions when size is absent, preserve
//      explicit size, and return the exact Derived, Conformed, and applicable Capped records.
//  11. Configured image generation: Given each listed MAI or Recraft model, descriptor generation
//      must use the configured endpoint and exact model ID and return the expected image bytes.
//  12. xAI latest model: Given either the bare or xai-qualified latest model selector, descriptor
//      generation must send grok-imagine-image-quality-latest unchanged to the expected generation
//      or editing route and return the exact provider image bytes.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
	providerconfig "github.com/shdeen/bildomat/internal/provider/config"
)

// TestSizeDeclaringCfg verifies invariant #1: Size-declaring configuration.
//
// What is being tested:
// Given image models with fixed or custom sizes, catalog loading must preserve the fixture provider
// identity and size support on every model, and NewProvider must return a generator.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSizeDeclaringCfg(t *testing.T) {
	provCfg := decodeFixtureCfg(t, `
	"models": [
		{"id": "fixed-size-image", "name": "Fixed Size Image", "media": "image", "params": [
			{"paramID": "size", "flagID": "size", "allowedValues": ["1024x1024", "1024x1536", "1536x1024"]}
		]},
		{"id": "custom-size-image", "name": "Custom Size Image", "media": "image", "params": [
			{"paramID": "size", "flagID": "size", "customSize": {"maxRatio": 3.0, "maxEdge": 3840, "minPx": 655360, "maxPx": 8294400, "edgeIncrem": 16, "longEdge": 1536}}
		]}
	],
	"config": {
		"imageAPI": {"genURL": "https://fixture.example/images", "inputMediaURL": "https://fixture.example/edits", "inputMediaPayloadType": "form", "inputMediaProvParam": "image[]", "inputMediaStyle": "parts", "fallbackExt": ".png"}
	}`)

	checkFixtureIdentity(t, provCfg)

	for _, model := range provCfg.Models {
		if !model.SupportsParam(params.FlagTypeSize) {
			t.Errorf("✗ model %s does not declare a size entry", model.ID)
		}
	}

	if constructedTestProvider(t, &provCfg) == nil {
		t.Errorf("✗ the constructor composed no generator over the decoded entry")
	}

	if !t.Failed() {
		t.Log("✓ the size-declaring fixture loads with its identity, size declared on every model, and a generator")
	}
}

// TestSizeFreeCfg verifies invariant #2: Size-free configuration.
//
// What is being tested:
// Given image and video models without size parameters, catalog loading must preserve the fixture
// provider identity and leave size unsupported on every model, and NewProvider must return a
// generator.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSizeFreeCfg(t *testing.T) {
	provCfg := decodeFixtureCfg(t, `
	"models": [
		{"id": "declared-aspect-set-image", "name": "Declared Aspect Set Image", "media": "image", "params": [
			{"paramID": "aspect_ratio", "flagID": "aspect-ratio", "allowedValues": ["1:1", "16:9", "9:16", "auto"]},
			{"paramID": "resolution", "flagID": "resolution", "allowedValues": ["1k", "2k"]}
		]},
		{"id": "range-duration-video", "name": "Range Duration Video", "media": "video", "params": [
			{"paramID": "aspect_ratio", "flagID": "aspect-ratio", "allowedValues": ["1:1", "16:9", "9:16"]},
			{"paramID": "duration", "flagID": "duration", "minValue": 1, "maxValue": 15}
		]}
	],
	"config": {
		"imageAPI": {"genURL": "https://fixture.example/images", "inputMediaURL": "https://fixture.example/edits", "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images", "fallbackExt": ".jpg"},
		"videoAPI": {"asyncJobsURL": "https://fixture.example/videos", "jobIDField": "request_id", "progressStatusText": ["pending"], "completedStatusText": "done", "failedStatusText": ["failed"], "urlPathSeq": ["video", "url"], "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images", "fallbackExt": ".mp4", "pollInterval": 5, "pollTimeout": 1800}
	}`)

	checkFixtureIdentity(t, provCfg)

	for _, model := range provCfg.Models {
		if model.SupportsParam(params.FlagTypeSize) {
			t.Errorf("✗ model %s declares size; the fixture declares none", model.ID)
		}
	}

	if constructedTestProvider(t, &provCfg) == nil {
		t.Errorf("✗ the constructor composed no generator over the decoded entry")
	}

	if !t.Failed() {
		t.Log("✓ the size-free fixture loads with its identity, no size declared on any model, and a generator")
	}
}

// TestAggregatorCfg verifies invariant #3: Aggregator configuration.
//
// What is being tested:
// Given an aggregator configuration with image and video APIs, catalog loading must preserve the
// fixture identity and Aggregator flag, and NewProvider must return a generator.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAggregatorCfg(t *testing.T) {
	provCfg := decodeFixtureCfg(t, `
	"aggregator": true,
	"models": [
		{"id": "vendor-one/relayed-image", "name": "Vendor One: Relayed Image", "media": "image", "params": [
			{"paramID": "aspect_ratio", "flagID": "aspect-ratio", "allowedValues": ["1:1", "16:9", "9:16"]}
		]},
		{"id": "vendor-two/relayed-video", "name": "Vendor Two: Relayed Video", "media": "video", "params": [
			{"paramID": "duration", "flagID": "duration", "allowedValues": ["4", "8"]}
		]}
	],
	"config": {
		"imageAPI": {"genURL": "https://fixture.example/images", "inputMediaPayloadType": "json", "inputMediaProvParam": "input_references", "inputMediaStyle": "nested", "fallbackExt": ".png"},
		"videoAPI": {"asyncJobsURL": "https://fixture.example/videos", "jobIDField": "id", "progressStatusText": ["pending", "in_progress"], "completedStatusText": "completed", "failedStatusText": ["failed", "cancelled"], "urlPathSeq": ["unsigned_urls", "0"], "urlContentPath": "/content?index=0", "authDownload": true, "inputMediaPayloadType": "json", "inputMediaProvParam": "input_references", "inputMediaStyle": "nested", "fallbackExt": ".mp4", "pollInterval": 30, "pollTimeout": 1800}
	}`)

	checkFixtureIdentity(t, provCfg)

	if !provCfg.Identity().Aggregator {
		t.Errorf("✗ identity = %+v, want the aggregator flag the fixture declares", provCfg.Identity())
	}

	if constructedTestProvider(t, &provCfg) == nil {
		t.Errorf("✗ the constructor composed no generator over the decoded entry")
	}

	if !t.Failed() {
		t.Log("✓ the aggregator fixture loads with its identity and composes a generator")
	}
}

// TestURLResponseImageCfg verifies invariant #4: URL-response image configuration.
//
// What is being tested:
// Given the URL-response image fixture, catalog loading must preserve the provider identity, model
// IDs in document order, and RespImageURL flag. NewProvider must return a generator.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestURLResponseImageCfg(t *testing.T) {
	provCfg := urlResponseImageFixtureCfg(t)

	checkFixtureIdentity(t, provCfg)

	declaredModelIDs := declaredModelIDs(t, urlResponseImageFixtureDocument)

	loadedModelIDs := make([]string, 0, len(provCfg.Models))
	for _, model := range provCfg.Models {
		loadedModelIDs = append(loadedModelIDs, model.ID)
	}

	if !slices.Equal(loadedModelIDs, declaredModelIDs) {
		t.Errorf("✗ loaded model records = %v, want the declared records %v", loadedModelIDs, declaredModelIDs)
	}

	if !provCfg.Config.ImageAPI.RespImageURL {
		t.Errorf("✗ image API = %+v, want the URL response the fixture declares", provCfg.Config.ImageAPI)
	}

	if constructedTestProvider(t, &provCfg) == nil {
		t.Error("✗ the constructor composed no generator over the decoded entry")
	}

	if !t.Failed() {
		t.Log("✓ the URL-response image fixture loads with its identity, exactly its declared models, and a generator")
	}
}

// TestImageGenerateRequests verifies invariant #5: Image generate requests.
//
// What is being tested:
// Given text inputs, the descriptor generator's Generate must use the configured generation route
// with exactly the expected model, prompt, count, seed, size, and supported negative-prompt fields.
// A strength-only request must stay on that route without a negative prompt; an image URL must
// select editing and appear as a bare media string alongside strength.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestImageGenerateRequests(t *testing.T) {
	provCfg := urlResponseImageFixtureCfg(t)
	strengthFlagID := params.FlagType("strength")
	seedFlagID := params.FlagType("seed")
	negativePromptFlagID := params.FlagType("negative-prompt")

	negativePromptModel := configuredModel(t, provCfg, negativePromptImageModelID)
	rasterModelWithoutNegativePrompt := configuredModel(t, provCfg, strengthOnlyImageModelID)
	harness := newURLArtifactHTTPHarness(t)
	generationPath := endpointPath(t, provCfg.Config.ImageAPI.GenURL)
	editingPath := endpointPath(t, provCfg.Config.ImageAPI.InputMediaURL)

	t.Run("text request carries configured generation fields", func(t *testing.T) {
		checkTextGenerationRequest(t, provCfg, negativePromptModel, seedFlagID, negativePromptFlagID, harness, generationPath)

		if !t.Failed() {
			t.Log("✓ the text request carries the configured generation fields")
		}
	})

	t.Run("strength stays on the text route and negative prompt stays model-scoped", func(t *testing.T) {
		checkStrengthTextRequest(t, provCfg, rasterModelWithoutNegativePrompt, negativePromptModel, strengthFlagID, negativePromptFlagID, harness, generationPath)

		if !t.Failed() {
			t.Log("✓ strength stays on the text route and negative prompt stays model-scoped")
		}
	})

	t.Run("one input selects editing and carries a bare media string", func(t *testing.T) {
		checkEditingRequest(t, provCfg, rasterModelWithoutNegativePrompt, strengthFlagID, harness, editingPath)

		if !t.Failed() {
			t.Log("✓ one input selects editing and carries a bare media string")
		}
	})

	if !t.Failed() {
		t.Log("✓ text and editing requests follow the configured routes and fields")
	}
}

// TestImageAdjustParams verifies invariant #6: Image parameter adjustment.
//
// What is being tested:
// Given excessive strength, the descriptor generator's AdjustParams must cap it at the declared
// maximum and return a Capped record. It must preserve vector aspect 16:9 without changes and
// derive a declared raster size from aspect or resolution without marking that input ignored.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestImageAdjustParams(t *testing.T) {
	provCfg := urlResponseImageFixtureCfg(t)
	strengthFlagID := params.FlagType("strength")
	rasterModelWithoutNegativePrompt := configuredModel(t, provCfg, strengthOnlyImageModelID)
	vectorModel := configuredModel(t, provCfg, vectorImageModelID)
	generator := constructedTestProvider(t, &provCfg)

	t.Run("strength above the declared maximum is capped", func(t *testing.T) {
		checkStrengthCap(t, generator, rasterModelWithoutNegativePrompt, strengthFlagID)

		if !t.Failed() {
			t.Log("✓ strength above the declared maximum is capped")
		}
	})

	t.Run("vector aspect remains a provider size value", func(t *testing.T) {
		checkVectorAspect(t, generator, vectorModel)

		if !t.Failed() {
			t.Log("✓ vector aspect remains a provider size value")
		}
	})

	t.Run("raster aspect derives a configured size", func(t *testing.T) {
		checkRasterSizeDerivation(t, generator, rasterModelWithoutNegativePrompt, params.FlagTypeAspect, "16:9")

		if !t.Failed() {
			t.Log("✓ raster aspect derives a configured size")
		}
	})

	t.Run("raster resolution derives a configured size without an ignored record", func(t *testing.T) {
		checkRasterSizeDerivation(t, generator, rasterModelWithoutNegativePrompt, params.FlagTypeResolution, "1080p")

		if !t.Failed() {
			t.Log("✓ raster resolution derives a configured size without an ignored record")
		}
	})

	if !t.Failed() {
		t.Log("✓ strength and sizing adjustments follow the declared constraints")
	}
}

// TestImageResponseHandling verifies invariant #7: Image response handling.
//
// What is being tested:
// Given an SVG result URL, the descriptor generator's Generate must return one file-backed .svg
// with exact bytes and send the adjusted vector aspect under its provider field. A structured
// failure must return ErrResponseServer with its message; an unstructured failure must return
// ErrResponseStatus without ErrResponseServer.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestImageResponseHandling(t *testing.T) {
	provCfg := urlResponseImageFixtureCfg(t)
	vectorModel := configuredModel(t, provCfg, vectorImageModelID)
	rasterModel := configuredModel(t, provCfg, strengthOnlyImageModelID)
	harness := newURLArtifactHTTPHarness(t)

	t.Run("SVG content type selects the SVG extension", func(t *testing.T) {
		checkSVGResponse(t, provCfg, vectorModel, harness)

		if !t.Failed() {
			t.Log("✓ SVG content type selects the SVG extension")
		}
	})

	t.Run("structured provider message is marked as server text", func(t *testing.T) {
		checkStructuredError(t, provCfg, rasterModel, harness)

		if !t.Failed() {
			t.Log("✓ a structured provider message is marked as server text")
		}
	})

	t.Run("bare provider body has no server-message classification", func(t *testing.T) {
		checkBareError(t, provCfg, rasterModel, harness)

		if !t.Failed() {
			t.Log("✓ a bare provider body has no server-message classification")
		}
	})

	if !t.Failed() {
		t.Log("✓ URL and failure responses retain their declared behavior")
	}
}

// TestDescriptorGenerate verifies invariant #8: Descriptor generation.
//
// What is being tested:
// Given image and video models, the descriptor generator's Generate must return the decoded image
// bytes or a file containing the exact video bytes. NewProvider must reject a missing API section
// with ErrProvConfigInvalid and name ImageAPI or VideoAPI as applicable.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDescriptorGenerate(t *testing.T) {
	video := []byte("VID-BYTES")
	srv := harnessSrv(t, video)
	t.Setenv("FIXT_KEY", "k")
	t.Run("an image run yields the artifact and the sent report", func(t *testing.T) {
		checkGenerateImage(t, srv.URL)

		if !t.Failed() {
			t.Log("✓ an image run yields the artifact and the sent report")
		}
	})
	t.Run("a video run yields the file-backed artifact under the duration key", func(t *testing.T) {
		checkGenerateVideo(t, srv.URL, video)

		if !t.Failed() {
			t.Log("✓ a video run yields the file-backed artifact under the duration key")
		}
	})
	t.Run("a medium without its API description fails classified", func(t *testing.T) {
		checkGenerateMissingAPI(t, srv.URL)

		if !t.Failed() {
			t.Log("✓ a medium without its API description fails classified")
		}
	})

	if !t.Failed() {
		t.Log("✓ the medium selects its description, the run yields artifacts with the sent report, and a missing description fails classified")
	}
}

// TestDescriptorAdjustParams verifies invariant #9: Descriptor adjust params.
//
// What is being tested:
// Given the fixture constraints, the descriptor generator's AdjustParams must derive or snap sizes,
// normalize or reject quality, bound counts and durations, and record undeclared flags as Ignored.
// Declared size, duration, and aspect members must remain unchanged. Every case must return exactly
// the expected parameter values and change records, regardless of record order.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDescriptorAdjustParams(t *testing.T) {
	resizing := resizingVideoFixtureCfg(t)
	nonResizing := nonResizingVideoFixtureCfg(t)

	cases := []adjustContractCase{
		{
			name: "an equal ratio derives the declared fixed size", provCfg: resizing, modelID: fixedSizeImageModelID,
			inputs: params.FlagInputs{params.FlagTypeAspect: "2:3"},
			gp:     params.Values{params.FlagTypeSize: "1024x1536"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeAspect, Type: params.ChangeDerived, InputVal: "2:3", WireVal: "1024x1536"},
			},
		},
		{
			name: "a quality outside the declared set rejects naming it", provCfg: resizing, modelID: fixedSizeImageModelID,
			inputs: params.FlagInputs{params.FlagTypeQuality: "ultra"},
			gp:     params.Values{},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeQuality, Type: params.ChangeRejected, InputVal: "ultra", Comment: "low|medium|high|auto"},
			},
		},
		{
			name: "a quality member passes as the declared spelling", provCfg: resizing, modelID: fixedSizeImageModelID,
			inputs: params.FlagInputs{params.FlagTypeQuality: "HIGH"},
			gp:     params.Values{params.FlagTypeQuality: "high"},
		},
		{
			name: "an over-maximum count caps", provCfg: resizing, modelID: fixedSizeImageModelID,
			inputs: params.FlagInputs{params.FlagTypeImageN: 12},
			gp:     params.Values{params.FlagTypeImageN: 10},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeImageN, Type: params.ChangeCapped, InputVal: "12", WireVal: "10", Comment: "max 10"},
			},
		},
		{
			name: "a zero count raises to the floor", provCfg: resizing, modelID: fixedSizeImageModelID,
			inputs: params.FlagInputs{params.FlagTypeImageN: 0},
			gp:     params.Values{params.FlagTypeImageN: 1},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeImageN, Type: params.ChangeRaised, InputVal: "0", WireVal: "1", Comment: "min 1"},
			},
		},
		{
			name: "an unconsumed supplied flag warns as ignored", provCfg: resizing, modelID: fixedSizeImageModelID,
			inputs: params.FlagInputs{params.FlagTypeThinkingLevel: "low"},
			gp:     params.Values{},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeThinkingLevel, Type: params.ChangeIgnored, Comment: fmt.Sprintf(generation.ReasonNotConsumed, fixedSizeImageModelID)},
			},
		},
		{
			name: "a declared duration range caps a video request", provCfg: nonResizing, modelID: rangeDurationVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeDuration: 20},
			gp:     params.Values{params.FlagTypeDuration: 15},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeDuration, Type: params.ChangeCapped, InputVal: "20", WireVal: "15", Comment: "max 15"},
			},
		},
		{
			name: "a pass-through aspect member passes verbatim", provCfg: nonResizing, modelID: declaredAspectSetImageModelID,
			inputs: params.FlagInputs{params.FlagTypeAspect: "auto"},
			gp:     params.Values{params.FlagTypeAspect: "auto"},
		},
		{
			name: "a duration outside the declared set snaps to the nearest member", provCfg: resizing, modelID: twoSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeDuration: 5},
			gp:     params.Values{params.FlagTypeDuration: 4},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeDuration, Type: params.ChangeSnapped, InputVal: "5", WireVal: "4"},
			},
		},
		// The two longest declared durations and the two declared 1080p sizes each prove a
		// declared member survives untouched with no record, so a membership the model
		// declares is never snapped away.
		{
			name: "the longest duration passes untouched on the two-size video model", provCfg: resizing, modelID: twoSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeDuration: 20},
			gp:     params.Values{params.FlagTypeDuration: 20},
		},
		{
			name: "the second longest duration passes untouched on the two-size video model", provCfg: resizing, modelID: twoSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeDuration: 16},
			gp:     params.Values{params.FlagTypeDuration: 16},
		},
		{
			name: "the longest duration passes untouched on the six-size video model", provCfg: resizing, modelID: sixSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeDuration: 20},
			gp:     params.Values{params.FlagTypeDuration: 20},
		},
		{
			name: "the second longest duration passes untouched on the six-size video model", provCfg: resizing, modelID: sixSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeDuration: 16},
			gp:     params.Values{params.FlagTypeDuration: 16},
		},
		{
			name: "the landscape 1080p size passes untouched on the six-size video model", provCfg: resizing, modelID: sixSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeSize: "1920x1080"},
			gp:     params.Values{params.FlagTypeSize: "1920x1080"},
		},
		{
			name: "the portrait 1080p size passes untouched on the six-size video model", provCfg: resizing, modelID: sixSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeSize: "1080x1920"},
			gp:     params.Values{params.FlagTypeSize: "1080x1920"},
		},
		// The two-size video model does not declare the 1080p sizes, so the same request
		// there selects the nearest size its own set declares.
		{
			name: "the landscape 1080p size is not offered on the two-size video model", provCfg: resizing, modelID: twoSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeSize: "1920x1080"},
			gp:     params.Values{params.FlagTypeSize: "1280x720"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeSize, Type: params.ChangeSnapped, InputVal: "1920x1080", WireVal: "1280x720"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkAdjustContract(t, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the one AdjustParams call returns the adjusted values and the complete record set")
	}
}

// TestDescriptorAdjustResizeDecision verifies invariant #10: Descriptor adjust resize decision.
//
// What is being tested:
// Given resizing video settings, the descriptor generator's AdjustParams must adopt landscape or
// portrait dimensions when size is absent, preserve explicit size, and return the exact Derived,
// Conformed, and applicable Capped records. Without resizing enabled it must return no conformance
// records. Undecodable image bytes must return ErrInputMediaDecode under ErrInputMedia and name the
// source path.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDescriptorAdjustResizeDecision(t *testing.T) {
	resizing := resizingVideoFixtureCfg(t)
	nonResizing := nonResizingVideoFixtureCfg(t)

	cases := []adjustContractCase{
		{
			name: "no requested size adopts from a landscape reference", provCfg: resizing, modelID: twoSizeVideoModelID,
			images: []media.Input{tinyPNG(t, 8, 4)},
			gp:     params.Values{params.FlagTypeSize: "1280x720"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeSize, Type: params.ChangeDerived, WireVal: "1280x720", Comment: generation.ReasonAdoptedFromReference},
				{FlagID: params.FlagTypeInputMedia, Type: params.ChangeConformed, InputVal: "8x4", WireVal: "1280x720", Comment: fmt.Sprintf(generation.ReasonMatchedToRequest, twoSizeVideoModelID)},
			},
		},
		{
			name: "no requested size adopts from a portrait reference", provCfg: resizing, modelID: twoSizeVideoModelID,
			images: []media.Input{tinyPNG(t, 4, 8)},
			gp:     params.Values{params.FlagTypeSize: "720x1280"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeSize, Type: params.ChangeDerived, WireVal: "720x1280", Comment: generation.ReasonAdoptedFromReference},
				{FlagID: params.FlagTypeInputMedia, Type: params.ChangeConformed, InputVal: "4x8", WireVal: "720x1280", Comment: fmt.Sprintf(generation.ReasonMatchedToRequest, twoSizeVideoModelID)},
			},
		},
		{
			name: "a requested size conforms without adoption", provCfg: resizing, modelID: twoSizeVideoModelID,
			inputs: params.FlagInputs{params.FlagTypeSize: "720x1280"},
			images: []media.Input{tinyPNG(t, 8, 4)},
			gp:     params.Values{params.FlagTypeSize: "720x1280"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeInputMedia, Type: params.ChangeConformed, InputVal: "8x4", WireVal: "720x1280", Comment: fmt.Sprintf(generation.ReasonMatchedToRequest, twoSizeVideoModelID)},
			},
		},
		{
			name: "an over-cap reference set decides the cap and conforms the kept first", provCfg: resizing, modelID: twoSizeVideoModelID,
			images: []media.Input{tinyPNG(t, 8, 4), tinyPNG(t, 4, 8)},
			gp:     params.Values{params.FlagTypeSize: "1280x720"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped, InputVal: "2", WireVal: "1", Comment: "max 1"},
				{FlagID: params.FlagTypeSize, Type: params.ChangeDerived, WireVal: "1280x720", Comment: generation.ReasonAdoptedFromReference},
				{FlagID: params.FlagTypeInputMedia, Type: params.ChangeConformed, InputVal: "8x4", WireVal: "1280x720", Comment: fmt.Sprintf(generation.ReasonMatchedToRequest, twoSizeVideoModelID)},
			},
		},
		{
			name: "a non-resizing video API records no conformance", provCfg: nonResizing, modelID: rangeDurationVideoModelID,
			images: []media.Input{tinyPNG(t, 8, 4)},
			gp:     params.Values{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkAdjustContract(t, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	t.Run("an unreadable first reference fails classified", func(t *testing.T) {
		p := constructedTestProvider(t, &resizing)
		model := configuredModel(t, resizing, twoSizeVideoModelID)
		badImage := media.Input{Bytes: []byte("NOT-AN-IMAGE"), MIME: "image/png", Filepath: "/tmp/bad.png"}

		_, err := p.AdjustParams(&model, params.FlagInputs{}, []media.Input{badImage}, nil)
		if err == nil || !errors.Is(err, errs.ErrInputMediaDecode) || !errors.Is(err, errs.ErrInputMedia) {
			t.Errorf("✗ AdjustParams error = %v, want the classified input-media decode failure", err)
		}

		if err != nil && !strings.Contains(err.Error(), "/tmp/bad.png") {
			t.Errorf("✗ error %q does not name the failing image's path", err)
		}

		if !t.Failed() {
			t.Log("✓ an unreadable first reference fails classified")
		}
	})

	if !t.Failed() {
		t.Log("✓ the resize decision's records and its classified failure return from the one AdjustParams call")
	}
}

// TestConfiguredImageGeneration verifies invariant #11: Configured image generation.
//
// What is being tested:
// Given each listed MAI or Recraft model, descriptor generation must use the configured endpoint
// and exact model ID and return the expected image bytes. MAI requests must carry five references
// and aspect auto; Recraft requests must retain the supplied style ID and flexible match choice.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestConfiguredImageGeneration(t *testing.T) {
	imageCases := []struct {
		modelID            string
		providerConfigJSON []byte
		providerID         string
	}{
		{"microsoft/mai-image-2.6", providerconfig.JSONOpenRouter, providerconfig.IDOpenRouter},
		{"microsoft/mai-image-2.6-flash", providerconfig.JSONOpenRouter, providerconfig.IDOpenRouter},
		{"recraftv4_styles", providerconfig.JSONRecraft, providerconfig.IDRecraft},
		{"recraftv4_styles_pro", providerconfig.JSONRecraft, providerconfig.IDRecraft},
		{"recraftv4_styles_vector", providerconfig.JSONRecraft, providerconfig.IDRecraft},
		{"recraftv4_styles_pro_vector", providerconfig.JSONRecraft, providerconfig.IDRecraft},
	}
	for _, imageCase := range imageCases {
		t.Run(imageCase.modelID, func(t *testing.T) {
			checkConfiguredImageGeneration(t, imageCase.providerConfigJSON, imageCase.providerID, imageCase.modelID)
		})
	}
}

// TestXAILatestModelRequest verifies invariant #12: xAI latest model.
//
// What is being tested:
// Given either the bare or xai-qualified latest model selector, descriptor generation must send
// grok-imagine-image-quality-latest unchanged to the expected generation or editing route and
// return the exact provider image bytes.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestXAILatestModelRequest(t *testing.T) {
	catalog, err := catalog.LoadCatalog(loadEnumFlags(t), catalog.Source{
		ProviderID: "xai", ConfigBytes: providerconfig.JSONXAI,
	})
	if err != nil {
		t.Fatalf("💣 load xAI catalog: %v", err)
	}

	const latestModelID = "grok-imagine-image-quality-latest"

	for _, modelSelector := range []string{latestModelID, "xai/" + latestModelID} {
		for _, requestPath := range []string{"/v1/images/generations", "/v1/images/edits"} {
			t.Run(modelSelector+requestPath, func(t *testing.T) {
				checkXAILatestRequest(t, catalog, modelSelector, latestModelID, requestPath)
			})
		}
	}

	if !t.Failed() {
		t.Log("✓ both selector forms preserve the latest model ID for generation and editing")
	}
}

// adjustContractCase supplies a provider, model, and inputs to AdjustParams and records the
// expected values and adjustments. Adjustment order does not affect the comparison.
type adjustContractCase struct {
	name    string
	provCfg catalog.Provider
	modelID string
	inputs  params.FlagInputs
	images  []media.Input
	gp      params.Values
	records []params.Adjustment
}

// The fixture provider identity every fixture configuration document declares.
const (
	fixtureProviderID          = "fixture-provider"
	fixtureProviderDisplayName = "Fixture Provider"
	fixtureProviderKeyEnvVar   = "FIXTURE_API_KEY"
)

// The fixture model IDs of the adjustment tables, each named by the constraint shape the model
// declares.
const (
	fixedSizeImageModelID         = "fixed-size-image"
	twoSizeVideoModelID           = "two-size-video"
	sixSizeVideoModelID           = "six-size-video"
	rangeDurationVideoModelID     = "range-duration-video"
	declaredAspectSetImageModelID = "declared-aspect-set-image"
)

// The fixture model IDs of the URL-response image fixture, each named by the parameter behavior the
// model carries.
const (
	negativePromptImageModelID = "negative-prompt-image"
	strengthOnlyImageModelID   = "strength-only-image"
	vectorImageModelID         = "vector-image"
)

// urlResponseImageFixtureDocument is the document body of the fixture provider whose image API
// answers synchronously with artifact URLs: a JSON generation route, a JSON editing route that
// takes one remote image as a bare string field, and three image models — a raster model declaring
// a negative prompt, a raster model declaring strength without a negative prompt, and a vector
// model whose provider size field takes aspect ratios. Each model carries one alias.
const urlResponseImageFixtureDocument = `
	"models": [
		{"id": "negative-prompt-image", "name": "Negative Prompt Image", "media": "image", "aliases": ["negative-prompt"], "params": [
			{"paramID": "size", "flagID": "size", "allowedValues": ["1024x1024", "1536x768", "768x1536", "1280x832", "832x1280"]},
			{"flagID": "aspect-ratio"},
			{"flagID": "resolution"},
			{"paramID": "n", "flagID": "num-images", "minValue": 1, "maxValue": 6},
			{"paramID": "random_seed", "flagID": "seed"},
			{"paramID": "negative_prompt", "flagID": "negative-prompt"},
			{"paramID": "strength", "flagID": "strength", "minValue": 0, "maxValue": 1},
			{"flagID": "input-media", "maxMultiple": 1}
		]},
		{"id": "strength-only-image", "name": "Strength Only Image", "media": "image", "aliases": ["strength-only"], "params": [
			{"paramID": "size", "flagID": "size", "allowedValues": ["1024x1024", "1536x768", "768x1536", "1280x832", "832x1280"]},
			{"flagID": "aspect-ratio"},
			{"flagID": "resolution"},
			{"paramID": "n", "flagID": "num-images", "minValue": 1, "maxValue": 6},
			{"paramID": "random_seed", "flagID": "seed"},
			{"paramID": "strength", "flagID": "strength", "minValue": 0, "maxValue": 1},
			{"flagID": "input-media", "maxMultiple": 1}
		]},
		{"id": "vector-image", "name": "Vector Image", "media": "image", "aliases": ["vector"], "params": [
			{"paramID": "size", "flagID": "aspect-ratio", "allowedValues": ["1:1", "2:1", "1:2", "3:2", "2:3", "16:9", "9:16"]},
			{"paramID": "n", "flagID": "num-images", "minValue": 1, "maxValue": 6},
			{"paramID": "random_seed", "flagID": "seed"},
			{"paramID": "strength", "flagID": "strength", "minValue": 0, "maxValue": 1},
			{"flagID": "input-media", "maxMultiple": 1}
		]}
	],
	"config": {
		"imageAPI": {"genURL": "https://fixture.example/images/generations", "inputMediaURL": "https://fixture.example/images/edits", "inputMediaPayloadType": "json", "inputMediaProvParam": "image_url", "inputMediaStyle": "string", "fallbackExt": ".png", "respImageURL": true}
	}`

// urlArtifactHTTPHarness records the latest generation POST and serves either a configured error or
// a URL response whose artifact download has the configured media type and bytes.
type urlArtifactHTTPHarness struct {
	test               testing.TB
	server             *httptest.Server
	requestPath        string
	requestBody        map[string]any
	responseStatusCode int
	responseBody       string
	successBody        string
	artifactStatusCode int
	artifactMIME       string
	artifactBytes      []byte
}

// imageRunOutcome contains the adjusted parameters, adjustment records, result, and error from one
// descriptor-driven image run against the URL artifact harness.
type imageRunOutcome struct {
	params  params.Values
	records []params.Adjustment
	result  generation.Result
	err     error
}

// checkXAILatestRequest resolves a selector and verifies its exact HTTP model ID, operation route,
// and returned image bytes.
func checkXAILatestRequest(t *testing.T, catalog *catalog.Catalog, modelSelector, modelID, requestPath string) {
	t.Helper()

	modelMatches, resolveErr := catalog.ResolveModelInput(modelSelector)
	if resolveErr != nil || len(modelMatches) != 1 {
		t.Fatalf("💣 resolve %q: matches=%v error=%v", modelSelector, modelMatches, resolveErr)
	}

	harness := newURLArtifactHTTPHarness(t)
	harness.successBody = fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString(harness.artifactBytes))

	var mediaInputs []media.Input
	if requestPath == "/v1/images/edits" {
		mediaInputs = []media.Input{{URL: "https://media.example/source.png", MIME: "image/png"}}
	}

	generation := runImageGeneration(t, catalog.Providers[0], modelMatches[0].Model, nil, mediaInputs, harness)
	if generation.err != nil || len(generation.result.Artifacts) != 1 {
		t.Fatalf("💣 image generation: artifacts=%d error=%v", len(generation.result.Artifacts), generation.err)
	}

	if harness.requestBody["model"] != modelID || harness.requestPath != requestPath {
		t.Errorf("✗ request = %s model=%v; require %s model=%s", harness.requestPath, harness.requestBody["model"], requestPath, modelID)
	}

	if !bytes.Equal(generation.result.Artifacts[0].Data, harness.artifactBytes) {
		t.Error("✗ returned image differs from the provider response")
	}

	if !t.Failed() {
		t.Log("✓ request preserves the model ID and returns the provider image")
	}
}

// checkConfiguredImageGeneration verifies the selected route, input fields, and returned bytes.
func checkConfiguredImageGeneration(t *testing.T, providerConfigJSON []byte, providerID, modelID string) {
	t.Helper()

	providerConfig := loadTestProvider(t, providerID, providerConfigJSON)

	model := configuredModel(t, providerConfig, modelID)
	harness := newURLArtifactHTTPHarness(t)
	flagInputs := params.FlagInputs{"style-id": "6f9c1a2b-4d5e-4678-9123-abcdef012345", "style-match": "flexible"}

	var mediaInputs []media.Input

	if providerConfig.ID == providerconfig.IDOpenRouter {
		harness.successBody = fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString(harness.artifactBytes))
		flagInputs = params.FlagInputs{params.FlagTypeAspect: "auto"}
		mediaInputs = []media.Input{{URL: "https://media.example/one.png"}, {URL: "https://media.example/two.png"}, {URL: "https://media.example/three.png"}, {URL: "https://media.example/four.png"}, {URL: "https://media.example/five.png"}}
	}

	generation := runImageGeneration(t, providerConfig, model, flagInputs, mediaInputs, harness)
	cleanupDownloadedArtifacts(t, generation.result.Artifacts)

	if generation.err != nil || len(generation.result.Artifacts) != 1 {
		t.Fatalf("💣 generation returned %d artifacts, error %v", len(generation.result.Artifacts), generation.err)
	}

	artifactBytes := generation.result.Artifacts[0].Data

	var readErr error
	if generation.result.Artifacts[0].TmpPath != "" {
		artifactBytes, readErr = os.ReadFile(generation.result.Artifacts[0].TmpPath)
	}

	if readErr != nil || !bytes.Equal(artifactBytes, harness.artifactBytes) {
		t.Errorf("✗ downloaded artifact differs from the provider output: bytes=%q error=%v", artifactBytes, readErr)
	}

	if harness.requestBody["model"] != model.ID || harness.requestPath != endpointPath(t, providerConfig.Config.ImageAPI.GenURL) {
		t.Errorf("✗ generation selected an incorrect model or endpoint: %s, %#v", harness.requestPath, harness.requestBody)
	}

	if providerConfig.ID == providerconfig.IDOpenRouter {
		references, referencesOK := harness.requestBody["input_references"].([]any)
		if !referencesOK || len(references) != 5 || harness.requestBody["aspect_ratio"] != "auto" {
			t.Errorf("✗ generation lost references or automatic aspect: %#v", harness.requestBody)
		}
	} else if harness.requestBody["style_id"] != flagInputs["style-id"] || harness.requestBody["style_match"] != "flexible" {
		t.Errorf("✗ generation lost its supplied style: %#v", harness.requestBody)
	}

	if !t.Failed() {
		t.Log("✓ configured image inputs reach generation and return an artifact")
	}
}

// loadEnumFlags loads the embedded enumeration for the decode fixtures.
func loadEnumFlags(t *testing.T) []params.Flag {
	t.Helper()

	flags := params.Flags()

	return flags
}

// urlResponseImageFixtureCfg decodes the fixture provider whose image API answers synchronously
// with artifact URLs; the document body is urlResponseImageFixtureDocument.
func urlResponseImageFixtureCfg(t *testing.T) catalog.Provider {
	t.Helper()

	return decodeFixtureCfg(t, urlResponseImageFixtureDocument)
}

// declaredModelIDs reads the model IDs a fixture document body declares, in document order, without
// going through the provider loader.
func declaredModelIDs(t *testing.T, documentBody string) []string {
	t.Helper()

	var declared struct {
		Models []struct {
			ID string `json:"id"`
		} `json:"models"`
	}

	if err := json.Unmarshal([]byte("{"+documentBody+"}"), &declared); err != nil {
		t.Fatalf("💣 fixture document body is not a JSON object: %v", err)
	}

	modelIDs := make([]string, 0, len(declared.Models))
	for _, model := range declared.Models {
		modelIDs = append(modelIDs, model.ID)
	}

	return modelIDs
}

// loadTestProvider loads one validated fixture provider through the catalog.
func loadTestProvider(test testing.TB, providerID string, configuration []byte) catalog.Provider {
	test.Helper()

	loadedCatalog, err := catalog.LoadCatalog(params.Flags(), catalog.Source{ProviderID: providerID, ConfigBytes: configuration})
	if err != nil {
		test.Fatalf("💣 fixture catalog failed to load: %v", err)
	}

	providerDescription, loaded := loadedCatalog.Provider(providerID)
	if !loaded {
		test.Fatalf("💣 fixture provider %s failed to load: %v", providerID, loadedCatalog.ConfigError(providerID))
	}

	return providerDescription
}

// decodeFixtureCfg decodes one fixture configuration document through the shared strict loader. The
// document body carries the models and config sections; the fixture provider identity is prepended.
func decodeFixtureCfg(t *testing.T, documentBody string) catalog.Provider {
	t.Helper()

	document := fmt.Sprintf(`{"id": %q, "displayName": %q, "apiKeyEnvVar": %q, %s}`,
		fixtureProviderID, fixtureProviderDisplayName, fixtureProviderKeyEnvVar, documentBody)

	provCfg := loadTestProvider(t, fixtureProviderID, []byte(document))

	return provCfg
}

// checkFixtureIdentity asserts the decoded configuration carries the fixture provider identity its
// document declares.
func checkFixtureIdentity(t *testing.T, provCfg catalog.Provider) {
	t.Helper()

	p := provCfg.Identity()
	if p.ID != fixtureProviderID || p.DisplayName != fixtureProviderDisplayName || p.APIKeyEnvVar != fixtureProviderKeyEnvVar {
		t.Errorf("✗ identity = %+v, want %s/%s/%s", p, fixtureProviderID, fixtureProviderDisplayName, fixtureProviderKeyEnvVar)
	}
}

// resizingVideoFixtureCfg decodes the fixture provider whose video API resizes local input images
// to the requested size. It declares a fixed-size image model (a snapping size set, a quality set,
// a count range) and two set-bounded video models differing only in their size sets: the two-size
// model declares the 720p sizes and the six-size model adds the 1792 and 1080p sizes; both cap
// input media at one.
func resizingVideoFixtureCfg(t *testing.T) catalog.Provider {
	t.Helper()

	return decodeFixtureCfg(t, `
	"models": [
		{"id": "fixed-size-image", "name": "Fixed Size Image", "media": "image", "params": [
			{"paramID": "aspect_ratio", "flagID": "aspect-ratio"},
			{"paramID": "quality", "flagID": "quality", "allowedValues": ["low", "medium", "high", "auto"]},
			{"paramID": "n", "flagID": "num-images", "minValue": 1, "maxValue": 10},
			{"paramID": "size", "flagID": "size", "allowedValues": ["1024x1024", "1024x1536", "1536x1024"]}
		]},
		{"id": "two-size-video", "name": "Two Size Video", "media": "video", "params": [
			{"paramID": "aspect_ratio", "flagID": "aspect-ratio"},
			{"paramID": "seconds", "flagID": "duration", "allowedValues": ["4", "8", "12", "16", "20"]},
			{"flagID": "input-media", "maxMultiple": 1},
			{"paramID": "size", "flagID": "size", "allowedValues": ["1280x720", "720x1280"]}
		]},
		{"id": "six-size-video", "name": "Six Size Video", "media": "video", "params": [
			{"paramID": "aspect_ratio", "flagID": "aspect-ratio"},
			{"paramID": "seconds", "flagID": "duration", "allowedValues": ["4", "8", "12", "16", "20"]},
			{"flagID": "input-media", "maxMultiple": 1},
			{"paramID": "size", "flagID": "size", "allowedValues": ["1280x720", "720x1280", "1792x1024", "1024x1792", "1920x1080", "1080x1920"]}
		]}
	],
	"config": {
		"imageAPI": {"genURL": "https://fixture.example/images", "inputMediaURL": "https://fixture.example/edits", "inputMediaPayloadType": "form", "inputMediaProvParam": "image[]", "inputMediaStyle": "parts", "fallbackExt": ".png"},
		"videoAPI": {"asyncJobsURL": "https://fixture.example/videos", "jobIDField": "id", "progressStatusText": ["queued", "in_progress"], "completedStatusText": "completed", "failedStatusText": ["failed"], "urlContentPath": "/content", "authDownload": true, "inputMediaPayloadType": "form", "inputMediaProvParam": "input_reference", "inputMediaStyle": "parts", "inputMediaMustResize": true, "fallbackExt": ".mp4", "pollInterval": 5, "pollTimeout": 1800}
	}`)
}

// nonResizingVideoFixtureCfg decodes the fixture provider whose video API sends local input images
// as supplied. It declares a range-bounded-duration video model capping input media at one and an
// image model whose aspect set includes the pass-through member auto.
func nonResizingVideoFixtureCfg(t *testing.T) catalog.Provider {
	t.Helper()

	return decodeFixtureCfg(t, `
	"models": [
		{"id": "declared-aspect-set-image", "name": "Declared Aspect Set Image", "media": "image", "params": [
			{"paramID": "aspect_ratio", "flagID": "aspect-ratio", "allowedValues": ["1:1", "16:9", "9:16", "4:3", "3:4", "auto"]},
			{"paramID": "resolution", "flagID": "resolution", "allowedValues": ["1k", "2k"]}
		]},
		{"id": "range-duration-video", "name": "Range Duration Video", "media": "video", "params": [
			{"paramID": "aspect_ratio", "flagID": "aspect-ratio", "allowedValues": ["1:1", "16:9", "9:16", "4:3", "3:4"]},
			{"paramID": "resolution", "flagID": "resolution", "allowedValues": ["480p", "720p"]},
			{"paramID": "duration", "flagID": "duration", "minValue": 1, "maxValue": 15},
			{"flagID": "input-media", "maxMultiple": 1}
		]}
	],
	"config": {
		"imageAPI": {"genURL": "https://fixture.example/images", "inputMediaURL": "https://fixture.example/edits", "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images", "fallbackExt": ".jpg"},
		"videoAPI": {"asyncJobsURL": "https://fixture.example/videos", "urlStartPath": "/generations", "jobIDField": "request_id", "progressStatusText": ["pending"], "completedStatusText": "done", "failedStatusText": ["expired", "failed"], "urlPathSeq": ["video", "url"], "urlPathRequired": true, "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images", "fallbackExt": ".mp4", "pollInterval": 5, "pollTimeout": 1800}
	}`)
}

// newURLArtifactHTTPHarness creates a synchronous generation and artifact-download server.
func newURLArtifactHTTPHarness(t *testing.T) *urlArtifactHTTPHarness {
	t.Helper()

	harness := &urlArtifactHTTPHarness{test: t, artifactMIME: "image/png", artifactBytes: []byte("IMAGE")}
	harness.server = httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			if harness.artifactStatusCode >= http.StatusBadRequest {
				responseWriter.WriteHeader(harness.artifactStatusCode)

				return
			}

			responseWriter.Header().Set("Content-Type", harness.artifactMIME)
			_, _ = responseWriter.Write(harness.artifactBytes)

			return
		}

		harness.requestPath = request.URL.Path

		requestBytes, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			t.Errorf("✗ read recorded generation request: %v", readErr)
			responseWriter.WriteHeader(http.StatusInternalServerError)

			return
		}

		if decodeErr := json.Unmarshal(requestBytes, &harness.requestBody); decodeErr != nil {
			t.Errorf("✗ decode recorded generation request: %v", decodeErr)
			responseWriter.WriteHeader(http.StatusInternalServerError)

			return
		}

		if harness.responseStatusCode >= http.StatusBadRequest {
			responseWriter.WriteHeader(harness.responseStatusCode)
			_, _ = responseWriter.Write([]byte(harness.responseBody))

			return
		}

		if harness.successBody != "" {
			_, _ = responseWriter.Write([]byte(harness.successBody))

			return
		}

		fmt.Fprintf(responseWriter, `{"data":[{"url":%q}]}`, harness.server.URL+"/artifact")
	}))
	t.Cleanup(harness.server.Close)

	return harness
}

// resetRequestRecording clears the recorded POST and restores a successful raster response.
func (harness *urlArtifactHTTPHarness) resetRequestRecording() {
	harness.test.Helper()

	harness.requestPath = ""
	harness.requestBody = nil
	harness.responseStatusCode = http.StatusOK
	harness.responseBody = ""
	harness.successBody = ""
	harness.artifactStatusCode = http.StatusOK
	harness.artifactMIME = "image/png"
	harness.artifactBytes = []byte("IMAGE")
}

// endpointPath returns the path declared by a provider endpoint.
func endpointPath(t *testing.T, endpoint string) string {
	t.Helper()

	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("💣 parse provider endpoint %q: %v", endpoint, err)
	}

	return parsedEndpoint.Path
}

// modelParamConfig returns one declared parameter or ends the test when the model lacks it.
func modelParamConfig(t *testing.T, modelConfig catalog.Model, flagID params.FlagType) params.Definition {
	t.Helper()

	paramConfig, declared := modelConfig.Param(flagID)
	if !declared {
		t.Fatalf("💣 model %q has no %q parameter", modelConfig.ID, flagID)
	}

	return paramConfig
}

// runImageGeneration points a copy of the fixture's image endpoints at the harness, adjusts the
// supplied parameters, and runs generation through the descriptor generator.
func runImageGeneration(t *testing.T, provCfg catalog.Provider, modelConfig catalog.Model, flagInputs params.FlagInputs, mediaInputs []media.Input, harness *urlArtifactHTTPHarness) imageRunOutcome {
	t.Helper()

	providerConfig := *provCfg.Config
	imageAPI := *providerConfig.ImageAPI

	imageAPI.GenURL = harness.server.URL + endpointPath(t, imageAPI.GenURL)
	if imageAPI.InputMediaURL != "" {
		imageAPI.InputMediaURL = harness.server.URL + endpointPath(t, imageAPI.InputMediaURL)
	}

	providerConfig.ImageAPI = &imageAPI
	provCfg.Config = &providerConfig

	t.Setenv(provCfg.APIKeyEnvVar, "fixture-key")
	generator := constructedTestProvider(t, &provCfg)

	preparedGeneration, err := generator.AdjustParams(&modelConfig, flagInputs, mediaInputs, nil)

	adjustedParams, changeRecords := preparedGeneration.Params, preparedGeneration.Changes
	if err != nil {
		return imageRunOutcome{params: adjustedParams, records: changeRecords, err: err}
	}

	generationResult, err := generator.Generate(t.Context(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: provCfg.Identity(), Model: modelConfig}, Prompt: params.GetSetIf(true, "fixture prompt").ValOr(""), Preparation: generation.Preparation{Params: adjustedParams, InputMedia: mediaInputs}, APIKey: os.Getenv((catalog.ProvModelPair{Provider: provCfg.Identity(), Model: modelConfig}).Provider.APIKeyEnvVar)})

	return imageRunOutcome{params: adjustedParams, records: changeRecords, result: generationResult, err: err}
}

// checkRequestFields verifies that the recorded JSON object has exactly the required field set.
func checkRequestFields(t *testing.T, requestBody map[string]any, requiredFields ...string) {
	t.Helper()

	requiredFieldSet := make(map[string]bool, len(requiredFields))
	for _, requiredField := range requiredFields {
		requiredFieldSet[requiredField] = true
	}

	for requestField := range requestBody {
		if !requiredFieldSet[requestField] {
			t.Errorf("✗ unexpected request field %q", requestField)
		}
	}

	for requiredField := range requiredFieldSet {
		if _, present := requestBody[requiredField]; !present {
			t.Errorf("✗ required request field %q is absent", requiredField)
		}
	}
}

// cleanupDownloadedArtifacts schedules removal of downloaded artifact files after the test.
func cleanupDownloadedArtifacts(t *testing.T, artifacts []artifact.Media) {
	t.Helper()

	for _, generatedMedia := range artifacts {
		if generatedMedia.TmpPath == "" {
			continue
		}

		artifactPath := generatedMedia.TmpPath

		t.Cleanup(func() { _ = os.Remove(artifactPath) })
	}
}

// hasParamChange reports whether the records contain the requested flag and change type.
func hasParamChange(test testing.TB, records []params.Adjustment, flagID params.FlagType, changeType params.Change) bool {
	test.Helper()

	for _, changeRecord := range records {
		if changeRecord.FlagID == flagID && changeRecord.Type == changeType {
			return true
		}
	}

	return false
}

// configuredModel returns one decoded config's model by id.
func configuredModel(t *testing.T, provCfg catalog.Provider, id string) catalog.Model {
	t.Helper()

	for _, model := range provCfg.Models {
		if model.ID == id {
			return model
		}
	}

	t.Fatalf("💣 no model %q in the decoded config", id)

	return catalog.Model{}
}

// tinyPNG encodes a real w×h PNG input image, so the resize decision reads genuine image
// dimensions.
func tinyPNG(t *testing.T, w, h int) media.Input {
	t.Helper()

	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("💣 fixture PNG encode failed: %v", err)
	}

	return media.Input{Bytes: buf.Bytes(), MIME: "image/png", Filepath: fmt.Sprintf("/tmp/ref-%dx%d.png", w, h)}
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

// checkAdjustContract checks that AdjustParams succeeds with the expected values and adjustments.
func checkAdjustContract(t *testing.T, c adjustContractCase) {
	t.Helper()

	p := constructedTestProvider(t, &c.provCfg)
	model := configuredModel(t, c.provCfg, c.modelID)

	preparedGeneration, err := p.AdjustParams(&model, c.inputs, c.images, nil)
	gp, records := preparedGeneration.Params, preparedGeneration.Changes

	if err != nil {
		t.Errorf("✗ AdjustParams error = %v, want nil", err)
	}

	checkAdjustOutcome(t, gp, records, c)
}

// checkAdjustOutcome compares adjusted values and adjustments, ignoring adjustment order.
func checkAdjustOutcome(t *testing.T, gp params.Values, records []params.Adjustment, c adjustContractCase) {
	t.Helper()

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

// harnessProviderCfg creates a decoded-entry fixture whose API descriptions point at the harness.
func harnessProviderCfg(test testing.TB, base string) catalog.Provider {
	test.Helper()

	return catalog.Provider{
		ID: "fixt", DisplayName: "Fixture", APIKeyEnvVar: "FIXT_KEY",
		Models: []catalog.Model{
			{ID: "fixt-img", Media: media.Image, Params: []params.Definition{{ParamID: "aspect_ratio", FlagID: params.FlagTypeAspect}}},
			{ID: "fixt-vid", Media: media.Video, Params: []params.Definition{{ParamID: "seconds", FlagID: params.FlagTypeDuration, AllowedValues: []string{"4", "8"}}}},
		},
		Config: &catalog.ProviderConfig{
			ImageAPI: &catalog.ImageAPI{
				GenURL:                base + "/gen",
				InputMediaPayloadType: "json", InputMediaProvParam: "image", InputMediaStyle: catalog.InputMediaSingle,
				FallbackExt: ".png",
			},
			VideoAPI: &catalog.VideoAPI{
				AsyncJobsURL: base + "/videos", JobIDField: "id",
				ProgressStatusText: []string{"pending"}, CompletedStatusText: "done", FailedStatusText: []string{"failed"},
				URLContentPath:        "/content",
				InputMediaPayloadType: "json", InputMediaProvParam: "image", InputMediaStyle: catalog.InputMediaSingle,
				FallbackExt: ".mp4", PollInterval: 1, PollTimeout: 5,
			},
		},
	}
}

// harnessSrv serves image bytes and video job responses for Generate tests.
func harnessSrv(t *testing.T, video []byte) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/gen":
			fmt.Fprintf(w, `{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString([]byte("IMG")))
		case r.URL.Path == "/videos" && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"id":"j1"}`)
		case strings.HasSuffix(r.URL.Path, "/content"):
			_, _ = w.Write(video)
		default:
			fmt.Fprint(w, `{"status":"done"}`)
		}
	}))
	t.Cleanup(srv.Close)

	return srv
}

// checkGenerateImage checks that Generate returns one artifact with the decoded image bytes.
func checkGenerateImage(t *testing.T, base string) {
	t.Helper()

	provCfg := harnessProviderCfg(t, base)

	res, err := constructedTestProvider(t, &provCfg).Generate(t.Context(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: provCfg.Identity(), Model: provCfg.Models[0]}, Prompt: params.GetSetIf(true, "p").ValOr(""), Preparation: generation.Preparation{Params: params.Values{params.FlagTypeAspect: "16:9"}}, APIKey: os.Getenv((catalog.ProvModelPair{Provider: provCfg.Identity(), Model: provCfg.Models[0]}).Provider.APIKeyEnvVar)})
	if err != nil {
		t.Fatalf("💣 image run failed: %v", err)
	}

	if len(res.Artifacts) != 1 || string(res.Artifacts[0].Data) != "IMG" {
		t.Errorf("✗ artifacts = %+v, want the one decoded image", res.Artifacts)
	}
}

// checkGenerateVideo checks that Generate returns one temporary file with the served video bytes.
func checkGenerateVideo(t *testing.T, base string, video []byte) {
	t.Helper()

	provCfg := harnessProviderCfg(t, base)

	res, err := constructedTestProvider(t, &provCfg).Generate(t.Context(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: provCfg.Identity(), Model: provCfg.Models[1]}, Prompt: params.GetSetIf(true, "p").ValOr(""), Preparation: generation.Preparation{Params: params.Values{params.FlagTypeDuration: 4}}, APIKey: os.Getenv((catalog.ProvModelPair{Provider: provCfg.Identity(), Model: provCfg.Models[1]}).Provider.APIKeyEnvVar)})
	if err != nil {
		t.Fatalf("💣 video run failed: %v", err)
	}

	if len(res.Artifacts) != 1 || res.Artifacts[0].TmpPath == "" {
		t.Fatalf("💣 artifacts = %+v, want one file-backed video", res.Artifacts)
	}

	saved, rerr := os.ReadFile(res.Artifacts[0].TmpPath)

	t.Cleanup(func() { _ = os.Remove(res.Artifacts[0].TmpPath) })

	if rerr != nil || !bytes.Equal(saved, video) {
		t.Errorf("✗ streamed bytes differ from the served video (%v)", rerr)
	}
}

// checkGenerateMissingAPI checks that NewProvider rejects a missing ImageAPI or VideoAPI with
// ErrProvConfigInvalid and the missing section name.
func checkGenerateMissingAPI(t *testing.T, base string) {
	t.Helper()

	imgOnly := harnessProviderCfg(t, base)
	imgOnly.Config.VideoAPI = nil

	_, err := NewProvider(&imgOnly)
	if err == nil || !errors.Is(err, errs.ErrProvConfigInvalid) || !strings.Contains(err.Error(), "VideoAPI") {
		t.Errorf("✗ err = %v, want the classified missing-description defect", err)
	}

	vidOnly := harnessProviderCfg(t, base)
	vidOnly.Config.ImageAPI = nil

	_, err = NewProvider(&vidOnly)
	if err == nil || !errors.Is(err, errs.ErrProvConfigInvalid) || !strings.Contains(err.Error(), "ImageAPI") {
		t.Errorf("✗ err = %v, want the classified missing-description defect", err)
	}
}

// checkTextGenerationRequest verifies the configured generation fields and route.
func checkTextGenerationRequest(t *testing.T, provCfg catalog.Provider, modelConfig catalog.Model, seedFlagID, negativePromptFlagID params.FlagType, harness *urlArtifactHTTPHarness, generationPath string) {
	t.Helper()
	harness.resetRequestRecording()

	sizeConfig := modelParamConfig(t, modelConfig, params.FlagTypeSize)
	if len(sizeConfig.AllowedValues) == 0 {
		t.Fatal("💣 selected raster model has no configured sizes")
	}

	flagInputs := params.FlagInputs{
		params.FlagTypeImageN: 2,
		seedFlagID:            17,
		params.FlagTypeSize:   sizeConfig.AllowedValues[0],
		negativePromptFlagID:  "avoid lettering",
	}
	imageRun := runImageGeneration(t, provCfg, modelConfig, flagInputs, nil, harness)
	cleanupDownloadedArtifacts(t, imageRun.result.Artifacts)

	if imageRun.err != nil {
		t.Fatalf("💣 text generation failed: %v", imageRun.err)
	}

	countField := modelParamConfig(t, modelConfig, params.FlagTypeImageN).ParamID
	seedField := modelParamConfig(t, modelConfig, seedFlagID).ParamID
	sizeField := sizeConfig.ParamID
	negativePromptField := modelParamConfig(t, modelConfig, negativePromptFlagID).ParamID
	checkRequestFields(t, harness.requestBody, "model", "prompt", countField, seedField, sizeField, negativePromptField)

	if harness.requestPath != generationPath {
		t.Errorf("✗ text request path = %q, want configured generation path %q", harness.requestPath, generationPath)
	}

	if harness.requestBody["model"] != modelConfig.ID || harness.requestBody["prompt"] != "fixture prompt" {
		t.Errorf("✗ text identity fields = %#v", harness.requestBody)
	}

	if harness.requestBody[countField] != float64(imageRun.params[params.FlagTypeImageN].(int)) {
		t.Errorf("✗ image count = %#v, want adjusted value %#v", harness.requestBody[countField], imageRun.params[params.FlagTypeImageN])
	}

	if harness.requestBody[seedField] != float64(imageRun.params[seedFlagID].(int)) || harness.requestBody[sizeField] != imageRun.params[params.FlagTypeSize] {
		t.Errorf("✗ seed or size differs from adjusted values: body=%#v adjusted=%#v", harness.requestBody, imageRun.params)
	}

	if harness.requestBody[negativePromptField] != imageRun.params[negativePromptFlagID] {
		t.Errorf("✗ negative prompt = %#v, want adjusted value %#v", harness.requestBody[negativePromptField], imageRun.params[negativePromptFlagID])
	}

	if _, present := harness.requestBody["response_format"]; present {
		t.Error("✗ text request contains response_format")
	}
}

// checkStrengthTextRequest verifies strength remains on the generation route.
func checkStrengthTextRequest(t *testing.T, provCfg catalog.Provider, modelConfig, negativePromptModel catalog.Model, strengthFlagID, negativePromptFlagID params.FlagType, harness *urlArtifactHTTPHarness, generationPath string) {
	t.Helper()
	harness.resetRequestRecording()

	imageRun := runImageGeneration(t, provCfg, modelConfig, params.FlagInputs{strengthFlagID: 0.5}, nil, harness)
	cleanupDownloadedArtifacts(t, imageRun.result.Artifacts)

	if imageRun.err != nil {
		t.Fatalf("💣 strength text generation failed: %v", imageRun.err)
	}

	strengthField := modelParamConfig(t, modelConfig, strengthFlagID).ParamID
	negativePromptField := modelParamConfig(t, negativePromptModel, negativePromptFlagID).ParamID
	checkRequestFields(t, harness.requestBody, "model", "prompt", strengthField)

	if harness.requestPath != generationPath {
		t.Errorf("✗ strength-only text request path = %q, want %q", harness.requestPath, generationPath)
	}

	if harness.requestBody[strengthField] != imageRun.params[strengthFlagID] {
		t.Errorf("✗ strength = %#v, want adjusted value %#v", harness.requestBody[strengthField], imageRun.params[strengthFlagID])
	}

	if _, present := harness.requestBody[negativePromptField]; present {
		t.Errorf("✗ model without negative-prompt support sent %q", negativePromptField)
	}

	if _, present := harness.requestBody["response_format"]; present {
		t.Error("✗ strength-only text request contains response_format")
	}
}

// checkEditingRequest verifies one input selects the editing route and bare field.
func checkEditingRequest(t *testing.T, provCfg catalog.Provider, modelConfig catalog.Model, strengthFlagID params.FlagType, harness *urlArtifactHTTPHarness, editingPath string) {
	t.Helper()
	harness.resetRequestRecording()

	mediaURL := "https://media.example/edit-source.png"
	mediaInputs := []media.Input{{URL: mediaURL, MIME: "image/png"}}
	imageRun := runImageGeneration(t, provCfg, modelConfig, params.FlagInputs{strengthFlagID: 0.5}, mediaInputs, harness)
	cleanupDownloadedArtifacts(t, imageRun.result.Artifacts)

	if imageRun.err != nil {
		t.Fatalf("💣 image-to-image generation failed: %v", imageRun.err)
	}

	strengthField := modelParamConfig(t, modelConfig, strengthFlagID).ParamID
	mediaField := provCfg.Config.ImageAPI.InputMediaProvParam
	checkRequestFields(t, harness.requestBody, "model", "prompt", mediaField, strengthField)

	if harness.requestPath != editingPath {
		t.Errorf("✗ editing request path = %q, want configured editing path %q", harness.requestPath, editingPath)
	}

	if harness.requestBody[mediaField] != mediaURL {
		t.Errorf("✗ editing media field = %#v, want the bare URL string", harness.requestBody[mediaField])
	}

	if harness.requestBody[strengthField] != imageRun.params[strengthFlagID] {
		t.Errorf("✗ editing strength = %#v, want adjusted value %#v", harness.requestBody[strengthField], imageRun.params[strengthFlagID])
	}
}

// checkStrengthCap verifies the declared maximum caps a larger strength value.
func checkStrengthCap(t *testing.T, generator generation.Generator, modelConfig catalog.Model, strengthFlagID params.FlagType) {
	t.Helper()

	strengthConfig := modelParamConfig(t, modelConfig, strengthFlagID)

	maximumStrength, hasMaximum := strengthConfig.MaxValue.ValIf()
	if !hasMaximum {
		t.Fatal("💣 selected model has no strength maximum")
	}

	preparedGeneration, err := generator.AdjustParams(&modelConfig, params.FlagInputs{strengthFlagID: maximumStrength + 0.4}, nil, nil)
	adjustedParams, changeRecords := preparedGeneration.Params, preparedGeneration.Changes

	if err != nil {
		t.Fatalf("💣 strength adjustment failed: %v", err)
	}

	if adjustedParams[strengthFlagID] != maximumStrength {
		t.Errorf("✗ capped strength = %#v, want declared maximum %#v", adjustedParams[strengthFlagID], maximumStrength)
	}

	if !hasParamChange(t, changeRecords, strengthFlagID, params.ChangeCapped) {
		t.Errorf("✗ strength changes = %+v, want a capped record", changeRecords)
	}
}

// checkVectorAspect checks that AdjustParams preserves the vector aspect ratio without adjustments.
func checkVectorAspect(t *testing.T, generator generation.Generator, modelConfig catalog.Model) {
	t.Helper()

	aspectConfig := modelParamConfig(t, modelConfig, params.FlagTypeAspect)

	const requestedAspect = "16:9"
	if !slices.Contains(aspectConfig.AllowedValues, requestedAspect) {
		t.Fatalf("💣 selected vector model does not declare %q", requestedAspect)
	}

	preparedGeneration, err := generator.AdjustParams(&modelConfig, params.FlagInputs{params.FlagTypeAspect: requestedAspect}, nil, nil)
	adjustedParams, changeRecords := preparedGeneration.Params, preparedGeneration.Changes

	if err != nil {
		t.Fatalf("💣 vector aspect adjustment failed: %v", err)
	}

	if adjustedParams[params.FlagTypeAspect] != requestedAspect {
		t.Errorf("✗ vector aspect = %#v, want unchanged %q", adjustedParams[params.FlagTypeAspect], requestedAspect)
	}

	if len(changeRecords) != 0 {
		t.Errorf("✗ unchanged vector aspect produced changes: %+v", changeRecords)
	}
}

// checkRasterSizeDerivation verifies an indirect raster size input selects a configured member
// without an ignored record.
func checkRasterSizeDerivation(t *testing.T, generator generation.Generator, modelConfig catalog.Model, sourceFlagID params.FlagType, sourceValue string) {
	t.Helper()

	sizeConfig := modelParamConfig(t, modelConfig, params.FlagTypeSize)

	preparedGeneration, err := generator.AdjustParams(&modelConfig, params.FlagInputs{sourceFlagID: sourceValue}, nil, nil)
	adjustedParams, changeRecords := preparedGeneration.Params, preparedGeneration.Changes

	if err != nil {
		t.Fatalf("💣 raster size adjustment failed: %v", err)
	}

	derivedSize, sizePresent := adjustedParams[params.FlagTypeSize].(string)
	if !sizePresent || !slices.Contains(sizeConfig.AllowedValues, derivedSize) {
		t.Errorf("✗ derived raster size = %#v, want a configured member", adjustedParams[params.FlagTypeSize])
	}

	if hasParamChange(t, changeRecords, sourceFlagID, params.ChangeIgnored) {
		t.Errorf("✗ consumed raster input was reported ignored: %+v", changeRecords)
	}
}

// checkSVGResponse verifies a vector URL download preserves SVG bytes and extension.
func checkSVGResponse(t *testing.T, provCfg catalog.Provider, vectorModel catalog.Model, harness *urlArtifactHTTPHarness) {
	t.Helper()
	harness.resetRequestRecording()
	harness.artifactMIME = "image/svg+xml"
	harness.artifactBytes = []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)

	imageRun := runImageGeneration(t, provCfg, vectorModel, params.FlagInputs{params.FlagTypeAspect: "16:9"}, nil, harness)
	cleanupDownloadedArtifacts(t, imageRun.result.Artifacts)

	if imageRun.err != nil {
		t.Fatalf("💣 vector generation failed: %v", imageRun.err)
	}

	if len(imageRun.result.Artifacts) != 1 || imageRun.result.Artifacts[0].TmpPath == "" || imageRun.result.Artifacts[0].FileExt != ".svg" {
		t.Fatalf("💣 vector artifacts = %+v, want one file-backed .svg", imageRun.result.Artifacts)
	}

	downloadedBytes, err := os.ReadFile(imageRun.result.Artifacts[0].TmpPath)
	if err != nil {
		t.Fatalf("💣 read downloaded SVG: %v", err)
	}

	if !bytes.Equal(downloadedBytes, harness.artifactBytes) {
		t.Errorf("✗ downloaded SVG bytes = %q, want %q", downloadedBytes, harness.artifactBytes)
	}

	aspectField := modelParamConfig(t, vectorModel, params.FlagTypeAspect).ParamID
	if harness.requestBody[aspectField] != imageRun.params[params.FlagTypeAspect] {
		t.Errorf("✗ vector provider size = %#v, want adjusted aspect %#v", harness.requestBody[aspectField], imageRun.params[params.FlagTypeAspect])
	}
}

// checkStructuredError verifies a documented message receives server classification.
func checkStructuredError(t *testing.T, provCfg catalog.Provider, rasterModel catalog.Model, harness *urlArtifactHTTPHarness) {
	t.Helper()
	harness.resetRequestRecording()
	harness.responseStatusCode = http.StatusBadRequest
	harness.responseBody = `{"error":{"message":"fixture provider refusal"}}`

	imageRun := runImageGeneration(t, provCfg, rasterModel, nil, nil, harness)
	if !errors.Is(imageRun.err, errs.ErrResponseServer) {
		t.Errorf("✗ structured response error = %v, want the server-message classification", imageRun.err)
	}

	if imageRun.err == nil || !strings.Contains(imageRun.err.Error(), "fixture provider refusal") {
		t.Errorf("✗ structured response error does not render the provider message: %v", imageRun.err)
	}
}

// checkBareError verifies an unstructured body retains status classification only.
func checkBareError(t *testing.T, provCfg catalog.Provider, rasterModel catalog.Model, harness *urlArtifactHTTPHarness) {
	t.Helper()
	harness.resetRequestRecording()
	harness.responseStatusCode = http.StatusBadGateway
	harness.responseBody = "fixture gateway body"

	imageRun := runImageGeneration(t, provCfg, rasterModel, nil, nil, harness)
	if !errors.Is(imageRun.err, errs.ErrResponseStatus) {
		t.Errorf("✗ bare response error = %v, want the response-status classification", imageRun.err)
	}

	if errors.Is(imageRun.err, errs.ErrResponseServer) {
		t.Errorf("✗ bare response body was marked as a server message: %v", imageRun.err)
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
