package media_test

// Invariants tested:
//  1. Reference fitting geometry: Given PNG, JPEG, and WebP references in the listed orientations,
//     media.Resize must return one PNG at the requested dimensions.
//  2. Immutable source media: Given a 12x8 PNG, media.Resize must return a 6x4 PNG without changing
//     the original bytes.

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/shdeen/bildomat/internal/media"
)

// TestFitRefsGeometry verifies invariant #1: Reference fitting geometry.
//
// What is being tested:
// Given PNG, JPEG, and WebP references in the listed orientations, media.Resize must return one PNG
// at the requested dimensions. For gradient references, sampled output pixels must come from the
// centered source crop within winTol.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFitRefsGeometry(t *testing.T) {
	cases := []struct {
		tag  string
		src  media.Input
		w, h int
		grad bool // gradient fixture: the output must be the exact centered source window
	}{
		{"landscape into landscape", pngRef(t, 1600, 1000), 1280, 720, false},
		{"landscape into portrait", gradRef(t, 1600, 1000), 720, 1280, true},
		{"portrait into landscape", gradRef(t, 1000, 1600), 1280, 720, true},
		{"portrait into portrait tier", pngRef(t, 900, 1600), 1024, 1792, false},
		{"jpeg source", jpgRef(t, 1600, 1000), 1280, 720, false},
		{"webp source", webpRef(t), 1280, 720, false},
	}
	for _, c := range cases {
		imgs, err := media.Resize(c.w, c.h, []media.Input{c.src})
		if err != nil {
			t.Errorf("✗ %s: Resize failed: %v", c.tag, err)

			continue
		}

		if len(imgs) != 1 {
			t.Errorf("✗ %s: %d fitted images, want 1", c.tag, len(imgs))

			continue
		}

		fitDims(t, c.tag, imgs[0], c.w, c.h)

		if c.grad {
			srcCfg, _, err := image.DecodeConfig(bytes.NewReader(c.src.Bytes))
			if err != nil {
				t.Fatalf("💣 decode the gradient fixture: %v", err)
			}

			checkWindow(t, c.tag, imgs[0], srcCfg.Width, srcCfg.Height, c.w, c.h)
		}
	}

	if !t.Failed() {
		t.Log("✓ references center-crop to the exact requested frame as PNG across orientations and source formats")
	}
}

// TestResizePreservesSources verifies invariant #2: Immutable source media.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Given a 12x8 PNG, media.Resize must return a 6x4 PNG without changing the original bytes. It must
// preserve the source path, first-frame anchor, and each tested timestamp state: absent, zero, or
// 1.25 seconds.
// Kind: permanent.
func TestResizePreservesSources(t *testing.T) {
	for _, timestamp := range []*float64{nil, new(0.0), new(1.25)} {
		source := pngRef(t, 12, 8)
		source.Time = timestamp
		source.FrameAnchor = media.FrameFirst
		originalBytes := bytes.Clone(source.Bytes)

		resized, err := media.Resize(6, 4, []media.Input{source})
		if err != nil || len(resized) != 1 {
			t.Errorf("✗ resize = %d results, %v", len(resized), err)

			continue
		}

		if !bytes.Equal(source.Bytes, originalBytes) {
			t.Error("✗ resizing changed the source buffer")
		}

		if resized[0].Filepath != source.Filepath || resized[0].FrameAnchor != source.FrameAnchor {
			t.Errorf("✗ source identity or anchor changed: %+v", resized[0])
		}

		sourceTime, sourceHasTime := source.FrameTime()

		resizedTime, resizedHasTime := resized[0].FrameTime()
		if sourceTime != resizedTime || sourceHasTime != resizedHasTime {
			t.Errorf("✗ timestamp changed: source=(%v,%t), resized=(%v,%t)", sourceTime, sourceHasTime, resizedTime, resizedHasTime)
		}

		fitDims(t, "resized", resized[0], 6, 4)
	}

	if !t.Failed() {
		t.Log("✓ source bytes and frame metadata remain intact")
	}
}

// winTol allows for resampling and color quantization when measuring source coordinates.
const winTol = 16.0

// pngRef creates a decodable in-memory PNG reference image of the given dimensions.
func pngRef(t *testing.T, w, h int) media.Input {
	t.Helper()

	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("💣 encode fixture PNG: %v", err)
	}

	return media.Input{Bytes: b.Bytes(), MIME: "image/png", Filepath: fmt.Sprintf("/in/ref-%dx%d.png", w, h)}
}

// jpgRef creates a decodable in-memory JPEG reference image.
func jpgRef(t *testing.T, w, h int) media.Input {
	t.Helper()

	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatalf("💣 encode fixture JPEG: %v", err)
	}

	return media.Input{Bytes: b.Bytes(), MIME: "image/jpeg", Filepath: "/in/ref.jpg"}
}

// webpRef loads the committed 1600x1000 WebP fixture.
func webpRef(t *testing.T) media.Input {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "ref-landscape.webp"))
	if err != nil {
		t.Fatalf("💣 read the WebP fixture: %v", err)
	}

	return media.Input{Bytes: data, MIME: "image/webp", Filepath: "/in/ref-landscape.webp"}
}

// gradRef creates a PNG with coordinate gradients for measuring crop placement. Red and blue scale
// with the source x and y coordinates, respectively.
func gradRef(t *testing.T, w, h int) media.Input {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / (w - 1)), //nolint:gosec // bounded 0..255 by construction
				G: 128,
				B: uint8(y * 255 / (h - 1)), //nolint:gosec // bounded 0..255 by construction
				A: 255,
			})
		}
	}

	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatalf("💣 encode gradient PNG: %v", err)
	}

	return media.Input{Bytes: b.Bytes(), MIME: "image/png", Filepath: fmt.Sprintf("/in/grad-%dx%d.png", w, h)}
}

// checkWindow compares sampled output coordinates with a centered crop, allowing winTol source
// pixels.
func checkWindow(t *testing.T, tag string, img media.Input, sw, sh, tw, th int) {
	t.Helper()

	src, _, err := image.Decode(bytes.NewReader(img.Bytes))
	if err != nil {
		t.Errorf("✗ %s: fitted image does not decode: %v", tag, err)

		return
	}

	cropW, cropH := float64(sw), float64(sh)
	x0, y0 := 0.0, 0.0

	if want := float64(tw) / float64(th); float64(sw)/float64(sh) > want {
		cropW = float64(sh) * want
		x0 = (float64(sw) - cropW) / 2
	} else {
		cropH = float64(sw) / want
		y0 = (float64(sh) - cropH) / 2
	}

	bounds := src.Bounds()
	for _, p := range [][2]int{{6, 6}, {tw / 2, th / 2}, {tw - 7, 6}, {6, th - 7}, {tw - 7, th - 7}} {
		px := src.At(bounds.Min.X+p[0], bounds.Min.Y+p[1])
		if px == nil {
			t.Errorf("✗ %s: pixel (%d,%d) has no color", tag, p[0], p[1])

			continue
		}

		r, _, b, _ := px.RGBA()
		gotX := float64(r>>8) / 255 * float64(sw-1)
		gotY := float64(b>>8) / 255 * float64(sh-1)
		wantX := x0 + (float64(p[0])+0.5)*cropW/float64(tw)

		wantY := y0 + (float64(p[1])+0.5)*cropH/float64(th)
		if math.Abs(gotX-wantX) > winTol || math.Abs(gotY-wantY) > winTol {
			t.Errorf("✗ %s: output (%d,%d) came from source (%.0f,%.0f), want the centered window position (%.0f,%.0f) ±%.0f",
				tag, p[0], p[1], gotX, gotY, wantX, wantY, winTol)
		}
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
