package output

// Invariants tested:
// 1. Independent alias visibility: A listing includes aliases only when requested and leaves
//    the source aliases available for subsequent listings and catalog lookups.

import (
	"slices"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
)

// TestListingAliasSelection verifies invariant #1: Independent alias visibility.
//
// What makes it or breaks it:
// Switching alias visibility preserves every provider and model, includes aliases only when
// requested, and never changes the source model's aliases.
//
// Test class: Core.
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
