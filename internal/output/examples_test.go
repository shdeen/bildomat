package output

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"golang.org/x/term"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. Provider usage examples: For model IDs with hyphens, slashes, regular-expression characters,
//     or neither separator, drawUsageExamples must return one real model key per medium and
//     nonempty search patterns that find a model. The keyword must find its named model; any alias
//     keyword must prefix a declared alias, and absent aliases must produce no alias example. The
//     any-part pattern must match literal text inside an ID rather than its prefix. Every search
//     command rendered by HelpTips must find a provider model.

// TestUsageExamples verifies invariant #1: Provider usage examples.
//
// What is being tested:
// For model IDs with hyphens, slashes, regular-expression characters, or neither separator,
// drawUsageExamples must return one real model key per medium and nonempty search patterns that
// find a model. The keyword must find its named model; any alias keyword must prefix a declared
// alias, and absent aliases must produce no alias example. The any-part pattern must match literal
// text inside an ID rather than its prefix. Every search command rendered by HelpTips must find a
// provider model.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestUsageExamples(t *testing.T) {
	type usageFixture struct {
		name   string
		prov   catalog.Provider
		models []catalog.Model
	}

	hyphenatedProv, hyphenatedModels := providerFixture(t)
	slashedProv, slashedModels := aggregatorFixture(t)
	barrenProv, barrenModels := barrenFixture(t)
	metaProv, metaModels := metaFixture(t)

	fixtures := []usageFixture{
		{"hyphenated", hyphenatedProv, hyphenatedModels},
		{"slashed", slashedProv, slashedModels},
		{"barren", barrenProv, barrenModels},
		{"metacharacters", metaProv, metaModels},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			examples := drawUsageExamples(fixture.prov.ID, fixture.models)
			pairs := pairsOf(t, fixture.prov, fixture.models)

			var wantMedia []media.Kind

			for _, media := range []media.Kind{media.Image, media.Video} {
				if len(modelsOfMedia(fixture.models, media)) > 0 {
					wantMedia = append(wantMedia, media)
				}
			}

			if len(examples.Keys) != len(wantMedia) {
				t.Errorf("✗ %d example keys %v, want one per medium %v", len(examples.Keys), examples.Keys, wantMedia)
			}

			for i, key := range examples.Keys {
				if i >= len(wantMedia) {
					break
				}

				modelIndex := slices.IndexFunc(fixture.models, func(model catalog.Model) bool { return fixture.prov.ID+"/"+model.ID == key })
				if modelIndex < 0 || fixture.models[modelIndex].Media != wantMedia[i] {
					t.Errorf("✗ example key %s is not a %s model of the provider", key, wantMedia[i])
				}
			}

			if matches, err := catalog.SearchDirectory(pairs, examples.Keyword, "", false); err != nil || examples.Keyword == "" || !slices.ContainsFunc(matches, func(pair catalog.ProvModelPair) bool { return pair.Model.ID == examples.KeywordMatch }) {
				t.Errorf("✗ the keyword %q does not find its named model %q by prefix (%v)", examples.Keyword, examples.KeywordMatch, err)
			}

			hasAlias := slices.ContainsFunc(fixture.models, func(model catalog.Model) bool { return len(model.Aliases) > 0 })
			switch {
			case !hasAlias && (examples.AliasKeyword != "" || examples.AliasMatch != ""):
				t.Errorf("✗ a catalog without aliases drew the alias example %q for %q", examples.AliasKeyword, examples.AliasMatch)
			case hasAlias:
				aliasDeclared := slices.ContainsFunc(fixture.models, func(model catalog.Model) bool { return slices.Contains(model.Aliases, examples.AliasMatch) })
				if examples.AliasKeyword == "" || !aliasDeclared || !strings.HasPrefix(examples.AliasMatch, examples.AliasKeyword) {
					t.Errorf("✗ the alias keyword %q does not open a declared alias %q", examples.AliasKeyword, examples.AliasMatch)
				}
			}

			for name, pattern := range map[string]string{"alternation": examples.Alternation, "any-part": examples.AnyPart, "across-tokens": examples.AcrossTokens} {
				matches, err := catalog.SearchDirectory(pairs, pattern, "", true)
				if err != nil || pattern == "" || len(matches) == 0 {
					t.Errorf("✗ the %s pattern %q finds no model of the provider (%v)", name, pattern, err)
				}
			}

			if !strings.Contains(examples.Alternation, "|") {
				t.Errorf("✗ the alternation pattern %q carries no bar", examples.Alternation)
			}

			if !strings.Contains(examples.AcrossTokens, "/") {
				t.Errorf("✗ the across-tokens pattern %q spans no slash", examples.AcrossTokens)
			}

			tips, err := HelpTips(providerWith(t, fixture.prov, fixture.models), term.IsTerminal(int(os.Stdout.Fd())))
			if err != nil {
				t.Fatalf("💣 the %s tips failed to render: %v", fixture.name, err)
			}

			checkRenderedSearchExamples(t, "the "+fixture.name+" tips", tips, pairs)

			anyPartLiteral, unquoteErr := regexpLiteral(t, examples.AnyPart)
			if unquoteErr != nil {
				t.Errorf("✗ the any-part term %q is not a literal pattern: %v", examples.AnyPart, unquoteErr)
			}

			if slices.ContainsFunc(fixture.models, func(model catalog.Model) bool { return strings.HasPrefix(model.ID, anyPartLiteral) }) {
				t.Errorf("✗ the any-part term %q opens a model ID instead of sitting inside one", examples.AnyPart)
			}

			if !t.Failed() {
				t.Log("✓ every example names a model of the provider or finds one under the search rules")
			}
		})
	}
}

// metaFixture returns a provider whose model IDs carry regular-expression metacharacters, so every
// pattern example must escape its tokens.
func metaFixture(test testing.TB) (catalog.Provider, []catalog.Model) {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "meta", DisplayName: "Meta", APIKeyEnvVar: "META_API_KEY"}
	models := []catalog.Model{
		{ID: "a+b/c[1]-d.e", Name: "Plus", Media: media.Image, Aliases: []string{"p+q"}, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}},
		{ID: "x(y)-z*", Name: "Paren", Media: media.Video, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}},
	}

	return prov, models
}

// barrenFixture returns image and video models whose IDs contain no hyphens or slashes and whose
// alias lists are empty.
func barrenFixture(test testing.TB) (catalog.Provider, []catalog.Model) {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "bare", DisplayName: "Bare", APIKeyEnvVar: "BARE_API_KEY"}
	models := []catalog.Model{
		{ID: "solo", Name: "Solo", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}},
		{ID: "duo", Name: "Duo", Media: media.Video, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}},
	}

	return prov, models
}

// shellWords splits fixture commands at unquoted spaces and tabs. It preserves single-quoted text
// and processes backslash escapes outside quotes.
func shellWords(test testing.TB, commandLine string) []string {
	test.Helper()

	var words []string

	word, quoted, inWord := "", false, false

	for i := 0; i < len(commandLine); i++ {
		c := commandLine[i]

		switch {
		case quoted && c == '\'':
			quoted = false
		case quoted:
			word += string(c)
		case c == '\'':
			quoted, inWord = true, true
		case c == '\\' && i+1 < len(commandLine):
			i++
			word += string(commandLine[i])
			inWord = true
		case c == ' ' || c == '\t':
			if inWord {
				words = append(words, word)
				word, inWord = "", false
			}
		default:
			word += string(c)
			inWord = true
		}
	}

	if inWord {
		words = append(words, word)
	}

	return words
}

// checkRenderedSearchExamples splits each rendered bild search command with shellWords and checks
// that its search finds at least one supplied model. It also requires at least one search example.
func checkRenderedSearchExamples(t *testing.T, pageName, page string, pairs []catalog.ProvModelPair) {
	t.Helper()

	examples := 0

	for _, pageLine := range pageLines(t, page) {
		words := shellWords(t, strings.TrimSpace(pageLine))
		if len(words) < 3 || words[0] != "bild" || words[1] != "search" {
			continue
		}

		examples++
		useRegexp := slices.Contains(words[2:], "-r")
		term := words[len(words)-1]

		matches, err := catalog.SearchDirectory(pairs, term, "", useRegexp)
		if err != nil || len(matches) == 0 {
			t.Errorf("✗ %s shows the example %q, which finds no model of the provider (%v)", pageName, pageLine, err)
		}
	}

	if examples == 0 {
		t.Errorf("✗ %s shows no search example", pageName)
	}
}

// regexpLiteral takes a pattern and returns the literal text it matches, or an error when the
// pattern is not a plain literal.
func regexpLiteral(test testing.TB, pattern string) (string, error) {
	test.Helper()

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	literal, complete := compiled.LiteralPrefix()
	if !complete {
		return "", fmt.Errorf("%q is not a literal", pattern)
	}

	return literal, nil
}
