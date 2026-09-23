package media_test

// Invariants tested:
//  1. Invalid requested reference size: Given a decodable PNG and dimensions 0x0, media.Resize must
//     return an error that wraps errs.ErrInputMediaSize.
//  2. Truncated reference data: Given a PNG truncated to 64 bytes, media.Resize must return an
//     error that wraps errs.ErrInputMediaDecode and names image 1.
//  3. Multiple-reference fitting: Given three images, media.Resize must return three 1280x720 PNGs
//     in source-path order.

import (
	"bytes"
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// TestFitRefsBadSize verifies invariant #1: Invalid requested reference size.
//
// What is being tested:
// Given a decodable PNG and dimensions 0x0, media.Resize must return an error that wraps
// errs.ErrInputMediaSize.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFitRefsBadSize(t *testing.T) {
	_, err := media.Resize(0, 0, []media.Input{pngRef(t, 640, 400)})
	if err == nil || !errors.Is(err, errs.ErrInputMediaSize) {
		t.Errorf("✗ Resize(invalid size, decodable image) = %v, want a wrapped ErrInputMediaSize", err)
	}

	if !t.Failed() {
		t.Log("✓ nonpositive requested dimensions on a decodable image classifies as the size input error")
	}
}

// TestFitRefsTruncated verifies invariant #2: Truncated reference data.
//
// What is being tested:
// Given a PNG truncated to 64 bytes, media.Resize must return an error that wraps
// errs.ErrInputMediaDecode and names image 1.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFitRefsTruncated(t *testing.T) {
	whole := pngRef(t, 640, 400)
	cut := media.Input{Bytes: whole.Bytes[:64], MIME: "image/png", Filepath: "/in/cut.png"}

	_, err := media.Resize(1280, 720, []media.Input{cut})
	if err == nil || !errors.Is(err, errs.ErrInputMediaDecode) || !strings.Contains(err.Error(), "image 1") {
		t.Errorf("✗ Resize(truncated image) = %v, want ErrInputMediaDecode naming image 1", err)
	}

	if !t.Failed() {
		t.Log("✓ a truncated reference fails the fit as a decode error naming its position")
	}
}

// TestFitRefsMulti verifies invariant #3: Multiple-reference fitting.
//
// What is being tested:
// Given three images, media.Resize must return three 1280x720 PNGs in source-path order. If the
// second image cannot decode, it must return ErrInputMediaDecode and name image 2.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFitRefsMulti(t *testing.T) {
	in := []media.Input{pngRef(t, 1600, 1000), pngRef(t, 640, 400), pngRef(t, 1000, 1600)}

	out, err := media.Resize(1280, 720, in)
	if err != nil {
		t.Fatalf("💣 multi-image conformance failed: %v", err)
	}

	if len(out) != len(in) {
		t.Fatalf("💣 %d outputs, want %d — cannot compare positions", len(out), len(in))
	}

	for i, o := range out {
		if o.Filepath != in[i].Filepath {
			t.Errorf("✗ output %d path = %q, want the source order preserved (%q)", i+1, o.Filepath, in[i].Filepath)
		}

		cfg, _, derr := image.DecodeConfig(bytes.NewReader(o.Bytes))
		if derr != nil || cfg.Width != 1280 || cfg.Height != 720 {
			t.Errorf("✗ output %d = %dx%d (%v), want the fitted 1280x720", i+1, cfg.Width, cfg.Height, derr)
		}

		if o.MIME != "image/png" {
			t.Errorf("✗ output %d MIME = %q, want the re-encoded PNG", i+1, o.MIME)
		}
	}

	// A later image's decode error must name its position.
	_, err = media.Resize(1280, 720, []media.Input{pngRef(t, 640, 400), badRef(t), pngRef(t, 640, 400)})
	if err == nil || !errors.Is(err, errs.ErrInputMediaDecode) || !strings.Contains(err.Error(), "image 2") {
		t.Errorf("✗ later-image failure = %v, want the decode sentinel naming image 2", err)
	}

	if !t.Failed() {
		t.Log("✓ every reference conforms in source order; a later failure names its position")
	}
}

// badRef returns an input whose bytes cannot decode as an image.
func badRef(test testing.TB) media.Input {
	test.Helper()

	return media.Input{Bytes: []byte("not an image"), MIME: "image/png", Filepath: "/in/bad.png"}
}
