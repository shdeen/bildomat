package catalog

// Invariants tested:
//  1. Valid provider configuration loading: Given validProvConfig, loadProviderConfig must succeed,
//     retain provider ID fixt and key variable FIXT_KEY, and return two models.
//  2. Built-in provider configuration loading: For every checked-in provider configuration,
//     loadProviderConfig must preserve the declared identity, aggregator flag, and complete model
//     records.
//  3. Video API configuration without video models: Given a provider with only an image model,
//     loadProviderConfig must accept a videoAPI section whose pollInterval is zero.
//  4. Parameter configuration schema: Given validParamsProvConfig with flag IDs, provider parameter
//     IDs, and constraints, loadProviderConfig must return no error.
//  5. Adapter API configuration: Given validAdapterCfg, loadProviderConfig must preserve the
//     AdapterAPI base URL and leave ImageAPI and VideoAPI nil.
//  6. Parameter requirement classification: Given required true, false, or absent on input-media,
//     loadProviderConfig must preserve true only for the explicit true case.
//  7. Documentation and prompt fields: Given provider and model docsURL fields and promptIgnored
//     true, loadProviderConfig must retain both exact documentation URLs and the model's
//     prompt-ignored value.
//  8. Model name and description: Given validProvConfig, loadProviderConfig must retain both model
//     names, preserve the image model's declared description, and leave the video model's
//     undeclared description empty.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/params"
)

// TestLoadValid verifies invariant #1: Valid provider configuration loading.
//
// What is being tested:
// Given validProvConfig, loadProviderConfig must succeed, retain provider ID fixt and key variable
// FIXT_KEY, and return two models.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLoadValid(t *testing.T) {
	provCfg, err := loadProviderConfig(params.Flags(), "fixture.json", []byte(validProvConfig))
	if err != nil {
		t.Fatalf("💣 valid config failed to load: %v", err)
	}

	if provCfg.ID != "fixt" || provCfg.APIKeyEnvVar != "FIXT_KEY" {
		t.Errorf("✗ loaded identity = %+v, want the declared provider", provCfg.Identity())
	}

	if len(provCfg.Models) != 2 {
		t.Errorf("✗ loaded %d models, want 2", len(provCfg.Models))
	}

	if !t.Failed() {
		t.Log("✓ a valid config decodes with its declared identity and models")
	}
}

// TestLoadRealProvConfigs verifies invariant #2: Built-in provider configuration loading.
//
// What is being tested:
// For every checked-in provider configuration, loadProviderConfig must preserve the declared
// identity, aggregator flag, and complete model records. Each file must declare models, and any
// declared video polling interval and timeout must be positive.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLoadRealProvConfigs(t *testing.T) {
	// Compare loaded identity and models with each configuration document, then check that any
	// declared video polling values are positive.
	cfgPaths, err := filepath.Glob(filepath.Join("..", "provider", "config", "*.json"))
	if err != nil || len(cfgPaths) == 0 {
		t.Fatalf("💣 no shipped provider configs found in the shared config directory: %v", err)
	}

	for _, cfgPath := range cfgPaths {
		cfgName := filepath.Base(cfgPath)

		t.Run(cfgName, func(t *testing.T) {
			checkRealProvConfig(t, cfgName)

			if !t.Failed() {
				t.Logf("✓ %s", cfgName)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every shipped provider config loads with full fidelity, and a video API carries positive integer-second poll fields")
	}
}

// TestLoadVidAPIWithoutVideoModels verifies invariant #3: Video API configuration without video
// models.
//
// What is being tested:
// Given a provider with only an image model, loadProviderConfig must accept a videoAPI section
// whose pollInterval is zero.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLoadVidAPIWithoutVideoModels(t *testing.T) {
	b := mutate(t, func(m map[string]any) {
		m["models"] = []any{firstModel(t, m)} // the image model only
		setIn(t, m, "config.videoAPI.pollInterval", 0)
	})
	if _, err := loadProviderConfig(params.Flags(), "fixture.json", b); err != nil {
		t.Errorf("✗ a video description with no declared video medium failed the poll pacing check: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ poll pacing applies only to a declared video medium")
	}
}

// TestLoadParamConfigSchema verifies invariant #4: Parameter configuration schema.
//
// What is being tested:
// Given validParamsProvConfig with flag IDs, provider parameter IDs, and constraints,
// loadProviderConfig must return no error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLoadParamConfigSchema(t *testing.T) {
	if _, err := loadProviderConfig(params.Flags(), "fixt.json", []byte(validParamsProvConfig)); err != nil {
		t.Errorf("✗ a valid params-schema provider config failed to load: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ the params schema loads cleanly")
	}
}

// TestAdapterAPIConfig verifies invariant #5: Adapter API configuration.
//
// What is being tested:
// Given validAdapterCfg, loadProviderConfig must preserve the AdapterAPI base URL and leave
// ImageAPI and VideoAPI nil.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdapterAPIConfig(t *testing.T) {
	provCfg, err := loadProviderConfig(params.Flags(), "adapter.json", []byte(validAdapterCfg))
	if err != nil {
		t.Fatalf("✗ a valid adapter config failed to load: %v", err)
	}

	if provCfg.Config.AdapterAPI == nil || provCfg.Config.AdapterAPI.APIBase != "https://example.test/v1" {
		t.Errorf("✗ AdapterAPI = %+v, want the declared section", provCfg.Config.AdapterAPI)
	}

	if provCfg.Config.ImageAPI != nil || provCfg.Config.VideoAPI != nil {
		t.Errorf("✗ an adapter config decoded engine API descriptions it never declared")
	}

	if !t.Failed() {
		t.Log("✓ an adapter config loads with its AdapterAPI section and no per-medium requirement")
	}
}

// TestParamRequirementClassification verifies invariant #6: Parameter requirement classification.
//
// What is being tested:
// Given required true, false, or absent on input-media, loadProviderConfig must preserve true only
// for the explicit true case. It must preserve aggregator true when declared and leave it false
// when absent.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestParamRequirementClassification(t *testing.T) {
	flags := []params.Flag{{FlagID: params.FlagTypeInputMedia, FlagName: "Input media", DataType: params.DataString, TextHint: "media-file", AllowMultiple: true}}

	cases := []struct {
		name   string
		record string
		want   bool
	}{
		{"a record marked required", `{"flagID": "input-media", "required": true, "maxMultiple": 1}`, true},
		{"a record marked not required", `{"flagID": "input-media", "required": false, "maxMultiple": 1}`, false},
		{"a record declaring no mark", `{"flagID": "input-media", "maxMultiple": 1}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			provCfg, err := loadProviderConfig(flags, "fixt.json", guidanceProvConfig(t, c.record))
			if err != nil {
				t.Fatalf("💣 %s failed to load: %v", c.name, err)
			}

			if inputCfg, _ := provCfg.Models[0].Param(params.FlagTypeInputMedia); inputCfg.Required != c.want {
				t.Errorf("✗ %s decoded as required=%v, want %v", c.name, inputCfg.Required, c.want)
			}

			if !t.Failed() {
				t.Logf("✓ %s decodes", c.name)
			}
		})
	}

	marked := []byte(`{
  "id": "fixt", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY", "aggregator": true,
  "models": [
    {"id": "m-img", "name": "Fixture Image", "media": "image", "params": [
      {"flagID": "input-media", "required": true, "maxMultiple": 1}
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

	provCfg, err := loadProviderConfig(flags, "fixt.json", marked)
	if err != nil {
		t.Fatalf("💣 the marked config failed to load: %v", err)
	}

	if !provCfg.Aggregator {
		t.Errorf("✗ the provider's aggregator mark did not decode")
	}

	unmarked, err := loadProviderConfig(flags, "fixt.json", guidanceProvConfig(t, `{"flagID": "input-media", "maxMultiple": 1}`))
	if err != nil {
		t.Fatalf("💣 the unmarked config failed to load: %v", err)
	}

	if unmarked.Aggregator {
		t.Errorf("✗ an absent aggregator mark decoded as true")
	}

	if !t.Failed() {
		t.Log("✓ the required mark decodes only when declared true, and the aggregator mark defaults to off")
	}
}

// TestDocsAndPromptFieldsLoad verifies invariant #7: Documentation and prompt fields.
//
// What is being tested:
// Given provider and model docsURL fields and promptIgnored true, loadProviderConfig must retain
// both exact documentation URLs and the model's prompt-ignored value.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDocsAndPromptFieldsLoad(t *testing.T) {
	configDocument := fmt.Appendf(nil, `{
	  "id": "fixt", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY", "docsURL": "https://docs.example.test/fixt",
	  "models": [{"id": "model-one", "name": "Model One", "media": "image", "promptIgnored": true, "docsURL": "https://docs.example.test/fixt/model-one"}],
	  "config": {"imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": %q, "inputMediaListProvParam": "images",
	    "fallbackExt": ".png"
	  }}
	}`, InputMediaSingle)

	prov, err := loadProviderConfig(params.Flags(), "fixt.json", configDocument)
	if err != nil {
		t.Fatalf("💣 the configuration failed to load: %v", err)
	}

	if prov.DocsURL != "https://docs.example.test/fixt" {
		t.Errorf("✗ provider docsURL = %q", prov.DocsURL)
	}

	if len(prov.Models) != 1 || !prov.Models[0].PromptIgnored || prov.Models[0].DocsURL != "https://docs.example.test/fixt/model-one" {
		t.Errorf("✗ model fields = %+v, want promptIgnored true and the model's docsURL", prov.Models)
	}

	if !t.Failed() {
		t.Log("✓ the documentation addresses and the prompt-ignored marker load from the configuration")
	}
}

// TestModelNameAndDescription verifies invariant #8: Model name and description.
//
// What is being tested:
// Given validProvConfig, loadProviderConfig must retain both model names, preserve the image
// model's declared description, and leave the video model's undeclared description empty.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestModelNameAndDescription(t *testing.T) {
	provCfg, err := loadProviderConfig(params.Flags(), "fixture.json", []byte(validProvConfig))
	if err != nil {
		t.Fatalf("💣 the fixture config failed to decode: %v", err)
	}

	if len(provCfg.Models) < 2 {
		t.Fatalf("💣 the fixture declares %d models, want at least 2", len(provCfg.Models))
	}

	if got := provCfg.Models[0].Name; got != "Fixture Image" {
		t.Errorf("✗ the image model's name = %q, want the declared name", got)
	}

	if got := provCfg.Models[0].Description; got != "A fixture image model." {
		t.Errorf("✗ the image model's description = %q, want the declared description", got)
	}

	// A description is optional: the provider publishes none for this model.
	if got := provCfg.Models[1].Description; got != "" {
		t.Errorf("✗ the video model's description = %q, want empty", got)
	}

	if got := provCfg.Models[1].Name; got != "Fixture Video" {
		t.Errorf("✗ the video model's name = %q, want the declared name", got)
	}

	if !t.Failed() {
		t.Log("✓ a model carries its declared name, and its description where one is declared")
	}
}

// validProvConfig is a minimal valid provider config with one image and one video model; tests
// corrupt one field per case.
const validProvConfig = `{
  "id": "fixt", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
  "models": [
    {"id": "m-img", "name": "Fixture Image", "description": "A fixture image model.", "media": "image", "params": [
      {"paramID": "aspect_ratio", "flagID": "aspect-ratio"},
      {"flagID": "input-media", "maxMultiple": 3}
    ]},
    {"id": "m-vid", "name": "Fixture Video", "media": "video", "params": [
      {"paramID": "duration", "flagID": "duration", "minValue": 1, "maxValue": 15},
      {"flagID": "input-media", "maxMultiple": 1}
    ]}
  ],
  "config": {
  "imageAPI": {
    "genURL": "https://example.test/img",
    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
    "fallbackExt": ".png"
  },
  "videoAPI": {
    "asyncJobsURL": "https://example.test/vid", "jobIDField": "id",
    "progressStatusText": ["pending"], "completedStatusText": "done", "failedStatusText": ["failed"],
    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
    "fallbackExt": ".mp4",
    "pollInterval": 5, "pollTimeout": 600
  }
  }
}`

// validParamsProvConfig is a minimal provider document with typed parameter definitions; tests
// corrupt one params entry per case.
const validParamsProvConfig = `{
  "id": "fixt", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
  "models": [
    {"id": "m-img", "name": "Fixture Image", "media": "image", "params": [
      {"flagID": "aspect-ratio", "paramID": "aspect_ratio", "allowedValues": ["1:1", "16:9"]},
      {"flagID": "size", "paramID": "size", "customSize": {"minEdge": 256, "maxEdge": 1440, "edgeIncrem": 32, "longEdge": 1024}},
      {"flagID": "num-images", "paramID": "n", "maxValue": 10},
      {"flagID": "input-media", "maxMultiple": 3}
    ]},
    {"id": "m-vid", "name": "Fixture Video", "media": "video", "params": [
      {"flagID": "duration", "paramID": "duration", "minValue": 1, "maxValue": 15},
      {"flagID": "input-media", "maxMultiple": 1}
    ]}
  ],
  "config": {
  "imageAPI": {
    "genURL": "https://example.test/img",
    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
    "fallbackExt": ".png"
  },
  "videoAPI": {
    "asyncJobsURL": "https://example.test/vid", "jobIDField": "id",
    "progressStatusText": ["pending"], "completedStatusText": "done", "failedStatusText": ["failed"],
    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
    "fallbackExt": ".mp4", "pollInterval": 5, "pollTimeout": 600
  }
  }
}`

// validAdapterCfg is a minimal valid adapter config: models over an AdapterAPI section, no engine
// API descriptions.
const validAdapterCfg = `{
  "id": "fixt", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
  "models": [
    {"id": "m-img", "name": "Fixture Image", "media": "image", "params": [
      {"paramID": "aspect_ratio", "flagID": "aspect-ratio"}
    ]},
    {"id": "m-vid", "name": "Fixture Video", "media": "video", "params": [
      {"paramID": "duration", "flagID": "duration", "allowedValues": ["4", "8"]}
    ]}
  ],
  "config": {
  "adapterAPI": {
    "apiBase": "https://example.test/v1",
    "pollInterval": 2, "pollTimeout": 300,
    "filePollInterval": 5, "filePollTimeout": 300,
	"pendingStatusText": ["Pending","Reasoning","Generating"],
	"readyStatusText": "Ready",
    "failedStatusText": ["Failed"],
    "imageFallbackExt": ".jpg", "videoFallbackExt": ".mp4"
  }
  }
}`

// mutate decodes validProvConfig into a generic map, applies edit, and re-encodes.
func mutate(t *testing.T, edit func(map[string]any)) []byte {
	t.Helper()

	var m map[string]any
	if err := json.Unmarshal([]byte(validProvConfig), &m); err != nil {
		t.Fatalf("💣 fixture does not parse: %v", err)
	}

	edit(m)

	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("💣 fixture re-encode failed: %v", err)
	}

	return b
}

// setIn traverses a dotted path in the fixture map and sets the leaf.
func setIn(test testing.TB, m map[string]any, path string, v any) {
	test.Helper()

	parts := strings.Split(path, ".")

	cur := m
	for _, p := range parts[:len(parts)-1] {
		cur = cur[p].(map[string]any)
	}

	cur[parts[len(parts)-1]] = v
}

// firstModel returns the fixture's first model.
func firstModel(test testing.TB, m map[string]any) map[string]any {
	test.Helper()

	return m["models"].([]any)[0].(map[string]any)
}

// realProvConfig reads one checked-in embedded provider config from the provider package's config
// directory (the same bytes the embed directives carry).
func realProvConfig(t *testing.T, cfgName string) []byte {
	t.Helper()

	// #nosec G304 -- cfgName selects a checked-in provider fixture supplied by the test.
	b, err := os.ReadFile(filepath.Join("..", "provider", "config", cfgName))
	if err != nil {
		t.Fatalf("💣 cannot read the checked-in provider config %s: %v", cfgName, err)
	}

	return b
}

// checkRealProvConfig compares loaded identity and models with the file and checks polling values.
func checkRealProvConfig(t *testing.T, cfgName string) {
	t.Helper()
	b := realProvConfig(t, cfgName)

	var d struct {
		ID           string  `json:"id"`
		DisplayName  string  `json:"displayName"`
		APIKeyEnvVar string  `json:"apiKeyEnvVar"`
		Aggregator   bool    `json:"aggregator"`
		Models       []Model `json:"models"`
		Config       struct {
			VideoAPI *VideoAPI `json:"videoAPI"`
		} `json:"config"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatalf("💣 %s does not decode into the descriptor structs: %v", cfgName, err)
	}

	provCfg, err := loadProviderConfig(params.Flags(), cfgName, b)
	if err != nil {
		t.Fatalf("💣 real config %s failed to load: %v", cfgName, err)
	}

	if provCfg.ID != d.ID || provCfg.DisplayName != d.DisplayName || provCfg.APIKeyEnvVar != d.APIKeyEnvVar || provCfg.Aggregator != d.Aggregator {
		t.Errorf("✗ %s identity = %+v, want the declared (%s, %s, %s, %v)", cfgName, provCfg.Identity(), d.ID, d.DisplayName, d.APIKeyEnvVar, d.Aggregator)
	}

	if len(d.Models) == 0 {
		t.Errorf("✗ %s declares no models — an empty shipped config is a repository defect", cfgName)
	}

	if !reflect.DeepEqual(provCfg.Models, d.Models) {
		t.Errorf("✗ %s loaded models differ from the declared set (%d loaded, %d declared)", cfgName, len(provCfg.Models), len(d.Models))
	}

	if d.Config.VideoAPI != nil && (d.Config.VideoAPI.PollInterval <= 0 || d.Config.VideoAPI.PollTimeout <= 0) {
		t.Errorf("✗ %s poll fields = (%d,%d), want positive integer seconds", cfgName, d.Config.VideoAPI.PollInterval, d.Config.VideoAPI.PollTimeout)
	}
}

// guidanceProvConfig returns a one-model image config declaring the given params entries, for the
// guidance and marking cases.
func guidanceProvConfig(test testing.TB, paramsJSON string) []byte {
	test.Helper()

	return []byte(`{
  "id": "fixt", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
  "models": [
    {"id": "m-img", "name": "Fixture Image", "media": "image", "params": [` + paramsJSON + `]}
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
