package output

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/term"
)

// Invariants tested:
//  1. Help tips section: Given a provider with image and video models, HelpTips must return the
//     help heading, one valid model key per medium, an info command for each key, and a search
//     command for the provider. Every rendered search example must find a provider model. Each line
//     must fit the page width and have no trailing spaces.

// TestHelpTips verifies invariant #1: Help tips section.
//
// What is being tested:
// Given a provider with image and video models, HelpTips must return the help heading, one valid
// model key per medium, an info command for each key, and a search command for the provider. Every
// rendered search example must find a provider model. Each line must fit the page width and have no
// trailing spaces.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestHelpTips(t *testing.T) {
	prov, models := providerFixture(t)

	tips, err := HelpTips(providerWith(t, prov, models), term.IsTerminal(int(os.Stdout.Fd())))
	if err != nil {
		t.Fatalf("💣 the tips failed to render: %v", err)
	}

	keys := checkKeysOfProvider(t, "the tips", tips, prov, models)
	if len(keys) != 2 {
		t.Errorf("✗ the tips name %v as example keys, want one image and one video model", keys)
	}

	for _, key := range keys {
		if !strings.Contains(tips, "bild info "+key) {
			t.Errorf("✗ the key %s renders as no info example:\n%s", key, tips)
		}
	}

	if !strings.Contains(tips, HelpTipsHeading+"\n") {
		t.Errorf("✗ the tips carry no heading line:\n%s", tips)
	}

	checkRenderedSearchExamples(t, "the tips", tips, pairsOf(t, prov, models))

	if !strings.Contains(tips, "bild search "+prov.ID+"\n") {
		t.Errorf("✗ the tips carry no search example naming %s:\n%s", prov.ID, tips)
	}

	for _, tipLine := range pageLines(t, tips) {
		if textWidth(tipLine) > pageWidth {
			t.Errorf("✗ the line runs past the page width: %q", tipLine)
		}

		if strings.TrimRight(tipLine, " ") != tipLine {
			t.Errorf("✗ the line carries trailing whitespace: %q", tipLine)
		}
	}

	if !t.Failed() {
		t.Log("✓ the help tips draw one key per medium and working search examples from the provider")
	}
}
