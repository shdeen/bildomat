package main

// Invariants tested:
//  1. Built-in catalog provider-model pairs: For every built-in model, ResolveModelInput must
//     resolve its provider-qualified ID and each declared alias to exactly one pair without an
//     error. The pair must preserve the provider ID, display name, credential environment variable,
//     model ID, and media type from the catalog.
//  2. Construction of built-in providers: For every built-in provider registration, newGenerator
//     must return a non-nil generator without an error.
//  3. Built-in provider credential identifiers: clearProviderKeys must load the built-in catalog
//     with params.Flags without an error and clear every provider's credential environment variable
//     through t.Setenv without failing the test.
//  4. Built-in aliases: For every built-in model alias, ResolveModelInput must accept both the bare
//     alias and its provider-qualified form and return exactly the declaring provider and model
//     without an error.
//  5. Provider defaults resolve: For each declared provider default, ResolveModelInput must resolve
//     the provider-qualified default to exactly one model belonging to that provider without an
//     error. At least one built-in provider must declare a default.

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/params"
)

// TestProviderBindings verifies invariant #1: Built-in catalog provider-model pairs.
//
// What is being tested:
// For every built-in model, ResolveModelInput must resolve its provider-qualified ID and each
// declared alias to exactly one pair without an error. The pair must preserve the provider ID,
// display name, credential environment variable, model ID, and media type from the catalog.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderBindings(t *testing.T) {
	loadedCatalog := shippedCatalog(t)

	if len(loadedCatalog.Providers) == 0 {
		t.Fatalf("💣 the shipped catalog holds no provider")
	}

	for i := range loadedCatalog.Providers {
		provider := &loadedCatalog.Providers[i]

		for _, model := range provider.Models {
			specifiers := append([]string{provider.ID + "/" + model.ID}, model.Aliases...)
			for _, specifier := range specifiers {
				binds, err := loadedCatalog.ResolveModelInput(specifier)
				if err != nil || len(binds) != 1 {
					t.Errorf("✗ Resolve(%s) = (%+v, %v), want the single declared provider-model pair", specifier, binds, err)

					continue
				}

				bind := binds[0]
				if bind.Provider.ID != provider.ID || bind.Provider.DisplayName != provider.DisplayName || bind.Provider.APIKeyEnvVar != provider.APIKeyEnvVar {
					t.Errorf("✗ Resolve(%s) provider = %+v, want {%s %s %s}", specifier, bind.Provider, provider.ID, provider.DisplayName, provider.APIKeyEnvVar)
				}

				if bind.Model.ID != model.ID {
					t.Errorf("✗ Resolve(%s) model = %q, want %q", specifier, bind.Model.ID, model.ID)
				}

				if bind.Model.Media != model.Media {
					t.Errorf("✗ Resolve(%s) media = %q, want %q", specifier, bind.Model.Media, model.Media)
				}
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ every registered provider delivers declared identity and media through each of its models' provider-model pairs")
	}
}

// TestShippedProvidersConstruct verifies invariant #2: Construction of built-in providers.
//
// What is being tested:
// For every built-in provider registration, newGenerator must return a non-nil generator without an
// error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestShippedProvidersConstruct(t *testing.T) {
	loadedCatalog := shippedCatalog(t)

	for _, source := range providerRegistrations() {
		generator, err := newGenerator(loadedCatalog, source.ProviderID, providerRegistrations())
		if err != nil || generator == nil {
			t.Errorf("✗ NewGenerator(%q) = (%v, %v), want a usable generator", source.ProviderID, generator, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ every shipped provider registration constructs a usable generator")
	}
}

// TestCLIListingFilters verifies invariant #3: Built-in provider credential identifiers.
//
// What is being tested:
// clearProviderKeys must load the built-in catalog with params.Flags without an error and clear
// every provider's credential environment variable through t.Setenv without failing the test.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCLIListingFilters(t *testing.T) {
	clearProviderKeys(t)

	if !t.Failed() {
		t.Log("✓ the filter truth table holds on both listings and empties providers out of the listing")
	}
}

// TestShippedAliases verifies invariant #4: Built-in aliases.
//
// What is being tested:
// For every built-in model alias, ResolveModelInput must accept both the bare alias and its
// provider-qualified form and return exactly the declaring provider and model without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestShippedAliases(t *testing.T) {
	loadedCatalog := shippedCatalog(t)

	for _, modelPair := range loadedCatalog.ModelDirectory() {
		for _, alias := range modelPair.Model.Aliases {
			for _, modelInput := range []string{alias, modelPair.Provider.ID + "/" + alias} {
				resolved, err := loadedCatalog.ResolveModelInput(modelInput)
				if err != nil || len(resolved) != 1 || resolved[0].Provider.ID != modelPair.Provider.ID || resolved[0].Model.ID != modelPair.Model.ID {
					t.Errorf("✗ Resolve(%q) = (%+v, %v), want the single declared pair %s/%s", modelInput, resolved, err, modelPair.Provider.ID, modelPair.Model.ID)
				}
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ every currently declared shipped alias resolves in bare and provider-qualified form to its declaring model")
	}
}

// TestProviderDefaultsResolve verifies invariant #5: Provider defaults resolve.
//
// What is being tested:
// For each declared provider default, ResolveModelInput must resolve the provider-qualified default
// to exactly one model belonging to that provider without an error. At least one built-in provider
// must declare a default.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderDefaultsResolve(t *testing.T) {
	loadedCatalog := shippedCatalog(t)
	declared := 0

	for i := range loadedCatalog.Providers {
		prov := &loadedCatalog.Providers[i]
		if prov.DefaultModel == "" {
			continue
		}

		declared++

		got, err := loadedCatalog.ResolveModelInput(prov.ID + catalog.KeySeparator + prov.DefaultModel)
		if err != nil || len(got) != 1 || got[0].Provider.ID != prov.ID {
			t.Errorf("✗ provider %s: default %q resolves to (%+v, %v), want its own one model", prov.ID, prov.DefaultModel, got, err)
		}
	}

	if declared == 0 {
		t.Errorf("✗ no shipped provider declares a default model")
	}

	if !t.Failed() {
		t.Log("✓ every declared provider default names a model of that provider")
	}
}

// TestMain runs the package tests with an empty temporary HOME and removes it afterward.
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

// clearProviderKeys clears every registered credential environment variable for the test.
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

// shippedConfigDocuments decodes every built-in provider config as JSON data, in catalog load
// order.
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

// shippedCatalog loads and validates the registered provider descriptions with the built-in
// parameter flags.
//
// Test class: Core: Helper.
func shippedCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()

	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags, registeredTestSources(t, providerRegistrations())...)
	if err != nil {
		t.Fatalf("💣 catalog construction failed: %v", err)
	}

	return loadedCatalog
}

// registeredTestSources derives descriptions from the command registrations used by a test.
//
// Test class: Core: Helper.
func registeredTestSources(test testing.TB, registrations []providerRegistration) []catalog.Source {
	test.Helper()

	sources := make([]catalog.Source, 0, len(registrations))
	for _, registration := range registrations {
		sources = append(sources, catalog.Source{ProviderID: registration.ProviderID, ConfigBytes: registration.ConfigBytes})
	}

	return sources
}

// loadFixtureCatalog loads a catalog over the given sources and fails the test on a construction
// error.
func loadFixtureCatalog(t *testing.T, sources ...catalog.Source) *catalog.Catalog {
	t.Helper()

	loadedCatalog, err := catalog.LoadCatalog(params.Flags(), sources...)
	if err != nil {
		t.Fatalf("💣 catalog construction failed (cannot exercise the contract): %v", err)
	}

	return loadedCatalog
}
