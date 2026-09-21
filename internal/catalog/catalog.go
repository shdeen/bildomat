// Package catalog loads, validates, and queries provider and model descriptions.
package catalog

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// Source identifies an embedded provider description supplied by the command.
//   - ProviderID: the provider identifier expected in the document
//   - ConfigBytes: the complete encoded provider description
type Source struct {
	ProviderID  string
	ConfigBytes []byte
}

// Catalog holds the providers the program serves. Loaded, it carries every
// provider whose configuration decoded and validated, in load order, the flag
// records those configurations were validated against, the load error of each
// provider that failed. Reduced, it is the
// JSON document of the list, search, and info commands: the providers and the
// flag records the command selects, and nothing of the loading state.
//   - Providers: the loaded providers in load order, or a document's selection
//   - Flags: the flag records the catalog was loaded against, or the records a
//     document's params reference
type Catalog struct {
	Providers []Provider    `json:"providers"`
	Flags     []params.Flag `json:"flags,omitempty"`
	// configErrors maps provider identifiers to configuration errors.
	configErrors map[string]error
}

// LoadCatalog decodes the registered sources, in order, and returns a validated
// catalog. A source that fails to decode or validate stores its error under its
// provider ID and withholds only that provider; a defect across sources (a
// document declaring another provider's ID, a duplicate alias, a duplicate
// provider-model pair) fails the whole load.
func LoadCatalog(flags []params.Flag, sources ...Source) (*Catalog, error) {
	catalog := &Catalog{
		Flags:        slices.Clone(flags),
		configErrors: map[string]error{},
	}
	for flagIndex := range catalog.Flags {
		catalog.Flags[flagIndex].Aliases = slices.Clone(catalog.Flags[flagIndex].Aliases)
		catalog.Flags[flagIndex].ExampleValues = slices.Clone(catalog.Flags[flagIndex].ExampleValues)
	}

	for _, source := range sources {
		cfgName := source.ProviderID + ".json"

		prov, err := loadProviderConfig(flags, cfgName, source.ConfigBytes)
		if err != nil {
			catalog.configErrors[source.ProviderID] = err

			continue
		}

		if prov.ID != source.ProviderID {
			fault := fmt.Sprintf(ConfigProviderSourceMismatch, cfgName, prov.ID, source.ProviderID)

			return nil, &errs.ConfigError{Path: cfgName, Provider: source.ProviderID, Problem: fault, Cause: errs.ErrProvConfigInvalid}
		}

		catalog.Providers = append(catalog.Providers, prov)
	}

	if err := catalog.check(); err != nil {
		return nil, err
	}

	return catalog, nil
}

// ConfigError returns the stored failure for a provider, or nil for a healthy description.
func (catalog *Catalog) ConfigError(providerID string) error {
	return catalog.configErrors[providerID]
}

// Provider takes a provider identifier and returns a deep copy of the loaded
// provider and whether the catalog holds it.
func (catalog *Catalog) Provider(providerID string) (Provider, bool) {
	prov, ok := catalog.loadedProvider(providerID)
	if !ok {
		return Provider{}, false
	}

	return prov.clone(), true
}

// ResolveModelInput returns the provider and model pairs named by a model specifier.
// It checks explicit provider/model pairs, aliases, then unqualified model identifiers.
func (catalog *Catalog) ResolveModelInput(modelSpecifier string) ([]ProvModelPair, error) {
	if providerID, modelID, ok := strings.Cut(modelSpecifier, "/"); ok {
		pair, found, configErr := catalog.resolveQualified(providerID, modelID)
		if configErr != nil {
			return nil, configErr
		}

		if found {
			return []ProvModelPair{pair}, nil
		}
	}

	if pair, aliased := catalog.resolveAlias(modelSpecifier); aliased {
		return []ProvModelPair{pair}, nil
	}

	return catalog.resolveBareModelID(modelSpecifier)
}

// CompareProviders takes two providers and orders them as every listing does:
// first-party providers before aggregators, and within each, by display name
// without regard to case.
//
//nolint:gocritic // slices.SortFunc requires value params.
func CompareProviders(firstProvider, secondProvider Provider) int {
	if firstProvider.Aggregator != secondProvider.Aggregator {
		if firstProvider.Aggregator {
			return 1
		}

		return -1
	}

	return strings.Compare(strings.ToLower(firstProvider.DisplayName), strings.ToLower(secondProvider.DisplayName))
}

// DefaultModelKey returns the fully qualified key of the default model the
// run uses when --model is omitted and the user config names none: the
// declared default of the first provider, in listing order, whose credential is available
// and which declares a default. It reports false when no provider qualifies.
func (catalog *Catalog) DefaultModelKey(availableProviders map[string]bool) (string, bool) {
	ordered := slices.Clone(catalog.Providers)
	slices.SortFunc(ordered, CompareProviders)

	for i := range ordered {
		prov := &ordered[i]
		if prov.DefaultModel == "" {
			continue
		}

		if !availableProviders[prov.ID] {
			continue
		}

		return prov.ID + KeySeparator + prov.DefaultModel, true
	}

	return "", false
}

// ModelDirectory returns the provider and model pairs of every loaded provider
// in load order, each carrying the provider's identity and a deep copy of the
// model.
func (catalog *Catalog) ModelDirectory() []ProvModelPair {
	var providerModelPairs []ProvModelPair

	for i := range catalog.Providers {
		prov := &catalog.Providers[i]
		for j := range prov.Models {
			providerModelPairs = append(providerModelPairs, ProvModelPair{Provider: prov.Identity(), Model: prov.Models[j].clone()})
		}
	}

	return providerModelPairs
}

// loadedProvider takes a provider identifier and returns the catalog's own
// provider value and whether the catalog holds it.
func (catalog *Catalog) loadedProvider(providerID string) (*Provider, bool) {
	for i := range catalog.Providers {
		if catalog.Providers[i].ID == providerID {
			return &catalog.Providers[i], true
		}
	}

	return nil, false
}

// resolveAlias returns the provider and model pair named by a model alias and whether it exists.
func (catalog *Catalog) resolveAlias(modelSpecifier string) (ProvModelPair, bool) {
	for i := range catalog.Providers {
		if modelPair, ok := catalog.Providers[i].findAlias(modelSpecifier); ok {
			return modelPair, true
		}
	}

	return ProvModelPair{}, false
}

// resolveQualified takes the provider ID and the model ID of a qualified
// specifier and returns the pair the provider declares under that model ID or
// alias and whether it declares one, or the provider's stored configuration
// error when its config failed to load. A provider the catalog does not hold
// declares nothing.
func (catalog *Catalog) resolveQualified(providerID, modelID string) (ProvModelPair, bool, error) {
	if configErr, hasError := catalog.configErrors[providerID]; hasError {
		return ProvModelPair{}, false, configErr
	}

	prov, ok := catalog.loadedProvider(providerID)
	if !ok {
		return ProvModelPair{}, false, nil
	}

	if pair, ok := prov.findModel(modelID); ok {
		return pair, true, nil
	}

	if pair, ok := prov.findAlias(modelID); ok {
		return pair, true, nil
	}

	return ProvModelPair{}, false, nil
}

// findModel returns the pair of the provider and the model carrying the model
// identifier, and whether the provider declares it.
func (prov *Provider) findModel(modelID string) (ProvModelPair, bool) {
	for i := range prov.Models {
		if prov.Models[i].ID == modelID {
			return ProvModelPair{Provider: prov.Identity(), Model: prov.Models[i].clone()}, true
		}
	}

	return ProvModelPair{}, false
}

// findAlias returns the pair of the provider and the model declaring the
// alias, and whether the provider declares it.
func (prov *Provider) findAlias(modelAlias string) (ProvModelPair, bool) {
	for i := range prov.Models {
		if slices.Contains(prov.Models[i].Aliases, modelAlias) {
			return ProvModelPair{Provider: prov.Identity(), Model: prov.Models[i].clone()}, true
		}
	}

	return ProvModelPair{}, false
}

// resolveBareModelID returns each provider and model pair whose model identifier matches an unqualified specifier.
func (catalog *Catalog) resolveBareModelID(modelSpecifier string) ([]ProvModelPair, error) {
	var modelMatches []ProvModelPair

	for i := range catalog.Providers {
		if pair, ok := catalog.Providers[i].findModel(modelSpecifier); ok {
			modelMatches = append(modelMatches, pair)
		}
	}

	if len(modelMatches) == 0 {
		return nil, &errs.ModelError{Specifier: modelSpecifier, Cause: errs.ErrModelResolveUnknown}
	}

	return modelMatches, nil
}

// check rejects duplicate provider/model pairs and aliases that hide another selection.
func (catalog *Catalog) check() error {
	modelOwners := map[string][]string{}
	selectedPairs := map[string]bool{}

	for providerIndex := range catalog.Providers {
		provider := &catalog.Providers[providerIndex]
		for modelIndex := range provider.Models {
			model := &provider.Models[modelIndex]

			modelKey := provider.ID + KeySeparator + model.ID
			if selectedPairs[modelKey] {
				return &errs.ConfigError{Provider: provider.ID, Problem: modelKey, Cause: errs.ErrProvConfigDupModel}
			}

			selectedPairs[modelKey] = true
			modelOwners[model.ID] = append(modelOwners[model.ID], modelKey)
			modelOwners[modelKey] = append(modelOwners[modelKey], modelKey)
		}
	}

	return catalog.checkModelAliases(modelOwners)
}

// checkModelAliases rejects repeated aliases and aliases that hide another model's identity.
func (catalog *Catalog) checkModelAliases(modelOwners map[string][]string) error {
	aliases := map[string]bool{}

	for providerIndex := range catalog.Providers {
		provider := &catalog.Providers[providerIndex]
		for modelIndex := range provider.Models {
			model := &provider.Models[modelIndex]

			modelKey := provider.ID + KeySeparator + model.ID
			for _, alias := range model.Aliases {
				collision := aliases[alias]
				for _, ownerKey := range modelOwners[alias] {
					collision = collision || ownerKey != modelKey
				}

				if collision {
					return &errs.ConfigError{Provider: provider.ID, Problem: fmt.Sprintf(AliasContextForm, modelKey, alias), Cause: errs.ErrProvConfigDupAlias}
				}

				aliases[alias] = true
			}
		}
	}

	return nil
}
