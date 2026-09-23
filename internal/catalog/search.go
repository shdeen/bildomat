package catalog

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
)

// searchResultLimit is the most matching models a search renders; a larger match set is reported as
// too many to display.
const searchResultLimit = 100

// KeySeparator joins provider and model IDs and separates their searchable tokens.
const KeySeparator = "/"

// ProvModelPair contains a provider and one of its models.
//   - Provider: the provider's identity; its models and request settings are read from the catalog
//   - Model: the selected model configuration
type ProvModelPair struct {
	Provider Provider
	Model    Model
}

// Label formats the provider and model identifiers for diagnostics.
func (pair *ProvModelPair) Label() string {
	return fmt.Sprintf("%s (%s)", pair.Provider.ID, pair.Model.ID)
}

// searchMatcher holds one compiled search term: the regular expression under --regex, or otherwise
// the lowercased prefix term.
//   - pattern: the compiled expression, nil for a prefix search
//   - prefix: the lowercased term of a prefix search
type searchMatcher struct {
	pattern *regexp.Regexp
	prefix  string
}

// SelectMedia takes provider-model pairs and the media selections, and returns the pairs whose
// model generates a selected medium, in input order.
func SelectMedia(pairs []ProvModelPair, imageSelected, videoSelected bool) []ProvModelPair {
	var selected []ProvModelPair

	for i := range pairs {
		if pairs[i].Model.mediaSelected(imageSelected, videoSelected) {
			selected = append(selected, pairs[i])
		}
	}

	return selected
}

// SearchDirectory selects matching pairs, applies exclusions, and preserves directory order. Prefix
// terms match key tokens or aliases without regard to case; regular expressions also match complete
// keys and return an error if invalid. Empty terms impose no filter, and searchResultLimit applies
// only when a nonempty search term is supplied.
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

	if searchTerm != "" && len(selected) > searchResultLimit {
		return nil, fmt.Errorf("%q, %w", searchTerm, errs.ErrSearchTooManyResults)
	}

	return selected, nil
}

// newSearchMatcher prepares a prefix or regular expression, classifying invalid expressions as
// search errors.
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

// matchesPair checks key tokens and aliases; regular expressions also match the complete key.
func (matcher *searchMatcher) matchesPair(pair *ProvModelPair) bool {
	key := pair.Provider.ID + KeySeparator + pair.Model.ID
	if matcher.pattern != nil && matcher.pattern.MatchString(key) {
		return true
	}

	identifiers := append(strings.Split(key, KeySeparator), pair.Model.Aliases...)

	return slices.ContainsFunc(identifiers, matcher.matchesIdentifier)
}

// matchesIdentifier applies the regular expression or case-insensitive prefix to an identifier.
func (matcher *searchMatcher) matchesIdentifier(identifier string) bool {
	if matcher.pattern != nil {
		return matcher.pattern.MatchString(identifier)
	}

	return strings.HasPrefix(strings.ToLower(identifier), matcher.prefix)
}
