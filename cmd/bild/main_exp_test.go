package main

// Invariants tested:
//  1. Generator construction: newGenerator must return nil and the specified classified error for
//     each invalid registration or construction result, and a non-nil generator without error for a
//     valid registration.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// TestCatalogNewGeneratorGuards verifies invariant #1: Generator construction.
//
// What is being tested:
// newGenerator must return no generator and the specified sentinel for a typed nil result, invalid
// provider configuration, absent constructor, nil interface result, or unregistered provider. A
// valid registration must return a non-nil generator without an error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCatalogNewGeneratorGuards(t *testing.T) {
	registrations := []providerRegistration{
		{ProviderID: "prov-a", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-a", "model-a")), New: nilPointerConstructor(t)},
		{ProviderID: "prov-b", ConfigBytes: []byte(`{ this is deliberately not JSON`), New: stubConstructor(t)},
		{ProviderID: "prov-c", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-c", "model-c"))},
		{ProviderID: "prov-d", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-d", "model-d")), New: stubConstructor(t)},
		{ProviderID: "prov-e", ConfigBytes: []byte(catalogFixtureCfg(t, "prov-e", "model-e")), New: nilInterfaceConstructor(t)},
	}

	sources := make([]catalog.Source, 0, len(registrations))
	for _, registration := range registrations {
		sources = append(sources, catalog.Source{ProviderID: registration.ProviderID, ConfigBytes: registration.ConfigBytes})
	}

	loadedCatalog := loadFixtureCatalog(t, sources...)

	constructionFailures := []struct {
		providerID string
		sentinel   error
	}{
		{providerID: "prov-a", sentinel: errs.ErrProvConfigConstructorFailed},
		{providerID: "prov-b", sentinel: errs.ErrProvConfigNotLoaded},
		{providerID: "prov-c", sentinel: errs.ErrProvConfigNoConstructor},
		{providerID: "prov-e", sentinel: errs.ErrProvConfigNilInterface},
		{providerID: "prov-ghost", sentinel: errs.ErrProvConfigNotLoaded},
	}

	for _, failure := range constructionFailures {
		if generator, err := newGenerator(loadedCatalog, failure.providerID, registrations); !errors.Is(err, failure.sentinel) || generator != nil {
			t.Errorf("✗ NewGenerator(%q) = (%v, %v), want %v and no generator", failure.providerID, generator, err, failure.sentinel)
		}
	}

	if generator, err := newGenerator(loadedCatalog, "prov-d", registrations); err != nil || generator == nil {
		t.Errorf("✗ NewGenerator(prov-d) = (%v, %v), want the constructed generator", generator, err)
	}

	if !t.Failed() {
		t.Log("✓ NewGenerator guards nil constructions, missing constructors, and broken configurations, and builds a healthy registration")
	}
}

// stubGenerator supplies an empty generator for constructor checks.
//   - test: the owning test context.
type stubGenerator struct{ test testing.TB }

// AdjustParams returns empty preparation without an error.
func (generator *stubGenerator) AdjustParams(*catalog.Model, params.FlagInputs, []media.Input, *metadata.Reuse) (generation.Preparation, error) {
	generator.test.Helper()

	return generation.Preparation{}, nil
}

// Generate returns an empty result without an error.
func (generator *stubGenerator) Generate(context.Context, *generation.Generation) (generation.Result, error) {
	generator.test.Helper()

	return generation.Result{}, nil
}

// stubConstructor returns a usable stub generator.
func stubConstructor(test testing.TB) func(*catalog.Provider) (generation.Generator, error) {
	test.Helper()

	return func(*catalog.Provider) (generation.Generator, error) { return &stubGenerator{test: test}, nil }
}

// nilPointerConstructor returns a constructor whose Generator result holds a nil *stubGenerator.
// That result compares unequal to a nil interface.
func nilPointerConstructor(test testing.TB) func(*catalog.Provider) (generation.Generator, error) {
	test.Helper()

	return func(*catalog.Provider) (generation.Generator, error) {
		var generator *stubGenerator

		return generator, nil
	}
}

// nilInterfaceConstructor returns a constructor that returns a nil Generator and no error.
func nilInterfaceConstructor(test testing.TB) func(*catalog.Provider) (generation.Generator, error) {
	test.Helper()

	return func(*catalog.Provider) (generation.Generator, error) { return nil, nil }
}

// catalogFixtureCfg renders one minimal healthy config document for catalog fixtures.
func catalogFixtureCfg(test testing.TB, providerID, modelID string) string {
	test.Helper()

	aliasField := ""

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
}`, providerID, modelID, aliasField, catalog.InputMediaSingle)
}
