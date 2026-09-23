package media

// Invariants tested:
//  1. MIME extension selection under malformed input: Given image or video MIME strings with case
//     differences, parameters, whitespace, or invalid trailing characters, ExtForMimeOr must return
//     the expected normalized leading subtype extension.
//  2. MIME extension safety under arbitrary input: For arbitrary MIME strings and fallback .mp4,
//     ExtForMimeOr must return an extension starting with a period and containing no slash,
//     backslash, semicolon, Unicode space, or control character.

import (
	"strings"
	"testing"
	"unicode"
)

// TestExtForHostile verifies invariant #1: MIME extension selection under malformed input.
//
// What is being tested:
// Given image or video MIME strings with case differences, parameters, whitespace, or invalid
// trailing characters, ExtForMimeOr must return the expected normalized leading subtype extension.
// It must return .bin when the media type or leading subtype is unusable.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestExtForHostile(t *testing.T) {
	// Well-formed and parameterized values normalize to the subtype token.
	normalized := map[string]string{
		"image/png; charset=utf-8": ".png",
		"video/mp4; codecs=avc1":   ".mp4",
		"IMAGE/PNG":                ".png",
		"image/ png ":              ".png",
		"image/jpeg;":              ".jpg",
	}
	for mime, want := range normalized {
		if got := ExtForMimeOr(mime, ".bin"); got != want {
			t.Errorf("\u2717 ExtForMimeOr(%q, .bin) = %q, want the parsed %q", mime, got, want)
		}
	}
	// Junk after a valid leading subtype token is discarded — the token is taken, and whatever
	// follows the first invalid character (a second slash, a control byte, a space, a Unicode
	// control or space) is dropped.
	trimmed := map[string]string{
		"a/b/c":          ".bin",
		"image/a/b":      ".a",
		"image/png\x00":  ".png",
		"image/p ng":     ".p",
		`image/a\b`:      ".a",
		"image/a\u0085b": ".a",
		"image/a\u009fb": ".a",
		"image/a\u00a0b": ".a",
		"image/a\u2000b": ".a",
	}
	for mime, want := range trimmed {
		if got := ExtForMimeOr(mime, ".bin"); got != want {
			t.Errorf("\u2717 ExtForMimeOr(%q, .bin) = %q, want the leading token %q", mime, got, want)
		}
	}
	// No usable leading subtype token → the fallback.
	for _, mime := range []string{";", "image/;x=y", "image/ ", "noslash"} {
		if got := ExtForMimeOr(mime, ".bin"); got != ".bin" {
			t.Errorf("\u2717 ExtForMimeOr(%q, .bin) = %q, want the .bin fallback (no subtype token)", mime, got)
		}
	}

	if !t.Failed() {
		t.Log("\u2713 the subtype token is parsed from the front; parameters and trailing junk are discarded")
	}
}

// FuzzExtOr verifies invariant #2: MIME extension safety under arbitrary input.
//
// What is being tested:
// For arbitrary MIME strings and fallback .mp4, ExtForMimeOr must return an extension starting with
// a period and containing no slash, backslash, semicolon, Unicode space, or control character.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzExtOr(f *testing.F) {
	for _, s := range []string{
		"image/png", "image/jpeg", "image/", "", "a/b/c", "octet",
		"image/png; charset=utf-8", "image/p ng",
		"image/a\u0085b", "image/a\u00a0b",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, mime string) {
		got := ExtForMimeOr(mime, ".mp4")
		if !strings.HasPrefix(got, ".") {
			t.Errorf("✗ ExtForMimeOr(%q, .mp4) = %q, want an extension starting with a dot", mime, got)
		}

		if strings.ContainsAny(got, `/\;`) {
			t.Errorf("✗ ExtForMimeOr(%q, .mp4) = %q carries a filename-hostile character", mime, got)
		}

		for _, r := range got {
			if unicode.IsControl(r) || unicode.IsSpace(r) {
				t.Errorf("✗ ExtForMimeOr(%q, .mp4) = %q carries a control or space rune", mime, got)
			}
		}

		if !t.Failed() {
			t.Logf("✓ the extension stayed filename-safe")
		}
	})
}
