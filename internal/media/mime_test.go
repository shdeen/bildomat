package media

// Invariants tested:
//  1. MIME-to-extension mapping: Given the listed image and video MIME values, extForMime must
//     return their expected extensions, including .jpg for JPEG and .svg for SVG.

import (
	"testing"
)

// TestExtForMime verifies invariant #1: MIME-to-extension mapping.
//
// What is being tested:
// Given the listed image and video MIME values, extForMime must return their expected extensions,
// including .jpg for JPEG and .svg for SVG. ExtForMimeOr must return .png for image/png and the
// .bin fallback for missing subtypes, slashless input, or application/pdf.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestExtForMime(t *testing.T) {
	cases := map[string]string{
		"image/svg+xml": ".svg", "image/jpg": ".jpg", "text/html": ".", "IMAGE/JPEG; charset=utf-8": ".jpg",
		"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "video/mp4": ".mp4",
	}
	for mime, want := range cases {
		if got := extForMime(mime); got != want {
			t.Errorf("✗ extForMime(%q) = %q, want %q", mime, got, want)
		}
	}
	// No slash means no type/subtype structure, and an empty subtype means no token: both fall
	// back to the default. A slashless string is not a MIME type, so it yields no extension of
	// its own.
	for _, mime := range []string{"", "image/", "octet-stream"} {
		if got := ExtForMimeOr(mime, ".bin"); got != ".bin" {
			t.Errorf("✗ ExtForMimeOr(%q, .bin) = %q, want the default .bin", mime, got)
		}
	}

	if got := ExtForMimeOr("image/png", ".bin"); got != ".png" {
		t.Errorf("✗ ExtForMimeOr(image/png, .bin) = %q, want .png", got)
	}

	if got := ExtForMimeOr("application/pdf", ".bin"); got != ".bin" {
		t.Errorf("✗ ExtForMimeOr(application/pdf, .bin) = %q, want the nonmedia fallback", got)
	}

	if !t.Failed() {
		t.Log("✓ extForMime maps subtypes (jpeg→.jpg); ExtForMimeOr falls back to the default for an unusable MIME")
	}
}
