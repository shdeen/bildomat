package main

// Invariants tested:
//  1. Print filename relative output path: With filename printing enabled and a relative output
//     directory, testGenerate must write exactly two absolute paths, each naming an existing
//     nonempty file.
//  2. Print filename output: With filename printing enabled, testGenerate must write exactly two
//     lines, each naming an existing nonempty file inside the selected output directory.
//  3. Saved results text: Given a results-file path, testGenerate must leave stdout empty and write
//     the model header and a Saved report to that file.
//  4. Saved results with printed filenames: Given a results-file path and filename printing
//     enabled, testGenerate must write exactly two stdout lines and include a Saved report in the
//     results file.
//  5. Thoughts sidecar content: Given adjusted include-thoughts=true, testGenerate must create
//     run.png and run.md. The sidecar must contain the quoted prompt and model fields and the
//     returned thought.
//  6. Order of generation notices: testGenerate must print the ignored Duration notice before the
//     Aspect ratio adjustment and print both before the Saved report.
//  7. Separation of flag help from model notes: cliRenderFlagEntry must omit ModelInfoComment text
//     and blank paragraph separators from the parameter flag's help entry.
//  8. Terminal reports for saved artifacts: Given two artifacts and terminal stdout, testGenerate
//     must print one completion report and two Saved reports, include the clay color escape
//     sequence, and omit Saved (.
//  9. Help tips fallback: When the rotation providers are absent, tipsHelpText must use exactly one
//     valid model key per medium from the first loaded provider, including an aggregator, and none
//     from the later provider.
//  10. Applied API key carriage: After applyAPIKeys and model resolution, apiKey must return the
//      configured credential rather than the environment value. Marshaling the catalog must omit
//      the configured credential.
//  11. Apply API keys unknown IDs: applyAPIKeys must return exactly the configured provider IDs
//      absent from the loaded catalog.
//  12. Provider copy carries applied key: After applyAPIKeys, apiKey must return the configured
//      credential for a provider copy obtained from the catalog.
//  13. Default model key: defaultModelKey must skip providers without credentials or defaults and
//      choose the first eligible provider in display-name order.
//  14. Provider API key lookup: apiKey must read the declared environment variable. If it is empty,
//      the error must match errs.ErrKeyMissing and name the variable, configuration key, and
//      configuration file.
//  15. API key precedence: apiKey must prefer a nonempty configured credential and fall back to the
//      environment when the configured value is empty.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"
)

// TestPrintFilenameRelativeOutputPath verifies invariant #1: Print filename relative output path.
//
// What is being tested:
// Given a relative output directory and filename printing enabled, the generation stages run by
// testGenerate must succeed and write exactly two stdout lines. Each line must be an absolute path
// to an existing nonempty file.
//
// Test class: Core.
func TestPrintFilenameRelativeOutputPath(t *testing.T) {
	loadedCatalog, genInputs, gen, _ := fixtImgGeneration(t)
	genInputs.OutPath = params.GetSetIf(true, "relative-out"+string(os.PathSeparator))

	t.Chdir(t.TempDir())

	app := testApp(t, loadedCatalog)

	stdout, err := captureGenerate(t, app, gen, genInputs, params.FlagInputs{}, true, "")
	if err != nil {
		t.Fatalf("💣 the fixture generation failed: %v", err)
	}

	pathLines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(pathLines) != 2 {
		t.Fatalf("💣 stdout carries %d lines, want exactly the 2 landed paths: %q", len(pathLines), stdout)
	}

	for _, landedPath := range pathLines {
		if !filepath.IsAbs(landedPath) {
			t.Errorf("✗ the path line %q is not absolute", landedPath)
		}

		if fi, statErr := os.Stat(landedPath); statErr != nil || fi.Size() == 0 {
			t.Errorf("✗ the path line %q names no existing non-empty file (%v)", landedPath, statErr)
		}
	}

	if !t.Failed() {
		t.Log("✓ a relative output directory still yields absolute path lines")
	}
}

// TestPrintFilenameLandedPathsOnly verifies invariant #2: Print filename output.
//
// What is being tested:
// Given two generated image artifacts and filename printing enabled, the generation stages run by
// testGenerate must succeed and write exactly two stdout lines. Each line must name an existing
// nonempty file inside the selected output directory.
//
// Test class: Core.
func TestPrintFilenameLandedPathsOnly(t *testing.T) {
	loadedCatalog, genInputs, gen, outDir := fixtImgGeneration(t)

	app := testApp(t, loadedCatalog)

	stdout, err := captureGenerate(t, app, gen, genInputs, params.FlagInputs{}, true, "")
	if err != nil {
		t.Fatalf("💣 the fixture generation failed: %v", err)
	}

	pathLines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(pathLines) != 2 {
		t.Fatalf("💣 stdout carries %d lines, want exactly the 2 landed paths: %q", len(pathLines), stdout)
	}

	for _, landedPath := range pathLines {
		if !strings.HasPrefix(landedPath, outDir+string(os.PathSeparator)) {
			t.Errorf("✗ the path line %q is outside the output directory", landedPath)
		}

		if fi, statErr := os.Stat(landedPath); statErr != nil || fi.Size() == 0 {
			t.Errorf("✗ the path line %q names no existing non-empty file (%v)", landedPath, statErr)
		}
	}

	if !t.Failed() {
		t.Log("✓ print-filename limits stdout to the landed paths in landing order")
	}
}

// TestSaveResultsFileText verifies invariant #3: Saved results text.
//
// What is being tested:
// Given a results-file path and filename printing disabled, the generation stages run by
// testGenerate must succeed and leave stdout empty. The results file must contain the fixt-img
// model header and a Saved report.
//
// Test class: Core.
func TestSaveResultsFileText(t *testing.T) {
	loadedCatalog, genInputs, gen, _ := fixtImgGeneration(t)

	app := testApp(t, loadedCatalog)
	resultsFilePath := filepath.Join(t.TempDir(), "results.txt")

	stdout, err := captureGenerate(t, app, gen, genInputs, params.FlagInputs{}, false, resultsFilePath)
	if err != nil {
		t.Fatalf("💣 the fixture generation failed: %v", err)
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty with a results path and no print-filename", stdout)
	}

	// #nosec G304 -- the path is the test-owned results file under TempDir.
	resultsContent, readErr := os.ReadFile(resultsFilePath)
	if readErr != nil {
		t.Fatalf("💣 the results file was not written: %v", readErr)
	}

	for _, requiredText := range []string{fmt.Sprintf(output.ModelHeader, "fixt-img"), formPrefix(t, output.SavedReport)} {
		if !strings.Contains(string(resultsContent), requiredText) {
			t.Errorf("✗ the results file lacks %q:\n%s", requiredText, resultsContent)
		}
	}

	if !t.Failed() {
		t.Log("✓ the results file receives the regular stdout content")
	}
}

// TestSaveResultsWithPrintFilename verifies invariant #4: Saved results with printed filenames.
//
// What is being tested:
// Given both a results-file path and filename printing enabled, the generation stages run by
// testGenerate must succeed. Stdout must contain exactly two lines, and the results file must
// contain a Saved report.
//
// Test class: Core.
func TestSaveResultsWithPrintFilename(t *testing.T) {
	loadedCatalog, genInputs, gen, _ := fixtImgGeneration(t)

	app := testApp(t, loadedCatalog)
	resultsFilePath := filepath.Join(t.TempDir(), "results.txt")

	stdout, err := captureGenerate(t, app, gen, genInputs, params.FlagInputs{}, true, resultsFilePath)
	if err != nil {
		t.Fatalf("💣 the fixture generation failed: %v", err)
	}

	if pathLines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"); len(pathLines) != 2 {
		t.Errorf("✗ stdout carries %d lines, want exactly the 2 landed paths: %q", len(pathLines), stdout)
	}

	// #nosec G304 -- the path is the test-owned results file under TempDir.
	resultsContent, readErr := os.ReadFile(resultsFilePath)
	if readErr != nil {
		t.Fatalf("💣 the results file was not written: %v", readErr)
	}

	if !strings.Contains(string(resultsContent), formPrefix(t, output.SavedReport)) {
		t.Errorf("✗ the results file lacks the Saved reports:\n%s", resultsContent)
	}

	if !t.Failed() {
		t.Log("✓ the combined form routes text to the file and paths to stdout")
	}
}

// TestGenerateSidecar verifies invariant #5: Thoughts sidecar content.
//
// What is being tested:
// Given include-thoughts in adjusted parameters and the returned thought one thought, the
// generation stages run by testGenerate must succeed and create run.png and run.md. The sidecar
// must contain that thought and the quoted YAML fields prompt: "a prompt" and model: "fixt-img".
//
// Test class: Core: Incidental.
// Pins quoted YAML front-matter values.
func TestGenerateSidecar(t *testing.T) {
	gen := &stubGen{
		test:     t,
		adjusted: params.Values{params.FlagTypeThoughts: true},
		result: generation.Result{
			Artifacts: []artifact.Media{{Data: []byte("IMG"), FileExt: ".png"}},
			Thoughts:  []string{"one thought"},
		},
	}
	loadedCatalog := fixtureCatalog(t, fixtSource(t, "fixt", "Fixture", "FIXT_KEY",
		`{"id": "fixt-img", "name": "fixt-img", "media": "image", "params": [{"flagID": "include-thoughts"}]}`))
	outDir := t.TempDir()
	genInputs := RunFlags{
		Model:   params.GetSetIf(true, "fixt-img"),
		Prompt:  params.GetSetIf(true, "a prompt"),
		OutPath: params.GetSetIf(true, filepath.Join(outDir, "run")),
	}
	app := testApp(t, loadedCatalog)
	combined := captureBoth(t, func() {
		if err := testGenerate(t, app, gen, genInputs, params.FlagInputs{params.FlagTypeThoughts: true}, false, ""); err != nil {
			t.Errorf("✗ generate failed: %v", err)
		}
	})
	_ = combined

	// #nosec G304 -- outDir is a test-owned temporary directory.
	b, err := os.ReadFile(filepath.Join(outDir, "run.md"))
	if err != nil {
		t.Errorf("✗ sidecar run.md missing beside the artifact: %v", err)
	} else {
		for _, want := range []string{`prompt: "a prompt"`, `model: "fixt-img"`, "one thought"} {
			if !strings.Contains(string(b), want) {
				t.Errorf("✗ sidecar missing %q in:\n%s", want, b)
			}
		}
	}

	if _, err := os.Stat(filepath.Join(outDir, "run.png")); err != nil {
		t.Errorf("✗ artifact run.png missing: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ requesting thoughts writes the sidecar beside the artifact under the shared stem")
	}
}

// TestGenerateOrdering verifies invariant #6: Order of generation notices.
//
// What is being tested:
// Given an ignored Duration adjustment and a changed Aspect ratio, the generation stages run by
// testGenerate must succeed. Their combined output must contain the ignored Duration notice, the
// Aspect ratio adjustment to 1536x1024, and the Saved report in that order.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestGenerateOrdering(t *testing.T) {
	gen := &stubGen{
		test:     t,
		adjusted: params.Values{params.FlagTypeSize: "1536x1024"},
		records: []params.Adjustment{
			{FlagID: params.FlagTypeDuration, Type: params.ChangeIgnored, Comment: fmt.Sprintf(generation.ReasonNotConsumed, "fixt-img")},
			{FlagID: params.FlagTypeAspect, Type: params.ChangeSnapped, InputVal: "3:2", WireVal: "1536x1024"},
		},
		result: generation.Result{
			Artifacts: []artifact.Media{{Data: []byte("IMG"), FileExt: ".png"}},
		},
	}
	loadedCatalog := fixtureCatalog(t, fixtSource(t, "fixt", "Fixture", "FIXT_KEY",
		`{"id": "fixt-img", "name": "fixt-img", "media": "image", "params": [{"paramID": "aspect_ratio", "flagID": "aspect-ratio"}]}`))
	outDir := t.TempDir()
	genInputs := RunFlags{
		Model:   params.GetSetIf(true, "fixt-img"),
		Prompt:  params.GetSetIf(true, "a prompt"),
		OutPath: params.GetSetIf(true, outDir+string(os.PathSeparator)),
	}
	parameterValues := params.FlagInputs{
		params.FlagTypeAspect:   "3:2",
		params.FlagTypeDuration: 5, // not consumed → the ignored warning
	}

	var genErr error

	app := testApp(t, loadedCatalog)

	combined := captureBoth(t, func() {
		genErr = testGenerate(t, app, gen, genInputs, parameterValues, false, "")
	})
	if genErr != nil {
		t.Fatalf("💣 generate failed: %v (output: %q)", genErr, combined)
	}

	marks := []string{
		fmt.Sprintf(output.FlagIgnored, "Duration", fmt.Sprintf(generation.ReasonNotConsumed, "fixt-img")),
		fmt.Sprintf(output.FlagAdjusted, "Aspect ratio", "'1536x1024'"),
		formPrefix(t, output.SavedReport),
	}
	last := -1

	for _, m := range marks {
		idx := strings.Index(combined, m)
		if idx < 0 {
			t.Errorf("✗ output missing %q in:\n%s", m, combined)

			continue
		}

		if idx < last {
			t.Errorf("✗ %q rendered out of order in:\n%s", m, combined)
		}

		last = idx
	}

	if !t.Failed() {
		t.Log("✓ ignored warnings → pre-submit change notices → artifacts, in order")
	}
}

// TestParamFlagUsageWithoutModelComments verifies invariant #7: Separation of flag help from model
// notes.
//
// What is being tested:
// Given an input-media flag and a model with ModelInfoComment text, cliRenderFlagEntry must omit
// that text from the flag's help entry. The entry must contain no blank paragraph separator.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestParamFlagUsageWithoutModelComments(t *testing.T) {
	// #nosec G101 -- the literal names the credential's environment variable, not a credential.
	prov := catalog.Provider{ID: "delta", DisplayName: "Delta", APIKeyEnvVar: "DELTA_API_KEY"}
	framesModel := catalog.Model{ID: "vid-frames", Name: "Vid Frames", Media: media.Video, Params: []params.Definition{
		{FlagID: params.FlagTypeInputMedia, ModelInfoComment: "Fixture keyframe guidance in full."},
		{FlagID: params.FlagTypeDuration},
	}}
	prov.Models = []catalog.Model{framesModel}

	mediaFlag := params.Flag{
		FlagID: params.FlagTypeInputMedia, DataType: params.DataString,
		FlagName: "Input media", Description: "File path or web URL.",
	}
	bild := &bildApp{catalog: &catalog.Catalog{Providers: []catalog.Provider{prov}, Flags: []params.Flag{mediaFlag}}}
	mediaUsage := bild.cliRenderFlagEntry(paramCLIFlag(&mediaFlag))

	if strings.Contains(mediaUsage, "Fixture keyframe guidance in full.") {
		t.Errorf("✗ the flag's help entry carries model-record text:\n%s", mediaUsage)
	}

	if strings.Contains(mediaUsage, "\n\n") {
		t.Errorf("✗ the flag's help entry carries an appended paragraph:\n%s", mediaUsage)
	}

	if !t.Failed() {
		t.Log("✓ a flag's help entry is a single paragraph with no model-record text")
	}
}

// TestTTYStyledSavedReport verifies invariant #8: Terminal reports for saved artifacts.
//
// What is being tested:
// Given two generated artifacts and terminal stdout, the generation stages run by testGenerate must
// succeed and print exactly one completion report and two Saved reports. Stdout must include the
// clay color escape sequence and must not contain Saved (.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestTTYStyledSavedReport(t *testing.T) {
	loadedCatalog, genInputs, gen, _ := fixtImgGeneration(t)

	app := testApp(t, loadedCatalog)

	var err error

	stdout := captureTerminalStdout(t, func() {
		err = testGenerate(t, app, gen, genInputs, params.FlagInputs{}, false, "")
	})
	if err != nil {
		t.Fatalf("💣 the fixture generation failed: %v", err)
	}

	if strings.Count(stdout, formPrefix(t, output.GenerationCompleted)) != 1 {
		t.Errorf("✗ the single completion report is missing: %q", stdout)
	}

	if strings.Count(stdout, formPrefix(t, output.SavedReport)) != 2 {
		t.Errorf("✗ the two styled Saved reports are missing: %q", stdout)
	}

	if strings.Contains(stdout, "Saved (") {
		t.Errorf("✗ a duration is tied to a file report: %q", stdout)
	}

	if !strings.Contains(stdout, "\x1b[38;5;173m") {
		t.Errorf("✗ the clay-colored path is missing: %q", stdout)
	}

	if !t.Failed() {
		t.Log("✓ the terminal display renders the completion report and styled saved reports")
	}
}

// TestHelpTipsFallback verifies invariant #9: Help tips fallback.
//
// What is being tested:
// Given a catalog without OpenAI, Google, or xAI, tipsHelpText must succeed and include exactly one
// valid model key per represented medium from the first loaded provider, whether standard or
// aggregator. No extracted key may name the later provider.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestHelpTipsFallback(t *testing.T) {
	aggregatorSource := catalog.Source{ProviderID: "hub", ConfigBytes: fmt.Appendf(nil, `{
	  "id": "hub", "displayName": "Hub", "apiKeyEnvVar": "HUB_KEY", "aggregator": true,
	  "models": [{"id": "vendor/hub-img", "name": "hub-img", "media": "image", "params": []}],
	  "config": {"imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": %q, "inputMediaListProvParam": "images",
	    "fallbackExt": ".png"
	  }}
	}`, catalog.InputMediaSingle)}
	standardSource := fixtSource(t, "solo", "Solo", "SOLO_KEY", `{"id": "solo-img", "name": "solo-img", "media": "image", "params": []}`)

	cases := []struct {
		name      string
		sources   []catalog.Source
		drawnFrom string
		notFrom   string
	}{
		{"aggregator first", []catalog.Source{aggregatorSource, standardSource}, "hub", "solo"},
		{"standard first", []catalog.Source{standardSource, aggregatorSource}, "solo", "hub"},
	}

	for _, fallbackCase := range cases {
		t.Run(fallbackCase.name, func(t *testing.T) {
			bild := &bildApp{catalog: fixtureCatalog(t, fallbackCase.sources...)}

			tips, err := bild.tipsHelpText()
			if err != nil {
				t.Fatalf("💣 the tips failed to render: %v", err)
			}

			drawnProvider, loaded := bild.catalog.Provider(fallbackCase.drawnFrom)
			if !loaded {
				configErr := bild.catalog.ConfigError(fallbackCase.drawnFrom)
				t.Fatalf("💣 the fixture provider %s did not load: %v", fallbackCase.drawnFrom, configErr)
			}

			checkProviderKeys(t, tips, fallbackCase.drawnFrom, drawnProvider.Models)

			if len(pageKeysOf(t, tips, fallbackCase.drawnFrom)) == 0 {
				t.Errorf("✗ the tips name no key of %s:\n%s", fallbackCase.drawnFrom, tips)
			}

			if keys := pageKeysOf(t, tips, fallbackCase.notFrom); len(keys) != 0 {
				t.Errorf("✗ the tips name %v of the later provider %s", keys, fallbackCase.notFrom)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the tips fall back to the first loaded provider with a model, aggregator or not")
	}
}

// TestApplyAPIKeysCarriage verifies invariant #10: Applied API key carriage.
//
// What is being tested:
// Given both an environment credential and a nonempty configured credential for prov-key,
// applyAPIKeys must return no unknown IDs. After ResolveModelInput selects model-key, apiKey must
// return the configured credential without an error. Marshaling the catalog must succeed without
// including that credential's value.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestApplyAPIKeysCarriage(t *testing.T) {
	loadedCatalog := keyCatalog(t)
	bild := &bildApp{catalog: loadedCatalog}

	t.Setenv("BILD_TEST_CATALOG_KEY", "environment-value")

	if unknownIDs := bild.applyAPIKeys(map[string]string{"prov-key": "applied-secret-value"}); len(unknownIDs) != 0 {
		t.Errorf("✗ ApplyAPIKeys reported %v, want no unknown IDs", unknownIDs)
	}

	pairs, err := loadedCatalog.ResolveModelInput("model-key")
	if err != nil || len(pairs) != 1 {
		t.Fatalf("💣 ResolveModelInput = (%v, %v), want the one fixture pair", pairs, err)
	}

	apiKey, err := bild.apiKey(&pairs[0].Provider)
	if err != nil {
		t.Errorf("✗ APIKey on the resolved provider returned an error: %v", err)
	}

	if apiKey != "applied-secret-value" {
		t.Errorf("✗ resolved provider APIKey = %q, want the applied key", apiKey)
	}

	encoded, err := json.Marshal(loadedCatalog)
	if err != nil {
		t.Fatalf("💣 catalog encoding failed: %v", err)
	}

	if bytes.Contains(encoded, []byte("applied-secret-value")) {
		t.Errorf("✗ the encoded catalog contains the applied key's value")
	}

	if !t.Failed() {
		t.Log("✓ an applied key survives resolution to the provider value and never encodes into the catalog document")
	}
}

// TestApplyAPIKeysUnknownIDs verifies invariant #11: Apply API keys unknown IDs.
//
// What is being tested:
// Given configured credentials for prov-key and not-a-provider, applyAPIKeys must return exactly
// not-a-provider because only prov-key exists in the loaded catalog.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestApplyAPIKeysUnknownIDs(t *testing.T) {
	loadedCatalog := keyCatalog(t)
	bild := &bildApp{catalog: loadedCatalog}

	unknownIDs := bild.applyAPIKeys(map[string]string{"prov-key": "applied-secret-value", "not-a-provider": "stray-value"})
	if !reflect.DeepEqual(unknownIDs, []string{"not-a-provider"}) {
		t.Errorf("✗ ApplyAPIKeys reported %v, want exactly the unknown ID", unknownIDs)
	}

	if !t.Failed() {
		t.Log("✓ key application reports exactly the IDs that name no loaded provider")
	}
}

// TestProviderCopyCarriesAppliedKey verifies invariant #12: Provider copy carries applied key.
//
// What is being tested:
// After applyAPIKeys sets a configured credential for prov-key, apiKey must return it for the
// provider copy returned by Catalog.Provider, even when the environment supplies a different value.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderCopyCarriesAppliedKey(t *testing.T) {
	loadedCatalog := keyCatalog(t)
	bild := &bildApp{catalog: loadedCatalog}

	t.Setenv("BILD_TEST_CATALOG_KEY", "environment-value")

	if unknownIDs := bild.applyAPIKeys(map[string]string{"prov-key": "applied-secret-value"}); len(unknownIDs) != 0 {
		t.Fatalf("💣 ApplyAPIKeys reported %v, want no unknown IDs", unknownIDs)
	}

	prov, ok := loadedCatalog.Provider("prov-key")
	if !ok {
		t.Fatalf("💣 the catalog does not hold the fixture provider")
	}

	apiKey, err := bild.apiKey(&prov)
	if err != nil {
		t.Errorf("✗ APIKey on the provider copy returned an error: %v", err)
	}

	if apiKey != "applied-secret-value" {
		t.Errorf("✗ provider copy APIKey = %q, want the applied key", apiKey)
	}

	if !t.Failed() {
		t.Log("✓ the catalog's deep provider copy carries the applied key")
	}
}

// TestDefaultModelKey verifies invariant #13: Default model key.
//
// What is being tested:
// defaultModelKey must return false when no provider has credentials or only a provider without a
// default has credentials. With only Zeta eligible, it must select zeta/model-z. After Alpha
// receives a configured credential, it must select alpha/model-a, which sorts first by display
// name.
// Test class: Expanded.
// Test layer: Coverage.
func TestDefaultModelKey(t *testing.T) {
	loadedCatalog := loadFixtureCatalog(t,
		catalog.Source{ProviderID: "zeta", ConfigBytes: []byte(defaultFixtureCfg(t, "zeta", "Zeta", "ZETA_FIXTURE_KEY", "model-z", "model-z"))},
		catalog.Source{ProviderID: "alpha", ConfigBytes: []byte(defaultFixtureCfg(t, "alpha", "Alpha", "ALPHA_FIXTURE_KEY", "model-a", "model-a"))},
		catalog.Source{ProviderID: "nodef", ConfigBytes: []byte(defaultFixtureCfg(t, "nodef", "Aaa No Default", "NODEF_FIXTURE_KEY", "model-n", ""))},
	)
	bild := &bildApp{catalog: loadedCatalog}

	for _, keyVar := range []string{"ZETA_FIXTURE_KEY", "ALPHA_FIXTURE_KEY", "NODEF_FIXTURE_KEY"} {
		t.Setenv(keyVar, "")
	}

	if key, ok := bild.defaultModelKey(); ok {
		t.Errorf("✗ with no keys the default key is %q, want none", key)
	}

	t.Setenv("NODEF_FIXTURE_KEY", "env-value")

	if key, ok := bild.defaultModelKey(); ok {
		t.Errorf("✗ a keyed provider without a declared default yielded %q, want none", key)
	}

	t.Setenv("ZETA_FIXTURE_KEY", "env-value")

	if key, ok := bild.defaultModelKey(); !ok || key != "zeta/model-z" {
		t.Errorf("✗ with the environment key for zeta the default key is (%q, %t), want zeta/model-z", key, ok)
	}

	if unknown := bild.applyAPIKeys(map[string]string{"alpha": "config-value"}); len(unknown) != 0 {
		t.Fatalf("💣 ApplyAPIKeys reported unknown IDs %v", unknown)
	}

	if key, ok := bild.defaultModelKey(); !ok || key != "alpha/model-a" {
		t.Errorf("✗ with alpha keyed through the config the default key is (%q, %t), want alpha/model-a first in listing order", key, ok)
	}

	if !t.Failed() {
		t.Log("✓ the default key follows listing order over the providers holding a key and a declared default")
	}
}

// TestProviderKey verifies invariant #14: Provider API key lookup.
//
// What is being tested:
// Given PROV_TEST_API_KEY=sk-value, apiKey must return sk-value without an error. When that
// variable is empty, it must return errs.ErrKeyMissing and name PROV_TEST_API_KEY, api-keys.prov,
// and ~/.bildomat/config.yml in the error.
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderKey(t *testing.T) {
	bild := &bildApp{}
	// #nosec G101 -- the literal names the credential's environment variable, not a credential.
	p := catalog.Provider{ID: "prov", DisplayName: "Prov", APIKeyEnvVar: "PROV_TEST_API_KEY"}

	t.Setenv("PROV_TEST_API_KEY", "sk-value")

	key, err := bild.apiKey(&p)
	if err != nil || key != "sk-value" {
		t.Errorf("✗ Key() with the env set = (%q, %v), want the value", key, err)
	}

	t.Setenv("PROV_TEST_API_KEY", "")

	_, err = bild.apiKey(&p)
	if err == nil || !errors.Is(err, errs.ErrKeyMissing) {
		t.Errorf("✗ Key() with the env unset = %v, want a wrapped errs.ErrKeyMissing", err)
	}

	if err != nil {
		for _, setting := range []string{p.APIKeyEnvVar, "api-keys.prov", "~/.bildomat/config.yml"} {
			if !strings.Contains(err.Error(), setting) {
				t.Errorf("✗ missing-key error %q omits %q", err.Error(), setting)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ the declared env var yields the key; a missing key names both configuration methods")
	}
}

// TestAPIKeyPrecedence verifies invariant #15: API key precedence.
//
// What is being tested:
// When both configuration and environment supply a credential, apiKey must return the nonempty
// configured value. After that configured value is cleared, apiKey must return the environment
// value. Both calls must succeed.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAPIKeyPrecedence(t *testing.T) {
	bild := &bildApp{apiKeys: map[string]string{"": "user-config-value"}}
	prov := catalog.Provider{APIKeyEnvVar: "BILD_TEST_KEY_VAR"} // #nosec G101 -- fixture value, not a credential

	t.Setenv("BILD_TEST_KEY_VAR", "environment-value")

	apiKey, err := bild.apiKey(&prov)
	if err != nil {
		t.Errorf("✗ APIKey with a user-config key returned an error: %v", err)
	}

	if apiKey != "user-config-value" { // #nosec G101 -- fixture value, not a credential
		t.Errorf("✗ APIKey = %q, want the user-config value over the environment", apiKey)
	}

	bild.apiKeys[prov.ID] = ""

	apiKey, err = bild.apiKey(&prov)
	if err != nil {
		t.Errorf("✗ APIKey with an empty user-config key returned an error: %v", err)
	}

	if apiKey != "environment-value" {
		t.Errorf("✗ APIKey = %q, want the environment fallback", apiKey)
	}

	if !t.Failed() {
		t.Log("✓ a user-config key outranks the environment, and an empty one falls back to it")
	}
}

// stubGen returns configured preparation and generation results.
//   - test: the owning test context.
//   - adjusted: parameter values returned by preparation.
//   - records: adjustments returned by preparation.
//   - result: artifacts and thoughts returned by generation.
//   - err: the generation error.
type stubGen struct {
	test     testing.TB
	adjusted params.Values
	records  []params.Adjustment
	result   generation.Result
	err      error
}

// formPrefix returns the text before the first formatting placeholder.
//
// Test class: Core: Helper.
func formPrefix(t *testing.T, form string) string {
	t.Helper()

	prefix, _, _ := strings.Cut(form, "%")

	return prefix
}

// realApp creates an application with the built-in catalog and the current process streams.
func realApp(t *testing.T) *bildApp {
	t.Helper()

	app := &bildApp{invocation: newInvocation(os.Stdin, os.Stdout, os.Stderr)}
	app.invocation.outcome = &output.GenerationOutcome{}

	app.catalog = shippedCatalog(t)

	return app
}

// testApp creates an application with the supplied catalog, empty pipe input, and a temporary
// output directory.
//
// Test class: Core: Helper.
func testApp(t *testing.T, loadedCatalog *catalog.Catalog) *bildApp {
	t.Helper()

	useStdin(t, "", false)

	app := &bildApp{invocation: newInvocation(os.Stdin, os.Stdout, os.Stderr)}
	app.invocation.outcome = &output.GenerationOutcome{}

	app.catalog = loadedCatalog
	app.defaultOutDir = t.TempDir()

	return app
}

// captureTerminalStdout runs fn with stdout on a pseudo-terminal and stderr discarded. It returns
// the terminal output.
func captureTerminalStdout(test testing.TB, fn func()) string {
	test.Helper()

	master, slave := openPTY(test)

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		test.Fatalf("💣 open %s: %v", os.DevNull, err)
	}

	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = slave, devNull

	received := make(chan string, 1)

	// This goroutine owns the master. After fn returns, closing the slave ends the read. The
	// goroutine closes the master and returns the captured text.
	go func() {
		var terminal bytes.Buffer

		_, _ = io.Copy(&terminal, master)
		_ = master.Close()

		received <- terminal.String()
	}()

	fn()

	os.Stdout, os.Stderr = originalOut, originalErr
	_ = devNull.Close()
	_ = slave.Close()

	return <-received
}

// fixtureCatalog loads the supplied provider descriptions with the built-in parameter flags.
//
// Test class: Core: Helper.
func fixtureCatalog(t *testing.T, sources ...catalog.Source) *catalog.Catalog {
	t.Helper()

	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags, sources...)
	if err != nil {
		t.Fatalf("💣 fixture catalog construction failed: %v", err)
	}

	return loadedCatalog
}

// fixtSource builds a provider configuration using the supplied model-config JSON fragment.
//
// Test class: Core: Helper.
func fixtSource(test testing.TB, providerID, displayName, keyEnv, modelConfigs string) catalog.Source {
	test.Helper()

	doc := fmt.Sprintf(`{
	  "id": %q, "displayName": %q, "apiKeyEnvVar": %q,
	  "models": [%s],
	  "config": {"imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": %q, "inputMediaListProvParam": "images",
	    "fallbackExt": ".png"
	  }}
	}`, providerID, displayName, keyEnv, modelConfigs, catalog.InputMediaSingle)

	return catalog.Source{ProviderID: providerID, ConfigBytes: []byte(doc)}
}

// testGenerate runs the command's actual generation stages with a configured test generator.
//
// Test class: Core: Helper.
func testGenerate(test testing.TB, app *bildApp, generator generation.Generator, genInputs RunFlags, userInputs params.FlagInputs, printFilename bool, resultsFilePath string) error {
	test.Helper()

	app.invocation = newInvocation(os.Stdin, os.Stdout, os.Stderr)
	app.invocation.printFilename = printFilename
	app.invocation.animate = app.invocation.styled && !printFilename

	app.invocation.outcome = output.NewGenerationOutcome(time.Now(), genInputs.Prompt.ValOr(""), getGenFlagsInput(&genInputs, userInputs))
	if err := app.invocation.openResults(resultsFilePath); err != nil {
		return err
	}

	pair, err := app.resolveModelInput(genInputs.Model.ValOr(app.defaultModel))
	if err != nil {
		return err
	}

	sourceArguments, _ := userInputs[params.FlagTypeInputMedia].([]string)

	retainedMedia, err := media.ReadInputs(generation.SelectSources(sourceArguments, &pair.Model))
	if err != nil {
		return err
	}

	outPath, pathChanges, err := resolveOutputTarget(&genInputs, &pair.Model, userInputs, app.defaultOutDir)
	if err != nil {
		return err
	}

	configuredApp := *app
	configuredApp.apiKeys = map[string]string{pair.Provider.ID: "test-key"}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	_, _, err = configuredApp.executeGeneration(ctx, generator, &pair, &genInputs, userInputs, retainedMedia, outPath, pathChanges, nil)

	return configuredApp.invocation.finish(configuredApp.invocation.reportGeneration(time.Now(), pair.Provider.DisplayName, pair.Model.Name, err))
}

// AdjustParams returns the fixture's configured adjusted values and records.
//
// Test class: Core: Helper.
func (s *stubGen) AdjustParams(_ *catalog.Model, _ params.FlagInputs, inputs []media.Input, _ *metadata.Reuse) (generation.Preparation, error) {
	s.test.Helper()

	return generation.Preparation{Params: s.adjusted, Changes: s.records, InputMedia: inputs}, nil
}

// Generate returns the fixture's configured result and error.
//
// Test class: Core: Helper.
func (s *stubGen) Generate(_ context.Context, request *generation.Generation) (generation.Result, error) {
	s.test.Helper()

	result := s.result
	result.Preparation = request.Clone()

	return result, s.err
}

// captureBoth runs fn with stdout AND stderr redirected into one pipe, so cross-stream ordering is
// observable.
//
// Test class: Core: Helper.
func captureBoth(t *testing.T, fn func()) string {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 os.Pipe: %v", err)
	}

	os.Stdout, os.Stderr = w, w
	done := make(chan string, 1)

	go func() { b, _ := io.ReadAll(r); done <- string(b) }()

	fn()

	os.Stdout, os.Stderr = origOut, origErr
	_ = w.Close()

	return <-done
}

// captureGenerate runs testGenerate with stdout captured and stderr discarded. It returns the
// captured output and run error. It reads after closing the pipe writer, so these tests must keep
// their output small enough to fit in the pipe buffer.
//
// Test class: Core: Helper.
func captureGenerate(t *testing.T, app *bildApp, generator generation.Generator, genInputs RunFlags, userInputs params.FlagInputs, printFilename bool, resultsFilePath string) (stdout string, genErr error) {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 stdout pipe: %v", err)
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("💣 open %s: %v", os.DevNull, err)
	}

	os.Stdout, os.Stderr = outW, devNull
	genErr = testGenerate(t, app, generator, genInputs, userInputs, printFilename, resultsFilePath)
	os.Stdout, os.Stderr = origOut, origErr

	_ = outW.Close()
	_ = devNull.Close()

	outBytes, _ := io.ReadAll(outR)

	return string(outBytes), genErr
}

// fixtImgGeneration returns the fixture catalog, run inputs, and stub generator for one
// two-artifact image generation into a fresh directory.
//
// Test class: Core: Helper.
func fixtImgGeneration(t *testing.T) (*catalog.Catalog, RunFlags, *stubGen, string) {
	t.Helper()

	loadedCatalog := fixtureCatalog(t, fixtSource(t, "fixt", "Fixture", "FIXT_KEY",
		`{"id": "fixt-img", "name": "fixt-img", "media": "image", "params": []}`))
	outDir := t.TempDir()
	genInputs := RunFlags{
		Model:   params.GetSetIf(true, "fixt-img"),
		Prompt:  params.GetSetIf(true, "two words"),
		OutPath: params.GetSetIf(true, outDir+string(os.PathSeparator)),
	}
	gen := &stubGen{test: t, result: generation.Result{Artifacts: []artifact.Media{
		{Data: []byte("img-one"), FileExt: ".png"},
		{Data: []byte("img-two"), FileExt: ".png"},
	}}}

	return loadedCatalog, genInputs, gen, outDir
}

// keyCatalog loads provider prov-key with one image model, model-key. The provider declares
// BILD_TEST_CATALOG_KEY as its credential environment variable.
func keyCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()

	configDocument := fmt.Sprintf(`{
	  "id": "prov-key", "displayName": "Key Fixture", "apiKeyEnvVar": "BILD_TEST_CATALOG_KEY",
	  "models": [{"id": "model-key", "name": "model-key", "media": "image"}],
	  "config": {"imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": %q, "inputMediaListProvParam": "images",
	    "fallbackExt": ".png"
	  }}
	}`, catalog.InputMediaSingle)

	return loadFixtureCatalog(t, catalog.Source{ProviderID: "prov-key", ConfigBytes: []byte(configDocument)})
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
	}`, providerID, displayName, keyVar, modelID, defaultField, catalog.InputMediaSingle)
}
