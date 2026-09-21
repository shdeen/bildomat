package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
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
	"github.com/urfave/cli/v3"
)

// Invariants tested:
//
//  1. CLI list catalog document: The `list --json` command emits only the selected catalog data.
//     The default and `--models` forms include every selected provider and model; `--providers`
//     omits models; media filters omit providers that have no matching model; and model records do
//     not include parameter definitions.
//  2. CLI search catalog document: The `search --json` command emits only matching models in the
//     same catalog structure as `list --json`, and its text output identifies the same model set.
//  3. CLI info catalog document: The `info --json` command emits only the named provider or model.
//     Included models carry their public parameter definitions and required markers but no
//     provider request keys. The document contains the referenced flag definitions, honors media
//     filters, and omits provider request configuration.
//  4. JSON flag availability: Root generation, `list`, and `info` expose both JSON flag forms while
//     `help` remains text-only.
//  5. CLI info dispatch: The `info` command accepts a provider identifier and every model specifier form
//     represented by the current catalog.
//  6. CLI generation JSON failure: A failed generation emits one JSON document at process end with
//     submitted user-facing flags, adjustments made before submission, timing, and the user-facing
//     error while suppressing regular output.
//  7. Print filename relative output path: A relative output directory supplied through
//     `--output-path` still yields absolute path lines that resolve to the saved files.
//  8. CLI print filename flag: The `--print-filename` flag parses, and a failed run saves no
//     artifact and writes nothing to stdout.
//  9. CLI save results flag: The `--save-results` flag parses, the results file is created when
//     the results flags apply, and stdout stays empty.
//  10. Help lists output flags: The general help page offers the two results flags.
//  11. CLI search matching: A provider identifier matches that provider's models and any aggregator
//     model containing the identifier as a token. The leading characters of a bare model
//     identifier's final token find that model, and regular expressions match with their declared
//     syntax.
//  12. CLI help: Every root help form, given alone, writes the general help page only to stdout
//     and exits successfully; the forms render the same page apart from the tips section, whose
//     examples are drawn at random per render.
//  13. CLI list filter flags: The `list` command accepts each filter in both its short and long
//     form.
//  14. CLI list models filter: The `--models` filter renders fully qualified model keys in
//     alphabetical provider-name order with aggregators last, then model-ID order.
//  15. CLI list filter combinations: The providers and models filters and the media filters are
//     inclusive. Each pair together renders what neither alone does, and the media filters
//     partition the directory.
//  16. CLI search listing: The `search` command renders matching models as the nested catalog by
//     default, as provider identifiers under `-p`, and as a flat model directory under `-m`.
//     Media filters narrow those results, and an unmatched search prints nothing.
//  17. CLI info standard provider page: A provider under the summary threshold renders the standard
//     page with its display name first, followed by its identifier, API key environment variable,
//     and every model's bare identifier, and carries no usage examples.
//  18. CLI info aggregator summary: A provider marked as an aggregator renders a summary instead of
//     a model roster. The summary names each displayed medium's leading vendor and one example model
//     key, including when the image filter is active, while JSON retains the complete hierarchy.
//  19. CLI info model card: A model card begins with the provider and includes the model name, bare identifier,
//     aliases, fully qualified key, API key environment variable, and the provider's display name
//     and identifier on one line.
//  20. Generate uses configured default model: With `default-model` set in user configuration, no
//     `--model` flag, and no provider credentials, the missing-credential error names the
//     configured model's provider environment variable rather than the built-in default's. An
//     explicit `--model` selection makes the error name that model's provider variable instead.
//  21. Help shows configured default model: With `default-model` set in user configuration, the
//     help page's `--model` entry shows the configured model as its default, and a valid
//     configuration file produces no stderr output.
//  22. CLI output path help: The general help page explains how each `--output-path` spelling is
//     interpreted.
//  23. CLI version: Both root version forms write the command's version only to stdout and exit
//     successfully.
//  24. Reserved command dispatch: The `list`, `help`, and `info` command words dispatch to their
//     informational commands when used as the exact first positional argument; `info` without its
//     required identifier returns a usage error.
//  25. CLI help page structure: The general help page includes every informational command and
//     public flag derived from catalog data while excluding the private diagnostic flag.
//  26. CLI command help pages: An informational command serves its own help page without running
//     the command action.
//  27. CLI providers listing: The provider listing writes every built-in provider alphabetically, with aggregators last,
//     together with its declared models.
//  28. CLI list providers filter: The providers filter renders provider identities alone, with no
//     model of the catalog appearing.
//  29. CLI info provider page: The `info` command writes only the selected provider's declared
//     identity and models, reports an unknown provider as a generation failure, keeps successful
//     stderr empty, and keeps failed stdout empty.
//  30. CLI info model page: The `info` command resolves supported model specifiers and writes only
//     the selected model's declared provider, parameters, and values without exposing provider
//     request names. It reports an unknown model as a generation failure, keeps successful stderr
//     empty, and keeps failed stdout empty.
//  31. Complete CLI construction: The application built from the built-in constructor and catalog
//     returns the configured version through a complete CLI invocation.
//  32. CLI search command: The `search` command refuses a missing term and a second term, accepts its providers,
//     models, media, regular expression, and JSON flags, documents them on its help page, and
//     appears among the commands on the general help page.
//  33. Print filename output: With `--print-filename`, stdout carries exactly the saved file paths,
//     one per line in save order, and none of the ordinary generation report.
//  34. Saved results text: With `--save-results`, the ordinary stdout content, including the run
//     header and saved file reports, is written to the results file, and stdout stays empty.
//  35. Saved results with printed filenames: Using `--save-results` with `--print-filename` writes
//     the run header and saved file reports to the results file while stdout contains only absolute
//     paths to the two saved files.
//  36. Help and version flags alone: A help flag beside a positional argument, on any command, and
//     the root version flag beside a positional argument that is no command word, are usage
//     errors carrying the combination message naming `--help` or `--version`, whichever form was
//     typed. A help or version flag beside another flag, or typed twice, is a usage error without
//     that message. Each exits 2 with compact usage on stderr and an empty stdout, or, when
//     `--json` was parsed, as one JSON document on stdout. A command's own help flag alone, the
//     help command with a topic, the version flag alone, and a help flag followed by the `--`
//     terminator succeed; a help form after the `--` terminator or as a flag's value is not a
//     flag. A root flag ahead of a command word is the flags-before-command usage error naming
//     the command.
//  37. Provider default model: With `--model` omitted and no configured `default-model`, the run
//     uses the declared default model of the first provider in listing order that holds an API
//     key; the run header names that model. With no such provider, the run is a usage error
//     carrying the no-default-model message, exit 2, in text and as the JSON document's one
//     error, and nothing is generated.
//  38. Help shows the provider default: The `--model` entry of the general help page shows the
//     resolved provider default as its default when one resolves, and shows no default when none
//     does.
//  39. Prompt-ignored model: A run naming a model configured as ignoring the prompt needs no
//     prompt: with no positional argument or a blank one it goes on to its provider's credential
//     stop, and under `--json` the document's prompt is empty; a run of any other model without a
//     prompt stays the prompt-missing usage error.
//  40. Help tips: The general help page ends with a tips section whose examples draw from one of
//     the providers openai, google, and xai: exactly one of the provider's own models per medium
//     as example keys, each rendering as a `bild info` example, and a `bild search` example naming
//     the provider ID.
//  41. Search with an exclusion term: `-x`/`--exclude` takes the exclusion term as its value, in
//     every form the command line accepts and before or after the search term. A search with both
//     terms lists exactly the models the search term matches that the exclusion term does not, in
//     text and in JSON, with `--regex` applied to both terms and the result limit applied after the
//     exclusion. The token after `-x` is always its value. A search given no term, a term typed
//     empty, a second positional argument, `-x` with no value, and a broken exclusion pattern are
//     usage errors. The help page shows a value word for `--exclude`.
//  42. Credential configuration guidance: A missing credential identifies its environment
//     variable, user configuration key, and configuration file in each output mode.

// stubGen is a fixture generator: AdjustParams returns the configured
// adjusted values and records, and Generate returns the configured result.
type stubGen struct {
	test     testing.TB
	adjusted params.Values
	records  []params.Adjustment
	result   generation.Result
	err      error
}

// Tests for bildApp: its construction, the command surface it builds, the
// generation flow, model resolution, and the informational pages.

// The config keys the tests read.
const (
	keyID          = "id"
	keyMedia       = "media"
	keyModels      = "models"
	keyParams      = "params"
	keyFlags       = "flags"
	keyProviders   = "providers"
	keyConfig      = "config"
	keyParamID     = "paramID"
	keyFlagID      = "flagID"
	keyRequired    = "required"
	keyProviderKey = "/"
	keyDisplayName = "displayName"
	keyAggregator  = "aggregator"
)

// flagRecordsPath locates the flag records from the command package's directory.
//
//nolint:gochecknoglobals // a read-only fixture path, written only at package load.
var flagRecordsPath = filepath.Join("..", "..", "internal", "params", "config", "paramflags.json")

// TestCLIListCatalogDocument verifies invariant #1: CLI list catalog document.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `list --json` command emits only the selected
// catalog data. The default and `--models` forms include every selected provider and model;
// `--providers` omits models; media filters omit providers that have no matching model; and model
// records do not include parameter definitions.
//
// Test class: Core.
func TestCLIListCatalogDocument(t *testing.T) {
	clearProviderKeys(t)

	everyKey := keysOfMedia(t, false, false)
	imageKeys := keysOfMedia(t, true, false)
	videoKeys := keysOfMedia(t, false, true)
	everything := expectedListing(t, everyKey, true)

	checkDocument(t, everything, "list", "--json")
	checkDocument(t, everything, "list", "-j", "--models")
	checkDocument(t, expectedListing(t, everyKey, false), "list", "-j", "--providers")
	checkDocument(t, expectedListing(t, imageKeys, true), "list", "-j", "--image")
	checkDocument(t, expectedListing(t, videoKeys, true), "list", "-j", "-v")
	checkDocument(t, expectedListing(t, videoKeys, false), "list", "-j", "-p", "-v")

	if !t.Failed() {
		t.Log("✓ list --json is the catalog reduced to the selection")
	}
}

// TestCLISearchCatalogDocument verifies invariant #2: CLI search catalog document.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `search --json` command emits only matching models
// in the same catalog structure as `list --json`, and its text output identifies the same model set.
//
// Test class: Core.
func TestCLISearchCatalogDocument(t *testing.T) {
	clearProviderKeys(t)

	configs := shippedConfigDocuments(t)
	if len(configs) == 0 {
		t.Fatalf("💣 no shipped config")
	}

	term, _ := configs[0][keyID].(string)

	for _, searchArgs := range [][]string{{term}, {"--regex", "^(" + term + "|flux)$"}, {"-v", term}} {
		_, flat, _ := captureCLI(t, append([]string{"search", "-m"}, searchArgs...)...)
		matchedKeys := modelListings(t, flat)

		if len(matchedKeys) == 0 {
			t.Fatalf("💣 search -m %s matches nothing", strings.Join(searchArgs, " "))
		}

		checkDocument(t, expectedListing(t, keySet(t, matchedKeys...), true), append([]string{"search", "-j"}, searchArgs...)...)
	}

	if !t.Failed() {
		t.Log("✓ search --json is the catalog reduced to the matches")
	}
}

// TestCLIInfoCatalogDocument verifies invariant #3: CLI info catalog document.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `info --json` command emits only the named provider
// or model. Included models carry their public parameter definitions and required markers but no
// provider request keys. The document contains the referenced flag definitions, honors media filters,
// and omits provider request configuration.
//
// Test class: Core.
func TestCLIInfoCatalogDocument(t *testing.T) {
	clearProviderKeys(t)

	records := flagRecordDocuments(t)
	requiredSeen := false

	configs := shippedConfigDocuments(t)
	if len(configs) == 0 {
		t.Fatalf("💣 no shipped config")
	}

	for _, config := range configs {
		providerID, _ := config[keyID].(string)

		checkDocument(t, expectedInfo(t, config, keysOfMedia(t, false, false)), "info", "--json", providerID)

		for _, model := range jsonObjects(t, config, keyModels) {
			key := modelKey(t, config, model)

			providers := []any{reducedProvider(t, config, keySet(t, key), true, true)}
			expected := wantInfoDocument(t, providers, records)

			code, stdout, stderr := captureCLI(t, "info", "-j", key)
			if code != 0 || stderr != "" {
				t.Fatalf("💣 info -j %s: exit %d, stderr %q", key, code, stderr)
			}

			document := decodeJSONObject(t, stdout)
			checkNoInternalKeys(t, "info -j "+key, document)

			if !reflect.DeepEqual(document, expected) {
				t.Errorf("✗ info -j %s: document differs from the config's reduction\n--- got ---\n%s\n--- want ---\n%s", key, stdout, indentedJSON(t, expected))
			}

			for _, param := range jsonObjects(t, model, keyParams) {
				if param[keyRequired] == true {
					requiredSeen = true
				}
			}
		}
	}

	if !requiredSeen {
		t.Errorf("✗ no shipped config marks a param required, so the required mark is unproved")
	}

	imageProvider := configs[0]
	imageID, _ := imageProvider[keyID].(string)
	checkDocument(t, expectedInfo(t, imageProvider, keysOfMedia(t, true, false)), "info", "-j", "-i", imageID)

	if !t.Failed() {
		t.Log("✓ info --json is the catalog reduced to the named provider or model, with params and their flag records")
	}
}

// TestCLIJSONFlags verifies invariant #4: JSON flag availability.
//
// What makes it or breaks it:
// This test passes only when this rule holds: Root generation, `list`, and `info` expose both JSON
// flag forms while `help` remains text-only.
//
// Test class: Core.
func TestCLIJSONFlags(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"list", "--help"}, {"info", "--help"}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 0 || stderr != "" {
			t.Errorf("✗ %v: exit %d, stderr %q", args, code, stderr)
		}

		if !strings.Contains(stdout, "-j, --json") {
			t.Errorf("✗ %v: help lacks -j, --json", args)
		}
	}

	_, helpOutput, _ := captureCLI(t, "help", "help")
	if strings.Contains(helpOutput, "--json") {
		t.Errorf("✗ help command unexpectedly exposes JSON output: %q", helpOutput)
	}

	if !t.Failed() {
		t.Log("✓ root generation, list, and info expose -j/--json, while help remains text-only")
	}
}

// TestCLIInfoDispatch verifies invariant #5: CLI info dispatch.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `info` command accepts a provider identifier and
// every model specifier form represented by the current catalog.
//
// Test class: Core.
func TestCLIInfoDispatch(t *testing.T) {
	clearProviderKeys(t)
	loadedCatalog := shippedCatalog(t)

	modelDirectory := loadedCatalog.ModelDirectory()
	if len(modelDirectory) == 0 {
		t.Fatalf("💣 the shipped catalog has no models")
	}

	providerID := modelDirectory[0].Provider.ID

	code, stdout, _ := captureCLI(t, "info", providerID)
	if code != 0 {
		t.Errorf("✗ info %s: exit %d, want 0", providerID, code)
	}

	if stdout == "" {
		t.Errorf("✗ info %s rendered no provider details", providerID)
	}

	var uniqueModel catalog.ProvModelPair

	for _, modelPair := range modelDirectory {
		resolved, err := loadedCatalog.ResolveModelInput(modelPair.Model.ID)
		if err == nil && len(resolved) == 1 {
			uniqueModel = modelPair

			break
		}
	}

	if uniqueModel.Model.ID == "" {
		t.Fatalf("💣 the shipped catalog has no unambiguous bare model identifier")
	}

	modelInputs := []string{
		uniqueModel.Model.ID,
		uniqueModel.Provider.ID + "/" + uniqueModel.Model.ID,
	}
	for _, modelPair := range modelDirectory {
		if len(modelPair.Model.Aliases) > 0 {
			alias := modelPair.Model.Aliases[0]
			modelInputs = append(modelInputs, alias, modelPair.Provider.ID+"/"+alias)

			break
		}
	}

	for _, modelInput := range modelInputs {
		code, stdout, _ = captureCLI(t, "info", modelInput)
		if code != 0 {
			t.Errorf("✗ info %s: exit %d, want 0", modelInput, code)
		}

		if stdout == "" {
			t.Errorf("✗ info %s rendered no model details", modelInput)
		}
	}

	if !t.Failed() {
		t.Log("✓ info accepts provider IDs and the catalog's bare, qualified, bare-alias, and qualified-alias model forms")
	}
}

// TestCLIGenerationJSONFailure verifies invariant #6: CLI generation JSON failure.
//
// What makes it or breaks it:
// This test passes only when this rule holds: A failed generation emits one JSON document at process
// end with submitted user-facing flags, pre-submit adjustments, timing, and the user-facing error
// while suppressing regular output.
//
// Test class: Core.
func TestCLIGenerationJSONFailure(t *testing.T) {
	clearProviderKeys(t)

	videoModel := fixedDurationVideoModel(t, shippedCatalog(t))
	specifier := qualifiedSpecifier(t, videoModel)
	submittedDuration, snappedDuration := durationOutsideFixedSet(t, videoModel)

	code, stdout, stderr := captureCLI(t, "--json", "--model", specifier, "--duration", submittedDuration, "two words")
	if code != 1 {
		t.Errorf("✗ exit %d, want 1", code)
	}

	if stderr != "" {
		t.Errorf("✗ stderr = %q, want empty", stderr)
	}

	document := decodeJSONObject(t, stdout)
	if document["status"] != "failed" {
		t.Errorf("✗ status = %#v, want failed", document["status"])
	}

	if document["provider"] != videoModel.Provider.DisplayName || document["model"] != videoModel.Model.ID || document["prompt"] != "two words" {
		t.Errorf("✗ resolved request identity is incomplete: %#v", document)
	}

	timestamp, ok := document["timestamp"].(string)
	if !ok {
		t.Errorf("✗ timestamp = %#v, want a string", document["timestamp"])
	} else if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
		t.Errorf("✗ timestamp %q is not RFC3339: %v", timestamp, err)
	}

	if duration, ok := document["durationMs"].(float64); !ok || duration < 0 {
		t.Errorf("✗ durationMs = %#v, want a nonnegative number", document["durationMs"])
	}

	flags := jsonObjectField(t, document, "flags")
	if flags["model"] != specifier || fmt.Sprint(flags["duration"]) != submittedDuration {
		t.Errorf("✗ submitted flags = %#v, want model=%s and duration=%s", flags, specifier, submittedDuration)
	}

	adjustments := jsonArrayField(t, document, "adjustments")
	if len(adjustments) != 1 {
		t.Errorf("✗ adjustments = %#v, want one duration adjustment", adjustments)
	} else if adjustment, ok := adjustments[0].(map[string]any); !ok {
		t.Errorf("✗ adjustment = %#v, want an object", adjustments[0])
	} else if adjustment["flag"] != string(params.FlagTypeDuration) || adjustment["submitted"] != submittedDuration || adjustment["used"] != snappedDuration {
		t.Errorf("✗ duration adjustment = %#v, want submitted %s and used %s", adjustment, submittedDuration, snappedDuration)
	}

	errors := jsonArrayField(t, document, "errors")
	if len(errors) != 1 || !strings.Contains(fmt.Sprint(errors[0]), videoModel.Provider.APIKeyEnvVar) {
		t.Errorf("✗ errors = %#v, want the user-facing credential error", errors)
	}

	for _, requestKey := range requestKeysOutsideFlagIDs(t, shippedCatalog(t), &videoModel.Model) {
		if forbidden := `"` + requestKey + `":`; strings.Contains(stdout, forbidden) {
			t.Errorf("✗ JSON output contains a wire name %q: %s", forbidden, stdout)
		}
	}

	if !t.Failed() {
		t.Log("✓ failed generation JSON is one user-facing document with submitted flags and adjustments, and no regular output")
	}
}

// TestPrintFilenameRelativeOutputPath verifies invariant #7: Print filename relative output path.
//
// What makes it or breaks it:
// This test passes only when this rule holds: A relative output directory supplied through
// `--output-path` still yields absolute path lines that resolve to the saved files.
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

// TestCLIPrintFilenameFlag verifies invariant #8: CLI print filename flag.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `--print-filename` flag parses, and a failed run
// saves no artifact and writes nothing to stdout.
//
// Test class: Core.
func TestCLIPrintFilenameFlag(t *testing.T) {
	code, stdout, stderr := captureCLI(t, "--print-filename", "--model", "no-such-model-pf", "two words")
	if code != 1 {
		t.Errorf("✗ exit = %d, want 1 for the operational failure", code)
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty when nothing lands", stdout)
	}

	if !strings.Contains(stderr, "no-such-model-pf") {
		t.Errorf("✗ stderr lacks the failing model input: %q", stderr)
	}

	if !t.Failed() {
		t.Log("✓ --print-filename parses and keeps stdout empty on a failed run")
	}
}

// TestCLISaveResultsFlag verifies invariant #9: CLI save results flag.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `--save-results` flag parses, the results file is
// created when the flags apply, and stdout stays empty.
//
// Test class: Core.
func TestCLISaveResultsFlag(t *testing.T) {
	resultsPath := filepath.Join(t.TempDir(), "results.txt")

	code, stdout, stderr := captureCLI(t, "--save-results", resultsPath, "--model", "no-such-model-sr", "two words")
	if code != 1 {
		t.Errorf("✗ exit = %d, want 1 for the operational failure", code)
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty with a results path", stdout)
	}

	if !strings.Contains(stderr, "no-such-model-sr") {
		t.Errorf("✗ stderr lacks the failing model input: %q", stderr)
	}

	if _, statErr := os.Stat(resultsPath); statErr != nil {
		t.Errorf("✗ the results file was not created: %v", statErr)
	}

	if !t.Failed() {
		t.Log("✓ --save-results parses, creates its file, and keeps stdout empty")
	}
}

// TestHelpListsOutputFlags verifies invariant #10: Help lists output flags.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The general help page offers the two results
// flags.
//
// Test class: Core.
func TestHelpListsOutputFlags(t *testing.T) {
	usage := helpText(t)

	for _, flagName := range []string{"--print-filename", "--save-results"} {
		if !strings.Contains(usage, flagName) {
			t.Errorf("✗ the help page lacks %s", flagName)
		}
	}

	if !t.Failed() {
		t.Log("✓ the help page offers the results flags")
	}
}

// TestCLISearchMatching verifies invariant #11: CLI search matching.
//
// What makes it or breaks it:
// This test passes only when this rule holds: A provider identifier matches that provider's models and
// any aggregator model containing the identifier as a token. The leading characters of a bare model
// identifier's final token find that model, and regular expressions match with their declared syntax.
//
// Test class: Core.
func TestCLISearchMatching(t *testing.T) {
	clearProviderKeys(t)

	directory := shippedCatalog(t).ModelDirectory()
	if len(directory) == 0 {
		t.Fatalf("💣 the shipped catalog has no models")
	}

	providerID := directory[0].Provider.ID

	code, stdout, stderr := captureCLI(t, "search", "-m", providerID)
	if code != 0 || stderr != "" {
		t.Fatalf("💣 search -m %s: exit %d, stderr %q", providerID, code, stderr)
	}

	got := modelListings(t, stdout)

	for i := range directory {
		pair := &directory[i]
		key := pair.Provider.ID + "/" + pair.Model.ID
		tokenMatch := slices.ContainsFunc(strings.Split(key, "/"), func(token string) bool { return strings.HasPrefix(token, providerID) })

		if tokenMatch != slices.Contains(got, key) {
			t.Errorf("✗ search %s: %s listed=%t, want %t by its key tokens", providerID, key, slices.Contains(got, key), tokenMatch)
		}
	}

	for i := range directory {
		pair := &directory[i]
		tokens := strings.Split(pair.Model.ID, "/")
		lastToken := tokens[len(tokens)-1]
		term := lastToken[:min(3, len(lastToken))]

		code, stdout, stderr := captureCLI(t, "search", "-m", term)
		if code != 0 || stderr != "" {
			t.Errorf("✗ search -m %s: exit %d, stderr %q", term, code, stderr)

			continue
		}

		if key := pair.Provider.ID + "/" + pair.Model.ID; !slices.Contains(modelListings(t, stdout), key) {
			t.Errorf("✗ the term %q does not find %s", term, key)
		}
	}

	code, stdout, stderr = captureCLI(t, "search", "-m", "-r", "^"+providerID+"/")
	if code != 0 || stderr != "" {
		t.Fatalf("💣 search -m -r ^%s/: exit %d, stderr %q", providerID, code, stderr)
	}

	for _, key := range modelListings(t, stdout) {
		if !strings.HasPrefix(key, providerID+"/") {
			t.Errorf("✗ the anchored pattern matched %s", key)
		}
	}

	if !t.Failed() {
		t.Log("✓ the search matches key tokens and aliases, and a pattern matches as written")
	}
}

// TestCLIHelp verifies invariant #12: CLI help.
//
// What makes it or breaks it:
// This test passes only when this rule holds: Every root help form, given alone, writes the general
// help page only to stdout and exits successfully, and the forms render the same page up to the
// tips section, whose examples vary from render to render.
//
// Test class: Core.
func TestCLIHelp(t *testing.T) {
	usage := beforeTips(t, helpText(t))

	cases := [][]string{{"-h"}, {"--help"}, {"-help"}}
	for _, args := range cases {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 0 {
			t.Errorf("✗ %v: exit %d, want 0", args, code)
		}

		if page := beforeTips(t, stdout); page != usage {
			t.Errorf("✗ %v: stdout differs from the expected usage text before the tips (got %d bytes, want %d)", args, len(page), len(usage))
		}

		if stderr != "" {
			t.Errorf("✗ %v: stderr = %q, want empty", args, stderr)
		}
	}

	if !t.Failed() {
		t.Log("✓ all help forms print the usage on stdout and exit 0")
	}
}

// TestCLIListSelectorFlags verifies invariant #13: CLI list filter flags.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `list` command accepts each filter in both its
// short and long form.
//
// Test class: Core.
func TestCLIListSelectorFlags(t *testing.T) {
	clearProviderKeys(t)

	for _, selector := range []string{"-p", "--providers", "-m", "--models", "-i", "--image", "-v", "--video"} {
		code, stdout, stderr := captureCLI(t, "list", selector)
		if code != 0 {
			t.Errorf("✗ list %s: exit %d, want 0", selector, code)
		}

		if stderr != "" {
			t.Errorf("✗ list %s: stderr = %q, want empty", selector, stderr)
		}

		if stdout == "" {
			t.Errorf("✗ list %s: stdout empty, want a listing", selector)
		}
	}

	if !t.Failed() {
		t.Log("✓ the four filters carry both forms")
	}
}

// TestCLIListModelsFilter verifies invariant #14: CLI list models filter.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `--models` filter renders exactly the catalog's
// fully qualified model keys in alphabetical provider-name order with aggregators last,
// then model-ID order.
//
// Test class: Core.
func TestCLIListModelsFilter(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "list", "-m")
	if code != 0 {
		t.Errorf("✗ exit %d, want 0", code)
	}

	if stderr != "" {
		t.Errorf("✗ stderr = %q, want empty", stderr)
	}

	if got, want := modelListings(t, stdout), catalogModelKeys(t); !slices.Equal(got, want) {
		t.Errorf("✗ the directory holds %d keys, want the catalog's %d", len(got), len(want))
	}

	if !t.Failed() {
		t.Log("✓ the models filter is exactly the catalog's sorted fully qualified directory")
	}
}

// TestCLIListFilterCombinations verifies invariant #15: CLI list filter combinations.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The providers and models filters and the media filters
// are inclusive: each pair together renders what neither alone does, and the media filters partition
// the directory.
//
// Test class: Core.
func TestCLIListFilterCombinations(t *testing.T) {
	clearProviderKeys(t)

	_, nestedListing, _ := captureCLI(t, "list")

	for _, pair := range [][]string{{"-p", "-m"}, {"-i", "-v"}} {
		code, stdout, _ := captureCLI(t, "list", pair[0], pair[1])
		if code != 0 {
			t.Errorf("✗ list %s %s: exit %d, want 0", pair[0], pair[1], code)
		}

		if stdout != nestedListing {
			t.Errorf("✗ list %s %s does not render the default nested listing", pair[0], pair[1])
		}
	}

	_, imageKeys, _ := captureCLI(t, "list", "-m", "-i")
	_, videoKeys, _ := captureCLI(t, "list", "-m", "-v")

	if got, want := len(modelListings(t, imageKeys))+len(modelListings(t, videoKeys)), len(catalogModelKeys(t)); got != want {
		t.Errorf("✗ the media filters partition into %d keys, want the catalog's %d", got, want)
	}

	if !t.Failed() {
		t.Log("✓ the providers, models, and media filters are inclusive and compose")
	}
}

// TestCLISearchListing verifies invariant #16: CLI search listing.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `search` command renders matching models as the
// nested catalog by default, as provider identifiers under `-p`, and as a flat model directory under
// `-m`. Media filters narrow those results, and an unmatched search prints nothing.
//
// Test class: Core.
func TestCLISearchListing(t *testing.T) {
	clearProviderKeys(t)

	directory := shippedCatalog(t).ModelDirectory()
	if len(directory) == 0 {
		t.Fatalf("💣 the shipped catalog has no models")
	}

	providerID := directory[0].Provider.ID

	_, nested, _ := captureCLI(t, "search", providerID)
	_, providersOnly, _ := captureCLI(t, "search", "-p", providerID)
	_, flat, _ := captureCLI(t, "search", "-m", providerID)

	displayName := directory[0].Provider.DisplayName
	firstKey := providerID + "/" + directory[0].Model.ID

	if !strings.Contains(nested, displayName) || !strings.Contains(nested, directory[0].Model.ID) {
		t.Errorf("✗ the default search does not render the provider with its matching models:\n%s", nested)
	}

	if !strings.Contains(providersOnly, displayName) || strings.Contains(providersOnly, directory[0].Model.ID) {
		t.Errorf("✗ -p does not render the providers alone:\n%s", providersOnly)
	}

	if !slices.Contains(modelListings(t, flat), firstKey) || strings.Contains(flat, displayName) {
		t.Errorf("✗ -m does not render the keys alone:\n%s", flat)
	}

	_, videoOnly, _ := captureCLI(t, "search", "-m", "-v", "-x", "zzz-matches-no-model")

	for _, key := range modelListings(t, videoOnly) {
		for i := range directory {
			if directory[i].Provider.ID+"/"+directory[i].Model.ID == key && directory[i].Model.Media != media.Video {
				t.Errorf("✗ the video filter kept the image model %s", key)
			}
		}
	}

	code, stdout, stderr := captureCLI(t, "search", "no-such-token-anywhere")
	if code != 0 || stdout != "" || stderr != "" {
		t.Errorf("✗ a search with no match: exit %d, stdout %q, stderr %q; want 0 and nothing", code, stdout, stderr)
	}

	if !t.Failed() {
		t.Log("✓ the search renders list's forms over the matches and nothing for no match")
	}
}

// TestCLIInfoStandardProviderPage verifies invariant #17: CLI info standard provider page.
//
// What makes it or breaks it:
// This test passes only when this rule holds: A provider under the summary threshold renders the
// standard page: the display name first, the provider ID and API-key variable, every model by its bare
// ID, and footer examples naming the provider's own models, one per medium shown, under the video
// filter too.
//
// Test class: Core.
func TestCLIInfoStandardProviderPage(t *testing.T) {
	clearProviderKeys(t)

	shippedProvider := firstStandardProvider(t, shippedCatalog(t))

	code, stdout, stderr := captureCLI(t, "info", shippedProvider.ID)
	if code != 0 || stderr != "" {
		t.Fatalf("💣 info %s: exit %d, stderr %q", shippedProvider.ID, code, stderr)
	}

	if firstLine, _, _ := strings.Cut(stdout, "\n"); !strings.Contains(firstLine, shippedProvider.DisplayName) {
		t.Errorf("✗ the page's first line does not name the provider: %q", firstLine)
	}

	for _, want := range []string{shippedProvider.APIKeyEnvVar, shippedProvider.ID} {
		if !strings.Contains(stdout, want) {
			t.Errorf("✗ the page lacks %q", want)
		}
	}

	pageLines := strings.Split(stdout, "\n")

	for _, model := range shippedProvider.Models {
		if !slices.ContainsFunc(pageLines, func(pageLine string) bool { return pageLine == model.ID || strings.HasPrefix(pageLine, model.ID+" ") }) {
			t.Errorf("✗ the page does not list the bare model ID %s", model.ID)
		}
	}

	// Inverse assertion: proves the examples left the provider page; discarded at the fold.
	if keys := pageKeysOf(t, stdout, shippedProvider.ID); len(keys) != 0 {
		t.Errorf("✗ the page names the keys %v, want no usage example", keys)
	}

	code, stdout, stderr = captureCLI(t, "info", shippedProvider.ID, "-v")
	if code != 0 || stderr != "" {
		t.Fatalf("💣 info %s -v: exit %d, stderr %q", shippedProvider.ID, code, stderr)
	}

	if keys := pageKeysOf(t, stdout, shippedProvider.ID); len(keys) != 0 {
		t.Errorf("✗ the video-filtered page names the keys %v, want no usage example", keys)
	}

	if !t.Failed() {
		t.Log("✓ the standard provider page renders from the shipped config")
	}
}

// TestCLIInfoAggregatorSummary verifies invariant #18: CLI info aggregator summary.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The marked aggregator renders the summary: no roster
// beyond one example key per medium shown, under the image filter too, each medium's top vendor named,
// and an unchanged JSON hierarchy.
//
// Test class: Core.
func TestCLIInfoAggregatorSummary(t *testing.T) {
	clearProviderKeys(t)

	loadedCatalog := shippedCatalog(t)

	var aggregatorID string

	for _, pair := range loadedCatalog.ModelDirectory() {
		if pair.Provider.Aggregator {
			aggregatorID = pair.Provider.ID

			break
		}
	}

	if aggregatorID == "" {
		t.Fatalf("💣 no shipped provider is marked as an aggregator")
	}

	shippedProvider, _ := loadedCatalog.Provider(aggregatorID)

	code, stdout, stderr := captureCLI(t, "info", aggregatorID)
	if code != 0 || stderr != "" {
		t.Fatalf("💣 info %s: exit %d, stderr %q", aggregatorID, code, stderr)
	}

	checkProviderKeys(t, stdout, aggregatorID, shippedProvider.Models)

	for _, vendor := range topVendors(t, shippedProvider.Models) {
		if !strings.Contains(stdout, vendor) {
			t.Errorf("✗ the summary does not name the top vendor %s", vendor)
		}
	}

	code, stdout, stderr = captureCLI(t, "info", aggregatorID, "--image")
	if code != 0 || stderr != "" {
		t.Fatalf("💣 info %s --image: exit %d, stderr %q", aggregatorID, code, stderr)
	}

	checkProviderKeys(t, stdout, aggregatorID, modelsOfMedia(t, shippedProvider.Models, media.Image))

	code, stdout, stderr = captureCLI(t, "info", "--json", aggregatorID)
	if code != 0 || stderr != "" {
		t.Fatalf("💣 info --json %s: exit %d, stderr %q", aggregatorID, code, stderr)
	}

	providers := jsonArrayField(t, decodeJSONObject(t, stdout), "providers")
	if len(providers) != 1 {
		t.Fatalf("💣 the aggregator's JSON document carries %d providers, want 1", len(providers))
	}

	provider, _ := providers[0].(map[string]any)
	if models := jsonArrayField(t, provider, "models"); len(models) != len(shippedProvider.Models) {
		t.Errorf("✗ the aggregator's JSON document carries %d models, want every one of its %d", len(models), len(shippedProvider.Models))
	}

	if !t.Failed() {
		t.Log("✓ the aggregator renders the summary in text and the whole reduced catalog in JSON")
	}
}

// TestCLIInfoModelCard verifies invariant #19: CLI info model card.
//
// What makes it or breaks it:
// This test passes only when this rule holds: A model renders the card opening with its provider, and
// carrying its bare ID, its aliases, its provider's display name with the provider ID on one line, the
// API-key variable, and its fully qualified key.
//
// Test class: Core.
func TestCLIInfoModelCard(t *testing.T) {
	clearProviderKeys(t)

	var aliased catalog.ProvModelPair

	for _, pair := range shippedCatalog(t).ModelDirectory() {
		if len(pair.Model.Aliases) > 1 && pair.Model.Media == media.Image {
			aliased = pair

			break
		}
	}

	if aliased.Model.ID == "" {
		t.Fatalf("💣 the shipped catalog has no image model with several aliases")
	}

	key := aliased.Provider.ID + "/" + aliased.Model.ID

	code, stdout, stderr := captureCLI(t, "info", key)
	if code != 0 || stderr != "" {
		t.Fatalf("💣 info %s: exit %d, stderr %q", key, code, stderr)
	}

	cardLines := strings.Split(stdout, "\n")
	if len(cardLines) == 0 || !strings.Contains(cardLines[0], aliased.Provider.DisplayName) {
		t.Errorf("✗ the card's first line does not name the provider:\n%s", stdout)
	}

	for _, want := range []string{key, strings.Join(aliased.Model.Aliases, ", "), aliased.Provider.APIKeyEnvVar} {
		if !strings.Contains(stdout, want) {
			t.Errorf("✗ the card lacks %q:\n%s", want, stdout)
		}
	}

	holdsBoth := func(first, second string) bool {
		t.Helper()

		return slices.ContainsFunc(cardLines, func(cardLine string) bool {
			return strings.Contains(cardLine, first) && strings.Contains(cardLine, second)
		})
	}

	if !holdsBoth(aliased.Provider.DisplayName, aliased.Provider.ID) {
		t.Errorf("✗ the card does not name the provider with its ID on one line:\n%s", stdout)
	}

	if !slices.ContainsFunc(cardLines, func(cardLine string) bool { return strings.HasSuffix(cardLine, " "+aliased.Model.ID) }) {
		t.Errorf("✗ the card does not carry the bare model ID by itself:\n%s", stdout)
	}

	if !t.Failed() {
		t.Log("✓ the model card renders its identity block from the shipped config")
	}
}

// TestGenerateUsesConfiguredDefaultModel verifies invariant #20: Generate uses configured default model.
//
// What makes it or breaks it:
// This test passes only when this rule holds: With `default-model` set in user configuration, no
// `--model` flag, and no provider credentials, the missing-credential error names the configured
// model's provider environment variable rather than the built-in default's. An explicit `--model`
// selection makes the error name that model's provider variable instead.
//
// Test class: Core.
func TestGenerateUsesConfiguredDefaultModel(t *testing.T) {
	loadedCatalog := shippedCatalog(t)
	builtinKeyVar := builtinDefaultPair(t, loadedCatalog).Provider.APIKeyEnvVar
	configuredModel := modelOutsideKeyVar(t, loadedCatalog, builtinKeyVar)
	configuredKeyVar := configuredModel.Provider.APIKeyEnvVar

	setUserConfig(t, "default-model: "+qualifiedSpecifier(t, configuredModel)+"\n")
	clearProviderKeys(t)

	code, _, stderr := captureCLI(t, "two words")
	if code != 1 {
		t.Errorf("✗ exit = %d, want the operational stop 1 (stderr: %q)", code, stderr)
	}

	if !strings.Contains(stderr, configuredKeyVar) {
		t.Errorf("✗ the stop %q does not name the configured default's variable %s", stderr, configuredKeyVar)
	}

	if strings.Contains(stderr, builtinKeyVar) {
		t.Errorf("✗ the stop %q names the built-in default's variable %s", stderr, builtinKeyVar)
	}

	flaggedModel := modelOutsideKeyVar(t, loadedCatalog, configuredKeyVar)
	flaggedSpecifier := qualifiedSpecifier(t, flaggedModel)
	flaggedKeyVar := flaggedModel.Provider.APIKeyEnvVar

	t.Setenv(flaggedKeyVar, "")

	code, _, stderr = captureCLI(t, "--model", flaggedSpecifier, "two words")
	if code != 1 {
		t.Errorf("✗ flagged exit = %d, want the operational stop 1 (stderr: %q)", code, stderr)
	}

	if !strings.Contains(stderr, flaggedKeyVar) {
		t.Errorf("✗ the flagged stop %q does not name %s", stderr, flaggedKeyVar)
	}

	if strings.Contains(stderr, configuredKeyVar) {
		t.Errorf("✗ the flagged stop %q names the configured default's variable %s", stderr, configuredKeyVar)
	}

	if !t.Failed() {
		t.Log("✓ the configured default model governs the credential stop, and the flag outranks it")
	}
}

// TestHelpShowsConfiguredDefaultModel verifies invariant #21: Help shows configured default model.
//
// What makes it or breaks it:
// This test passes only when this rule holds: With `default-model` set in user configuration, the help
// page's `--model` entry shows the configured model as its default and a healthy config file produces
// no stderr output.
//
// Test class: Core.
func TestHelpShowsConfiguredDefaultModel(t *testing.T) {
	loadedCatalog := shippedCatalog(t)
	builtinKeyVar := builtinDefaultPair(t, loadedCatalog).Provider.APIKeyEnvVar
	configuredSpecifier := qualifiedSpecifier(t, modelOutsideKeyVar(t, loadedCatalog, builtinKeyVar))

	setUserConfig(t, "default-model: "+configuredSpecifier+"\n")

	code, stdout, stderr := captureCLI(t, "help")
	if code != 0 {
		t.Errorf("✗ exit = %d, want 0 (stderr: %q)", code, stderr)
	}

	if stderr != "" {
		t.Errorf("✗ a healthy config file produced stderr output: %q", stderr)
	}

	if !strings.Contains(stdout, fmt.Sprintf(FlagDefaultSuffix, configuredSpecifier)) {
		t.Errorf("✗ the help page does not show the configured model as the --model default:\n%s", stdout)
	}

	if !t.Failed() {
		t.Log("✓ the help page renders the configured default model in the --model entry")
	}
}

// TestCLIOutputPathHelp verifies invariant #22: CLI output path help.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The general help page explains how each `--output-path`
// spelling is interpreted.
//
// Test class: Core.
func TestCLIOutputPathHelp(t *testing.T) {
	clearProviderKeys(t)

	pageText := strings.Join(strings.Fields(helpText(t)), " ")

	wantText := strings.Join(strings.Fields(OutputPathFlagHelp), " ")
	if !strings.Contains(pageText, wantText) {
		t.Errorf("✗ the output-path entry does not explain the approved path rules")
	}

	if !t.Failed() {
		t.Log("✓ the output-path help explains directory, extension, existing-path, and filename-stem interpretation")
	}
}

// TestCLIVersion verifies invariant #23: CLI version.
//
// What makes it or breaks it:
// This test passes only when this rule holds: Both root version forms write the command's version only
// to stdout and exit successfully.
//
// Test class: Core: Incidental.
func TestCLIVersion(t *testing.T) {
	for _, args := range [][]string{{"-v"}, {"--version"}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 0 {
			t.Errorf("✗ %v: exit %d, want 0", args, code)
		}

		if stdout != versionText(t) {
			t.Errorf("✗ %v: stdout = %q, want %q", args, stdout, versionText(t))
		}

		if stderr != "" {
			t.Errorf("✗ %v: stderr = %q, want empty", args, stderr)
		}
	}

	if !t.Failed() {
		t.Log("✓ both version forms print the version line on stdout and exit 0")
	}
}

// TestCLIReservedExact verifies invariant #24: Reserved command dispatch.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `list`, `help`, and `info` command words dispatch to
// their informational commands when used as the exact first positional argument; `info` without its
// required identifier returns a usage error.
//
// Test class: Core: Incidental.
func TestCLIReservedExact(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "list")
	if code != 0 {
		t.Errorf("✗ bare list: exit %d, want 0", code)
	}

	if !strings.Contains(stdout, shippedCatalog(t).Providers[0].DisplayName) {
		t.Errorf("✗ bare list: stdout lacks the listing")
	}

	if stderr != "" {
		t.Errorf("✗ bare list: stderr = %q, want empty", stderr)
	}

	code, stdout, _ = captureCLI(t, "help")
	if code != 0 {
		t.Errorf("✗ bare help: exit %d, want 0", code)
	}

	if beforeTips(t, stdout) != beforeTips(t, helpText(t)) {
		t.Errorf("✗ bare help: the page differs from the --help page before the tips")
	}

	// info alone carries no argument, so it is a usage error, not a page.
	code, stdout, stderr = captureCLI(t, "info")
	if code == 0 {
		t.Errorf("✗ bare info: exit 0, want a failure (argument missing)")
	}

	if stdout != "" {
		t.Errorf("✗ bare info: stdout = %q, want empty", stdout)
	}

	if stderr == "" {
		t.Errorf("✗ bare info: stderr empty, want the error and usage")
	}

	if !t.Failed() {
		t.Log("✓ each reserved word alone dispatches as its command with its cataloged outcome")
	}
}

// TestCLIHelpPageStructure verifies invariant #25: CLI help page structure.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The general help page includes every informational
// command and public data-driven flag while excluding the private diagnostic flag.
//
// Test class: Core: Incidental.
func TestCLIHelpPageStructure(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "--help")
	if code != 0 {
		t.Errorf("✗ exit %d, want 0", code)
	}

	if stderr != "" {
		t.Errorf("✗ stderr = %q, want empty", stderr)
	}

	for _, want := range []string{
		"--aspect-ratio", "--input-media", "--model", "--output-path",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("✗ the help lacks %q", want)
		}
	}

	if strings.Contains(stdout, "--output-dir") || strings.Contains(stdout, "-O,") {
		t.Errorf("✗ the help still names the removed output-dir flag")
	}

	if got := helpCommandNames(t, stdout); !slices.Equal(got, []string{"list", "info", "search", "help"}) {
		t.Errorf("✗ the commands list is %v, want [list info search help]", got)
	}

	if !strings.Contains(stdout, shippedCatalog(t).Providers[0].DisplayName) {
		t.Errorf("✗ the help lacks the catalog-computed provider support indication")
	}

	if strings.Contains(stdout, "debug") {
		t.Errorf("✗ the help names the private debug flag")
	}

	if !t.Failed() {
		t.Log("✓ the general help renders the commands and the two data-driven flag sections")
	}
}

// TestCLICommandHelpPages verifies invariant #26: CLI command help pages.
//
// What makes it or breaks it:
// This test passes only when this rule holds: An informational command serves its own help page
// without running the command action.
//
// Test class: Core: Incidental.
func TestCLICommandHelpPages(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, _ := captureCLI(t, "list", "--help")
	if code != 0 {
		t.Errorf("✗ list --help: exit %d, want 0", code)
	}

	for _, want := range []string{"--providers", "--models", "--image", "--video", ListUsageForm} {
		if !strings.Contains(stdout, want) {
			t.Errorf("✗ the list page lacks %q", want)
		}
	}

	if strings.Contains(stdout, shippedCatalog(t).Providers[0].DisplayName) {
		t.Errorf("✗ list --help ran the listing instead of the help")
	}

	code, stdout, _ = captureCLI(t, "info", "--help")
	if code != 0 {
		t.Errorf("✗ info --help: exit %d, want 0", code)
	}

	if !strings.Contains(stdout, InfoUsageForm) {
		t.Errorf("✗ the info page lacks its usage line")
	}

	if !t.Failed() {
		t.Log("✓ each command serves its own help page")
	}
}

// TestCLIProvidersListing verifies invariant #27: CLI providers listing.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The provider listing writes every built-in provider in
// alphabetical display-name order, with aggregators last, together with its declared models.
//
// Test class: Core: Incidental.
func TestCLIProvidersListing(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "list")
	if code != 0 {
		t.Errorf("✗ exit %d, want 0", code)
	}

	if stderr != "" {
		t.Errorf("✗ stderr = %q, want empty", stderr)
	}

	providers := shippedConfigDocuments(t)
	slices.SortFunc(providers, compareProviderDocuments)

	previousPosition := -1

	for _, provider := range providers {
		providerName, _ := provider[keyDisplayName].(string)

		providerPosition := strings.Index(stdout, providerName)
		if providerPosition < 0 {
			t.Errorf("✗ the listing lacks %s", providerName)

			continue
		}

		if providerPosition < previousPosition {
			t.Errorf("✗ %s renders out of alphabetical provider order with aggregators last", providerName)
		}

		previousPosition = providerPosition
	}

	for _, provider := range providers {
		for _, model := range jsonObjects(t, provider, keyModels) {
			modelID, _ := model[keyID].(string)
			if !strings.Contains(stdout, modelID) {
				t.Errorf("✗ the listing lacks the model %s", modelID)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ the providers listing renders every provider alphabetically, with aggregators last, and its models")
	}
}

// TestCLIListProvidersFilter verifies invariant #28: CLI list providers filter.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The providers filter renders provider identities
// alone, with no model of the catalog appearing.
//
// Test class: Core: Incidental.
func TestCLIListProvidersFilter(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "list", "-p")
	if code != 0 {
		t.Errorf("✗ exit %d, want 0", code)
	}

	if stderr != "" {
		t.Errorf("✗ stderr = %q, want empty", stderr)
	}

	for _, provider := range shippedCatalog(t).Providers {
		if !strings.Contains(stdout, provider.DisplayName) {
			t.Errorf("✗ the providers filter lacks %s", provider.DisplayName)
		}
	}

	for _, modelKey := range catalogModelKeys(t) {
		modelID := modelKey[strings.Index(modelKey, "/")+1:]
		if strings.Contains(stdout, modelID) {
			t.Errorf("✗ the providers filter leaked the model %s", modelID)

			break
		}
	}

	if !t.Failed() {
		t.Log("✓ the providers filter renders provider identities and no models")
	}
}

// TestCLIInfoProviderPage verifies invariant #29: CLI info provider page.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `info` command writes only the selected provider's
// declared identity and models, reports an unknown provider as a generation failure, keeps successful
// stderr empty, and keeps failed stdout empty.
//
// Test class: Core: Incidental.
func TestCLIInfoProviderPage(t *testing.T) {
	clearProviderKeys(t)

	loadedCatalog := shippedCatalog(t)
	shippedProvider := firstStandardProvider(t, loadedCatalog)

	code, stdout, stderr := captureCLI(t, "info", shippedProvider.ID)
	if code != 0 {
		t.Errorf("✗ info %s: exit %d, want 0", shippedProvider.ID, code)
	}

	if stderr != "" {
		t.Errorf("✗ info %s: stderr = %q, want empty", shippedProvider.ID, stderr)
	}

	wants := make([]string, 0, 3+len(shippedProvider.Models))
	wants = append(wants, shippedProvider.DisplayName, shippedProvider.ID, shippedProvider.APIKeyEnvVar)

	for _, model := range shippedProvider.Models {
		wants = append(wants, model.ID)
	}

	for _, want := range wants {
		if !strings.Contains(stdout, want) {
			t.Errorf("✗ the provider view lacks %q", want)
		}
	}

	if leakedModelID := foreignModelID(t, loadedCatalog, &shippedProvider); strings.Contains(stdout, leakedModelID) {
		t.Errorf("✗ the provider view leaks another provider's model %s", leakedModelID)
	}

	code, stdout, stderr = captureCLI(t, "info", "bogus-prov")
	if code != 1 {
		t.Errorf("✗ info bogus-prov: exit %d, want 1", code)
	}

	// Provider matching continues silently to model resolution, so an unmatched argument is
	// reported once, by model resolution.
	if !strings.Contains(stderr, fmt.Sprintf(output.ModelUnknown, "bogus-prov")) {
		t.Errorf("✗ info bogus-prov: the unknown-specifier message is missing: %q", stderr)
	}

	if stdout != "" {
		t.Errorf("✗ info bogus-prov: stdout = %q, want empty", stdout)
	}

	if !t.Failed() {
		t.Log("✓ info renders one provider's information and reports an unmatched argument once")
	}
}

// TestCLIInfoModelPage verifies invariant #30: CLI info model page.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `info` command resolves supported model specifiers
// and writes only the selected model's declared provider, parameters, and values without exposing wire
// names, reports an unknown model as a generation failure, keeps successful stderr empty, and keeps
// failed stdout empty.
//
// Test class: Core: Incidental.
func TestCLIInfoModelPage(t *testing.T) {
	clearProviderKeys(t)

	loadedCatalog := shippedCatalog(t)

	fixedSetModel := fixedSetModelResolvingAlone(t, loadedCatalog)
	fixedSetWants := []string{fixedSetModel.Provider.ID, fixedSetModel.Model.ID}

	for _, param := range fixedSetModel.Model.Params {
		fixedSetWants = append(append(fixedSetWants, string(param.FlagID)), param.AllowedValues...)
	}

	durationModel := fixedDurationModelResolvingAlone(t, loadedCatalog)
	durationWants := append([]string{string(params.FlagTypeDuration)}, fixedDurationSet(t, &durationModel.Model)...)

	ruleModel := ruleDescribedModelResolvingAlone(t, loadedCatalog)

	requestKeyModel := modelWithHiddenRequestKey(t, loadedCatalog)
	hiddenRequestKey := requestKeyOutsideDeclaredText(t, loadedCatalog, requestKeyModel)

	var requestKeyWants []string
	for _, param := range fixedSetParams(t, &requestKeyModel.Model) {
		requestKeyWants = append(requestKeyWants, param.AllowedValues...)
	}

	views := []struct {
		specifier    string
		wants        []string
		wireSpelling string
	}{
		{fixedSetModel.Model.ID, fixedSetWants, ""},
		{durationModel.Model.ID, durationWants, ""},
		{ruleModel.Model.ID, ruleDescriptions(t, &ruleModel.Model), ""},
		{qualifiedSpecifier(t, requestKeyModel), requestKeyWants, hiddenRequestKey},
	}
	for _, view := range views {
		code, stdout, stderr := captureCLI(t, "info", view.specifier)
		if code != 0 {
			t.Errorf("✗ info %s: exit %d, want 0", view.specifier, code)
		}

		if stderr != "" {
			t.Errorf("✗ model %s: stderr = %q, want empty", view.specifier, stderr)
		}

		flatView := strings.Join(strings.Fields(stdout), " ")
		for _, want := range view.wants {
			if !strings.Contains(flatView, want) {
				t.Errorf("✗ the %s view lacks %q", view.specifier, want)
			}
		}

		if view.wireSpelling != "" && strings.Contains(stdout, view.wireSpelling) {
			t.Errorf("✗ the %s view renders the wire spelling %q", view.specifier, view.wireSpelling)
		}
	}

	code, stdout, stderr := captureCLI(t, "info", "no-such-model-xyz")
	if code != 1 {
		t.Errorf("✗ info no-such-model-xyz: exit %d, want 1", code)
	}

	if !strings.Contains(stderr, fmt.Sprintf(output.ModelUnknown, "no-such-model-xyz")) {
		t.Errorf("✗ the unknown-specifier message is missing: %q", stderr)
	}

	if stdout != "" {
		t.Errorf("✗ info no-such-model-xyz: stdout = %q, want empty", stdout)
	}

	if !t.Failed() {
		t.Log("✓ info resolves as the specifier does and renders the declared data")
	}
}

// TestRunCLIRealChain verifies invariant #31: Complete CLI construction.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The application built from the built-in constructor and
// catalog returns the configured version through a complete CLI invocation.
//
// Test class: Core: Incidental.
func TestRunCLIRealChain(t *testing.T) {
	code, stdout, stderr := captureCLI(t, "-v")
	if code != 0 {
		t.Errorf("✗ run(-v) over the real chain = exit %d, want 0", code)
	}

	if stdout != versionText(t) {
		t.Errorf("✗ run(-v) stdout = %q, want the exact version line %q", stdout, versionText(t))
	}

	if stderr != "" {
		t.Errorf("✗ run(-v) stderr = %q, want empty", stderr)
	}

	if !t.Failed() {
		t.Log("✓ the real construction chain creates and serves the bild identity")
	}
}

// TestCLISearchCommand verifies invariant #32: CLI search command.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `search` command refuses a missing term and a second term, accepts its
// providers, models, media, regular-expression, and JSON flags, documents them on its help page, and
// joins the general help page's commands.
//
// Test class: Core: Incidental.
func TestCLISearchCommand(t *testing.T) {
	clearProviderKeys(t)

	for _, args := range [][]string{{"search", "flux"}, {"search", "-r", "flux"}, {"search", "--regex", "flux"}, {"search", "-m", "-i", "flux"}, {"search", "-p", "-v", "flux"}, {"search", "--models", "--image", "flux"}} {
		code, _, stderr := captureCLI(t, args...)
		if code != 0 || stderr != "" {
			t.Errorf("✗ %v: exit %d, stderr %q, want 0 and empty", args, code, stderr)
		}
	}

	for _, args := range [][]string{{"search"}, {"search", "flux", "dream"}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: exit %d, stdout %q, stderr %q; want a usage error", args, code, stdout, stderr)
		}
	}

	_, helpPage, _ := captureCLI(t, "search", "--help")
	for _, flagName := range []string{"--providers", "--models", "--image", "--video", "--regex", "--json", "-r,"} {
		if !strings.Contains(helpPage, flagName) {
			t.Errorf("✗ the search help page lacks %s", flagName)
		}
	}

	_, mainHelp, _ := captureCLI(t, "--help")
	if !slices.Contains(helpCommandNames(t, mainHelp), "search") {
		t.Errorf("✗ the general help page does not list search")
	}

	_, listHelp, _ := captureCLI(t, "list", "--help")
	if strings.Contains(listHelp, "--search") || strings.Contains(listHelp, "--regex") {
		t.Errorf("✗ the list help page still documents a search flag")
	}

	if !t.Failed() {
		t.Log("✓ search refuses a missing or second term, carries its flags, and joins the command surface")
	}
}

// TestPrintFilenameLandedPathsOnly verifies invariant #33: Print filename output.
//
// What makes it or breaks it:
// This test passes only when this rule holds: With `--print-filename`, stdout carries exactly the
// saved file paths, one per line in the order they were saved, and none of the regular generation details.
//
// Test class: Core: Incidental.
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

// TestSaveResultsFileText verifies invariant #34: Saved results text.
//
// What makes it or breaks it:
// This test passes only when this rule holds: With `--save-results`, the regular stdout content—the
// run header and saved-artifact reports—is saved in the results file, and stdout stays empty.
//
// Test class: Core: Incidental.
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

// TestSaveResultsWithPrintFilename verifies invariant #35: Saved results with printed filenames.
//
// What makes it or breaks it:
// This test passes only when this rule holds: Using `--save-results` with `--print-filename` sends the
// run header and saved-artifact reports to the results file while stdout contains only absolute paths
// to the two saved files.
//
// Test class: Core: Incidental.
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

// TestHelpVersionFlagsAlone verifies invariant #36: Help and version flags alone.
//
// What makes it or breaks it:
// This test passes only when every rule below holds.
//   - A help flag beside a positional argument, in any of its forms and on any command, is a usage
//     error: exit 2, an empty stdout, and stderr carrying the combination message naming `--help`
//     with compact usage. The root version flag beside a positional argument that is no command
//     word is the same error naming `--version`.
//   - A help or version flag beside another flag, a help or version flag typed twice, and
//     `search -m -h` are usage errors: exit 2, an empty stdout, compact usage on stderr, and no
//     combination message.
//   - A help or version flag beside `--json` is a usage error printed as one JSON document on
//     stdout with an empty stderr.
//   - A command's own help flag alone, the help command with a topic, the version flag alone, the
//     help flag followed by the `--` terminator, and a command's `-v` succeed with exit 0, and
//     `-h --` prints the general help page.
//   - A root flag ahead of a command word is the flags-before-command usage error naming the
//     command, exit 2: with compact usage on stderr and an empty stdout, or, when `--json` is
//     among the flags, as one JSON document on stdout with an empty stderr.
//   - A help form after the `--` terminator or as a value-taking flag's value is not a flag: those
//     runs exit 1 on their own outcome with no combination message.
//
// Test class: Core.
func TestHelpVersionFlagsAlone(t *testing.T) {
	clearProviderKeys(t)

	combinationStart, _, _ := strings.Cut(HelpFlagCombinedForm, "%s")

	for _, c := range []struct {
		args      []string
		namedFlag string
	}{
		{[]string{"info", "google", "-h"}, "--help"},
		{[]string{"search", "veo", "--help"}, "--help"},
		{[]string{"a cat", "-h"}, "--help"},
		{[]string{"-h", "a cat"}, "--help"},
		{[]string{"-help", "a cat"}, "--help"},
		{[]string{"-h", "list"}, "--help"},
		{[]string{"-help", "list"}, "--help"},
		{[]string{"-v", "a cat"}, "--version"},
		{[]string{"a cat", "-v"}, "--version"},
	} {
		code, stdout, stderr := captureCLI(t, c.args...)
		if code != 2 || stdout != "" {
			t.Errorf("✗ %v: exit %d, stdout %q; want the usage error 2 with an empty stdout", c.args, code, stdout)
		}

		if !strings.Contains(stderr, fmt.Sprintf(HelpFlagCombinedForm, c.namedFlag)) || !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: stderr lacks the combination message naming %s with compact usage: %q", c.args, c.namedFlag, stderr)
		}
	}

	for _, args := range [][]string{
		{"list", "-h", "-p"},
		{"-h", "-v"},
		{"-v", "-h"},
		{"-version", "-m", "gemini"},
		{"--version", "-m", "gemini"},
		{"-h", "-h"},
		{"-h", "--help"},
		{"-v", "-v"},
		{"search", "-m", "-h"},
	} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 || stdout != "" {
			t.Errorf("✗ %v: exit %d, stdout %q; want the usage error 2 with an empty stdout", args, code, stdout)
		}

		if !strings.Contains(stderr, "Usage:") || strings.Contains(stderr, combinationStart) {
			t.Errorf("✗ %v: stderr = %q, want compact usage without the combination message", args, stderr)
		}
	}

	for _, args := range [][]string{{"--help", "--json"}, {"list", "-j", "-h"}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 || stderr != "" {
			t.Errorf("✗ %v: exit %d, stderr %q; want the usage error 2 with an empty stderr", args, code, stderr)
		}

		if documentErrors := jsonArrayField(t, decodeJSONObject(t, stdout), "errors"); len(documentErrors) != 1 {
			t.Errorf("✗ %v: errors = %#v, want one error in the JSON document", args, documentErrors)
		}
	}

	for _, args := range [][]string{{"list", "-h"}, {"help", "list"}, {"-v"}, {"list", "-v"}} {
		code, _, stderr := captureCLI(t, args...)
		if code != 0 {
			t.Errorf("✗ %v: exit %d, want 0 (stderr: %q)", args, code, stderr)
		}
	}

	code, stdout, stderr := captureCLI(t, "-h", "--")
	if code != 0 || stderr != "" || beforeTips(t, stdout) != beforeTips(t, helpText(t)) {
		t.Errorf("✗ [-h --]: exit %d, stderr %q; want the general help page on stdout and exit 0", code, stderr)
	}

	// A command's flags follow its word: a flag ahead of the word invokes the
	// root command, and the word is then out of place.
	for _, c := range []struct {
		args        []string
		commandWord string
	}{
		{[]string{"-v", "list"}, "list"},
		{[]string{"-m", "gemini", "info", "google"}, "info"},
	} {
		code, stdout, stderr := captureCLI(t, c.args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, fmt.Sprintf(FlagsBeforeCommandForm, c.commandWord)) || !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: exit %d, stdout %q, stderr %q; want the flags-before-command usage error naming %s", c.args, code, stdout, stderr, c.commandWord)
		}
	}

	for _, c := range []struct {
		args        []string
		commandWord string
	}{
		{[]string{"-j", "list", "-v"}, "list"},
		{[]string{"--json", "search", "-v", "veo"}, "search"},
	} {
		code, stdout, stderr := captureCLI(t, c.args...)
		if code != 2 || stderr != "" {
			t.Errorf("✗ %v: exit %d, stderr %q; want the usage error 2 with an empty stderr", c.args, code, stderr)
		}

		documentErrors := jsonArrayField(t, decodeJSONObject(t, stdout), "errors")
		if len(documentErrors) != 1 || fmt.Sprint(documentErrors[0]) != fmt.Sprintf(FlagsBeforeCommandForm, c.commandWord) {
			t.Errorf("✗ %v: errors = %#v, want the one flags-before-command message naming %s", c.args, documentErrors, c.commandWord)
		}
	}

	// A help form after the terminator is the prompt, and one given as a
	// flag's value is that value: neither is the help flag, so each run goes
	// on to its own outcome, never the combination error.
	specifier := qualifiedSpecifier(t, builtinDefaultPair(t, shippedCatalog(t)))
	for _, args := range [][]string{{"--model", specifier, "--", "-h"}, {"-m", "-h", "a cat"}} {
		code, _, stderr := captureCLI(t, args...)
		if code != 1 || strings.Contains(stderr, combinationStart) {
			t.Errorf("✗ %v: exit %d, stderr %q; want 1 with no combination error", args, code, stderr)
		}
	}

	if !t.Failed() {
		t.Log("✓ a help or version flag beside other input is a usage error, and alone it serves its page")
	}
}

// TestGenerateUsesProviderDefault verifies invariant #37: Provider default model.
//
// What makes it or breaks it:
// This test passes only when this rule holds: With `--model` omitted and no configured
// `default-model`, the run uses the declared default model of the first provider in listing order
// that holds an API key; the run header names that model. With no such provider, the run is a usage
// error carrying the no-default-model message, exit 2, in text and as the JSON document's one
// error, and nothing is generated.
//
// Test class: Core.
func TestGenerateUsesProviderDefault(t *testing.T) {
	loadedCatalog := shippedCatalog(t)
	clearProviderKeys(t)
	blockNetwork(t)

	first, second := firstTwoProvidersWithDefaults(t, loadedCatalog)

	// The second provider alone holds a key: its default is the run's model.
	setUserConfig(t, fmt.Sprintf("api-keys:\n  %s: placeholder-never-sent\n", second.ID))

	code, stdout, _ := captureCLI(t, "-o", t.TempDir()+"/", "two words")
	if code != 1 {
		t.Errorf("✗ one keyed provider: exit %d, want the request failure 1", code)
	}

	if !strings.Contains(stdout, fmt.Sprintf(output.ModelHeader, defaultModelOf(t, second).Name)) {
		t.Errorf("✗ one keyed provider: the header does not name %s's default model:\n%s", second.ID, stdout)
	}

	// Both hold keys: the first in listing order wins.
	setUserConfig(t, fmt.Sprintf("api-keys:\n  %s: placeholder-never-sent\n  %s: placeholder-never-sent\n", second.ID, first.ID))

	code, stdout, _ = captureCLI(t, "-o", t.TempDir()+"/", "two words")
	if code != 1 {
		t.Errorf("✗ two keyed providers: exit %d, want the request failure 1", code)
	}

	if !strings.Contains(stdout, fmt.Sprintf(output.ModelHeader, defaultModelOf(t, first).Name)) {
		t.Errorf("✗ two keyed providers: the header does not name %s's default model:\n%s", first.ID, stdout)
	}

	// No provider holds a key: the usage error names the remedy.
	setUserConfig(t, "{}\n")

	outDir := t.TempDir()

	code, stdout, stderr := captureCLI(t, "-o", outDir+"/", "two words")
	if code != 2 || stdout != "" || !strings.Contains(stderr, NoDefaultModel) || !strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ no keyed provider: exit %d, stdout %q, stderr %q; want 2, empty, and the no-default-model usage error", code, stdout, stderr)
	}

	if entries, err := os.ReadDir(outDir); err != nil || len(entries) != 0 {
		t.Errorf("✗ no keyed provider: the output directory is not empty: %v, %v", entries, err)
	}

	code, stdout, _ = captureCLI(t, "--json", "-o", outDir+"/", "two words")
	documentErrors, _ := decodeJSONObject(t, stdout)["errors"].([]any)

	if code != 2 || len(documentErrors) != 1 || documentErrors[0] != NoDefaultModel {
		t.Errorf("✗ no keyed provider under --json: exit %d, errors %v; want 2 and the no-default-model message alone", code, documentErrors)
	}

	if !t.Failed() {
		t.Log("✓ the first keyed provider's default model is the run's model, and no keyed provider is a usage error")
	}
}

// TestHelpShowsProviderDefault verifies invariant #38: Help shows the provider default.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The `--model` entry of the general help page shows the
// resolved provider default as its default when one resolves, and shows no default when none does.
//
// Test class: Core.
func TestHelpShowsProviderDefault(t *testing.T) {
	loadedCatalog := shippedCatalog(t)
	clearProviderKeys(t)

	_, second := firstTwoProvidersWithDefaults(t, loadedCatalog)
	setUserConfig(t, fmt.Sprintf("api-keys:\n  %s: placeholder-never-sent\n", second.ID))

	code, stdout, stderr := captureCLI(t, "help")
	if code != 0 || stderr != "" {
		t.Errorf("✗ help with one keyed provider: exit %d, stderr %q", code, stderr)
	}

	resolvedKey := second.ID + catalog.KeySeparator + second.DefaultModel
	if !strings.Contains(stdout, fmt.Sprintf(FlagDefaultSuffix, resolvedKey)) {
		t.Errorf("✗ the --model entry does not show %s as the default:\n%s", resolvedKey, stdout)
	}

	setUserConfig(t, "{}\n")

	code, stdout, stderr = captureCLI(t, "help")
	if code != 0 || stderr != "" {
		t.Errorf("✗ help with no keyed provider: exit %d, stderr %q", code, stderr)
	}

	if strings.Contains(stdout, strings.Split(FlagDefaultSuffix, "%s")[0]) {
		t.Errorf("✗ the --model entry shows a default with no keyed provider:\n%s", stdout)
	}

	if !t.Failed() {
		t.Log("✓ the help page shows the resolved provider default, and no default when none resolves")
	}
}

// TestPromptIgnoredModelNeedsNoPrompt verifies invariant #39: Prompt-ignored model.
//
// What makes it or breaks it:
// This test passes only when this rule holds: A run naming a model configured as ignoring the prompt
// needs no prompt: with no positional argument or a blank one it goes on to its provider's
// credential stop (exit 1, the stop naming the provider's key variable), and under `--json` the
// document's prompt is empty; a run of any other model without a prompt stays the prompt-missing
// usage error (exit 2).
//
// Test class: Core.
func TestPromptIgnoredModelNeedsNoPrompt(t *testing.T) {
	loadedCatalog := shippedCatalog(t)
	setUserConfig(t, "{}\n")
	clearProviderKeys(t)

	ignoring := promptIgnoredPair(t, loadedCatalog)
	specifier := qualifiedSpecifier(t, ignoring)
	inputImage := fixturePNG(t)

	for _, args := range [][]string{
		{"--model", specifier, "-i", inputImage},
		{"--model", specifier, "-i", inputImage, "   "},
	} {
		code, _, stderr := captureCLI(t, args...)
		if code != 1 || !strings.Contains(stderr, ignoring.Provider.APIKeyEnvVar) {
			t.Errorf("✗ %v: exit %d, stderr %q; want the credential stop naming %s", args, code, stderr, ignoring.Provider.APIKeyEnvVar)
		}
	}

	code, stdout, _ := captureCLI(t, "--json", "--model", specifier, "-i", inputImage)
	if document := decodeJSONObject(t, stdout); code != 1 || document["prompt"] != "" {
		t.Errorf("✗ --json without a prompt: exit %d, prompt %v; want 1 and an empty prompt", code, document["prompt"])
	}

	reading := builtinDefaultPair(t, loadedCatalog)
	if reading.Model.PromptIgnored {
		t.Fatalf("💣 the built-in default %s ignores the prompt; the control needs a model that reads one", reading.Model.ID)
	}

	code, _, stderr := captureCLI(t, "--model", qualifiedSpecifier(t, reading))
	if code != 2 || !strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ a prompt-reading model without a prompt: exit %d, stderr %q; want the prompt-missing usage error", code, stderr)
	}

	if !t.Failed() {
		t.Log("✓ a prompt-ignored model runs without a prompt, and every other model still requires one")
	}
}

// TestHelpTips verifies invariant #40: Help tips.
//
// What makes it or breaks it:
// This test passes only when this rule holds: The general help page ends with a tips section whose
// examples draw from one of the providers openai, google, and xai: exactly one of that provider's
// own models per medium as example keys, no key of the other two, each key rendering as a
// `bild info` example that succeeds, and a `bild search` example naming the provider ID.
//
// Test class: Core.
func TestHelpTips(t *testing.T) {
	loadedCatalog := shippedCatalog(t)
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("💣 --help: exit %d, stderr %q", code, stderr)
	}

	_, tips, found := strings.Cut(stdout, output.HelpTipsHeading)
	if !found {
		t.Fatalf("💣 the help page carries no tips section:\n%s", stdout)
	}

	var drawnFrom []string

	for _, providerID := range []string{"openai", "google", "xai"} {
		if len(pageKeysOf(t, tips, providerID)) > 0 {
			drawnFrom = append(drawnFrom, providerID)
		}
	}

	if len(drawnFrom) != 1 {
		t.Fatalf("💣 the tips draw from %v, want exactly one of openai, google, and xai:\n%s", drawnFrom, tips)
	}

	prov, ok := loadedCatalog.Provider(drawnFrom[0])
	if !ok {
		t.Fatalf("💣 the shipped catalog lacks %s", drawnFrom[0])
	}

	checkProviderKeys(t, tips, prov.ID, prov.Models)

	for _, key := range pageKeysOf(t, tips, prov.ID) {
		if !strings.Contains(tips, "bild info "+key) {
			t.Errorf("✗ the key %s renders as no info example", key)
		}

		if infoCode, _, infoStderr := captureCLI(t, "info", key); infoCode != 0 || infoStderr != "" {
			t.Errorf("✗ the example 'bild info %s' fails: exit %d, stderr %q", key, infoCode, infoStderr)
		}
	}

	if !strings.Contains(tips, "bild search "+prov.ID+"\n") {
		t.Errorf("✗ the tips carry no search example naming %s:\n%s", prov.ID, tips)
	}

	if !t.Failed() {
		t.Log("✓ the help tips draw their examples from one rotation provider")
	}
}

// TestCLISearchExclusionTerm verifies invariant #41: Search with an exclusion term.
//
// What makes it or breaks it:
// This test passes only when every rule below holds over the built-in catalog.
//   - `search -m openai -x openrouter` exits 0 and lists exactly the keys `openai` matches that
//     `openrouter` does not, and so do the forms with the flag ahead of the term, `--exclude
//     openrouter`, `--exclude=openrouter`, and `-x=openrouter`. The catalog must hold a model both
//     terms match, or the rule is unproven and the test fails.
//   - `search -m mini -r -x router` lists exactly the keys the expression `mini` matches that the
//     expression `router` does not. That set must differ from the one a prefix reading of `router`
//     gives, or the rule is unproven and the test fails.
//   - `search -j openai -x openrouter` prints one JSON document carrying exactly the first rule's
//     models.
//   - With more models than the result limit, `search -m -r . -x openrouter` exits 0 with every key
//     `openrouter` does not match, and `search -m -r . -x zzz-matches-no-model` exits 1 with the
//     too-many-results message. The catalog must make `search -m -r .` alone a refused search and
//     the first result fit the limit, or the rule is unproven and the test fails.
//   - `search -r flux -x (`, `search openai -x`, `search`, `search ”`, and `search -x ”` exit 2 with
//     compact usage on stderr and an empty stdout. The broken pattern's error names `(`. The empty
//     terms carry the `SearchTermEmpty` text and the missing term the `SearchTermMissing` text.
//   - `search -j -x ”` exits 2 with one JSON document whose one error is the `SearchTermEmpty` text.
//   - `search -m -x -h` exits 0 and lists every model key, because `-h` is the exclusion term.
//   - The `--exclude` entry of `search --help` shows a value word in angle brackets.
//
// Test class: Core.
func TestCLISearchExclusionTerm(t *testing.T) {
	clearProviderKeys(t)

	directory := shippedCatalog(t).ModelDirectory()
	keysMatching := func(term string, useRegexp bool) []string {
		t.Helper()

		matched := []string{}

		for i := range directory {
			matches, err := catalog.SearchDirectory(directory[i:i+1], term, "", useRegexp)
			if err != nil {
				t.Fatalf("💣 the single-model search for %q failed: %v", term, err)
			}

			if len(matches) == 1 {
				matched = append(matched, directory[i].Provider.ID+"/"+directory[i].Model.ID)
			}
		}

		slices.Sort(matched)

		return matched
	}
	without := func(keys, removedKeys []string) []string {
		return slices.DeleteFunc(slices.Clone(keys), func(key string) bool { return slices.Contains(removedKeys, key) })
	}
	listedKeys := func(stdout string) []string {
		keys := modelListings(t, stdout)
		slices.Sort(keys)

		return keys
	}

	openaiKeys, openrouterKeys := keysMatching("openai", false), keysMatching("openrouter", false)
	bothTermKeys := without(openaiKeys, openrouterKeys)

	if len(bothTermKeys) == len(openaiKeys) || len(bothTermKeys) == 0 {
		t.Fatalf("💣 the catalog holds no model that both openai and openrouter match, or none that only openai matches, so the rule is unproven")
	}

	for _, args := range [][]string{
		{"search", "-m", "openai", "-x", "openrouter"},
		{"search", "-m", "-x", "openrouter", "openai"},
		{"search", "-m", "openai", "--exclude", "openrouter"},
		{"search", "-m", "openai", "--exclude=openrouter"},
		{"search", "-m", "openai", "-x=openrouter"},
	} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 0 || stderr != "" || !slices.Equal(listedKeys(stdout), bothTermKeys) {
			t.Errorf("✗ %v: exit %d, stderr %q, keys %v; want 0, empty, and %v", args, code, stderr, listedKeys(stdout), bothTermKeys)
		}
	}

	regexKeys := without(keysMatching("mini", true), keysMatching("router", true))
	if slices.Equal(regexKeys, without(keysMatching("mini", true), keysMatching("router", false))) {
		t.Fatalf("💣 reading router as a prefix gives the same models as reading it as an expression, so the rule is unproven")
	}

	code, stdout, stderr := captureCLI(t, "search", "-m", "mini", "-r", "-x", "router")
	if code != 0 || stderr != "" || !slices.Equal(listedKeys(stdout), regexKeys) {
		t.Errorf("✗ [search -m mini -r -x router]: exit %d, stderr %q, keys %v; want 0, empty, and %v", code, stderr, listedKeys(stdout), regexKeys)
	}

	code, stdout, stderr = captureCLI(t, "search", "-j", "openai", "-x", "openrouter")
	if code != 0 || stderr != "" {
		t.Errorf("✗ [search -j openai -x openrouter]: exit %d, stderr %q; want 0 and empty", code, stderr)
	}

	documentKeys := []string{}

	for _, providerRecord := range jsonArrayField(t, decodeJSONObject(t, stdout), "providers") {
		providerFields, _ := providerRecord.(map[string]any)
		for _, modelRecord := range jsonArrayField(t, providerFields, "models") {
			modelFields, _ := modelRecord.(map[string]any)
			documentKeys = append(documentKeys, fmt.Sprint(providerFields["id"])+"/"+fmt.Sprint(modelFields["id"]))
		}
	}

	slices.Sort(documentKeys)

	if !slices.Equal(documentKeys, bothTermKeys) {
		t.Errorf("✗ the JSON document carries %v, want %v", documentKeys, bothTermKeys)
	}

	everyKey := keysMatching(".", true)
	pastLimitKeys := without(everyKey, openrouterKeys)

	if len(everyKey) <= catalog.SearchResultLimit || len(pastLimitKeys) > catalog.SearchResultLimit {
		t.Fatalf("💣 the catalog holds %d models and %d outside openrouter, so the limit rule is unproven", len(everyKey), len(pastLimitKeys))
	}

	if code, _, _ = captureCLI(t, "search", "-m", "-r", "."); code != 1 {
		t.Fatalf("💣 [search -m -r .] exited %d, want the refused search 1, so the limit rule is unproven", code)
	}

	code, stdout, stderr = captureCLI(t, "search", "-m", "-r", ".", "-x", "openrouter")
	if code != 0 || stderr != "" || !slices.Equal(listedKeys(stdout), pastLimitKeys) {
		t.Errorf("✗ [search -m -r . -x openrouter]: exit %d, stderr %q, %d keys; want 0, empty, and the %d keys outside openrouter", code, stderr, len(listedKeys(stdout)), len(pastLimitKeys))
	}

	code, stdout, stderr = captureCLI(t, "search", "-m", "-r", ".", "-x", "zzz-matches-no-model")
	if code != 1 || stdout != "" || !strings.Contains(stderr, output.SearchTooManyResults) {
		t.Errorf("✗ [search -m -r . -x zzz-matches-no-model]: exit %d, stdout %q, stderr %q; want 1 with the too-many-results message", code, stdout, stderr)
	}

	for _, usageCase := range []struct {
		args     []string
		wantText string
	}{
		{[]string{"search", "-r", "flux", "-x", "("}, `"("`},
		{[]string{"search", "openai", "-x"}, ""},
		{[]string{"search"}, SearchTermMissing},
		{[]string{"search", ""}, SearchTermEmpty},
		{[]string{"search", "-x", ""}, SearchTermEmpty},
	} {
		code, stdout, stderr := captureCLI(t, usageCase.args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, usageCase.wantText) {
			t.Errorf("✗ %q: exit %d, stdout %q, stderr %q; want the usage error 2 carrying %q", usageCase.args, code, stdout, stderr, usageCase.wantText)
		}
	}

	code, stdout, stderr = captureCLI(t, "search", "-j", "-x", "")
	if code != 2 || stderr != "" {
		t.Errorf("✗ [search -j -x '']: exit %d, stderr %q; want 2 and empty", code, stderr)
	}

	if documentErrors := jsonArrayField(t, decodeJSONObject(t, stdout), "errors"); len(documentErrors) != 1 || fmt.Sprint(documentErrors[0]) != SearchTermEmpty {
		t.Errorf("✗ [search -j -x '']: errors = %#v, want the one empty-term message", documentErrors)
	}

	code, stdout, stderr = captureCLI(t, "search", "-m", "-x", "-h")
	if code != 0 || stderr != "" || !slices.Equal(listedKeys(stdout), everyKey) {
		t.Errorf("✗ [search -m -x -h]: exit %d, stderr %q, %d keys; want 0, empty, and all %d model keys", code, stderr, len(listedKeys(stdout)), len(everyKey))
	}

	_, helpPage, _ := captureCLI(t, "search", "--help")
	if !regexp.MustCompile(`-x, --exclude <[^>]+>`).MatchString(helpPage) {
		t.Errorf("✗ the search help page shows no value word for --exclude:\n%s", helpPage)
	}

	if !t.Failed() {
		t.Log("✓ the exclusion term is a flag value, and both terms narrow the search together")
	}
}

// TestCLIAPIKeyConfigGuidance verifies invariant #42: Credential configuration guidance.
//
// What makes it or breaks it:
// A missing credential exits with generation failure and names its environment variable,
// user configuration key, and configuration file in text, JSON, and diagnostic output.
//
// Test class: Core.
func TestCLIAPIKeyConfigGuidance(t *testing.T) {
	setUserConfig(t, "{}\n")
	clearProviderKeys(t)
	modelPair := builtinDefaultPair(t, shippedCatalog(t))

	for _, outputFlag := range []string{"", "--json", "--debug"} {
		t.Run(outputFlag, func(t *testing.T) {
			t.Chdir(t.TempDir())

			arguments := []string{"--model", qualifiedSpecifier(t, modelPair)}
			if outputFlag != "" {
				arguments = append(arguments, outputFlag)
			}

			arguments = append(arguments, "a blue circle")

			exitCode, stdout, stderr := captureCLI(t, arguments...)
			if exitCode != 1 {
				t.Errorf("✗ missing credentials returned exit %d: %s%s", exitCode, stdout, stderr)
			}

			for _, setting := range []string{modelPair.Provider.APIKeyEnvVar, "api-keys." + modelPair.Provider.ID, "~/.bildomat/config.yml"} {
				if !strings.Contains(stdout+stderr, setting) {
					t.Errorf("✗ credential error omits %q: %s%s", setting, stdout, stderr)
				}
			}

			if !t.Failed() {
				t.Log("✓ the missing-credential error identifies the environment variable and config key")
			}
		})
	}
}

// TestMain points HOME at an empty scratch home directory for the whole
// package run, so no test reads the developer's real user config file. A
// config file at the real home would otherwise leak credentials, a default
// model, and an output directory into every invocation of the app boundary.
// A test that needs a config file builds its own scratch home through
// setUserConfig, whose per-test HOME override still applies.
//
// What makes it or breaks it:
// This package harness succeeds only if it creates and selects an isolated home directory, runs the
// package tests, removes that directory, and returns the package test exit code.
//
// Test role: Package harness.
func TestMain(m *testing.M) {
	scratchHome, err := os.MkdirTemp("", "bild-test-home-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "💣 scratch home creation failed:", err)
		os.Exit(1)
	}

	if err := os.Setenv("HOME", scratchHome); err != nil {
		fmt.Fprintln(os.Stderr, "💣 scratch home selection failed:", err)
		os.Exit(1)
	}

	exitCode := m.Run()
	_ = os.RemoveAll(scratchHome)

	os.Exit(exitCode)
}

// formPrefix takes a copy-catalog form and returns its text before the first
// placeholder: the fixed opening a rendered line carries whatever its values.
//
// Test class: Core: Helper.
func formPrefix(t *testing.T, form string) string {
	t.Helper()

	prefix, _, _ := strings.Cut(form, "%")

	return prefix
}

// realApp constructs the app exactly as the process does, failing the test on a
// construction defect.
//
// Test class: Core: Helper.
func realApp(t *testing.T) *bildApp {
	t.Helper()

	app := &bildApp{invocation: newInvocation(os.Stdout, os.Stderr)}
	app.invocation.outcome = &output.GenerationOutcome{}

	app.catalog = shippedCatalog(t)

	return app
}

// helpText captures the general help page the command renders for --help — the reference
// every help form and usage-printing error path is compared against.
// Test class: Core: Helper.
func helpText(t *testing.T) string {
	t.Helper()

	code, stdout, stderr := captureCLI(t, "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("💣 the --help reference capture failed: exit %d, stderr %q", code, stderr)
	}

	return stdout
}

// versionText renders the version line the library prints for --version.
// Test class: Core: Helper.
func versionText(t *testing.T) string {
	t.Helper()

	return "bild version " + appVersion() + "\n"
}

// captureCLI runs the command surface as the process does, over an empty application that the
// selected command's hook loads, with stdout and stderr captured.
//
// Test class: Core: Helper.
func captureCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()

	argv := append([]string{"bild"}, args...)

	origOut, origErr := os.Stdout, os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 stdout pipe: %v", err)
	}

	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 stderr pipe: %v", err)
	}

	os.Stdout, os.Stderr = outW, errW
	outCh := make(chan string, 1)
	errCh := make(chan string, 1)

	go func() { b, _ := io.ReadAll(outR); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(errR); errCh <- string(b) }()

	code = runExitCode(createCommand(&bildApp{}).Run(context.Background(), argv))
	os.Stdout, os.Stderr = origOut, origErr
	_ = outW.Close()
	_ = errW.Close()

	return code, <-outCh, <-errCh
}

// captureInteractiveCLI runs the app boundary with the supplied terminal input and captures
// stdout and stderr separately.
//
// Test class: Core: Helper.
func captureInteractiveCLI(t *testing.T, stdin string, tty bool, args ...string) (code int, stdout, stderr string) {
	t.Helper()

	argv := append([]string{"bild"}, args...)

	useStdin(t, stdin, tty)

	app := &bildApp{defaultOutDir: t.TempDir()}

	origOut, origErr := os.Stdout, os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 stdout pipe: %v", err)
	}

	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 stderr pipe: %v", err)
	}

	os.Stdout, os.Stderr = outW, errW
	outCh := make(chan string, 1)
	errCh := make(chan string, 1)

	go func() { b, _ := io.ReadAll(outR); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(errR); errCh <- string(b) }()

	code = runExitCode(createCommand(app).Run(context.Background(), argv))
	os.Stdout, os.Stderr = origOut, origErr
	_ = outW.Close()
	_ = errW.Close()

	return code, <-outCh, <-errCh
}

// decodeJSONObject decodes exactly one JSON document and fails when stdout contains a
// second value or non-JSON text.
//
// Test class: Core: Helper.
func decodeJSONObject(t *testing.T, stdout string) map[string]any {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(stdout))

	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("💣 stdout is not a JSON object: %v\n%s", err, stdout)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("💣 stdout contains data after its JSON document: %v", err)
	}

	return document
}

// jsonObjectField returns one object-valued JSON field.
//
// Test class: Core: Helper.
func jsonObjectField(t *testing.T, document map[string]any, fieldName string) map[string]any {
	t.Helper()

	field, ok := document[fieldName].(map[string]any)
	if !ok {
		t.Fatalf("💣 JSON field %q = %#v, want an object", fieldName, document[fieldName])
	}

	return field
}

// jsonArrayField returns one array-valued JSON field.
//
// Test class: Core: Helper.
func jsonArrayField(t *testing.T, document map[string]any, fieldName string) []any {
	t.Helper()

	field, ok := document[fieldName].([]any)
	if !ok {
		t.Fatalf("💣 JSON field %q = %#v, want an array", fieldName, document[fieldName])
	}

	return field
}

// func firstLine(s string) string {
// 	if i := strings.IndexByte(s, '\n'); i >= 0 {
// 		return s[:i]
// 	}

// 	return s
// }

// clearProviderKeys empties every registered provider's key environment
// variable for the test's lifetime. The provider set comes from the live
// registration list, never from a written-out list, so the helper covers
// every provider the program includes, present and future. With the package's
// scratch home holding no config file (see TestMain), a generation-shaped
// invocation then has no credential source left and deterministically stops
// at its missing-credential line, never reaching a live provider.
//
// Test class: Core: Helper.
func clearProviderKeys(t *testing.T) {
	t.Helper()

	paramFlags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(paramFlags, registeredTestSources(t, providerRegistrations())...)
	if err != nil {
		t.Fatalf("💣 catalog load failed: %v", err)
	}

	for i := range loadedCatalog.Providers {
		t.Setenv(loadedCatalog.Providers[i].APIKeyEnvVar, "")
	}
}

// setUserConfig points HOME at a scratch home directory holding a
// .bildomat/config.yml with the given content, and returns the file's path.
//
// Test class: Core: Helper.
func setUserConfig(t *testing.T, content string) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".bildomat")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("💣 config directory creation failed: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("💣 config file write failed: %v", err)
	}

	return configPath
}

// firstStandardProvider returns the first registered provider that is not an
// aggregator: the provider whose details page is the standard roster page.
//
// Test class: Core: Helper.
func firstStandardProvider(t *testing.T, loadedCatalog *catalog.Catalog) catalog.Provider {
	t.Helper()

	for i := range loadedCatalog.Providers {
		if !loadedCatalog.Providers[i].Aggregator {
			provider, _ := loadedCatalog.Provider(loadedCatalog.Providers[i].ID)

			return provider
		}
	}

	t.Fatalf("💣 the catalog holds no provider that is not an aggregator")

	return catalog.Provider{}
}

// promptIgnoredPair returns the first shipped model, in catalog order, that is
// configured as ignoring the prompt, with its provider's identity.
//
// Test class: Core: Helper.
func promptIgnoredPair(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if pair.Model.PromptIgnored {
			return pair
		}
	}

	t.Fatalf("💣 no shipped model is configured as ignoring the prompt")

	return catalog.ProvModelPair{}
}

// fixturePNG writes a small PNG image into a scratch directory and returns its path.
//
// Test class: Core: Helper.
func fixturePNG(t *testing.T) string {
	t.Helper()

	imagePath := filepath.Join(t.TempDir(), "fixture.png")

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatalf("💣 PNG encoding failed: %v", err)
	}

	if err := os.WriteFile(imagePath, encoded.Bytes(), 0o600); err != nil {
		t.Fatalf("💣 fixture image write failed: %v", err)
	}

	return imagePath
}

// builtinDefaultPair returns the declared default model of the first provider
// in listing order that declares one, as a provider-model pair.
//
// Test class: Core: Helper.
func builtinDefaultPair(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	first, _ := firstTwoProvidersWithDefaults(t, loadedCatalog)

	return catalog.ProvModelPair{Provider: first.Identity(), Model: defaultModelOf(t, first)}
}

// firstTwoProvidersWithDefaults returns the first two providers, in listing
// order, that declare a default model.
//
// Test class: Core: Helper.
func firstTwoProvidersWithDefaults(t *testing.T, loadedCatalog *catalog.Catalog) (catalog.Provider, catalog.Provider) {
	t.Helper()

	ordered := slices.Clone(loadedCatalog.Providers)
	slices.SortFunc(ordered, catalog.CompareProviders)

	var withDefaults []catalog.Provider

	for i := range ordered {
		if ordered[i].DefaultModel != "" {
			withDefaults = append(withDefaults, ordered[i])
		}
	}

	if len(withDefaults) < 2 {
		t.Fatalf("💣 fewer than two shipped providers declare a default model: %d", len(withDefaults))
	}

	return withDefaults[0], withDefaults[1]
}

// defaultModelOf returns the model the provider declares as its default.
//
// Test class: Core: Helper.
func defaultModelOf(t *testing.T, prov catalog.Provider) catalog.Model {
	t.Helper()

	for i := range prov.Models {
		if prov.Models[i].ID == prov.DefaultModel {
			return prov.Models[i]
		}
	}

	t.Fatalf("💣 provider %s declares the default model %q but no such model", prov.ID, prov.DefaultModel)

	return catalog.Model{}
}

// blockNetwork routes every HTTP and HTTPS request through an unreachable
// proxy so a run with a placeholder key fails at the request, never reaching
// a provider.
//
// Test class: Core: Helper.
func blockNetwork(t *testing.T) {
	t.Helper()
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
}

// bareIDResolvesAlone reports whether the pair's bare model ID resolves to
// that pair and no other.
//
// Test class: Core: Helper.
func bareIDResolvesAlone(t *testing.T, loadedCatalog *catalog.Catalog, pair catalog.ProvModelPair) bool {
	t.Helper()

	pairs, err := loadedCatalog.ResolveModelInput(pair.Model.ID)

	return err == nil && len(pairs) == 1 && pairs[0].Provider.ID == pair.Provider.ID
}

// qualifiedSpecifier returns the pair's fully qualified model specifier: the
// provider ID and the model ID joined by the key separator.
//
// Test class: Core: Helper.
func qualifiedSpecifier(t *testing.T, pair catalog.ProvModelPair) string {
	t.Helper()

	return pair.Provider.ID + catalog.KeySeparator + pair.Model.ID
}

// fixedDurationSet returns the allowed values of the model's duration
// parameter, or nil when the model declares no duration parameter with a
// fixed set.
//
// Test class: Core: Helper.
func fixedDurationSet(t *testing.T, model *catalog.Model) []string {
	t.Helper()

	durationParam, declared := model.Param(params.FlagTypeDuration)
	if !declared || len(durationParam.AllowedValues) == 0 {
		return nil
	}

	return durationParam.AllowedValues
}

// fixedDurationVideoModel returns the first video model whose duration
// parameter declares a fixed set of allowed values.
//
// Test class: Core: Helper.
func fixedDurationVideoModel(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if pair.Model.Media == media.Video && fixedDurationSet(t, &pair.Model) != nil {
			return pair
		}
	}

	t.Fatalf("💣 the shipped catalog holds no video model whose duration parameter declares a fixed set")

	return catalog.ProvModelPair{}
}

// durationOutsideFixedSet returns a duration one second above the largest
// value of the model's fixed duration set, and that largest value: the
// submitted duration and the value the snap to the nearest allowed value
// yields for it.
//
// Test class: Core: Helper.
func durationOutsideFixedSet(t *testing.T, pair catalog.ProvModelPair) (submittedDuration, snappedDuration string) {
	t.Helper()

	largestAllowed := 0

	for _, allowedValue := range fixedDurationSet(t, &pair.Model) {
		allowedSeconds, err := strconv.Atoi(allowedValue)
		if err != nil {
			t.Fatalf("💣 the duration set of %s holds the non-integer value %q", pair.Model.ID, allowedValue)
		}

		largestAllowed = max(largestAllowed, allowedSeconds)
	}

	if largestAllowed == 0 {
		t.Fatalf("💣 the model %s declares no fixed duration set", pair.Model.ID)
	}

	return strconv.Itoa(largestAllowed + 1), strconv.Itoa(largestAllowed)
}

// modelOutsideKeyVar returns the first model whose provider reads its API
// key from an environment variable other than the given one.
//
// Test class: Core: Helper.
func modelOutsideKeyVar(t *testing.T, loadedCatalog *catalog.Catalog, keyVar string) catalog.ProvModelPair {
	t.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if pair.Provider.APIKeyEnvVar != keyVar {
			return pair
		}
	}

	t.Fatalf("💣 the shipped catalog holds no model of a provider outside the key variable %s", keyVar)

	return catalog.ProvModelPair{}
}

// firstModelOfMedia returns the first model of the catalog, in catalog order,
// that produces the given medium.
//
// Test class: Core: Helper.
func firstModelOfMedia(test testing.TB, loadedCatalog *catalog.Catalog, media media.Kind) catalog.ProvModelPair {
	test.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if pair.Model.Media == media {
			return pair
		}
	}

	test.Fatalf("💣 the shipped catalog holds no %s model", media)

	return catalog.ProvModelPair{}
}

// fixedSetModelResolvingAlone returns the first model that resolves alone by
// its bare ID and declares a parameter with a fixed set of allowed values.
//
// Test class: Core: Helper.
func fixedSetModelResolvingAlone(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if bareIDResolvesAlone(t, loadedCatalog, pair) && fixedSetParams(t, &pair.Model) != nil {
			return pair
		}
	}

	t.Fatalf("💣 the shipped catalog holds no model resolving alone by its bare ID that declares a parameter with a fixed set")

	return catalog.ProvModelPair{}
}

// fixedDurationModelResolvingAlone returns the first model that resolves alone
// by its bare ID and whose duration parameter declares a fixed set.
//
// Test class: Core: Helper.
func fixedDurationModelResolvingAlone(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if bareIDResolvesAlone(t, loadedCatalog, pair) && fixedDurationSet(t, &pair.Model) != nil {
			return pair
		}
	}

	t.Fatalf("💣 the shipped catalog holds no model resolving alone by its bare ID whose duration parameter declares a fixed set")

	return catalog.ProvModelPair{}
}

// ruleDescribedModelResolvingAlone returns the first model that resolves alone
// by its bare ID and declares a parameter with a rule description.
//
// Test class: Core: Helper.
func ruleDescribedModelResolvingAlone(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if bareIDResolvesAlone(t, loadedCatalog, pair) && ruleDescriptions(t, &pair.Model) != nil {
			return pair
		}
	}

	t.Fatalf("💣 the shipped catalog holds no model resolving alone by its bare ID that declares a parameter with a rule description")

	return catalog.ProvModelPair{}
}

// modelWithHiddenRequestKey returns the first model that declares a parameter
// with a fixed set and a request key that no declared text spells.
//
// Test class: Core: Helper.
func modelWithHiddenRequestKey(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if requestKeyOutsideDeclaredText(t, loadedCatalog, pair) != "" && fixedSetParams(t, &pair.Model) != nil {
			return pair
		}
	}

	t.Fatalf("💣 the shipped catalog holds no model declaring a fixed set and a request key that no declared text spells")

	return catalog.ProvModelPair{}
}

// fixedSetParams returns the model's parameters that declare a fixed set of
// allowed values, or nil when none does.
//
// Test class: Core: Helper.
func fixedSetParams(t *testing.T, model *catalog.Model) []params.Definition {
	t.Helper()

	var parameterValues []params.Definition

	for _, param := range model.Params {
		if len(param.AllowedValues) > 0 {
			parameterValues = append(parameterValues, param)
		}
	}

	return parameterValues
}

// ruleDescriptions returns the rule descriptions the model's parameters
// declare, or nil when none declares one.
//
// Test class: Core: Helper.
func ruleDescriptions(t *testing.T, model *catalog.Model) []string {
	t.Helper()

	var descriptions []string

	for _, param := range model.Params {
		if param.RuleDescription != "" {
			descriptions = append(descriptions, param.RuleDescription)
		}
	}

	return descriptions
}

// declaredText returns every user-facing text a provider's details pages can
// draw from its configuration: the provider's identity, and each of its
// models' identity, aliases, description, and parameter texts and values.
//
// Test class: Core: Helper.
func declaredText(t *testing.T, provider *catalog.Provider) string {
	t.Helper()

	texts := []string{provider.ID, provider.DisplayName, provider.APIKeyEnvVar}

	for _, model := range provider.Models {
		texts = append(texts, model.ID, model.Name, model.Description)
		texts = append(texts, model.Aliases...)

		for _, param := range model.Params {
			texts = append(texts, string(param.FlagID), param.RuleDescription, param.ModelInfoComment)
			texts = append(texts, param.AllowedValues...)
		}
	}

	return strings.Join(texts, "\n")
}

// flagRecordText returns every user-facing text of the catalog's flag
// records: names, aliases, descriptions, comments, hints, and examples.
//
// Test class: Core: Helper.
func flagRecordText(t *testing.T, loadedCatalog *catalog.Catalog) string {
	t.Helper()

	var texts []string

	for _, flagRecord := range loadedCatalog.Flags {
		texts = append(texts, string(flagRecord.FlagID), flagRecord.FlagName, flagRecord.Description, flagRecord.Comment, flagRecord.TextHint)
		texts = append(texts, flagRecord.Aliases...)
		texts = append(texts, flagRecord.ExampleValues...)
	}

	return strings.Join(texts, "\n")
}

// foreignModelID returns a model ID of a provider other than the given one
// that no declared text of the given provider spells: a model whose ID on the
// given provider's page can only be a leak.
//
// Test class: Core: Helper.
func foreignModelID(t *testing.T, loadedCatalog *catalog.Catalog, provider *catalog.Provider) string {
	t.Helper()

	ownText := declaredText(t, provider)

	for i := range loadedCatalog.Providers {
		if loadedCatalog.Providers[i].ID == provider.ID {
			continue
		}

		for _, model := range loadedCatalog.Providers[i].Models {
			if !strings.Contains(ownText, model.ID) {
				return model.ID
			}
		}
	}

	t.Fatalf("💣 the shipped catalog holds no model outside %s whose ID its declared text does not spell", provider.ID)

	return ""
}

// requestKeyOutsideDeclaredText returns the first request key of the pair's
// model that neither the provider's declared text nor the flag records
// spell, or an empty string when every request key is spelled somewhere.
//
// Test class: Core: Helper.
func requestKeyOutsideDeclaredText(t *testing.T, loadedCatalog *catalog.Catalog, pair catalog.ProvModelPair) string {
	t.Helper()

	provider, registered := loadedCatalog.Provider(pair.Provider.ID)
	if !registered {
		t.Fatalf("💣 the catalog holds no provider %q for the pair", pair.Provider.ID)
	}

	spelledText := declaredText(t, &provider) + "\n" + flagRecordText(t, loadedCatalog)

	for _, param := range pair.Model.Params {
		if param.ParamID != "" && !strings.Contains(spelledText, param.ParamID) {
			return param.ParamID
		}
	}

	return ""
}

// requestKeysOutsideFlagIDs returns the model's request keys that are not
// also the ID of a flag record: the keys a user-facing document must not
// carry.
//
// Test class: Core: Helper.
func requestKeysOutsideFlagIDs(t *testing.T, loadedCatalog *catalog.Catalog, model *catalog.Model) []string {
	t.Helper()

	var requestKeys []string

	for _, param := range model.Params {
		isFlagID := slices.ContainsFunc(loadedCatalog.Flags, func(flagRecord params.Flag) bool { return string(flagRecord.FlagID) == param.ParamID })
		if param.ParamID != "" && !isFlagID {
			requestKeys = append(requestKeys, param.ParamID)
		}
	}

	return requestKeys
}

// --- the flag help entries: the invocation forms on their own line, the
// description on the line beneath ---

// namesLineIndex finds the help page line that introduces a flag by its short and long
// forms, returning -1 when no line does.
func namesLineIndex(test testing.TB, pageLines []string, shortForm, longName string) int {
	test.Helper()

	prefix := "-" + shortForm + ", --" + longName

	for i, line := range pageLines {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == prefix || strings.HasPrefix(trimmed, prefix+" ") {
			return i
		}
	}

	return -1
}

// leadingSpaces counts a line's indentation.
func leadingSpaces(test testing.TB, line string) int {
	test.Helper()

	return len(line) - len(strings.TrimLeft(line, " "))
}

// containedToken keeps a fuzzed token's filesystem reach inside the fuzz sandbox: absolute
// and home-anchored forms become relative and parent traversal collapses, so the generation
// arm's output-directory creation cannot leave the per-execution working directory. This is
// the fuzz input space's one deliberate constraint.
func containedToken(test testing.TB, token string) string {
	test.Helper()

	token = strings.TrimLeft(token, "/~")

	return strings.ReplaceAll(token, "..", ".")
}

// classifiedDispatchError reports whether a dispatch error belongs to the command
// boundary's documented classifications.
func classifiedDispatchError(test testing.TB, err error) bool {
	test.Helper()

	for _, root := range []error{
		errs.ErrCLI, errs.ErrModelResolve, errs.ErrKeyMissing,
		errs.ErrInputMedia, errs.ErrOutputFile,
	} {
		if errors.Is(err, root) {
			return true
		}
	}
	// The library reports an unusable command word with its own exit-coded
	// error, which the run boundary renders as a usage error.
	var libraryExit cli.ExitCoder

	return errors.As(err, &libraryExit)
}

// fixtureProviderCfg renders one minimal healthy descriptor config for the command-boundary
// catalog fixtures.
func fixtureProviderCfg(test testing.TB, providerID, modelID string) string {
	test.Helper()

	return fmt.Sprintf(`{
	  "id": %[1]q, "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
	  "models": [
	    {"id": %[2]q, "name": %[2]q, "media": "image", "params": [
	      {"paramID": "aspect_ratio", "flagID": "aspect-ratio"}
	    ]}
	  ],
	  "config": {"imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaListProvParam": "images", "inputMediaStyle": %[3]q,
	    "fallbackExt": ".png"
	  }}
	}`, providerID, modelID, catalog.InputMediaSingle)
}

// quietCommand takes an application that already holds its catalog and returns its command
// surface with the library's own output discarded. It removes the hooks that load the
// application, because a hook would replace the application's catalog with the built-in one; the
// rules those hooks enforce are tested through captureCLI.
//
// Test class: Core: Helper.
func quietCommand(app *bildApp) *cli.Command {
	command := createCommand(app)
	command.Writer = io.Discard
	command.ErrWriter = io.Discard
	command.Before = nil

	for _, subCmd := range command.Commands {
		subCmd.Before = nil
	}

	return command
}

// fixtureApp creates an app over the given catalog with test terminal wiring, returning the app
// for direct command runs.
func fixtureApp(t *testing.T, loadedCatalog *catalog.Catalog) *bildApp {
	t.Helper()
	app := testApp(t, loadedCatalog)

	return app
}

// modelListings splits a listing's stdout into its non-empty lines.
// Test class: Core: Helper.
func modelListings(test testing.TB, stdout string) []string {
	test.Helper()

	var lines []string

	for ln := range strings.SplitSeq(stdout, "\n") {
		if ln != "" {
			lines = append(lines, ln)
		}
	}

	return lines
}

// testApp creates a real app, then rewires it onto the given catalog with test terminal
// wiring and a scratch default output directory.
func testApp(t *testing.T, loadedCatalog *catalog.Catalog) *bildApp {
	t.Helper()

	useStdin(t, "", false)

	app := &bildApp{invocation: newInvocation(os.Stdout, os.Stderr)}
	app.invocation.outcome = &output.GenerationOutcome{}

	app.catalog = loadedCatalog
	app.defaultOutDir = t.TempDir()

	return app
}

// useStdin replaces the process's standard input for the test with the given
// text: read from a raw-mode pseudo-terminal when tty is set, so the prompts
// ask and the bytes arrive unchanged, and from a pipe otherwise. The input
// ends after the text either way.
//
// Test class: Core: Helper.
func useStdin(test testing.TB, text string, tty bool) {
	test.Helper()

	originalStdin := os.Stdin

	test.Cleanup(func() { os.Stdin = originalStdin })

	if !tty {
		readEnd, writeEnd, err := os.Pipe()
		if err != nil {
			test.Fatalf("💣 stdin pipe: %v", err)
		}

		os.Stdin = readEnd
		written := make(chan struct{})

		// The write runs beside the test so text beyond the pipe's capacity
		// cannot block the setup; closing the read end first unblocks a writer
		// nothing has read, and the wait leaves no writer behind.
		go func() {
			defer close(written)

			_, _ = io.WriteString(writeEnd, text)
			_ = writeEnd.Close()
		}()

		test.Cleanup(func() {
			_ = readEnd.Close()

			<-written
		})

		return
	}

	master, slave := openPTY(test)
	os.Stdin = slave

	// The terminal is in raw mode, so the bytes arrive unchanged. Closing the
	// master ends the input with EOF, but it also discards whatever the
	// program has not read yet, so the writer closes it only once the input is
	// drained, or once the test ends. The writer alone touches the master; the
	// test closes the slave after the writer is done. A write the program
	// never reads blocks once the terminal's input queue is full, so the test
	// ends it with a write deadline before waiting for the writer.
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer func() { _ = master.Close() }()

		_, _ = io.WriteString(master, text)

		for {
			pending, open := inputPending(test, slave)
			if !open || pending == 0 {
				return
			}

			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
			}
		}
	}()

	test.Cleanup(func() {
		_ = master.SetWriteDeadline(time.Now())

		close(stop)
		<-done

		_ = slave.Close()
	})
}

// captureTerminalStdout runs fn with standard output on a pseudo-terminal and
// standard error discarded, and returns what the terminal received.
//
// Test class: Core: Helper.
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

	// The reader alone touches the master; closing the slave after fn ends
	// its reading with EOF, and it closes the master once the text is in hand.
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

// fixtureCatalog creates a catalog over the given fixture sources — each an engine-class
// config declared in the descriptor set so the set tie holds — failing the test on a
// construction defect.
func fixtureCatalog(t *testing.T, sources ...catalog.Source) *catalog.Catalog {
	t.Helper()

	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags, sources...)
	if err != nil {
		t.Fatalf("💣 fixture catalog construction failed: %v", err)
	}

	return loadedCatalog
}

// fixtSource renders one fixture provider's config source over the given model configs (a JSON
// fragment).
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
func testGenerate(test testing.TB, app *bildApp, generator generation.Generator, genInputs RunFlags, userInputs params.FlagInputs, printFilename bool, resultsFilePath string) error {
	test.Helper()

	app.invocation = newInvocation(os.Stdout, os.Stderr)
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
func (s *stubGen) AdjustParams(_ *catalog.Model, _ params.FlagInputs, inputs []media.Input, _ *metadata.Reuse) (generation.Preparation, error) {
	s.test.Helper()

	return generation.Preparation{Params: s.adjusted, Changes: s.records, InputMedia: inputs}, nil
}

// Generate returns the fixture's configured result and error.
func (s *stubGen) Generate(_ context.Context, request *generation.Generation) (generation.Result, error) {
	s.test.Helper()

	result := s.result
	result.Preparation = request.Clone()

	return result, s.err
}

// ambigCatalog creates a catalog with one bare id declared by two providers, so the
// bare-id search across the whole directory is ambiguous. (The construction-defect contracts the old
// collector boundary held — a broken config and a duplicate pair — live in catalog's
// catalog tests: isolation replaced the first-failure chain stop, per the cataloged
// config-isolation change.)
func ambigCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()

	return fixtureCatalog(t,
		fixtSource(t, "alpha", "Alpha", "ALPHA_KEY", `{"id": "dup-model", "name": "dup-model", "media": "image", "params": []}`),
		fixtSource(t, "beta", "Beta", "BETA_KEY", `{"id": "dup-model", "name": "dup-model", "media": "image", "params": []}`),
	)
}

// The reprompt boundary: the non-TTY path renders the candidates and returns
// the exit-1 error; the interactive path re-resolves a corrected model input
// and aborts on blank input or EOF.

// resolveWith runs the app's resolution over one stdin script, capturing the terminal
// output (the loop writes only to stderr, so the combined capture is exactly the former
// error-writer content).
func resolveWith(t *testing.T, loadedCatalog *catalog.Catalog, modelInput, stdin string, tty bool) (bind catalog.ProvModelPair, rendered string, err error) {
	t.Helper()
	app := testApp(t, loadedCatalog)
	useStdin(t, stdin, tty)
	rendered = captureBoth(t, func() {
		app.invocation = newInvocation(os.Stdout, os.Stderr)
		bind, err = app.resolveModelInput(modelInput)
	})

	return bind, rendered, err
}

// captureBoth runs fn with stdout AND stderr redirected into one pipe, so cross-stream
// ordering is observable.
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

// --- the command surface: an invocation whose first positional is exactly a
// reserved command word dispatches as that command; every other first
// positional is a prompt ---

// --- the general help page: the commands list, the two flag sections, and
// the data-driven flag entries ---

// --- per-command help pages and version dispatch on a command ---

// catalogModelKeys derives fully qualified model keys from the built-in configs,
// ordered by provider name with aggregators last, then by model identifier.
//
// Test class: Core: Helper.
func catalogModelKeys(t *testing.T) []string {
	t.Helper()

	document := expectedListing(t, keysOfMedia(t, false, false), true)

	var qualifiedModelIDs []string

	for _, provider := range jsonObjects(t, document, keyProviders) {
		for _, model := range jsonObjects(t, provider, keyModels) {
			qualifiedModelIDs = append(qualifiedModelIDs, modelKey(t, provider, model))
		}
	}

	return qualifiedModelIDs
}

// helpCommandNames returns the command names the help page's COMMANDS section
// lists, in page order and stripped of the alias comma.
//
// Test class: Core: Helper.
func helpCommandNames(test testing.TB, page string) []string {
	test.Helper()

	var names []string

	inside := false

	for pageLine := range strings.SplitSeq(page, "\n") {
		switch {
		case strings.HasPrefix(pageLine, "COMMANDS:"):
			inside = true
		case strings.HasPrefix(pageLine, "OPTIONS:"):
			inside = false
		case inside && strings.TrimSpace(pageLine) != "":
			names = append(names, strings.TrimSuffix(strings.Fields(pageLine)[0], ","))
		}
	}

	return names
}

// --- the generation arm: flag parsing, and the double-dash terminator
// putting a dash-led word through as the prompt ---

// captureGenerate runs testGenerate with standard output captured and standard
// error silenced, and returns the run error with the captured stdout. The read
// is synchronous after the write end closes, which the small capture volumes of
// these tests keep within the pipe buffer.
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

// fixtImgGeneration returns the fixture catalog, run inputs, and stub
// generator for one two-artifact image generation into a fresh directory.
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

// --- the search command and the three info pages over the built-in catalog,
// every expectation derived from the catalog the binary embeds ---

// modelsOfMedia takes models and a medium and returns the models of that medium.
//
// Test class: Core: Helper.
func modelsOfMedia(test testing.TB, models []catalog.Model, media media.Kind) []catalog.Model {
	test.Helper()

	var selected []catalog.Model

	for i := range models {
		if models[i].Media == media {
			selected = append(selected, models[i])
		}
	}

	return selected
}

// pageKeysOf takes a page and a provider ID and returns the distinct fully
// qualified keys of that provider the page names, in page order.
//
// Test class: Core: Helper.
func pageKeysOf(t *testing.T, page, providerID string) []string {
	t.Helper()

	var keys []string

	// A quoted pattern is a search example, not a key.
	for _, match := range regexp.MustCompile(`(^|[^'])(`+regexp.QuoteMeta(providerID)+`/[^\s,'"]+)`).FindAllStringSubmatch(page, -1) {
		key := strings.TrimSuffix(match[2], ".")
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}

	return keys
}

// beforeTips takes a general help page and returns the page up to its tips
// section, the part every help form renders alike.
//
// Test class: Core: Helper.
func beforeTips(t *testing.T, page string) string {
	t.Helper()

	before, _, found := strings.Cut(page, output.HelpTipsHeading)
	if !found {
		t.Fatalf("💣 the help page carries no tips section:\n%s", page)
	}

	return before
}

// checkProviderKeys asserts that every fully qualified key of the provider a
// page names belongs to the given models, and that the page names exactly
// one key per medium those models span.
//
// Test class: Core: Helper.
func checkProviderKeys(t *testing.T, page, providerID string, models []catalog.Model) {
	t.Helper()

	keys := pageKeysOf(t, page, providerID)

	for _, key := range keys {
		if !slices.ContainsFunc(models, func(model catalog.Model) bool { return providerID+"/"+model.ID == key }) {
			t.Errorf("✗ the page names %s, which is no model of %s", key, providerID)
		}
	}

	var observedMedia []media.Kind

	for _, key := range keys {
		for i := range models {
			if providerID+"/"+models[i].ID == key && !slices.Contains(observedMedia, models[i].Media) {
				observedMedia = append(observedMedia, models[i].Media)
			}
		}
	}

	var wantMedia []media.Kind

	for i := range models {
		if !slices.Contains(wantMedia, models[i].Media) {
			wantMedia = append(wantMedia, models[i].Media)
		}
	}

	if len(keys) != len(wantMedia) || len(observedMedia) != len(wantMedia) {
		t.Errorf("✗ the page names the keys %v, want exactly one per medium of %v", keys, wantMedia)
	}
}

// topVendors takes models and returns, for each medium with a model, the
// vendor declaring the most of that medium's models, ties broken by name; a
// model's vendor is the first slash-delimited token of its bare ID.
//
// Test class: Core: Helper.
func topVendors(test testing.TB, models []catalog.Model) []string {
	test.Helper()

	var vendors []string

	for _, media := range []media.Kind{media.Image, media.Video} {
		counts := map[string]int{}

		for i := range models {
			if models[i].Media == media {
				vendor, _, _ := strings.Cut(models[i].ID, "/")
				counts[vendor]++
			}
		}

		top := ""
		for vendor, count := range counts {
			if top == "" || count > counts[top] || (count == counts[top] && vendor < top) {
				top = vendor
			}
		}

		if top != "" {
			vendors = append(vendors, top)
		}
	}

	return vendors
}

// shippedConfigDocuments decodes every built-in provider config as JSON data, in catalog
// load order.
//
// Test class: Core: Helper.
func shippedConfigDocuments(t *testing.T) []map[string]any {
	t.Helper()

	sources := providerRegistrations()
	documents := make([]map[string]any, 0, len(sources))

	for _, source := range sources {
		var document map[string]any
		if err := json.Unmarshal(source.ConfigBytes, &document); err != nil {
			t.Fatalf("💣 %s config does not decode: %v", source.ProviderID, err)
		}

		documents = append(documents, document)
	}

	return documents
}

// flagRecordDocuments decodes the flag records as JSON data, in file order.
//
// Test class: Core: Helper.
func flagRecordDocuments(t *testing.T) []map[string]any {
	t.Helper()

	content, err := os.ReadFile(flagRecordsPath) //nolint:gosec // a fixed repository file, not user input
	if err != nil {
		t.Fatalf("💣 flag records: %v", err)
	}

	var records []map[string]any
	if err := json.Unmarshal(content, &records); err != nil {
		t.Fatalf("💣 flag records do not decode: %v", err)
	}

	return records
}

// withoutEmptyValues returns the value with every empty member removed at any depth: an
// empty string, false, null, an empty array, and an empty object. A number is never empty.
//
// Test class: Core: Helper.
func withoutEmptyValues(test testing.TB, value any) any {
	test.Helper()

	switch typed := value.(type) {
	case map[string]any:
		reduced := map[string]any{}

		for key, member := range typed {
			if kept, empty := reducedMember(test, member); !empty {
				reduced[key] = kept
			}
		}

		return reduced
	case []any:
		reduced := make([]any, 0, len(typed))
		for _, member := range typed {
			reduced = append(reduced, withoutEmptyValues(test, member))
		}

		return reduced
	}

	return value
}

// reducedMember returns a member with its empty values removed and whether the member
// itself is empty.
//
// Test class: Core: Helper.
func reducedMember(test testing.TB, member any) (any, bool) {
	test.Helper()

	switch typed := member.(type) {
	case nil:
		return nil, true
	case string:
		return typed, typed == ""
	case bool:
		return typed, !typed
	case []any:
		return withoutEmptyValues(test, typed), len(typed) == 0
	case map[string]any:
		reduced, _ := withoutEmptyValues(test, typed).(map[string]any)

		return reduced, len(reduced) == 0
	}

	return member, false
}

// jsonObjects returns the members of an array field as objects, or fails.
//
// Test class: Core: Helper.
func jsonObjects(t *testing.T, document map[string]any, field string) []map[string]any {
	t.Helper()

	members, _ := document[field].([]any)

	objects := make([]map[string]any, 0, len(members))
	for _, member := range members {
		object, ok := member.(map[string]any)
		if !ok {
			t.Fatalf("💣 %s member = %#v, want an object", field, member)
		}

		objects = append(objects, object)
	}

	return objects
}

// modelKey returns a model's fully qualified key under its provider's config.
//
// Test class: Core: Helper.
func modelKey(test testing.TB, config, model map[string]any) string {
	test.Helper()

	providerID, _ := config[keyID].(string)
	modelID, _ := model[keyID].(string)

	return providerID + keyProviderKey + modelID
}

// keysOfMedia returns the fully qualified keys of every built-in model whose medium is
// among the selected media; with neither selected, every model's key.
//
// Test class: Core: Helper.
func keysOfMedia(t *testing.T, imageSelected, videoSelected bool) map[string]bool {
	t.Helper()

	keys := map[string]bool{}

	for _, config := range shippedConfigDocuments(t) {
		for _, model := range jsonObjects(t, config, keyModels) {
			selected := (!imageSelected && !videoSelected) || (model[keyMedia] == "image" && imageSelected) || (model[keyMedia] == "video" && videoSelected)
			if selected {
				keys[modelKey(t, config, model)] = true
			}
		}
	}

	return keys
}

// keySet returns the keys as a set.
//
// Test class: Core: Helper.
func keySet(test testing.TB, keys ...string) map[string]bool {
	test.Helper()

	set := map[string]bool{}
	for _, key := range keys {
		set[key] = true
	}

	return set
}

// reducedModel returns a config model as the document carries it: without its params on a
// listing, with its params minus their provider request keys on an info page, and without
// empty values.
//
// Test class: Core: Helper.
func reducedModel(t *testing.T, model map[string]any, withParams bool) map[string]any {
	t.Helper()

	reduced := map[string]any{}

	for key, value := range model {
		if key != keyParams {
			reduced[key] = value
		}
	}

	if withParams {
		paramObjects := jsonObjects(t, model, keyParams)
		parameterValues := make([]any, 0, len(paramObjects))

		for _, param := range paramObjects {
			reducedParam := map[string]any{}

			for key, value := range param {
				if key != keyParamID {
					reducedParam[key] = value
				}
			}

			parameterValues = append(parameterValues, reducedParam)
		}

		reduced[keyParams] = parameterValues
	}

	result, _ := withoutEmptyValues(t, reduced).(map[string]any)

	return result
}

// reducedProvider returns a config provider as the document carries it: its identity, the
// models whose keys are kept where the document lists models, and neither its request
// settings nor any empty value. It returns nil when the provider keeps no model.
//
// Test class: Core: Helper.
func reducedProvider(t *testing.T, config map[string]any, keptKeys map[string]bool, withModels, withParams bool) map[string]any {
	t.Helper()

	var models []any

	configuredModels := jsonObjects(t, config, keyModels)
	if !withParams {
		slices.SortFunc(configuredModels, compareModelDocuments)
	}

	for _, model := range configuredModels {
		if keptKeys[modelKey(t, config, model)] {
			modelDocument := reducedModel(t, model, withParams)
			if !withParams {
				delete(modelDocument, "aliases")
			}

			models = append(models, modelDocument)
		}
	}

	if len(models) == 0 {
		return nil
	}

	reduced := map[string]any{}

	for key, value := range config {
		if key != keyModels && key != keyConfig {
			reduced[key] = value
		}
	}

	if withModels {
		reduced[keyModels] = models
	}

	providerID, _ := config[keyID].(string)
	reduced["apiKeyConfigKey"] = "api-keys." + providerID

	result, _ := withoutEmptyValues(t, reduced).(map[string]any)

	return result
}

// referencedFlags returns the flag records the providers' params reference, once each, in
// record order and without empty values.
//
// Test class: Core: Helper.
func referencedFlags(t *testing.T, providers []any, records []map[string]any) []any {
	t.Helper()

	referenced := map[any]bool{}

	for _, member := range providers {
		provider, _ := member.(map[string]any)
		for _, model := range jsonObjects(t, provider, keyModels) {
			for _, param := range jsonObjects(t, model, keyParams) {
				referenced[param[keyFlagID]] = true
			}
		}
	}

	var flags []any

	for _, record := range records {
		if referenced[record[keyFlagID]] {
			flags = append(flags, withoutEmptyValues(t, record))
		}
	}

	return flags
}

// expectedListing derives the listing document over every built-in config: the providers
// keeping a model among the kept keys, with those models where the listing includes
// models, and no params.
//
// Test class: Core: Helper.
func expectedListing(t *testing.T, keptKeys map[string]bool, withModels bool) map[string]any {
	t.Helper()

	var listedProviders []any

	providers := shippedConfigDocuments(t)
	slices.SortFunc(providers, compareProviderDocuments)

	for _, config := range providers {
		if provider := reducedProvider(t, config, keptKeys, withModels, false); provider != nil {
			listedProviders = append(listedProviders, provider)
		}
	}

	return map[string]any{keyProviders: listedProviders}
}

// compareProviderDocuments orders configured provider names alphabetically, with aggregators last.
//
// Test class: Core: Helper.
func compareProviderDocuments(firstProvider, secondProvider map[string]any) int {
	firstAggregator, _ := firstProvider[keyAggregator].(bool)

	secondAggregator, _ := secondProvider[keyAggregator].(bool)
	if firstAggregator != secondAggregator {
		if firstAggregator {
			return 1
		}

		return -1
	}

	firstName, _ := firstProvider[keyDisplayName].(string)
	secondName, _ := secondProvider[keyDisplayName].(string)

	return strings.Compare(strings.ToLower(firstName), strings.ToLower(secondName))
}

// compareModelDocuments orders configured model identifiers alphabetically.
//
// Test class: Core: Helper.
func compareModelDocuments(firstModel, secondModel map[string]any) int {
	firstID, _ := firstModel[keyID].(string)
	secondID, _ := secondModel[keyID].(string)

	return strings.Compare(firstID, secondID)
}

// expectedInfo derives the info document of one provider's config: the provider with its
// kept models carrying their params, and the flag records those params reference.
//
// Test class: Core: Helper.
func expectedInfo(t *testing.T, config map[string]any, keptKeys map[string]bool) map[string]any {
	t.Helper()

	provider := reducedProvider(t, config, keptKeys, true, true)
	if provider == nil {
		t.Fatalf("💣 the %v config keeps no selected model", config[keyID])
	}

	return wantInfoDocument(t, []any{provider}, flagRecordDocuments(t))
}

// wantInfoDocument assembles the expected info document over reduced
// providers: the providers beside the flag records their params reference.
// When the params reference no flag record, the flags entry is absent, as
// the document rule omits an empty value.
//
// Test class: Core: Helper.
func wantInfoDocument(t *testing.T, providers []any, records []map[string]any) map[string]any {
	t.Helper()

	document := map[string]any{keyProviders: providers}
	if flags := referencedFlags(t, providers, records); len(flags) != 0 {
		document[keyFlags] = flags
	}

	return document
}

// indentedJSON returns the value as indented JSON for a failure message.
//
// Test class: Core: Helper.
func indentedJSON(t *testing.T, value any) string {
	t.Helper()

	text, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("💣 encoding the expected document: %v", err)
	}

	return string(text)
}

// checkDocument runs the command and compares its one JSON document with the expected
// document as data.
//
// Test class: Core: Helper.
func checkDocument(t *testing.T, expected map[string]any, args ...string) {
	t.Helper()

	code, stdout, stderr := captureCLI(t, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("💣 bild %s: exit %d, stderr %q", strings.Join(args, " "), code, stderr)
	}

	document := decodeJSONObject(t, stdout)
	if reflect.DeepEqual(document, expected) {
		return
	}

	t.Errorf("✗ bild %s: document differs from the configs' reduction\n--- got ---\n%s\n--- want ---\n%s", strings.Join(args, " "), stdout, indentedJSON(t, expected))
}

// checkNoInternalKeys fails when any object in the document carries a provider request key
// or a provider's request settings.
//
// Test class: Core: Helper.
func checkNoInternalKeys(t *testing.T, label string, value any) {
	t.Helper()

	switch typed := value.(type) {
	case map[string]any:
		for key, member := range typed {
			if key == keyParamID || key == keyConfig {
				t.Errorf("✗ %s: the document carries the internal key %q", label, key)
			}

			checkNoInternalKeys(t, label, member)
		}
	case []any:
		for _, member := range typed {
			checkNoInternalKeys(t, label, member)
		}
	}
}

// shippedCatalog loads the real catalog over the built-in config sources — also the proof
// that the built-in sources validate into a catalog.
func shippedCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()

	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags, registeredTestSources(t, providerRegistrations())...)
	if err != nil {
		t.Fatalf("💣 catalog construction failed: %v", err)
	}

	return loadedCatalog
}
