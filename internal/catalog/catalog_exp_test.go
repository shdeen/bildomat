package catalog

// Invariants tested:
//  1. Catalog isolation: Given healthy providers around a malformed provider, ResolveModelInput
//     must still resolve the healthy pair and alias, return the malformed provider's decode error
//     for a qualified request, and reject its unqualified names as unknown.
//  2. Catalog configuration error: Given the isolation fixture, ConfigError must return
//     ErrProvConfigDecode for the malformed provider and nil for the healthy and unknown provider
//     IDs.
//  3. Independent request settings: After mutating image fixed fields, video status lists and URL
//     path, and adapter status lists returned by Catalog.Provider, a later Provider call must
//     return each original value.
//  4. Duplicate aliases: Given two providers whose models declare the same alias, LoadCatalog must
//     return ErrProvConfigDupAlias under ErrProvConfig and name the alias.
//  5. Duplicate models: Given two models with the same ID under one provider, LoadCatalog must
//     return ErrProvConfigDupModel and name the provider/model pair.
//  6. Catalog source identity: Given a source ID that differs from the provider ID inside its JSON,
//     LoadCatalog must return ErrProvConfigInvalid.
//  7. Alias and model identity separation: Given an alias that names another model, Catalog.check
//     must return ErrProvConfig in either traversal order, within one provider or across two
//     providers.
//  8. Model specifier resolution under arbitrary input: For arbitrary model specifiers,
//     ResolveModelInput must return an error under ErrModelResolve or at least one provider/model
//     pair.

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// TestCatalogIsolation verifies invariant #1: Catalog isolation.
//
// What is being tested:
// Given healthy providers around a malformed provider, ResolveModelInput must still resolve the
// healthy pair and alias, return the malformed provider's decode error for a qualified request, and
// reject its unqualified names as unknown. ModelDirectory must list only healthy providers in load
// order, and Provider must omit the malformed provider.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCatalogIsolation(t *testing.T) {
	catalog := isolationCatalog(t)
	checkHealthyResolution(t, catalog)
	checkBrokenResolution(t, catalog)

	// The directory carries the healthy pairs in load order and omits prov-b.
	providerModelPairs := catalog.ModelDirectory()
	if len(providerModelPairs) != 2 || providerModelPairs[0].Provider.ID != "prov-a" || providerModelPairs[1].Provider.ID != "prov-c" {
		t.Errorf("✗ ModelDirectory() = %+v, want prov-a then prov-c with prov-b omitted", providerModelPairs)
	}

	// The broken provider's decoded config is unavailable; the healthy one is not.
	if _, ok := catalog.Provider("prov-b"); ok {
		t.Errorf("✗ Provider(prov-b) returned a provider for a broken config")
	}

	if prov, ok := catalog.Provider("prov-a"); !ok || prov.ID != "prov-a" {
		t.Errorf("✗ Provider(prov-a) = (%+v, %v), want the decoded provider", prov, ok)
	}

	if !t.Failed() {
		t.Log("✓ a broken config breaks only its own provider: stored error on the qualified path, unknown aliases and ids, omitted listings")
	}
}

// TestCatalogConfigError verifies invariant #2: Catalog configuration error.
//
// What is being tested:
// Given the isolation fixture, ConfigError must return ErrProvConfigDecode for the malformed
// provider and nil for the healthy and unknown provider IDs.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCatalogConfigError(t *testing.T) {
	catalog := isolationCatalog(t)

	storedErr := catalog.ConfigError("prov-b")

	broken := storedErr != nil
	if !broken || !errors.Is(storedErr, errs.ErrProvConfigDecode) {
		t.Errorf("✗ ConfigError(prov-b) = (%v, %v), want the stored decode error", storedErr, storedErr != nil)
	}

	if storedErr := catalog.ConfigError("prov-a"); storedErr != nil {
		t.Errorf("✗ ConfigError(prov-a) = (%v, %v), want none for a healthy provider", storedErr, storedErr != nil)
	}

	if storedErr := catalog.ConfigError("no-such"); storedErr != nil {
		t.Errorf("✗ ConfigError(no-such) = (%v, %v), want none for an unknown ID", storedErr, storedErr != nil)
	}

	if !t.Failed() {
		t.Log("✓ the stored config error is retrievable for the broken provider only")
	}
}

// TestCatalogCloneNestedCarriers verifies invariant #3: Independent request settings.
//
// What is being tested:
// After mutating image fixed fields, video status lists and URL path, and adapter status lists
// returned by Catalog.Provider, a later Provider call must return each original value.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCatalogCloneNestedCarriers(t *testing.T) {
	nestedCfg := `{
	  "id": "prov-n", "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
	  "models": [
	    {"id": "model-i", "name": "model-i", "media": "image", "params": []},
	    {"id": "model-v", "name": "model-v", "media": "video", "params": [
	      {"paramID": "seconds", "flagID": "duration", "allowedValues": ["4"]}
	    ]}
	  ],
	  "config": {
	  "imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": %q, "inputMediaListProvParam": "images",
	    "fixedProvFields": {"fixed": "declared"},
	    "fallbackExt": ".png"
	  },
	  "videoAPI": {
	    "asyncJobsURL": "https://example.test/vid", "urlStartPath": "name", "jobIDField": "id",
	    "progressStatusText": ["running"], "completedStatusText": "done", "failedStatusText": ["failed"],
	    "urlPathSeq": ["response", "video", "uri"],
	    "urlContentPath": "uri", "fallbackExt": ".mp4",
	    "pollInterval": 1, "pollTimeout": 2
	  },
	  "adapterAPI": {
	    "apiBase": "https://example.test/adapter", "pollInterval": 1, "pollTimeout": 2,
	    "pendingStatusText": ["Pending", "Reasoning", "Generating"],
		"readyStatusText": "Ready",
		"failedStatusText": ["Error"],
	    "imageFallbackExt": ".jpg", "videoFallbackExt": ".mp4"
	  }
	  }
}`
	catalog := loadFixtureCatalog(t,
		Source{ProviderID: "prov-n", ConfigBytes: []byte(fmt.Sprintf(nestedCfg, InputMediaSingle))})

	mutated, ok := catalog.Provider("prov-n")
	if !ok {
		t.Fatalf("💣 the nested fixture did not load (cannot exercise the contract)")
	}

	mutated.Config.ImageAPI.FixedProvFields["fixed"] = "tampered"
	mutated.Config.VideoAPI.ProgressStatusText[0] = "tampered"
	mutated.Config.VideoAPI.FailedStatusText[0] = "tampered"
	mutated.Config.VideoAPI.URLPathSeq[0] = "tampered"
	mutated.Config.AdapterAPI.PendingStatusText[0] = "tampered"
	mutated.Config.AdapterAPI.FailedStatusText[0] = "tampered"

	served, _ := catalog.Provider("prov-n")
	if served.Config.ImageAPI.FixedProvFields["fixed"] != "declared" {
		t.Errorf("✗ mutating a served copy's ImageAPI.FixedProvFields reached the catalog")
	}

	if served.Config.VideoAPI.ProgressStatusText[0] != "running" {
		t.Errorf("✗ mutating a served copy's VideoAPI.ProgressStatusText reached the catalog")
	}

	if served.Config.VideoAPI.FailedStatusText[0] != "failed" {
		t.Errorf("✗ mutating a served copy's VideoAPI.FailedStatusText reached the catalog")
	}

	if served.Config.VideoAPI.URLPathSeq[0] != "response" {
		t.Errorf("✗ mutating a served copy's VideoAPI.URLPathSeq reached the catalog")
	}

	if served.Config.AdapterAPI.PendingStatusText[0] != "Pending" {
		t.Errorf("✗ mutating a served copy's AdapterAPI.PendingStatusText reached the catalog")
	}

	if served.Config.AdapterAPI.FailedStatusText[0] != "Error" {
		t.Errorf("✗ mutating a served copy's AdapterAPI.FailedStatusText reached the catalog")
	}

	if !t.Failed() {
		t.Log("✓ every nested map and slice served by the catalog is isolated from caller mutation")
	}
}

// TestCatalogDupAlias verifies invariant #4: Duplicate aliases.
//
// What is being tested:
// Given two providers whose models declare the same alias, LoadCatalog must return
// ErrProvConfigDupAlias under ErrProvConfig and name the alias.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCatalogDupAlias(t *testing.T) {
	_, err := LoadCatalog(params.Flags(),
		Source{ProviderID: "prov-a", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-a", "model-a", "shared"))},
		Source{ProviderID: "prov-c", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-c", "model-c", "shared"))},
	)
	if !errors.Is(err, errs.ErrProvConfigDupAlias) || !errors.Is(err, errs.ErrProvConfig) {
		t.Errorf("✗ a cross-config alias collision = %v, want the duplicate-alias sentinel under the provider-config root", err)
	}

	if err != nil && !strings.Contains(err.Error(), "shared") {
		t.Errorf("✗ the collision error does not name the alias: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ an alias declared twice fails catalog construction with the new precise sentinel")
	}
}

// TestCatalogDupModel verifies invariant #5: Duplicate models.
//
// What is being tested:
// Given two models with the same ID under one provider, LoadCatalog must return
// ErrProvConfigDupModel and name the provider/model pair.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCatalogDupModel(t *testing.T) {
	dupModelCfg := strings.Replace(catalogFixtureCfg(t, "prov-a", "model-a", ""),
		`"models": [`,
		`"models": [
	    {"id": "model-a", "name": "model-a", "media": "image", "params": [
	      {"paramID": "aspect_ratio", "flagID": "aspect-ratio"}
	    ]},`, 1)

	_, err := LoadCatalog(params.Flags(),
		Source{ProviderID: "prov-a", ConfigBytes: []byte(dupModelCfg)},
	)
	if !errors.Is(err, errs.ErrProvConfigDupModel) {
		t.Errorf("✗ a duplicate provider-model pair = %v, want the duplicate-pair sentinel", err)
	}

	if err != nil && !strings.Contains(err.Error(), "prov-a/model-a") {
		t.Errorf("✗ the duplicate-pair error does not name the pair: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a duplicate provider-model pair fails catalog construction with the new precise sentinel")
	}
}

// TestCatalogSourceIdentity verifies invariant #6: Catalog source identity.
//
// What is being tested:
// Given a source ID that differs from the provider ID inside its JSON, LoadCatalog must return
// ErrProvConfigInvalid.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCatalogSourceIdentity(t *testing.T) {
	_, err := LoadCatalog(params.Flags(),
		Source{ProviderID: "prov-a", ConfigBytes: []byte(catalogFixtureCfg(t, "other", "model-a", ""))},
	)
	if !errors.Is(err, errs.ErrProvConfigInvalid) {
		t.Errorf("✗ a source/identity disagreement = %v, want the invalid-content sentinel", err)
	}

	if !t.Failed() {
		t.Log("✓ a config whose identity disagrees with its declared provider ID fails loudly")
	}
}

// TestCatalogAliasModelCollision verifies invariant #7: Alias and model identity separation.
//
// What is being tested:
// Given an alias that names another model, Catalog.check must return ErrProvConfig in either
// traversal order, within one provider or across two providers. It must accept a unique alias equal
// to its own model ID.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestCatalogAliasModelCollision(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, sameProvider := range []bool{false, true} {
			providers := []Provider{
				{ID: "first", Models: []Model{{ID: "original", Aliases: []string{"shadowed"}}}},
				{ID: "second", Models: []Model{{ID: "shadowed"}}},
			}
			if sameProvider {
				providers[0].Models = append(providers[0].Models, providers[1].Models...)

				providers = providers[:1]
				if reverse {
					slices.Reverse(providers[0].Models)
				}
			} else if reverse {
				slices.Reverse(providers)
			}

			if err := (&Catalog{Providers: providers}).check(); !errors.Is(err, errs.ErrProvConfig) {
				t.Errorf("✗ alias shadow accepted (reverse=%v, same provider=%v): %v", reverse, sameProvider, err)
			}
		}
	}

	uniqueSelfAlias := &Catalog{Providers: []Provider{{ID: "owner", Models: []Model{{ID: "model", Aliases: []string{"model"}}}}}}
	if err := uniqueSelfAlias.check(); err != nil {
		t.Errorf("✗ unique self-ID alias rejected: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ model identity remains selectable regardless of declaration order")
	}
}

// FuzzResolve verifies invariant #8: Model specifier resolution under arbitrary input.
//
// What is being tested:
// For arbitrary model specifiers, ResolveModelInput must return an error under ErrModelResolve or
// at least one provider/model pair. Every successful pair must appear in that catalog's
// ModelDirectory.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzResolve(f *testing.F) {
	for _, seed := range []string{"al-a", "model-a", "prov-a/model-a", "prov-b/x", "", "a/b/c", "AL-A", " model-a ", "no-such"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, modelSpecifier string) {
		catalog := loadFixtureCatalog(t,
			Source{ProviderID: "prov-a", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-a", "model-a", "al-a"))},
			Source{ProviderID: "prov-c", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-c", "model-c", ""))},
		)

		provModelPairs, err := catalog.ResolveModelInput(modelSpecifier)
		if err != nil {
			if !errors.Is(err, errs.ErrModelResolve) {
				t.Errorf("✗ resolution of %q failed outside the model-resolution category: %v", modelSpecifier, err)
			}

			return
		}

		if len(provModelPairs) == 0 {
			t.Errorf("✗ resolution of %q returned neither a pair nor a failure", modelSpecifier)

			return
		}

		for _, provModelPair := range provModelPairs {
			servedPair := false

			for _, directoryPair := range catalog.ModelDirectory() {
				if directoryPair.Provider.ID == provModelPair.Provider.ID && directoryPair.Model.ID == provModelPair.Model.ID {
					servedPair = true

					break
				}
			}

			if !servedPair {
				t.Errorf("✗ resolution of %q returned %s/%s, which the catalog does not serve", modelSpecifier, provModelPair.Provider.ID, provModelPair.Model.ID)
			}
		}

		if !t.Failed() {
			t.Logf("✓ resolution is total: a classified failure or served pairs")
		}
	})
}

// isolationCatalog is the isolation fixture: healthy prov-a and prov-c around a corrupted prov-b
// source.
func isolationCatalog(t *testing.T) *Catalog {
	t.Helper()

	return loadFixtureCatalog(t,
		Source{ProviderID: "prov-a", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-a", "model-a", "al-a"))},
		Source{ProviderID: "prov-b", ConfigBytes: []byte(`{ this is deliberately not JSON`)},
		Source{ProviderID: "prov-c", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-c", "model-c", ""))},
	)
}

// checkHealthyResolution asserts the healthy providers still resolve by pair and alias beside the
// broken source.
func checkHealthyResolution(t *testing.T, catalog *Catalog) {
	t.Helper()

	if pairs, err := catalog.ResolveModelInput("prov-a/model-a"); err != nil || len(pairs) != 1 || pairs[0].Model.ID != "model-a" {
		t.Errorf("✗ Resolve(prov-a/model-a) = (%+v, %v), want the healthy pair alone", pairs, err)
	}

	if pairs, err := catalog.ResolveModelInput("al-a"); err != nil || len(pairs) != 1 || pairs[0].Model.ID != "model-a" {
		t.Errorf("✗ Resolve(al-a) = (%+v, %v), want the healthy alias resolution alone", pairs, err)
	}
}

// checkBrokenResolution asserts the broken provider's resolution outcomes: the stored decode error
// on the qualified path, unknown specifiers for its aliases and bare ids.
func checkBrokenResolution(t *testing.T, catalog *Catalog) {
	t.Helper()

	_, err := catalog.ResolveModelInput("prov-b/model-b")
	if !errors.Is(err, errs.ErrProvConfigDecode) {
		t.Errorf("✗ Resolve(prov-b/model-b) = %v, want the stored decode error", err)
	}

	if err != nil && !strings.Contains(err.Error(), "prov-b.json") {
		t.Errorf("✗ the stored error does not name the broken config: %v", err)
	}

	for _, specifier := range []string{"al-b", "model-b"} {
		if _, err := catalog.ResolveModelInput(specifier); !errors.Is(err, errs.ErrModelResolveUnknown) {
			t.Errorf("✗ Resolve(%q) = %v, want an unknown-specifier error", specifier, err)
		}
	}
}
