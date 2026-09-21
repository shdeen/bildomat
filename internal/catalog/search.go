package catalog

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
)

// SearchResultLimit is the most matching models a search renders; a larger
// match set is reported as too many to display.
const SearchResultLimit = 100

// KeySeparator joins a provider ID and a model ID into the fully qualified
// provider/model key, and splits that key into the tokens a prefix term may
// open.
const KeySeparator = "/"

// ProvModelPair contains a provider and one of its models.
//   - Provider: the provider's identity; its models and request settings are
//     read from the catalog
//   - Model: the selected model configuration
type ProvModelPair struct {
	Provider Provider
	Model    Model
}

// Label returns the pair as diagnostics name it: the provider ID with the
// model ID in parentheses.
func (pair *ProvModelPair) Label() string {
	return fmt.Sprintf("%s (%s)", pair.Provider.ID, pair.Model.ID)
}

// searchMatcher holds one compiled search term: the regular expression under
// --regex, or otherwise the lowercased prefix term.
//   - pattern: the compiled expression, nil for a prefix search
//   - prefix: the lowercased term of a prefix search
type searchMatcher struct {
	pattern *regexp.Regexp
	prefix  string
}

// SelectMedia takes provider-model pairs and the media selections, and returns
// the pairs whose model generates a selected medium, in input order.
func SelectMedia(pairs []ProvModelPair, imageSelected, videoSelected bool) []ProvModelPair {
	var selected []ProvModelPair

	for i := range pairs {
		if pairs[i].Model.MediaSelected(imageSelected, videoSelected) {
			selected = append(selected, pairs[i])
		}
	}

	return selected
}

// SearchDirectory takes provider-model pairs, a search term, an exclusion
// term, and whether the terms are regular expressions, and returns the pairs
// the search term matches and the exclusion term does not, in input order.
// An empty string means that term is absent: an absent search term makes
// every pair a candidate, and an absent exclusion term excludes nothing.
// Without a regular expression, a term matches a model when any
// slash-delimited token of its fully qualified key or any alias begins with
// the term, case-insensitively; with one, when the expression matches the
// key, a token, or an alias. It returns the CLI search-pattern error naming
// the term that does not compile, before any pair is matched, so a broken
// exclusion pattern is reported whatever the search term matches. It returns
// the too-many-results error when a search term is given and the result,
// after the exclusion, exceeds SearchResultLimit; an exclusion term alone has
// no limit.
func SearchDirectory(pairs []ProvModelPair, searchTerm, excludeTerm string, useRegexp bool) ([]ProvModelPair, error) {
	termMatcher, err := newSearchMatcher(searchTerm, useRegexp)
	if err != nil {
		return nil, err
	}

	exclusionMatcher, err := newSearchMatcher(excludeTerm, useRegexp)
	if err != nil {
		return nil, err
	}

	var selected []ProvModelPair

	for i := range pairs {
		if !termMatcher.matchesPair(&pairs[i]) {
			continue
		}

		if excludeTerm != "" && exclusionMatcher.matchesPair(&pairs[i]) {
			continue
		}

		selected = append(selected, pairs[i])
	}

	if searchTerm != "" && len(selected) > SearchResultLimit {
		return nil, fmt.Errorf("%q, %w", searchTerm, errs.ErrSearchTooManyResults)
	}

	return selected, nil
}

// newSearchMatcher takes a search term and whether it is a regular expression,
// and returns the matcher for it, or the CLI search-pattern error when the
// expression does not compile.
func newSearchMatcher(term string, useRegexp bool) (searchMatcher, error) {
	if !useRegexp {
		return searchMatcher{prefix: strings.ToLower(term)}, nil
	}

	pattern, err := regexp.Compile(term)
	if err != nil {
		return searchMatcher{}, fmt.Errorf("%q, %w, %w", term, errs.ErrCLISearchPattern, err)
	}

	return searchMatcher{pattern: pattern}, nil
}

// matchesPair reports
// whether the pair matches: a regular expression against the fully qualified
// key, each of its tokens, and each alias; a prefix term against each token
// and each alias.
func (matcher *searchMatcher) matchesPair(pair *ProvModelPair) bool {
	key := pair.Provider.ID + KeySeparator + pair.Model.ID
	if matcher.pattern != nil && matcher.pattern.MatchString(key) {
		return true
	}

	identifiers := append(strings.Split(key, KeySeparator), pair.Model.Aliases...)

	return slices.ContainsFunc(identifiers, matcher.matchesIdentifier)
}

// matchesIdentifier reports whether an identifier matches: wherever the regular expression matches, or, for a
// prefix term, when the lowercased item begins with it.
func (matcher *searchMatcher) matchesIdentifier(identifier string) bool {
	if matcher.pattern != nil {
		return matcher.pattern.MatchString(identifier)
	}

	return strings.HasPrefix(strings.ToLower(identifier), matcher.prefix)
}
