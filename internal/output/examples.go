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

// The measures and separators of the examples. A key's tokens are split on the key separator and a
// name's tokens on the dash.
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

// usageExamples contains the model keys and search terms shown in a page footer.
//   - Keys: one fully qualified key per medium the models span, image first
//   - Keyword: a term some model's slash-delimited token begins with
//   - KeywordMatch: the bare ID of the model the keyword matches
//   - AliasKeyword: a term some alias begins with, empty when no model declares an alias
//   - AliasMatch: the alias the alias keyword matches
//   - Alternation: a pattern of two terms joined by a bar
//   - AnyPart: an escaped term selected from inside a model ID
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

// drawUsageExamples draws model and search examples from a provider's selected models. Empty input
// returns no examples.
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

// randomModel returns a randomly selected model from a nonempty slice.
func randomModel(models []catalog.Model) *catalog.Model {
	// #nosec G404 -- the examples need variety, not unpredictability.
	return &models[rand.IntN(len(models))]
}

// preferInnerTokens selects models whose IDs contain a slash, or returns all models if none do.
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

// exampleKeys returns a random fully qualified model key per medium, images before videos.
func exampleKeys(providerID string, models []catalog.Model) []string {
	var keys []string

	for _, media := range []media.Kind{media.Image, media.Video} {
		if mediaModels := modelsOfMedia(models, media); len(mediaModels) > 0 {
			keys = append(keys, providerID+catalog.KeySeparator+randomModel(mediaModels).ID)
		}
	}

	return keys
}

// keywordExample returns up to keywordLength runes from the last slash-delimited model ID token.
func keywordExample(model *catalog.Model) string {
	tokens := strings.Split(model.ID, catalog.KeySeparator)
	last := []rune(tokens[len(tokens)-1])

	if len(last) <= keywordLength {
		return string(last)
	}

	return string(last[:keywordLength])
}

// aliasExample selects a random alias and returns its first hyphen-delimited token followed by the
// full alias. It returns two empty strings when no aliases exist.
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

// alternationExample joins two random eligible model-name tokens with a regex alternation. With
// fewer than two tokens, it uses the provider ID and a random model ID. All terms are escaped.
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

// anyPartExample returns an escaped token from inside a model ID that prefixes no model ID. If no
// token qualifies, it removes the first rune of a random ID and escapes the remainder; an ID
// shorter than two runes produces no text.
func anyPartExample(models []catalog.Model) string {
	var inner []string

	for i := range models {
		tokens := nameTokens(models[i].ID)
		if len(tokens) < 2 {
			continue
		}

		for _, token := range tokens[1:] {
			if exampleTerm(token) && !opensAnyID(models, token) && !slices.Contains(inner, token) {
				inner = append(inner, token)
			}
		}
	}

	if len(inner) == 0 {
		id := []rune(randomModel(models).ID)
		if len(id) < 2 {
			return ""
		}

		return regexp.QuoteMeta(string(id[1:]))
	}

	// #nosec G404 -- the examples need variety, not unpredictability.
	return regexp.QuoteMeta(inner[rand.IntN(len(inner))])
}

// acrossTokensExample returns an escaped search pattern spanning the last slash in a model key. It
// joins the neighboring name tokens, or returns empty text if either token is absent.
func acrossTokensExample(providerID string, model *catalog.Model) string {
	tokens := strings.Split(providerID+catalog.KeySeparator+model.ID, catalog.KeySeparator)
	before := nameTokens(tokens[len(tokens)-2])
	after := nameTokens(tokens[len(tokens)-1])

	if len(before) == 0 || len(after) == 0 {
		return ""
	}

	return regexp.QuoteMeta(before[len(before)-1]) + catalog.KeySeparator + regexp.QuoteMeta(after[0])
}

// nameTokens splits a model ID at slashes and hyphens, dropping empty tokens.
func nameTokens(modelID string) []string {
	return strings.FieldsFunc(modelID, isTokenSeparator)
}

// isTokenSeparator reports whether a rune is a slash or hyphen.
func isTokenSeparator(r rune) bool {
	return string(r) == catalog.KeySeparator || string(r) == "-"
}

// exampleTerm accepts tokens of at least minTermLength runes that are not solely digits and dots.
func exampleTerm(token string) bool {
	return len([]rune(token)) >= minTermLength && strings.Trim(token, numericTokenChars) != ""
}

// opensAnyID reports whether any model ID starts with the supplied term.
func opensAnyID(models []catalog.Model, term string) bool {
	for i := range models {
		if strings.HasPrefix(models[i].ID, term) {
			return true
		}
	}

	return false
}

// pageFooter contains examples and explanatory sentences for the footer template.
//   - Examples: the usage examples drawn from the page's models
//   - AnyPartArgument: the any-part term as a shell argument: bare, or single-quoted when it
//     carries escapes
//   - KeysSentence: the examples sentence naming the example keys, wrapped
//   - MatchSentence: the sentence naming what the keyword examples match, wrapped
//   - VendorExamples: the media filter and highest-count vendor for each selected medium
type pageFooter struct {
	Examples        usageExamples
	AnyPartArgument string
	KeysSentence    []string
	MatchSentence   []string
	VendorExamples  []vendorExample
}

// footerData prepares model and search examples with explanatory text wrapped to the page width.
func footerData(providerID string, shown []catalog.Model) pageFooter {
	examples := drawUsageExamples(providerID, shown)
	footer := pageFooter{Examples: examples, AnyPartArgument: shellArgument(examples.AnyPart)}

	if len(shown) == 0 {
		return footer
	}

	footer.KeysSentence = wrapWords(fmt.Sprintf(ExamplesSentence, strings.Join(examples.Keys, itemJoiner)), pageWidth)

	if examples.Keyword == "" {
		return footer
	}

	sentence := fmt.Sprintf(KeywordMatchExample, examples.Keyword, examples.KeywordMatch)
	if examples.AliasKeyword != "" {
		sentence += itemJoiner + fmt.Sprintf(AliasMatchExample, examples.AliasKeyword, examples.AliasMatch)
	}

	footer.MatchSentence = wrapWords(sentence+".", pageWidth)

	return footer
}

// shellArgument encloses patterns containing backslashes in single quotes.
func shellArgument(pattern string) string {
	if !strings.Contains(pattern, patternEscape) {
		return pattern
	}

	return singleQuote + pattern + singleQuote
}
