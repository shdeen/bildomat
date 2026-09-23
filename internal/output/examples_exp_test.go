package output

import (
	"regexp"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
//  1. Optional example terms: For model IDs made only of slashes and hyphens, drawUsageExamples and
//     HelpTips must retain the real model key. Every nonempty generated regular expression must
//     compile and match that key, and help must omit search commands with missing arguments. When
//     keyword and alias examples are absent, footerData must return no match sentence.

// TestExamplesWithoutNameTokens verifies invariant #1: Optional example terms.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// For model IDs made only of slashes and hyphens, drawUsageExamples and HelpTips must retain the
// real model key. Every nonempty generated regular expression must compile and match that key, and
// help must omit search commands with missing arguments. When keyword and alias examples are
// absent, footerData must return no match sentence.
// Kind: permanent.
func TestExamplesWithoutNameTokens(t *testing.T) {
	for _, modelID := range []string{"/", "---", "/-//"} {
		t.Run(modelID, func(t *testing.T) {
			provider := catalog.Provider{ID: "provider", Models: []catalog.Model{{ID: modelID, Name: "Separator model", Media: media.Image}}}

			examples := drawUsageExamples(provider.ID, provider.Models)
			if len(examples.Keys) != 1 || examples.Keys[0] != provider.ID+"/"+modelID {
				t.Errorf("✗ the real model key was lost: %+v", examples.Keys)
			}

			for _, expression := range []string{examples.AnyPart, examples.AcrossTokens, examples.Alternation} {
				if expression == "" {
					continue
				}

				pattern, err := regexp.Compile(expression)
				if err != nil {
					t.Errorf("✗ unusable example expression %q: %v", expression, err)

					continue
				}

				if !pattern.MatchString(provider.ID + "/" + modelID) {
					t.Errorf("✗ example %q does not match the real model key", expression)
				}
			}

			help, err := HelpTips(&provider, false)
			if err != nil || !strings.Contains(help, provider.ID+"/"+modelID) {
				t.Errorf("✗ help could not render the model key: %q, %v", help, err)
			}

			for helpLine := range strings.SplitSeq(help, "\n") {
				command := strings.TrimSpace(helpLine)
				if command == "bild search" || command == "bild search -r" || command == "bild search -r ''" {
					t.Errorf("✗ an optional example has no argument: %q", helpLine)
				}
			}

			if examples.Keyword == "" && examples.AliasKeyword == "" && len(footerData(provider.ID, provider.Models).MatchSentence) != 0 {
				t.Errorf("✗ a missing keyword still produced a match sentence")
			}

			if !t.Failed() {
				t.Log("✓ separator-only IDs retain valid examples and omit unavailable terms")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ unusual IDs do not panic or invent optional examples")
	}
}
