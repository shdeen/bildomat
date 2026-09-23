package params_test

// Invariants tested:
//  1. Non-finite ratios: Given aspect or resolution ratios containing NaN, infinity, or division
//     that overflows or underflows, params.Adjust must return no values and exactly one Dropped
//     record identifying the original flag and input.
//  2. Parameter adjustment at a tie boundary: Given 810p between the declared 1k and 2k heights,
//     params.Adjust must select the first tier, 1k, and return the exact Snapped record from 810p
//     to 1k.
//  3. Custom-size adjustment without a long edge: Given aspect 16:9 and custom bounds without
//     LongEdge, params.Adjust must omit size and return exactly the Dropped aspect record.
//  4. Invalid thinking levels: Given thinking level zz and allowed values low and high,
//     params.Adjust must return exactly one Rejected record with input zz and comment low|high.
//  5. Image count limits: Given image counts 20, 0, and -3, params.Adjust must return 10, 1, and 1
//     with the exact Capped or Raised record.
//  6. Impossible dimension declarations: Given minimum edge 1024 above maximum edge 1000,
//     params.Adjust must reject explicit size, resolution dimensions, and aspect-derived dimensions
//     with ErrProvConfigInvalid.
//  7. Parameter adjustment under arbitrary input: For arbitrary inputs across the fixture models,
//     params.Adjust must keep size, quality, and resolution values inside their declared sets and
//     duration inside its declared set or range.
//  8. Explicit size derivation under arbitrary input: For arbitrary explicit size strings on the
//     fixed-size fixture, params.Adjust must return a declared size member or omit size and include
//     a Dropped size record, without an adjustment error.
//  9. Free-form size derivation under arbitrary input: For arbitrary aspect and resolution inputs
//     on the custom-size fixture, params.Adjust must return no adjustment error.

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// TestNonFiniteRatios verifies invariant #1: Non-finite ratios.
//
// What is being tested:
// Given aspect or resolution ratios containing NaN, infinity, or division that overflows or
// underflows, params.Adjust must return no values and exactly one Dropped record identifying the
// original flag and input.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestNonFiniteRatios(t *testing.T) {
	cases := []adjustCase{
		{
			name: "a nan aspect drops with a record, nothing sent",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "nan:1"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDropped, "nan:1", "", "")},
		},
		{
			name: "an inf aspect drops with a record, nothing sent",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "inf:1"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDropped, "inf:1", "", "")},
		},
		{
			name: "a nan resolution drops with a record, nothing sent",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "1:nan"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeDropped, "1:nan", "", "")},
		},
		{
			name: "an inf resolution drops with a record, nothing sent",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "inf:1"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeDropped, "inf:1", "", "")},
		},
		{
			name: "an overflowing finite-component aspect drops with a record",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "1e308:1e-308"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDropped, "1e308:1e-308", "", "")},
		},
		{
			name: "an underflowing finite-component aspect drops with a record",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "1e-308:1e308"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDropped, "1e-308:1e308", "", "")},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runAdjust(t, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ nan/inf ratio forms drop with a record and send nothing")
	}
}

// TestAdjustTieBoundary verifies invariant #2: Parameter adjustment at a tie boundary.
//
// What is being tested:
// Given 810p between the declared 1k and 2k heights, params.Adjust must select the first tier, 1k,
// and return the exact Snapped record from 810p to 1k.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestAdjustTieBoundary(t *testing.T) {
	model := fixedAspectImageModel(t)

	gp, paramChanges := adjustParameters(t, params.FlagInputs{params.FlagTypeResolution: "810p"}, &model)
	if adjustedRes, _ := gp[params.FlagTypeResolution].(string); adjustedRes != "1k" {
		t.Errorf("✗ the 810p tier tie = %q, want the first declared member 1k", adjustedRes)
	}

	want := []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeSnapped, "810p", "1k", "")}
	if !reflect.DeepEqual(paramChanges, want) {
		t.Errorf("✗ tie records = %+v, want %+v", paramChanges, want)
	}

	if !t.Failed() {
		t.Log("✓ an equidistant representative height adjusts to the first declared tier")
	}
}

// TestAdjustCustomNoLongEdge verifies invariant #3: Custom-size adjustment without a long edge.
//
// What is being tested:
// Given aspect 16:9 and custom bounds without LongEdge, params.Adjust must omit size and return
// exactly the Dropped aspect record.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestAdjustCustomNoLongEdge(t *testing.T) {
	m := replaceParamCfg(t, customModel(t), params.Definition{FlagID: params.FlagTypeSize, ParamID: "size", CustomSize: &params.SizeBounds{MaxRatio: 3}})

	gp, paramChanges := adjustParameters(t, params.FlagInputs{params.FlagTypeAspect: "16:9"}, &m)
	if adjustedSize, ok := gp[params.FlagTypeSize]; ok {
		t.Errorf("✗ a size was invented without a LongEdge: %q", adjustedSize)
	}

	want := []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDropped, "16:9", "", "")}
	if !reflect.DeepEqual(paramChanges, want) {
		t.Errorf("✗ records = %+v, want the aspect dropped", paramChanges)
	}

	if !t.Failed() {
		t.Log("✓ ratio derivation without a LongEdge drops the aspect instead of inventing a size")
	}
}

// TestThinkingLevelRejectsNamingModelSet verifies invariant #4: Invalid thinking levels.
//
// What is being tested:
// Given thinking level zz and allowed values low and high, params.Adjust must return exactly one
// Rejected record with input zz and comment low|high.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestThinkingLevelRejectsNamingModelSet(t *testing.T) {
	model := parameterModel{
		label: "fixture-thinking-image",
		definitions: []params.Definition{
			{FlagID: params.FlagTypeThinkingLevel, ParamID: "thinking_level", AllowedValues: []string{"low", "high"}},
		},
	}
	_, paramChanges := adjustParameters(t, params.FlagInputs{params.FlagTypeThinkingLevel: "zz"}, &model)

	want := []params.Adjustment{ch(t, params.FlagTypeThinkingLevel, params.ChangeRejected, "zz", "", "low|high")}
	if !reflect.DeepEqual(paramChanges, want) {
		t.Errorf("✗ changes = %+v, want %+v", paramChanges, want)
	}

	if !t.Failed() {
		t.Log("✓ an unknown thinking-level token rejects naming the model's own set")
	}
}

// TestCapNOutcomes verifies invariant #5: Image count limits.
//
// What is being tested:
// Given image counts 20, 0, and -3, params.Adjust must return 10, 1, and 1 with the exact Capped or
// Raised record. It must preserve in-range count 5 and unlimited count 7 without records.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCapNOutcomes(t *testing.T) {
	capFree := replaceParamCfg(t, trioModel(t), params.Definition{FlagID: params.FlagTypeImageN, ParamID: "n"})

	cases := []struct {
		n, want int
		model   parameterModel
		change  *params.Adjustment
	}{
		{20, 10, trioModel(t), &params.Adjustment{
			FlagID: params.FlagTypeImageN, Type: params.ChangeCapped, InputVal: "20",
			WireVal: "10", Comment: "max 10",
		}},
		{0, 1, trioModel(t), &params.Adjustment{FlagID: params.FlagTypeImageN, Type: params.ChangeRaised, InputVal: "0", WireVal: "1", Comment: "min 1"}},
		{-3, 1, trioModel(t), &params.Adjustment{FlagID: params.FlagTypeImageN, Type: params.ChangeRaised, InputVal: "-3", WireVal: "1", Comment: "min 1"}},
		{5, 5, trioModel(t), nil},
		{7, 7, capFree, nil},
	}
	for _, c := range cases {
		gp, paramChanges := adjustParameters(t, params.FlagInputs{params.FlagTypeImageN: c.n}, &c.model)
		if gp[params.FlagTypeImageN] != c.want {
			t.Errorf("✗ count %d = %v, want %d", c.n, gp[params.FlagTypeImageN], c.want)
		}

		switch {
		case c.change == nil && len(paramChanges) != 0:
			t.Errorf("✗ count %d recorded %+v, want no record", c.n, paramChanges)
		case c.change != nil && (len(paramChanges) != 1 || paramChanges[0] != *c.change):
			t.Errorf("✗ count %d records = %+v, want exactly %+v", c.n, paramChanges, c.change)
		}
	}

	if !t.Failed() {
		t.Log("✓ the count-cap outcomes hold: below-one raises to one, over-max caps, in-range and cap-free pass")
	}
}

// TestImpossibleDimensionDeclarations verifies invariant #6: Impossible dimension declarations.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given minimum edge 1024 above maximum edge 1000, params.Adjust must reject explicit size,
// resolution dimensions, and aspect-derived dimensions with ErrProvConfigInvalid. The error must
// name the model and both bounds, and the result must contain no values or records.
// Kind: permanent.
func TestImpossibleDimensionDeclarations(t *testing.T) {
	definitions := params.Definitions{
		{FlagID: params.FlagTypeSize, CustomSize: &params.SizeBounds{MinEdge: 1024, MaxEdge: 1000, LongEdge: 1536}},
		{FlagID: params.FlagTypeResolution},
		{FlagID: params.FlagTypeAspect},
	}
	for _, inputs := range []params.FlagInputs{
		{params.FlagTypeSize: "512x512"},
		{params.FlagTypeResolution: "512x512"},
		{params.FlagTypeAspect: "16:9"},
	} {
		adjusted, changes, err := params.Adjust(inputs, definitions, "impossible-dimensions")
		if !errors.Is(err, errs.ErrProvConfigInvalid) {
			t.Errorf("✗ inputs %v returned error %v without configuration classification", inputs, err)
		}

		if err != nil {
			for _, context := range []string{"impossible-dimensions", "1024", "1000"} {
				if !strings.Contains(err.Error(), context) {
					t.Errorf("✗ error %q omits conflicting context %q", err, context)
				}
			}
		}

		if len(adjusted) != 0 || len(changes) != 0 {
			t.Errorf("✗ conflicting bounds returned values %v and notices %+v", adjusted, changes)
		}
	}

	if !t.Failed() {
		t.Log("✓ every dimension source rejects conflicting limits with context and no successful result")
	}
}

// FuzzAdjust verifies invariant #7: Parameter adjustment under arbitrary input.
//
// What is being tested:
// For arbitrary inputs across the fixture models, params.Adjust must keep size, quality, and
// resolution values inside their declared sets and duration inside its declared set or range.
// Supplied consumed aspect must remain as aspect, size, or a record; constrained quality and
// resolution must likewise remain represented by a value or record.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzAdjust(f *testing.F) {
	f.Add("16:9", "1080p", "high", 5)
	f.Add("", "1024x768", "ultra", 0)
	f.Add("garbage", "weird", "", 100)
	f.Add("100:1", "", "AUTO", -7)
	f.Add("nan:1", "inf:1", "", 4)
	f.Fuzz(func(t *testing.T, aspect, res, quality string, dur int) {
		cfg := params.FlagInputs{params.FlagTypeDuration: dur}
		if aspect != "" {
			cfg[params.FlagTypeAspect] = aspect
		}

		if res != "" {
			cfg[params.FlagTypeResolution] = res
		}

		if quality != "" {
			cfg[params.FlagTypeQuality] = quality
		}

		for _, model := range []parameterModel{trioModel(t), fixedSizeVideoModel(t), restrictedSizeVideoModel(t), fixedAspectImageModel(t), rangedDurationVideoModel(t)} {
			gp, paramChanges := adjustParameters(t, cfg, &model)
			checkAdjustInvariants(t, model, cfg, gp, paramChanges)
		}

		if !t.Failed() {
			t.Logf("✓ the fixed-set, quality, and duration invariants held")
		}
	})
}

// FuzzSizeSpec verifies invariant #8: Explicit size derivation under arbitrary input.
//
// What is being tested:
// For arbitrary explicit size strings on the fixed-size fixture, params.Adjust must return a
// declared size member or omit size and include a Dropped size record, without an adjustment error.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSizeSpec(f *testing.F) {
	for _, s := range []string{"1024x1024", "999x999", "garbage", "", "0x0", "1024X1536", " 1280 x 720 ", "16:9"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, size string) {
		model := trioModel(t)
		gp, paramChanges := adjustParameters(t, params.FlagInputs{params.FlagTypeSize: size}, &model)
		checkSizeClassified(t, size, gp, paramChanges)

		if !t.Failed() {
			t.Logf("✓ the explicit size classified as a sent member or a drop")
		}
	})
}

// FuzzFreeFormSize verifies invariant #9: Free-form size derivation under arbitrary input.
//
// What is being tested:
// For arbitrary aspect and resolution inputs on the custom-size fixture, params.Adjust must return
// no adjustment error. Any returned size must parse, have edges divisible by 16, and stay within
// the fixture's maximum edge and pixel count.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzFreeFormSize(f *testing.F) {
	f.Add("16:9", "")
	f.Add("", "4000x4000")
	f.Add("", "16:9")
	f.Add("1:1000", "")
	f.Fuzz(func(t *testing.T, aspect, resolution string) {
		cfg := params.FlagInputs{}
		if aspect != "" {
			cfg[params.FlagTypeAspect] = aspect
		}

		if resolution != "" {
			cfg[params.FlagTypeResolution] = resolution
		}

		model := customModel(t)

		gp, _ := adjustParameters(t, cfg, &model)
		if adjustedSize, _ := gp[params.FlagTypeSize].(string); adjustedSize != "" {
			checkCustomSize(t, aspect, resolution, adjustedSize)
		}

		if !t.Failed() {
			t.Logf("✓ any produced custom-mode size satisfied the declared constraints")
		}
	})
}

// recordFor reports whether any change record names param.
func recordFor(test testing.TB, paramChanges []params.Adjustment, param params.FlagType) bool {
	test.Helper()

	for _, c := range paramChanges {
		if c.FlagID == param {
			return true
		}
	}

	return false
}

// checkAdjustInvariants checks declared value sets and ranges and accounts for supplied aspect,
// quality, resolution, and duration inputs.
func checkAdjustInvariants(t *testing.T, model parameterModel, cfg params.FlagInputs, gp params.Values, paramChanges []params.Adjustment) {
	t.Helper()

	sizeCfg, _ := model.definitions.Param(params.FlagTypeSize)
	if adjustedSize, _ := gp[params.FlagTypeSize].(string); adjustedSize != "" && len(sizeCfg.AllowedValues) > 0 && !slices.Contains(sizeCfg.AllowedValues, adjustedSize) {
		t.Errorf("✗ %s: size %q not a declared member", model.label, adjustedSize)
	}

	checkAspectInvariant(t, model, cfg, gp, paramChanges)
	checkQualityInvariant(t, model, cfg, gp, paramChanges)
	checkResInvariant(t, model, cfg, gp, paramChanges)
	checkDurInvariant(t, model, gp)
}

// sentIn reports whether the adjusted map carries param.
func sentIn(test testing.TB, gp params.Values, param params.FlagType) bool {
	test.Helper()

	_, ok := gp[param]

	return ok
}

// checkAspectInvariant asserts a supplied consumed aspect is classified: sent as an aspect value,
// folded into a selected size, or recorded — never a silent vanish.
func checkAspectInvariant(t *testing.T, model parameterModel, cfg params.FlagInputs, gp params.Values, paramChanges []params.Adjustment) {
	t.Helper()

	_, aspectDeclared := model.definitions.Param(params.FlagTypeAspect)
	if _, ok := cfg[params.FlagTypeAspect]; !ok || !aspectDeclared {
		return
	}

	if !sentIn(t, gp, params.FlagTypeAspect) && !sentIn(t, gp, params.FlagTypeSize) && !recordFor(t, paramChanges, params.FlagTypeAspect) {
		t.Errorf("✗ %s: supplied aspect neither sent, folded into a size, nor recorded", model.label)
	}
}

// checkQualityInvariant asserts a supplied checked quality is classified: an accepted member sent,
// or a record explaining the miss.
func checkQualityInvariant(t *testing.T, model parameterModel, cfg params.FlagInputs, gp params.Values, paramChanges []params.Adjustment) {
	t.Helper()

	qualityCfg, cfgOK := model.definitions.Param(params.FlagTypeQuality)
	if _, ok := cfg[params.FlagTypeQuality]; !ok || !cfgOK || len(qualityCfg.AllowedValues) == 0 {
		return
	}

	adjustedQuality, _ := gp[params.FlagTypeQuality].(string)
	switch {
	case sentIn(t, gp, params.FlagTypeQuality) && !slices.Contains(qualityCfg.AllowedValues, adjustedQuality):
		t.Errorf("✗ %s: quality %q sent but not an accepted member", model.label, adjustedQuality)
	case !sentIn(t, gp, params.FlagTypeQuality) && !recordFor(t, paramChanges, params.FlagTypeQuality):
		t.Errorf("✗ %s: supplied quality neither sent nor recorded", model.label)
	}
}

// checkResInvariant asserts a supplied checked resolution is classified: a declared member sent,
// folded into a selected size, or recorded.
func checkResInvariant(t *testing.T, model parameterModel, cfg params.FlagInputs, gp params.Values, paramChanges []params.Adjustment) {
	t.Helper()

	resCfg, cfgOK := model.definitions.Param(params.FlagTypeResolution)
	if _, ok := cfg[params.FlagTypeResolution]; !ok || !cfgOK || len(resCfg.AllowedValues) == 0 {
		return
	}

	adjustedRes, _ := gp[params.FlagTypeResolution].(string)
	if sentIn(t, gp, params.FlagTypeResolution) && !slices.Contains(resCfg.AllowedValues, adjustedRes) {
		t.Errorf("✗ %s: resolution %q sent but not a declared member", model.label, adjustedRes)
	}

	if !sentIn(t, gp, params.FlagTypeResolution) && !sentIn(t, gp, params.FlagTypeSize) && !recordFor(t, paramChanges, params.FlagTypeResolution) {
		t.Errorf("✗ %s: supplied resolution neither sent, folded into a size, nor recorded", model.label)
	}
}

// checkDurInvariant asserts a supplied duration on a consuming model always resolves into the
// declared set or range and is sent.
func checkDurInvariant(t *testing.T, model parameterModel, gp params.Values) {
	t.Helper()

	durationCfg, ok := model.definitions.Param(params.FlagTypeDuration)
	if !ok {
		return
	}

	if !sentIn(t, gp, params.FlagTypeDuration) {
		t.Errorf("✗ %s: a supplied duration on a consuming model was not sent", model.label)

		return
	}

	adjustedSecs, _ := gp[params.FlagTypeDuration].(int)

	declaredDurations := make([]int, 0, len(durationCfg.AllowedValues))
	for _, declaredValue := range durationCfg.AllowedValues {
		if seconds, parseErr := strconv.Atoi(declaredValue); parseErr == nil {
			declaredDurations = append(declaredDurations, seconds)
		}
	}

	if len(durationCfg.AllowedValues) > 0 && !slices.Contains(declaredDurations, adjustedSecs) {
		t.Errorf("✗ %s: duration %d not in the declared set", model.label, adjustedSecs)
	}

	maxSecs, maxDeclared := durationCfg.MaxValue.ValIf()
	if maxDeclared && (adjustedSecs < int(durationCfg.MinValue.ValOr(0)) || adjustedSecs > int(maxSecs)) {
		t.Errorf("✗ %s: duration %d outside [%d,%d]", model.label, adjustedSecs, int(durationCfg.MinValue.ValOr(0)), int(maxSecs))
	}
}

// checkSizeClassified asserts one explicit-size outcome: a declared member sent, or the value
// dropped with nothing sent — every supplied value classified, the empty string included.
func checkSizeClassified(t *testing.T, size string, gp params.Values, paramChanges []params.Adjustment) {
	t.Helper()

	model := trioModel(t)

	trioSizeCfg, _ := model.definitions.Param(params.FlagTypeSize)
	if adjustedSize, ok := gp[params.FlagTypeSize]; ok {
		if !slices.Contains(trioSizeCfg.AllowedValues, adjustedSize.(string)) {
			t.Errorf("✗ --size %q sent %q, not a declared member", size, adjustedSize)
		}

		return
	}

	dropped := false

	for _, c := range paramChanges {
		if c.FlagID == params.FlagTypeSize && c.Type == params.ChangeDropped {
			dropped = true
		}
	}

	if !dropped {
		t.Errorf("✗ --size %q neither sent nor dropped", size)
	}
}

// checkCustomSize checks size syntax, 16-pixel alignment, and maximum edge and area.
func checkCustomSize(t *testing.T, aspect, resolution, size string) {
	t.Helper()

	w, h, ok := params.ParseDimensions(size)
	if !ok {
		t.Errorf("✗ (%q,%q) → unparseable %q", aspect, resolution, size)

		return
	}

	b := freeFormBounds(t)

	if w%16 != 0 || h%16 != 0 {
		t.Errorf("✗ %q edges not multiples of 16", size)
	}

	if w > b.MaxEdge || h > b.MaxEdge {
		t.Errorf("✗ %q exceeds max edge %d", size, b.MaxEdge)
	}

	if w*h > b.MaxPx {
		t.Errorf("✗ %q exceeds max pixels %d", size, b.MaxPx)
	}
}
