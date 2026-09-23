package generation

// Invariants tested:
//  1. Reference fitting error category: Given malformed dimensions or undecodable image bytes,
//     media.Resize and ConformInputMedia must return an error that wraps errs.ErrInputMedia,
//     including when ConformInputMedia must infer the size.
//  2. Actual conformance across retained images: Given one matching image and one smaller image,
//     ConformInputMedia must report only the resize from 640x360 to 1280x720.

import (
	"errors"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestFitRefsErrors verifies invariant #1: Reference fitting error category.
//
// What is being tested:
// Given malformed dimensions or undecodable image bytes, media.Resize and ConformInputMedia must
// return an error that wraps errs.ErrInputMedia, including when ConformInputMedia must infer the
// size.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFitRefsErrors(t *testing.T) {
	model := conformVideoModel(t)

	if _, err := resizeTestInputs(t, "not-a-size", []media.Input{badRef(t)}); err == nil || !errors.Is(err, errs.ErrInputMedia) {
		t.Errorf("✗ resizeInputMedias(invalid size) = %v, want a wrapped ErrInputMedia", err)
	}
	// size unset → the decision decodes the first image's config for adoption; undecodable
	// bytes error.
	if _, err := ConformInputMedia(&model, params.Values{}, []media.Input{badRef(t)}); err == nil || !errors.Is(err, errs.ErrInputMedia) {
		t.Errorf("✗ ConformInputMedias(undecodable, size unset) = %v, want a wrapped ErrInputMedia", err)
	}
	// size set → the full image decodes for fitting; undecodable bytes error.
	if _, err := resizeTestInputs(t, "1280x720", []media.Input{badRef(t)}); err == nil || !errors.Is(err, errs.ErrInputMedia) {
		t.Errorf("✗ resizeInputMedias(undecodable, size set) = %v, want a wrapped ErrInputMedia", err)
	}

	if !t.Failed() {
		t.Log("✓ an invalid size and an undecodable image (both size modes) wrap in input-media errors")
	}
}

// TestConformanceActualImageChanges verifies invariant #2: Actual conformance across retained
// images.
//
// What is being tested:
// Given one matching image and one smaller image, ConformInputMedia must report only the resize
// from 640x360 to 1280x720. When no size was supplied, it must adopt the matching reference size
// and return one Derived record. If a later image cannot decode, it must return ErrInputMediaDecode
// with the earlier resize record.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestConformanceActualImageChanges(t *testing.T) {
	model := conformVideoModel(t)
	requested := params.Values{params.FlagTypeSize: "1280x720"}

	changes, err := ConformInputMedia(&model, requested, []media.Input{pngRef(t, 1280, 720), pngRef(t, 640, 360)})
	if err != nil {
		t.Errorf("✗ valid conformance failed: %v", err)
	}

	if len(changes) != 1 || changes[0].Type != params.ChangeConformed || changes[0].InputVal != "640x360" || changes[0].WireVal != "1280x720" {
		t.Errorf("✗ actual transformations reported incorrectly: %+v", changes)
	}

	adopted := params.Values{}

	changes, err = ConformInputMedia(&model, adopted, []media.Input{pngRef(t, 1280, 720)})
	if err != nil || adopted[params.FlagTypeSize] != "1280x720" || len(changes) != 1 || changes[0].Type != params.ChangeDerived {
		t.Errorf("✗ matching reference adoption: parameters=%v changes=%+v error=%v", adopted, changes, err)
	}

	changes, err = ConformInputMedia(&model, requested, []media.Input{pngRef(t, 640, 360), badRef(t)})
	if !errors.Is(err, errs.ErrInputMediaDecode) || len(changes) != 1 || changes[0].InputVal != "640x360" {
		t.Errorf("✗ later decode failure lost completed changes: %+v, %v", changes, err)
	}

	if !t.Failed() {
		t.Log("✓ every retained image contributes only actual conformance changes")
	}
}

// badRef returns an input whose bytes cannot decode as an image.
func badRef(test testing.TB) media.Input {
	test.Helper()

	return media.Input{Bytes: []byte("not an image"), MIME: "image/png", Filepath: "/in/bad.png"}
}
