package output

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. Provider listing: Given the fixture catalog, PrintListing must show provider names, model
//     IDs, aliases, and separate image and video headings, with Alpha Labs before Beta. Under the
//     video filter, it must show vid-one and omit img-one and the image-only Beta provider.
//  2. Model listing: Given the fixture catalog, PrintListing must return the four fully qualified
//     model keys in the specified provider and model order, one per line. With the image filter, it
//     must include alpha/img-one and omit vid-one.
//  3. Flag support note: For flags supported by every model in the relevant medium, FlagSupportNote
//     must return no note. For narrower support, it must name supporting providers, mark Alpha Labs
//     when only some of its image models support the flag, and omit individual model IDs and
//     unsupported providers.

// TestPrintProvidersListing verifies invariant #1: Provider listing.
//
// What is being tested:
// Given the fixture catalog, PrintListing must show provider names, model IDs, aliases, and
// separate image and video headings, with Alpha Labs before Beta. Under the video filter, it must
// show vid-one and omit img-one and the image-only Beta provider.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintProvidersListing(t *testing.T) {
	full := captureStdout(t, func() {
		_ = PrintListing(os.Stdout, ListingPage(catalog.SelectMedia(infoFixture(t), true, true), true, true), true, true, false)
	})
	for _, want := range []string{"Alpha Labs", "alpha", "Beta", "img-one", "img-three", "vid-one", "img-two", "ione", "vone", "Image models:", "Video models:"} {
		if !strings.Contains(full, want) {
			t.Errorf("✗ the unfiltered listing lacks %q:\n%s", want, full)
		}
	}

	if alphaAt, betaAt := strings.Index(full, "Alpha Labs"), strings.Index(full, "Beta"); alphaAt < 0 || betaAt < 0 || alphaAt > betaAt {
		t.Errorf("✗ provider order: Alpha Labs at %d, Beta at %d, expected alphabetical display names", alphaAt, betaAt)
	}

	videoOnly := captureStdout(t, func() {
		_ = PrintListing(os.Stdout, ListingPage(catalog.SelectMedia(infoFixture(t), false, true), true, true), true, true, false)
	})
	if strings.Contains(videoOnly, "Beta") || strings.Contains(videoOnly, "img-two") {
		t.Errorf("✗ a provider with no video models still renders under the video filter:\n%s", videoOnly)
	}

	if !strings.Contains(videoOnly, "vid-one") || strings.Contains(videoOnly, "img-one") {
		t.Errorf("✗ the video filter kept the wrong models:\n%s", videoOnly)
	}

	if !t.Failed() {
		t.Log("✓ the providers listing renders in order with media separation, and filtering omits emptied providers")
	}
}

// TestPrintModelsListing verifies invariant #2: Model listing.
//
// What is being tested:
// Given the fixture catalog, PrintListing must return the four fully qualified model keys in the
// specified provider and model order, one per line. With the image filter, it must include
// alpha/img-one and omit vid-one.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintModelsListing(t *testing.T) {
	allModels := captureStdout(t, func() {
		_ = PrintListing(os.Stdout, ListingPage(catalog.SelectMedia(infoFixture(t), true, true), true, false), false, true, false)
	})

	want := []string{"alpha/img-one", "alpha/img-three", "alpha/vid-one", "beta/img-two"}

	got := strings.Split(strings.TrimRight(allModels, "\n"), "\n")
	if len(got) != len(want) {
		t.Fatalf("💣 %d directory lines, want %d: %q (cannot compare positions)", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("✗ directory line %d = %q, want %q", i, got[i], want[i])
		}
	}

	imageOnly := strings.TrimRight(captureStdout(t, func() {
		_ = PrintListing(os.Stdout, ListingPage(catalog.SelectMedia(infoFixture(t), true, false), true, false), false, true, false)
	}), "\n")
	if strings.Contains(imageOnly, "vid-one") || !strings.Contains(imageOnly, "alpha/img-one") {
		t.Errorf("✗ the image filter kept the wrong keys: %q", imageOnly)
	}

	if !t.Failed() {
		t.Log("✓ the models listing preserves provider and model order and the media selection")
	}
}

// TestFlagSupportNote verifies invariant #3: Flag support note.
//
// What is being tested:
// For flags supported by every model in the relevant medium, FlagSupportNote must return no note.
// For narrower support, it must name supporting providers, mark Alpha Labs when only some of its
// image models support the flag, and omit individual model IDs and unsupported providers.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFlagSupportNote(t *testing.T) {
	pairs := infoFixture(t)

	cases := []struct {
		param   params.FlagType
		has     []string
		lacks   []string
		isEmpty bool
	}{
		// Every model of every provider consumes aspect-ratio: universal, no note.
		{param: params.FlagTypeAspect, isEmpty: true},
		// Only alpha's video model consumes duration, and alpha's video roster consumes it
		// whole; beta has no video models, so it neither supports nor withholds: universal
		// within the media scope, no note.
		{param: params.FlagTypeDuration, isEmpty: true},
		// Only img-one consumes quality — narrower than alpha's image roster, so the note
		// marks the provider as supporting only some of its models.
		{param: params.FlagTypeQuality, has: []string{fmt.Sprintf(ProviderSelectModels, "Alpha Labs")}, lacks: []string{"Beta", "img-one"}},
		// num-images: beta's image roster consumes it whole, so beta is named plainly;
		// alpha's only through img-one, so alpha is marked.
		{param: params.FlagTypeImageN, has: []string{"Beta", fmt.Sprintf(ProviderSelectModels, "Alpha Labs")}, lacks: []string{"img-one"}},
	}
	for _, c := range cases {
		note := FlagSupportNote(c.param, pairs)
		if c.isEmpty {
			if note != "" {
				t.Errorf("✗ FlagSupportNote(%s) = %q, want empty (universal support)", c.param, note)
			}

			continue
		}

		if note == "" {
			t.Errorf("✗ FlagSupportNote(%s) is empty, want a support note", c.param)

			continue
		}

		for _, want := range c.has {
			if !strings.Contains(note, want) {
				t.Errorf("✗ FlagSupportNote(%s) = %q, lacks %q", c.param, note, want)
			}
		}

		for _, absent := range c.lacks {
			if strings.Contains(note, absent) {
				t.Errorf("✗ FlagSupportNote(%s) = %q, unexpectedly carries %q", c.param, note, absent)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ the support note is empty for universal support, names a provider plainly when it supports the flag throughout, and marks it when only some models do")
	}
}

// infoFixture creates a two-provider catalog view exercising every constraint shape the information
// renderings must carry: allowed values, a two-bound numeric range, a lone maximum, a repeat
// maximum, size bounds, a rule description, and per-model aliases. Wire spellings are distinctive
// tokens so their absence from every rendering is assertable.
func infoFixture(test testing.TB) []catalog.ProvModelPair {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	alpha := catalog.Provider{ID: "alpha", DisplayName: "Alpha Labs", APIKeyEnvVar: "ALPHA_API_KEY"}
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	beta := catalog.Provider{ID: "beta", DisplayName: "Beta", APIKeyEnvVar: "BETA_API_KEY"}
	imgOne := catalog.Model{ID: "img-one", Name: "Image One", Media: media.Image, Aliases: []string{"ione"}, Params: []params.Definition{
		{ParamID: "aspect_wire", FlagID: params.FlagTypeAspect},
		{ParamID: "quality_wire", FlagID: params.FlagTypeQuality, AllowedValues: []string{"low", "high"}},
		{ParamID: "number_wire", FlagID: flagTypeNumberFixture, MinValue: params.GetSetIf(true, 0.0), MaxValue: params.GetSetIf(true, 2.0)},
		{ParamID: "count_wire", FlagID: params.FlagTypeImageN, MaxValue: params.GetSetIf(true, 10.0)},
		{FlagID: params.FlagTypeInputMedia, MaxMultiple: 4},
		{FlagID: params.FlagTypeSize, CustomSize: &params.SizeBounds{MinEdge: 64, MaxPx: 4194304, LongEdge: 1024}},
	}}
	imgThree := catalog.Model{ID: "img-three", Name: "Image Three", Media: media.Image, Params: []params.Definition{
		{ParamID: "aspect_wire", FlagID: params.FlagTypeAspect},
	}}
	vidOne := catalog.Model{ID: "vid-one", Name: "Video One", Media: media.Video, Aliases: []string{"vone"}, Params: []params.Definition{
		{ParamID: "aspect_wire", FlagID: params.FlagTypeAspect},
		{
			ParamID: "dur_wire", FlagID: params.FlagTypeDuration, AllowedValues: []string{"4", "6", "8"},
			RuleDescription: "forced to 8 on the fixture's test condition",
		},
	}}
	imgTwo := catalog.Model{ID: "img-two", Name: "Image Two", Media: media.Image, Params: []params.Definition{
		{ParamID: "aspect_wire", FlagID: params.FlagTypeAspect},
		{ParamID: "count_wire", FlagID: params.FlagTypeImageN, MaxValue: params.GetSetIf(true, 6.0)},
	}}

	return []catalog.ProvModelPair{
		{Provider: alpha, Model: imgOne},
		{Provider: alpha, Model: imgThree},
		{Provider: alpha, Model: vidOne},
		{Provider: beta, Model: imgTwo},
	}
}
