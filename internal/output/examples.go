package output

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
)

// File: internal/output/examples.go
// The usage examples the help tips and the aggregator summary's footer draw
// from a provider's catalog:
// one model key per medium and the search terms that illustrate the search
// rules, each chosen at random so that every example is a command that
// returns something for this provider.

// The measures and separators of the examples. A key's tokens are split on
// the key separator and a name's tokens on the dash.
//   - keywordLength: the characters of a token the keyword example takes
//   - minTermLength: the shortest token a pattern example draws
//   - alternationSeparator: the bar between the terms of an alternation pattern
//   - numericTokenChars: the characters a token made of digits and dots consists of
const (
	keywordLength        = 3
	minTermLength        = 3
	alternationSeparator = "|"
	numericTokenChars    = "0123456789."
)

// usageExamples carries the examples a page's footer names.
//   - Keys: one fully qualified key per medium the models span, image first
//   - Keyword: a term some model's slash-delimited token begins with
//   - KeywordMatch: the bare ID of the model the keyword matches
//   - AliasKeyword: a term some alias begins with, empty when no model declares an alias
//   - AliasMatch: the alias the alias keyword matches
//   - Alternation: a pattern of two terms joined by a bar
//   - AnyPart: a term found inside a model's ID and opening no model's ID
//   - AcrossTokens: a pattern spanning two consecutive slash-delimited tokens of a key
type usageExamples struct {
	Keys         []string
	Keyword      string
	KeywordMatch string
	AliasKeyword string
	AliasMatch   string
	Alternation  string
	AnyPart      string
	AcrossTokens string
}

// drawUsageExamples takes a provider ID and the models a page shows, and
// returns the footer's examples drawn at random from them; no models draw
// no examples.
func drawUsageExamples(providerID string, models []catalog.Model) usageExamples {
	if len(models) == 0 {
		return usageExamples{}
	}

	keywordModel := randomModel(preferInnerTokens(models))
	examples := usageExamples{
		Keys:         exampleKeys(providerID, models),
		Keyword:      keywordExample(keywordModel),
		KeywordMatch: keywordModel.ID,
		Alternation:  alternationExample(providerID, models),
		AnyPart:      anyPartExample(models),
		AcrossTokens: acrossTokensExample(providerID, randomModel(preferInnerTokens(models))),
	}
	examples.AliasKeyword, examples.AliasMatch = aliasExample(models)

	return examples
}

// randomModel takes models and returns one of them at random.
func randomModel(models []catalog.Model) *catalog.Model {
	// #nosec G404 -- the examples need variety, not unpredictability.
	return &models[rand.IntN(len(models))]
}

// preferInnerTokens takes models and returns those whose bare ID holds a
// slash, or all of them when none does, so an example can illustrate an
// inner token wherever the catalog has one.
func preferInnerTokens(models []catalog.Model) []catalog.Model {
	var slashed []catalog.Model

	for i := range models {
		if strings.Contains(models[i].ID, catalog.KeySeparator) {
			slashed = append(slashed, models[i])
		}
	}

	if len(slashed) == 0 {
		return models
	}

	return slashed
}

// exampleKeys takes a provider ID and models and returns one fully qualified
// key per medium with a model, image first, each model chosen at random.
func exampleKeys(providerID string, models []catalog.Model) []string {
	var keys []string

	for _, media := range []media.Kind{media.Image, media.Video} {
		if mediaModels := modelsOfMedia(models, media); len(mediaModels) > 0 {
			keys = append(keys, providerID+catalog.KeySeparator+randomModel(mediaModels).ID)
		}
	}

	return keys
}

// keywordExample takes a model and returns the opening characters of the
// last slash-delimited token of its ID, the whole token when it is shorter.
func keywordExample(model *catalog.Model) string {
	tokens := strings.Split(model.ID, catalog.KeySeparator)
	last := []rune(tokens[len(tokens)-1])

	if len(last) <= keywordLength {
		return string(last)
	}

	return string(last[:keywordLength])
}

// aliasExample takes models and returns a term some alias begins with — the
// first hyphen-delimited token of an alias chosen at random — and that
// alias; empty strings when no model declares an alias.
func aliasExample(models []catalog.Model) (term, alias string) {
	var aliases []string
	for i := range models {
		aliases = append(aliases, models[i].Aliases...)
	}

	if len(aliases) == 0 {
		return "", ""
	}

	// #nosec G404 -- the examples need variety, not unpredictability.
	alias = aliases[rand.IntN(len(aliases))]
	term, _, _ = strings.Cut(alias, "-")

	return term, alias
}

// alternationExample takes a provider ID and models and returns a pattern of
// two distinct hyphen-delimited tokens of the models' IDs joined by a bar,
// digits-only and short tokens excluded, chosen at random; when fewer than two such
// tokens exist, the provider ID and a random model's ID. Each term is
// escaped so that it matches itself literally.
func alternationExample(providerID string, models []catalog.Model) string {
	var tokens []string

	for i := range models {
		for _, token := range nameTokens(models[i].ID) {
			if exampleTerm(token) && !slices.Contains(tokens, token) {
				tokens = append(tokens, token)
			}
		}
	}

	if len(tokens) < 2 {
		return regexp.QuoteMeta(providerID) + alternationSeparator + regexp.QuoteMeta(randomModel(models).ID)
	}

	// #nosec G404 -- the examples need variety, not unpredictability.
	order := rand.Perm(len(tokens))

	return regexp.QuoteMeta(tokens[order[0]]) + alternationSeparator + regexp.QuoteMeta(tokens[order[1]])
}

// anyPartExample takes models and returns a hyphen-delimited token found
// inside a model's ID — never its first token, never digits only or short,
// and never the opening of any model's ID — chosen at random; when none exists, a
// random model's ID without its first character. The term is escaped so
// that it matches itself literally.
func anyPartExample(models []catalog.Model) string {
	var inner []string

	for i := range models {
		for _, token := range nameTokens(models[i].ID)[1:] {
			if exampleTerm(token) && !opensAnyID(models, token) && !slices.Contains(inner, token) {
				inner = append(inner, token)
			}
		}
	}

	if len(inner) == 0 {
		id := []rune(randomModel(models).ID)

		return regexp.QuoteMeta(string(id[1:]))
	}

	// #nosec G404 -- the examples need variety, not unpredictability.
	return regexp.QuoteMeta(inner[rand.IntN(len(inner))])
}

// acrossTokensExample takes a provider ID and a model and returns a pattern
// spanning two consecutive slash-delimited tokens of the model's fully
// qualified key: the last hyphen-delimited token of the first and the first
// hyphen-delimited token of the second, over the key's last slash, each
// escaped so that it matches itself literally.
func acrossTokensExample(providerID string, model *catalog.Model) string {
	tokens := strings.Split(providerID+catalog.KeySeparator+model.ID, catalog.KeySeparator)
	before := nameTokens(tokens[len(tokens)-2])
	after := nameTokens(tokens[len(tokens)-1])

	if len(before) == 0 || len(after) == 0 {
		return regexp.QuoteMeta(tokens[len(tokens)-2]) + catalog.KeySeparator + regexp.QuoteMeta(tokens[len(tokens)-1])
	}

	return regexp.QuoteMeta(before[len(before)-1]) + catalog.KeySeparator + regexp.QuoteMeta(after[0])
}

// nameTokens takes a model ID and returns its tokens split on every slash
// and hyphen, empty tokens dropped.
func nameTokens(modelID string) []string {
	return strings.FieldsFunc(modelID, isTokenSeparator)
}

// isTokenSeparator takes a rune and reports whether it separates the tokens
// of a model ID: a slash or a hyphen.
func isTokenSeparator(r rune) bool {
	return string(r) == catalog.KeySeparator || string(r) == "-"
}

// exampleTerm takes a token and reports whether a pattern example may draw
// it: at least the minimum length and not digits and dots alone.
func exampleTerm(token string) bool {
	return len([]rune(token)) >= minTermLength && strings.Trim(token, numericTokenChars) != ""
}

// opensAnyID takes models and a term and reports whether any model's ID
// begins with the term.
func opensAnyID(models []catalog.Model, term string) bool {
	for i := range models {
		if strings.HasPrefix(models[i].ID, term) {
			return true
		}
	}

	return false
}

// pageFooter carries the data the footer's template text names.
//   - Examples: the usage examples drawn from the page's models
//   - AnyPartArgument: the any-part term as a shell argument: bare, or single-quoted when it carries escapes
//   - KeysSentence: the examples sentence naming the example keys, wrapped
//   - MatchSentence: the sentence naming what the keyword examples match, wrapped
//   - VendorExamples: per selected medium on the summary, the media filter and the top vendor
type pageFooter struct {
	Examples        usageExamples
	AnyPartArgument string
	KeysSentence    []string
	MatchSentence   []string
	VendorExamples  []vendorExample
}

// footerData takes a provider ID and the models a page shows, and returns
// the footer's data: the examples, the sentence naming the example keys, and
// the sentence naming what the keyword examples match, each wrapped within
// the page width.
func footerData(providerID string, shown []catalog.Model) pageFooter {
	examples := drawUsageExamples(providerID, shown)
	footer := pageFooter{Examples: examples, AnyPartArgument: shellArgument(examples.AnyPart)}

	if len(shown) == 0 {
		return footer
	}

	footer.KeysSentence = wrapWords(fmt.Sprintf(ExamplesSentence, strings.Join(examples.Keys, itemJoiner)), infoContentWidth)

	sentence := fmt.Sprintf(KeywordMatchExample, examples.Keyword, examples.KeywordMatch)
	if examples.AliasKeyword != "" {
		sentence += itemJoiner + fmt.Sprintf(AliasMatchExample, examples.AliasKeyword, examples.AliasMatch)
	}

	footer.MatchSentence = wrapWords(sentence+".", infoContentWidth)

	return footer
}

// shellArgument takes a pattern and returns it as the argument a shell passes
// through unchanged: bare when it holds no escape, and otherwise within
// single quotes.
func shellArgument(pattern string) string {
	if !strings.Contains(pattern, patternEscape) {
		return pattern
	}

	return singleQuote + pattern + singleQuote
}
