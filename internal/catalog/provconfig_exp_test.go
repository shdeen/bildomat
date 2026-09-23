package catalog

// Invariants tested:
//  1. Provider configuration decode errors: Given malformed JSON, unknown fields, or trailing
//     garbage, loadProviderConfig must return ErrProvConfigDecode; a second complete document must
//     return ErrProvConfigTrailing.
//  2. Provider configuration validation errors: Given each invalid media value, name, payload
//     layout, flag, frame description, API section, or polling value in the table,
//     loadProviderConfig must return ErrProvConfigInvalid under ErrProvConfig and identify the
//     configuration label and fault.
//  3. Parameter configuration validation: Given an unknown flag, conflicting constraints, a
//     negative media limit, inverted bounds, a wrongly typed allowed value, or a duplicate
//     parameter definition, loadProviderConfig must return ErrProvConfigInvalid.
//  4. Invalid adapter polling settings: Given zero or negative main polling values, or a negative
//     file polling interval, loadProviderConfig must return ErrProvConfigInvalid for the adapter
//     configuration.
//  5. Image count floor: Given allowed image counts containing zero or minus one,
//     loadProviderConfig must return ErrProvConfigInvalid.
//  6. Missing provider configuration section: Given a provider document with no config section and
//     no models, loadProviderConfig must return ErrProvConfigInvalid and include ConfigMissing in
//     its error text.
//  7. String input media style validation: Given the string input-media layout, loadProviderConfig
//     must succeed.
//  8. Parameter guidance validation: Given a parameter with no hint, examples, or constraints,
//     loadProviderConfig must return ErrProvConfigInvalid with the expected message naming its
//     model and flag.
//  9. Adapter fallback extensions: Given image and video adapter models, loadProviderConfig must
//     reject either missing applicable fallback extension with ErrProvConfigInvalid.
//  10. Provider default model check: Given a default model ID absent from the provider,
//      loadProviderConfig must return ErrProvConfigInvalid and name the configuration label.
//  11. Unambiguous provider descriptions: Starting from a valid nested-media configuration,
//      loadProviderConfig must reject each missing media description, empty model ID, incompatible
//      encoding, missing plural field, empty path segment, or overlapping parameter path with
//      ErrProvConfigInvalid and the configuration label.
//  12. Applicable polling settings: Given an adapter that declares job statuses, loadProviderConfig
//      must reject absent main polling values.
//  13. Provider configuration loading under arbitrary input: For arbitrary configuration bytes,
//      loadProviderConfig must return an error under ErrProvConfig or a provider value.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// TestLoadDecodeErrors verifies invariant #1: Provider configuration decode errors.
//
// What is being tested:
// Given malformed JSON, unknown fields, or trailing garbage, loadProviderConfig must return
// ErrProvConfigDecode; a second complete document must return ErrProvConfigTrailing. Each error
// must also match ErrProvConfig and name the configuration label and expected decoding fault.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestLoadDecodeErrors(t *testing.T) {
	cases := []struct {
		name     string
		bytes    []byte
		hint     string
		sentinel error
	}{
		{"malformed JSON", []byte(`{"provider": `), "EOF", errs.ErrProvConfigDecode},
		{"unknown top-level key", mutate(t, func(m map[string]any) { m["Bogus"] = 1 }), "Bogus", errs.ErrProvConfigDecode},
		{"unknown model key", mutate(t, func(m map[string]any) { firstModel(t, m)["Bogus"] = 1 }), "Bogus", errs.ErrProvConfigDecode},
		{"trailing document", []byte(validProvConfig + "\n{}"), "trailing", errs.ErrProvConfigTrailing},
		{"trailing garbage", []byte(validProvConfig + "\nxyz"), "invalid character", errs.ErrProvConfigDecode},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := loadProviderConfig(params.Flags(), "fixture.json", c.bytes)
			if err == nil {
				t.Fatalf("💣 no error for %s", c.name)
			}

			if !errors.Is(err, c.sentinel) || !errors.Is(err, errs.ErrProvConfig) {
				t.Errorf("✗ error does not match %v and the ErrProvConfig root: %v", c.sentinel, err)
			}

			if !strings.Contains(err.Error(), "fixture.json") {
				t.Errorf("✗ error does not name the config label: %v", err)
			}

			if !strings.Contains(err.Error(), c.hint) {
				t.Errorf("✗ error does not carry the decode fault (%q): %v", c.hint, err)
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ malformed JSON and trailing documents retain their distinct provider-config classifications")
	}
}

// TestLoadValidationErrors verifies invariant #2: Provider configuration validation errors.
//
// What is being tested:
// Given each invalid media value, name, payload layout, flag, frame description, API section, or
// polling value in the table, loadProviderConfig must return ErrProvConfigInvalid under
// ErrProvConfig and identify the configuration label and fault.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestLoadValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		edit func(map[string]any)
		hint string
	}{
		{"bad media", func(m map[string]any) { setIn(t, firstModel(t, m), "media", "audio") }, "audio"},
		{"missing model name", func(m map[string]any) { delete(firstModel(t, m), "name") }, "name"},
		{"blank model name", func(m map[string]any) { setIn(t, firstModel(t, m), "name", "") }, "name"},
		{"bad InputMediaPayloadType", func(m map[string]any) { setIn(t, m, "config.imageAPI.inputMediaPayloadType", "yaml") }, "InputMediaPayloadType"},
		{"bad InputMediaStyle", func(m map[string]any) { setIn(t, m, "config.videoAPI.inputMediaStyle", "flat") }, "InputMediaStyle"},
		{"unknown StringParams flag", func(m map[string]any) { setIn(t, m, "config.stringParams", []any{"not-a-flag"}) }, "StringParams"},
		{"incomplete frame media description", func(m map[string]any) { setIn(t, m, "config.videoAPI.frameMediaProvParam", "frames") }, "frame-media"},
		{"missing image description", func(m map[string]any) { delete(m["config"].(map[string]any), "imageAPI") }, "image"},
		{"missing video description", func(m map[string]any) { delete(m["config"].(map[string]any), "videoAPI") }, "video"},
		{"negative pace", func(m map[string]any) { setIn(t, m, "config.videoAPI.pollInterval", -1) }, "PollInterval"},
		{"negative deadline", func(m map[string]any) { setIn(t, m, "config.videoAPI.pollTimeout", -1) }, "PollTimeout"},
		{"zero pace", func(m map[string]any) { setIn(t, m, "config.videoAPI.pollInterval", 0) }, "PollInterval"},
		{"zero deadline", func(m map[string]any) { setIn(t, m, "config.videoAPI.pollTimeout", 0) }, "PollTimeout"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := loadProviderConfig(params.Flags(), "fixture.json", mutate(t, c.edit))
			if err == nil {
				t.Fatalf("💣 no error for %s", c.name)
			}

			if !errors.Is(err, errs.ErrProvConfigInvalid) || !errors.Is(err, errs.ErrProvConfig) {
				t.Errorf("✗ error does not match ErrProvConfigInvalid and the ErrProvConfig root: %v", err)
			}

			if !strings.Contains(err.Error(), "fixture.json") {
				t.Errorf("✗ error does not name the config label: %v", err)
			}

			if !strings.Contains(err.Error(), c.hint) {
				t.Errorf("✗ error does not name the fault (%q): %v", c.hint, err)
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ each independently invalid field fails as ErrProvConfigInvalid naming the label and fault")
	}
}

// TestParamConfigValidationErrors verifies invariant #3: Parameter configuration validation.
//
// What is being tested:
// Given an unknown flag, conflicting constraints, a negative media limit, inverted bounds, a
// wrongly typed allowed value, or a duplicate parameter definition, loadProviderConfig must return
// ErrProvConfigInvalid.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestParamConfigValidationErrors(t *testing.T) {
	cases := map[string]string{
		"flagID naming no enumerated flag":             `{"flagID": "sixe", "paramID": "size"}`,
		"allowed values and a range both declared":     `{"flagID": "duration", "paramID": "duration", "allowedValues": ["4"], "maxValue": 15}`,
		"size bounds and allowed values both declared": `{"flagID": "size", "paramID": "size", "allowedValues": ["1024x1024"], "customSize": {"longEdge": 1024}}`,
		"negative repeat maximum":                      `{"flagID": "input-media", "maxMultiple": -1}`,
		"range minimum over its maximum":               `{"flagID": "duration", "paramID": "duration", "minValue": 15, "maxValue": 1}`,
		"allowed value failing the flag's data type":   `{"flagID": "duration", "paramID": "duration", "allowedValues": ["x"]}`,
	}
	for name, paramConfig := range cases {
		doc := strings.Replace(validParamsProvConfig,
			`{"flagID": "duration", "paramID": "duration", "minValue": 1, "maxValue": 15}`,
			paramConfig, 1)

		_, err := loadProviderConfig(params.Flags(), "fixt.json", []byte(doc))
		if !errors.Is(err, errs.ErrProvConfigInvalid) {
			t.Errorf("✗ %s: error = %v, want the invalid-content sentinel", name, err)
		}
	}

	dup := strings.Replace(validParamsProvConfig,
		`{"flagID": "duration", "paramID": "duration", "minValue": 1, "maxValue": 15}`,
		`{"flagID": "duration", "paramID": "duration", "minValue": 1, "maxValue": 15},
     {"flagID": "duration", "paramID": "duration", "minValue": 1, "maxValue": 15}`, 1)
	if _, err := loadProviderConfig(params.Flags(), "fixt.json", []byte(dup)); !errors.Is(err, errs.ErrProvConfigInvalid) {
		t.Errorf("✗ duplicate entries for one flag: error = %v, want the invalid-content sentinel", err)
	}

	if !t.Failed() {
		t.Log("✓ invalid params entries fail with the classified invalid sentinel")
	}
}

// TestAdapterAPIPacingErrors verifies invariant #4: Invalid adapter polling settings.
//
// What is being tested:
// Given zero or negative main polling values, or a negative file polling interval,
// loadProviderConfig must return ErrProvConfigInvalid for the adapter configuration.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestAdapterAPIPacingErrors(t *testing.T) {
	cases := map[string]string{
		"negative interval":       `"pollInterval": -1, "pollTimeout": 300`,
		"zero interval with wait": `"pollInterval": 0, "pollTimeout": 300`,
		"negative timeout":        `"pollInterval": 2, "pollTimeout": -5`,
		"zero timeout with pace":  `"pollInterval": 2, "pollTimeout": 0`,
		"negative file pacing":    `"pollInterval": 2, "pollTimeout": 300, "filePollInterval": -1, "filePollTimeout": 300`,
	}
	for name, pacing := range cases {
		doc := strings.Replace(validAdapterCfg,
			`"pollInterval": 2, "pollTimeout": 300,
    "filePollInterval": 5, "filePollTimeout": 300,`,
			pacing+",", 1)
		if _, err := loadProviderConfig(params.Flags(), "adapter.json", []byte(doc)); !errors.Is(err, errs.ErrProvConfigInvalid) {
			t.Errorf("✗ %s: error = %v, want the invalid-content sentinel", name, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ declared adapter poll pacing must be positive; a wholly undeclared pair passes")
	}
}

// TestCountFloorUnbreachable verifies invariant #5: Image count floor.
//
// What is being tested:
// Given allowed image counts containing zero or minus one, loadProviderConfig must return
// ErrProvConfigInvalid. With valid allowed counts one and three, params.Adjust must omit supplied
// counts zero and minus one, return one Rejected record, and return no error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCountFloorUnbreachable(t *testing.T) {
	countSetCfg := func(members string) []byte {
		t.Helper()

		return []byte(`{
  "id": "fixt", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
  "models": [
    {"id": "m-img", "name": "Fixture Image", "media": "image", "params": [
      {"paramID": "n", "flagID": "num-images", "allowedValues": [` + members + `]}
    ]}
  ],
  "config": {
    "imageAPI": {
      "genURL": "https://example.test/img",
      "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
      "fallbackExt": ".png"
    }
  }
}`)
	}
	for _, belowOne := range []string{`"-1", "3"`, `"0", "3"`} {
		if _, err := loadProviderConfig(params.Flags(), "fixture.json", countSetCfg(belowOne)); !errors.Is(err, errs.ErrProvConfigInvalid) {
			t.Errorf("✗ a num-images set with members %s decoded with error %v, want the invalid-content sentinel", belowOne, err)
		}
	}

	provCfg, err := loadProviderConfig(params.Flags(), "fixture.json", countSetCfg(`"1", "3"`))
	if err != nil {
		t.Fatalf("💣 a valid num-images set failed to decode: %v", err)
	}

	for _, suppliedCount := range []int{0, -1} {
		gp, paramChanges, adjustmentErr := params.Adjust(params.FlagInputs{params.FlagTypeImageN: suppliedCount}, provCfg.Models[0].Params, provCfg.Models[0].ID)
		if adjustmentErr != nil {
			t.Errorf("✗ parameter adjustment failed: %v", adjustmentErr)
		}

		if adjustedCount, ok := gp[params.FlagTypeImageN]; ok {
			t.Errorf("✗ a supplied count of %d transmitted as %v; a below-one count must never transmit", suppliedCount, adjustedCount)
		}

		if len(paramChanges) != 1 || paramChanges[0].Type != params.ChangeRejected {
			t.Errorf("✗ a supplied count of %d recorded %+v, want one Rejected record", suppliedCount, paramChanges)
		}
	}

	if !t.Failed() {
		t.Log("✓ no path through strict decoding and adjustment transmits a num-images count below one")
	}
}

// TestLoadProviderConfigMissingConfig verifies invariant #6: Missing provider configuration
// section.
//
// What is being tested:
// Given a provider document with no config section and no models, loadProviderConfig must return
// ErrProvConfigInvalid and include ConfigMissing in its error text.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestLoadProviderConfigMissingConfig(t *testing.T) {
	document := []byte(`{"id": "bare", "displayName": "Bare", "apiKeyEnvVar": "BARE_KEY", "models": []}`)

	_, err := loadProviderConfig(params.Flags(), "bare.json", document)
	if !errors.Is(err, errs.ErrProvConfigInvalid) {
		t.Fatalf("✗ a document without a config section loaded: %v", err)
	}

	if !strings.Contains(err.Error(), ConfigMissing) {
		t.Errorf("✗ the fault %q does not name the missing config section", err)
	}

	if !t.Failed() {
		t.Log("✓ a provider document without its config section is refused")
	}
}

// TestStringInputMediaStyleValidation verifies invariant #7: String input media style validation.
//
// What is being tested:
// Given the string input-media layout, loadProviderConfig must succeed. Given an unknown layout, it
// must return ErrProvConfigInvalid.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestStringInputMediaStyleValidation(t *testing.T) {
	stringLayoutConfig := mutate(t, func(document map[string]any) {
		setIn(t, document, "config.imageAPI.inputMediaStyle", "string")
	})
	if _, err := loadProviderConfig(params.Flags(), "fixture.json", stringLayoutConfig); err != nil {
		t.Errorf("✗ the string input-media layout failed validation: %v", err)
	}

	unknownLayoutConfig := mutate(t, func(document map[string]any) {
		setIn(t, document, "config.imageAPI.inputMediaStyle", "unknown-layout")
	})
	if _, err := loadProviderConfig(params.Flags(), "fixture.json", unknownLayoutConfig); !errors.Is(err, errs.ErrProvConfigInvalid) {
		t.Errorf("✗ unknown input-media layout error = %v, want the invalid-configuration classification", err)
	}

	if !t.Failed() {
		t.Log("✓ the string layout validates and an unknown layout remains invalid")
	}
}

// TestParamGuidanceValidation verifies invariant #8: Parameter guidance validation.
//
// What is being tested:
// Given a parameter with no hint, examples, or constraints, loadProviderConfig must return
// ErrProvConfigInvalid with the expected message naming its model and flag. It must accept each
// tested guidance form or constraint and accept a Boolean flag without extra guidance.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestParamGuidanceValidation(t *testing.T) {
	bareSeed := params.Flag{FlagID: "seed", FlagName: "Seed", DataType: params.DataInteger}
	hintedSeed := params.Flag{FlagID: "seed", FlagName: "Seed", DataType: params.DataInteger, TextHint: "number"}
	exampledSeed := params.Flag{FlagID: "seed", FlagName: "Seed", DataType: params.DataInteger, ExampleValues: []string{"7"}}
	bareSwitch := params.Flag{FlagID: "generate-audio", FlagName: "Generate audio", DataType: params.DataBoolean}
	bareSize := params.Flag{FlagID: params.FlagTypeSize, FlagName: "Size", DataType: params.DataString}

	cases := []struct {
		name    string
		flags   []params.Flag
		params  string
		wantErr bool
	}{
		{"bare param under a bare flag", []params.Flag{bareSeed}, `{"paramID": "seed", "flagID": "seed"}`, true},
		{"bare param under a hinted flag", []params.Flag{hintedSeed}, `{"paramID": "seed", "flagID": "seed"}`, false},
		{"bare param under an exampled flag", []params.Flag{exampledSeed}, `{"paramID": "seed", "flagID": "seed"}`, false},
		{"allowed values under a bare flag", []params.Flag{bareSeed}, `{"paramID": "seed", "flagID": "seed", "allowedValues": ["1", "2"]}`, false},
		{"a range bound under a bare flag", []params.Flag{bareSeed}, `{"paramID": "seed", "flagID": "seed", "minValue": 0}`, false},
		{"size bounds under a bare flag", []params.Flag{bareSize}, `{"paramID": "size", "flagID": "size", "customSize": {"minEdge": 64}}`, false},
		{"a bare boolean flag", []params.Flag{bareSwitch}, `{"paramID": "generate_audio", "flagID": "generate-audio"}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := loadProviderConfig(c.flags, "fixt.json", guidanceProvConfig(t, c.params))
			if c.wantErr {
				if !errors.Is(err, errs.ErrProvConfigInvalid) {
					t.Errorf("✗ %s loaded (%v), want the invalid-config error", c.name, err)

					return
				}

				if want := fmt.Sprintf(ParamGuidanceMissing, "m-img", "seed"); !strings.Contains(err.Error(), want) {
					t.Errorf("✗ %s: the fault %q does not name the model and flag: want %q", c.name, err.Error(), want)
				}

				if !t.Failed() {
					t.Logf("✓ %s is refused", c.name)
				}

				return
			}

			if err != nil {
				t.Errorf("✗ %s failed to load: %v", c.name, err)
			}

			if !t.Failed() {
				t.Logf("✓ %s loads", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ a guidance-less parameter is refused and every guidance form or declared constraint is accepted")
	}
}

// TestAdapterFallbackExtRequired verifies invariant #9: Adapter fallback extensions.
//
// What is being tested:
// Given image and video adapter models, loadProviderConfig must reject either missing applicable
// fallback extension with ErrProvConfigInvalid. Given only an image model, it must accept the image
// fallback without a video fallback.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestAdapterFallbackExtRequired(t *testing.T) {
	missingImage := strings.Replace(validAdapterCfg, `"imageFallbackExt": ".jpg", `, "", 1)
	if _, err := loadProviderConfig(params.Flags(), "adapter.json", []byte(missingImage)); !errors.Is(err, errs.ErrProvConfigInvalid) {
		t.Errorf("✗ an adapter with image models and no image fallback extension = %v, want the invalid-content sentinel", err)
	}

	missingVideo := strings.Replace(validAdapterCfg, `"videoFallbackExt": ".mp4"`, `"videoFallbackExt": ""`, 1)
	if _, err := loadProviderConfig(params.Flags(), "adapter.json", []byte(missingVideo)); !errors.Is(err, errs.ErrProvConfigInvalid) {
		t.Errorf("✗ an adapter with video models and no video fallback extension = %v, want the invalid-content sentinel", err)
	}

	imageOnlyCfg := `{
  "id": "fixt", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
  "models": [
    {"id": "m-img", "name": "Fixture Image", "media": "image", "params": [
      {"paramID": "aspect_ratio", "flagID": "aspect-ratio"}
    ]}
  ],
  "config": {
  "adapterAPI": {
    "apiBase": "https://example.test/v1",
    "pollInterval": 2, "pollTimeout": 300,
    "pendingStatusText": ["Pending"], "readyStatusText": "Ready",
    "failedStatusText": ["Failed"], "imageFallbackExt": ".jpg"
  }
  }
}`
	if _, err := loadProviderConfig(params.Flags(), "adapter.json", []byte(imageOnlyCfg)); err != nil {
		t.Errorf("✗ an image-only adapter without a video fallback extension failed: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ each produced medium requires its declared fallback extension, and only its own")
	}
}

// TestProviderDefaultModelCheck verifies invariant #10: Provider default model check.
//
// What is being tested:
// Given a default model ID absent from the provider, loadProviderConfig must return
// ErrProvConfigInvalid and name the configuration label. Given a declared model ID, it must succeed
// and retain that default.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestProviderDefaultModelCheck(t *testing.T) {
	_, err := loadProviderConfig(params.Flags(), "fixt.json", []byte(defaultFixtureCfg(t, "fixt", "Fixture", "FIXT_KEY", "model-one", "model-missing")))
	if !errors.Is(err, errs.ErrProvConfigInvalid) || !strings.Contains(err.Error(), "fixt.json") {
		t.Errorf("✗ a default naming no model loaded or failed without the invalid-config sentinel and label: %v", err)
	}

	prov, err := loadProviderConfig(params.Flags(), "fixt.json", []byte(defaultFixtureCfg(t, "fixt", "Fixture", "FIXT_KEY", "model-one", "model-one")))
	if err != nil || prov.DefaultModel != "model-one" {
		t.Errorf("✗ a default naming a declared model: (%q, %v), want model-one and no error", prov.DefaultModel, err)
	}

	if !t.Failed() {
		t.Log("✓ a provider default must name one of the provider's models")
	}
}

// TestProviderDescriptionConflicts verifies invariant #11: Unambiguous provider descriptions.
//
// What is being tested:
// Starting from a valid nested-media configuration, loadProviderConfig must reject each missing
// media description, empty model ID, incompatible encoding, missing plural field, empty path
// segment, or overlapping parameter path with ErrProvConfigInvalid and the configuration label.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestProviderDescriptionConflicts(t *testing.T) {
	validDescription := strings.ReplaceAll(validParamsProvConfig, `"single-or-array"`, `"nested"`)
	if _, err := loadProviderConfig(params.Flags(), "fixt.json", []byte(validDescription)); err != nil {
		t.Fatalf("💣 valid configuration setup failed: %v", err)
	}

	configurations := []struct{ name, description string }{
		{"missing applicable media description", strings.Replace(validDescription, `"inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "nested", "inputMediaListProvParam": "images",`, "", 1)},
		{"empty model ID", strings.Replace(validDescription, `"id": "m-img"`, `"id": ""`, 1)},
		{"JSON file parts", strings.Replace(validDescription, `"nested"`, `"parts"`, 1)},
		{"multipart nested values", strings.Replace(validDescription, `"inputMediaPayloadType": "json"`, `"inputMediaPayloadType": "form"`, 1)},
		{"missing plural field", strings.Replace(strings.ReplaceAll(validDescription, `, "inputMediaListProvParam": "images"`, ""), `"nested"`, `"single-or-array"`, 1)},
		{"empty path segment", strings.Replace(validDescription, `"paramID": "aspect_ratio"`, `"paramID": "request..aspect"`, 1)},
		{"equal parameter paths", strings.Replace(validDescription, `"paramID": "aspect_ratio"`, `"paramID": "size"`, 1)},
		{"ancestor parameter path", strings.Replace(validDescription, `"paramID": "aspect_ratio"`, `"paramID": "size.aspect"`, 1)},
		{"ancestor separated in sort order", strings.Replace(strings.Replace(validDescription, `"paramID": "aspect_ratio"`, `"paramID": "size.aspect"`, 1), `"paramID": "n"`, `"paramID": "size-other"`, 1)},
	}
	for _, configuration := range configurations {
		_, err := loadProviderConfig(params.Flags(), "fixt.json", []byte(configuration.description))
		if !errors.Is(err, errs.ErrProvConfigInvalid) || !strings.Contains(fmt.Sprint(err), "fixt.json") {
			t.Errorf("✗ %s lacked a classified configuration error: %v", configuration.name, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ invalid identity, encodings, and paths fail at description loading")
	}
}

// TestApplicableAdapterPollingRequired verifies invariant #12: Applicable polling settings.
//
// What is being tested:
// Given an adapter that declares job statuses, loadProviderConfig must reject absent main polling
// values. With positive main values, it must accept absent file polling or a positive file pair and
// reject either incomplete file pair.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestApplicableAdapterPollingRequired(t *testing.T) {
	for _, intervals := range []struct {
		mainInterval, mainTimeout, fileInterval, fileTimeout int
		valid                                                bool
	}{
		{0, 0, 0, 0, false}, {1, 5, 0, 0, true}, {1, 5, 1, 0, false}, {1, 5, 0, 1, false}, {1, 5, 1, 5, true},
	} {
		configuration := fmt.Sprintf(`{"id":"fixt","models":[{"id":"image","name":"Image","media":"image"}],"config":{"adapterAPI":{"imageFallbackExt":".png","pendingStatusText":["pending"],"readyStatusText":"done","pollInterval":%d,"pollTimeout":%d,"filePollInterval":%d,"filePollTimeout":%d}}}`, intervals.mainInterval, intervals.mainTimeout, intervals.fileInterval, intervals.fileTimeout)

		_, err := loadProviderConfig(params.Flags(), "fixt.json", []byte(configuration))
		if intervals.valid && err != nil {
			t.Errorf("✗ valid polling settings rejected: %+v: %v", intervals, err)
		}

		if !intervals.valid && !errors.Is(err, errs.ErrProvConfigInvalid) {
			t.Errorf("✗ invalid polling settings accepted: %+v: %v", intervals, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ main polling is required when used and file polling remains conditional")
	}
}

// FuzzLoad verifies invariant #13: Provider configuration loading under arbitrary input.
//
// What is being tested:
// For arbitrary configuration bytes, loadProviderConfig must return an error under ErrProvConfig or
// a provider value. If that value contains models, its provider ID must be nonempty.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzLoad(f *testing.F) {
	f.Add([]byte(validProvConfig))
	f.Add([]byte(`{`))
	f.Add([]byte(``))
	f.Add([]byte(`{"id":"x","models":[]}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		provCfg, err := loadProviderConfig(params.Flags(), "fuzz.json", b)
		switch {
		case err != nil:
			if !errors.Is(err, errs.ErrProvConfig) {
				t.Errorf("✗ a decode error outside the provider-config category: %v", err)
			}
		case provCfg.ID == "" && len(provCfg.Models) > 0:
			t.Errorf("✗ a decoded config carries models under an empty provider identity")
		}

		if !t.Failed() {
			t.Logf("✓ the bytes decoded to a valid config or failed inside the provider-config category")
		}
	})
}

// defaultFixtureCfg returns a one-model provider configuration declaring the given display name,
// key variable, model, and default model (none when empty).
func defaultFixtureCfg(test testing.TB, providerID, displayName, keyVar, modelID, defaultModel string) string {
	test.Helper()

	defaultField := ""
	if defaultModel != "" {
		defaultField = fmt.Sprintf(`"defaultModel": %q,`, defaultModel)
	}

	return fmt.Sprintf(`{
	  "id": %[1]q, "displayName": %[2]q, "apiKeyEnvVar": %[3]q, %[5]s
	  "models": [{"id": %[4]q, "name": %[4]q, "media": "image"}],
	  "config": {"imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaListProvParam": "images", "inputMediaStyle": %[6]q,
	    "fallbackExt": ".png"
	  }}
	}`, providerID, displayName, keyVar, modelID, defaultField, InputMediaSingle)
}
