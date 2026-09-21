package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// File: internal/output/list.go
// The three text listings the list and search commands render over a reduced
// catalog, the template functions those listings call, and the flag-support
// note the help page renders. The listings' JSON form is the reduced catalog
// itself (catalog.go).

// modelListingLines returns fully qualified model keys with any included
// aliases in the supplied order; the flat directory template calls it.
func modelListingLines(providers []catalog.Provider) []string {
	var modelLines []string

	for i := range providers {
		for j := range providers[i].Models {
			model := &providers[i].Models[j]
			modelKey := providers[i].ID + catalog.KeySeparator + model.ID
			modelLines = append(modelLines, modelNotice(modelKey, model.Aliases))
		}
	}

	return modelLines
}

// FlagSupportNote takes a parameter name and provider-model pairs and returns the providers
// that support the parameter for each media type that uses it. It returns an empty string
// when every provider with models of those media types fully supports the parameter and
// marks partial provider support as "select models".
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
			// No models of the flag's media: neither supports nor withholds.
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

// listedProviders takes provider-model pairs and returns each provider once in first-occurrence order.
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

// modelsOfMedia takes models and media and returns the matching models in
// input order; the nested listing template calls it per medium.
func modelsOfMedia(models []catalog.Model, mediaKind media.Kind) []catalog.Model {
	var kept []catalog.Model

	for i := range models {
		if models[i].Media == mediaKind {
			kept = append(kept, models[i])
		}
	}

	return kept
}

// modelNotice takes the identifier a listing shows for a model and the
// model's aliases, and returns the listing entry: the identifier with any
// aliases in the alias clause.
func modelNotice(identifier string, aliases []string) string {
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

// PrintListing writes the selected provider/model listing as text or JSON.
// It returns template, encoding, and delivery failures to the command.
func PrintListing(destination io.Writer, page catalog.Catalog, providersSelected, modelsSelected, jsonOutput bool) error {
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
