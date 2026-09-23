package google

// Invariants tested:
//  1. Google generator: Loading the embedded Google configuration must return ID google, display
//     name Google, and environment variable GOOGLE_API_KEY. NewProvider must construct a nonnil
//     generator, and veoFamily must classify the configured models as three Veo models and six
//     Interactions models.
//  2. Google parameter adjustment: For Veo, AdjustParams must snap duration five to four, force
//     duration eight for 1080p or reference images, report capping five images to three, and snap
//     aspect 3:2 to 16:9. It must preserve supplied person-generation values and report unsupported
//     quality as ignored. For Gemini 3.1 Flash Image, it must reject thinking level zz. Every case
//     must return the exact parameter map and complete adjustment records without an error.
//  3. File availability polling settings: Loading the embedded Google configuration must return
//     FilePollInterval=5 and FilePollTimeout=1800, both measured in seconds.
//  4. Image request envelope: For image Interactions, Generate must POST to /interactions with the
//     Google credential and put ordered base64 image blocks before the prompt. It must place
//     resolution 512 and aspect 16:9 in response_format and requested thinking settings in
//     generation_config. A request without optional parameters must omit generation_config and
//     contain only type=image in response_format. Background, store, stream, and unrequested MIME
//     fields must be absent in the checked requests.
//  5. Interaction step traversal: For image Interactions, Generate must return the first inline
//     image as PNG bytes, collect thought summaries in order, and poll a file URI until ACTIVE
//     before downloading its bytes with credentials. An error step must return ErrResponseGen with
//     its code, message, and details and no artifacts, even after an image block. With no image, it
//     must return ErrResponseNoData and no artifacts, using only the first available text as
//     diagnostic detail.
//  6. Thinking-level model constraints: AdjustParams and Generate must send low or high unchanged
//     for Gemini 3 Pro Image and 3.1 Flash Lite Image, and minimal or high unchanged for 3.1 Flash
//     Image. For the tested unsupported levels, they must omit thinking_level, return exactly one
//     Rejected record naming the accepted pair, and still generate one artifact successfully.
//  7. Video task selection: For Omni video, AdjustParams and Generate must omit generation_config
//     with no media or a video input, select image_to_video for one image, and select
//     reference_to_video for two or six images. They must preserve all image bytes, MIME types, and
//     order before the prompt without adjustments. A video input must appear as an inline video
//     block.
//  8. Video envelope: For Omni video with aspect 16:9, Generate must preserve the selected model
//     and prompt and put type=video, delivery=uri, and the aspect in response_format. It must omit
//     background, store, stream, image_size, and mime_type. Without optional parameters,
//     response_format must contain exactly type and delivery.
//  9. Video steps: For video Interactions, Generate must poll a file from PROCESSING to ACTIVE,
//     then download it with credentials and return its exact bytes in a temporary .mp4 file. Inline
//     base64 must return the decoded MP4 bytes without a file. An unidentifiable downloaded format
//     must retain its bytes with the .mp4 fallback extension. A text-only response must return
//     ErrResponseNoData containing that text and no artifacts.
//  10. Flash Lite resolution adjustment: Given resolution 2K for Gemini 3.1 Flash Lite Image,
//      AdjustParams and Generate must succeed, return exactly one Snapped record from 2K to 1K, and
//      send response_format.image_size=1K.
//  11. Gemini 2.5 Flash parameter support: For Gemini 2.5 Flash Image, AdjustGeneration must report
//      exactly resolution, thinking level, and thoughts as ignored. AdjustParams followed by
//      Generate must still send aspect 16:9, omit generation_config and response_format.image_size,
//      and succeed.
//  12. Provider output ownership: Given requested thoughts and an interaction containing a thought
//      summary and image, AdjustParams and Generate must succeed without writing to stdout or
//      stderr. The result must contain one artifact and the supplied thought summary.
//  13. Gemini Omni 1.1 generation: For Gemini Omni 1.1 Flash, AdjustParams and Generate must accept
//      360p, 720p, 1080p, and 4k without adjustments. Each request must send the exact model ID,
//      type=video, aspect 9:16, and the requested resolution. The result must contain one .mp4
//      artifact with the exact inline response bytes.
//  14. Veo HTTP input media: Given an HTTP image input, Generate must download it once without the
//      Google credential, then send its bytes in image.bytesBase64Encoded and omit gcsUri. The Veo
//      generation must succeed.
//  15. Veo envelope: Generate must POST to the selected Veo model's predictLongRunning route with
//      the Google credential. Lite must send one image beside the prompt; the other Veo models must
//      send ordered asset references with their original bytes and MIME types. Supplied aspect,
//      duration, and resolution must use their provider field names. Unrequested resolution must be
//      absent, and a request without media or parameters must omit image, referenceImages, and
//      parameters.
//  16. Veo flow: Generate must complete a Veo job after two credentialed polls and return its
//      inline MP4 bytes. A provider error must stop after one poll with ErrResponseGen and its
//      message; an unfinished job must return ErrTransportTimeout without artifacts within the
//      test's time bound. Inline bytes must take precedence over a URI. Same-origin downloads must
//      carry credentials, and external downloads must omit them. Empty samples must be skipped when
//      a usable video follows; otherwise Generate must return ErrResponseNoData with any filtering
//      reasons. The embedded configuration must declare a 15-second polling interval and
//      1800-second timeout.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestGoogleGenerator verifies invariant #1: Google generator.
//
// What is being tested:
// Loading the embedded Google configuration must return ID google, display name Google, and
// environment variable GOOGLE_API_KEY. NewProvider must construct a nonnil generator, and veoFamily
// must classify the configured models as three Veo models and six Interactions models.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestGoogleGenerator(t *testing.T) {
	provCfg, _, err := decodeGoogleCfg(t)
	if err != nil {
		t.Fatalf("💣 the embedded config failed to decode: %v", err)
	}

	if provCfg.ID != "google" || provCfg.DisplayName != "Google" || provCfg.APIKeyEnvVar != "GOOGLE_API_KEY" {
		t.Errorf("✗ identity = %+v, want google/Google/GOOGLE_API_KEY", provCfg.Identity())
	}

	if constructedTestProvider(t, &provCfg) == nil {
		t.Errorf("✗ the constructor composed no generator over the decoded config")
	}

	veoModels, interactionModels := 0, 0

	for _, model := range provCfg.Models {
		if veoFamily(&model) {
			veoModels++
		} else {
			interactionModels++
		}
	}

	if veoModels != 3 || interactionModels != 6 {
		t.Errorf("✗ family split = %d veo / %d interactions models, want 3/6", veoModels, interactionModels)
	}

	if !t.Failed() {
		t.Log("✓ one google config composes one generator over the 3+6 family models")
	}
}

// TestGoogleAdjustParams verifies invariant #2: Google parameter adjustment.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// What is being tested:
// For Veo, AdjustParams must snap duration five to four, force duration eight for 1080p or
// reference images, report capping five images to three, and snap aspect 3:2 to 16:9. It must
// preserve supplied person-generation values and report unsupported quality as ignored. For Gemini
// 3.1 Flash Image, it must reject thinking level zz. Every case must return the exact parameter map
// and complete adjustment records without an error.
func TestGoogleAdjustParams(t *testing.T) {
	provCfg, _, err := decodeGoogleCfg(t)
	if err != nil {
		t.Fatalf("💣 the embedded config failed to decode: %v", err)
	}

	p := constructedTestProvider(t, &provCfg)

	cases := []adjustContractCase{
		{
			name: "a duration outside the declared set snaps to the nearest member", modelID: "veo-3.1-generate-preview",
			inputs: params.FlagInputs{params.FlagTypeDuration: 5},
			gp:     params.Values{params.FlagTypeDuration: 4},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeDuration, Type: params.ChangeSnapped, InputVal: "5", WireVal: "4"},
			},
		},
		{
			name: "a 1080p resolution forces the duration to 8", modelID: "veo-3.1-generate-preview",
			inputs: params.FlagInputs{params.FlagTypeDuration: 4, params.FlagTypeResolution: "1080p"},
			gp:     params.Values{params.FlagTypeDuration: 8, params.FlagTypeResolution: "1080p"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeDuration, Type: params.ChangeForced, InputVal: "4", WireVal: "8", Comment: "1080p"},
			},
		},
		{
			name: "reference images force an unrequested duration to 8", modelID: "veo-3.1-generate-preview",
			images: 2,
			gp:     params.Values{params.FlagTypeDuration: 8},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeDuration, Type: params.ChangeForced, InputVal: "", WireVal: "8", Comment: ReasonReferenceImages},
			},
		},
		{
			name: "an over-cap reference set decides the cap without re-forcing an 8", modelID: "veo-3.1-generate-preview",
			inputs: params.FlagInputs{params.FlagTypeDuration: 8},
			images: 5,
			gp:     params.Values{params.FlagTypeDuration: 8},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped, InputVal: "5", WireVal: "3", Comment: "max 3"},
			},
		},
		{
			name: "an aspect outside the declared set snaps by ratio", modelID: "veo-3.1-generate-preview",
			inputs: params.FlagInputs{params.FlagTypeAspect: "3:2"},
			gp:     params.Values{params.FlagTypeAspect: "16:9"},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeAspect, Type: params.ChangeSnapped, InputVal: "3:2", WireVal: "16:9"},
			},
		},
		{
			// The person-generation policy is free text the provider validates by
			// request mode, so a text-only request passes it through unchanged.
			name: "a person-generation policy passes through a text-only request", modelID: "veo-3.1-generate-preview",
			inputs: params.FlagInputs{personGenerationFlag: "allow_all"},
			gp:     params.Values{personGenerationFlag: "allow_all"},
		},
		{
			// With an input image the same policy still passes through unchanged; only
			// the duration is forced by the reference image.
			name: "a person-generation policy passes through an image-to-video request", modelID: "veo-3.1-generate-preview",
			inputs: params.FlagInputs{personGenerationFlag: "allow_adult"},
			images: 1,
			gp:     params.Values{personGenerationFlag: "allow_adult", params.FlagTypeDuration: 8},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeDuration, Type: params.ChangeForced, InputVal: "", WireVal: "8", Comment: ReasonReferenceImages},
			},
		},
		{
			name: "an unconsumed supplied flag warns as ignored", modelID: "veo-3.1-generate-preview",
			inputs: params.FlagInputs{params.FlagTypeQuality: "high"},
			gp:     params.Values{},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeQuality, Type: params.ChangeIgnored, Comment: fmt.Sprintf(generation.ReasonNotConsumed, "veo-3.1-generate-preview")},
			},
		},
		{
			name: "an Interactions model takes the shared shaping alone", modelID: "gemini-3.1-flash-image",
			inputs: params.FlagInputs{params.FlagTypeThinkingLevel: "zz"},
			gp:     params.Values{},
			records: []params.Adjustment{
				{FlagID: params.FlagTypeThinkingLevel, Type: params.ChangeRejected, InputVal: "zz", Comment: "minimal|high"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var images []media.Input
			for i := range c.images {
				images = append(images, fixtureInputMedia(t, string(rune('a'+i)), "/tmp/adj.png"))
			}

			model := mustModel(t, c.modelID)

			preparedGeneration, err := p.AdjustParams(&model, c.inputs, images, nil)
			gp, records := preparedGeneration.Params, preparedGeneration.Changes

			if err != nil {
				t.Errorf("✗ AdjustParams error = %v, want nil (google has no fallible adjustment)", err)
			}

			checkAdjustCase(t, gp, records, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the one AdjustParams call returns the adjusted values and the complete record set with a nil error")
	}
}

// TestFilePins verifies invariant #3: File availability polling settings.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// What is being tested:
// Loading the embedded Google configuration must return FilePollInterval=5 and
// FilePollTimeout=1800, both measured in seconds.
func TestFilePins(t *testing.T) {
	_, adapterAPI, err := decodeGoogleCfg(t)
	if err != nil {
		t.Fatalf("💣 the embedded config failed to decode: %v", err)
	}

	if adapterAPI.FilePollInterval != 5 {
		t.Errorf("✗ file poll interval = %d, want 5s", adapterAPI.FilePollInterval)
	}

	if adapterAPI.FilePollTimeout != 1800 {
		t.Errorf("✗ file poll timeout = %d, want 1800s", adapterAPI.FilePollTimeout)
	}

	if !t.Failed() {
		t.Log("✓ the Files probe runs at the 5s pace and 1800s deadline")
	}
}

// TestImgEnvelope verifies invariant #4: Image request envelope.
//
// Test class: Expanded.
// Test layer: Coverage.
//
// What is being tested:
// For image Interactions, Generate must POST to /interactions with the Google credential and put
// ordered base64 image blocks before the prompt. It must place resolution 512 and aspect 16:9 in
// response_format and requested thinking settings in generation_config. A request without optional
// parameters must omit generation_config and contain only type=image in response_format.
// Background, store, stream, and unrequested MIME fields must be absent in the checked requests.
func TestImgEnvelope(t *testing.T) {
	t.Run("blocks are sent image-before-text", envBlocks)
	t.Run("the smallest size token is sent as 512", envSize)
	t.Run("generation config carries its fields", envConfig)
	t.Run("a bare run omits the optional objects", envMinimal)

	if !t.Failed() {
		t.Log("✓ the Interactions request body holds on every leg")
	}
}

// TestSteps verifies invariant #5: Interaction step traversal.
//
// What is being tested:
// For image Interactions, Generate must return the first inline image as PNG bytes, collect thought
// summaries in order, and poll a file URI until ACTIVE before downloading its bytes with
// credentials. An error step must return ErrResponseGen with its code, message, and details and no
// artifacts, even after an image block. With no image, it must return ErrResponseNoData and no
// artifacts, using only the first available text as diagnostic detail.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSteps(t *testing.T) {
	t.Run("the first image block yields the artifact", stepInline)
	t.Run("thought summaries collect in order", stepThoughts)
	t.Run("a uri block routes through the files lifecycle", stepURI)
	t.Run("an error-status step is terminal", stepErrTerminal)
	t.Run("an error outranks content", stepErrOverContent)
	t.Run("unattached text is the no-data diagnostic", stepStray)
	t.Run("an empty traversal is a plain no-data error", stepEmpty)

	if !t.Failed() {
		t.Log("✓ the step traversal holds on every preference and failure leg")
	}
}

// TestLevelGate verifies invariant #6: Thinking-level model constraints.
//
// What is being tested:
// AdjustParams and Generate must send low or high unchanged for Gemini 3 Pro Image and 3.1 Flash
// Lite Image, and minimal or high unchanged for 3.1 Flash Image. For the tested unsupported levels,
// they must omit thinking_level, return exactly one Rejected record naming the accepted pair, and
// still generate one artifact successfully.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLevelGate(t *testing.T) {
	levelCases := []struct {
		id             string
		passes, denies []string
	}{
		{"gemini-3-pro-image", []string{"low", "high"}, []string{"minimal", "medium"}},
		{"gemini-3.1-flash-image", []string{"minimal", "high"}, []string{"low", "medium"}},
		{"gemini-3.1-flash-lite-image", []string{"low", "high"}, []string{"minimal", "medium"}},
	}
	for _, levelCase := range levelCases {
		for _, v := range levelCase.passes {
			t.Run(levelCase.id+" passes "+v, func(t *testing.T) {
				checkLevelPass(t, levelCase.id, v)

				if !t.Failed() {
					t.Logf("✓ %s", levelCase.id+" passes "+v)
				}
			})
		}

		for _, v := range levelCase.denies {
			t.Run(levelCase.id+" rejects "+v, func(t *testing.T) {
				checkLevelReject(t, levelCase.id, v, strings.Join(levelCase.passes, "|"))

				if !t.Failed() {
					t.Logf("✓ %s", levelCase.id+" rejects "+v)
				}
			})
		}
	}

	if !t.Failed() {
		t.Log("\u2713 each thinking-capable model enforces its own accepted pair and rejects the rest")
	}
}

// TestVidTask verifies invariant #7: Video task selection.
//
// What is being tested:
// For Omni video, AdjustParams and Generate must omit generation_config with no media or a video
// input, select image_to_video for one image, and select reference_to_video for two or six images.
// They must preserve all image bytes, MIME types, and order before the prompt without adjustments.
// A video input must appear as an inline video block.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidTask(t *testing.T) {
	t.Run("zero input images omit the task field", vidTask0)
	t.Run("one input image selects image_to_video", vidTask1)
	t.Run("two input images select reference_to_video", vidTask2)
	t.Run("six input images all reach the request body", vidTask6)
	t.Run("one input video sets no task", vidTaskVideo)

	if !t.Failed() {
		t.Log("✓ the task selection follows the enforced input-media count on every leg")
	}
}

// TestVidEnvelope verifies invariant #8: Video envelope.
//
// What is being tested:
// For Omni video with aspect 16:9, Generate must preserve the selected model and prompt and put
// type=video, delivery=uri, and the aspect in response_format. It must omit background, store,
// stream, image_size, and mime_type. Without optional parameters, response_format must contain
// exactly type and delivery.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidEnvelope(t *testing.T) {
	t.Run("aspect is sent in the video format", vidEnvAspect)
	t.Run("a bare run sends the two-key format alone", vidEnvBare)

	if !t.Failed() {
		t.Log("✓ the video response format holds whether or not the aspect is set")
	}
}

// TestVidSteps verifies invariant #9: Video steps.
//
// What is being tested:
// For video Interactions, Generate must poll a file from PROCESSING to ACTIVE, then download it
// with credentials and return its exact bytes in a temporary .mp4 file. Inline base64 must return
// the decoded MP4 bytes without a file. An unidentifiable downloaded format must retain its bytes
// with the .mp4 fallback extension. A text-only response must return ErrResponseNoData containing
// that text and no artifacts.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidSteps(t *testing.T) {
	t.Run("a uri block routes through the files lifecycle", vidStepURI)
	t.Run("a base64 video block decodes", vidStepInline)
	t.Run("an unidentifiable video falls back to .mp4", vidStepFallback)
	t.Run("unattached text is the no-data diagnostic", vidStepStray)

	if !t.Failed() {
		t.Log("✓ the video delivery traversal holds on every leg")
	}
}

// TestLiteSnap verifies invariant #10: Flash Lite resolution adjustment.
//
// What is being tested:
// Given resolution 2K for Gemini 3.1 Flash Lite Image, AdjustParams and Generate must succeed,
// return exactly one Snapped record from 2K to 1K, and send response_format.image_size=1K.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLiteSnap(t *testing.T) {
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, imgStep(t, pngPix(t)))
	cfg := params.FlagInputs{params.FlagTypeResolution: "2K"}

	_, rep, err := seqRun(t, r, "gemini-3.1-flash-lite-image", "make it noir", cfg)
	if err != nil {
		t.Errorf("✗ the adjustment run failed: %v", err)

		return
	}

	want := []params.Adjustment{{FlagID: params.FlagTypeResolution, Type: params.ChangeSnapped, InputVal: "2K", WireVal: "1K"}}
	if !reflect.DeepEqual(rep, want) {
		t.Errorf("✗ records = %+v, want exactly the 2K→1K Snapped record", rep)
	}

	rf := sub(t, jmap(t, r.first(t).body), "response_format")
	wantStr(t, rf, "image_size", "1K")

	if !t.Failed() {
		t.Log("✓ 2K adjusts to the nearest allowed 1K through the declared set and is sent as 1K")
	}
}

// Test25Flash verifies invariant #11: Gemini 2.5 Flash parameter support.
//
// What is being tested:
// For Gemini 2.5 Flash Image, AdjustGeneration must report exactly resolution, thinking level, and
// thoughts as ignored. AdjustParams followed by Generate must still send aspect 16:9, omit
// generation_config and response_format.image_size, and succeed.
//
// Test class: Expanded.
// Test layer: Coverage.
func Test25Flash(t *testing.T) {
	_, model := mustProviderModel(t, "gemini-2.5-flash-image")
	cfg := params.FlagInputs{
		params.FlagTypeAspect:        "16:9",
		params.FlagTypeResolution:    "2K",
		params.FlagTypeThinkingLevel: "high",
		params.FlagTypeThoughts:      true,
	}
	wantIgnored := []params.FlagType{params.FlagTypeResolution, params.FlagTypeThinkingLevel, params.FlagTypeThoughts}
	slices.Sort(wantIgnored)

	prepared, preparationErr := generation.AdjustGeneration(&model, cfg, nil)
	if preparationErr != nil {
		t.Fatalf("💣 preparation failed: %v", preparationErr)
	}

	ignoredRecords := prepared.Changes
	gotIgnored := make([]params.FlagType, 0, len(ignoredRecords))

	for _, record := range ignoredRecords {
		if record.Type != params.ChangeIgnored {
			t.Errorf("✗ non-Ignored record among the ignored-parameter warnings: %+v", record)
		}

		gotIgnored = append(gotIgnored, record.FlagID)
	}

	slices.Sort(gotIgnored)

	if !slices.Equal(gotIgnored, wantIgnored) {
		t.Errorf("✗ ignored parameters = %v, want %v", gotIgnored, wantIgnored)
	}

	r, sc := interSrv(t, nil)

	sc.body = interBody(t, imgStep(t, pngPix(t)))
	if _, _, err := seqRun(t, r, "gemini-2.5-flash-image", "make it noir", cfg); err != nil {
		t.Errorf("✗ the 2.5-flash run failed: %v", err)

		return
	}

	body := jmap(t, r.first(t).body)

	// This model declares no generation-config parameter at all, so the whole object is absent,
	// which satisfies the "sends neither field" invariant more directly than checking the two
	// thinking fields inside it.
	wantAbsent(t, body, "generation_config")

	rf := sub(t, body, "response_format")
	wantStr(t, rf, "aspect_ratio", "16:9")
	wantAbsent(t, rf, "image_size")

	if !t.Failed() {
		t.Log("✓ the 2.5-flash model warns for thinking/resolution and sends neither field")
	}
}

// TestSilent verifies invariant #12: Provider output ownership.
//
// What is being tested:
// Given requested thoughts and an interaction containing a thought summary and image, AdjustParams
// and Generate must succeed without writing to stdout or stderr. The result must contain one
// artifact and the supplied thought summary.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSilent(t *testing.T) {
	r, sc := interSrv(t, nil)
	png := pngPix(t)
	sc.body = interBody(t, thoughtStep(t, "t1", nil), imgStep(t, png))

	t.Setenv("GOOGLE_API_KEY", "hk")
	prov, m := mustProviderModel(t, "gemini-3-pro-image")
	p := harnessInteractionsProvider(t, r.srv.URL, 5*time.Millisecond, time.Second)

	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err1 := os.Pipe()

	errR, errW, err2 := os.Pipe()
	if err1 != nil || err2 != nil {
		t.Fatalf("💣 pipe setup failed: %v / %v", err1, err2)
	}

	os.Stdout, os.Stderr = outW, errW
	preparedGeneration, adjErr := p.AdjustParams(&m, params.FlagInputs{params.FlagTypeThoughts: true}, nil, nil)
	adjusted := preparedGeneration.Params

	res, runErr := p.Generate(context.Background(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: prov, Model: m}, Prompt: params.GetSetIf(true, "make it noir").ValOr(""), Preparation: generation.Preparation{Params: adjusted}, APIKey: os.Getenv((catalog.ProvModelPair{Provider: prov, Model: m}).Provider.APIKeyEnvVar)})
	if adjErr != nil {
		runErr = adjErr
	}

	_ = outW.Close()
	_ = errW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	outB, _ := io.ReadAll(outR)
	errB, _ := io.ReadAll(errR)

	for _, a := range res.Artifacts {
		rmArtifact(t, a)
	}

	if runErr != nil {
		t.Errorf("✗ the harness run failed: %v", runErr)

		return
	}

	if len(outB) != 0 {
		t.Errorf("✗ the provider layer wrote %d bytes to stdout: %q", len(outB), outB)
	}

	if len(errB) != 0 {
		t.Errorf("✗ the provider layer wrote %d bytes to stderr: %q", len(errB), errB)
	}

	if len(res.Artifacts) != 1 || !slices.Equal(res.Thoughts, []string{"t1"}) {
		t.Errorf("✗ the run's facts did not travel through the result: %+v", res)
	}

	if !t.Failed() {
		t.Log("✓ every fact travels through return values; both streams stay empty")
	}
}

// TestOmni11Generation verifies invariant #13: Gemini Omni 1.1 generation.
//
// What is being tested:
// For Gemini Omni 1.1 Flash, AdjustParams and Generate must accept 360p, 720p, 1080p, and 4k
// without adjustments. Each request must send the exact model ID, type=video, aspect 9:16, and the
// requested resolution. The result must contain one .mp4 artifact with the exact inline response
// bytes.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestOmni11Generation(test *testing.T) {
	for _, resolution := range []string{"360p", "720p", "1080p", "4k"} {
		test.Run(resolution, func(test *testing.T) {
			responseServer, responseScript := interSrv(test, nil)
			videoBytes := []byte("omni-video")
			responseScript.body = interBody(test, vidStep(test, videoBytes))

			generationResult, adjustments, generationErr := seqRun(test, responseServer,
				"gemini-omni-1.1-flash", omniVideoPrompt, params.FlagInputs{
					params.FlagTypeAspect: "9:16", params.FlagTypeResolution: resolution,
				})
			if generationErr != nil {
				test.Fatalf("generation failed: %v", generationErr)
			}

			if len(adjustments) != 0 {
				test.Errorf("supported controls were changed: %+v", adjustments)
			}

			requestDocument := jmap(test, responseServer.first(test).body)
			wantStr(test, requestDocument, "model", "gemini-omni-1.1-flash")
			responseFormat := sub(test, requestDocument, "response_format")
			wantStr(test, responseFormat, "type", "video")
			wantStr(test, responseFormat, "aspect_ratio", "9:16")
			wantStr(test, responseFormat, "resolution", resolution)

			if len(generationResult.Artifacts) != 1 ||
				!slices.Equal(generationResult.Artifacts[0].Data, videoBytes) ||
				generationResult.Artifacts[0].FileExt != ".mp4" {
				test.Errorf("video response was not preserved: %+v", generationResult.Artifacts)
			}
		})
	}
}

// TestVeoHTTPInputMedia verifies invariant #14: Veo HTTP input media.
//
// What is being tested:
// Given an HTTP image input, Generate must download it once without the Google credential, then
// send its bytes in image.bytesBase64Encoded and omit gcsUri. The Veo generation must succeed.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVeoHTTPInputMedia(t *testing.T) {
	imageBytes := pngPix(t)
	mediaServer := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBytes)
	})

	operationServer, operationScript := veoOpSrv(t, nil)
	operationScript.polls = []string{doneOp(t, inlineVid(t, []byte("VID")))}
	mediaURL := mediaServer.srv.URL + "/reference.png"
	inputs := []media.Input{{URL: mediaURL, MIME: "image/png"}}

	if _, err := veoCall(t, operationServer, "veo-3.1-lite-generate-preview", params.Values{}, inputs, 5*time.Millisecond, time.Second); err != nil {
		t.Errorf("✗ Veo URL-input run failed: %v", err)

		return
	}

	if len(mediaServer.paths()) != 1 {
		t.Errorf("✗ media download count = %d, want 1", len(mediaServer.paths()))
	} else if mediaServer.first(t).apiKey != "" {
		t.Errorf("✗ media download carried the Google API key")
	}

	startBody := jmap(t, operationServer.first(t).body)
	instance := item(t, arr(t, startBody, "instances"), 0)
	imageObject := sub(t, instance, "image")
	wantStr(t, imageObject, "bytesBase64Encoded", b64s(t, imageBytes))
	wantAbsent(t, imageObject, "gcsUri")

	if !t.Failed() {
		t.Log("✓ Veo downloads HTTP media bare and sends validated inline bytes")
	}
}

// TestVeoEnvelope verifies invariant #15: Veo envelope.
//
// What is being tested:
// Generate must POST to the selected Veo model's predictLongRunning route with the Google
// credential. Lite must send one image beside the prompt; the other Veo models must send ordered
// asset references with their original bytes and MIME types. Supplied aspect, duration, and
// resolution must use their provider field names. Unrequested resolution must be absent, and a
// request without media or parameters must omit image, referenceImages, and parameters.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVeoEnvelope(t *testing.T) {
	t.Run("cap-1 model sends one start image", envCap1)
	t.Run("cap-3 models send asset references", envCap3)
	t.Run("parameters are included only when set", envParams)
	t.Run("a submitted resolution is sent under parameters", envResSent)
	t.Run("a bare run sends no parameters", envBare)

	if !t.Failed() {
		t.Log("✓ the cap-selected request-body conversion holds on every leg")
	}
}

// TestVeoFlow verifies invariant #16: Veo flow.
//
// What is being tested:
// Generate must complete a Veo job after two credentialed polls and return its inline MP4 bytes. A
// provider error must stop after one poll with ErrResponseGen and its message; an unfinished job
// must return ErrTransportTimeout without artifacts within the test's time bound. Inline bytes must
// take precedence over a URI. Same-origin downloads must carry credentials, and external downloads
// must omit them. Empty samples must be skipped when a usable video follows; otherwise Generate
// must return ErrResponseNoData with any filtering reasons. The embedded configuration must declare
// a 15-second polling interval and 1800-second timeout.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVeoFlow(t *testing.T) {
	t.Run("completes after two polls", flowCompletes)
	t.Run("a poll-body error object is terminal", flowPollErr)
	t.Run("deadline wraps the timeout sentinel", flowDeadline)
	t.Run("the production pair is 15s per 1800s", flowPins)
	t.Run("inline bytes are preferred over the uri", flowInlinePref)
	t.Run("a uri sample is fetched directly", flowURIDirect)
	t.Run("an external-host uri sample downloads bare", flowURIExternal)
	t.Run("a nil-video sample fails classified", flowNilVideo)
	t.Run("empty entries are skipped", flowSkipsEmpty)
	t.Run("an all-empty list carries the filter reasons", flowFilterReasons)

	if !t.Failed() {
		t.Log("✓ the operation flow holds on every completion, failure, and delivery leg")
	}
}

// adjustContractCase contains a model ID, inputs, and the exact parameter values and adjustment
// records expected from AdjustParams. Record order is not checked.
type adjustContractCase struct {
	name    string
	modelID string
	inputs  params.FlagInputs
	images  int
	gp      params.Values
	records []params.Adjustment
}

// recordedReq is one recorded harness request.
type recordedReq struct {
	reqMethod, reqPath, apiKey string
	body                       []byte
}

// recServer is a recording test server: it captures every request's method, path (with query),
// x-goog-api-key header value, and body, then serves handle.
type recServer struct {
	test         testing.TB
	mu           sync.Mutex
	recordedReqs []recordedReq
	srv          *httptest.Server
}

// personGenerationFlag identifies the free-text person-generation parameter.
const personGenerationFlag params.FlagType = "person-generation"

// interScript contains the interaction response body. Tests set it after starting the server so the
// body can include the server URL.
type interScript struct{ body string }

// errStep is the error-status model_output step fixture.
const errStep = `{"type":"model_output","status":"error","error":{"code":400,"message":"blocked by policy","details":[{"reason":"safety"}]}}`

// omniVideoPrompt is the prompt every Omni Flash video request in these tests sends.
const omniVideoPrompt = "make it move"

// opScript contains the operation responses. Tests set them after starting the server so the
// responses can include the server URL.
type opScript struct{ polls []string }

// decodeGoogleCfg loads the embedded Google configuration and returns its provider description and
// required AdapterAPI settings.
func decodeGoogleCfg(test testing.TB) (catalog.Provider, catalog.AdapterAPI, error) {
	test.Helper()

	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags, catalog.Source{ProviderID: ProviderID, ConfigBytes: ConfigJSON})
	if err != nil {
		return catalog.Provider{}, catalog.AdapterAPI{}, err
	}

	provCfg, loaded := loadedCatalog.Provider(ProviderID)
	if !loaded {
		return catalog.Provider{}, catalog.AdapterAPI{}, loadedCatalog.ConfigError(ProviderID)
	}

	if provCfg.Config.AdapterAPI == nil {
		return catalog.Provider{}, catalog.AdapterAPI{}, errors.New(AdapterAPIMissing)
	}

	return provCfg, *provCfg.Config.AdapterAPI, nil
}

// countRecords counts the records equal to want exactly.
func countRecords(test testing.TB, records []params.Adjustment, want params.Adjustment) int {
	test.Helper()

	n := 0

	for _, record := range records {
		if record == want {
			n++
		}
	}

	return n
}

// mustModel returns the decoded config's model by id.
func mustModel(t *testing.T, id string) catalog.Model {
	t.Helper()
	_, m := mustProviderModel(t, id)

	return m
}

// checkAdjustCase asserts one contract case's adjusted values and record multiset exactly.
func checkAdjustCase(t *testing.T, gp params.Values, records []params.Adjustment, c adjustContractCase) {
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
		if got, wantN := countRecords(t, records, want), countRecords(t, c.records, want); got != wantN {
			t.Errorf("✗ record %+v appears %d time(s), want %d", want, got, wantN)
		}
	}
}

// pngPix returns a real, decodable one-pixel PNG for inline-result fixtures.
func pngPix(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("💣 PNG fixture encode failed: %v", err)
	}

	return buf.Bytes()
}

// fixtureInputMedia creates one input image carrying distinct marker bytes, so request-body
// assertions can prove per-input placement and order.
func fixtureInputMedia(test testing.TB, marker, path string) media.Input {
	test.Helper()

	return media.Input{Bytes: []byte(marker), MIME: "image/png", Filepath: path}
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
			reqPath:   req.URL.RequestURI(),
			apiKey:    req.Header.Get("X-Goog-Api-Key"),
			body:      body,
		})
		r.mu.Unlock()
		handle(w, req)
	}))
	t.Cleanup(r.srv.Close)

	return r
}

// paths returns the recorded request paths in order.
func (r *recServer) paths() []string {
	r.test.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	ps := make([]string, 0, len(r.recordedReqs))
	for _, h := range r.recordedReqs {
		ps = append(ps, h.reqPath)
	}

	return ps
}

// first returns the first recorded request, failing the test when none happened.
func (r *recServer) first(t *testing.T) recordedReq {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.recordedReqs) == 0 {
		t.Fatal("💣 no request recorded, need the first")
	}

	return r.recordedReqs[0]
}

// keyOf returns the recorded credential header of the first request at path, and whether that path
// was reached at all.
func (r *recServer) keyOf(path string) (string, bool) {
	r.test.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, h := range r.recordedReqs {
		if h.reqPath == path {
			return h.apiKey, true
		}
	}

	return "", false
}

// reqsAt returns every recorded request at path, in order.
func (r *recServer) reqsAt(path string) []recordedReq {
	r.test.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	var out []recordedReq

	for _, h := range r.recordedReqs {
		if h.reqPath == path {
			out = append(out, h)
		}
	}

	return out
}

// wantMethod asserts every recorded request at path used the expected HTTP method.
func wantMethod(t *testing.T, r *recServer, path, method string) {
	t.Helper()

	for _, h := range r.reqsAt(path) {
		if h.reqMethod != method {
			t.Errorf("✗ %s was requested with %s, want %s", path, h.reqMethod, method)
		}
	}
}

// jmap decodes a recorded JSON body into a generic map.
func jmap(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("💣 recorded body is not JSON: %v\n%s", err, body)
	}

	return m
}

// sub returns the object at objKey, failing when absent or not an object.
func sub(t *testing.T, m map[string]any, objKey string) map[string]any {
	t.Helper()

	v, ok := m[objKey].(map[string]any)
	if !ok {
		t.Fatalf("💣 %q is not an object: %v", objKey, m[objKey])
	}

	return v
}

// arr returns the list at objKey, failing when absent or not a list.
func arr(t *testing.T, m map[string]any, objKey string) []any {
	t.Helper()

	v, ok := m[objKey].([]any)
	if !ok {
		t.Fatalf("💣 %q is not a list: %v", objKey, m[objKey])
	}

	return v
}

// item returns list element i as an object, failing when out of range.
func item(t *testing.T, list []any, i int) map[string]any {
	t.Helper()

	if i >= len(list) {
		t.Fatalf("💣 list holds %d elements, need index %d", len(list), i)
	}

	m, ok := list[i].(map[string]any)
	if !ok {
		t.Fatalf("💣 list element %d is not an object: %v", i, list[i])
	}

	return m
}

// wantStr asserts m[objKey] is exactly want.
func wantStr(t *testing.T, m map[string]any, objKey, want string) {
	t.Helper()

	got, ok := m[objKey].(string)
	if !ok || got != want {
		t.Errorf("✗ %q = %v, want %q", objKey, m[objKey], want)
	}
}

// wantAbsent asserts objKey does not appear in m.
func wantAbsent(t *testing.T, m map[string]any, objKey string) {
	t.Helper()

	if _, ok := m[objKey]; ok {
		t.Errorf("✗ %q unexpectedly present: %v", objKey, m[objKey])
	}
}

// mustProviderModel returns the decoded config's identity and its declared model with id.
func mustProviderModel(t *testing.T, id string) (catalog.Provider, catalog.Model) {
	t.Helper()

	provCfg, _, err := decodeGoogleCfg(t)
	if err != nil {
		t.Fatalf("💣 the embedded config failed to decode: %v", err)
	}

	for _, model := range provCfg.Models {
		if model.ID == id {
			return provCfg.Identity(), model
		}
	}

	t.Fatalf("💣 model %q missing", id)

	return catalog.Provider{}, catalog.Model{}
}

// subReq creates a submit-step request over one declared model.
func subReq(t *testing.T, id, prompt string, inputs []media.Input) generation.Generation {
	t.Helper()
	p, m := mustProviderModel(t, id)

	return generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: p, Model: m}, Prompt: params.GetSetIf(true, prompt).ValOr(""), Preparation: generation.Preparation{InputMedia: inputs}, APIKey: os.Getenv((catalog.ProvModelPair{Provider: p, Model: m}).Provider.APIKeyEnvVar)}
}

// rmArtifact removes a file-backed artifact's temp file at cleanup.
func rmArtifact(t *testing.T, generatedMedia artifact.Media) {
	t.Helper()

	if generatedMedia.TmpPath != "" {
		t.Cleanup(func() { _ = os.Remove(generatedMedia.TmpPath) })
	}
}

// fileBacked verifies that an artifact has a temporary file containing the expected bytes and no
// inline data.
func fileBacked(t *testing.T, generatedMedia artifact.Media, content string) {
	t.Helper()
	rmArtifact(t, generatedMedia)

	if len(generatedMedia.Data) != 0 {
		t.Errorf("✗ artifact carries inline bytes beside its file: %d bytes", len(generatedMedia.Data))
	}

	if generatedMedia.TmpPath == "" {
		t.Errorf("✗ artifact carries no file path")

		return
	}

	got, err := os.ReadFile(generatedMedia.TmpPath)
	if err != nil {
		t.Errorf("✗ artifact file unreadable: %v", err)

		return
	}

	if string(got) != content {
		t.Errorf("✗ artifact file holds %q, want %q", got, content)
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

// interSrv records POST /interactions and serves the scripted interaction; any other path continues
// to extra (404 when nil).
func interSrv(t *testing.T, extra http.HandlerFunc) (*recServer, *interScript) {
	t.Helper()

	sc := &interScript{}
	r := newRecServer(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/interactions" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, sc.body)

			return
		}

		if extra != nil {
			extra(w, req)

			return
		}

		http.NotFound(w, req)
	})

	return r, sc
}

// interBody wraps step fixtures into a completed interaction.
func interBody(test testing.TB, steps ...string) string {
	test.Helper()

	return `{"status":"completed","steps":[` + strings.Join(steps, ",") + `]}`
}

// imgStep creates a model_output step carrying one inline PNG image block.
func imgStep(test testing.TB, data []byte) string {
	test.Helper()

	return `{"type":"model_output","content":[{"type":"image","mime_type":"image/png","data":"` + b64s(test, data) + `"}]}`
}

// thoughtStep creates a thought step with the supplied summary text and optional interim image.
func thoughtStep(test testing.TB, text string, interim []byte) string {
	test.Helper()

	blocks := `{"type":"text","text":"` + text + `"}`
	if len(interim) > 0 {
		blocks += `,{"type":"image","mime_type":"image/png","data":"` + b64s(test, interim) + `"}`
	}

	return `{"type":"thought","signature":"sig-` + text + `","summary":[` + blocks + `]}`
}

// harnessInteractionsProvider returns a provider using the local server and file polling limits.
func harnessInteractionsProvider(test testing.TB, base string, pace, wait time.Duration) *Provider {
	test.Helper()

	adapterAPI := catalog.AdapterAPI{APIBase: base, ImageFallbackExt: ".png", VideoFallbackExt: ".mp4"}

	adapterAPI.FilePollInterval = catalog.PollSeconds((pace + time.Second - 1) / time.Second)
	adapterAPI.FilePollTimeout = catalog.PollSeconds((wait+time.Second-1)/time.Second) + adapterAPI.FilePollInterval

	return &Provider{adapterAPI: &adapterAPI}
}

// imgCall drives the Interactions image generation flow against a harness.
func imgCall(t *testing.T, r *recServer, id string, gp params.Values, inputs []media.Input) ([]artifact.Media, []string, error) {
	t.Helper()
	t.Setenv("GOOGLE_API_KEY", "k")
	req := subReq(t, id, "make it noir", inputs)
	req.Params = gp

	res, err := harnessInteractionsProvider(t, r.srv.URL, 5*time.Millisecond, time.Second).Generate(context.Background(), &req)
	for _, a := range res.Artifacts {
		rmArtifact(t, a)
	}

	return res.Artifacts, res.Thoughts, err
}

// seqRun adjusts parameters and generates Interactions media against the local server. It returns
// the generation result and change records, and registers artifact cleanup.
func seqRun(t *testing.T, r *recServer, id, prompt string, cfg params.FlagInputs) (generation.Result, []params.Adjustment, error) {
	t.Helper()
	t.Setenv("GOOGLE_API_KEY", "hk")
	prov, m := mustProviderModel(t, id)
	p := harnessInteractionsProvider(t, r.srv.URL, 5*time.Millisecond, time.Second)

	preparedGeneration, err := p.AdjustParams(&m, cfg, nil, nil)
	adjusted, rep := preparedGeneration.Params, preparedGeneration.Changes

	if err != nil {
		t.Fatalf("💣 AdjustParams failed: %v", err)
	}

	res, err := p.Generate(context.Background(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: prov, Model: m}, Prompt: params.GetSetIf(true, prompt).ValOr(""), Preparation: generation.Preparation{Params: adjusted}, APIKey: os.Getenv((catalog.ProvModelPair{Provider: prov, Model: m}).Provider.APIKeyEnvVar)})
	for _, a := range res.Artifacts {
		rmArtifact(t, a)
	}

	return res, rep, err
}

// imgBody runs one harness generation and returns the recorded create body.
func imgBody(t *testing.T, id string, gp params.Values, inputs []media.Input) map[string]any {
	t.Helper()
	r, sc := interSrv(t, nil)

	sc.body = interBody(t, imgStep(t, pngPix(t)))
	if _, _, err := imgCall(t, r, id, gp, inputs); err != nil {
		t.Errorf("✗ harness run failed: %v", err)
	}

	h := r.first(t)
	if h.reqPath != "/interactions" {
		t.Errorf("✗ create path = %q, want /interactions", h.reqPath)
	}

	if h.reqMethod != http.MethodPost {
		t.Errorf("✗ create method = %s, want POST", h.reqMethod)
	}

	if h.apiKey != "k" {
		t.Errorf("✗ create request missing the x-goog-api-key credential")
	}

	return jmap(t, h.body)
}

// envBlocks verifies that reference-image blocks precede the text block in input order.
func envBlocks(t *testing.T) {
	t.Helper()
	refs := []media.Input{fixtureInputMedia(t, "REF-A", "/tmp/a.png"), {Bytes: []byte("REF-B"), MIME: "image/jpeg", Filepath: "/tmp/b.jpg"}}
	body := imgBody(t, "gemini-3-pro-image", params.Values{}, refs)
	wantStr(t, body, "model", "gemini-3-pro-image")

	for _, key := range []string{"background", "store", "stream"} {
		wantAbsent(t, body, key)
	}

	input := arr(t, body, "input")
	if len(input) != 3 {
		t.Errorf("✗ %d input blocks, want 3 (two images before the text)", len(input))

		return
	}

	first := item(t, input, 0)
	wantStr(t, first, "type", "image")
	wantStr(t, first, "mime_type", "image/png")
	wantStr(t, first, "data", b64s(t, []byte("REF-A")))
	second := item(t, input, 1)
	wantStr(t, second, "mime_type", "image/jpeg")
	wantStr(t, second, "data", b64s(t, []byte("REF-B")))
	text := item(t, input, 2)
	wantStr(t, text, "type", "text")
	wantStr(t, text, "text", "make it noir")

	if !t.Failed() {
		t.Log("✓ one image block per reference precedes the text block, in input order")
	}
}

// envSize verifies that resolution and aspect values enter the image response format.
func envSize(t *testing.T) {
	t.Helper()

	gp := params.Values{params.FlagTypeResolution: "512", params.FlagTypeAspect: "16:9"}
	rf := sub(t, imgBody(t, "gemini-3.1-flash-image", gp, nil), "response_format")
	wantStr(t, rf, "type", "image")
	wantStr(t, rf, "aspect_ratio", "16:9")
	wantStr(t, rf, "image_size", "512")
	wantAbsent(t, rf, "mime_type")

	if !t.Failed() {
		t.Log("✓ a 512 resolution request is sent as the typed image_size \"512\" token")
	}
}

// envConfig verifies that thinking values enter the generation config.
func envConfig(t *testing.T) {
	t.Helper()

	gp := params.Values{params.FlagTypeThinkingLevel: "high", params.FlagTypeThoughts: true}
	gc := sub(t, imgBody(t, "gemini-3-pro-image", gp, nil), "generation_config")

	wantStr(t, gc, "thinking_level", "high")
	wantStr(t, gc, "thinking_summaries", "auto")

	if !t.Failed() {
		t.Log("✓ thinking_level and thinking_summaries are sent as their generation_config fields")
	}
}

// envMinimal verifies that an unconfigured request omits generation config and sends only the
// required image response type.
func envMinimal(t *testing.T) {
	t.Helper()
	body := imgBody(t, "gemini-3-pro-image", params.Values{}, nil)
	wantAbsent(t, body, "generation_config")
	rf := sub(t, body, "response_format")
	wantStr(t, rf, "type", "image")

	if len(rf) != 1 {
		t.Errorf("✗ response_format carries unexpected keys: %v", rf)
	}

	input := arr(t, body, "input")
	if len(input) != 1 {
		t.Errorf("✗ %d input blocks with no references, want the text block alone", len(input))
	}

	if !t.Failed() {
		t.Log("✓ nothing sent means no generation_config and a type-only response_format")
	}
}

// stepsCall serves one interaction body and drives the traversal on the pro model.
func stepsCall(t *testing.T, body string) ([]artifact.Media, []string, error) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = body

	return imgCall(t, r, "gemini-3-pro-image", params.Values{}, nil)
}

// stepInline verifies that the first inline image wins over later image candidates.
func stepInline(t *testing.T) {
	t.Helper()
	png := pngPix(t)
	// Three candidate images across two model_output steps: the first block of the first step
	// must win, so a last-match traversal cannot pass.
	first := `{"type":"model_output","content":[` +
		`{"type":"image","mime_type":"image/png","data":"` + b64s(t, png) + `"},` +
		`{"type":"image","mime_type":"image/png","data":"` + b64s(t, []byte("SECOND-IMG")) + `"}]}`

	artifacts, thoughts, err := stepsCall(t, interBody(t, first, imgStep(t, []byte("THIRD-IMG"))))
	if err != nil {
		t.Errorf("✗ the step traversal failed: %v", err)

		return
	}

	if len(artifacts) != 1 || string(artifacts[0].Data) != string(png) || artifacts[0].TmpPath != "" {
		t.Errorf("✗ artifacts = %+v, want exactly the FIRST inline image decoded", artifacts)
	}

	if len(artifacts) == 1 && artifacts[0].FileExt != ".png" {
		t.Errorf("✗ FileExt = %q, want .png", artifacts[0].FileExt)
	}

	if len(thoughts) != 0 {
		t.Errorf("✗ thoughts = %v, want none", thoughts)
	}

	if !t.Failed() {
		t.Log("✓ the first image block wins over the later candidates")
	}
}

// stepThoughts verifies that thought summaries retain step order and exclude interim images from
// the artifact result.
func stepThoughts(t *testing.T) {
	t.Helper()
	png := pngPix(t)

	artifacts, thoughts, err := stepsCall(t, interBody(t, thoughtStep(t, "t1", png), thoughtStep(t, "t2", nil), imgStep(t, png)))
	if err != nil {
		t.Errorf("✗ the step traversal failed: %v", err)

		return
	}

	if !slices.Equal(thoughts, []string{"t1", "t2"}) {
		t.Errorf("✗ thoughts = %v, want [t1 t2] in step order", thoughts)
	}

	if len(artifacts) != 1 {
		t.Errorf("✗ %d artifacts beside the thoughts, want 1", len(artifacts))
	}

	if !t.Failed() {
		t.Log("✓ thought-step summary texts collect in order; interim image blocks stay out")
	}
}

// stepURI verifies that a same-origin image URI is polled and downloaded with credentials into a
// file-backed artifact.
func stepURI(t *testing.T) {
	t.Helper()

	states := []string{`{"state":"PROCESSING"}`, `{"state":"ACTIVE"}`}
	n := 0
	r, sc := interSrv(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.RequestURI() == "/files/f7:download?alt=media":
			w.Header().Set("Content-Type", "image/png")
			_, _ = io.WriteString(w, "IMG-DL")
		case req.URL.Path == "/files/f7":
			i := n
			if i >= len(states) {
				i = len(states) - 1
			}

			n++
			_, _ = io.WriteString(w, states[i])
		default:
			http.NotFound(w, req)
		}
	})
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"image","mime_type":"image/png","uri":"`+
		r.srv.URL+`/files/f7:download?alt=media"}]}`)

	artifacts, _, err := imgCall(t, r, "gemini-3-pro-image", params.Values{}, nil)
	if err != nil {
		t.Errorf("✗ the step traversal failed: %v", err)

		return
	}

	if len(artifacts) != 1 {
		// Fatal: without exactly one artifact the file inspection below would index past
		// the slice.
		t.Fatalf("💣 %d artifacts, want 1 — cannot inspect the artifact", len(artifacts))
	}

	fileBacked(t, artifacts[0], "IMG-DL")

	want := []string{"/interactions", "/files/f7", "/files/f7", "/files/f7:download?alt=media"}
	if got := r.paths(); !slices.Equal(got, want) {
		t.Errorf("✗ request sequence = %v, want %v", got, want)
	}

	wantMethod(t, r, "/files/f7", http.MethodGet)
	wantMethod(t, r, "/files/f7:download?alt=media", http.MethodGet)
	// The probe and the same-origin download both carry the credential through the composed
	// traversal — a bare leg would 403 in production.
	if key, ok := r.keyOf("/files/f7"); !ok || key != "k" {
		t.Errorf("✗ the file-state probe carried x-goog-api-key %q, want the credential", key)
	}

	if key, ok := r.keyOf("/files/f7:download?alt=media"); !ok || key != "k" {
		t.Errorf("✗ the file download carried x-goog-api-key %q, want the credential", key)
	}

	if !t.Failed() {
		t.Log("✓ the uri normalizes to files/f7, GET-probes PROCESSING→ACTIVE, then GET-downloads, credentialed per leg")
	}
}

// stepErrTerminal verifies that a terminal error step returns its code, message, and details
// through the response-error chain.
func stepErrTerminal(t *testing.T) {
	t.Helper()

	artifacts, _, err := stepsCall(t, interBody(t, errStep))
	if err == nil {
		t.Errorf("✗ the error-status step returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseGen) {
		t.Errorf("✗ error outside the response chain: %v", err)
	}

	for _, want := range []string{"blocked by policy", "400", "safety"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("✗ error missing the %q diagnostic: %v", want, err)
		}
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ an error interaction yielded artifacts: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ the error-status step surfaces code, message, and details under the response chain")
	}
}

// stepErrOverContent verifies that a terminal error step takes precedence over an image candidate
// in the same traversal.
func stepErrOverContent(t *testing.T) {
	t.Helper()

	artifacts, _, err := stepsCall(t, interBody(t, imgStep(t, pngPix(t)), errStep))
	if err == nil {
		t.Errorf("✗ error-plus-content returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseGen) {
		t.Errorf("✗ error outside the response chain: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ the error-status step lost to content: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ the error-status step outranks the image block in the traversal")
	}
}

// stepStray verifies that the first unattached text becomes the diagnostic when no image is
// returned.
func stepStray(t *testing.T) {
	t.Helper()
	// Two unattached texts across two steps: the first must be carried as the diagnostic, so a
	// last-match traversal cannot pass.
	artifacts, _, err := stepsCall(t, interBody(t,
		`{"type":"model_output","content":[{"type":"text","text":"I cannot draw that"}]}`,
		`{"type":"model_output","content":[{"type":"text","text":"a later caption"}]}`))
	if err == nil {
		t.Errorf("✗ a no-image response returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ error outside the no-data chain: %v", err)
	}

	if !strings.Contains(err.Error(), "I cannot draw that") {
		t.Errorf("✗ the no-data error does not carry the first unattached text: %v", err)
	}

	if strings.Contains(err.Error(), "a later caption") {
		t.Errorf("✗ the no-data error carries a later text over the first: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from a no-image response: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ the FIRST unattached text is carried by the no-data error as its diagnostic")
	}
}

// stepEmpty verifies that an interaction with neither image nor text returns a no-data error and no
// artifacts.
func stepEmpty(t *testing.T) {
	t.Helper()

	artifacts, _, err := stepsCall(t, interBody(t))
	if err == nil {
		t.Errorf("✗ an empty interaction returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ error outside the no-data chain: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from an empty interaction: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ no image and no text is a plain no-data error")
	}
}

// levelRun drives one thinking-level value through the provider Generate flow.
func levelRun(t *testing.T, id, v string) (*recServer, generation.Result, []params.Adjustment, error) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, imgStep(t, pngPix(t)))
	cfg := params.FlagInputs{params.FlagTypeThinkingLevel: v}
	res, rep, err := seqRun(t, r, id, "make it noir", cfg)

	return r, res, rep, err
}

// checkLevelReject asserts one out-of-subset level rejects with the exact record while the run
// succeeds with the field omitted.
func checkLevelReject(t *testing.T, id, v, declared string) {
	t.Helper()

	r, res, rep, err := levelRun(t, id, v)
	if err != nil {
		t.Errorf("✗ the rejected-level run failed: %v", err)

		return
	}

	want := []params.Adjustment{{FlagID: params.FlagTypeThinkingLevel, Type: params.ChangeRejected, InputVal: v, Comment: declared}}
	if !reflect.DeepEqual(rep, want) {
		t.Errorf("✗ records = %+v, want exactly the Rejected record naming the model's declared pair", rep)
	}

	body := jmap(t, r.first(t).body)
	if gc, ok := body["generation_config"].(map[string]any); ok {
		wantAbsent(t, gc, "thinking_level")
	}

	if len(res.Artifacts) != 1 {
		t.Errorf("✗ the run did not complete with an artifact")
	}

	if !t.Failed() {
		t.Logf("✓ %s rejects %s with the field omitted and the run succeeding", id, v)
	}
}

// checkLevelPass asserts one in-subset level is sent untouched.
func checkLevelPass(t *testing.T, id, v string) {
	t.Helper()

	r, _, rep, err := levelRun(t, id, v)
	if err != nil {
		t.Errorf("✗ the passing-level run failed: %v", err)

		return
	}

	if len(rep) != 0 {
		t.Errorf("✗ a passing level produced records: %+v", rep)
	}

	gc := sub(t, jmap(t, r.first(t).body), "generation_config")
	wantStr(t, gc, "thinking_level", v)

	if !t.Failed() {
		t.Logf("✓ %s sends %s untouched", id, v)
	}
}

// vidStep creates a model_output step carrying one inline MP4 video block.
func vidStep(test testing.TB, data []byte) string {
	test.Helper()

	return `{"type":"model_output","content":[{"type":"video","mime_type":"video/mp4","data":"` + b64s(test, data) + `"}]}`
}

// vidCall drives the Interactions video generation flow against a harness with the Omni video
// prompt and returns the artifacts.
func vidCall(t *testing.T, r *recServer, gp params.Values, inputs []media.Input) ([]artifact.Media, error) {
	t.Helper()
	t.Setenv("GOOGLE_API_KEY", "k")
	req := subReq(t, "gemini-omni-flash-preview", omniVideoPrompt, inputs)
	req.Params = gp

	res, err := harnessInteractionsProvider(t, r.srv.URL, 5*time.Millisecond, time.Second).Generate(context.Background(), &req)
	for _, a := range res.Artifacts {
		rmArtifact(t, a)
	}

	return res.Artifacts, err
}

// vidSeq adjusts the supplied images and generates an Omni Flash video against the local server. It
// returns the generation result and change records, and registers artifact cleanup.
func vidSeq(t *testing.T, r *recServer, inputs []media.Input) (generation.Result, []params.Adjustment, error) {
	t.Helper()
	t.Setenv("GOOGLE_API_KEY", "hk")
	prov, m := mustProviderModel(t, "gemini-omni-flash-preview")
	p := harnessInteractionsProvider(t, r.srv.URL, 5*time.Millisecond, time.Second)

	preparedGeneration, err := p.AdjustParams(&m, params.FlagInputs{}, inputs, nil)
	rep := preparedGeneration.Changes

	if err != nil {
		t.Fatalf("💣 AdjustParams failed: %v", err)
	}

	res, err := p.Generate(context.Background(), &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: prov, Model: m}, Prompt: params.GetSetIf(true, omniVideoPrompt).ValOr(""), Preparation: preparedGeneration, APIKey: os.Getenv((catalog.ProvModelPair{Provider: prov, Model: m}).Provider.APIKeyEnvVar)})
	for _, a := range res.Artifacts {
		rmArtifact(t, a)
	}

	return res, rep, err
}

// vidTaskBody runs one omni generation and returns the recorded create body and the reported
// pre-submit records.
func vidTaskBody(t *testing.T, inputs []media.Input) (map[string]any, []params.Adjustment) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, vidStep(t, []byte("INLINE-VID")))

	_, rep, err := vidSeq(t, r, inputs)
	if err != nil {
		t.Errorf("✗ harness run failed: %v", err)
	}

	h := r.first(t)
	if h.reqPath != "/interactions" {
		t.Errorf("✗ create path = %q, want /interactions", h.reqPath)
	}

	if h.reqMethod != http.MethodPost {
		t.Errorf("✗ create method = %s, want POST", h.reqMethod)
	}

	return jmap(t, h.body), rep
}

// vidRefs creates n distinct marker input images with alternating MIME types, so the recorded-body
// assertions can prove each block carries its own input image's bytes and MIME, in input order.
func vidRefs(test testing.TB, n int) []media.Input {
	test.Helper()

	mimes := []string{"image/png", "image/jpeg"}

	refs := make([]media.Input, 0, n)
	for i := range n {
		refs = append(refs, media.Input{
			Bytes:    []byte("REF-" + strconv.Itoa(i)),
			MIME:     mimes[i%len(mimes)],
			Filepath: "/tmp/ref-" + strconv.Itoa(i) + ".png",
		})
	}

	return refs
}

// wantTask asserts the recorded body's video_config.task is exactly want.
func wantTask(t *testing.T, body map[string]any, want string) {
	t.Helper()
	vc := sub(t, sub(t, body, "generation_config"), "video_config")
	wantStr(t, vc, "task", want)
}

// wantVidInput verifies that the request contains each image in input order with its MIME type and
// base64 bytes, followed by the Omni video prompt.
func wantVidInput(t *testing.T, body map[string]any, refs []media.Input) {
	t.Helper()

	input := arr(t, body, "input")
	if len(input) != len(refs)+1 {
		t.Errorf("✗ %d input blocks, want %d (the images before the text)", len(input), len(refs)+1)

		return
	}

	for i, ref := range refs {
		blk := item(t, input, i)
		wantStr(t, blk, "type", "image")
		wantStr(t, blk, "mime_type", ref.MIME)
		wantStr(t, blk, "data", b64s(t, ref.Bytes))
	}

	text := item(t, input, len(refs))
	wantStr(t, text, "type", "text")
	wantStr(t, text, "text", omniVideoPrompt)
}

// vidTask0 verifies that a request without input images omits video task configuration.
func vidTask0(t *testing.T) {
	t.Helper()
	body, rep := vidTaskBody(t, nil)
	wantAbsent(t, body, "generation_config")
	wantVidInput(t, body, nil)

	if len(rep) != 0 {
		t.Errorf("✗ a zero-image run reported records: %+v", rep)
	}

	if !t.Failed() {
		t.Log("✓ zero input images send no task field (and no generation_config at all)")
	}
}

// vidTaskVideo verifies that a video input sends no task field and reaches the request as a video
// block.
func vidTaskVideo(t *testing.T) {
	t.Helper()

	clip := []media.Input{{Bytes: []byte("CLIP-BYTES"), MIME: "video/mp4", Filepath: "/tmp/clip.mp4"}}
	body, _ := vidTaskBody(t, clip)
	wantAbsent(t, body, "generation_config")

	input := arr(t, body, "input")
	if len(input) != 2 {
		t.Fatalf("✗ %d input blocks, want the video before the text", len(input))
	}

	block := item(t, input, 0)
	wantStr(t, block, "type", "video")
	wantStr(t, block, "mime_type", "video/mp4")
	wantStr(t, block, "data", b64s(t, clip[0].Bytes))

	if !t.Failed() {
		t.Log("✓ a video input sends no task and reaches the request as a video block")
	}
}

// vidTask1 verifies that one input image selects image-to-video and is transmitted unchanged.
func vidTask1(t *testing.T) {
	t.Helper()
	refs := vidRefs(t, 1)
	body, rep := vidTaskBody(t, refs)
	wantTask(t, body, "image_to_video")
	wantVidInput(t, body, refs)

	if len(rep) != 0 {
		t.Errorf("✗ a one-image run reported records: %+v", rep)
	}

	if !t.Failed() {
		t.Log("✓ one input image selects image_to_video and is sent in the request body verbatim")
	}
}

// vidTask2 verifies that two input images select reference-to-video and retain their distinct
// payloads.
func vidTask2(t *testing.T) {
	t.Helper()
	refs := vidRefs(t, 2)
	body, rep := vidTaskBody(t, refs)
	wantTask(t, body, "reference_to_video")
	wantVidInput(t, body, refs)

	if len(rep) != 0 {
		t.Errorf("✗ a two-image run reported records: %+v", rep)
	}

	if !t.Failed() {
		t.Log("✓ two input images select reference_to_video, each block its own image")
	}
}

// vidTask6 verifies that all six image references reach the request in order without adjustment
// records.
func vidTask6(t *testing.T) {
	t.Helper()
	refs := vidRefs(t, 6)
	body, rep := vidTaskBody(t, refs)
	wantTask(t, body, "reference_to_video")
	wantVidInput(t, body, refs)

	if len(rep) != 0 {
		t.Errorf("✗ a six-image run reported records: %+v", rep)
	}

	if !t.Failed() {
		t.Log("\u2713 all six input images reach the request body with nothing dropped")
	}
}

// vidBody runs one harness video generation through the submit step and returns the recorded create
// body.
func vidBody(t *testing.T, gp params.Values, inputs []media.Input) map[string]any {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, vidStep(t, []byte("INLINE-VID")))

	if _, err := vidCall(t, r, gp, inputs); err != nil {
		t.Errorf("✗ harness run failed: %v", err)
	}

	return jmap(t, r.first(t).body)
}

// vidEnvAspect verifies that a supplied aspect enters the video response format with the required
// type and delivery fields.
func vidEnvAspect(t *testing.T) {
	t.Helper()

	gp := params.Values{params.FlagTypeAspect: "16:9"}
	body := vidBody(t, gp, nil)
	wantStr(t, body, "model", "gemini-omni-flash-preview")
	wantVidInput(t, body, nil)

	for _, key := range []string{"background", "store", "stream"} {
		wantAbsent(t, body, key)
	}

	rf := sub(t, body, "response_format")
	wantStr(t, rf, "type", "video")
	wantStr(t, rf, "delivery", "uri")
	wantStr(t, rf, "aspect_ratio", "16:9")
	wantAbsent(t, rf, "image_size")
	wantAbsent(t, rf, "mime_type")

	if !t.Failed() {
		t.Log("✓ the submitted aspect is sent in the type/delivery/aspect_ratio format")
	}
}

// vidEnvBare verifies that a request without optional values sends only the video type and
// URI-delivery fields.
func vidEnvBare(t *testing.T) {
	t.Helper()
	rf := sub(t, vidBody(t, params.Values{}, nil), "response_format")
	wantStr(t, rf, "type", "video")
	wantStr(t, rf, "delivery", "uri")

	if len(rf) != 2 {
		t.Errorf("✗ response_format carries unexpected keys: %v", rf)
	}

	if !t.Failed() {
		t.Log("✓ nothing sent means exactly the type and delivery keys")
	}
}

// vidFileSrv creates a harness serving the interaction create, the scripted /files/<id> states in
// order, and the download body under ctype.
func vidFileSrv(t *testing.T, id, ctype, content string, states ...string) (*recServer, *interScript) {
	t.Helper()

	n := 0

	return interSrv(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.RequestURI() == "/files/"+id+":download?alt=media":
			w.Header().Set("Content-Type", ctype)
			_, _ = io.WriteString(w, content)
		case req.URL.Path == "/files/"+id:
			i := n
			if i >= len(states) {
				i = len(states) - 1
			}

			n++
			_, _ = io.WriteString(w, states[i])
		default:
			http.NotFound(w, req)
		}
	})
}

// vidStepURI verifies that a same-origin video URI is polled and downloaded with credentials into a
// file-backed artifact.
func vidStepURI(t *testing.T) {
	t.Helper()
	r, sc := vidFileSrv(t, "v7", "video/mp4", "VID-DL",
		`{"state":"PROCESSING"}`, `{"state":"ACTIVE"}`)
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"video","mime_type":"video/mp4","uri":"`+
		r.srv.URL+`/files/v7:download?alt=media"}]}`)

	artifacts, err := vidCall(t, r, params.Values{}, nil)
	if err != nil {
		t.Errorf("✗ the step traversal failed: %v", err)

		return
	}

	if len(artifacts) != 1 {
		// Fatal: without exactly one artifact the file inspection below would index past
		// the slice.
		t.Fatalf("💣 %d artifacts, want 1 — cannot inspect the artifact", len(artifacts))
	}

	fileBacked(t, artifacts[0], "VID-DL")

	if artifacts[0].FileExt != ".mp4" {
		t.Errorf("✗ FileExt = %q, want .mp4", artifacts[0].FileExt)
	}

	want := []string{"/interactions", "/files/v7", "/files/v7", "/files/v7:download?alt=media"}
	if got := r.paths(); !slices.Equal(got, want) {
		t.Errorf("✗ request sequence = %v, want %v", got, want)
	}

	wantMethod(t, r, "/files/v7", http.MethodGet)
	wantMethod(t, r, "/files/v7:download?alt=media", http.MethodGet)

	if key, ok := r.keyOf("/files/v7"); !ok || key != "k" {
		t.Errorf("✗ the file-state probe carried x-goog-api-key %q, want the credential", key)
	}

	if key, ok := r.keyOf("/files/v7:download?alt=media"); !ok || key != "k" {
		t.Errorf("✗ the file download carried x-goog-api-key %q, want the credential", key)
	}

	if !t.Failed() {
		t.Log("✓ the uri normalizes to files/v7, GET-probes PROCESSING→ACTIVE, then GET-downloads file-backed, credentialed per leg")
	}
}

// vidStepInline verifies that an inline video block becomes a bytes-backed video artifact.
func vidStepInline(t *testing.T) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, vidStep(t, []byte("INLINE-VID")))

	artifacts, err := vidCall(t, r, params.Values{}, nil)
	if err != nil {
		t.Errorf("✗ the step traversal failed: %v", err)

		return
	}

	if len(artifacts) != 1 || string(artifacts[0].Data) != "INLINE-VID" || artifacts[0].TmpPath != "" {
		t.Errorf("✗ artifacts = %+v, want exactly the inline video decoded bytes-backed", artifacts)
	}

	if len(artifacts) == 1 && artifacts[0].FileExt != ".mp4" {
		t.Errorf("✗ FileExt = %q, want .mp4", artifacts[0].FileExt)
	}

	if !t.Failed() {
		t.Log("✓ a base64 video block is still decoded, bytes-backed")
	}
}

// vidStepFallback verifies that an otherwise unidentifiable video uses the configured MP4 fallback
// extension while retaining the downloaded bytes.
func vidStepFallback(t *testing.T) {
	t.Helper()
	// A GENUINE video that identifies no extension: testdata/sample.flv is a real FLV file
	// (flv1 stream, 160x120, 0.5 s — ffprobe-verified at creation), a container Go's byte sniff
	// does not recognize (DetectContentType yields application/octet-stream). The block carries
	// no mime_type and the download declares a non-identifying content type: metadata and bytes
	// identify no extension, so the .mp4 fallback must decide — and must not be a relabeling of
	// arbitrary non-video bytes.
	vid, err := os.ReadFile("testdata/sample.flv")
	if err != nil {
		t.Fatalf("💣 fixture read failed: %v", err)
	}

	r, sc := vidFileSrv(t, "v9", "application/octet-stream", string(vid),
		`{"state":"ACTIVE"}`)
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"video","uri":"`+
		r.srv.URL+`/files/v9:download?alt=media"}]}`)

	artifacts, err := vidCall(t, r, params.Values{}, nil)
	if err != nil {
		t.Errorf("✗ the step traversal failed: %v", err)

		return
	}

	if len(artifacts) != 1 {
		// Fatal: without exactly one artifact the file inspection below would index past
		// the slice.
		t.Fatalf("💣 %d artifacts, want 1 — cannot inspect the artifact", len(artifacts))
	}

	fileBacked(t, artifacts[0], string(vid))

	if artifacts[0].FileExt != ".mp4" {
		t.Errorf("✗ FileExt = %q, want exactly .mp4", artifacts[0].FileExt)
	}

	if !t.Failed() {
		t.Log("✓ a genuine unidentifiable video is written file-backed with FileExt exactly .mp4")
	}
}

// vidStepStray verifies that unattached text becomes the diagnostic when no video block is
// returned.
func vidStepStray(t *testing.T) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"text","text":"I cannot animate that"}]}`)

	artifacts, err := vidCall(t, r, params.Values{}, nil)
	if err == nil {
		t.Errorf("✗ a no-video response returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ error outside the no-data chain: %v", err)
	}

	if !strings.Contains(err.Error(), "I cannot animate that") {
		t.Errorf("✗ the no-data error does not carry the unattached text: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from a no-video response: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ no video block plus unattached text is the no-data error carrying the text")
	}
}

// veoOpSrv creates a recording server for Veo starts, operation polls, and an optional additional
// request handler.
func veoOpSrv(t *testing.T, extra http.HandlerFunc) (*recServer, *opScript) {
	t.Helper()

	sc := &opScript{}
	n := 0
	r := newRecServer(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case strings.HasSuffix(req.URL.Path, ":predictLongRunning"):
			_, _ = io.WriteString(w, `{"name":"ops/op1"}`)
		case req.URL.Path == "/ops/op1":
			if len(sc.polls) == 0 {
				http.NotFound(w, req)

				return
			}

			i := n
			if i >= len(sc.polls) {
				i = len(sc.polls) - 1
			}

			n++

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, sc.polls[i])
		case extra != nil:
			extra(w, req)
		default:
			http.NotFound(w, req)
		}
	})

	return r, sc
}

// b64s base64-encodes fixture bytes for the JSON fixtures.
func b64s(test testing.TB, b []byte) string {
	test.Helper()

	return base64.StdEncoding.EncodeToString(b)
}

// doneOp creates a completed operation body around the given samples JSON.
func doneOp(test testing.TB, samples string) string {
	test.Helper()

	return `{"done":true,"response":{"generateVideoResponse":{"generatedSamples":[` + samples + `]}}}`
}

// inlineVid creates one inline video sample.
func inlineVid(test testing.TB, data []byte) string {
	test.Helper()

	return `{"video":{"encodedVideo":"` + b64s(test, data) + `","encoding":"video/mp4"}}`
}

// uriVid creates one URI-delivery video sample.
func uriVid(test testing.TB, url string) string {
	test.Helper()

	return `{"video":{"uri":"` + url + `","encoding":"video/mp4"}}`
}

// harnessVeoProvider returns a provider using the local server and operation polling limits.
func harnessVeoProvider(test testing.TB, base string, pace, wait time.Duration) *Provider {
	test.Helper()

	adapterAPI := catalog.AdapterAPI{APIBase: base, VideoFallbackExt: ".mp4"}

	adapterAPI.PollInterval = catalog.PollSeconds((pace + time.Second - 1) / time.Second)
	adapterAPI.PollTimeout = catalog.PollSeconds((wait + time.Second - 1) / time.Second)

	return &Provider{adapterAPI: &adapterAPI}
}

// veoCall drives the Veo generation flow against a harness.
func veoCall(t *testing.T, r *recServer, id string, gp params.Values, inputs []media.Input, pace, wait time.Duration) ([]artifact.Media, error) {
	t.Helper()
	t.Setenv("GOOGLE_API_KEY", "k")
	req := subReq(t, id, "a red bicycle", inputs)
	req.Params = gp

	res, err := harnessVeoProvider(t, r.srv.URL, pace, wait).Generate(context.Background(), &req)
	for _, a := range res.Artifacts {
		rmArtifact(t, a)
	}

	return res.Artifacts, err
}

// liteCall drives the cap-1 lite model with defaults shared by the flow cases.
func liteCall(t *testing.T, r *recServer) ([]artifact.Media, error) {
	t.Helper()

	return veoCall(t, r, "veo-3.1-lite-generate-preview", params.Values{}, nil, 5*time.Millisecond, time.Second)
}

// pollCount counts recorded operation polls.
func pollCount(test testing.TB, r *recServer) int {
	test.Helper()

	n := 0

	for _, p := range r.paths() {
		if p == "/ops/op1" {
			n++
		}
	}

	return n
}

// veoStart runs one harness generation and returns the recorded start body.
func veoStart(t *testing.T, id string, gp params.Values, inputs []media.Input) map[string]any {
	t.Helper()
	r, sc := veoOpSrv(t, nil)
	sc.polls = []string{doneOp(t, inlineVid(t, []byte("VID")))}

	if _, err := veoCall(t, r, id, gp, inputs, 5*time.Millisecond, time.Second); err != nil {
		t.Errorf("✗ harness run failed: %v", err)
	}

	h := r.first(t)
	if h.reqPath != "/models/"+id+":predictLongRunning" {
		t.Errorf("✗ start path = %q, want /models/%s:predictLongRunning", h.reqPath, id)
	}

	if h.reqMethod != http.MethodPost {
		t.Errorf("✗ start method = %s, want POST", h.reqMethod)
	}

	if h.apiKey != "k" {
		t.Errorf("✗ start request missing the x-goog-api-key credential")
	}

	return jmap(t, h.body)
}

// envCap1 verifies that the one-image model sends its image beside the prompt.
func envCap1(t *testing.T) {
	t.Helper()
	body := veoStart(t, "veo-3.1-lite-generate-preview", params.Values{},
		[]media.Input{fixtureInputMedia(t, "IMG-A", "/tmp/a.png")})
	inst := item(t, arr(t, body, "instances"), 0)
	wantStr(t, inst, "prompt", "a red bicycle")
	img := sub(t, inst, "image")
	wantStr(t, img, "bytesBase64Encoded", b64s(t, []byte("IMG-A")))
	wantStr(t, img, "mimeType", "image/png")
	wantAbsent(t, inst, "referenceImages")

	if !t.Failed() {
		t.Log("✓ the cap-1 model sends image with bytesBase64Encoded beside the prompt")
	}
}

// envCap3 verifies that the three-image models send ordered asset references with their bytes and
// MIME types.
func envCap3(t *testing.T) {
	t.Helper()

	for _, id := range []string{"veo-3.1-generate-preview", "veo-3.1-fast-generate-preview"} {
		body := veoStart(t, id, params.Values{},
			[]media.Input{fixtureInputMedia(t, "IMG-A", "/tmp/a.png"), {Bytes: []byte("IMG-B"), MIME: "image/jpeg", Filepath: "/tmp/b.jpg"}})
		inst := item(t, arr(t, body, "instances"), 0)
		wantStr(t, inst, "prompt", "a red bicycle")
		wantAbsent(t, inst, "image")

		refs := arr(t, inst, "referenceImages")
		if len(refs) != 2 {
			t.Errorf("✗ %s: %d referenceImages, want 2", id, len(refs))

			continue
		}

		wants := []struct{ marker, mime string }{{"IMG-A", "image/png"}, {"IMG-B", "image/jpeg"}}
		for i, w := range wants {
			ref := item(t, refs, i)
			wantStr(t, ref, "referenceType", "asset")
			img := sub(t, ref, "image")
			wantStr(t, img, "bytesBase64Encoded", b64s(t, []byte(w.marker)))
			wantStr(t, img, "mimeType", w.mime)
		}
	}

	if !t.Failed() {
		t.Log("✓ both cap-3 models send ordered asset referenceImages carrying bytes and MIME, with no start image")
	}
}

// envParams verifies that supplied aspect and duration values use their provider parameter names
// while an unset resolution remains absent.
func envParams(t *testing.T) {
	t.Helper()

	gp := params.Values{params.FlagTypeAspect: "16:9", params.FlagTypeDuration: 8}
	body := veoStart(t, "veo-3.1-lite-generate-preview", gp, nil)
	parameterValues := sub(t, body, "parameters")
	wantStr(t, parameterValues, "aspectRatio", "16:9")

	dur, ok := parameterValues["durationSeconds"].(float64)
	if !ok || dur != 8 {
		t.Errorf("✗ durationSeconds = %v (%T), want the integer 8", parameterValues["durationSeconds"], parameterValues["durationSeconds"])
	}

	wantAbsent(t, parameterValues, "resolution")

	if !t.Failed() {
		t.Log("✓ submitted parameters are sent under their provider parameter names; the unset resolution is absent")
	}
}

// envResSent verifies that a supplied resolution enters the request params.
func envResSent(t *testing.T) {
	t.Helper()

	gp := params.Values{params.FlagTypeResolution: "1080p"}
	body := veoStart(t, "veo-3.1-lite-generate-preview", gp, nil)
	parameterValues := sub(t, body, "parameters")
	wantStr(t, parameterValues, "resolution", "1080p")

	if !t.Failed() {
		t.Log("✓ a submitted resolution is sent as params.resolution")
	}
}

// envBare verifies that a request without optional values omits parameters and reference image
// fields.
func envBare(t *testing.T) {
	t.Helper()
	body := veoStart(t, "veo-3.1-lite-generate-preview", params.Values{}, nil)
	wantAbsent(t, body, "parameters")
	inst := item(t, arr(t, body, "instances"), 0)
	wantAbsent(t, inst, "image")
	wantAbsent(t, inst, "referenceImages")

	if !t.Failed() {
		t.Log("✓ nothing sent means no parameters object and no reference-image field")
	}
}

// flowCompletes verifies that a started operation completes through polling and yields its inline
// video sample with credentials on both API requests.
func flowCompletes(t *testing.T) {
	t.Helper()
	r, sc := veoOpSrv(t, nil)
	sc.polls = []string{`{"done":false}`, doneOp(t, inlineVid(t, []byte("VID2")))}

	artifacts, err := veoCall(t, r, "veo-3.1-lite-generate-preview", params.Values{}, nil, 5*time.Millisecond, 2*time.Second)
	if err != nil {
		t.Errorf("✗ flow failed: %v", err)

		return
	}

	if len(artifacts) != 1 || string(artifacts[0].Data) != "VID2" || artifacts[0].FileExt != ".mp4" {
		t.Errorf("✗ artifacts = %+v, want one inline .mp4 carrying VID2", artifacts)
	}

	if got := pollCount(t, r); got != 2 {
		t.Errorf("✗ %d operation polls recorded, want 2", got)
	}

	wantMethod(t, r, "/models/veo-3.1-lite-generate-preview:predictLongRunning", http.MethodPost)
	wantMethod(t, r, "/ops/op1", http.MethodGet)
	// Both provider-API legs carry the credential: a bare start or poll would 401 in
	// production.
	if key, ok := r.keyOf("/models/veo-3.1-lite-generate-preview:predictLongRunning"); !ok || key != "k" {
		t.Errorf("✗ the start carried x-goog-api-key %q, want the credential", key)
	}

	if key, ok := r.keyOf("/ops/op1"); !ok || key != "k" {
		t.Errorf("✗ the operation poll carried x-goog-api-key %q, want the credential", key)
	}

	if !t.Failed() {
		t.Log("✓ the POSTed operation completes on the second credentialed GET poll and yields the inline sample")
	}
}

// flowPollErr verifies that an operation error is terminal and returns no artifacts.
func flowPollErr(t *testing.T) {
	t.Helper()
	r, sc := veoOpSrv(t, nil)
	sc.polls = []string{`{"error":{"code":13,"message":"quota gone"}}`}

	artifacts, err := liteCall(t, r)
	if err == nil {
		t.Errorf("✗ the error operation returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseGen) {
		t.Errorf("✗ error outside the generation-failure chain: %v", err)
	}

	if !strings.Contains(err.Error(), "quota gone") {
		t.Errorf("✗ error does not surface the operation diagnostic: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ a failed operation returned artifacts: %+v", artifacts)
	}

	if got := pollCount(t, r); got != 1 {
		t.Errorf("✗ %d polls after the terminal error, want 1", got)
	}

	if !t.Failed() {
		t.Log("✓ the poll-body error object is terminal and surfaced")
	}
}

// flowDeadline verifies that an operation that never completes stops at the configured timeout
// without artifacts.
func flowDeadline(t *testing.T) {
	t.Helper()
	r, sc := veoOpSrv(t, nil)
	sc.polls = []string{`{"done":false}`}
	start := time.Now()
	artifacts, err := veoCall(t, r, "veo-3.1-lite-generate-preview", params.Values{}, nil, 10*time.Millisecond, 120*time.Millisecond)
	elapsed := time.Since(start)

	if !errors.Is(err, errs.ErrTransportTimeout) {
		t.Errorf("✗ err = %v, want the transport-timeout chain", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ a deadlined flow returned artifacts: %+v", artifacts)
	}

	if elapsed > 1500*time.Millisecond {
		t.Errorf("✗ the 120ms-deadline flow ran %v, past its proportionate bound", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ a never-completing operation deadlines at the configured timeout")
	}
}

// flowPins verifies the operation polling interval and timeout loaded from the built-in Google
// configuration.
func flowPins(t *testing.T) {
	t.Helper()

	_, adapterAPI, err := decodeGoogleCfg(t)
	if err != nil {
		t.Fatalf("💣 the embedded config failed to decode: %v", err)
	}

	if adapterAPI.PollInterval != 15 {
		t.Errorf("✗ operation poll interval = %d, want 15s", adapterAPI.PollInterval)
	}

	if adapterAPI.PollTimeout != 1800 {
		t.Errorf("✗ operation poll timeout = %d, want 1800s", adapterAPI.PollTimeout)
	}

	if !t.Failed() {
		t.Log("✓ the production-wired poll pair is exactly 15s pace / 1800s deadline")
	}
}

// flowInlinePref verifies that inline video bytes take precedence over a URI in the same sample.
func flowInlinePref(t *testing.T) {
	t.Helper()
	r, sc := veoOpSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "NEVER")
	})
	sc.polls = []string{doneOp(t, `{"video":{"encodedVideo":"`+b64s(t, []byte("INL"))+`","encoding":"video/mp4","uri":"`+r.srv.URL+`/never"}}`)}

	artifacts, err := liteCall(t, r)
	if err != nil {
		t.Errorf("✗ flow failed: %v", err)

		return
	}

	if len(artifacts) != 1 || string(artifacts[0].Data) != "INL" || artifacts[0].TmpPath != "" {
		t.Errorf("✗ artifacts = %+v, want the inline bytes with no file", artifacts)
	}

	for _, p := range r.paths() {
		if strings.Contains(p, "/never") {
			t.Errorf("✗ the uri was fetched although inline bytes were present")
		}
	}

	if !t.Failed() {
		t.Log("✓ an inline sample never touches its uri")
	}
}

// flowURIDirect verifies that a completed same-origin sample URI is downloaded directly with
// credentials and without a Files-state probe.
func flowURIDirect(t *testing.T) {
	t.Helper()
	r, sc := veoOpSrv(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/dl/v7" {
			http.NotFound(w, req)

			return
		}

		w.Header().Set("Content-Type", "video/mp4")
		_, _ = io.WriteString(w, "VIA-URI")
	})
	sc.polls = []string{doneOp(t, uriVid(t, r.srv.URL+"/dl/v7"))}

	artifacts, err := liteCall(t, r)
	if err != nil {
		t.Errorf("✗ flow failed: %v", err)

		return
	}

	if len(artifacts) != 1 {
		// Fatal: without exactly one artifact the checks below would index past the slice.
		t.Fatalf("💣 %d artifacts, want 1 — cannot inspect the artifact", len(artifacts))
	}

	fileBacked(t, artifacts[0], "VIA-URI")

	if artifacts[0].FileExt != ".mp4" {
		t.Errorf("✗ FileExt = %q, want .mp4", artifacts[0].FileExt)
	}

	for _, p := range r.paths() {
		if strings.Contains(p, "/files/") {
			t.Errorf("✗ the ready uri went through the file-state probe: %s", p)
		}
	}

	key, hitDl := r.keyOf("/dl/v7")
	if !hitDl {
		t.Errorf("✗ the sample uri was never fetched")
	} else if key != "k" {
		t.Errorf("✗ the same-origin sample download lost its credential (got %q)", key)
	}

	wantMethod(t, r, "/dl/v7", http.MethodGet)

	if !t.Failed() {
		t.Log("✓ a completed sample's uri is GET-fetched directly, credentialed on its origin, with no files probe")
	}
}

// flowURIExternal verifies that Generate downloads an external video URL without sending the Google
// credential and preserves the downloaded bytes.
func flowURIExternal(t *testing.T) {
	t.Helper()
	ext := newRecServer(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/ext/v9" {
			http.NotFound(w, req)

			return
		}

		w.Header().Set("Content-Type", "video/mp4")
		_, _ = io.WriteString(w, "EXT-URI")
	})
	r, sc := veoOpSrv(t, nil)
	sc.polls = []string{doneOp(t, uriVid(t, ext.srv.URL+"/ext/v9"))}

	artifacts, err := liteCall(t, r)
	if err != nil {
		t.Errorf("✗ flow failed: %v", err)

		return
	}

	if len(artifacts) != 1 {
		// Fatal: without exactly one artifact the checks below would index past the slice.
		t.Fatalf("💣 %d artifacts, want 1 — cannot inspect the artifact", len(artifacts))
	}

	fileBacked(t, artifacts[0], "EXT-URI")

	key, hitExt := ext.keyOf("/ext/v9")
	if !hitExt {
		t.Errorf("✗ the external sample uri was never fetched")
	} else if key != "" {
		t.Errorf("✗ the API credential leaked to the foreign origin: x-goog-api-key = %q", key)
	}

	if !t.Failed() {
		t.Log("✓ an external-host sample uri downloads bare — the credential stays on the API origin")
	}
}

// flowNilVideo verifies that a sample without video bytes or a URI returns a classified no-data
// error and no artifacts.
func flowNilVideo(t *testing.T) {
	t.Helper()
	r, sc := veoOpSrv(t, nil)
	sc.polls = []string{doneOp(t, `{"video":{}}`)}

	artifacts, err := liteCall(t, r)
	if err == nil {
		t.Errorf("✗ a nil-video sample returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ error outside the no-data chain: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ a nil-video sample yielded artifacts: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ a video object carrying neither bytes nor a uri fails classified")
	}
}

// flowSkipsEmpty verifies that empty samples are skipped when a later sample contains a video.
func flowSkipsEmpty(t *testing.T) {
	t.Helper()
	r, sc := veoOpSrv(t, nil)
	sc.polls = []string{doneOp(t, `{},`+inlineVid(t, []byte("KEEP")))}

	artifacts, err := liteCall(t, r)
	if err != nil {
		t.Errorf("✗ flow failed: %v", err)

		return
	}

	if len(artifacts) != 1 || string(artifacts[0].Data) != "KEEP" {
		t.Errorf("✗ artifacts = %+v, want the one non-empty sample", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ samples carrying no video are skipped")
	}
}

// flowFilterReasons verifies that an all-empty sample list returns its filtering reasons through
// the no-data error.
func flowFilterReasons(t *testing.T) {
	t.Helper()
	r, sc := veoOpSrv(t, nil)
	sc.polls = []string{`{"done":true,"response":{"generateVideoResponse":{"generatedSamples":[{}],"raiMediaFilteredReasons":["blocked: safety category X"]}}}`}

	artifacts, err := liteCall(t, r)
	if err == nil {
		t.Errorf("✗ an all-empty samples list returned no error")

		return
	}

	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ error outside the no-data chain: %v", err)
	}

	if !strings.Contains(err.Error(), "blocked: safety category X") {
		t.Errorf("✗ the no-data error does not carry the filter reasons: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from an all-empty list: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ the no-data error surfaces raiMediaFilteredReasons")
	}
}
