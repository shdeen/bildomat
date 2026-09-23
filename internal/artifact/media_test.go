package artifact

import (
	"bytes"
	"errors"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// Invariants tested:
// 1. Media construction: New must preserve the supplied bytes in Data and leave TmpPath empty.

// TestNewArt verifies invariant #1: Media construction.
//
// What is being tested:
// New must preserve the supplied bytes in Data and leave TmpPath empty. It must choose the
// extension from a supported declared MIME type, detected content, or the fallback, in that order,
// and return ErrResponseNoData for empty data.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestNewArt(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		mime string
		fall string
		want string
	}{
		{"declared MIME wins over magic", jpegStub(t), "image/webp", pngExt, ".webp"},
		{"no MIME sniffs the bytes", jpegStub(t), "", pngExt, ".jpg"},
		{"unusable MIME sniffs the bytes", jpegStub(t), "application/json", pngExt, ".jpg"},
		{"unidentifiable falls to the declared fallback", []byte("plain text payload"), "", ".webp", ".webp"},
		{"svg+xml maps to .svg, never .svg+xml", svgStub(t), "image/svg+xml", pngExt, ".svg"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			generatedMedia, err := New(c.data, c.mime, c.fall)
			if err != nil {
				t.Errorf("✗ unexpected error: %v", err)

				return
			}

			if generatedMedia.FileExt != c.want {
				t.Errorf("✗ FileExt = %q, want %q", generatedMedia.FileExt, c.want)
			}

			if !bytes.Equal(generatedMedia.Data, c.data) || generatedMedia.TmpPath != "" {
				t.Errorf("✗ artifact not exclusively bytes-backed with the given data: %+v", generatedMedia)
			}

			checkCarriage(t, c.name, generatedMedia)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}
	// New must reject empty data with ErrResponseNoData.
	if _, err := New(nil, "image/png", pngExt); !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ New(empty) err = %v, want the no-data sentinel", err)
	}

	if !t.Failed() {
		t.Log("✓ New follows the truthful chain, carries bytes exclusively, and rejects empty data")
	}
}

// pngExt is the image extension used by the artifact fixtures.
const pngExt = ".png"

// jpegStub returns enough of a JPEG header for content-type detection.
func jpegStub(test testing.TB) []byte {
	test.Helper()

	return []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00}
}

// svgStub returns SVG content whose extension must come from its declared MIME type.
func svgStub(test testing.TB) []byte {
	test.Helper()

	return []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"/>`)
}

// checkCarriage requires exactly one data source and a nonempty extension.
func checkCarriage(t *testing.T, tag string, generatedMedia Media) {
	t.Helper()

	if (len(generatedMedia.Data) > 0) == (generatedMedia.TmpPath != "") {
		t.Errorf("✗ %s: artifact carriage not exclusive: %d inline bytes, SrcPath %q", tag, len(generatedMedia.Data), generatedMedia.TmpPath)
	}

	if generatedMedia.FileExt == "" {
		t.Errorf("✗ %s: artifact has no extension", tag)
	}
}
