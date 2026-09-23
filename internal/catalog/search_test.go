package catalog

// Invariants tested:
//  1. Prefix search: Given the listed prefix terms, SearchDirectory must return exactly the
//     expected provider/model keys in fixture order, matching key tokens and aliases
//     case-insensitively.
//  2. Regular-expression search: Given the listed regular expressions, SearchDirectory must return
//     exactly the expected keys from full-key, token, and alias matches, respecting case unless the
//     expression changes it.
//  3. Media selection: Given each image/video selection, SelectMedia must return the expected count
//     with only selected media kinds.
//  4. Exclusion search: Given an exclusion term, SearchDirectory must return the fixture keys
//     outside the corresponding prefix or regexp matches, in their original order.
//  5. Search with both terms: Given both search and exclusion terms, SearchDirectory must return
//     the search matches with the exclusion matches removed, in their original order.
//  6. Empty search term: Given empty search and exclusion terms, SearchDirectory must return every
//     fixture key in its original order without an error.

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// TestSearchDirectoryPrefix verifies invariant #1: Prefix search.
//
// What is being tested:
// Given the listed prefix terms, SearchDirectory must return exactly the expected provider/model
// keys in fixture order, matching key tokens and aliases case-insensitively. A term with no match
// must return no keys and no error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSearchDirectoryPrefix(t *testing.T) {
	cases := []struct {
		term string
		want []string
	}{
		{"dream", []string{"alpine/dream-pixel", "alpaca/dreambyte/image"}},
		{"mini", []string{"minimax/hailuo"}},
		{"alp", []string{"alpine/pixel-image-2", "alpine/dream-pixel", "alpaca/alpine/pixel-image-2", "alpaca/bytedance/seedream", "alpaca/bytedance/seedance", "alpaca/dreambyte/image", "alpaca/gardenia/lumen"}},
		{"alpine", []string{"alpine/pixel-image-2", "alpine/dream-pixel", "alpaca/alpine/pixel-image-2"}},
		{"out", []string{"forest/brush-tools/outpainting-v1"}},
		{"PIXEL", []string{"alpine/pixel-image-2", "alpaca/alpine/pixel-image-2"}},
		{"moon", []string{"alpaca/gardenia/lumen"}},
		{"seedream", []string{"alpaca/bytedance/seedream"}},
		{"nothing-matches", nil},
	}
	for _, c := range cases {
		t.Run(c.term, func(t *testing.T) {
			matches, err := SearchDirectory(searchFixture(t), c.term, "", false)
			if err != nil {
				t.Fatalf("💣 SearchDirectory(%q) failed: %v", c.term, err)
			}

			if got := matchKeys(t, matches); !slices.Equal(got, c.want) {
				t.Errorf("✗ matches for %q = %v, want %v", c.term, got, c.want)
			}

			if !t.Failed() {
				t.Logf("✓ %q matches the expected models", c.term)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the prefix search matches the key tokens and aliases only")
	}
}

// TestSearchDirectoryRegexp verifies invariant #2: Regular-expression search.
//
// What is being tested:
// Given the listed regular expressions, SearchDirectory must return exactly the expected keys from
// full-key, token, and alias matches, respecting case unless the expression changes it. An invalid
// opening parenthesis must return ErrCLISearchPattern under ErrCLI.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSearchDirectoryRegexp(t *testing.T) {
	cases := []struct {
		pattern string
		want    []string
	}{
		{"seed(ream|ance)", []string{"alpaca/bytedance/seedream", "alpaca/bytedance/seedance"}},
		{"^gardenia/", nil},
		{"/gardenia/", []string{"alpaca/gardenia/lumen"}},
		{"^pixel", []string{"alpine/pixel-image-2", "alpaca/alpine/pixel-image-2"}},
		{"DREAM", nil},
		{"(?i)dream", []string{"alpine/dream-pixel", "alpaca/bytedance/seedream", "alpaca/dreambyte/image"}},
		{"melon$", []string{"alpaca/gardenia/lumen"}},
		{"^alpaca/alpine/", []string{"alpaca/alpine/pixel-image-2"}},
	}
	for _, c := range cases {
		t.Run(c.pattern, func(t *testing.T) {
			matches, err := SearchDirectory(searchFixture(t), c.pattern, "", true)
			if err != nil {
				t.Fatalf("💣 SearchDirectory(%q, regexp, false) failed: %v", c.pattern, err)
			}

			if got := matchKeys(t, matches); !slices.Equal(got, c.want) {
				t.Errorf("✗ matches for /%s/ = %v, want %v", c.pattern, got, c.want)
			}

			if !t.Failed() {
				t.Logf("✓ /%s/ matches the expected models", c.pattern)
			}
		})
	}

	_, err := SearchDirectory(searchFixture(t), "(", "", true)
	if !errors.Is(err, errs.ErrCLISearchPattern) || !errors.Is(err, errs.ErrCLI) {
		t.Errorf("✗ an uncompilable pattern returned %v, want the CLI search-pattern error", err)
	}

	if !t.Failed() {
		t.Log("✓ the regexp search matches the key, its tokens, and the aliases, and rejects a broken pattern as usage")
	}
}

// TestSelectMedia verifies invariant #3: Media selection.
//
// What is being tested:
// Given each image/video selection, SelectMedia must return the expected count with only selected
// media kinds. Selecting both must preserve every provider/model key in its original order.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSelectMedia(t *testing.T) {
	pairs := searchFixture(t)

	cases := []struct {
		image, video bool
		wantCount    int
		wantMedia    []media.Kind
	}{
		{true, true, len(pairs), []media.Kind{media.Image, media.Video}},
		{true, false, 7, []media.Kind{media.Image}},
		{false, true, 2, []media.Kind{media.Video}},
		{false, false, 0, nil},
	}
	for _, c := range cases {
		selected := SelectMedia(pairs, c.image, c.video)
		if len(selected) != c.wantCount {
			t.Errorf("✗ SelectMedia(image=%t, video=%t) kept %d pairs, want %d", c.image, c.video, len(selected), c.wantCount)
		}

		for i := range selected {
			if !slices.Contains(c.wantMedia, selected[i].Model.Media) {
				t.Errorf("✗ SelectMedia(image=%t, video=%t) kept a %s model", c.image, c.video, selected[i].Model.Media)
			}
		}
	}

	if !slices.Equal(matchKeys(t, SelectMedia(pairs, true, true)), matchKeys(t, pairs)) {
		t.Errorf("✗ SelectMedia reordered the directory")
	}

	if !t.Failed() {
		t.Log("✓ the media selection keeps the selected media in directory order")
	}
}

// TestSearchDirectoryExclude verifies invariant #4: Exclusion search.
//
// What is being tested:
// Given an exclusion term, SearchDirectory must return the fixture keys outside the corresponding
// prefix or regexp matches, in their original order. Excluding a term that matches nothing must
// also return a directory larger than searchResultLimit without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSearchDirectoryExclude(t *testing.T) {
	fixture := searchFixture(t)
	allKeys := matchKeys(t, fixture)

	for _, c := range []struct {
		term      string
		useRegexp bool
	}{
		{"dream", false},
		{"alp", false},
		{"PIXEL", false},
		{"nothing-matches", false},
		{"seed(ream|ance)", true},
		{"^alpine/", true},
	} {
		included, err := SearchDirectory(fixture, c.term, "", c.useRegexp)
		if err != nil {
			t.Fatalf("💣 the inclusion search for %q failed: %v", c.term, err)
		}

		excluded, err := SearchDirectory(fixture, "", c.term, c.useRegexp)
		if err != nil {
			t.Errorf("✗ the exclusion search for %q failed: %v", c.term, err)

			continue
		}

		includedKeys := matchKeys(t, included)
		expectedKeys := slices.DeleteFunc(slices.Clone(allKeys), func(key string) bool { return slices.Contains(includedKeys, key) })

		if got := matchKeys(t, excluded); !slices.Equal(got, expectedKeys) {
			t.Errorf("✗ exclusion of %q = %v, want the directory minus the matches %v", c.term, got, expectedKeys)
		}
	}

	// #nosec G101 -- the literal names the credential's environment variable, not a credential.
	many := Provider{ID: "many", DisplayName: "Many", APIKeyEnvVar: "MANY_API_KEY"}

	wide := make([]ProvModelPair, 0, searchResultLimit+1)
	for i := range searchResultLimit + 1 {
		wide = append(wide, ProvModelPair{Provider: many, Model: Model{ID: fmt.Sprintf("m-%d", i), Name: "M", Media: media.Image}})
	}

	if excluded, err := SearchDirectory(wide, "", "nothing-matches", false); err != nil || len(excluded) != searchResultLimit+1 {
		t.Errorf("✗ an exclusion past the limit returned %d, %v; want every model and no error", len(excluded), err)
	}

	if !t.Failed() {
		t.Log("✓ exclusion returns the directory minus the matches, without the result limit")
	}
}

// TestSearchDirectoryBothTerms verifies invariant #5: Search with both terms.
//
// What is being tested:
// Given both search and exclusion terms, SearchDirectory must return the search matches with the
// exclusion matches removed, in their original order. The listed combinations must each remove a
// match; the alp-without-pixel case must return exactly the five named fixture keys.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSearchDirectoryBothTerms(t *testing.T) {
	fixture := searchFixture(t)

	for _, c := range []struct {
		searchTerm, excludeTerm string
		useRegexp               bool
	}{
		{"alp", "pixel", false},
		{"ALPACA", "Byte", false},
		{"dream", "alpine", false},
		{"^alp", "seed(ream|ance)", true},
		{"image", "^alpaca/", true},
	} {
		included, err := SearchDirectory(fixture, c.searchTerm, "", c.useRegexp)
		if err != nil {
			t.Fatalf("💣 the search for %q alone failed: %v", c.searchTerm, err)
		}

		excludedAlone, err := SearchDirectory(fixture, c.excludeTerm, "", c.useRegexp)
		if err != nil {
			t.Fatalf("💣 the search for %q alone failed: %v", c.excludeTerm, err)
		}

		removedKeys := matchKeys(t, excludedAlone)
		expectedKeys := slices.DeleteFunc(matchKeys(t, included), func(key string) bool { return slices.Contains(removedKeys, key) })

		if len(expectedKeys) == len(included) {
			t.Fatalf("💣 excluding %q removes nothing from the matches of %q, so the row proves nothing", c.excludeTerm, c.searchTerm)
		}

		matches, err := SearchDirectory(fixture, c.searchTerm, c.excludeTerm, c.useRegexp)
		if err != nil {
			t.Errorf("✗ the search for %q without %q failed: %v", c.searchTerm, c.excludeTerm, err)

			continue
		}

		if matchedKeys := matchKeys(t, matches); !slices.Equal(matchedKeys, expectedKeys) {
			t.Errorf("✗ the search for %q without %q = %v, want %v", c.searchTerm, c.excludeTerm, matchedKeys, expectedKeys)
		}
	}

	matches, err := SearchDirectory(fixture, "alp", "pixel", false)
	writtenOutKeys := []string{"alpine/dream-pixel", "alpaca/bytedance/seedream", "alpaca/bytedance/seedance", "alpaca/dreambyte/image", "alpaca/gardenia/lumen"}

	if err != nil || !slices.Equal(matchKeys(t, matches), writtenOutKeys) {
		t.Errorf("✗ the search for alp without pixel = %v, %v; want %v", matchKeys(t, matches), err, writtenOutKeys)
	}

	if !t.Failed() {
		t.Log("✓ both terms return the search term's matches without the exclusion term's, in directory order")
	}
}

// TestSearchDirectoryEmptyTerm verifies invariant #6: Empty search term.
//
// What is being tested:
// Given empty search and exclusion terms, SearchDirectory must return every fixture key in its
// original order without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSearchDirectoryEmptyTerm(t *testing.T) {
	pairs := searchFixture(t)

	matches, err := SearchDirectory(pairs, "", "", false)
	if err != nil {
		t.Fatalf("💣 SearchDirectory(\"\") failed: %v", err)
	}

	if !slices.Equal(matchKeys(t, matches), matchKeys(t, pairs)) {
		t.Errorf("✗ the empty term matched %v, want the whole directory in order", matchKeys(t, matches))
	}

	if !t.Failed() {
		t.Log("✓ the empty term matches the whole directory in order")
	}
}

// searchFixture creates a four-provider directory carrying every identifier shape the matching
// rules distinguish: a bare model ID, an ID whose leading token is a vendor, an alias, and a
// provider whose ID is also a model token under another provider.
func searchFixture(test testing.TB) []ProvModelPair {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	alpine := Provider{ID: "alpine", DisplayName: "Alpine", APIKeyEnvVar: "ALPINE_API_KEY"}
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	alpaca := Provider{ID: "alpaca", DisplayName: "Alpaca", APIKeyEnvVar: "ALPACA_API_KEY", Aggregator: true}
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	minimax := Provider{ID: "minimax", DisplayName: "MiniMax", APIKeyEnvVar: "MINIMAX_API_KEY"}
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	forest := Provider{ID: "forest", DisplayName: "Forest Labs", APIKeyEnvVar: "FOREST_API_KEY"}

	return []ProvModelPair{
		{Provider: alpine, Model: Model{ID: "pixel-image-2", Name: "Pixel Image 2", Media: media.Image, Aliases: []string{"pixel"}}},
		{Provider: alpine, Model: Model{ID: "dream-pixel", Name: "Dream Pixel", Media: media.Image}},
		{Provider: alpaca, Model: Model{ID: "alpine/pixel-image-2", Name: "Pixel Image 2", Media: media.Image}},
		{Provider: alpaca, Model: Model{ID: "bytedance/seedream", Name: "Seedream", Media: media.Image}},
		{Provider: alpaca, Model: Model{ID: "bytedance/seedance", Name: "Seedance", Media: media.Video}},
		{Provider: alpaca, Model: Model{ID: "dreambyte/image", Name: "Dreambyte Image", Media: media.Image}},
		{Provider: alpaca, Model: Model{ID: "gardenia/lumen", Name: "Lumen", Media: media.Image, Aliases: []string{"moon-melon"}}},
		{Provider: minimax, Model: Model{ID: "hailuo", Name: "Hailuo", Media: media.Video}},
		{Provider: forest, Model: Model{ID: "brush-tools/outpainting-v1", Name: "Brush Outpainting", Media: media.Image}},
	}
}

// matchKeys takes matched pairs and returns their fully qualified keys in result order.
func matchKeys(test testing.TB, matches []ProvModelPair) []string {
	test.Helper()

	keys := make([]string, 0, len(matches))
	for i := range matches {
		keys = append(keys, matches[i].Provider.ID+"/"+matches[i].Model.ID)
	}

	return keys
}
