package catalog

// Invariants tested:
//  1. Search result limit: Given exactly searchResultLimit prefix matches, SearchDirectory must
//     return every match without an error.
//  2. Search term boundaries: Given the listed full-key, oversized, nonmatching Unicode,
//     control-character, and leading-space terms, SearchDirectory must return no matches and no
//     error.
//  3. Result limit after exclusion: Given more matches than searchResultLimit, SearchDirectory must
//     succeed when exclusion leaves exactly that limit and return ErrSearchTooManyResults when
//     exclusion leaves one extra match.
//  4. Broken exclusion pattern: Given the invalid regexp exclusion (, SearchDirectory must return
//     nil matches and ErrCLISearchPattern with the quoted invalid term in its text, whether the
//     search term matches models or not.
//  5. Catalog search under arbitrary input: For arbitrary search and exclusion terms,
//     SearchDirectory must return only ErrCLISearchPattern or ErrSearchTooManyResults on failure
//     and never exceed the result limit on success.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// TestSearchDirectoryLimit verifies invariant #1: Search result limit.
//
// What is being tested:
// Given exactly searchResultLimit prefix matches, SearchDirectory must return every match without
// an error. One extra prefix or regexp match must return ErrSearchTooManyResults, with the prefix
// error also matching ErrSearch.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSearchDirectoryLimit(t *testing.T) {
	// #nosec G101 -- the literal names the credential's environment variable, not a credential.
	many := Provider{ID: "many", DisplayName: "Many", APIKeyEnvVar: "MANY_API_KEY"}

	directory := func(modelCount int) []ProvModelPair {
		t.Helper()

		pairs := make([]ProvModelPair, 0, modelCount)
		for i := range modelCount {
			pairs = append(pairs, ProvModelPair{Provider: many, Model: Model{ID: fmt.Sprintf("m-%d", i), Name: "M", Media: media.Image}})
		}

		return pairs
	}

	matches, err := SearchDirectory(directory(searchResultLimit), "m-", "", false)
	if err != nil || len(matches) != searchResultLimit {
		t.Errorf("✗ a search at the limit returned %d matches, %v; want %d, nil", len(matches), err, searchResultLimit)
	}

	_, err = SearchDirectory(directory(searchResultLimit+1), "m-", "", false)
	if !errors.Is(err, errs.ErrSearchTooManyResults) || !errors.Is(err, errs.ErrSearch) {
		t.Errorf("✗ a search past the limit returned %v, want the too-many-results error", err)
	}

	_, err = SearchDirectory(directory(searchResultLimit+1), "^m", "", true)
	if !errors.Is(err, errs.ErrSearchTooManyResults) {
		t.Errorf("✗ a regexp search past the limit returned %v, want the too-many-results error", err)
	}

	if !t.Failed() {
		t.Log("✓ matches up to the limit render and one more is refused")
	}
}

// TestSearchDirectoryAdversarialTerms verifies invariant #2: Search term boundaries.
//
// What is being tested:
// Given the listed full-key, oversized, nonmatching Unicode, control-character, and leading-space
// terms, SearchDirectory must return no matches and no error. Searching an empty directory must
// also return no matches and no error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSearchDirectoryAdversarialTerms(t *testing.T) {
	pairs := searchFixture(t)

	for _, term := range []string{"alpine/dream-pixel", "/", "pixel-image-2-and-then-some", "ÄLPINE", "\x00", " alp"} {
		matches, err := SearchDirectory(pairs, term, "", false)
		if err != nil {
			t.Errorf("✗ the prefix term %q failed: %v", term, err)
		}

		if len(matches) != 0 {
			t.Errorf("✗ the prefix term %q matched %v, want none", term, matchKeys(t, matches))
		}
	}

	matches, err := SearchDirectory(nil, "", "", false)
	if err != nil || len(matches) != 0 {
		t.Errorf("✗ the empty directory returned %v, %v", matchKeys(t, matches), err)
	}

	if !t.Failed() {
		t.Log("✓ key-shaped, oversized, and unusual terms match nothing and never fail")
	}
}

// TestSearchDirectoryLimitAfterExclusion verifies invariant #3: Result limit after exclusion.
//
// What is being tested:
// Given more matches than searchResultLimit, SearchDirectory must succeed when exclusion leaves
// exactly that limit and return ErrSearchTooManyResults when exclusion leaves one extra match.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSearchDirectoryLimitAfterExclusion(t *testing.T) {
	// #nosec G101 -- the literal names the credential's environment variable, not a credential.
	many := Provider{ID: "many", DisplayName: "Many", APIKeyEnvVar: "MANY_API_KEY"}

	directory := func(remainingCount, excludedCount int) []ProvModelPair {
		t.Helper()

		pairs := make([]ProvModelPair, 0, remainingCount+excludedCount)
		for i := range remainingCount {
			pairs = append(pairs, ProvModelPair{Provider: many, Model: Model{ID: fmt.Sprintf("m-remaining-%d", i), Name: "M", Media: media.Image}})
		}

		for i := range excludedCount {
			pairs = append(pairs, ProvModelPair{Provider: many, Model: Model{ID: fmt.Sprintf("m-excluded-%d", i), Name: "M", Media: media.Image}})
		}

		return pairs
	}

	matches, err := SearchDirectory(directory(searchResultLimit, 5), "m-", "m-excluded", false)
	if err != nil || len(matches) != searchResultLimit {
		t.Errorf("✗ an exclusion leaving the limit returned %d matches, %v; want %d, nil", len(matches), err, searchResultLimit)
	}

	_, err = SearchDirectory(directory(searchResultLimit+1, 5), "m-", "m-excluded", false)
	if !errors.Is(err, errs.ErrSearchTooManyResults) {
		t.Errorf("✗ an exclusion leaving one over the limit returned %v, want the too-many-results error", err)
	}

	if !t.Failed() {
		t.Log("✓ the result limit applies to the result after the exclusion")
	}
}

// TestSearchDirectoryBrokenExcludePattern verifies invariant #4: Broken exclusion pattern.
//
// What is being tested:
// Given the invalid regexp exclusion (, SearchDirectory must return nil matches and
// ErrCLISearchPattern with the quoted invalid term in its text, whether the search term matches
// models or not.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSearchDirectoryBrokenExcludePattern(t *testing.T) {
	for _, searchTerm := range []string{"dream", "zzz-matches-no-model"} {
		matches, err := SearchDirectory(searchFixture(t), searchTerm, "(", true)
		if !errors.Is(err, errs.ErrCLISearchPattern) || matches != nil {
			t.Errorf("✗ the search for %q without the broken pattern returned %v, %v; want the search-pattern error and no matches", searchTerm, matches, err)

			continue
		}

		if !strings.Contains(err.Error(), `"("`) {
			t.Errorf("✗ the search-pattern error %q does not name the broken exclusion term", err.Error())
		}
	}

	if !t.Failed() {
		t.Log("✓ a broken exclusion pattern is the search-pattern error whatever the search term matches")
	}
}

// FuzzSearchDirectory verifies invariant #5: Catalog search under arbitrary input.
//
// What is being tested:
// For arbitrary search and exclusion terms, SearchDirectory must return only ErrCLISearchPattern or
// ErrSearchTooManyResults on failure and never exceed the result limit on success. In prefix mode,
// each returned model must match the search term and must not match a nonempty exclusion term
// across its tokens and aliases.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSearchDirectory(f *testing.F) {
	for _, seed := range []string{"", "alp", "ALP", "seed(ream|ance)", "^gardenia/", "(", "[", "\\", "*", "moon", "/", "pixel-image-2"} {
		f.Add(seed, "", false)
		f.Add(seed, "", true)
		f.Add("alp", seed, false)
		f.Add("^alp", seed, true)
		f.Add("", seed, false)
	}

	f.Fuzz(func(t *testing.T, term, excludeTerm string, useRegexp bool) {
		matches, err := SearchDirectory(searchFixture(t), term, excludeTerm, useRegexp)
		if err != nil {
			if !errors.Is(err, errs.ErrCLISearchPattern) && !errors.Is(err, errs.ErrSearchTooManyResults) {
				t.Errorf("✗ SearchDirectory(%q, %q, %t) failed outside its two failure classes: %v", term, excludeTerm, useRegexp, err)
			}

			if !t.Failed() {
				t.Logf("✓ %q without %q failed within the two failure classes", term, excludeTerm)
			}

			return
		}

		if len(matches) > searchResultLimit {
			t.Errorf("✗ SearchDirectory(%q, %q, %t) returned %d matches past the limit", term, excludeTerm, useRegexp, len(matches))
		}

		if useRegexp {
			if !t.Failed() {
				t.Logf("✓ the pattern %q without %q searched within the limit", term, excludeTerm)
			}

			return
		}

		for i := range matches {
			pair := &matches[i]
			items := append(strings.Split(pair.Provider.ID+"/"+pair.Model.ID, "/"), pair.Model.Aliases...)

			if !anyItemBegins(t, items, term) {
				t.Errorf("✗ the match %s/%s carries no token or alias beginning with %q", pair.Provider.ID, pair.Model.ID, term)
			}

			if excludeTerm != "" && anyItemBegins(t, items, excludeTerm) {
				t.Errorf("✗ the match %s/%s carries a token or alias beginning with the exclusion term %q", pair.Provider.ID, pair.Model.ID, excludeTerm)
			}
		}

		if !t.Failed() {
			t.Logf("✓ %q without %q searched within the limit, and every match fits both terms", term, excludeTerm)
		}
	})
}

// anyItemBegins takes a model's searchable items and a prefix term and reports whether any item
// begins with the term, case-insensitively, as a prefix search compares them.
func anyItemBegins(test testing.TB, items []string, term string) bool {
	test.Helper()

	loweredTerm := strings.ToLower(term)

	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), loweredTerm) {
			return true
		}
	}

	return false
}
