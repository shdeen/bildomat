package catalog

// Invariants tested:
//  1. Catalog encoding: Given a Catalog with one provider and no flags, JSON marshaling must emit
//     exactly the provider identity under providers.
//  2. Resolve pair miss continues: Given prov-a/nested-id with no matching qualified model,
//     ResolveModelInput must resolve the identical bare model ID under prov-c.
//  3. Resolve qualified alias: Given prov-a/al-a, ResolveModelInput must return exactly the model-a
//     pair under prov-a, whose model declares that alias.
//  4. Provider listing order: Given the mixed-case display names and an aggregator that sorts first
//     alphabetically, sorting with CompareProviders must order the ordinary providers alpha, beta,
//     zeta and place the aggregator last.
//  5. Explicit provider qualification: Given a qualified model and an identical slash-containing
//     bare ID, ResolveModelInput must choose the qualified provider.
//  6. Model configuration isolation: Given each fixture model as its provider's only model,
//     LoadCatalog and ConfigError must report no error.

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// TestCatalogEncoding verifies invariant #1: Catalog encoding.
//
// What is being tested:
// Given a Catalog with one provider and no flags, JSON marshaling must emit exactly the provider
// identity under providers. With one flag, it must include exactly that flag's nonempty ID, type,
// name, and description fields in its record.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCatalogEncoding(t *testing.T) {
	reduced := encodedKeys(t, Catalog{Providers: []Provider{{ID: "p", DisplayName: "P", APIKeyEnvVar: "P_KEY"}}})
	if !reflect.DeepEqual(reduced, map[string]any{"providers": []any{map[string]any{"id": "p", "displayName": "P", "apiKeyEnvVar": "P_KEY"}}}) {
		t.Errorf("✗ a catalog without flags encodes %v", reduced)
	}

	withFlags := encodedKeys(t, Catalog{Providers: []Provider{}, Flags: []params.Flag{{FlagID: params.FlagTypeSize, DataType: params.DataString, FlagName: "Size", Description: "d"}}})
	flags, _ := withFlags["flags"].([]any)

	if len(flags) != 1 {
		t.Fatalf("✗ a catalog with one flag encodes %v", withFlags)
	}

	record, _ := flags[0].(map[string]any)
	if !reflect.DeepEqual(record, map[string]any{"flagID": string(params.FlagTypeSize), "dataType": string(params.DataString), "flagName": "Size", "description": "d"}) {
		t.Errorf("✗ a flag record encodes empty values: %v", record)
	}

	if !t.Failed() {
		t.Log("✓ a catalog encodes its providers and flags and nothing else")
	}
}

// TestResolvePairMissFallsThrough verifies invariant #2: Resolve pair miss continues.
//
// What is being tested:
// Given prov-a/nested-id with no matching qualified model, ResolveModelInput must resolve the
// identical bare model ID under prov-c. Given prov-a/undeclared-id with no match anywhere, it must
// return ErrModelResolveUnknown and name the full input.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolvePairMissFallsThrough(t *testing.T) {
	catalog := loadFixtureCatalog(t,
		Source{ProviderID: "prov-a", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-a", "model-a", ""))},
		Source{ProviderID: "prov-c", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-c", "prov-a/nested-id", ""))},
	)

	pairs, err := catalog.ResolveModelInput("prov-a/nested-id")
	if err != nil || len(pairs) != 1 {
		t.Errorf("✗ a pair miss over a slash-carrying catalog id = (%+v, %v), want resolution through the bare-id search", pairs, err)
	} else if pairs[0].Provider.ID != "prov-c" || pairs[0].Model.ID != "prov-a/nested-id" {
		t.Errorf("✗ resolved %s/%s, want prov-c's prov-a/nested-id", pairs[0].Provider.ID, pairs[0].Model.ID)
	}

	_, err = catalog.ResolveModelInput("prov-a/undeclared-id")
	if err == nil || !errors.Is(err, errs.ErrModelResolveUnknown) {
		t.Errorf("✗ a pair miss matching nothing = %v, want ErrModelResolveUnknown", err)
	}

	if err != nil && !strings.Contains(err.Error(), "prov-a/undeclared-id") {
		t.Errorf("✗ the unknown-specifier failure does not name the full specifier: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a provider-qualified pair miss falls through the bare-id search; an unmatched one fails as unknown")
	}
}

// TestResolveQualifiedAlias verifies invariant #3: Resolve qualified alias.
//
// What is being tested:
// Given prov-a/al-a, ResolveModelInput must return exactly the model-a pair under prov-a, whose
// model declares that alias.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolveQualifiedAlias(t *testing.T) {
	catalog := loadFixtureCatalog(t,
		Source{ProviderID: "prov-a", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-a", "model-a", "al-a"))},
		Source{ProviderID: "prov-c", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-c", "model-c", "al-c"))},
	)

	pairs, err := catalog.ResolveModelInput("prov-a/al-a")
	if err != nil || len(pairs) != 1 {
		t.Fatalf("💣 Resolve(prov-a/al-a) = (%+v, %v), want one model", pairs, err)
	}

	if pairs[0].Provider.ID != "prov-a" || pairs[0].Model.ID != "model-a" {
		t.Errorf("✗ Resolve(prov-a/al-a) = %s/%s, want prov-a/model-a", pairs[0].Provider.ID, pairs[0].Model.ID)
	}

	if !t.Failed() {
		t.Log("✓ a provider-qualified alias resolves to the model declaring it under that provider")
	}
}

// TestCompareProvidersOrder verifies invariant #4: Provider listing order.
//
// What is being tested:
// Given the mixed-case display names and an aggregator that sorts first alphabetically, sorting
// with CompareProviders must order the ordinary providers alpha, beta, zeta and place the
// aggregator last.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCompareProvidersOrder(t *testing.T) {
	providers := []Provider{
		{ID: "agg-a", DisplayName: "Aardvark Gateway", Aggregator: true},
		{ID: "zeta", DisplayName: "zeta"},
		{ID: "beta", DisplayName: "Beta"},
		{ID: "alpha", DisplayName: "alpha"},
	}
	slices.SortFunc(providers, CompareProviders)

	got := make([]string, 0, len(providers))
	for i := range providers {
		got = append(got, providers[i].ID)
	}

	if want := []string{"alpha", "beta", "zeta", "agg-a"}; !slices.Equal(got, want) {
		t.Errorf("✗ listing order = %v, want %v", got, want)
	}

	if !t.Failed() {
		t.Log("✓ providers order by display name with aggregators last")
	}
}

// TestQualifiedModelPrecedence verifies invariant #5: Explicit provider qualification.
//
// What is being tested:
// Given a qualified model and an identical slash-containing bare ID, ResolveModelInput must choose
// the qualified provider. If the named provider does not exist, it must resolve the matching bare
// ID under its provider.
//
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
func TestQualifiedModelPrecedence(t *testing.T) {
	catalog := &Catalog{Providers: []Provider{
		{ID: "bare", Models: []Model{{ID: "chosen/model"}, {ID: "missing/model"}}},
		{ID: "chosen", Models: []Model{{ID: "model"}}},
	}}
	for _, selection := range []struct{ input, provider string }{{"chosen/model", "chosen"}, {"missing/model", "bare"}} {
		matches, err := catalog.ResolveModelInput(selection.input)
		if err != nil || len(matches) != 1 {
			t.Errorf("✗ resolution %q: %v, %v", selection.input, matches, err)

			continue
		}

		if matches[0].Provider.ID != selection.provider {
			t.Errorf("✗ %q selected provider %q instead of %q", selection.input, matches[0].Provider.ID, selection.provider)
		}
	}

	if !t.Failed() {
		t.Log("✓ explicit qualification and slash-containing bare identifiers resolve correctly")
	}
}

// TestModelConfigIsolation verifies invariant #6: Model configuration isolation.
//
// What is being tested:
// Given each fixture model as its provider's only model, LoadCatalog and ConfigError must report no
// error. Across the fixture models, each alias must have only one owner.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestModelConfigIsolation(t *testing.T) {
	provCfg := urlResponseImageFixtureCfg(t)
	modelByAlias := map[string]string{}

	for _, modelConfig := range provCfg.Models {
		isolatedProvider := provCfg
		isolatedProvider.Models = []Model{modelConfig}

		providerDocument, err := json.Marshal(isolatedProvider)
		if err != nil {
			t.Fatalf("💣 encode isolated model %q: %v", modelConfig.ID, err)
		}

		isolatedCatalog, err := LoadCatalog(loadEnumFlags(t), Source{ProviderID: isolatedProvider.ID, ConfigBytes: providerDocument})
		if err == nil {
			err = isolatedCatalog.ConfigError(isolatedProvider.ID)
		}

		if err != nil {
			t.Errorf("✗ model %q does not validate independently: %v", modelConfig.ID, err)
		}

		for _, modelAlias := range modelConfig.Aliases {
			if owningModelID, aliasPresent := modelByAlias[modelAlias]; aliasPresent {
				t.Errorf("✗ alias %q belongs to both %q and %q", modelAlias, owningModelID, modelConfig.ID)
			}

			modelByAlias[modelAlias] = modelConfig.ID
		}
	}

	if !t.Failed() {
		t.Log("✓ every fixture model validates independently and every alias has one owner")
	}
}

// catalogFixtureCfg renders one minimal healthy config document for catalog fixtures.
func catalogFixtureCfg(test testing.TB, providerID, modelID, alias string) string {
	test.Helper()

	aliasField := ""
	if alias != "" {
		aliasField = fmt.Sprintf(`"aliases": [%q],`, alias)
	}

	return fmt.Sprintf(`{
	  "id": %[1]q, "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
	  "models": [
	    {"id": %[2]q, "name": %[2]q, "media": "image", %[3]s "params": [
	      {"paramID": "aspect_ratio", "flagID": "aspect-ratio"}
	    ]}
	  ],
	  "config": {
	  "imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaListProvParam": "images", "inputMediaStyle": %[4]q,
	    "fallbackExt": ".png"
	  }
	  }
}`, providerID, modelID, aliasField, InputMediaSingle)
}

// loadFixtureCatalog loads a catalog over the given sources and fails the test on a construction
// error.
func loadFixtureCatalog(t *testing.T, sources ...Source) *Catalog {
	t.Helper()

	catalog, err := LoadCatalog(params.Flags(), sources...)
	if err != nil {
		t.Fatalf("💣 catalog construction failed (cannot exercise the contract): %v", err)
	}

	return catalog
}

// urlResponseImageFixtureDocument supplies three image models with distinct parameter sets and
// aliases for independent configuration validation.
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

// urlResponseImageFixtureCfg loads the provider used for independent model validation.
func urlResponseImageFixtureCfg(t *testing.T) Provider {
	t.Helper()

	return decodeFixtureCfg(t, urlResponseImageFixtureDocument)
}

// decodeFixtureCfg decodes one fixture configuration document through the shared strict loader. The
// document body carries the models and config sections; the fixture provider identity is prepended.
func decodeFixtureCfg(t *testing.T, documentBody string) Provider {
	t.Helper()

	document := fmt.Sprintf(`{"id": %q, "displayName": %q, "apiKeyEnvVar": %q, %s}`,
		fixtureProviderID, fixtureProviderDisplayName, fixtureProviderKeyEnvVar, documentBody)

	provCfg := loadTestProvider(t, fixtureProviderID, []byte(document))

	return provCfg
}

// loadTestProvider loads one validated fixture provider through the catalog.
func loadTestProvider(test testing.TB, providerID string, configuration []byte) Provider {
	test.Helper()

	loadedCatalog, err := LoadCatalog(params.Flags(), Source{ProviderID: providerID, ConfigBytes: configuration})
	if err != nil {
		test.Fatalf("💣 fixture catalog failed to load: %v", err)
	}

	providerDescription, loaded := loadedCatalog.Provider(providerID)
	if !loaded {
		test.Fatalf("💣 fixture provider %s failed to load: %v", providerID, loadedCatalog.ConfigError(providerID))
	}

	return providerDescription
}

// loadEnumFlags returns the generated flag definitions used to validate fixture models.
func loadEnumFlags(t *testing.T) []params.Flag {
	t.Helper()

	flags := params.Flags()

	return flags
}

// Provider identity for the independent-model validation fixture.
//   - fixtureProviderID: the catalog identifier
//   - fixtureProviderDisplayName: the display name
//   - fixtureProviderKeyEnvVar: the declared credential environment variable
const (
	fixtureProviderID          = "fixture-provider"
	fixtureProviderDisplayName = "Fixture Provider"
	fixtureProviderKeyEnvVar   = "FIXTURE_API_KEY"
)
