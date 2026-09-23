package main

// Invariants tested:
//  1. CLI list catalog document: bild list --json must exit 0, leave stderr empty, and emit exactly
//     the selected catalog data. Default and --models output must include the selected models
//     without parameter definitions; --providers must omit models. Image and video filters must
//     include only matching models and providers that contain them.
//  2. CLI search catalog document: For the supplied provider term, regular expression, and
//     video-filtered term, bild search --json must exit 0 with empty stderr and emit exactly the
//     catalog records corresponding to the nonempty model set listed by search -m.
//  3. CLI info catalog document: For every built-in provider and model, bild info --json must exit
//     0 with empty stderr and emit exactly the selected public catalog records, including model
//     parameters and referenced flag definitions. Model documents must omit paramID and config
//     keys. The catalog must exercise a required parameter, and the image-filtered provider
//     document must include only image models.
//  4. JSON flag availability: Root, list, and info help must succeed without diagnostics and show
//     -j, --json. Help for the help command must omit --json.
//  5. CLI info dispatch: bild info must exit 0 and print nonempty output for the selected provider
//     ID, a uniquely resolving bare model ID, its provider-qualified form, and the available bare
//     and provider-qualified alias forms.
//  6. CLI generation JSON failure: With missing credentials and a duration requiring adjustment,
//     bild --json must emit one failed outcome with submitted flags, the duration adjustment,
//     request identity, valid timing fields, and one credential error, leaving stderr empty.
//  7. CLI print filename flag: Given an unknown model and --print-filename, bild must exit 1 with
//     empty stdout and name the model on stderr.
//  8. CLI save results flag: Given an unknown model and --save-results, bild must create the
//     results file, exit 1 with empty stdout, and name the model on stderr.
//  9. Help lists output flags: bild --help must succeed without diagnostics and include both
//     --print-filename and --save-results in its output.
//  10. CLI search matching: For the selected provider ID, bild search -m must list exactly the
//      models whose slash-delimited key tokens start with that ID. Searching the first three
//      characters of each model's last ID token must include that model. With a regular expression
//      anchored to the provider prefix, every listed key must use that prefix. All searches must
//      exit 0 with empty stderr.
//  11. CLI help: bild -h, bild --help, and bild -help must each exit 0, leave stderr empty, and
//      print the same general help page before the variable tips section.
//  12. CLI list filter flags: bild list must accept -p/--providers, -m/--models, -i/--image, and
//      -v/--video separately. Each invocation must exit 0, print a nonempty listing, and leave
//      stderr empty.
//  13. CLI list models filter: bild list -m must exit 0 with empty stderr and list exactly the
//      catalog's fully qualified model keys, ordered by provider display name with aggregators
//      last, then by model ID.
//  14. CLI list filter combinations: bild list with both provider/model filters or both media
//      filters must print the default nested listing. The image-only and video-only model counts
//      must sum to the catalog total.
//  15. CLI search listing: For a provider-ID term, bild search must show the selected provider and
//      model, search -p must show the provider without that model ID, and search -m must show the
//      model key without the provider display name. With -v, no listed catalog model may be an
//      image. A term matching no model must exit 0 and leave both streams empty.
//  16. CLI info standard provider page: For a standard provider, bild info must put its name on the
//      first line, include its ID and credential variable, and list every bare model ID. Neither
//      its full page nor its video-filtered page may contain provider-qualified example keys.
//  17. CLI info aggregator summary: For an aggregator, bild info must show one valid example key
//      per medium and each medium's largest vendor by model count. The image filter must restrict
//      examples to image models, while JSON must retain the full model count.
//  18. CLI info model card: For an image model with multiple aliases, bild info must exit 0 with
//      empty stderr. The card must name the provider on its first line, include the fully qualified
//      key, comma-separated aliases, and credential variable, place the provider name and ID
//      together on one line, and include a line ending with the bare model ID.
//  19. Generate uses configured default model: With credentials absent and default-model
//      configured, bild must exit 1 and name that model's provider credential variable in stderr,
//      omitting the built-in default's variable. An explicit --model must instead make stderr name
//      the selected model's credential variable and omit the configured default's variable.
//  20. Configured default in help: With default-model configured, bild help must exit 0 with empty
//      stderr and include the configured model in the formatted default suffix.
//  21. CLI output path help: bild --help must succeed without diagnostics and include the complete
//      OutputPathFlagHelp text when whitespace is normalized.
//  22. CLI version: bild -v and bild --version must each exit 0, leave stderr empty, and print
//      exactly the version line derived from appVersion.
//  23. Reserved command dispatch: bild list must exit 0, print a built-in provider name, and leave
//      stderr empty. bild help must exit 0 and match the general help page before tips. bild info
//      without an argument must fail, leave stdout empty, and write nonempty stderr.
//  24. CLI help page structure: bild --help must include the four checked public flags, the four
//      informational commands in order, and a provider name, while omitting the removed output-dir
//      flag and debug.
//  25. CLI command help pages: bild list --help must show its usage and filter flags without a
//      provider listing. bild info --help must show its usage. Both must exit 0.
//  26. CLI providers listing: bild list must exit 0 with empty stderr and include every built-in
//      provider name in alphabetical display-name order with aggregators last. Its output must
//      contain every declared model ID.
//  27. CLI list providers filter: bild list -p must exit 0 with empty stderr, include every
//      provider display name, and omit every catalog model ID.
//  28. CLI info provider page: bild info must render the selected provider's identity and model IDs
//      while omitting the chosen foreign model ID. An unknown identifier must exit 1 with empty
//      stdout and the unknown-model message on stderr.
//  29. CLI info model page: bild info must render the selected models' public parameter IDs,
//      values, and rule descriptions and omit the checked provider request key. An unknown model
//      must exit 1 with empty stdout and its error on stderr.
//  30. Complete CLI construction: Through production command construction, bild -v must exit 0,
//      print the exact version line, and leave stderr empty.
//  31. CLI search command: bild search must accept the tested filter and regex flags and reject
//      missing or surplus terms with usage. Its help must document its filters, regex, and JSON;
//      general help must list search; list help must omit search flags.
//  32. Provider default model: Without --model or configured default-model, bild must select the
//      eligible provider's default, print its model header, and exit 1 when the blocked request
//      fails. If two providers have keys, it must select the first in listing order. With no
//      eligible provider, text mode must exit 2 with empty stdout, usage and NoDefaultModel on
//      stderr, and an empty output directory. JSON mode must exit 2 with exactly one NoDefaultModel
//      error.
//  33. Help shows the provider default: With one eligible provider credential, bild help must exit
//      0 with empty stderr and show that provider's qualified default model in the default suffix.
//      With no credentials, help must still succeed without diagnostics and omit the default
//      suffix.
//  34. Help tips: bild --help must exit 0 with empty stderr and include a tips section drawing
//      model keys from exactly one of OpenAI, Google, and xAI. It must show one valid key per
//      represented medium, use each key in a successful bild info example, and include a bild
//      search example naming that provider.
//  35. Search with an exclusion term: bild search -m openai with -x openrouter must exit 0 with
//      empty stderr and list exactly the inclusion matches minus exclusion matches, across all
//      tested flag placements and assignment forms. The JSON form must return the same keys in one
//      document. With -r, both mini and router must act as regex terms and produce the
//      corresponding difference. Excluding openrouter from a regex matching every key must reduce
//      results below the limit and succeed; excluding an unmatched term must retain the limit
//      failure with exit 1 and empty stdout. Missing or empty terms, a missing exclusion value, and
//      an invalid exclusion regex must exit 2 with empty stdout and usage plus the relevant message
//      on stderr. JSON with an empty exclusion must emit one SearchTermEmpty error and empty
//      stderr. search -m -x -h must treat -h as the exclusion value and list every key
//      successfully. Search help must show an angle-bracketed value placeholder for --exclude.
//  36. Prompt-ignored model: For a model configured to ignore prompts, bild with no prompt or a
//      blank prompt must exit 1 and name its missing credential variable. Its JSON form must record
//      an empty prompt. For the control model that consumes a prompt, omitting the prompt must exit
//      2 and print usage.
//  37. Generation argument parsing: Given each multiword prompt beginning with providers, models,
//      provider, or model as one argument, bild --json must retain that full prompt, exit 1, and
//      omit PARAMETER FLAGS from stderr.
//  38. Help layout for flags: bild --help must render the checked root entries and each aliased
//      parameter flag with names indented four spaces and a separate description indented eight.
//      The aspect-ratio entry must follow a blank line, and input-media must omit duplicate
//      bracketed forms.
//  39. Visibility of diagnostic flags: bild --help must exit 0, leave stderr empty, and omit debug
//      from stdout.
//  40. Scope of prompt confirmation: Given nonterminal hellp or terminal two words, bild --json
//      must exit 1, retain the prompt, and omit the confirmation question.
//  41. Results flags after cancellation: Given a terminal reply of n to prompt hellp, bild
//      --print-filename must exit 0 and leave stdout empty. With --json and --save-results, it must
//      also exit 0 with empty stdout and write a valid JSON object whose status is canceled to the
//      results file.
//  42. Required parameter marker: The built-in catalog must contain a required parameter. bild info
//      for its qualified model key must exit 0 and include output.RequiredLabel on that parameter's
//      flag line.
//  43. Flags after the prompt: bild hello world -m zzz-not-a-model, with hello world supplied as
//      one argument, must exit 1 and write exactly the same stderr as placing -m before the prompt.
//      The error must name zzz-not-a-model.
//  44. Trimmed prompt: Given a prompt surrounded by whitespace, bild --json must emit a document
//      whose prompt is the trimmed value.
//  45. CLI empty flag value: Given -m followed by an empty string and prompt p, parseCfg must
//      succeed, invoke its capture action, and return Model explicitly set to the empty string.
//  46. Explicit zero-valued generation flags: Given --strength 0, -N 0, -n=false, and -o with an
//      empty string, parseCfg must succeed and invoke its capture action. The returned parameters
//      must contain numeric zero strength, integer zero num-images, and false include-thoughts;
//      OutPath must be explicitly set to the empty string.
//  47. Commas within input media URLs: parseCfg must preserve the fixture HTTPS URL containing a
//      comma as one input-media source, with or without its first: frame prefix.
//  48. Dash-prefixed flag values: Given -o -m p, parseCfg must succeed and invoke its capture
//      action with OutPath set to -m, Prompt set to p, and Model unset.
//  49. Long-form generation flags: parseCfg must capture the supplied long-form run flags and model
//      parameters with their declared values and types.
//  50. Short-form generation flags: parseCfg must capture the supplied short-form run flags and
//      model parameters with their declared values and types.
//  51. Omitted generation flags: Given only prompt p, parseCfg must succeed and invoke its capture
//      action with Model and OutPath unset, Prompt explicitly set to p, and an empty parameter map.
//  52. Repeated CLI flag precedence: Given repeated -m and --model flags, parseCfg must succeed and
//      set Model to the last supplied value, regardless of which flag form appears last.
//  53. Positional CLI arguments: parseCfg must accept flags on either side of prompt p and retain
//      their values. Two positional arguments must return errs.ErrCLIOneArg without invoking
//      capture. Given --output-path out/ -- -m, it must capture Prompt=-m and OutPath=out/ with
//      Model unset.
//  54. Repeated CLI input media: Given alternating -i and --input-media flags for a.png, b.jpg, and
//      c.webp, parseCfg must succeed, invoke capture, and retain exactly those three sources in
//      that order.
//  55. Comma-separated CLI input media: Given -i a.png,b.jpg, parseCfg must succeed, invoke
//      capture, and return exactly a.png and b.jpg as separate input-media sources in that order.
//  56. CLI syntax forms: parseCfg must accept -model and --model with either separate or assigned
//      values and capture Model=x and Prompt=p. Given -n p, it must capture include-thoughts=true
//      without consuming the prompt. Given -N 0x10 p, it must capture num-images as integer 16.
//      Every case must succeed and invoke capture.
//  57. Persistence selection: After a credential failure, bild must write a valid failed record
//      only when persistence or debug is enabled. Debug must override persist-record=false.
//  58. Silent records: bild must omit the record basename from stdout and its full path from
//      stderr. Enabling persistence alone must preserve the ordinary error text.

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
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"
	"github.com/urfave/cli/v3"
)

// TestCLIListCatalogDocument verifies invariant #1: CLI list catalog document.
//
// What is being tested:
// bild list --json must exit 0, leave stderr empty, and emit exactly the selected catalog data.
// Default and --models output must include the selected models without parameter definitions;
// --providers must omit models. Image and video filters must include only matching models and
// providers that contain them.
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
// What is being tested:
// For the supplied provider term, regular expression, and video-filtered term, bild search --json
// must exit 0 with empty stderr and emit exactly the catalog records corresponding to the nonempty
// model set listed by search -m.
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
// What is being tested:
// For every built-in provider and model, bild info --json must exit 0 with empty stderr and emit
// exactly the selected public catalog records, including model parameters and referenced flag
// definitions. Model documents must omit paramID and config keys. The catalog must exercise a
// required parameter, and the image-filtered provider document must include only image models.
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
// What is being tested:
// bild --help, bild list --help, and bild info --help must exit 0, leave stderr empty, and show -j,
// --json on stdout. The output of bild help help must omit --json.
// Test class: Core: Incidental.
// Pins the short/long flag separator in help.
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
// What is being tested:
// bild info must exit 0 and print nonempty output for the selected provider ID, a uniquely
// resolving bare model ID, its provider-qualified form, and the available bare and
// provider-qualified alias forms.
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
// What is being tested:
// With credentials absent and a duration outside a model's fixed set, bild --json must exit 1,
// leave stderr empty, and emit one failed outcome. The document must identify the provider, model,
// and prompt; preserve the submitted flags; record the snapped duration; include a valid timestamp
// and nonnegative durationMs; and contain exactly one error naming the credential variable. It must
// omit provider request keys that differ from public flag IDs.
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

// TestCLIPrintFilenameFlag verifies invariant #7: CLI print filename flag.
//
// What is being tested:
// Given --print-filename and unknown model no-such-model-pf, bild must exit 1, leave stdout empty,
// and name the failing model in stderr.
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

// TestCLISaveResultsFlag verifies invariant #8: CLI save results flag.
//
// What is being tested:
// Given --save-results and unknown model no-such-model-sr, bild must exit 1, create the results
// file, leave stdout empty, and name the failing model in stderr.
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

// TestHelpListsOutputFlags verifies invariant #9: Help lists output flags.
//
// What is being tested:
// bild --help must succeed without diagnostics and include both --print-filename and --save-results
// in its output.
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

// TestCLISearchMatching verifies invariant #10: CLI search matching.
//
// What is being tested:
// For the selected provider ID, bild search -m must list exactly the models whose slash-delimited
// key tokens start with that ID. Searching the first three characters of each model's last ID token
// must include that model. With a regular expression anchored to the provider prefix, every listed
// key must use that prefix. All searches must exit 0 with empty stderr.
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

// TestCLIHelp verifies invariant #11: CLI help.
//
// What is being tested:
// bild -h, bild --help, and bild -help must each exit 0, leave stderr empty, and print the same
// general help page before the variable tips section.
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

// TestCLIListSelectorFlags verifies invariant #12: CLI list filter flags.
//
// What is being tested:
// bild list must accept -p/--providers, -m/--models, -i/--image, and -v/--video separately. Each
// invocation must exit 0, print a nonempty listing, and leave stderr empty.
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

// TestCLIListModelsFilter verifies invariant #13: CLI list models filter.
//
// What is being tested:
// bild list -m must exit 0 with empty stderr and list exactly the catalog's fully qualified model
// keys, ordered by provider display name with aggregators last, then by model ID.
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

// TestCLIListFilterCombinations verifies invariant #14: CLI list filter combinations.
//
// What is being tested:
// bild list -p -m and bild list -i -v must each exit 0 and print exactly the default nested
// listing. The counts from list -m -i and list -m -v must sum to the catalog's total model count.
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

// TestCLISearchListing verifies invariant #15: CLI search listing.
//
// What is being tested:
// For a provider-ID term, bild search must show the selected provider and model, search -p must
// show the provider without that model ID, and search -m must show the model key without the
// provider display name. With -v, no listed catalog model may be an image. A term matching no model
// must exit 0 and leave both streams empty.
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

// TestCLIInfoStandardProviderPage verifies invariant #16: CLI info standard provider page.
//
// What is being tested:
// For a standard provider, bild info must exit 0 with empty stderr, name the provider on the first
// line, include its ID and credential variable, and list every bare model ID. Neither the full page
// nor its successful -v form may contain a provider-qualified example key.
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

// TestCLIInfoAggregatorSummary verifies invariant #17: CLI info aggregator summary.
//
// What is being tested:
// For an aggregator, bild info must exit 0 with empty stderr and show exactly one valid example
// model key per represented medium, plus each medium's vendor with the most models. The successful
// --image form must restrict example keys to image models. The successful --json form must contain
// one provider and the full number of its models.
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

// TestCLIInfoModelCard verifies invariant #18: CLI info model card.
//
// What is being tested:
// For an image model with multiple aliases, bild info must exit 0 with empty stderr. The card must
// name the provider on its first line, include the fully qualified key, comma-separated aliases,
// and credential variable, place the provider name and ID together on one line, and include a line
// ending with the bare model ID.
//
// Test class: Core: Incidental.
// Pins comma-separated aliases in the model card.
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

// TestGenerateUsesConfiguredDefaultModel verifies invariant #19: Generate uses configured default
// model.
//
// What is being tested:
// With credentials absent and default-model configured, bild must exit 1 and name that model's
// provider credential variable in stderr, omitting the built-in default's variable. An explicit
// --model must instead make stderr name the selected model's credential variable and omit the
// configured default's variable.
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

// TestHelpShowsConfiguredDefaultModel verifies invariant #20: Configured default in help.
//
// What is being tested:
// With default-model configured, bild help must exit 0 with empty stderr and include the configured
// model in the formatted default suffix.
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

// TestCLIOutputPathHelp verifies invariant #21: CLI output path help.
//
// What is being tested:
// bild --help must succeed without diagnostics and include the complete OutputPathFlagHelp text
// when whitespace is normalized.
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

// TestCLIVersion verifies invariant #22: CLI version.
//
// What is being tested:
// bild -v and bild --version must each exit 0, leave stderr empty, and print exactly the version
// line derived from appVersion.
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

// TestCLIReservedExact verifies invariant #23: Reserved command dispatch.
//
// What is being tested:
// bild list must exit 0, print a built-in provider name, and leave stderr empty. bild help must
// exit 0 and match the general help page before tips. bild info without an argument must fail,
// leave stdout empty, and write nonempty stderr.
//
// Test class: Core.
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

// TestCLIHelpPageStructure verifies invariant #24: CLI help page structure.
//
// What is being tested:
// bild --help must exit 0 with empty stderr and show --aspect-ratio, --input-media, --model,
// --output-path, and a built-in provider name. Its command list must be exactly list, info, search,
// help. The page must omit --output-dir, -O, and debug.
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

// TestCLICommandHelpPages verifies invariant #25: CLI command help pages.
//
// What is being tested:
// bild list --help must exit 0 and show its usage form and four listing filters without a built-in
// provider's display name. bild info --help must exit 0 and show its own usage form.
//
// Test class: Core.
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

// TestCLIProvidersListing verifies invariant #26: CLI providers listing.
//
// What is being tested:
// bild list must exit 0 with empty stderr and include every built-in provider name in alphabetical
// display-name order with aggregators last. Its output must contain every declared model ID.
//
// Test class: Core.
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

// TestCLIListProvidersFilter verifies invariant #27: CLI list providers filter.
//
// What is being tested:
// bild list -p must exit 0 with empty stderr, include every provider display name, and omit every
// catalog model ID.
//
// Test class: Core.
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

// TestCLIInfoProviderPage verifies invariant #28: CLI info provider page.
//
// What is being tested:
// For a standard provider, bild info must exit 0 with empty stderr and include its name, ID,
// credential variable, and all model IDs while omitting the selected foreign model ID. For
// bogus-prov, it must exit 1, leave stdout empty, and write the formatted unknown-model message to
// stderr.
//
// Test class: Core.
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

// TestCLIInfoModelPage verifies invariant #29: CLI info model page.
//
// What is being tested:
// For the selected built-in models, bild info must exit 0 with empty stderr and include the
// required public parameter IDs, allowed values, and rule descriptions. The model with a distinct
// provider request key must omit that key. For no-such-model-xyz, info must exit 1, leave stdout
// empty, and write the formatted unknown-model message to stderr.
//
// Test class: Core.
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

// TestRunCLIRealChain verifies invariant #30: Complete CLI construction.
//
// What is being tested:
// Through the production createCommand and runExitCode path, bild -v must exit 0, leave stderr
// empty, and print exactly the version line derived from appVersion.
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

// TestCLISearchCommand verifies invariant #31: CLI search command.
//
// What is being tested:
// bild search must accept the supplied regex, provider/model, and media flag forms with exit 0 and
// empty stderr. Missing or surplus terms must exit 2 with empty stdout and usage on stderr. Search
// help must list its filter, regex, and JSON flags; general help must list search; list help must
// omit --search and --regex.
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

// TestGenerateUsesProviderDefault verifies invariant #32: Provider default model.
//
// What is being tested:
// Without --model or configured default-model, bild must select the eligible provider's default,
// print its model header, and exit 1 when the blocked request fails. If two providers have keys, it
// must select the first in listing order. With no eligible provider, text mode must exit 2 with
// empty stdout, usage and NoDefaultModel on stderr, and an empty output directory. JSON mode must
// exit 2 with exactly one NoDefaultModel error.
//
// Test class: Core: Incidental.
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

// TestHelpShowsProviderDefault verifies invariant #33: Help shows the provider default.
//
// What is being tested:
// With one eligible provider credential, bild help must exit 0 with empty stderr and show that
// provider's qualified default model in the default suffix. With no credentials, help must still
// succeed without diagnostics and omit the default suffix.
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

// TestHelpTips verifies invariant #34: Help tips.
//
// What is being tested:
// bild --help must exit 0 with empty stderr and include a tips section drawing model keys from
// exactly one of OpenAI, Google, and xAI. It must show one valid key per represented medium, use
// each key in a successful bild info example, and include a bild search example naming that
// provider.
//
// Test class: Core: Incidental.
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

// TestCLISearchExclusionTerm verifies invariant #35: Search with an exclusion term.
//
// What is being tested:
// bild search -m openai with -x openrouter must exit 0 with empty stderr and list exactly the
// inclusion matches minus exclusion matches, across all tested flag placements and assignment
// forms. The JSON form must return the same keys in one document. With -r, both mini and router
// must act as regex terms and produce the corresponding difference. Excluding openrouter from a
// regex matching every key must reduce results below the limit and succeed; excluding an unmatched
// term must retain the limit failure with exit 1 and empty stdout. Missing or empty terms, a
// missing exclusion value, and an invalid exclusion regex must exit 2 with empty stdout and usage
// plus the relevant message on stderr. JSON with an empty exclusion must emit one SearchTermEmpty
// error and empty stderr. search -m -x -h must treat -h as the exclusion value and list every key
// successfully. Search help must show an angle-bracketed value placeholder for --exclude.
//
// Test class: Core: Incidental.
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

	if code, _, stderr = captureCLI(t, "search", "-m", "-r", "."); code != 1 || !strings.Contains(stderr, output.SearchTooManyResults) {
		t.Fatalf("💣 [search -m -r .] exited %d with %q; the catalog must exceed the search limit for this check", code, stderr)
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

// TestPromptIgnoredModelNeedsNoPrompt verifies invariant #36: Prompt-ignored model.
//
// What is being tested:
// For a model configured to ignore prompts, bild with no prompt or a blank prompt must exit 1 and
// name its missing credential variable. Its JSON form must record an empty prompt. For the control
// model that consumes a prompt, omitting the prompt must exit 2 and print usage.
//
// Test class: Expanded.
// Test layer: Coverage.
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

// TestCLIReservedQuoted verifies invariant #37: Generation argument parsing.
//
// What is being tested:
// Given each multiword prompt beginning with providers, models, provider, or model as one argument,
// bild --json must exit 1 and preserve the full prompt in stdout. Stderr must omit PARAMETER FLAGS
// usage text.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIReservedQuoted(t *testing.T) {
	clearProviderKeys(t)
	specifier := qualifiedSpecifier(t, builtinDefaultPair(t, shippedCatalog(t)))

	for _, prompt := range []string{
		"providers overview", "models list",
		"provider page info", "model in white halter-top",
	} {
		code, stdout, stderr := captureCLI(t, "--json", "--model", specifier, prompt)
		if code != 1 {
			t.Errorf("✗ %q: exit %d, want 1 (credential stop)", prompt, code)
		}

		if !strings.Contains(stdout, `"prompt": "`+prompt+`"`) {
			t.Errorf("✗ %q: the full quoted prompt is not the prompt: %q", prompt, stdout)
		}

		if strings.Contains(stderr, "PARAMETER FLAGS") {
			t.Errorf("✗ %q: usage text rendered for a quoted prompt", prompt)
		}
	}

	if !t.Failed() {
		t.Log("✓ a quoted multiword prompt led by a reserved word keeps generating")
	}
}

// TestCLIHelpFlagEntryLayout verifies invariant #38: Help layout for flags.
//
// What is being tested:
// bild --help must show the specified help, version, and output-path entries. For every parameter
// flag with an alias, its names must be indented four spaces and its nonempty description must
// follow on a separate line indented eight spaces. The aspect-ratio entry must have a preceding
// blank line, and input-media must omit repeated forms in brackets.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIHelpFlagEntryLayout(t *testing.T) {
	clearProviderKeys(t)

	page := helpText(t)
	for _, wantEntry := range []string{
		"    -h, --help\n        show help\n",
		"    -v, --version\n        print the version\n",
		"    -o, --output-path <path>\n        Path for saving generated media.",
	} {
		if !strings.Contains(page, wantEntry) {
			t.Errorf("✗ the page lacks the entry:\n%s", wantEntry)
		}
	}

	if strings.Contains(page, "[ --input-media") {
		t.Errorf("✗ the repeatable flag still duplicates its forms in brackets")
	}

	paramFlags := params.Flags()

	pageLines := strings.Split(page, "\n")

	for _, paramFlag := range paramFlags {
		longName := string(paramFlag.FlagID)

		// This layout check requires both short and long forms, so skip flags without an
		// alias.
		if len(paramFlag.Aliases) == 0 {
			continue
		}

		at := namesLineIndex(t, pageLines, paramFlag.Aliases[0], longName)
		if at < 0 {
			t.Errorf("✗ %s: no line introduces the flag as its short and long forms", longName)

			continue
		}

		namesLine := pageLines[at]
		if strings.Contains(namesLine, paramFlag.Description) {
			t.Errorf("✗ %s: the names line carries the description: %q", longName, namesLine)
		}

		if at+1 >= len(pageLines) {
			t.Errorf("✗ %s: no line follows the names line", longName)

			continue
		}

		descLine := pageLines[at+1]
		if strings.TrimSpace(descLine) == "" {
			t.Errorf("✗ %s: the line beneath the names carries no description", longName)

			continue
		}

		if leadingSpaces(t, descLine) <= leadingSpaces(t, namesLine) {
			t.Errorf("✗ %s: the description is not indented beneath the names: %q", longName, descLine)
		}

		if leadingSpaces(t, namesLine) != 4 || leadingSpaces(t, descLine) != 8 {
			t.Errorf("✗ %s: indentation is %d/%d, want 4/8", longName, leadingSpaces(t, namesLine), leadingSpaces(t, descLine))
		}
	}

	if !strings.Contains(page, "\n\n    -a, --aspect-ratio") {
		t.Errorf("✗ the flag entries are not separated by a blank line")
	}

	if !t.Failed() {
		t.Log("✓ every flag renders its names at one indent step and its description at two, one blank line apart")
	}
}

// TestCLIRawErrorsHiddenFromHelp verifies invariant #39: Visibility of diagnostic flags.
//
// What is being tested:
// bild --help must exit 0, leave stderr empty, and omit debug from stdout.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIRawErrorsHiddenFromHelp(t *testing.T) {
	code, stdout, stderr := captureCLI(t, "--help")
	if code != 0 {
		t.Errorf("✗ exit %d, want 0", code)
	}

	if strings.Contains(stdout, "debug") {
		t.Errorf("✗ the help names the private debug flag")
	}

	if stderr != "" {
		t.Errorf("✗ stderr = %q, want empty", stderr)
	}

	if !t.Failed() {
		t.Log("✓ the help never names the private debug flag")
	}
}

// TestCLISingleWordPromptConfirmationScope verifies invariant #40: Scope of prompt confirmation.
//
// What is being tested:
// Given nonterminal input with prompt hellp, or terminal input with prompt two words, bild --json
// must exit 1 and retain the complete prompt in its JSON output. Stderr must omit the single-word
// confirmation question despite input containing no.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLISingleWordPromptConfirmationScope(t *testing.T) {
	clearProviderKeys(t)
	specifier := qualifiedSpecifier(t, builtinDefaultPair(t, shippedCatalog(t)))

	for _, testCase := range []struct {
		name   string
		stdin  string
		tty    bool
		prompt string
	}{
		{name: "nonterminal single word", stdin: "no\n", tty: false, prompt: "hellp"},
		{name: "terminal multiword", stdin: "no\n", tty: true, prompt: "two words"},
	} {
		code, stdout, stderr := captureInteractiveCLI(t, testCase.stdin, testCase.tty, "--json", "--model", specifier, testCase.prompt)
		if code != 1 {
			t.Errorf("✗ %s: exit %d, want 1 at the missing-credential boundary", testCase.name, code)
		}

		if strings.Contains(stderr, formBody(t, output.SingleWordConfirmation)) {
			t.Errorf("✗ %s: unexpected confirmation: %q", testCase.name, stderr)
		}

		if !strings.Contains(stdout, `"prompt": "`+testCase.prompt+`"`) {
			t.Errorf("✗ %s: generation did not retain the prompt: %q", testCase.name, stdout)
		}
	}

	if !t.Failed() {
		t.Log("✓ confirmation is limited to single-word prompts read from a terminal")
	}
}

// TestCancelUnderOutputModes verifies invariant #41: Results flags after cancellation.
//
// What is being tested:
// Given a terminal reply of n to prompt hellp, bild --print-filename must exit 0 and leave stdout
// empty. With --json and --save-results, it must also exit 0 with empty stdout and write a valid
// JSON object whose status is canceled to the results file.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCancelUnderOutputModes(t *testing.T) {
	code, stdout, _ := captureInteractiveCLI(t, "n\n", true, "--print-filename", "hellp")
	if code != 0 || stdout != "" {
		t.Errorf("✗ canceled -p run: exit %d stdout %q, want 0 with empty stdout", code, stdout)
	}

	resultsPath := filepath.Join(t.TempDir(), "results.json")

	code, stdout, _ = captureInteractiveCLI(t, "n\n", true, "--json", "--save-results", resultsPath, "hellp")
	if code != 0 || stdout != "" {
		t.Errorf("✗ canceled -s run: exit %d stdout %q, want 0 with empty stdout", code, stdout)
	}

	// #nosec G304 -- resultsPath is built from this test's own temporary directory.
	resultsContent, readErr := os.ReadFile(resultsPath)
	if readErr != nil {
		t.Fatalf("💣 the canceled run's results file was not written: %v", readErr)
	}

	var canceledDocument map[string]any
	if unmarshalErr := json.Unmarshal(resultsContent, &canceledDocument); unmarshalErr != nil {
		t.Fatalf("💣 the results file is not one JSON document: %v", unmarshalErr)
	}

	if canceledDocument["status"] != "canceled" {
		t.Errorf("✗ the results document status = %v, want canceled", canceledDocument["status"])
	}

	if !t.Failed() {
		t.Log("✓ cancellation honors the results flags")
	}
}

// TestCLIInfoRequiredMarker verifies invariant #42: Required parameter marker.
//
// What is being tested:
// The built-in catalog must contain a required parameter. bild info for its qualified model key
// must exit 0 and include output.RequiredLabel on that parameter's flag line.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIInfoRequiredMarker(t *testing.T) {
	clearProviderKeys(t)

	var (
		markedPair catalog.ProvModelPair
		markedFlag params.FlagType
	)

	for _, pair := range shippedCatalog(t).ModelDirectory() {
		for _, paramCfg := range pair.Model.Params {
			if paramCfg.Required {
				markedPair, markedFlag = pair, paramCfg.FlagID
			}
		}
	}

	if markedFlag == "" {
		t.Fatalf("💣 no shipped config marks a parameter required")
	}

	key := markedPair.Provider.ID + "/" + markedPair.Model.ID

	code, stdout, _ := captureCLI(t, "info", key)
	if code != 0 {
		t.Fatalf("💣 info %s: exit %d", key, code)
	}

	for cardLine := range strings.SplitSeq(stdout, "\n") {
		if strings.Contains(cardLine, "--"+string(markedFlag)) {
			if strings.Contains(cardLine, " "+output.RequiredLabel) {
				if !t.Failed() {
					t.Log("✓ the required parameter opens its entry with the required classification")
				}

				return
			}

			t.Fatalf("💣 the --%s entry on %s does not open with the required classification: %q", markedFlag, key, cardLine)
		}
	}

	t.Errorf("✗ the card for %s has no --%s entry", key, markedFlag)
}

// TestFlagsAfterPrompt verifies invariant #43: Flags after the prompt.
//
// What is being tested:
// bild hello world -m zzz-not-a-model, with hello world supplied as one argument, must exit 1 and
// write exactly the same stderr as placing -m before the prompt. The error must name
// zzz-not-a-model.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFlagsAfterPrompt(t *testing.T) {
	clearProviderKeys(t)

	codeAfter, _, stderrAfter := captureCLI(t, "hello world", "-m", "zzz-not-a-model")
	codeBefore, _, stderrBefore := captureCLI(t, "-m", "zzz-not-a-model", "hello world")

	if codeAfter != codeBefore || codeAfter != 1 {
		t.Errorf("✗ exit codes after/before = %d/%d, want 1 for both", codeAfter, codeBefore)
	}

	if stderrAfter != stderrBefore {
		t.Errorf("✗ stderr with the flag after the prompt:\n%s\nwant the same as with it before:\n%s", stderrAfter, stderrBefore)
	}

	if !strings.Contains(stderrAfter, "zzz-not-a-model") {
		t.Errorf("✗ the failure does not name the model given after the prompt:\n%s", stderrAfter)
	}

	if !t.Failed() {
		t.Log("✓ a flag after the prompt governs the run like one before it")
	}
}

// TestPromptTrimmed verifies invariant #44: Trimmed prompt.
//
// What is being tested:
// Given --json, an unknown model, and prompt '  a cat  ', bild must emit JSON that decodes with
// prompt equal to a cat.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPromptTrimmed(t *testing.T) {
	clearProviderKeys(t)

	_, stdout, _ := captureCLI(t, "--json", "-m", "zzz-not-a-model", "  a cat  ")

	var document struct {
		Prompt string `json:"prompt"`
	}

	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("💣 the JSON document does not decode: %v\n%s", err, stdout)
	}

	if document.Prompt != "a cat" {
		t.Errorf("✗ recorded prompt = %q, want %q", document.Prompt, "a cat")
	}

	if !t.Failed() {
		t.Log("✓ the recorded prompt is trimmed")
	}
}

// TestCLIEmptyFlagValue verifies invariant #45: CLI empty flag value.
//
// What is being tested:
// Given -m followed by an empty string and prompt p, parseCfg must succeed, invoke its capture
// action, and return Model explicitly set to the empty string.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIEmptyFlagValue(t *testing.T) {
	cfg, called, err := parseCfg(t, "-m", "", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantSet(t, "Model", cfg.genInputs.Model, "")

	if !t.Failed() {
		t.Log("✓ an empty flag value arrives set to the empty string")
	}
}

// TestCLIConfigZeroSet verifies invariant #46: Explicit zero-valued generation flags.
//
// What is being tested:
// Given --strength 0, -N 0, -n=false, and -o with an empty string, parseCfg must succeed and invoke
// its capture action. The returned parameters must contain numeric zero strength, integer zero
// num-images, and false include-thoughts; OutPath must be explicitly set to the empty string.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIConfigZeroSet(t *testing.T) {
	cfg, called, err := parseCfg(t, "--strength", "0", "-N", "0", "-n=false", "-o", "", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantParam(t, cfg.params, "strength", 0.0)
	wantParam(t, cfg.params, params.FlagTypeImageN, 0)
	wantParam(t, cfg.params, params.FlagTypeThoughts, false)
	wantSet(t, "OutPath", cfg.genInputs.OutPath, "")

	if !t.Failed() {
		t.Log("✓ explicit zero values arrive set, not unset")
	}
}

// TestCLIInputMediaURLComma verifies invariant #47: Commas within input media URLs.
//
// What is being tested:
// Given the fixture HTTPS URL containing a comma, with or without the first: prefix, parseCfg must
// succeed and invoke its capture action. The input-media slice must contain exactly the original
// source string.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIInputMediaURLComma(t *testing.T) {
	cloudinaryURL := "https://res.cloudinary.com/zenbusiness/q_auto,w_1050/v1670445040/logaster/logaster-2013-06-jpg.avif"

	for _, source := range []string{cloudinaryURL, "first:" + cloudinaryURL} {
		cfg, called, err := parseCfg(t, "-i", source, "p")
		if err != nil || !called {
			t.Fatalf("💣 run failed for %q: called=%v err=%v", source, called, err)
		}

		if sources := inputMediaSources(t, cfg.params); !slices.Equal(sources, []string{source}) {
			t.Errorf("✗ input-media sources = %v, want [%s]", sources, source)
		}
	}

	if !t.Failed() {
		t.Log("✓ a comma within an HTTP(S) input source remains part of the URL, with or without a frame prefix")
	}
}

// TestCLIDashValue verifies invariant #48: Dash-prefixed flag values.
//
// What is being tested:
// Given -o -m p, parseCfg must succeed and invoke its capture action with OutPath set to -m, Prompt
// set to p, and Model unset.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIDashValue(t *testing.T) {
	cfg, called, err := parseCfg(t, "-o", "-m", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantSet(t, "OutPath", cfg.genInputs.OutPath, "-m")
	wantSet(t, "Prompt", cfg.genInputs.Prompt, "p")
	wantUnset(t, "Model", cfg.genInputs.Model)

	if !t.Failed() {
		t.Log("✓ a flag consumes a dash-leading value, as the stdlib parser did")
	}
}

// TestCLIConfigLong verifies invariant #49: Long-form generation flags.
//
// What is being tested:
// Given the supplied long flags, parseCfg must succeed and invoke its capture action. Model,
// OutPath, and Prompt must be set to x, P1, and p. The parameters must retain aspect 16:9,
// resolution 1k, size 1280x720, quality high, duration 8, thinking-level low, thoughts true,
// num-images 3, output-format png, and the single input a.png.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIConfigLong(t *testing.T) {
	cfg, called, err := parseCfg(t,
		"--model", "x", "--aspect-ratio", "16:9", "--resolution", "1k",
		"--size", "1280x720", "--quality", "high",
		"--duration", "8", "--thinking-level", "low", "--include-thoughts",
		"--output-path", "P1", "--num-images", "3", "--output-format", "png",
		"--input-media", "a.png", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	checkAllGenerationFlagValues(t, cfg)

	if !t.Failed() {
		t.Log("✓ all long-form flags arrive in the parsed carriers with their values")
	}
}

// TestCLIConfigShort verifies invariant #50: Short-form generation flags.
//
// What is being tested:
// Given the supplied short flags, parseCfg must succeed and invoke its capture action. Model,
// OutPath, and Prompt must be set to x, P1, and p. The parameters must retain aspect 16:9,
// resolution 1k, size 1280x720, quality high, duration 8, thinking-level low, thoughts true,
// num-images 3, output-format png, and the single input a.png.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIConfigShort(t *testing.T) {
	cfg, called, err := parseCfg(t,
		"-m", "x", "-a", "16:9", "-r", "1k", "-s", "1280x720", "-q", "high",
		"-d", "8", "-l", "low", "-n", "-o", "P1", "-N", "3",
		"-f", "png", "-i", "a.png", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	checkAllGenerationFlagValues(t, cfg)

	if !t.Failed() {
		t.Log("✓ all short-form flags arrive in the parsed carriers")
	}
}

// TestCLIConfigUnset verifies invariant #51: Omitted generation flags.
//
// What is being tested:
// Given only prompt p, parseCfg must succeed and invoke its capture action with Model and OutPath
// unset, Prompt explicitly set to p, and an empty parameter map.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIConfigUnset(t *testing.T) {
	cfg, called, err := parseCfg(t, "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantUnset(t, "Model", cfg.genInputs.Model)
	wantUnset(t, "OutPath", cfg.genInputs.OutPath)
	wantSet(t, "Prompt", cfg.genInputs.Prompt, "p")

	if len(cfg.params) != 0 {
		t.Errorf("✗ the parameter map carries %d entries with only a prompt supplied: %v", len(cfg.params), cfg.params)
	}

	if !t.Failed() {
		t.Log("✓ omitted flags stay unset; the prompt is always set")
	}
}

// TestCLIConfigLastWins verifies invariant #52: Repeated CLI flag precedence.
//
// What is being tested:
// Given repeated -m and --model flags, parseCfg must succeed and set Model to the last supplied
// value, regardless of which flag form appears last.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIConfigLastWins(t *testing.T) {
	cfg, _, err := parseCfg(t, "-m", "a", "--model", "b", "p")
	if err != nil {
		t.Fatalf("💣 run failed: %v", err)
	}

	wantSet(t, "Model", cfg.genInputs.Model, "b")

	cfg, _, err = parseCfg(t, "--model", "b", "-m", "a", "p")
	if err != nil {
		t.Fatalf("💣 run failed: %v", err)
	}

	wantSet(t, "Model", cfg.genInputs.Model, "a")

	if !t.Failed() {
		t.Log("✓ a flag repeated across long and short forms is last-wins")
	}
}

// TestCLIPositionals verifies invariant #53: Positional CLI arguments.
//
// What is being tested:
// parseCfg must accept flags on either side of prompt p and retain their values. Two positional
// arguments must return errs.ErrCLIOneArg without invoking capture. Given --output-path out/ -- -m,
// it must capture Prompt=-m and OutPath=out/ with Model unset.
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIPositionals(t *testing.T) {
	cfg, called, err := parseCfg(t, "p", "-m", "x")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantSet(t, "Prompt", cfg.genInputs.Prompt, "p")
	wantSet(t, "Model", cfg.genInputs.Model, "x")

	_, called, err = parseCfg(t, "p", "out")
	if called || !errors.Is(err, errs.ErrCLIOneArg) {
		t.Errorf("✗ two positional arguments: called=%v err=%v, want the argument-count error and no run", called, err)
	}

	cfg, called, err = parseCfg(t, "--output-path", "out/", "--", "-m")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantSet(t, "Prompt", cfg.genInputs.Prompt, "-m")
	wantSet(t, "OutPath", cfg.genInputs.OutPath, "out/")
	wantUnset(t, "Model", cfg.genInputs.Model)

	cfg, called, err = parseCfg(t, "-a", "1:1", "p", "-r", "1k")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantParam(t, cfg.params, params.FlagTypeAspect, "1:1")
	wantParam(t, cfg.params, params.FlagTypeResolution, "1k")

	if !t.Failed() {
		t.Log("✓ flags parse on either side of the prompt, a second positional is rejected, and -- is honored")
	}
}

// TestCLIInputMediaRepeats verifies invariant #54: Repeated CLI input media.
//
// What is being tested:
// Given alternating -i and --input-media flags for a.png, b.jpg, and c.webp, parseCfg must succeed,
// invoke capture, and retain exactly those three sources in that order.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIInputMediaRepeats(t *testing.T) {
	cfg, called, err := parseCfg(t, "-i", "a.png", "--input-media", "b.jpg", "-i", "c.webp", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	want := []string{"a.png", "b.jpg", "c.webp"}

	paths := inputMediaSources(t, cfg.params)
	if len(paths) != len(want) {
		t.Errorf("✗ input-media paths = %v, want %v", paths, want)
	} else {
		for i := range want {
			if paths[i] != want[i] {
				t.Errorf("✗ input-media paths[%d] = %q, want %q", i, paths[i], want[i])
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ repeated -i/--input-media preserves source order")
	}
}

// TestCLIInputMediaCommaList verifies invariant #55: Comma-separated CLI input media.
//
// What is being tested:
// Given -i a.png,b.jpg, parseCfg must succeed, invoke capture, and return exactly a.png and b.jpg
// as separate input-media sources in that order.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIInputMediaCommaList(t *testing.T) {
	cfg, called, err := parseCfg(t, "-i", "a.png,b.jpg", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	want := []string{"a.png", "b.jpg"}
	if sources := inputMediaSources(t, cfg.params); !slices.Equal(sources, want) {
		t.Errorf("✗ input-media sources = %v, want %v", sources, want)
	}

	if !t.Failed() {
		t.Log("✓ one -i/--input-media accepts a comma-separated source list")
	}
}

// TestCLISyntaxForms verifies invariant #56: CLI syntax forms.
//
// What is being tested:
// parseCfg must accept -model and --model with either separate or assigned values and capture
// Model=x and Prompt=p. Given -n p, it must capture include-thoughts=true without consuming the
// prompt. Given -N 0x10 p, it must capture num-images as integer 16. Every case must succeed and
// invoke capture.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLISyntaxForms(t *testing.T) {
	for _, args := range [][]string{
		{"-model=x", "p"}, {"--model=x", "p"}, {"-model", "x", "p"}, {"--model", "x", "p"},
	} {
		cfg, called, err := parseCfg(t, args...)
		if err != nil || !called {
			t.Fatalf("💣 run failed for %v: called=%v err=%v", args, called, err)
		}

		wantSet(t, "Model", cfg.genInputs.Model, "x")
		wantSet(t, "Prompt", cfg.genInputs.Prompt, "p")
	}

	cfg, called, err := parseCfg(t, "-n", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantParam(t, cfg.params, params.FlagTypeThoughts, true)
	wantSet(t, "Prompt", cfg.genInputs.Prompt, "p")

	cfg, called, err = parseCfg(t, "-N", "0x10", "p")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantParam(t, cfg.params, params.FlagTypeImageN, 16)

	if !t.Failed() {
		t.Log("✓ dash variants, =-forms, bool no-value, and base-0 integers all parse per the contract")
	}
}

// TestPersistenceSelection verifies invariant #57: Persistence selection and invariant #58: Silent
// records.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// When the selected Veo model lacks credentials, bild must exit 1. With --persist-record or
// --debug, it must create a valid failed record containing a request and an empty
// provider-responses array; --debug must override --persist-record=false. Without either enabled
// option, the record must remain absent. Stdout must omit the record basename, stderr must omit its
// full path, and persistence alone must leave the ordinary error text unchanged.
func TestPersistenceSelection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GOOGLE_API_KEY", "")

	for _, testCase := range []struct {
		name    string
		flags   []string
		persist bool
	}{
		{name: "absent"},
		{name: "false", flags: []string{"--persist-record=false"}},
		{name: "explicit", flags: []string{"--persist-record"}, persist: true},
		{name: "debug", flags: []string{"--debug"}, persist: true},
		{name: "debug_false", flags: []string{"--debug", "--persist-record=false"}, persist: true},
		{name: "filename", flags: []string{"--persist-record", "--print-filename"}, persist: true},
		{name: "json", flags: []string{"--persist-record", "--json"}, persist: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			arguments := append([]string{"--model", "google/veo-3.1-generate-preview", "--output-path", filepath.Join(directory, "boat.mp4")}, testCase.flags...)
			arguments = append(arguments, "a paper boat")

			code, stdout, stderr := captureCLI(t, arguments...)
			if code != 1 {
				t.Errorf("✗ credential failure exit %d", code)
			}

			recordPath := filepath.Join(directory, "boat.bild.json")

			body, err := os.ReadFile(recordPath) //nolint:gosec // Read this test's generated record in its temporary directory.
			if testCase.persist {
				var document map[string]json.RawMessage
				if err != nil || json.Unmarshal(body, &document) != nil {
					t.Errorf("✗ missing valid record: %v", err)
				}

				if string(document["provider-responses"]) != "[]" || len(document["request"]) == 0 || string(document["status"]) != `"failed"` {
					t.Errorf("✗ incomplete failure record: %s", body)
				}
			} else if !os.IsNotExist(err) {
				t.Errorf("✗ unexpected record: %v", err)
			}

			if strings.Contains(stdout, filepath.Base(recordPath)) || strings.Contains(stderr, recordPath) {
				t.Errorf("✗ record reported: %q / %q", stdout, stderr)
			}

			if testCase.name == "explicit" {
				plainDir := t.TempDir()

				_, _, plainError := captureCLI(t, "--model", "google/veo-3.1-generate-preview", "--output-path", filepath.Join(plainDir, "boat.mp4"), "a paper boat")
				if stderr != plainError {
					t.Errorf("✗ persistence enabled debug output: %q versus %q", stderr, plainError)
				}
			}

			if !t.Failed() {
				t.Log("✓ persistence selection and silent failure record")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ persistence is opt-in and implied by debug")
	}
}

// Configuration keys read by these tests.
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

// helpText captures the general help page and fails if the command returns an error or diagnostic.
//
// Test class: Core: Helper.
func helpText(t *testing.T) string {
	t.Helper()

	code, stdout, stderr := captureCLI(t, "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("💣 the --help reference capture failed: exit %d, stderr %q", code, stderr)
	}

	return stdout
}

// versionText returns the expected version report for the current build.
//
// Test class: Core: Helper.
func versionText(t *testing.T) string {
	t.Helper()

	return "bild version " + appVersion() + "\n"
}

// captureCLI runs the production command and returns its exit status, stdout, and stderr.
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

// captureInteractiveCLI runs the production command with supplied input and captures both output
// streams.
func captureInteractiveCLI(t *testing.T, stdin string, tty bool, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	useStdin(t, stdin, tty)

	originalStdout := os.Stdout

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 stdout pipe: %v", err)
	}

	os.Stdout = writer
	captured := make(chan string, 1)

	go func() {
		defer reader.Close()

		content, _ := io.ReadAll(reader)
		captured <- string(content)
	}()

	stderr = captureDiagnosticOutput(t, tty, func() {
		app := &bildApp{defaultOutDir: t.TempDir()}
		code = runExitCode(createCommand(app).Run(context.Background(), append([]string{"bild"}, args...)))
	})
	os.Stdout = originalStdout
	_ = writer.Close()

	return code, <-captured, stderr
}

// captureDiagnosticOutput captures stderr from a pipe or pseudo-terminal and removes color escapes.
func captureDiagnosticOutput(test testing.TB, interactive bool, execute func()) string {
	test.Helper()

	var reader, writer *os.File
	if interactive {
		reader, writer = openPTY(test)
	} else {
		var err error

		reader, writer, err = os.Pipe()
		if err != nil {
			test.Fatalf("💣 diagnostic pipe: %v", err)
		}
	}

	originalStderr := os.Stderr
	os.Stderr = writer
	captured := make(chan string, 1)

	go func() {
		defer reader.Close()
		// A closed terminal slave may end the master read with EIO; its bytes are still the
		// complete diagnostic capture.
		content, _ := io.ReadAll(reader)
		captured <- string(content)
	}()

	execute()

	os.Stderr = originalStderr
	_ = writer.Close()

	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(<-captured, "")
}

// decodeJSONObject decodes exactly one JSON document and fails when stdout contains a second value
// or non-JSON text.
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

// setUserConfig points HOME at a scratch home directory holding a .bildomat/config.yml with the
// given content, and returns the file's path.
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

// firstStandardProvider returns the first registered provider that is not an aggregator: the
// provider whose details page is the standard roster page.
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

// promptIgnoredPair returns the first shipped model, in catalog order, that is configured as
// ignoring the prompt, with its provider's identity.
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

// builtinDefaultPair returns the declared default model of the first provider in listing order that
// declares one, as a provider-model pair.
//
// Test class: Core: Helper.
func builtinDefaultPair(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	first, _ := firstTwoProvidersWithDefaults(t, loadedCatalog)

	return catalog.ProvModelPair{Provider: first.Identity(), Model: defaultModelOf(t, first)}
}

// firstTwoProvidersWithDefaults returns the first two providers, in listing order, that declare a
// default model.
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

// defaultModelOf returns the model identified by the provider's DefaultModel field.
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

// blockNetwork requests an unreachable loopback proxy for provider calls. The standard transport
// caches these settings, so the test process must start with them too.
//
// Test class: Core: Helper.
func blockNetwork(t *testing.T) {
	t.Helper()
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
}

// bareIDResolvesAlone reports whether the pair's bare model ID resolves to that pair and no other.
//
// Test class: Core: Helper.
func bareIDResolvesAlone(t *testing.T, loadedCatalog *catalog.Catalog, pair catalog.ProvModelPair) bool {
	t.Helper()

	pairs, err := loadedCatalog.ResolveModelInput(pair.Model.ID)

	return err == nil && len(pairs) == 1 && pairs[0].Provider.ID == pair.Provider.ID
}

// qualifiedSpecifier returns the pair's fully qualified model specifier: the provider ID and the
// model ID joined by the key separator.
//
// Test class: Core: Helper.
func qualifiedSpecifier(t *testing.T, pair catalog.ProvModelPair) string {
	t.Helper()

	return pair.Provider.ID + catalog.KeySeparator + pair.Model.ID
}

// fixedDurationSet returns the allowed values of the model's duration parameter, or nil when the
// model declares no duration parameter with a fixed set.
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

// fixedDurationVideoModel returns the first video model whose duration parameter declares a fixed
// set of allowed values.
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

// durationOutsideFixedSet returns one second above the largest allowed duration and the allowed
// maximum.
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

// modelOutsideKeyVar returns the first model whose provider reads its API key from an environment
// variable other than the given one.
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

// fixedSetModelResolvingAlone returns the first model that resolves alone by its bare ID and
// declares a parameter with a fixed set of allowed values.
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

// fixedDurationModelResolvingAlone returns the first model that resolves alone by its bare ID and
// whose duration parameter declares a fixed set.
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

// ruleDescribedModelResolvingAlone returns the first model that resolves alone by its bare ID and
// declares a parameter with a rule description.
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

// modelWithHiddenRequestKey returns the first model with fixed allowed values and a provider
// request key absent from the provider's public text and the catalog's flag descriptions.
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

// fixedSetParams returns the model's parameters that declare a fixed set of allowed values, or nil
// when none does.
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

// ruleDescriptions returns the nonempty RuleDescription values from the model's parameters, or nil.
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

// declaredText joins the provider identity and each model's identity, aliases, description,
// parameter descriptions, and allowed values for checks against public output.
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

// flagRecordText returns every user-facing text of the catalog's flag records: names, aliases,
// descriptions, comments, hints, and examples.
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

// foreignModelID returns an identifier from another provider that does not appear in the selected
// provider's declared text.
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

// requestKeyOutsideDeclaredText returns the first model request key absent from the provider's
// public text and the catalog's flag descriptions. It returns an empty string if none qualifies.
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

// requestKeysOutsideFlagIDs returns nonempty model request keys that differ from every catalog flag
// ID.
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

// namesLineIndex finds the help page line that introduces a flag by its short and long forms,
// returning -1 when no line does.
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

// modelListings splits a listing's stdout into its non-empty lines.
//
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

// useStdin replaces the process's standard input for the test with the given text: read from a
// raw-mode pseudo-terminal when tty is set, so the prompts ask and the bytes arrive unchanged, and
// from a pipe otherwise. The input ends after the text either way.
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

		// The write runs beside the test so text beyond the pipe's capacity cannot block
		// the setup; closing the read end first unblocks a writer nothing has read, and the
		// wait leaves no writer behind.
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

	// The terminal is in raw mode, so the bytes arrive unchanged. Closing the master ends the
	// input with EOF, but it also discards whatever the program has not read yet, so the writer
	// closes it only once the input is drained, or once the test ends. The writer alone touches
	// the master; the test closes the slave after the writer is done. A write the program never
	// reads blocks once the terminal's input queue is full, so the test ends it with a write
	// deadline before waiting for the writer.
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

// catalogModelKeys derives fully qualified model keys from the built-in configs, ordered by
// provider name with aggregators last, then by model identifier.
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

// helpCommandNames extracts names from the help page's COMMANDS section in page order, removing
// each trailing alias comma.
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

// modelsOfMedia returns only models that produce the specified medium.
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

// pageKeysOf extracts distinct, fully qualified model keys for the specified provider from a page,
// preserving their order of appearance.
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

// beforeTips returns the general help text preceding the tips section and fails if that section is
// absent.
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

// checkProviderKeys requires exactly one valid example model key per represented medium.
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

// topVendors takes models and returns, for each medium with a model, the vendor declaring the most
// of that medium's models, ties broken by name; a model's vendor is the first slash-delimited token
// of its bare ID.
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

// withoutEmptyValues returns the value with every empty member removed at any depth: an empty
// string, false, null, an empty array, and an empty object. A number is never empty.
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

// reducedMember returns a member with its empty values removed and whether the member itself is
// empty.
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

// keysOfMedia returns the fully qualified keys of every built-in model whose medium is among the
// selected media; with neither selected, every model's key.
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

// reducedModel returns a config model as the document carries it: without its params on a listing,
// with its params minus their provider request keys on an info page, and without empty values.
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

// reducedProvider returns a config provider as the document carries it: its identity, the models
// whose keys are kept where the document lists models, and neither its request settings nor any
// empty value. It returns nil when the provider keeps no model.
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

// referencedFlags returns one copy of each flag record referenced by the providers' parameters,
// preserving record order and omitting empty values.
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

// expectedListing builds the expected document from built-in provider configs and selected model
// keys. It includes providers with matching models, optionally includes those models, and omits
// parameters.
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

// expectedInfo builds the expected info document for one provider and its selected models,
// including their parameters and referenced flag records.
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

// wantInfoDocument combines the reduced providers with their referenced flag records. It omits the
// flags field when no parameter references a flag record.
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

// checkDocument runs the command and compares its one JSON document with the expected document as
// data.
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

// checkNoInternalKeys fails when any object in the document carries a provider request key or a
// provider's request settings.
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

// formVerbPattern matches one formatting verb of a catalog form, indexed or not.
func formVerbPattern(t testing.TB) *regexp.Regexp {
	t.Helper()

	return regexp.MustCompile(`%\[?[0-9]*\]?[-+# 0-9.]*[a-zA-Z]`)
}

// formBody returns the longest literal text between formatting placeholders in a catalog string.
func formBody(t testing.TB, form string) string {
	t.Helper()

	longest := ""

	for _, segment := range formVerbPattern(t).Split(form, -1) {
		if len(segment) > len(longest) {
			longest = segment
		}
	}

	return strings.TrimSpace(longest)
}

// parsedRun holds captured generation options and parameter values.
//   - genInputs: parsed options and the prompt.
//   - params: explicitly supplied model parameters.
type parsedRun struct {
	genInputs RunFlags
	params    params.FlagInputs
}

// testCommand captures parsed inputs using the production flag definitions and input readers.
func testCommand(t *testing.T, capture func(RunFlags, params.FlagInputs) error) *cli.Command {
	t.Helper()
	app := realApp(t)

	paramFlags := params.Flags()

	app.catalog = shippedCatalog(t)

	return &cli.Command{
		Name:                      "bild",
		DisableSliceFlagSeparator: true,
		MutuallyExclusiveFlags:    createFlags(app.catalog.Flags),
		OnUsageError: func(_ context.Context, _ *cli.Command, err error, _ bool) error {
			return fmt.Errorf("%w: %w", errs.ErrCLIFlagParse, err)
		},
		Action: func(_ context.Context, c *cli.Command) error {
			if c.Args().Len() > 1 {
				return argCountError(c.Name, 1, c.Args().Len())
			}

			genInputs := createRunInputs(c)
			if genInputs.Prompt.ValOr("") == "" {
				return errs.ErrCLIPromptMissing
			}

			return capture(genInputs, createParamInputs(c, paramFlags))
		},
	}
}

// parseCfg returns captured generation inputs, whether capture ran, and the parsing error.
func parseCfg(t *testing.T, args ...string) (cfg parsedRun, called bool, err error) {
	t.Helper()
	cmd := testCommand(t, func(genInputs RunFlags, parameterValues params.FlagInputs) error {
		cfg = parsedRun{genInputs: genInputs, params: parameterValues}
		called = true

		return nil
	})
	cmd.Writer = io.Discard
	cmd.ErrWriter = io.Discard
	err = cmd.Run(context.Background(), append([]string{"bild"}, args...))

	return cfg, called, err
}

// wantSet asserts the Nullable was explicitly set to want.
func wantSet[T comparable](t *testing.T, name string, n params.Nullable[T], want T) {
	t.Helper()

	val, ok := n.ValIf()
	if !ok {
		t.Errorf("✗ %s: unset, want set to %v", name, want)

		return
	}

	if val != want {
		t.Errorf("✗ %s = %v, want %v", name, val, want)
	}
}

// wantUnset asserts the Nullable was left unset.
func wantUnset[T comparable](t *testing.T, name string, n params.Nullable[T]) {
	t.Helper()

	if val, ok := n.ValIf(); ok {
		t.Errorf("✗ %s set to %v, want unset", name, val)
	}
}

// wantParam asserts the parameter arrived in the map with the given typed value.
func wantParam(t *testing.T, parameterValues params.FlagInputs, param params.FlagType, want any) {
	t.Helper()

	val, ok := parameterValues[param]
	if !ok {
		t.Errorf("✗ %s: absent, want %v", param, want)

		return
	}

	if val != want {
		t.Errorf("✗ %s = %v (%T), want %v (%T)", param, val, val, want, want)
	}
}

// inputMediaSources reads the repeatable input-media value from the map.
func inputMediaSources(test testing.TB, parameterValues params.FlagInputs) []string {
	test.Helper()

	paths, _ := parameterValues[params.FlagTypeInputMedia].([]string)

	return paths
}

// checkAllGenerationFlagValues checks the common parse outcome for the long-form and short-form
// generation flag tests.
func checkAllGenerationFlagValues(t *testing.T, cfg parsedRun) {
	t.Helper()

	wantSet(t, "Model", cfg.genInputs.Model, "x")
	wantParam(t, cfg.params, params.FlagTypeAspect, "16:9")
	wantParam(t, cfg.params, params.FlagTypeResolution, "1k")
	wantParam(t, cfg.params, params.FlagTypeSize, "1280x720")
	wantParam(t, cfg.params, params.FlagTypeQuality, "high")
	wantParam(t, cfg.params, params.FlagTypeDuration, 8)
	wantParam(t, cfg.params, params.FlagTypeThinkingLevel, "low")
	wantParam(t, cfg.params, params.FlagTypeThoughts, true)
	wantSet(t, "OutPath", cfg.genInputs.OutPath, "P1")
	wantParam(t, cfg.params, params.FlagTypeImageN, 3)
	wantParam(t, cfg.params, params.FlagTypeOutputFormat, "png")
	wantSet(t, "Prompt", cfg.genInputs.Prompt, "p")

	if paths := inputMediaSources(t, cfg.params); len(paths) != 1 || paths[0] != "a.png" {
		t.Errorf("✗ input-media paths = %v, want [a.png]", paths)
	}
}
