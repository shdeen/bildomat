package params

// Invariants tested:
//  1. Finite aspect ratios at numeric boundaries: Given spaced, signed, zero-padded, or decimal
//     ratios, parseRatio must return the listed positive values within tolerance and reject
//     malformed separators or components.
//  2. Nearest aspect selection under extreme input: Given extreme ratios and a fixed aspect set,
//     aspectNearest must return a set member or an empty string.
//  3. Free-form sizing under extreme ratios: Given 4:1, 10:1, 100:1, and 1:10, deriveSize must
//     return parseable dimensions satisfying the fixture's ratio cap, edge cap, pixel range, and
//     16-pixel alignment.
//  4. Free-form bounds under extreme input: Given the listed pixel, edge, ratio, and increment
//     boundaries, clampFree must return the exact expected dimensions, including 816x816 below the
//     pixel floor and unchanged 3840x2160 at the ceiling.
//  5. Sizing with incomplete constraints: Given no LongEdge, deriveSize must return an empty
//     string.
//  6. Simultaneous dimension constraints: Given the listed extreme requests, including math.MaxInt
//     edges and conflicting adjustment pressures, clampFree must return positive dimensions
//     satisfying every declared edge, pixel, ratio, and increment bound.
//  7. Dimension feasibility: For generated bounded declarations and positive requests, clampFree
//     must agree with exhaustive feasibility: return positive dimensions satisfying every
//     constraint when a pair exists, or 0x0 when none exists.
//  8. Free-form size constraints: Given the listed extreme ratios and dimensions, deriveSize and
//     clampFree must satisfy the fixture's ratio cap, edge cap, pixel range, and 16-pixel alignment
//     simultaneously.
//  9. Width-height and aspect-ratio parsing: ParseDimensions must accept 1024x768 and reject the
//     listed malformed or nonpositive dimensions.
//  10. Discrete integer snapping: Given each listed allowed set and inputs from math.MinInt through
//      math.MaxInt, snapInt must return a set member.
//  11. Resolution label height mapping: Given the listed K, P, dimension, numeric, and
//      underscore-suffix labels, repHeight must return the exact expected heights.
//  12. Rounded ratio correction: Given 40x20, ratio cap 1.5, and increment 16, clampFree must
//      return exactly 32x32.
//  13. Aspect ratio parsing under arbitrary input: For arbitrary strings, any ratio that parseRatio
//      accepts must be positive and finite.
//  14. Width-height parsing under arbitrary input: For arbitrary strings, any dimensions that
//      ParseDimensions accepts must have positive width and height.
//  15. Discrete integer snapping under arbitrary input: For arbitrary integers and allowed values
//      4, 8, and 12, snapInt must return one of those three values.
//  16. Resolution height classification under arbitrary input: For arbitrary labels, any height
//      that repHeight accepts must be positive, and nearestRes must map it to a member of resSet.
//  17. Nearest aspect selection under arbitrary input: For arbitrary strings and the fixed aspect
//      set, aspectNearest must return an empty string or a declared member.
//  18. Free-form size clamping under arbitrary input: For arbitrary positive dimensions and each
//      satisfiable fixture of bounds, clampFree must return positive dimensions satisfying every
//      declared edge, pixel, ratio, and increment constraint.

import (
	"math"
	"slices"
	"testing"
)

// TestParseFiniteBoundary verifies invariant #1: Finite aspect ratios at numeric boundaries.
//
// What is being tested:
// Given spaced, signed, zero-padded, or decimal ratios, parseRatio must return the listed positive
// values within tolerance and reject malformed separators or components. ParseDimensions must
// accept the listed uppercase and spaced separators with positive edges and reject malformed,
// nonpositive, or overflowing dimensions.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestParseFiniteBoundary(t *testing.T) {
	okRatios := map[string]float64{
		"16:9 ":   16.0 / 9.0,
		" 3 : 2 ": 1.5,
		"+3:2":    1.5,
		"3:+2":    1.5,
		"03:02":   1.5,
		"1.5:1":   1.5,
	}
	for in, want := range okRatios {
		if r, ok := parseRatio(in); !ok || r <= 0 || math.Abs(r-want) > 1e-9 {
			t.Errorf("✗ parseRatio(%q) = (%v,%v), want (%v,true) positive", in, r, ok, want)
		}
	}

	for _, bad := range []string{"16:9:3", "16/9", ":", "::", "16:", ":9", "", "  ", "abc"} {
		if _, ok := parseRatio(bad); ok {
			t.Errorf("✗ parseRatio(%q) parsed, want reject", bad)
		}
	}

	for _, good := range []string{"1024X768", " 1024 x 768 ", "1x1", "16x9"} {
		w, h, ok := ParseDimensions(good)
		if !ok || w <= 0 || h <= 0 {
			t.Errorf("✗ ParseDimensions(%q) = (%d,%d,%v), want positive edges", good, w, h, ok)
		}
	}

	for _, bad := range []string{"1024x768x1", "1024 768", "0x100", "100x0", "-5x10", "1024x", "x768", "99999999999999999999x1", ""} {
		if _, _, ok := ParseDimensions(bad); ok {
			t.Errorf("✗ ParseDimensions(%q) parsed, want reject", bad)
		}
	}

	if !t.Failed() {
		t.Log("✓ parseRatio/ParseDimensions accept finite boundary forms and reject malformed ones, with positive results")
	}
}

// TestAspectNearestAdversarial verifies invariant #2: Nearest aspect selection under extreme input.
//
// What is being tested:
// Given extreme ratios and a fixed aspect set, aspectNearest must return a set member or an empty
// string. Padded 1:1 must return 1:1 with membership true; malformed, empty, and zero ratios must
// return an empty string with membership false.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestAspectNearestAdversarial(t *testing.T) {
	set := []string{"16:9", "9:16", "1:1"}
	for _, in := range []string{"7:5", "1000000:1", "1:1000000", "100:1", "  16:9  ", "9:16"} {
		got, _ := aspectNearest(in, set)
		if got != "" && !slices.Contains(set, got) {
			t.Errorf("✗ aspectNearest(%q) = %q, not a member of the set", in, got)
		}
	}

	if got, member := aspectNearest(" 1:1 ", set); got != "1:1" || !member {
		t.Errorf("✗ aspectNearest(' 1:1 ') = (%q,%v), want (1:1,true) trimmed passthrough", got, member)
	}

	for _, bad := range []string{"garbage", "", "16", "16:0", "0:0"} {
		if got, member := aspectNearest(bad, set); got != "" || member {
			t.Errorf("✗ aspectNearest(%q) = (%q,%v), want ('',false) for an unparseable value", bad, got, member)
		}
	}

	if !t.Failed() {
		t.Log("✓ aspectNearest returns a set member or '', passes in-set values through, rejects unparseable input")
	}
}

// TestFreeSizeExtreme verifies invariant #3: Free-form sizing under extreme ratios.
//
// What is being tested:
// Given 4:1, 10:1, 100:1, and 1:10, deriveSize must return parseable dimensions satisfying the
// fixture's ratio cap, edge cap, pixel range, and 16-pixel alignment.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFreeSizeExtreme(t *testing.T) {
	b := freeFormBounds(t)

	for _, a := range []string{"4:1", "10:1", "100:1", "1:10"} {
		r, ok := parseRatio(a)
		if !ok {
			t.Fatalf("💣 fixture aspect %q does not parse", a)
		}

		size := deriveSize(b, r)

		w, h, ok := ParseDimensions(size)
		if !ok {
			t.Errorf("✗ deriveSize(%q) = %q, not a parseable WxH", a, size)

			continue
		}

		assertBounds(t, "aspect "+a, w, h)
	}

	if !t.Failed() {
		t.Log("✓ deriveSize caps steep ratios within the free-form constraints")
	}
}

// TestBoundsAdversarial verifies invariant #4: Free-form bounds under extreme input.
//
// What is being tested:
// Given the listed pixel, edge, ratio, and increment boundaries, clampFree must return the exact
// expected dimensions, including 816x816 below the pixel floor and unchanged 3840x2160 at the
// ceiling. For a pixel band that admits no pair of 16-pixel increments, it must return 0x0.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestBoundsAdversarial(t *testing.T) {
	cases := []struct {
		name         string
		b            SizeBounds
		inW, inH     int
		wantW, wantH int
	}{
		{"just below the pixel floor grows", freeFormBounds(t), 816, 800, 816, 816},
		{"the exact pixel ceiling passes unchanged", freeFormBounds(t), 3840, 2160, 3840, 2160},
		{"ratio exactly 3:1 rounds within the cap", freeFormBounds(t), 3000, 1000, 3008, 1008},
		{"the post-rounding ceiling repair shrinks the long edge", SizeBounds{MaxPx: 10000, EdgeIncrem: 16}, 112, 96, 96, 96},
		{"the step-less ratio cap truncates", SizeBounds{MaxRatio: 2}, 10, 1, 2, 1},
		// A MaxEdge that is not a step multiple: rounding ends past the cap and the repair
		// loop must shrink the long edge back inside it.
		{"a non-step-multiple edge cap shrinks back inside", SizeBounds{MaxEdge: 90, EdgeIncrem: 16}, 200, 50, 80, 16},
		// The portrait mirror of the ceiling repair: the taller edge shrinks.
		{"the portrait ceiling repair shrinks the taller edge", SizeBounds{MaxPx: 10000, EdgeIncrem: 16}, 96, 112, 96, 96},
	}
	for _, c := range cases {
		if w, h := clampFree(c.b, c.inW, c.inH); w != c.wantW || h != c.wantH {
			t.Errorf("✗ %s: clampFree(%d,%d) = %dx%d, want %dx%d", c.name, c.inW, c.inH, w, h, c.wantW, c.wantH)
		}
	}
	// No pair of positive 16-pixel increments has an area in this band.
	if w, h := clampFree(SizeBounds{MinPx: 10000, MaxPx: 10100, EdgeIncrem: 16}, 96, 96); w != 0 || h != 0 {
		t.Errorf("✗ an unsatisfiable band returned usable edges %dx%d", w, h)
	}

	if !t.Failed() {
		t.Log("✓ the boundary and repair paths hold at the floor, ceiling, exact ratio cap, and unsatisfiable band")
	}
}

// TestSizingDegenerate verifies invariant #5: Sizing with incomplete constraints.
//
// What is being tested:
// Given no LongEdge, deriveSize must return an empty string. nearestSize and PickSize must skip
// malformed size strings, choose the sole parseable candidate even as an orientation fallback, and
// return an empty string when no candidate parses.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSizingDegenerate(t *testing.T) {
	if got := deriveSize(SizeBounds{MaxRatio: 3}, 1.5); got != "" {
		t.Errorf("✗ deriveSize with no LongEdge = %q, want empty (nothing to derive from)", got)
	}

	if got := nearestSize([]string{"junk", "1024x1024"}, 1.0); got != "1024x1024" {
		t.Errorf("✗ nearestSize skipping a junk member = %q, want 1024x1024", got)
	}

	if got := nearestSize([]string{"junk"}, 1.0); got != "" {
		t.Errorf("✗ nearestSize over junk-only members = %q, want empty", got)
	}

	if got := PickSize([]string{"junk", "720x1280"}, true, 720, Nullable[float64]{}); got != "720x1280" {
		t.Errorf("✗ pickSize orientation fallback = %q, want the only parseable member", got)
	}

	if got := PickSize([]string{"junk"}, true, 720, Nullable[float64]{}); got != "" {
		t.Errorf("✗ pickSize over junk-only members = %q, want empty", got)
	}

	if !t.Failed() {
		t.Log("✓ derivation and selection degrade cleanly on degenerate bounds configs")
	}
}

// TestDimensionsAllConstraints verifies invariant #6: Simultaneous dimension constraints.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Test kind: permanent.
// What is being tested:
// Given the listed extreme requests, including math.MaxInt edges and conflicting adjustment
// pressures, clampFree must return positive dimensions satisfying every declared edge, pixel,
// ratio, and increment bound.
func TestDimensionsAllConstraints(t *testing.T) {
	cases := []struct {
		name          string
		bounds        SizeBounds
		width, height int
	}{
		{"BFL extreme landscape", SizeBounds{MinEdge: 64, MaxPx: 4194304}, 1000000, 1},
		{"BFL extreme portrait", SizeBounds{MinEdge: 64, MaxPx: 4194304}, 1, 1000000},
		{"opposing area and ratio", SizeBounds{MinEdge: 64, MaxPx: 65536, MinPx: 32768, MaxRatio: 2, EdgeIncrem: 16}, 1000000, 1},
		{"rounded edge boundary", SizeBounds{MinEdge: 65, MaxEdge: 100, MinPx: 6400, MaxPx: 9216, EdgeIncrem: 16}, 100, 65},
		{"overflowing requested area", SizeBounds{MinEdge: 64, MaxPx: 4194304, EdgeIncrem: 16}, math.MaxInt, math.MaxInt},
		{"overflowing requested edge", SizeBounds{MinEdge: 64, MaxPx: 4194304, EdgeIncrem: 16}, math.MaxInt, 1},
	}
	for _, dimensionCase := range cases {
		t.Run(dimensionCase.name, func(t *testing.T) {
			width, height := clampFree(dimensionCase.bounds, dimensionCase.width, dimensionCase.height)
			shortEdge, longEdge := min(width, height), max(width, height)

			bounds := dimensionCase.bounds
			if shortEdge <= 0 || shortEdge < bounds.MinEdge || (bounds.MaxEdge > 0 && longEdge > bounds.MaxEdge) {
				t.Errorf("✗ edges %dx%d violate %+v", width, height, bounds)
			} else {
				if bounds.MaxPx > 0 && longEdge > bounds.MaxPx/shortEdge {
					t.Errorf("✗ area exceeds %d: %dx%d", bounds.MaxPx, width, height)
				}

				if bounds.MinPx > 0 && longEdge < (bounds.MinPx-1)/shortEdge+1 {
					t.Errorf("✗ area below %d: %dx%d", bounds.MinPx, width, height)
				}

				if bounds.MaxRatio > 0 && float64(longEdge)/float64(shortEdge) > bounds.MaxRatio {
					t.Errorf("✗ ratio exceeds %g: %dx%d", bounds.MaxRatio, width, height)
				}

				if bounds.EdgeIncrem > 0 && (width%bounds.EdgeIncrem != 0 || height%bounds.EdgeIncrem != 0) {
					t.Errorf("✗ dimensions do not satisfy increment: %dx%d", width, height)
				}
			}

			if !t.Failed() {
				t.Log("✓ every dimension constraint holds")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ feasible boundary inputs stay valid")
	}
}

// TestDimensionDeclarations verifies invariant #7: Dimension feasibility.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// For the enumerated bounds and small, distorted, or math.MaxInt requests, clampFree must return
// positive dimensions satisfying every constraint whenever exhaustive enumeration finds a feasible
// pair. Otherwise it must return 0x0.
// Kind: permanent.
func TestDimensionDeclarations(t *testing.T) {
	for _, maximumEdge := range []int{1, 8, 17} {
		for _, minimumEdge := range []int{0, 3, 9} {
			for _, step := range []int{1, 4, 8} {
				for _, ratio := range []float64{0, 1, 1.5} {
					for _, minimumPixels := range []int{0, 16, 65} {
						for _, maximumPixels := range []int{0, 15, 64} {
							bounds := SizeBounds{MinEdge: minimumEdge, MaxEdge: maximumEdge, EdgeIncrem: step, MaxRatio: ratio, MinPx: minimumPixels, MaxPx: maximumPixels}
							for _, dimensions := range [][2]int{{1, 1}, {100, 3}, {math.MaxInt, 1}} {
								checkDimensionFeasibility(t, bounds, dimensions[0], dimensions[1])
							}
						}
					}
				}
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ feasible and impossible declarations agree with exhaustive dimensions")
	}
}

// TestFreeSizeConstraints verifies invariant #8: Free-form size constraints.
//
// What is being tested:
// Given the listed extreme ratios and dimensions, deriveSize and clampFree must satisfy the
// fixture's ratio cap, edge cap, pixel range, and 16-pixel alignment simultaneously.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFreeSizeConstraints(t *testing.T) {
	b := freeFormBounds(t)

	aspects := []string{"10:1", "1:10", "100:1", "1:100", "3:1", "1:3", "16:9", "1:1", "2.99:1", "3.01:1"}
	for _, a := range aspects {
		r, ok := parseRatio(a)
		if !ok {
			t.Fatalf("💣 fixture aspect %q does not parse", a)
		}

		size := deriveSize(b, r)

		w, h, ok := ParseDimensions(size)
		if !ok {
			t.Errorf("✗ deriveSize(%q) = %q, not a WxH size", a, size)

			continue
		}

		assertBounds(t, "aspect "+a, w, h)
	}

	sizes := [][2]int{{10000, 1000}, {1000, 10000}, {20, 20}, {16, 9999}, {5000, 5000}, {640, 1024}}
	for _, c := range sizes {
		w, h := clampFree(b, c[0], c[1])
		assertBounds(t, fmtWH(c[0], c[1]), w, h)
	}

	if !t.Failed() {
		t.Log("✓ every free-form size satisfies ratio, edge, pixel-range, and multiple-of-16 constraints after rounding")
	}
}

// TestParseHelpers verifies invariant #9: Width-height and aspect-ratio parsing.
//
// What is being tested:
// ParseDimensions must accept 1024x768 and reject the listed malformed or nonpositive dimensions.
// parseRatio must return 1.5 for 3:2 and reject malformed, non-finite, overflowing, or underflowing
// ratios. fmtWH must format 1280 and 720 as 1280x720.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestParseHelpers(t *testing.T) {
	if _, _, ok := ParseDimensions("1024x768"); !ok {
		t.Errorf("✗ valid WxH should parse")
	}

	for _, bad := range []string{"x", "1024x", "0x0", "-1x2", "axb", "1024"} {
		if _, _, ok := ParseDimensions(bad); ok {
			t.Errorf("✗ %q should not parse as WxH", bad)
		}
	}

	if r, ok := parseRatio("3:2"); !ok || r != 1.5 {
		t.Errorf("✗ parseRatio(3:2) = (%v, %v), want (1.5, true)", r, ok)
	}
	// Non-finite components must be rejected: ParseFloat accepts nan/inf spellings, and a NaN
	// or Inf ratio would silently corrupt every nearest-by-distance selection downstream. So
	// must a division that overflows to infinity or underflows to zero from finite components.
	for _, bad := range []string{
		"", "a:b", "16:0", "16", "1:2:3",
		"nan:1", "1:nan", "inf:1", "1:inf", "-inf:1", "nan:nan",
		"1e308:1e-308", "1e-308:1e308",
	} {
		if _, ok := parseRatio(bad); ok {
			t.Errorf("✗ %q should not parse as ratio", bad)
		}
	}

	if got := fmtWH(1280, 720); got != "1280x720" {
		t.Errorf("✗ fmtWH(1280,720) = %q, want 1280x720", got)
	}

	if !t.Failed() {
		t.Log("✓ parse helpers accept valid, reject malformed")
	}
}

// TestSnapIntRange verifies invariant #10: Discrete integer snapping.
//
// What is being tested:
// Given each listed allowed set and inputs from math.MinInt through math.MaxInt, snapInt must
// return a set member. Given 6 and the set 4, 8, 12, it must resolve the tie to 4.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSnapIntRange(t *testing.T) {
	sets := [][]int{{8}, {4, 6, 8}, {4, 8, 12}}
	for _, set := range sets {
		for _, n := range []int{math.MinInt, -100, 0, 1, 5, 10, 100, math.MaxInt} {
			got := snapInt(n, set)
			if !slices.Contains(set, got) {
				t.Errorf("✗ snapInt(%d, %v) = %d, not a member of the set", n, set, got)
			}
		}
	}

	if got := snapInt(6, []int{4, 8, 12}); got != 4 {
		t.Errorf("✗ snapInt(6, [4 8 12]) = %d, want 4 (equidistant tie resolves to the first member)", got)
	}

	if !t.Failed() {
		t.Log("✓ snapInt always returns a member of the allowed set; ties resolve to the first member")
	}
}

// TestRepHeightTable verifies invariant #11: Resolution label height mapping.
//
// What is being tested:
// Given the listed K, P, dimension, numeric, and underscore-suffix labels, repHeight must return
// the exact expected heights. It must reject malformed suffixes, nonpositive heights, and
// overflowing K values.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRepHeightTable(t *testing.T) {
	cases := map[string]int{
		"4k":         2160,
		"8k":         4320,
		"720p":       720,
		"1k":         540,
		"2k":         1080,
		"1024x768":   768,
		"900":        900,
		"true_1080p": 1080,
		"hd_2k":      1080,
	}
	for in, want := range cases {
		if h, ok := repHeight(in); !ok || h != want {
			t.Errorf("✗ repHeight(%q) = (%d,%v), want (%d,true)", in, h, ok, want)
		}
	}

	for _, bad := range []string{"true_", "_", "a_b", "true_garbage", "-5p", "0p", "17080700000000000k", "9223372036854775807k"} {
		if h, ok := repHeight(bad); ok {
			t.Errorf("✗ repHeight(%q) = (%d,true), want unparseable (overflowing digits are outside the mapper's domain)", bad, h)
		}
	}

	if !t.Failed() {
		t.Log("✓ the representative-height table holds, trailing-token labels included")
	}
}

// TestRoundedRatioCorrection verifies invariant #12: Rounded ratio correction.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given 40x20, ratio cap 1.5, and increment 16, clampFree must return exactly 32x32.
// Kind: permanent.
func TestRoundedRatioCorrection(t *testing.T) {
	width, height := clampFree(SizeBounds{MaxRatio: 1.5, EdgeIncrem: 16}, 40, 20)
	if width != 32 || height != 32 {
		t.Errorf("✗ dimensions = %dx%d, want the established 32x32 correction", width, height)
	}

	if !t.Failed() {
		t.Log("✓ ordinary ratio correction is preserved")
	}
}

// FuzzDimensionDeclarations verifies invariant #7: Dimension feasibility.
// Test class: Expanded.
// Test layer: Fuzzing.
// What is being tested:
// For generated bounded declarations and positive requests, clampFree must agree with exhaustive
// feasibility: return positive dimensions satisfying every constraint when a pair exists, or 0x0
// when none exists.
// Kind: permanent.
func FuzzDimensionDeclarations(f *testing.F) {
	f.Add(uint8(7), uint8(2), uint8(4), uint8(1), uint16(32), uint16(64), uint16(100), uint16(1))
	f.Add(uint8(17), uint8(9), uint8(8), uint8(2), uint16(65), uint16(15), uint16(1), uint16(1))
	f.Fuzz(func(t *testing.T, maxEdge, minEdge, step, ratio uint8, minPixels, maxPixels, width, height uint16) {
		bounds := SizeBounds{MaxEdge: int(maxEdge%32) + 1, MinEdge: int(minEdge % 33), EdgeIncrem: int(step%8) + 1, MaxRatio: float64(ratio%6) / 2, MinPx: int(minPixels % 1025), MaxPx: int(maxPixels % 1025)}
		checkDimensionFeasibility(t, bounds, int(width)+1, int(height)+1)

		if !t.Failed() {
			t.Log("✓ dimension result satisfies the declared feasible set")
		}
	})
}

// FuzzParseRatio verifies invariant #13: Aspect ratio parsing under arbitrary input.
//
// What is being tested:
// For arbitrary strings, any ratio that parseRatio accepts must be positive and finite.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzParseRatio(f *testing.F) {
	for _, s := range []string{"16:9", "1:1", "", "a:b", "16:0", "-1:2", "16:9:3", "3.5:2", "nan:1", "1:inf", "1e308:1e-308", "1e-308:1e308"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		// An accepted ratio must be positive AND finite — NaN, Inf, and a quotient that
		// overflowed or underflowed all defeat every nearest-by-distance comparison
		// downstream.
		if r, ok := parseRatio(s); ok && (!(r > 0) || math.IsInf(r, 0)) {
			t.Errorf("✗ parseRatio(%q) ok but r=%v is not positive finite", s, r)
		}

		if !t.Failed() {
			t.Logf("✓ a parsed ratio stayed positive and finite")
		}
	})
}

// FuzzParseWHsize verifies invariant #14: Width-height parsing under arbitrary input.
//
// What is being tested:
// For arbitrary strings, any dimensions that ParseDimensions accepts must have positive width and
// height.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzParseWHsize(f *testing.F) {
	for _, s := range []string{"1024x768", "x", "1024x", "0x0", "-1x2", "axb", "100X100"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if w, h, ok := ParseDimensions(s); ok && (w <= 0 || h <= 0) {
			t.Errorf("✗ ParseDimensions(%q) ok but %dx%d not positive", s, w, h)
		}

		if !t.Failed() {
			t.Logf("✓ parsed edges stayed positive")
		}
	})
}

// FuzzSnapInt verifies invariant #15: Discrete integer snapping under arbitrary input.
//
// What is being tested:
// For arbitrary integers and allowed values 4, 8, and 12, snapInt must return one of those three
// values.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSnapInt(f *testing.F) {
	for _, n := range []int{4, 8, 12, 0, -7, 100} {
		f.Add(n)
	}

	f.Fuzz(func(t *testing.T, n int) {
		set := []int{4, 8, 12}
		if got := snapInt(n, set); !slices.Contains(set, got) {
			t.Errorf("✗ snapInt(%d) = %d, want a member of %v", n, got, set)
		}

		if !t.Failed() {
			t.Logf("✓ the adjustment stayed inside the allowed set")
		}
	})
}

// FuzzResHeight verifies invariant #16: Resolution height classification under arbitrary input.
//
// What is being tested:
// For arbitrary labels, any height that repHeight accepts must be positive, and nearestRes must map
// it to a member of resSet.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzResHeight(f *testing.F) {
	for _, s := range []string{"720p", "4k", "1024x768", "garbage", "", "2k", "true_1080p", "17080700000000000k"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if h, ok := repHeight(s); ok {
			if h <= 0 {
				t.Errorf("✗ repHeight(%q) = %d, want positive", s, h)
			} else if nearest := nearestRes(h, resSet); !slices.Contains(resSet, nearest) {
				t.Errorf("✗ nearestRes(%d from %q) = %q, not a member of %v", h, s, nearest, resSet)
			}
		}

		if !t.Failed() {
			t.Logf("✓ the input classified; a parseable height mapped to a set member")
		}
	})
}

// FuzzAspectNearest verifies invariant #17: Nearest aspect selection under arbitrary input.
//
// What is being tested:
// For arbitrary strings and the fixed aspect set, aspectNearest must return an empty string or a
// declared member. If it reports membership, its result must be a declared member.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzAspectNearest(f *testing.F) {
	for _, s := range []string{"16:9", "7:5", "1:1000000", "auto", "garbage", ""} {
		f.Add(s)
	}

	set := []string{"16:9", "9:16", "1:1", "4:3"}

	f.Fuzz(func(t *testing.T, in string) {
		got, member := aspectNearest(in, set)
		if got != "" && !slices.Contains(set, got) {
			t.Errorf("✗ aspectNearest(%q) = %q, not a member of the set", in, got)
		}

		if member && !slices.Contains(set, got) {
			t.Errorf("✗ aspectNearest(%q) reported in-set membership for a non-member %q", in, got)
		}

		if !t.Failed() {
			t.Logf("✓ the aspect classified as a set member or empty")
		}
	})
}

// FuzzClampFree verifies invariant #18: Free-form size clamping under arbitrary input.
//
// What is being tested:
// For arbitrary positive dimensions and each satisfiable fixture of bounds, clampFree must return
// positive dimensions satisfying every declared edge, pixel, ratio, and increment constraint.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzClampFree(f *testing.F) {
	for _, c := range [][2]int{{1536, 154}, {10000, 1000}, {20, 20}, {16, 9999}, {5000, 5000}, {1, 1}, {3840, 1280}, {200, 50}, {96, 112}} {
		f.Add(c[0], c[1])
	}

	f.Fuzz(func(t *testing.T, w, h int) {
		if w > 0 && h > 0 {
			for _, b := range fuzzBounds(t) {
				cw, ch := clampFree(b, w, h)
				checkFreeBounds(t, b, w, h, cw, ch)
			}
		}

		if !t.Failed() {
			t.Logf("✓ in-domain inputs held every declared free-form constraint simultaneously")
		}
	})
}

// boundsViolations names every declared constraint the size breaks.
func boundsViolations(test testing.TB, b SizeBounds, cw, ch int) []string {
	test.Helper()

	long, short := max(cw, ch), min(cw, ch)
	checks := []struct {
		declared bool
		broken   bool
		name     string
	}{
		{b.EdgeIncrem > 0, b.EdgeIncrem > 0 && (cw%b.EdgeIncrem != 0 || ch%b.EdgeIncrem != 0), "edges off the step"},
		{b.MaxEdge > 0, long > b.MaxEdge, "long edge over the cap"},
		{b.MinEdge > 0, short < b.MinEdge, "short edge under the floor"},
		{b.MaxRatio > 0, float64(long)/float64(short) > b.MaxRatio+1e-9, "ratio over the cap"},
		{b.MinPx > 0, cw*ch < b.MinPx, "pixels under the floor"},
		{b.MaxPx > 0, cw*ch > b.MaxPx, "pixels over the ceiling"},
	}

	var out []string

	for _, c := range checks {
		if c.declared && c.broken {
			out = append(out, c.name)
		}
	}

	return out
}

// checkDimensionFeasibility compares adjustment to the complete bounded set of dimensions.
func checkDimensionFeasibility(t *testing.T, bounds SizeBounds, requestedWidth, requestedHeight int) {
	t.Helper()

	feasible := false

	for width := 1; width <= bounds.MaxEdge; width++ {
		for height := 1; height <= bounds.MaxEdge; height++ {
			if len(boundsViolations(t, bounds, width, height)) == 0 {
				feasible = true

				break
			}
		}

		if feasible {
			break
		}
	}

	width, height := clampFree(bounds, requestedWidth, requestedHeight)
	if !feasible {
		if width != 0 || height != 0 {
			t.Errorf("✗ impossible bounds %+v returned %dx%d", bounds, width, height)
		}

		return
	}

	if width <= 0 || height <= 0 {
		t.Errorf("✗ feasible bounds %+v rejected %dx%d", bounds, requestedWidth, requestedHeight)

		return
	}

	if violations := boundsViolations(t, bounds, width, height); len(violations) > 0 {
		t.Errorf("✗ bounds %+v returned %dx%d: %v", bounds, width, height, violations)
	}
}

// fuzzBounds returns satisfiable combinations of edge, pixel, ratio, and increment constraints.
func fuzzBounds(test testing.TB) []SizeBounds {
	test.Helper()

	return []SizeBounds{
		freeFormBounds(test),
		{MinEdge: 256, MaxEdge: 1440, EdgeIncrem: 32, LongEdge: 1024},
		{MinEdge: 64, MaxPx: 4194304, LongEdge: 1024},
		{MaxEdge: 90, EdgeIncrem: 16},
		{MaxRatio: 2, MinPx: 4096, EdgeIncrem: 8},
	}
}

// checkFreeBounds asserts every constraint b declares holds simultaneously on the constrained
// result, collecting each violation by name.
func checkFreeBounds(t *testing.T, b SizeBounds, w, h, cw, ch int) {
	t.Helper()

	if cw <= 0 || ch <= 0 {
		t.Errorf("✗ clampFree(%d,%d) = %dx%d, non-positive edges", w, h, cw, ch)

		return
	}

	for _, v := range boundsViolations(t, b, cw, ch) {
		t.Errorf("✗ clampFree(%d,%d) = %dx%d, %s", w, h, cw, ch, v)
	}
}
