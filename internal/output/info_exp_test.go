package output

import (
	"os"
	"testing"

	"golang.org/x/term"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. Option boundary forms: Given custom flag records and one image model, PrintModelInfo must
//     show a lone decimal minimum, required allowed values, a long option heading, and a lone
//     maximum, with aligned continuation lines. PrintProviderInfo must show the model before its
//     options and report one image model. Both pages must fit the page width without trailing
//     whitespace.

// TestOptionDetailEdges verifies invariant #1: Option boundary forms.
//
// What is being tested:
// Given custom flag records and one image model, PrintModelInfo must show a lone decimal minimum,
// required allowed values, a long option heading, and a lone maximum, with aligned continuation
// lines. PrintProviderInfo must show the model before its options and report one image model. Both
// pages must fit the page width without trailing whitespace.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestOptionDetailEdges(t *testing.T) {
	flags := []params.Flag{
		{FlagID: flagTypeNumberFixture, DataType: params.DataNumber, FlagName: "Number fixture", TextHint: "number", Description: "A number fixture."},
		{FlagID: params.FlagTypeQuality, DataType: params.DataString, FlagName: "Quality", Aliases: []string{"q"}, TextHint: "level", Description: "Image quality level."},
		{FlagID: "thirty-two-wide-flag-name", DataType: params.DataString, FlagName: "Wide", TextHint: "value", Description: "A wide flag."},
		{FlagID: "seed", DataType: params.DataInteger, FlagName: "Seed", TextHint: "number", Description: "Sampling seed."},
	}
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	pair := catalog.ProvModelPair{
		Provider: catalog.Provider{ID: "solo", DisplayName: "Solo", APIKeyEnvVar: "SOLO_API_KEY"},
		Model: catalog.Model{ID: "only", Name: "Only", Media: media.Image, Params: []params.Definition{
			{FlagID: flagTypeNumberFixture, MinValue: params.GetSetIf(true, 0.5)},
			{FlagID: params.FlagTypeQuality, AllowedValues: []string{"low", "high"}, Required: true},
			{FlagID: "thirty-two-wide-flag-name"},
			{FlagID: "seed", MaxValue: params.GetSetIf(true, 99.0)},
		}},
	}

	card := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &pair, flags, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the edge card", card)

	lines := pageLines(t, card)
	headings := []string{flagHeading(t, flags, flagTypeNumberFixture), flagHeading(t, flags, params.FlagTypeQuality), flagHeading(t, flags, "thirty-two-wide-flag-name"), flagHeading(t, flags, "seed")}
	checkOption(t, lines, headings[0], "A number fixture.", "0.5")
	checkOption(t, lines, headings[1], RequiredLabel, "Image quality level.", "low", "high")
	checkOption(t, lines, headings[2], "A wide flag.")
	checkOption(t, lines, headings[3], "Sampling seed.", "99")
	checkOptionsAligned(t, "the edge card", lines, headings...)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, pair.Provider, []catalog.Model{pair.Model}), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the one-model page", page)
	checkCounts(t, "the one-model page", page, 1, 1, 0)
	checkInOrder(t, "the one-model page", page, "only", headings[0])

	if !t.Failed() {
		t.Log("✓ the lone minimum, the required mark among values, the wide heading, the bare parameter, and the one-model page render")
	}
}
