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

// Catalog contains validated providers, their flag definitions, and individual loading failures.
//   - Providers: the loaded providers in source order
//   - Flags: the parameter flag definitions used for validation
type Catalog struct {
	Providers []Provider    `json:"providers"`
	Flags     []params.Flag `json:"flags,omitempty"`
	// configErrors maps provider identifiers to configuration errors.
	configErrors map[string]error
}

// LoadCatalog loads provider descriptions in source order and validates them against flags.
// Individual configuration failures are retained in the catalog and exclude that provider; a source
// identity mismatch or a conflicting provider/model identifier or alias fails the load.
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

// ConfigError returns a provider's stored loading failure, or nil when none was recorded.
func (catalog *Catalog) ConfigError(providerID string) error {
	return catalog.configErrors[providerID]
}

// Provider returns a deep copy of the named provider and whether it was loaded.
func (catalog *Catalog) Provider(providerID string) (Provider, bool) {
	prov, ok := catalog.loadedProvider(providerID)
	if !ok {
		return Provider{}, false
	}

	return prov.clone(), true
}

// ResolveModelInput returns the provider and model pairs named by a model specifier. It checks
// explicit provider/model pairs, aliases, then unqualified model identifiers.
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

// CompareProviders sorts first-party providers before aggregators, then by display name without
// regard to case.
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

// DefaultModelKey returns the declared default of the first available provider in listing order. It
// reports false when no available provider declares a default.
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

// ModelDirectory returns the provider and model pairs of every loaded provider in load order, each
// carrying the provider's identity and a deep copy of the model.
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

// loadedProvider returns the catalog's stored provider, without copying nested collections.
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

// resolveQualified finds a model ID or alias within the named provider. It returns that provider's
// stored configuration failure, or reports no match if absent.
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

// resolveBareModelID returns each provider and model pair whose model identifier matches an
// unqualified specifier.
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
