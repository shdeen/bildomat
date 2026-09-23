package params

// Invariants tested:
//  1. Free-form size clamping: Given 4000x4000 and the free-form bounds fixture, clampFree must
//     return edges no larger than 3840 and divisible by 16, with no more than 8294400 pixels.
//  2. Derived free-form size: Given ratio 16:9 and the free-form bounds fixture, deriveSize must
//     return parseable dimensions with edges divisible by 16 and no larger than 3840.
//  3. Resolution height classification: Given the listed resolution labels, repHeight must return
//     their expected pixel heights and nearestRes must map each height to a member of resSet.
//  4. Nearest aspect membership: Given 7:5 and each fixture aspect set, aspectNearest must return a
//     set member.
//  5. Free-form size clamping branches: With the free-form bounds fixture, clampFree must reduce
//     5000x4000 within MaxEdge, enlarge 100x100 to at least MinPx, and preserve 1024x1024 exactly.
//  6. Free-form size constraints: With the free-form bounds fixture, deriveSize must return
//     parseable portrait dimensions for 9:16 and dimensions with width-to-height ratio at most 3.05
//     for 10:1.
//  7. Minimum edge enforcement: Given 1000x100, minimum edge 256, and increment eight, clampFree
//     must return both edges at least 256 and divisible by eight.
//  8. Numerical resolution selection: Given 1280x720 or 720x1280, repHeight must return 720.

import (
	"slices"
	"testing"
)

// TestClampSize verifies invariant #1: Free-form size clamping.
//
// What is being tested:
// Given 4000x4000 and the free-form bounds fixture, clampFree must return edges no larger than 3840
// and divisible by 16, with no more than 8294400 pixels.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestClampSize(t *testing.T) {
	b := freeFormBounds(t)

	w, h := clampFree(b, 4000, 4000)
	if w > 3840 || h > 3840 {
		t.Errorf("✗ edge exceeds 3840: %dx%d", w, h)
	}

	if w*h > 8294400 {
		t.Errorf("✗ pixels %d exceed 8294400", w*h)
	}

	if w%16 != 0 || h%16 != 0 {
		t.Errorf("✗ edges not multiples of 16: %dx%d", w, h)
	}

	if !t.Failed() {
		t.Logf("✓ 4000x4000 constrained → %dx%d", w, h)
	}
}

// TestDeriveSize verifies invariant #2: Derived free-form size.
//
// What is being tested:
// Given ratio 16:9 and the free-form bounds fixture, deriveSize must return parseable dimensions
// with edges divisible by 16 and no larger than 3840. The area must be between 655360 and 8294400
// pixels, and the ratio must remain within six percent of 16:9.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDeriveSize(t *testing.T) {
	b := freeFormBounds(t)
	size := deriveSize(b, 16.0/9.0)

	w, h, ok := ParseDimensions(size)
	if !ok {
		t.Fatalf("💣 not a WxH size: %q", size)
	}

	if w%16 != 0 || h%16 != 0 {
		t.Errorf("✗ edges not multiples of 16: %dx%d", w, h)
	}

	r := float64(w) / float64(h)

	want := 16.0 / 9.0
	if r < want*0.94 || r > want*1.06 {
		t.Errorf("✗ ratio %.3f not ~16:9 (within ×16 rounding)", r)
	}

	if w > 3840 || h > 3840 || w*h < 655360 || w*h > 8294400 {
		t.Errorf("✗ outside the free-form bounds fixture: %dx%d (%d px)", w, h, w*h)
	}

	if !t.Failed() {
		t.Logf("✓ 16:9 → %s within limits", size)
	}
}

// TestResHeightClassify verifies invariant #3: Resolution height classification.
//
// What is being tested:
// Given the listed resolution labels, repHeight must return their expected pixel heights and
// nearestRes must map each height to a member of resSet. repHeight must reject every malformed
// label in the test.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResHeightClassify(t *testing.T) {
	parseable := map[string]int{
		"720p": 720, "1080p": 1080, "4k": 2160, "8k": 4320, "2k": 1080,
		"1024x768": 768, "1920x1080": 1080, "1080": 1080,
	}
	for in, want := range parseable {
		h, ok := repHeight(in)
		if !ok || h != want {
			t.Errorf("✗ repHeight(%q) = (%d,%v), want (%d,true)", in, h, ok, want)
		}

		if nearest := nearestRes(h, resSet); !slices.Contains(resSet, nearest) {
			t.Errorf("✗ nearestRes(%d, %v) = %q, not a member", h, resSet, nearest)
		}
	}

	for _, bad := range []string{"garbage", "", "px", "p", "1x2x3", "k", "abc4k"} {
		if h, ok := repHeight(bad); ok {
			t.Errorf("✗ repHeight(%q) = (%d,true), want unparseable", bad, h)
		}
	}

	if !t.Failed() {
		t.Log("✓ repHeight classifies parseable vs malformed; a parseable height maps to a set member")
	}
}

// TestAspectNearestInSet verifies invariant #4: Nearest aspect membership.
//
// What is being tested:
// Given 7:5 and each fixture aspect set, aspectNearest must return a set member. Given the first
// member of each set, it must return that exact value with the membership flag true.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAspectNearestInSet(t *testing.T) {
	sets := map[string][]string{
		"two-aspect-model": {"16:9", "9:16"},
		"image-model":      {"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20"},
		"video-model":      {"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"},
	}
	for name, set := range sets {
		got, _ := aspectNearest("7:5", set) // parseable, out of every set
		if !slices.Contains(set, got) {
			t.Errorf("✗ aspectNearest(7:5, %s) = %q, not a member of the documented set", name, got)
		}

		if g, member := aspectNearest(set[0], set); g != set[0] || !member {
			t.Errorf("✗ aspectNearest(%q in-set, %s) = (%q,%v), want (%q,true)", set[0], name, g, member, set[0])
		}
	}

	if !t.Failed() {
		t.Log("✓ aspectNearest returns a member of each served model's set; in-set values pass through")
	}
}

// TestFreeSizeClampBranches verifies invariant #5: Free-form size clamping branches.
//
// What is being tested:
// With the free-form bounds fixture, clampFree must reduce 5000x4000 within MaxEdge, enlarge
// 100x100 to at least MinPx, and preserve 1024x1024 exactly.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFreeSizeClampBranches(t *testing.T) {
	b := freeFormBounds(t)

	w, h := clampFree(b, 5000, 4000)
	if w > b.MaxEdge || h > b.MaxEdge {
		t.Errorf("✗ clampFree(5000,4000) = %dx%d, want within max edge", w, h)
	}

	if sw, sh := clampFree(b, 100, 100); sw*sh < b.MinPx {
		t.Errorf("✗ clampFree(100,100) = %dx%d (%d px), want an upscaled size ≥ minPx", sw, sh, sw*sh)
	}

	if pw, ph := clampFree(b, 1024, 1024); pw != 1024 || ph != 1024 {
		t.Errorf("✗ clampFree(1024,1024) = %dx%d, want passthrough", pw, ph)
	}

	if !t.Failed() {
		t.Log("✓ clampFree constrains over-large WxH, upscales tiny WxH to the pixel floor, and passes a within-limits size through")
	}
}

// TestFreeFormSizeEdges verifies invariant #6: Free-form size constraints.
//
// What is being tested:
// With the free-form bounds fixture, deriveSize must return parseable portrait dimensions for 9:16
// and dimensions with width-to-height ratio at most 3.05 for 10:1.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFreeFormSizeEdges(t *testing.T) {
	b := freeFormBounds(t)
	ps := deriveSize(b, 9.0/16.0)

	pw, ph, ok := ParseDimensions(ps)
	if !ok {
		t.Fatalf("💣 not a WxH size: %q", ps)
	}

	if ph <= pw {
		t.Errorf("✗ 9:16 should be portrait, got %dx%d", pw, ph)
	}

	es := deriveSize(b, 10)

	ew, eh, ok := ParseDimensions(es)
	if !ok {
		t.Fatalf("💣 not a WxH size: %q", es)
	}

	if float64(ew)/float64(eh) > 3.0+0.05 {
		t.Errorf("✗ 10:1 not capped near 3:1, got %dx%d", ew, eh)
	}

	if !t.Failed() {
		t.Log("✓ derivation keeps portrait orientation and the 3:1 ratio cap")
	}
}

// TestBoundsMinEdge verifies invariant #7: Minimum edge enforcement.
//
// What is being tested:
// Given 1000x100, minimum edge 256, and increment eight, clampFree must return both edges at least
// 256 and divisible by eight.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBoundsMinEdge(t *testing.T) {
	b := SizeBounds{MinEdge: 256, EdgeIncrem: 8}

	w, h := clampFree(b, 1000, 100)
	if min(w, h) < 256 {
		t.Errorf("✗ clampFree(MinEdge 256) = %dx%d, short edge below the floor", w, h)
	}

	if w%8 != 0 || h%8 != 0 {
		t.Errorf("✗ clampFree(EdgeIncrem 8) = %dx%d, edges not multiples of 8", w, h)
	}

	if !t.Failed() {
		t.Log("✓ the per-edge floor repairs a below-floor edge under a non-16 step")
	}
}

// TestResolutionNumericalSelection verifies invariant #8: Numerical resolution selection.
// Test class: Expanded.
// Test layer: Coverage.
// Test kind: permanent.
// What is being tested:
// Given 1280x720 or 720x1280, repHeight must return 720. nearestRes must choose 720p over an
// uninterpretable hd label and return an empty string when every label lacks a numeric meaning.
func TestResolutionNumericalSelection(t *testing.T) {
	for _, dimensions := range []string{"1280x720", "720x1280"} {
		height, ok := repHeight(dimensions)
		if !ok || height != 720 {
			t.Errorf("✗ %s represents %d, want 720", dimensions, height)
		}
	}

	if label := nearestRes(100, []string{"hd", "720p"}); label != "720p" {
		t.Errorf("✗ selected %q, want 720p", label)
	}

	if label := nearestRes(100, []string{"hd", "fhd"}); label != "" {
		t.Errorf("✗ invented a numerical meaning for %q", label)
	}

	if !t.Failed() {
		t.Log("✓ numerical selection uses only known short edges")
	}
}

// resSet is a representative resolution set for nearest-by-height assertions.
//
//nolint:gochecknoglobals // a read-only fixture, written only at package load.
var resSet = []string{"480p", "720p", "1080p"}

// freeFormBounds returns the free-form size bounds fixture: a ratio cap, an edge cap, a pixel
// range, an edge increment, and a long edge.
func freeFormBounds(test testing.TB) SizeBounds {
	test.Helper()

	return SizeBounds{MaxRatio: 3, MaxEdge: 3840, MinPx: 655360, MaxPx: 8294400, EdgeIncrem: 16, LongEdge: 1536}
}

// assertBounds checks the fixture's ratio cap, edge cap, pixel range, and 16-pixel alignment.
func assertBounds(t *testing.T, tag string, w, h int) {
	t.Helper()

	b := freeFormBounds(t)

	long, short := max(w, h), min(w, h)
	if w%16 != 0 || h%16 != 0 {
		t.Errorf("✗ %s → %dx%d: edges are not multiples of 16", tag, w, h)
	}

	if long > b.MaxEdge {
		t.Errorf("✗ %s → %dx%d: edge %d exceeds the %d max", tag, w, h, long, b.MaxEdge)
	}

	if float64(long)/float64(short) > b.MaxRatio+1e-9 {
		t.Errorf("✗ %s → %dx%d: ratio %.4f exceeds the %.1f cap", tag, w, h, float64(long)/float64(short), b.MaxRatio)
	}

	if px := w * h; px < b.MinPx || px > b.MaxPx {
		t.Errorf("✗ %s → %dx%d: %d pixels outside [%d, %d]", tag, w, h, px, b.MinPx, b.MaxPx)
	}
}
