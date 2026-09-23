package google

// Invariants tested:
//  1. Request field ownership: When an Interactions parameter maps to a required field, its parent,
//     or its descendant, NewProvider must return an error matching ErrProvConfig. The cases cover
//     model, input, response type and delivery, thinking summaries, and the video task.
//  2. Interaction response failures: For image Interactions, Generate must classify HTTP 400 as
//     ErrResponseStatus and preserve the provider message, and classify an unreachable host under
//     ErrTransport. Malformed JSON, an unusable file URI, or invalid base64 must return
//     ErrResponseDecode with the checked model or URI context. Unusable steps must return
//     ErrResponseNoData. A FAILED file must return ErrResponseGen naming the resource. Malformed
//     JSON, unusable steps, and FAILED files must return no artifacts.
//  3. Video response failures: For video Interactions, Generate must return ErrResponseGen naming a
//     FAILED file, ErrResponseNoData for a block without data or URI or an image-only response, and
//     ErrResponseDecode naming invalid inline data or an unusable URI. An interrupted download must
//     return ErrTransportDownload and leave no new temporary download. FAILED files, missing video
//     content, and interrupted downloads must return no artifacts.
//  4. Conflicting configured paths: Given equal parameter paths or one path inside another,
//     catalog.LoadCatalog must exclude the Google provider from selection. ConfigError must match
//     ErrProvConfigInvalid and name both conflicting paths.
//  5. Veo response atomicity: After a successful first video download, Generate must return no
//     artifacts and remove all new temporary downloads when the second sample fails. HTTP 404 must
//     return ErrTransportStatus, an interrupted download must return ErrTransportDownload, and
//     invalid inline base64 must return ErrResponseDecode.
//  6. Veo request and polling failures: For Veo, Generate must return ErrResponseNoData naming the
//     model when creation omits the operation name, ErrResponseStatus with the message for HTTP 400
//     creation, and ErrTransport for an unreachable host. Polling HTTP 500 must stop after one poll
//     with ErrResponseStatus. Repeatedly dropped polls must end with ErrTransportTimeout that
//     retains ErrTransportRequest.
//  7. Malformed Veo responses: When Veo creation returns malformed JSON, Generate must return
//     ErrResponseDecode naming the model and no artifacts. When polling returns malformed JSON, it
//     must return ErrResponseDecode and no artifacts after one poll, within 1.5 seconds.
//  8. Veo input media conflicts: Given mixed image and video inputs, two videos, or a closing frame
//     without an opening frame, validateVeoInputMedia must return an error matching ErrInputMedia.
//  9. Required request fields: For arbitrary quality parameter paths, catalog.LoadCatalog must
//     attach an error to any unavailable provider. If validation and NewProvider accept the
//     mapping, interactionImageBody must either fail with a nil body or preserve the selected
//     model, supplied prompt, response type image, and thinking_summaries=auto.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestRequiredRequestPaths verifies invariant #1: Request field ownership.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// When an Interactions parameter maps to a required field, its parent, or its descendant,
// NewProvider must return an error matching ErrProvConfig. The cases cover model, input, response
// type and delivery, thinking summaries, and the video task.
func TestRequiredRequestPaths(t *testing.T) {
	for _, requestPath := range []string{
		"model", "model.name", "input", "input.content", "response_format",
		"response_format.type", "response_format.type.name", "response_format.delivery",
		"generation_config", "generation_config.thinking_summaries",
		"generation_config.thinking_summaries.mode", "generation_config.video_config",
		"generation_config.video_config.task", "generation_config.video_config.task.name",
	} {
		t.Run(requestPath, func(t *testing.T) {
			providerDescription, _, err := decodeGoogleCfg(t)
			if err != nil {
				t.Fatalf("💣 configuration setup: %v", err)
			}

			for modelIndex := range providerDescription.Models {
				model := &providerDescription.Models[modelIndex]
				if model.Family == "veo" {
					continue
				}

				model.Params = append(model.Params, params.Definition{FlagID: params.FlagTypeQuality, ParamID: requestPath})
			}

			_, err = NewProvider(&providerDescription)
			if !errors.Is(err, errs.ErrProvConfig) {
				t.Errorf("✗ constructor accepts required field conflict %q: %v", requestPath, err)
			}

			if !t.Failed() {
				t.Log("✓ conflicting mapping fails before generation")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ required Interactions fields cannot be displaced by configured parameters")
	}
}

// TestInterEdges verifies invariant #2: Interaction response failures.
//
// What is being tested:
// For image Interactions, Generate must classify HTTP 400 as ErrResponseStatus and preserve the
// provider message, and classify an unreachable host under ErrTransport. Malformed JSON, an
// unusable file URI, or invalid base64 must return ErrResponseDecode with the checked model or URI
// context. Unusable steps must return ErrResponseNoData. A FAILED file must return ErrResponseGen
// naming the resource. Malformed JSON, unusable steps, and FAILED files must return no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestInterEdges(t *testing.T) {
	t.Run("non-2xx create", interBadCreate)
	t.Run("unreachable host", interNoHost)
	t.Run("undecodable body", interBadBody)
	t.Run("malformed steps", interJunkSteps)
	t.Run("unextractable uri", interBadURI)
	t.Run("undecodable image data", interBadData)
	t.Run("failed file through the traversal", interFileFailed)

	if !t.Failed() {
		t.Log("✓ every create and traversal edge fails classified")
	}
}

// TestVidEdges verifies invariant #3: Video response failures.
//
// What is being tested:
// For video Interactions, Generate must return ErrResponseGen naming a FAILED file,
// ErrResponseNoData for a block without data or URI or an image-only response, and
// ErrResponseDecode naming invalid inline data or an unusable URI. An interrupted download must
// return ErrTransportDownload and leave no new temporary download. FAILED files, missing video
// content, and interrupted downloads must return no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVidEdges(t *testing.T) {
	t.Run("failed file through the traversal", vidFileFailed)
	t.Run("a video block with neither data nor uri falls to no-data", vidNoCarriage)
	t.Run("an image block never satisfies the video traversal", vidImgBlock)
	t.Run("undecodable video data", vidBadData)
	t.Run("unextractable uri", vidBadURI)
	t.Run("dropped download cleans up", vidDropDl)

	if !t.Failed() {
		t.Log("✓ every video-path edge fails classified with no leftovers")
	}
}

// TestConflictingConfiguredPaths verifies invariant #4: Conflicting configured paths.
//
// What is being tested:
// Given equal parameter paths or one path inside another, catalog.LoadCatalog must exclude the
// Google provider from selection. ConfigError must match ErrProvConfigInvalid and name both
// conflicting paths.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestConflictingConfiguredPaths(t *testing.T) {
	cases := []struct {
		name   string
		params []params.Definition
	}{
		{
			name: "two parameters declaring one path",
			params: []params.Definition{
				{FlagID: params.FlagTypeQuality, ParamID: "response_format.mime_type"},
				{FlagID: params.FlagTypeOutputFormat, ParamID: "response_format.mime_type"},
			},
		},
		{
			name: "a parameter declaring a path inside another's",
			params: []params.Definition{
				{FlagID: params.FlagTypeQuality, ParamID: "response_format.mime_type"},
				{FlagID: params.FlagTypeOutputFormat, ParamID: "response_format.mime_type.subtype"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			providerDescription, _, err := decodeGoogleCfg(t)
			if err != nil {
				t.Fatalf("💣 setup failed: %v", err)
			}

			providerDescription.Models[0].Params = c.params

			encoded, err := json.Marshal(providerDescription)
			if err != nil {
				t.Fatalf("💣 setup failed: %v", err)
			}

			loaded, err := catalog.LoadCatalog(params.Flags(), catalog.Source{ProviderID: ProviderID, ConfigBytes: encoded})
			if err != nil {
				t.Fatalf("💣 setup failed: %v", err)
			}

			configurationErr := loaded.ConfigError(ProviderID)
			if !errors.Is(configurationErr, errs.ErrProvConfigInvalid) {
				t.Errorf("✗ error=%v; want the conflicting configured paths rejected", configurationErr)
			}

			if _, available := loaded.Provider(ProviderID); available {
				t.Error("✗ a conflicting provider remains available for generation")
			}

			for _, definition := range c.params {
				if configurationErr == nil || !strings.Contains(configurationErr.Error(), definition.ParamID) {
					t.Errorf("✗ error=%v omits conflicting path %q", configurationErr, definition.ParamID)
				}
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ conflicting configured paths prevent provider selection")
	}
}

// TestVeoAtomic verifies invariant #5: Veo response atomicity.
//
// What is being tested:
// After a successful first video download, Generate must return no artifacts and remove all new
// temporary downloads when the second sample fails. HTTP 404 must return ErrTransportStatus, an
// interrupted download must return ErrTransportDownload, and invalid inline base64 must return
// ErrResponseDecode.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVeoAtomic(t *testing.T) {
	cases := []atomCase{
		{name: "second download non-2xx", serveB: serveGone(t), want: errs.ErrTransportStatus},
		{name: "second download drops mid-body", serveB: serveDrop(t), want: errs.ErrTransportDownload},
		{
			name:   "second sample decode fails",
			second: `{"video":{"encodedVideo":"!!!not-base64!!!","encoding":"video/mp4"}}`,
			want:   errs.ErrResponseDecode,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkAtomic(t, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ sample normalization stays atomic across every failure variant")
	}
}

// TestVeoEdges verifies invariant #6: Veo request and polling failures.
//
// What is being tested:
// For Veo, Generate must return ErrResponseNoData naming the model when creation omits the
// operation name, ErrResponseStatus with the message for HTTP 400 creation, and ErrTransport for an
// unreachable host. Polling HTTP 500 must stop after one poll with ErrResponseStatus. Repeatedly
// dropped polls must end with ErrTransportTimeout that retains ErrTransportRequest.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVeoEdges(t *testing.T) {
	t.Run("missing operation name", edgeNoName)
	t.Run("non-2xx start", edgeBadStart)
	t.Run("unreachable host", edgeNoHost)
	t.Run("non-2xx poll is terminal", edgeBadPoll)
	t.Run("dropped polls retry to the deadline", edgeDropPoll)

	if !t.Failed() {
		t.Log("✓ every start and poll edge fails classified")
	}
}

// TestVeoBadJSON verifies invariant #7: Malformed Veo responses.
//
// What is being tested:
// When Veo creation returns malformed JSON, Generate must return ErrResponseDecode naming the model
// and no artifacts. When polling returns malformed JSON, it must return ErrResponseDecode and no
// artifacts after one poll, within 1.5 seconds.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVeoBadJSON(t *testing.T) {
	t.Run("malformed start body", func(t *testing.T) {
		r := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{not json`)
		})

		artifacts, err := liteCall(t, r)
		if !errors.Is(err, errs.ErrResponseDecode) {
			t.Errorf("✗ err = %v, want the decode chain", err)
		}

		if err != nil && !strings.Contains(err.Error(), "veo-3.1-lite-generate-preview") {
			t.Errorf("✗ error does not name the model: %v", err)
		}

		if len(artifacts) != 0 {
			t.Errorf("✗ artifacts from a malformed start: %+v", artifacts)
		}

		if !t.Failed() {
			t.Log("✓ a malformed start body is a classified decode error")
		}
	})
	t.Run("malformed poll body is terminal", func(t *testing.T) {
		r, sc := veoOpSrv(t, nil)
		sc.polls = []string{`{not json`}
		start := time.Now()
		artifacts, err := veoCall(t, r, "veo-3.1-lite-generate-preview", params.Values{}, nil, 10*time.Millisecond, 2*time.Second)
		elapsed := time.Since(start)

		if !errors.Is(err, errs.ErrResponseDecode) {
			t.Errorf("✗ err = %v, want the decode chain, not a retry toward the deadline", err)
		}

		if got := pollCount(t, r); got != 1 {
			t.Errorf("✗ %d polls on malformed JSON, want 1 (terminal)", got)
		}

		if elapsed > 1500*time.Millisecond {
			t.Errorf("✗ the malformed poll ran %v, want a prompt terminal return", elapsed)
		}

		if len(artifacts) != 0 {
			t.Errorf("✗ artifacts from a malformed poll: %+v", artifacts)
		}

		if !t.Failed() {
			t.Log("✓ a malformed poll body returns promptly as a classified decode error")
		}
	})

	if !t.Failed() {
		t.Log("✓ malformed 2xx bodies are terminal classified decode errors at both legs")
	}
}

// TestVeoInputMediaConflicts verifies invariant #8: Veo input media conflicts.
//
// What is being tested:
// Given mixed image and video inputs, two videos, or a closing frame without an opening frame,
// validateVeoInputMedia must return an error matching ErrInputMedia.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestVeoInputMediaConflicts(t *testing.T) {
	tests := []struct {
		name   string
		inputs []media.Input
	}{
		{
			name: "mixed image and video",
			inputs: []media.Input{
				{URL: "https://media.example/open.png", MIME: "image/png"},
				{URL: "https://media.example/source.mp4", MIME: "video/mp4"},
			},
		},
		{
			name: "two input videos",
			inputs: []media.Input{
				{URL: "https://media.example/one.mp4", MIME: "video/mp4"},
				{URL: "https://media.example/two.mp4", MIME: "video/mp4"},
			},
		},
		{
			name:   "closing frame without opening frame",
			inputs: []media.Input{{URL: "https://media.example/close.png", MIME: "image/png", FrameAnchor: media.FrameLast}},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validateVeoInputMedia(testCase.inputs); err == nil || !errors.Is(err, errs.ErrInputMedia) {
				t.Errorf("✗ validation error = %v, want input-media rejection", err)
			}
		})
	}
}

// FuzzRequiredRequestFields verifies invariant #9: Required request fields.
// What is being tested:
// For arbitrary quality parameter paths, catalog.LoadCatalog must attach an error to any
// unavailable provider. If validation and NewProvider accept the mapping, interactionImageBody must
// either fail with a nil body or preserve the selected model, supplied prompt, response type image,
// and thinking_summaries=auto.
// Test class: Expanded.
// Test layer: Fuzzing.
// Kind: permanent.
func FuzzRequiredRequestFields(f *testing.F) {
	for _, path := range []string{"model", "model.name", "input", "input.value", "response_format", "response_format.type", "response_format.type.name", "response_format.quality", "generation_config", "generation_config.thinking_summaries", "generation_config.thinking_level", "generation_config.video_config.task", "quality", ""} {
		f.Add(path)
	}

	f.Fuzz(func(t *testing.T, requestPath string) {
		description, _, err := decodeGoogleCfg(t)
		if err != nil {
			t.Fatalf("💣 setup failed: %v", err)
		}

		var model catalog.Model

		for modelIndex := range description.Models {
			if description.Models[modelIndex].Media == media.Image {
				model = description.Models[modelIndex]

				break
			}
		}

		if model.ID == "" {
			t.Fatal("💣 setup has no configured image model for the Interactions mapping check")
		}

		model.Params = params.Definitions{{FlagID: params.FlagTypeQuality, ParamID: requestPath}}
		description.Models = []catalog.Model{model}
		description.DefaultModel = model.ID

		encoded, err := json.Marshal(description)
		if err != nil {
			t.Fatalf("💣 setup failed: %v", err)
		}

		loaded, err := catalog.LoadCatalog(params.Flags(), catalog.Source{ProviderID: ProviderID, ConfigBytes: encoded})
		if err != nil {
			t.Fatalf("💣 setup failed: %v", err)
		}

		validated, available := loaded.Provider(ProviderID)
		if !available {
			if loaded.ConfigError(ProviderID) == nil {
				t.Error("✗ unavailable provider has no configuration error")
			}

			return
		}

		if _, err := NewProvider(&validated); err != nil {
			return
		}

		body, err := interactionImageBody(&validated.Models[0], "required prompt", params.Values{params.FlagTypeQuality: "wire-quality", params.FlagTypeThoughts: true}, nil)
		if err != nil {
			if body != nil {
				t.Errorf("✗ failed composition returned a request: %v", body)
			}

			return
		}

		if body["model"] != model.ID {
			t.Errorf("✗ mapping %q displaced model: %v", requestPath, body)
		}

		inputs, inputPresent := body["input"].([]map[string]any)
		if !inputPresent || len(inputs) != 1 || inputs[0]["text"] != "required prompt" {
			t.Errorf("✗ mapping %q displaced prompt input: %v", requestPath, body)
		}

		responseFormat, formatPresent := body["response_format"].(map[string]any)
		if !formatPresent || responseFormat["type"] != "image" {
			t.Errorf("✗ mapping %q displaced response type: %v", requestPath, body)
		}

		generationConfig, configPresent := body["generation_config"].(map[string]any)
		if !configPresent || generationConfig["thinking_summaries"] != "auto" {
			t.Errorf("✗ mapping %q displaced thoughts: %v", requestPath, body)
		}

		if !t.Failed() {
			t.Log("✓ accepted mapping preserves required request fields")
		}
	})
}

// atomCase is one atomicity scenario: the second sample either downloads from /dl/b through serveB
// or is the literal sample JSON.
type atomCase struct {
	name   string
	second string
	serveB http.HandlerFunc
	want   error
}

// ownTempDir gives the test its own download directory so cleanup assertions cannot see files
// created by other packages.
func ownTempDir(t *testing.T) {
	t.Helper()
	t.Setenv("TMPDIR", t.TempDir())
}

// dlNames snapshots the bild-dl-* files currently in the OS temp dir.
func dlNames(t *testing.T) map[string]bool {
	t.Helper()

	names, err := filepath.Glob(filepath.Join(os.TempDir(), "bild-dl-*"))
	if err != nil {
		t.Fatalf("💣 temp-dir glob failed: %v", err)
	}

	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}

	return m
}

// noNewDl asserts no bild-dl-* file beyond the before set remains.
func noNewDl(t *testing.T, before map[string]bool) {
	t.Helper()

	for n := range dlNames(t) {
		if !before[n] {
			t.Errorf("✗ leftover download temp file: %s", n)
		}
	}
}

// interBadCreate checks that a rejected image request preserves the provider message.
func interBadCreate(t *testing.T) {
	t.Helper()
	r := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"nope"}}`)
	})

	_, _, err := imgCall(t, r, "gemini-3-pro-image", params.Values{}, nil)
	if !errors.Is(err, errs.ErrResponseStatus) {
		t.Errorf("✗ err = %v, want the error-status chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "nope") {
		t.Errorf("✗ error does not surface the provider message: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a non-2xx create surfaces the provider message under the status chain")
	}
}

// interNoHost checks transport classification for an unreachable Interactions endpoint.
func interNoHost(t *testing.T) {
	t.Helper()
	t.Setenv("GOOGLE_API_KEY", "k")
	req := subReq(t, "gemini-3-pro-image", "p", nil)

	_, err := harnessInteractionsProvider(t, "http://127.0.0.1:1", time.Millisecond, 50*time.Millisecond).Generate(context.Background(), &req)
	if !errors.Is(err, errs.ErrTransport) {
		t.Errorf("✗ err = %v, want the transport chain", err)
	}

	if !t.Failed() {
		t.Log("✓ an unreachable host fails classified in the transport chain")
	}
}

// interBadBody checks that invalid interaction JSON returns a model-specific decode error and no
// artifacts.
func interBadBody(t *testing.T) {
	t.Helper()
	// Malformed 2xx JSON is a classified decode error with diagnostic context, never a silent
	// empty traversal.
	artifacts, _, err := stepsCall(t, `not json at all`)
	if !errors.Is(err, errs.ErrResponseDecode) {
		t.Errorf("✗ err = %v, want the decode chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "gemini-3-pro-image") {
		t.Errorf("✗ error does not name the model: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from an undecodable body: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ an undecodable interaction is a classified decode error")
	}
}

// interJunkSteps checks that unusable response blocks return no data and no artifacts.
func interJunkSteps(t *testing.T) {
	t.Helper()

	artifacts, _, err := stepsCall(t,
		`{"steps":[{"type":"weird"},{"type":"model_output","content":[{"type":"blob"},{"type":"image"}]},{"type":"thought"}]}`)
	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ err = %v, want the no-data chain", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from malformed steps: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ malformed steps and blocks carrying neither data nor a uri fall through to no-data")
	}
}

// interBadURI checks that an unusable image URI returns a decode error naming the reference.
func interBadURI(t *testing.T) {
	t.Helper()

	_, _, err := stepsCall(t, interBody(t, `{"type":"model_output","content":[{"type":"image","mime_type":"image/png","uri":"https://example.com/no-file-here"}]}`))
	if !errors.Is(err, errs.ErrResponseDecode) {
		t.Errorf("✗ err = %v, want the decode chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "no-file-here") {
		t.Errorf("✗ error does not name the uri: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ an unextractable uri is a classified decode error naming the reference")
	}
}

// interBadData checks decode classification for invalid inline image data.
func interBadData(t *testing.T) {
	t.Helper()

	_, _, err := stepsCall(t, interBody(t, `{"type":"model_output","content":[{"type":"image","mime_type":"image/png","data":"!!!not-base64!!!"}]}`))
	if !errors.Is(err, errs.ErrResponseDecode) {
		t.Errorf("✗ err = %v, want the decode chain", err)
	}

	if !t.Failed() {
		t.Log("✓ undecodable image data is a classified decode error")
	}
}

// interFileFailed checks that a failed image file preserves its resource identity and returns no
// artifact.
func interFileFailed(t *testing.T) {
	t.Helper()
	r, sc := interSrv(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/f9" {
			_, _ = io.WriteString(w, `{"state":"FAILED","error":{"message":"processing broke"}}`)

			return
		}

		http.NotFound(w, req)
	})
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"image","mime_type":"image/png","uri":"`+
		r.srv.URL+`/files/f9:download?alt=media"}]}`)

	artifacts, _, err := imgCall(t, r, "gemini-3-pro-image", params.Values{}, nil)
	if !errors.Is(err, errs.ErrResponseGen) {
		t.Errorf("✗ err = %v, want the generation-failure chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "files/f9") {
		t.Errorf("✗ error does not name the resource: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from a FAILED file: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ a FAILED file surfaces through the traversal under the generation-failure chain")
	}
}

// vidFileFailed checks that a failed video file preserves its resource identity and returns no
// artifact.
func vidFileFailed(t *testing.T) {
	t.Helper()
	r, sc := vidFileSrv(t, "v9", "video/mp4", "X",
		`{"state":"FAILED","error":{"message":"encode broke"}}`)
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"video","mime_type":"video/mp4","uri":"`+
		r.srv.URL+`/files/v9:download?alt=media"}]}`)

	artifacts, err := vidCall(t, r, params.Values{}, nil)
	if !errors.Is(err, errs.ErrResponseGen) {
		t.Errorf("✗ err = %v, want the generation-failure chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "files/v9") {
		t.Errorf("✗ error does not name the resource: %v", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from a FAILED file: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ a FAILED file surfaces through the video traversal under the generation-failure chain")
	}
}

// vidNoCarriage checks that a video block without data or a URI returns no artifact.
func vidNoCarriage(t *testing.T) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"video","mime_type":"video/mp4"}]}`)

	artifacts, err := vidCall(t, r, params.Values{}, nil)
	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ err = %v, want the no-data chain", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from a block carrying neither data nor a uri: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ a video block with neither data nor uri falls through to no-data")
	}
}

// vidImgBlock checks that an image block cannot satisfy a video response.
func vidImgBlock(t *testing.T) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, imgStep(t, pngPix(t)))

	artifacts, err := vidCall(t, r, params.Values{}, nil)
	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ err = %v, want the no-data chain", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ the video traversal delivered an image block: %+v", artifacts)
	}

	if !t.Failed() {
		t.Log("✓ an image block never satisfies the video traversal")
	}
}

// vidBadData checks that invalid inline video data returns a decode error naming the block.
func vidBadData(t *testing.T) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"video","mime_type":"video/mp4","data":"!!!not-base64!!!"}]}`)

	_, err := vidCall(t, r, params.Values{}, nil)
	if !errors.Is(err, errs.ErrResponseDecode) {
		t.Errorf("✗ err = %v, want the decode chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "video"+" "+blockDataContext) {
		t.Errorf("✗ error does not name the video block: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ undecodable video data is a classified decode error naming the block")
	}
}

// vidBadURI checks that an unusable video URI returns a decode error naming the reference.
func vidBadURI(t *testing.T) {
	t.Helper()
	r, sc := interSrv(t, nil)
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"video","mime_type":"video/mp4","uri":"https://example.com/no-file-here"}]}`)

	_, err := vidCall(t, r, params.Values{}, nil)
	if !errors.Is(err, errs.ErrResponseDecode) {
		t.Errorf("✗ err = %v, want the decode chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "no-file-here") {
		t.Errorf("✗ error does not name the uri: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ an unextractable video uri is a classified decode error naming the reference")
	}
}

// vidDropDl checks interrupted video download classification and temporary-file cleanup.
func vidDropDl(t *testing.T) {
	t.Helper()
	ownTempDir(t)
	before := dlNames(t)
	r, sc := interSrv(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.RequestURI() == "/files/vd:download?alt=media":
			w.Header().Set("Content-Length", "1048576")
			_, _ = io.WriteString(w, "partial")
			// Flush the headers and partial body so the client is mid-stream — not
			// pre-response — when the connection aborts.
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		case req.URL.Path == "/files/vd":
			_, _ = io.WriteString(w, `{"state":"ACTIVE"}`)
		default:
			http.NotFound(w, req)
		}
	})
	sc.body = interBody(t, `{"type":"model_output","content":[{"type":"video","mime_type":"video/mp4","uri":"`+
		r.srv.URL+`/files/vd:download?alt=media"}]}`)

	artifacts, err := vidCall(t, r, params.Values{}, nil)
	if !errors.Is(err, errs.ErrTransportDownload) {
		t.Errorf("✗ err = %v, want the download-failure chain", err)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ artifacts from a dropped download: %+v", artifacts)
	}

	noNewDl(t, before)

	if !t.Failed() {
		t.Log("✓ a dropped video download fails classified and leaves no bild-dl-* file")
	}
}

// serveGone fails the second download with a non-2xx status.
func serveGone(test testing.TB) http.HandlerFunc {
	test.Helper()

	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "gone")
	}
}

// serveDrop flushes a partial response body, then aborts the connection to simulate an interrupted
// download.
func serveDrop(test testing.TB) http.HandlerFunc {
	test.Helper()

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		_, _ = io.WriteString(w, "partial")
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	}
}

// checkAtomic drives one two-sample completion whose first URI downloads and whose second fails,
// asserting the classified error, zero artifacts, and zero leftover temp files.
func checkAtomic(t *testing.T, c atomCase) {
	t.Helper()
	ownTempDir(t)
	before := dlNames(t)
	r, sc := veoOpSrv(t, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/dl/a":
			w.Header().Set("Content-Type", "video/mp4")
			_, _ = io.WriteString(w, "SAMPLE-A")
		case "/dl/b":
			if c.serveB != nil {
				c.serveB(w, req)

				return
			}

			http.NotFound(w, req)
		default:
			http.NotFound(w, req)
		}
	})

	second := c.second
	if second == "" {
		second = uriVid(t, r.srv.URL+"/dl/b")
	}

	sc.polls = []string{doneOp(t, uriVid(t, r.srv.URL+"/dl/a")+","+second)}

	artifacts, err := liteCall(t, r)
	if !errors.Is(err, c.want) {
		t.Errorf("✗ err = %v, want %v", err, c.want)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ a failed normalization returned artifacts: %+v", artifacts)
	}

	noNewDl(t, before)

	if !t.Failed() {
		t.Log("✓ the later failure is classified, yields zero artifacts, and leaves zero temp files")
	}
}

// edgeNoName checks that an unnamed Veo operation returns a no-data error naming the model.
func edgeNoName(t *testing.T) {
	t.Helper()
	r := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	})

	_, err := liteCall(t, r)
	if !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ err = %v, want the no-data chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "veo-3.1-lite-generate-preview") {
		t.Errorf("✗ error does not name the model: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a nameless start response fails as no-data naming the model")
	}
}

// edgeBadStart checks that a rejected Veo submission preserves the provider message.
func edgeBadStart(t *testing.T) {
	t.Helper()
	r := newRecServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"bad start"}}`)
	})

	_, err := liteCall(t, r)
	if !errors.Is(err, errs.ErrResponseStatus) {
		t.Errorf("✗ err = %v, want the error-status chain", err)
	}

	if err != nil && !strings.Contains(err.Error(), "bad start") {
		t.Errorf("✗ error does not surface the provider message: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a non-2xx start surfaces the provider message under the status chain")
	}
}

// edgeNoHost checks transport classification for an unreachable Veo endpoint.
func edgeNoHost(t *testing.T) {
	t.Helper()
	t.Setenv("GOOGLE_API_KEY", "k")
	req := subReq(t, "veo-3.1-lite-generate-preview", "p", nil)

	_, err := harnessVeoProvider(t, "http://127.0.0.1:1", time.Millisecond, 50*time.Millisecond).Generate(context.Background(), &req)
	if !errors.Is(err, errs.ErrTransport) {
		t.Errorf("✗ err = %v, want the transport chain", err)
	}

	if !t.Failed() {
		t.Log("✓ an unreachable host fails classified in the transport chain")
	}
}

// edgeBadPoll checks that an HTTP 500 response stops Veo polling after one request.
func edgeBadPoll(t *testing.T) {
	t.Helper()

	polls := 0
	r := newRecServer(t, func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, ":predictLongRunning") {
			_, _ = io.WriteString(w, `{"name":"ops/op1"}`)

			return
		}

		polls++

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	})

	_, err := liteCall(t, r)
	if !errors.Is(err, errs.ErrResponseStatus) {
		t.Errorf("✗ err = %v, want the error-status chain", err)
	}

	if polls != 1 {
		t.Errorf("✗ %d polls after the terminal status, want 1", polls)
	}

	if !t.Failed() {
		t.Log("✓ a non-2xx poll is terminal under the status chain")
	}
}

// edgeDropPoll checks that repeated connection failures remain in the polling timeout error.
func edgeDropPoll(t *testing.T) {
	t.Helper()
	r := newRecServer(t, func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, ":predictLongRunning") {
			_, _ = io.WriteString(w, `{"name":"ops/op1"}`)

			return
		}
		// Abort every poll pre-response: a transient transport failure the loop retries
		// until its deadline.
		panic(http.ErrAbortHandler)
	})
	t.Setenv("GOOGLE_API_KEY", "k")
	req := subReq(t, "veo-3.1-lite-generate-preview", "p", nil)

	_, err := harnessVeoProvider(t, r.srv.URL, 5*time.Millisecond, 60*time.Millisecond).Generate(context.Background(), &req)
	if !errors.Is(err, errs.ErrTransportTimeout) {
		t.Errorf("✗ err = %v, want the transport-timeout chain carrying the transient", err)
	}

	if err != nil && !errors.Is(err, errs.ErrTransportRequest) {
		t.Errorf("✗ the deadline error does not carry the last transient failure: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ dropped polls stay transient and surface as the deadline timeout")
	}
}
