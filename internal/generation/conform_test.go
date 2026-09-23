package generation

// Invariants tested:
//  1. Absent input media: Given no reference images, media.Resize must return nil images without an
//     error, and ConformInputMedia must return no change records without an error.
//  2. Reference fitting change records: Given no requested size, ConformInputMedia must choose
//     1280x720 for the landscape reference or 720x1280 for the portrait reference.

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestFitRefsPass verifies invariant #1: Absent input media.
//
// What is being tested:
// Given no reference images, media.Resize must return nil images without an error, and
// ConformInputMedia must return no change records without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFitRefsPass(t *testing.T) {
	imgs, err := resizeTestInputs(t, "1280x720", nil)
	if err != nil || imgs != nil {
		t.Errorf("✗ resizeInputMedias(no images) = (%v,%v), want a clean passthrough", imgs, err)
	}

	gp := params.Values{params.FlagTypeSize: "1280x720"}

	model := conformVideoModel(t)
	if records, cErr := ConformInputMedia(&model, gp, nil); cErr != nil || len(records) != 0 {
		t.Errorf("✗ ConformInputMedias(no images) = (%v,%v), want no records and no error", records, cErr)
	}

	if !t.Failed() {
		t.Log("✓ a no-image run passes through the fit and the decision untouched")
	}
}

// TestFitRefsRecords verifies invariant #2: Reference fitting change records.
//
// What is being tested:
// Given no requested size, ConformInputMedia must choose 1280x720 for the landscape reference or
// 720x1280 for the portrait reference. The landscape case must produce a Derived size record and a
// Conformed reference record, and resizing must produce a PNG of that size. Given an explicit
// 1280x720 size, ConformInputMedia must retain it and return only the Conformed record.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFitRefsRecords(t *testing.T) {
	t.Run("adopted size", func(t *testing.T) {
		checkAdopted(t)

		if !t.Failed() {
			t.Log("✓ adopted size")
		}
	})
	t.Run("requested size", func(t *testing.T) {
		gp := params.Values{params.FlagTypeSize: "1280x720"}
		model := conformVideoModel(t)

		paramChanges, err := ConformInputMedia(&model, gp, []media.Input{pngRef(t, 1600, 1000)})
		if err != nil {
			t.Fatalf("💣 the resize decision failed: %v", err)
		}

		if gp[params.FlagTypeSize] != "1280x720" {
			t.Errorf("✗ resolved size = %q, want the requested 1280x720", gp[params.FlagTypeSize])
		}

		checkConf(t, paramChanges, "1600x1000", "1280x720", "restricted-size-video")

		if len(paramChanges) != 1 {
			t.Errorf("✗ records = %+v, want exactly one Conformed and no Derived for a requested size", paramChanges)
		}

		if !t.Failed() {
			t.Log("✓ requested size")
		}
	})
	t.Run("portrait first image adopts the portrait tier", func(t *testing.T) {
		gp := params.Values{}

		model := conformVideoModel(t)
		if _, err := ConformInputMedia(&model, gp, []media.Input{pngRef(t, 1000, 1600)}); err != nil {
			t.Fatalf("💣 the resize decision failed: %v", err)
		}

		if gp[params.FlagTypeSize] != "720x1280" {
			t.Errorf("✗ adopted size = %q, want the portrait base tier 720x1280", gp[params.FlagTypeSize])
		}

		if !t.Failed() {
			t.Log("✓ portrait first image adopts the portrait tier")
		}
	})

	if !t.Failed() {
		t.Log("✓ adoption picks the base tier by orientation and both effects return as Conformed/Derived records")
	}
}

// pngRef creates a decodable in-memory PNG reference image of the given dimensions.
func pngRef(t *testing.T, w, h int) media.Input {
	t.Helper()

	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("💣 encode fixture PNG: %v", err)
	}

	return media.Input{Bytes: b.Bytes(), MIME: "image/png", Filepath: fmt.Sprintf("/in/ref-%dx%d.png", w, h)}
}

// conformVideoModel returns a video model with landscape and portrait sizes and one allowed
// reference.
func conformVideoModel(t *testing.T) catalog.Model {
	t.Helper()

	return catalog.Model{
		ID: "restricted-size-video", Media: media.Video,
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution, ParamID: "resolution"},
			{FlagID: params.FlagTypeDuration, ParamID: "seconds", AllowedValues: []string{"4", "8", "12", "16", "20"}},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 1},
			{FlagID: params.FlagTypeSize, ParamID: "size", AllowedValues: []string{"1280x720", "720x1280"}},
		},
	}
}

// fitDims checks the fitted image's PNG MIME type, decoded format, and exact dimensions.
func fitDims(t *testing.T, tag string, img media.Input, w, h int) {
	t.Helper()

	if img.MIME != "image/png" {
		t.Errorf("✗ %s: fitted MIME = %q, want image/png", tag, img.MIME)
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(img.Bytes))
	if err != nil {
		t.Errorf("✗ %s: fitted image does not decode: %v", tag, err)

		return
	}

	if format != "png" || cfg.Width != w || cfg.Height != h {
		t.Errorf("✗ %s: fitted image = %s %dx%d, want png %dx%d", tag, format, cfg.Width, cfg.Height, w, h)
	}
}

// pickKind returns the first record of a kind, or nil.
func pickKind(test testing.TB, paramChanges []params.Adjustment, kind params.Change) *params.Adjustment {
	test.Helper()

	for i := range paramChanges {
		if paramChanges[i].Type == kind {
			return &paramChanges[i]
		}
	}

	return nil
}

// checkConf requires a Conformed input-media record with the expected source and target dimensions
// and a comment naming the model.
func checkConf(t *testing.T, paramChanges []params.Adjustment, requested, used, model string) {
	t.Helper()

	c := pickKind(t, paramChanges, params.ChangeConformed)
	if c == nil || c.FlagID != params.FlagTypeInputMedia || c.WireVal != used || c.InputVal != requested {
		t.Errorf("✗ Conformed record = %+v, want input-media %s → %s", c, requested, used)

		return
	}

	if !strings.Contains(c.Comment, model) {
		t.Errorf("✗ Conformed detail %q does not name the %s request the source was matched to", c.Comment, model)
	}
}

// checkAdopted checks landscape size adoption, the resized PNG, and the two adjustment records.
func checkAdopted(t *testing.T) {
	t.Helper()

	gp := params.Values{}
	inputs := []media.Input{pngRef(t, 1600, 1000)}
	model := conformVideoModel(t)

	paramChanges, err := ConformInputMedia(&model, gp, inputs)
	if err != nil {
		t.Fatalf("💣 the resize decision failed: %v", err)
	}

	if gp[params.FlagTypeSize] != "1280x720" {
		t.Errorf("✗ adopted size = %q, want the landscape base tier 1280x720", gp[params.FlagTypeSize])
	}

	imgs, err := resizeTestInputs(t, "1280x720", inputs)
	if err != nil {
		t.Fatalf("💣 resizeInputMedias failed: %v", err)
	}

	if len(imgs) == 1 {
		fitDims(t, "adopted", imgs[0], 1280, 720)
	} else {
		t.Errorf("✗ %d fitted images, want 1", len(imgs))
	}

	checkConf(t, paramChanges, "1600x1000", "1280x720", "restricted-size-video")

	if d := pickKind(t, paramChanges, params.ChangeDerived); d == nil || d.FlagID != params.FlagTypeSize || d.WireVal != "1280x720" {
		t.Errorf("✗ Derived record = %+v, want the adopted size 1280x720 under the size param", d)
	}

	if len(paramChanges) != 2 {
		t.Errorf("✗ %d records, want exactly the Conformed and Derived pair: %+v", len(paramChanges), paramChanges)
	}
}

// resizeTestInputs applies the production dimension parser before resizing the references.
func resizeTestInputs(test testing.TB, requestedSize string, inputs []media.Input) ([]media.Input, error) {
	test.Helper()

	width, height, _ := params.ParseDimensions(requestedSize)

	return media.Resize(width, height, inputs)
}
