package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. Independent presentation records: ModelInfoPage must copy aliases, allowed values, size
//     bounds, flag aliases, and flag examples so that edits to the returned page leave the input
//     records unchanged. A later edit to the source model's aliases must leave an already prepared
//     ListingPage unchanged.

// TestPresentationRecordIsolation verifies invariant #1: Independent presentation records.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// ModelInfoPage must copy aliases, allowed values, size bounds, flag aliases, and flag examples so
// that edits to the returned page leave the input records unchanged. A later edit to the source
// model's aliases must leave an already prepared ListingPage unchanged.
// Kind: permanent.
func TestPresentationRecordIsolation(t *testing.T) {
	provider := documentFixture(t)
	provider.Models[0].Params[0].CustomSize = &params.SizeBounds{MinEdge: 16}
	flags := documentFlags(t)
	flags[3].ExampleValues = []string{"1024x1024"}
	pair := catalog.ProvModelPair{Provider: provider, Model: provider.Models[0]}

	sourceBefore, err := json.Marshal([]any{pair, flags})
	if err != nil {
		t.Fatalf("💣 source encoding: %v", err)
	}

	page := ModelInfoPage(&pair, flags)
	page.Providers[0].Models[0].Aliases[0] = "changed-alias"
	page.Providers[0].Models[0].Params[0].AllowedValues[0] = "32x32"

	page.Providers[0].Models[0].Params[0].CustomSize.MinEdge = 32
	for index := range page.Flags {
		if len(page.Flags[index].Aliases) > 0 {
			page.Flags[index].Aliases[0] = "changed-flag"
		}

		if len(page.Flags[index].ExampleValues) > 0 {
			page.Flags[index].ExampleValues[0] = "changed-example"
		}
	}

	sourceAfter, err := json.Marshal([]any{pair, flags})
	if err != nil {
		t.Fatalf("💣 source encoding after page mutation: %v", err)
	}

	if !bytes.Equal(sourceBefore, sourceAfter) {
		t.Errorf("✗ changing a presentation record changed its source: before %s, after %s", sourceBefore, sourceAfter)
	}

	listing := ListingPage([]catalog.ProvModelPair{pair}, true, true)

	listingBefore, err := json.Marshal(listing)
	if err != nil {
		t.Fatalf("💣 listing encoding: %v", err)
	}

	pair.Model.Aliases[0] = "later-source-alias"

	listingAfter, err := json.Marshal(listing)
	if err != nil {
		t.Fatalf("💣 listing encoding after source mutation: %v", err)
	}

	if !bytes.Equal(listingBefore, listingAfter) {
		t.Errorf("✗ source changes rewrote an already prepared listing: before %s, after %s", listingBefore, listingAfter)
	}

	if !t.Failed() {
		t.Log("✓ presentation records and their source definitions own independent values")
	}
}
