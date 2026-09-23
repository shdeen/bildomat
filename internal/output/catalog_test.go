package output

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. Independent alias visibility: When includeAliases changes from false to true and back,
//     ListingPage must retain the provider and model, show aliases only when requested, and leave
//     the source model's aliases unchanged.
//  2. Listing document: Given selected provider-model pairs, ListingPage must return provider
//     identities and the selected models without parameters. Disabling model output must return
//     identities alone; selecting video must exclude image models and providers with no video
//     model. Given unsorted providers and models, ListingPage must sort providers by display name
//     without regard to case, place aggregators last, and sort model IDs lexically, with beta-10
//     before beta-2. Provider-only output must use the same provider order.
//  3. Provider info document: Given a provider and selected models, ProviderInfoPage must retain
//     provider identity and model parameters, omit request configuration and parameter request
//     keys, and include each referenced flag once in flag-record order. Selecting images must
//     exclude the video model and its duration flag.
//  4. Model info document: Given one selected model, ModelInfoPage must return its provider
//     identity, that model with its required parameter mark, and only the referenced flags. It must
//     omit provider request configuration and parameter request keys.

// TestListingAliasSelection verifies invariant #1: Independent alias visibility.
//
// What is being tested:
// When includeAliases changes from false to true and back, ListingPage must retain the provider and
// model, show aliases only when requested, and leave the source model's aliases unchanged.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestListingAliasSelection(t *testing.T) {
	providerModels := []catalog.ProvModelPair{
		{
			Provider: catalog.Provider{ID: "provider"},
			Model:    catalog.Model{ID: "model", Media: media.Image, Aliases: []string{"shortcut"}},
		},
	}

	for _, includeAliases := range []bool{false, true, false} {
		listing := ListingPage(catalog.SelectMedia(providerModels, true, true), true, includeAliases)
		if len(listing.Providers) != 1 || len(listing.Providers[0].Models) != 1 {
			t.Fatalf("💣 the listing lost its provider or model: %+v", listing)
		}

		listedAliases := listing.Providers[0].Models[0].Aliases
		if includeAliases && !slices.Equal(listedAliases, []string{"shortcut"}) {
			t.Errorf("✗ an alias-enabled listing has aliases %q", listedAliases)
		}

		if !includeAliases && len(listedAliases) != 0 {
			t.Errorf("✗ a listing with hidden aliases contains %q", listedAliases)
		}

		if !slices.Equal(providerModels[0].Model.Aliases, []string{"shortcut"}) {
			t.Errorf("✗ rendering a listing changed the catalog's aliases")
		}
	}

	if !t.Failed() {
		t.Log("✓ alias visibility changes listings without changing catalog aliases")
	}
}

// TestListingDocument verifies invariant #2: Listing document.
//
// What is being tested:
// Given selected provider-model pairs, ListingPage must return provider identities and the selected
// models without parameters. Disabling model output must return identities alone; selecting video
// must exclude image models and providers with no video model.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestListingDocument(t *testing.T) {
	prov := documentFixture(t)
	pairs := pairsOf(t, prov, prov.Models)
	identity := identityOf(t, prov)

	withModels := identity
	withModels.Models = []catalog.Model{listedModel(t, prov.Models[0]), listedModel(t, prov.Models[1]), listedModel(t, prov.Models[2])}
	checkCatalog(t, "every model", ListingPage(catalog.SelectMedia(pairs, true, true), true, true), catalog.Catalog{Providers: []catalog.Provider{withModels}})

	checkCatalog(t, "providers only", ListingPage(catalog.SelectMedia(pairs, true, true), false, true), catalog.Catalog{Providers: []catalog.Provider{identity}})

	videoOnly := identity
	videoOnly.Models = []catalog.Model{listedModel(t, prov.Models[2])}
	checkCatalog(t, "video filter", ListingPage(catalog.SelectMedia(pairs, false, true), true, true), catalog.Catalog{Providers: []catalog.Provider{videoOnly}})

	imagePairs := pairsOf(t, prov, prov.Models[:2])
	checkCatalog(t, "no video model under the video filter", ListingPage(catalog.SelectMedia(imagePairs, false, true), true, true), catalog.Catalog{Providers: []catalog.Provider{}})

	if !t.Failed() {
		t.Log("✓ the listing document is the catalog reduced to the selected pairs")
	}
}

// TestListingOrder verifies invariant #2: Listing document.
//
// What is being tested:
// Given unsorted providers and models, ListingPage must sort providers by display name without
// regard to case, place aggregators last, and sort model IDs lexically, with beta-10 before beta-2.
// Provider-only output must use the same provider order.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestListingOrder(t *testing.T) {
	providers := []catalog.Provider{
		{ID: "first-aggregator", DisplayName: "Delta", Aggregator: true},
		{ID: "first-provider", DisplayName: "Zeta"},
		{ID: "second-aggregator", DisplayName: "alpha", Aggregator: true},
		{ID: "second-provider", DisplayName: "beta"},
	}
	models := []catalog.Model{
		{ID: "beta-2", Media: media.Image},
		{ID: "alpha-3", Media: media.Image},
		{ID: "beta-10", Media: media.Video},
	}

	providerModels := make([]catalog.ProvModelPair, 0, len(providers)*len(models))

	for _, provider := range providers {
		providerModels = append(providerModels, pairsOf(t, provider, models)...)
	}

	providerIDs := []string{"second-provider", "first-provider", "second-aggregator", "first-aggregator"}
	modelIDs := []string{"alpha-3", "beta-10", "beta-2"}

	for _, includeModels := range []bool{true, false} {
		listing := ListingPage(catalog.SelectMedia(providerModels, true, true), includeModels, true)
		if len(listing.Providers) != len(providerIDs) {
			t.Fatalf("💣 provider count = %d, expected %d", len(listing.Providers), len(providerIDs))
		}

		for providerIndex, provider := range listing.Providers {
			if provider.ID != providerIDs[providerIndex] {
				t.Errorf("✗ provider at position %d = %q, expected %q", providerIndex, provider.ID, providerIDs[providerIndex])
			}

			if !includeModels {
				continue
			}

			var listedModelIDs []string

			for _, model := range provider.Models {
				listedModelIDs = append(listedModelIDs, model.ID)
			}

			if !reflect.DeepEqual(listedModelIDs, modelIDs) {
				t.Errorf("✗ %s model order = %q, expected %q", provider.ID, listedModelIDs, modelIDs)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ provider names and model IDs are alphabetical, with aggregators last")
	}
}

// TestProviderInfoDocument verifies invariant #3: Provider info document.
//
// What is being tested:
// Given a provider and selected models, ProviderInfoPage must retain provider identity and model
// parameters, omit request configuration and parameter request keys, and include each referenced
// flag once in flag-record order. Selecting images must exclude the video model and its duration
// flag.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderInfoDocument(t *testing.T) {
	prov := documentFixture(t)
	records := documentFlags(t)

	whole := identityOf(t, prov)
	whole.Models = []catalog.Model{withoutRequestKeys(t, prov.Models[0]), withoutRequestKeys(t, prov.Models[1]), withoutRequestKeys(t, prov.Models[2])}
	checkCatalog(t, "every medium", ProviderInfoPage(&prov, catalog.SelectMedia(pairsOf(t, prov, prov.Models), true, true), records), catalog.Catalog{
		Providers: []catalog.Provider{whole},
		Flags:     flagsByID(t, records, params.FlagTypeDuration, params.FlagTypeImageN, params.FlagTypeSize, params.FlagTypeInputMedia),
	})

	imageOnly := identityOf(t, prov)
	imageOnly.Models = []catalog.Model{withoutRequestKeys(t, prov.Models[0]), withoutRequestKeys(t, prov.Models[1])}
	checkCatalog(t, "image filter", ProviderInfoPage(&prov, catalog.SelectMedia(pairsOf(t, prov, prov.Models), true, false), records), catalog.Catalog{
		Providers: []catalog.Provider{imageOnly},
		Flags:     flagsByID(t, records, params.FlagTypeImageN, params.FlagTypeSize, params.FlagTypeInputMedia),
	})

	if !t.Failed() {
		t.Log("✓ the provider info document is the catalog reduced to the provider, with params and their records")
	}
}

// TestModelInfoDocument verifies invariant #4: Model info document.
//
// What is being tested:
// Given one selected model, ModelInfoPage must return its provider identity, that model with its
// required parameter mark, and only the referenced flags. It must omit provider request
// configuration and parameter request keys.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestModelInfoDocument(t *testing.T) {
	prov := documentFixture(t)
	records := documentFlags(t)
	pair := catalog.ProvModelPair{Provider: prov, Model: prov.Models[1]}

	expected := identityOf(t, prov)
	expected.Models = []catalog.Model{withoutRequestKeys(t, prov.Models[1])}
	checkCatalog(t, "one model", ModelInfoPage(&pair, records), catalog.Catalog{
		Providers: []catalog.Provider{expected},
		Flags:     flagsByID(t, records, params.FlagTypeSize, params.FlagTypeInputMedia),
	})

	if !expected.Models[0].Params[1].Required {
		t.Errorf("✗ the fixture's required mark did not survive the reduction")
	}

	if !t.Failed() {
		t.Log("✓ the model info document is the catalog reduced to the one model")
	}
}

// documentFixture returns a provider with request configuration, two image models, and one video
// model. Its parameters include provider request keys and one required input-media declaration.
func documentFixture(test testing.TB) catalog.Provider {
	test.Helper()

	return catalog.Provider{ //nolint:gosec // the key variable is an environment-variable name, not a credential
		ID: "alpha", DisplayName: "Alpha Labs", APIKeyEnvVar: "ALPHA_API_KEY",
		Config: &catalog.ProviderConfig{StringParams: []params.FlagType{params.FlagTypeDuration}, ImageAPI: &catalog.ImageAPI{GenURL: "https://alpha.test/images"}},
		Models: []catalog.Model{
			{ID: "img-one", Name: "Image One", Description: "The first image model.", Media: media.Image, Aliases: []string{"one"}, Params: []params.Definition{
				{ParamID: "size_wire", FlagID: params.FlagTypeSize, AllowedValues: []string{"1024x1024"}},
				{ParamID: "n_wire", FlagID: params.FlagTypeImageN, MinValue: params.GetSetIf(true, 1.0), MaxValue: params.GetSetIf(true, 4.0)},
			}},
			{ID: "img-two", Name: "Image Two", Media: media.Image, Params: []params.Definition{
				{ParamID: "size_wire", FlagID: params.FlagTypeSize, AllowedValues: []string{"512x512"}},
				{FlagID: params.FlagTypeInputMedia, MaxMultiple: 2, Required: true},
			}},
			{ID: "vid-one", Name: "Video One", Media: media.Video, Params: []params.Definition{
				{ParamID: "seconds", FlagID: params.FlagTypeDuration, AllowedValues: []string{"4", "8"}},
			}},
		},
	}
}

// documentFlags returns flag records in a different order from the fixture parameters so tests can
// check which order the document uses.
func documentFlags(test testing.TB) []params.Flag {
	test.Helper()

	return []params.Flag{
		{FlagID: params.FlagTypeDuration, DataType: params.DataInteger, FlagName: "Duration", TextHint: "seconds"},
		{FlagID: params.FlagTypeImageN, DataType: params.DataInteger, FlagName: "Images", Aliases: []string{"N"}, TextHint: "count"},
		{FlagID: params.FlagTypeQuality, DataType: params.DataString, FlagName: "Quality", TextHint: "level"},
		{FlagID: params.FlagTypeSize, DataType: params.DataString, FlagName: "Size", Aliases: []string{"s"}, TextHint: "WxH"},
		{FlagID: params.FlagTypeInputMedia, DataType: params.DataString, FlagName: "Input media", Aliases: []string{"i"}, TextHint: "media-file", AllowMultiple: true},
	}
}

// identityOf returns the provider's identity alone: no models, no request settings.
func identityOf(test testing.TB, prov catalog.Provider) catalog.Provider {
	test.Helper()

	return catalog.Provider{ID: prov.ID, DisplayName: prov.DisplayName, APIKeyEnvVar: prov.APIKeyEnvVar, Aggregator: prov.Aggregator}
}

// listedModel returns a model as a listing carries it: without params.
func listedModel(test testing.TB, model catalog.Model) catalog.Model {
	test.Helper()

	listed := model
	listed.Params = nil

	return listed
}

// withoutRequestKeys returns a copy of the model with each parameter's provider request key
// cleared.
func withoutRequestKeys(test testing.TB, model catalog.Model) catalog.Model {
	test.Helper()

	detailed := model
	detailed.Params = make([]params.Definition, 0, len(model.Params))

	for _, param := range model.Params {
		param.ParamID = ""
		detailed.Params = append(detailed.Params, param)
	}

	return detailed
}

// flagsByID returns the records of the given flags, in the records' order.
func flagsByID(test testing.TB, records []params.Flag, ids ...params.FlagType) []params.Flag {
	test.Helper()

	var selected []params.Flag

	for _, record := range records {
		for _, id := range ids {
			if record.FlagID == id {
				selected = append(selected, record)
			}
		}
	}

	return selected
}

// checkCatalog compares every public JSON field, including omitted fields and array order.
func checkCatalog(t *testing.T, label string, document CatalogPage, expected catalog.Catalog) {
	t.Helper()

	documentBytes, documentErr := json.Marshal(document)

	expectedBytes, expectedErr := json.Marshal(expected)
	if documentErr != nil || expectedErr != nil {
		t.Fatalf("💣 document comparison encoding: %v, %v", documentErr, expectedErr)
	}

	var actualJSON, expectedJSON map[string]any
	if err := json.Unmarshal(documentBytes, &actualJSON); err != nil {
		t.Fatalf("💣 actual document decode: %v", err)
	}

	if err := json.Unmarshal(expectedBytes, &expectedJSON); err != nil {
		t.Fatalf("💣 expected document decode: %v", err)
	}

	for _, provider := range expectedJSON["providers"].([]any) {
		providerFields := provider.(map[string]any)
		providerFields["apiKeyConfigKey"] = "api-keys." + providerFields["id"].(string)
	}

	if !reflect.DeepEqual(actualJSON, expectedJSON) {
		t.Errorf("✗ %s: document %#v, expected %#v", label, actualJSON, expectedJSON)
	}
}
