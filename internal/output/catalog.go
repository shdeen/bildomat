package output

import (
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/params"
)

// File: internal/output/catalog.go
// The JSON info pages of list, search, and info: the catalog reduced to what
// the command selects, carrying providers, models, params, and flag records
// under the configs' own keys. The reductions leave out the providers'
// request settings and the params' provider request keys, which no
// user-facing page carries.

// ListingPage takes the command-selected provider-model pairs and returns the
// requested listing without selecting the media again. Providers sort alphabetically by display name,
// ignoring case, with aggregators last. Their models sort by identifier and
// carry no params and include aliases only when requested. A listing selecting
// nothing carries an empty providers array.
func ListingPage(provModelPairs []catalog.ProvModelPair, withModels, includeAliases bool) catalog.Catalog {
	page := catalog.Catalog{Providers: []catalog.Provider{}}
	positions := map[string]int{}

	for i := range provModelPairs {
		pair := &provModelPairs[i]

		position, listed := positions[pair.Provider.ID]
		if !listed {
			position = len(page.Providers)
			positions[pair.Provider.ID] = position
			page.Providers = append(page.Providers, pair.Provider.Identity())
		}

		if withModels {
			model := pair.Model

			model.Params = nil
			if !includeAliases {
				model.Aliases = nil
			}

			page.Providers[position].Models = append(page.Providers[position].Models, model)
		}
	}

	for i := range page.Providers {
		slices.SortFunc(page.Providers[i].Models, compareModelIDs)
	}

	slices.SortFunc(page.Providers, catalog.CompareProviders)

	return page
}

// compareModelIDs orders model identifiers alphabetically.
//
//nolint:gocritic // slices.SortFunc requires value params.
func compareModelIDs(firstModel, secondModel catalog.Model) int {
	return strings.Compare(firstModel.ID, secondModel.ID)
}

// ProviderInfoPage takes a provider, the flag records, and the media
// selections, and returns the catalog reduced to that provider: its identity,
// its models of the selected media in config order carrying their params,
// and the flag records those params reference.
func ProviderInfoPage(prov *catalog.Provider, paramFlags []params.Flag, printImage, printVideo bool) catalog.Catalog {
	reduced := prov.Identity()

	for i := range prov.Models {
		if prov.Models[i].MediaSelected(printImage, printVideo) {
			reduced.Models = append(reduced.Models, detailedModel(&prov.Models[i]))
		}
	}

	return catalog.Catalog{Providers: []catalog.Provider{reduced}, Flags: referencedFlags(reduced.Models, paramFlags)}
}

// ModelInfoPage takes a provider-model pair and the flag records and
// returns the catalog reduced to that one model: its provider's identity, the
// model carrying its params, and the flag records those params reference.
func ModelInfoPage(provModelPair *catalog.ProvModelPair, paramFlags []params.Flag) catalog.Catalog {
	reduced := provModelPair.Provider.Identity()
	reduced.Models = []catalog.Model{detailedModel(&provModelPair.Model)}

	return catalog.Catalog{Providers: []catalog.Provider{reduced}, Flags: referencedFlags(reduced.Models, paramFlags)}
}

// detailedModel returns the model as an info page carries it: its params
// without their provider request keys.
func detailedModel(model *catalog.Model) catalog.Model {
	detailed := *model
	detailed.Params = slices.Clone(model.Params)

	for i := range detailed.Params {
		detailed.Params[i].ParamID = ""
	}

	return detailed
}

// referencedFlags takes models and the flag records and returns the records
// the models' params reference, once each, in the records' order.
func referencedFlags(models []catalog.Model, paramFlags []params.Flag) []params.Flag {
	referenced := map[params.FlagType]bool{}

	for i := range models {
		parameterValues := models[i].Params
		for j := range parameterValues {
			referenced[parameterValues[j].FlagID] = true
		}
	}

	var flags []params.Flag

	for i := range paramFlags {
		if referenced[paramFlags[i].FlagID] {
			flags = append(flags, paramFlags[i])
		}
	}

	return flags
}
