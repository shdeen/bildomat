package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// File: internal/output/list.go The three text listings, their template functions, and the
// flag-support note used by help. Text and JSON listings use the same selected presentation records
// from catalog.go.

// modelListingLines returns fully qualified model keys with any included aliases in the supplied
// order; the flat directory template calls it.
func modelListingLines(providers []providerRecord) []string {
	var modelLines []string

	for i := range providers {
		for j := range providers[i].Models {
			model := &providers[i].Models[j]
			modelKey := providers[i].ID + catalog.KeySeparator + model.ID
			modelLines = append(modelLines, modelRowText(modelKey, model.Aliases))
		}
	}

	return modelLines
}

// FlagSupportNote names providers that support the flag, marking partial support as "select
// models". It considers only media kinds for which some model declares the flag. If every model in
// those media kinds supports it, the note is empty.
func FlagSupportNote(param params.FlagType, provModelPairs []catalog.ProvModelPair) string {
	scopeMedia := map[media.Kind]bool{}

	for i := range provModelPairs {
		pair := &provModelPairs[i]
		if pair.Model.SupportsParam(param) {
			scopeMedia[pair.Model.Media] = true
		}
	}

	var supporters []string

	universal := true

	providers := listedProviders(provModelPairs)

	for providerIndex := range providers {
		prov := &providers[providerIndex]
		inScope, supporting := 0, 0

		for pairIndex := range provModelPairs {
			pair := &provModelPairs[pairIndex]
			if pair.Provider.ID != prov.ID || !scopeMedia[pair.Model.Media] {
				continue
			}

			inScope++

			if pair.Model.SupportsParam(param) {
				supporting++
			}
		}

		switch {
		case inScope == 0:
			// Providers with no models in these media kinds do not affect the support
			// note.
		case supporting == 0:
			universal = false
		case supporting == inScope:
			supporters = append(supporters, prov.DisplayName)
		default:
			universal = false

			supporters = append(supporters, fmt.Sprintf(ProviderSelectModels, prov.DisplayName))
		}
	}

	if universal {
		return ""
	}

	return strings.Join(supporters, ", ")
}

// listedProviders takes provider-model pairs and returns each provider once in first-occurrence
// order.
func listedProviders(provModelPairs []catalog.ProvModelPair) []catalog.Provider {
	var providers []catalog.Provider

	listed := map[string]bool{}

	for pairIndex := range provModelPairs {
		pair := &provModelPairs[pairIndex]
		if listed[pair.Provider.ID] {
			continue
		}

		listed[pair.Provider.ID] = true
		providers = append(providers, pair.Provider)
	}

	return providers
}

// modelsOfMedia returns models of the requested medium in input order.
func modelsOfMedia(models []catalog.Model, mediaKind media.Kind) []catalog.Model {
	var kept []catalog.Model

	for i := range models {
		if models[i].Media == mediaKind {
			kept = append(kept, models[i])
		}
	}

	return kept
}

// modelRowText formats a model identifier with its aliases, or returns the identifier alone when
// there are no aliases.
func modelRowText(identifier string, aliases []string) string {
	if len(aliases) == 0 {
		return identifier
	}

	return fmt.Sprintf(ModelAliasListing, identifier, aliasWord(aliases), strings.Join(aliases, ", "))
}

// aliasWord returns the singular or plural alias label for an alias count.
func aliasWord(aliases []string) string {
	if len(aliases) > 1 {
		return AliasWordPlural
	}

	return AliasWordSingular
}

// PrintListing writes the selected provider/model listing as text or JSON. It returns template,
// encoding, and delivery failures to the command.
func PrintListing(destination io.Writer, page CatalogPage, providersSelected, modelsSelected, jsonOutput bool) error {
	if jsonOutput {
		return PrintJSON(destination, page)
	}

	switch {
	case providersSelected && modelsSelected:
		return writePage(destination, pageListNested, page)
	case providersSelected:
		return writePage(destination, pageListProviders, page)
	default:
		return writePage(destination, pageListModels, page)
	}
}
