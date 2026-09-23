package params_test

// Invariants tested:
//  1. Explicit size adjustment: Given explicit sizes, params.Adjust must retain allowed values,
//     snap 999x999 to 1024x1024, and constrain 512x512 to 816x816 under custom bounds.
//  2. Custom-size parameter adjustment: Given custom size bounds, params.Adjust must let resolution
//     override aspect, constrain 512x512 to 816x816, retain 1024x1024, and derive the listed sizes
//     from ratios.
//  3. Fixed-size parameter adjustment: Given fixed allowed sizes, params.Adjust must choose the
//     listed dimensions from explicit dimensions, ratios, or resolution tiers, using resolution
//     before aspect.
//  4. Separate aspect and resolution adjustment: Given separate aspect and resolution sets,
//     params.Adjust must preserve members with declared spelling, snap 7:5 to 4:3, map 900p to 2k
//     and 1024x768 to 1k, and drop malformed values.
//  5. Parameter type adjustment: Given typed parameter inputs, params.Adjust must normalize
//     declared string members, reject nonmembers, snap or bound duration and image counts, and
//     preserve unconstrained finite numbers and Booleans.
//  6. Numeric parameter range adjustment: Given bounds zero through one, params.Adjust must cap 2
//     to 1, raise -0.5 to 0, and retain 0.7.
//  7. Numeric parameter set membership: Given numeric allowed sets, params.Adjust must retain
//     integer 3 and number 0.5 without records.
//  8. Typed parameter reads: Given an absent string parameter, params.Value[string] must return an
//     empty string without an error.
//  9. Declared size selection: Given declared dimensions, params.Adjust must prefer the nearest
//     short edge and use the requested ratio to break a tie, returning the exact listed size.
//  10. Nonnumerical resolution labels: Given allowed resolution labels HD and FHD, params.Adjust
//      must normalize padded hd to HD without a record.

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// TestAdjustExplicitSize verifies invariant #1: Explicit size adjustment.
//
// What is being tested:
// Given explicit sizes, params.Adjust must retain allowed values, snap 999x999 to 1024x1024, and
// constrain 512x512 to 816x816 under custom bounds. A valid size must supersede aspect and
// resolution with the listed Ignored records; an invalid size must produce a Dropped record before
// aspect selects a size. Models without size constraints must forward valid sizes, and models
// without size support must omit them. Each case must return the exact expected values and ordered
// records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdjustExplicitSize(t *testing.T) {
	rawSize := fixedSizeVideoModel(t)
	rawSize.label = "raw-size-model"
	rawSize = replaceParamCfg(t, rawSize, params.Definition{FlagID: params.FlagTypeSize, ParamID: "size"})

	cases := []adjustCase{
		{
			name: "fixed-set member passes as sent",
			cfg:  params.FlagInputs{params.FlagTypeSize: "1024x1536"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1536"}, sent: []params.FlagType{params.FlagTypeSize},
		},
		{
			name: "fixed-set miss adjusts to the nearest by ratio distance",
			cfg:  params.FlagInputs{params.FlagTypeSize: "999x999"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeSize, params.ChangeSnapped, "999x999", "1024x1024", "")},
		},
		{
			name: "valid size supersedes a supplied aspect",
			cfg:  params.FlagInputs{params.FlagTypeSize: "1024x1024", params.FlagTypeAspect: "3:2"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeIgnored, "", "", fmt.Sprintf(params.ReasonSupersededBySize, "1024x1024"))},
		},
		{
			name: "valid size supersedes a supplied resolution",
			cfg:  params.FlagInputs{params.FlagTypeSize: "1024x1024", params.FlagTypeResolution: "720p"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeIgnored, "", "", fmt.Sprintf(params.ReasonSupersededBySize, "1024x1024"))},
		},
		{
			name: "valid size supersedes both flags, one record each",
			cfg:  params.FlagInputs{params.FlagTypeSize: "1024x1024", params.FlagTypeAspect: "3:2", params.FlagTypeResolution: "720p"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{
				ch(t, params.FlagTypeAspect, params.ChangeIgnored, "", "", fmt.Sprintf(params.ReasonSupersededBySize, "1024x1024")),
				ch(t, params.FlagTypeResolution, params.ChangeIgnored, "", "", fmt.Sprintf(params.ReasonSupersededBySize, "1024x1024")),
			},
		},
		{
			name: "unparseable size drops and the modes take over",
			cfg:  params.FlagInputs{params.FlagTypeSize: "garbage", params.FlagTypeAspect: "16:9"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1536x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{
				ch(t, params.FlagTypeSize, params.ChangeDropped, "garbage", "", ""),
				ch(t, params.FlagTypeAspect, params.ChangeSnapped, "16:9", "1536x1024", ""),
			},
		},
		{
			name: "custom bounds constrain an out-of-bounds size with a record",
			cfg:  params.FlagInputs{params.FlagTypeSize: "512x512"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "816x816"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeSize, params.ChangeDerived, "512x512", "816x816", "custom-size-image limits")},
		},
		{
			name: "custom bounds pass a within-bounds size without a record",
			cfg:  params.FlagInputs{params.FlagTypeSize: "1024x1024"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1024"}, sent: []params.FlagType{params.FlagTypeSize},
		},
		{
			name: "no declared constraint forwards the size raw",
			cfg:  params.FlagInputs{params.FlagTypeSize: "1280x720"}, model: rawSize,
			want: params.Values{params.FlagTypeSize: "1280x720"}, sent: []params.FlagType{params.FlagTypeSize},
		},
		{
			name: "a non-consuming model takes no size and no supersede records",
			cfg:  params.FlagInputs{params.FlagTypeSize: "1024x1024", params.FlagTypeAspect: "7:5"}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeAspect: "4:3"}, sent: []params.FlagType{params.FlagTypeAspect},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeSnapped, "7:5", "4:3", "")},
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
		t.Log("✓ explicit size validates, precedes, supersedes, and drops per the precedence table")
	}
}

// TestAdjustCustomMode verifies invariant #2: Custom-size parameter adjustment.
//
// What is being tested:
// Given custom size bounds, params.Adjust must let resolution override aspect, constrain 512x512 to
// 816x816, retain 1024x1024, and derive the listed sizes from ratios. It must drop malformed
// resolution before using aspect, drop malformed aspect without a size, and return no values for
// absent inputs. Each case must match the exact expected values and ordered records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdjustCustomMode(t *testing.T) {
	cases := []adjustCase{
		{
			name: "WxH resolution constrains into bounds and wins",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "512x512", params.FlagTypeAspect: "16:9"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "816x816"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeDerived, "512x512", "816x816", "custom-size-image limits")},
		},
		{
			name: "within-bounds WxH resolution passes through without a record",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "1024x1024"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1024"}, sent: []params.FlagType{params.FlagTypeSize},
		},
		{
			name: "ratio-only resolution derives at the long edge",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "16:9"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "1536x864"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeDerived, "16:9", "1536x864", "")},
		},
		{
			name: "invalid resolution drops; the aspect still derives",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "garbage", params.FlagTypeAspect: "16:9"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "1536x864"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{
				ch(t, params.FlagTypeResolution, params.ChangeDropped, "garbage", "", ""),
				ch(t, params.FlagTypeAspect, params.ChangeDerived, "16:9", "1536x864", ""),
			},
		},
		{
			name: "aspect alone derives at the long edge",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "16:9"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "1536x864"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDerived, "16:9", "1536x864", "")},
		},
		{
			name: "a steep aspect derives through the repair loop",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "10:1"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "1408x480"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDerived, "10:1", "1408x480", "")},
		},
		{
			name: "portrait aspect derives portrait",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "9:16"}, model: customModel(t),
			want: params.Values{params.FlagTypeSize: "864x1536"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDerived, "9:16", "864x1536", "")},
		},
		{
			name: "unparseable aspect drops with nothing sent",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "garbage"}, model: customModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDropped, "garbage", "", "")},
		},
		{
			name: "neither flag sends nothing",
			cfg:  params.FlagInputs{}, model: customModel(t),
			want: params.Values{},
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
		t.Log("✓ the custom mode interprets, constrains, derives, and drops per the declared bounds")
	}
}

// TestAdjustFixedMode verifies invariant #3: Fixed-size parameter adjustment.
//
// What is being tested:
// Given fixed allowed sizes, params.Adjust must choose the listed dimensions from explicit
// dimensions, ratios, or resolution tiers, using resolution before aspect. It must preserve exact
// members, record derived or snapped choices, and drop malformed inputs before considering any
// valid fallback. Each case must return the exact expected values and ordered records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdjustFixedMode(t *testing.T) {
	cases := []adjustCase{
		{
			name: "trio aspect adjusts to the nearest by ratio distance",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "16:9"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1536x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeSnapped, "16:9", "1536x1024", "")},
		},
		{
			name: "trio 3:2 selects the landscape member",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "3:2"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1536x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDerived, "3:2", "1536x1024", "")},
		},
		{
			name: "ratio-form resolution outranks the aspect",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "1:1", params.FlagTypeResolution: "16:9"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1536x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeSnapped, "16:9", "1536x1024", "")},
		},
		{
			name: "landscape high tier from aspect plus tier label",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "16:9", params.FlagTypeResolution: "1080p"}, model: fixedSizeVideoModel(t),
			want: params.Values{params.FlagTypeSize: "1792x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{
				ch(t, params.FlagTypeAspect, params.ChangeSnapped, "16:9", "1792x1024", ""),
				ch(t, params.FlagTypeResolution, params.ChangeDerived, "1080p", "1792x1024", ""),
			},
		},
		{
			name: "portrait high tier",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "9:16", params.FlagTypeResolution: "1080p"}, model: fixedSizeVideoModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1792"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{
				ch(t, params.FlagTypeAspect, params.ChangeSnapped, "9:16", "1024x1792", ""),
				ch(t, params.FlagTypeResolution, params.ChangeDerived, "1080p", "1024x1792", ""),
			},
		},
		{
			name: "landscape base tier",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "16:9", params.FlagTypeResolution: "720p"}, model: fixedSizeVideoModel(t),
			want: params.Values{params.FlagTypeSize: "1280x720"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{
				ch(t, params.FlagTypeAspect, params.ChangeDerived, "16:9", "1280x720", ""),
				ch(t, params.FlagTypeResolution, params.ChangeDerived, "720p", "1280x720", ""),
			},
		},
		{
			name: "a tier the model does not declare caps to its base",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "16:9", params.FlagTypeResolution: "1080p"}, model: restrictedSizeVideoModel(t),
			want: params.Values{params.FlagTypeSize: "1280x720"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{
				ch(t, params.FlagTypeAspect, params.ChangeDerived, "16:9", "1280x720", ""),
				ch(t, params.FlagTypeResolution, params.ChangeDerived, "1080p", "1280x720", ""),
			},
		},
		{
			name: "an exact in-set WxH resolution passes without a record",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "1280x720"}, model: restrictedSizeVideoModel(t),
			want: params.Values{params.FlagTypeSize: "1280x720"}, sent: []params.FlagType{params.FlagTypeSize},
		},
		{
			name: "a tier label alone defaults to landscape",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "720p"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1024x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeDerived, "720p", "1024x1024", "")},
		},
		{
			name: "an unintelligible resolution drops; the aspect still selects",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "16:9", params.FlagTypeResolution: "garbage"}, model: trioModel(t),
			want: params.Values{params.FlagTypeSize: "1536x1024"}, sent: []params.FlagType{params.FlagTypeSize},
			changes: []params.Adjustment{
				ch(t, params.FlagTypeResolution, params.ChangeDropped, "garbage", "", ""),
				ch(t, params.FlagTypeAspect, params.ChangeSnapped, "16:9", "1536x1024", ""),
			},
		},
		{
			name: "an unparseable aspect alone drops with nothing sent",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "garbage"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDropped, "garbage", "", "")},
		},
		{
			name: "neither flag sends nothing",
			cfg:  params.FlagInputs{}, model: restrictedSizeVideoModel(t),
			want: params.Values{},
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
		t.Log("✓ the fixed-set mode selects by orientation, tier, and ratio distance per the declared set")
	}
}

// TestAdjustSplitMode verifies invariant #4: Separate aspect and resolution adjustment.
//
// What is being tested:
// Given separate aspect and resolution sets, params.Adjust must preserve members with declared
// spelling, snap 7:5 to 4:3, map 900p to 2k and 1024x768 to 1k, and drop malformed values. With
// only 1K allowed it must map 2K to 1K; without sets it must retain both raw inputs. Every case
// must match the expected values and records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdjustSplitMode(t *testing.T) {
	cases := []adjustCase{
		{
			name: "in-set aspect passes without a record",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "16:9"}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeAspect: "16:9"}, sent: []params.FlagType{params.FlagTypeAspect},
		},
		{
			name: "a trimmed in-set aspect passes as the declared spelling",
			cfg:  params.FlagInputs{params.FlagTypeAspect: " 16:9 "}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeAspect: "16:9"}, sent: []params.FlagType{params.FlagTypeAspect},
		},
		{
			name: "a non-ratio pass-through member passes untouched",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "auto"}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeAspect: "auto"}, sent: []params.FlagType{params.FlagTypeAspect},
		},
		{
			name: "a non-ratio pass-through member matches case-insensitively",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "AUTO"}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeAspect: "auto"}, sent: []params.FlagType{params.FlagTypeAspect},
		},
		{
			name: "an out-of-set aspect adjusts to the nearest by ratio distance",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "7:5"}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeAspect: "4:3"}, sent: []params.FlagType{params.FlagTypeAspect},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeSnapped, "7:5", "4:3", "")},
		},
		{
			name: "an unparseable aspect drops with nothing sent",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "garbage"}, model: fixedAspectImageModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeAspect, params.ChangeDropped, "garbage", "", "")},
		},
		{
			name: "an in-set resolution passes without a record",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "1k"}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeResolution: "1k"}, sent: []params.FlagType{params.FlagTypeResolution},
		},
		{
			name: "the moved 900p boundary adjusts to 2k",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "900p"}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeResolution: "2k"}, sent: []params.FlagType{params.FlagTypeResolution},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeSnapped, "900p", "2k", "")},
		},
		{
			name: "a WxH resolution adjusts to the nearest by representative height",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "1024x768"}, model: fixedAspectImageModel(t),
			want: params.Values{params.FlagTypeResolution: "1k"}, sent: []params.FlagType{params.FlagTypeResolution},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeSnapped, "1024x768", "1k", "")},
		},
		{
			name: "an unintelligible resolution drops with nothing sent",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "bogus"}, model: fixedAspectImageModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeDropped, "bogus", "", "")},
		},
		{
			name: "the flash-lite subset adjusts 2K to 1K",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "2K"}, model: singleResolutionModel(t),
			want: params.Values{params.FlagTypeResolution: "1K"}, sent: []params.FlagType{params.FlagTypeResolution},
			changes: []params.Adjustment{ch(t, params.FlagTypeResolution, params.ChangeSnapped, "2K", "1K", "")},
		},
		{
			name: "an in-set resolution matches case-insensitively as declared",
			cfg:  params.FlagInputs{params.FlagTypeResolution: "1k"}, model: singleResolutionModel(t),
			want: params.Values{params.FlagTypeResolution: "1K"}, sent: []params.FlagType{params.FlagTypeResolution},
		},
		{
			name: "raw forward: no declared sets forward both values raw",
			cfg:  params.FlagInputs{params.FlagTypeAspect: "21:9", params.FlagTypeResolution: "512"}, model: rawModel(t),
			want: params.Values{params.FlagTypeAspect: "21:9", params.FlagTypeResolution: "512"}, sent: []params.FlagType{params.FlagTypeAspect, params.FlagTypeResolution},
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
		t.Log("✓ the split mode checks, adjusts, passes, and drops each flag against its own declared set")
	}
}

// TestAdjustGates verifies invariant #5: Parameter type adjustment.
//
// What is being tested:
// Given typed parameter inputs, params.Adjust must normalize declared string members, reject
// nonmembers, snap or bound duration and image counts, and preserve unconstrained finite numbers
// and Booleans. It must drop NaN and infinity and omit undeclared flags. Every case must return the
// exact values and ordered records shown, including allowed-set comments for rejected strings.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdjustGates(t *testing.T) {
	cases := []adjustCase{
		{
			name: "quality member passes lowercased",
			cfg:  params.FlagInputs{params.FlagTypeQuality: "HIGH"}, model: trioModel(t),
			want: params.Values{params.FlagTypeQuality: "high"}, sent: []params.FlagType{params.FlagTypeQuality},
		},
		{
			name: "quality miss rejects naming the set",
			cfg:  params.FlagInputs{params.FlagTypeQuality: "bogus"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeQuality, params.ChangeRejected, "bogus", "", "low|medium|high|auto")},
		},
		{
			name: "quality with no declared set forwards raw",
			cfg:  params.FlagInputs{params.FlagTypeQuality: "ultra"}, model: formatChoiceImageModel(t),
			want: params.Values{params.FlagTypeQuality: "ultra"}, sent: []params.FlagType{params.FlagTypeQuality},
		},
		{
			name: "output format member passes",
			cfg:  params.FlagInputs{params.FlagTypeOutputFormat: "png"}, model: trioModel(t),
			want: params.Values{params.FlagTypeOutputFormat: "png"}, sent: []params.FlagType{params.FlagTypeOutputFormat},
		},
		{
			name: "output format miss rejects naming the set",
			cfg:  params.FlagInputs{params.FlagTypeOutputFormat: "gif"}, model: trioModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeOutputFormat, params.ChangeRejected, "gif", "", "png|jpeg|webp")},
		},
		{
			name: "thinking level in the model's set passes",
			cfg:  params.FlagInputs{params.FlagTypeThinkingLevel: "high"}, model: rawModel(t),
			want: params.Values{params.FlagTypeThinkingLevel: "high"}, sent: []params.FlagType{params.FlagTypeThinkingLevel},
		},
		{
			name: "thinking level outside the model's set rejects, value omitted",
			cfg:  params.FlagInputs{params.FlagTypeThinkingLevel: "bogus"}, model: rawModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeThinkingLevel, params.ChangeRejected, "bogus", "", "low|high")},
		},
		{
			name: "thinking level accepts case-insensitively",
			cfg:  params.FlagInputs{params.FlagTypeThinkingLevel: "HIGH"}, model: rawModel(t),
			want: params.Values{params.FlagTypeThinkingLevel: "high"}, sent: []params.FlagType{params.FlagTypeThinkingLevel},
		},
		{
			name: "thinking level accepts a padded spelling",
			cfg:  params.FlagInputs{params.FlagTypeThinkingLevel: " low "}, model: rawModel(t),
			want: params.Values{params.FlagTypeThinkingLevel: "low"}, sent: []params.FlagType{params.FlagTypeThinkingLevel},
		},
		{
			name: "duration adjusts to the nearest of the declared set",
			cfg:  params.FlagInputs{params.FlagTypeDuration: 5}, model: restrictedSizeVideoModel(t),
			want: params.Values{params.FlagTypeDuration: 4}, sent: []params.FlagType{params.FlagTypeDuration},
			changes: []params.Adjustment{ch(t, params.FlagTypeDuration, params.ChangeSnapped, "5", "4", "")},
		},
		{
			name: "an in-set duration passes without a record",
			cfg:  params.FlagInputs{params.FlagTypeDuration: 8}, model: restrictedSizeVideoModel(t),
			want: params.Values{params.FlagTypeDuration: 8}, sent: []params.FlagType{params.FlagTypeDuration},
		},
		{
			name: "duration over the declared range caps",
			cfg:  params.FlagInputs{params.FlagTypeDuration: 20}, model: rangedDurationVideoModel(t),
			want: params.Values{params.FlagTypeDuration: 15}, sent: []params.FlagType{params.FlagTypeDuration},
			changes: []params.Adjustment{ch(t, params.FlagTypeDuration, params.ChangeCapped, "20", "15", "max 15")},
		},
		{
			name: "duration under the declared range raises",
			cfg:  params.FlagInputs{params.FlagTypeDuration: 0}, model: rangedDurationVideoModel(t),
			want: params.Values{params.FlagTypeDuration: 1}, sent: []params.FlagType{params.FlagTypeDuration},
			changes: []params.Adjustment{ch(t, params.FlagTypeDuration, params.ChangeRaised, "0", "1", "min 1")},
		},
		{
			name: "duration with no declared constraint forwards raw",
			cfg:  params.FlagInputs{params.FlagTypeDuration: 7}, model: openVideoModel(t),
			want: params.Values{params.FlagTypeDuration: 7}, sent: []params.FlagType{params.FlagTypeDuration},
		},
		{
			name: "num-images over the cap caps",
			cfg:  params.FlagInputs{params.FlagTypeImageN: 20}, model: trioModel(t),
			want: params.Values{params.FlagTypeImageN: 10}, sent: []params.FlagType{params.FlagTypeImageN},
			changes: []params.Adjustment{ch(t, params.FlagTypeImageN, params.ChangeCapped, "20", "10", "max 10")},
		},
		{
			name: "num-images below one raises to one",
			cfg:  params.FlagInputs{params.FlagTypeImageN: 0}, model: trioModel(t),
			want: params.Values{params.FlagTypeImageN: 1}, sent: []params.FlagType{params.FlagTypeImageN},
			changes: []params.Adjustment{ch(t, params.FlagTypeImageN, params.ChangeRaised, "0", "1", "min 1")},
		},
		{
			name: "a negative num-images raises to one",
			cfg:  params.FlagInputs{params.FlagTypeImageN: -3}, model: trioModel(t),
			want: params.Values{params.FlagTypeImageN: 1}, sent: []params.FlagType{params.FlagTypeImageN},
			changes: []params.Adjustment{ch(t, params.FlagTypeImageN, params.ChangeRaised, "-3", "1", "min 1")},
		},
		{
			name: "an in-range num-images passes",
			cfg:  params.FlagInputs{params.FlagTypeImageN: 5}, model: trioModel(t),
			want: params.Values{params.FlagTypeImageN: 5}, sent: []params.FlagType{params.FlagTypeImageN},
		},
		{
			name: "a number forwards raw where consumed",
			cfg:  params.FlagInputs{flagTypeNumberFixture: 0.7}, model: rawModel(t),
			want: params.Values{flagTypeNumberFixture: 0.7}, sent: []params.FlagType{flagTypeNumberFixture},
		},
		{
			name: "an explicit zero number forwards as sent",
			cfg:  params.FlagInputs{flagTypeNumberFixture: 0.0}, model: rawModel(t),
			want: params.Values{flagTypeNumberFixture: 0.0}, sent: []params.FlagType{flagTypeNumberFixture},
		},
		{
			name: "a nan number drops with a record, never encoded",
			cfg:  params.FlagInputs{flagTypeNumberFixture: math.NaN()}, model: rawModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, flagTypeNumberFixture, params.ChangeDropped, "NaN", "", "")},
		},
		{
			name: "an inf number drops with a record, never encoded",
			cfg:  params.FlagInputs{flagTypeNumberFixture: math.Inf(1)}, model: rawModel(t),
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, flagTypeNumberFixture, params.ChangeDropped, "+Inf", "", "")},
		},
		{
			name: "include-thoughts forwards where consumed",
			cfg:  params.FlagInputs{params.FlagTypeThoughts: true}, model: rawModel(t),
			want: params.Values{params.FlagTypeThoughts: true}, sent: []params.FlagType{params.FlagTypeThoughts},
		},
		{
			name: "an explicit-false include-thoughts forwards as sent",
			cfg:  params.FlagInputs{params.FlagTypeThoughts: false}, model: rawModel(t),
			want: params.Values{params.FlagTypeThoughts: false}, sent: []params.FlagType{params.FlagTypeThoughts},
		},
		{
			name: "an unconsumed supplied flag is not adjusted and not sent",
			cfg:  params.FlagInputs{params.FlagTypeQuality: "high", params.FlagTypeDuration: 5}, model: rawModel(t),
			want: params.Values{},
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
		t.Log("✓ the uniform checks reject, adjust, constrain, cap, raise, and raw-forward per declared data")
	}
}

// TestAdjustDeclaredRanges verifies invariant #6: Numeric parameter range adjustment.
//
// What is being tested:
// Given bounds zero through one, params.Adjust must cap 2 to 1, raise -0.5 to 0, and retain 0.7.
// Without bounds it must retain 2; with image-count minimum two it must raise 1 to 2. Each
// adjustment must return the exact Capped or Raised record, and unchanged values must return no
// records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdjustDeclaredRanges(t *testing.T) {
	rangedNumber := replaceParamCfg(t, rawModel(t), params.Definition{
		FlagID: flagTypeNumberFixture, ParamID: "number_fixture",
		MinValue: params.GetSetIf(true, 0.0), MaxValue: params.GetSetIf(true, 1.0),
	})
	rangedCount := replaceParamCfg(t, trioModel(t), params.Definition{
		FlagID: params.FlagTypeImageN, ParamID: "n",
		MinValue: params.GetSetIf(true, 2.0), MaxValue: params.GetSetIf(true, 10.0),
	})

	cases := []adjustCase{
		{
			name: "a number over its declared range caps",
			cfg:  params.FlagInputs{flagTypeNumberFixture: 2.0}, model: rangedNumber,
			want: params.Values{flagTypeNumberFixture: 1.0}, sent: []params.FlagType{flagTypeNumberFixture},
			changes: []params.Adjustment{ch(t, flagTypeNumberFixture, params.ChangeCapped, "2", "1", "max 1")},
		},
		{
			name: "a number under its declared range raises",
			cfg:  params.FlagInputs{flagTypeNumberFixture: -0.5}, model: rangedNumber,
			want: params.Values{flagTypeNumberFixture: 0.0}, sent: []params.FlagType{flagTypeNumberFixture},
			changes: []params.Adjustment{ch(t, flagTypeNumberFixture, params.ChangeRaised, "-0.5", "0", "min 0")},
		},
		{
			name: "a number inside its declared range passes untouched",
			cfg:  params.FlagInputs{flagTypeNumberFixture: 0.7}, model: rangedNumber,
			want: params.Values{flagTypeNumberFixture: 0.7}, sent: []params.FlagType{flagTypeNumberFixture},
		},
		{
			name: "the same number with no declared range forwards raw",
			cfg:  params.FlagInputs{flagTypeNumberFixture: 2.0}, model: rawModel(t),
			want: params.Values{flagTypeNumberFixture: 2.0}, sent: []params.FlagType{flagTypeNumberFixture},
		},
		{
			name: "num-images under a declared minimum above one raises to it",
			cfg:  params.FlagInputs{params.FlagTypeImageN: 1}, model: rangedCount,
			want: params.Values{params.FlagTypeImageN: 2}, sent: []params.FlagType{params.FlagTypeImageN},
			changes: []params.Adjustment{ch(t, params.FlagTypeImageN, params.ChangeRaised, "1", "2", "min 2")},
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
		t.Log("✓ a declared numeric range constrains its own flag's supplied value, and only where declared")
	}
}

// TestAdjustNumericMembership verifies invariant #7: Numeric parameter set membership.
//
// What is being tested:
// Given numeric allowed sets, params.Adjust must retain integer 3 and number 0.5 without records.
// It must omit integer 2 and number 0.7 and return the exact Rejected record naming their allowed
// sets.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdjustNumericMembership(t *testing.T) {
	countSet := replaceParamCfg(t, trioModel(t), params.Definition{
		FlagID: params.FlagTypeImageN, ParamID: "n", AllowedValues: []string{"1", "3"},
	})
	numberSet := replaceParamCfg(t, rawModel(t), params.Definition{
		FlagID: flagTypeNumberFixture, ParamID: "number_fixture", AllowedValues: []string{"0.5", "1"},
	})

	cases := []adjustCase{
		{
			name: "an integer member of its declared set passes as sent",
			cfg:  params.FlagInputs{params.FlagTypeImageN: 3}, model: countSet,
			want: params.Values{params.FlagTypeImageN: 3}, sent: []params.FlagType{params.FlagTypeImageN},
		},
		{
			name: "an integer outside its declared set rejects naming the set",
			cfg:  params.FlagInputs{params.FlagTypeImageN: 2}, model: countSet,
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, params.FlagTypeImageN, params.ChangeRejected, "2", "", "1|3")},
		},
		{
			name: "a number member of its declared set passes as sent",
			cfg:  params.FlagInputs{flagTypeNumberFixture: 0.5}, model: numberSet,
			want: params.Values{flagTypeNumberFixture: 0.5}, sent: []params.FlagType{flagTypeNumberFixture},
		},
		{
			name: "a number outside its declared set rejects naming the set",
			cfg:  params.FlagInputs{flagTypeNumberFixture: 0.7}, model: numberSet,
			want:    params.Values{},
			changes: []params.Adjustment{ch(t, flagTypeNumberFixture, params.ChangeRejected, "0.7", "", "0.5|1")},
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
		t.Log("✓ a declared numeric closed set governs its own flag: typed members pass, misses reject naming the set")
	}
}

// TestParamValue verifies invariant #8: Typed parameter reads.
//
// What is being tested:
// Given an absent string parameter, params.Value[string] must return an empty string without an
// error. It must return a stored string unchanged and reject a stored integer with
// ErrParamValueTypeMismatch under ErrParamValue.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestParamValue(t *testing.T) {
	parameterValues := params.Values{params.FlagTypeSize: "1024x1024", params.FlagTypeDuration: 8}

	if absent, err := params.Value[string](parameterValues, params.FlagTypeQuality); err != nil || absent != "" {
		t.Errorf("✗ an absent parameter read as (%q, %v), want the zero value without error", absent, err)
	}

	if size, err := params.Value[string](parameterValues, params.FlagTypeSize); err != nil || size != "1024x1024" {
		t.Errorf("✗ a stored string read as (%q, %v), want the value without error", size, err)
	}

	if _, err := params.Value[string](parameterValues, params.FlagTypeDuration); !errors.Is(err, errs.ErrParamValueTypeMismatch) || !errors.Is(err, errs.ErrParamValue) {
		t.Errorf("✗ a stored value of another type read with %v, want the type mismatch sentinel under its root", err)
	}

	if !t.Failed() {
		t.Log("✓ typed parameter reads distinguish absent, matching, and mismatched stored values")
	}
}

// TestDeclaredSizeSelection verifies invariant #9: Declared size selection.
//
// What is being tested:
// Given declared dimensions, params.Adjust must prefer the nearest short edge and use the requested
// ratio to break a tie, returning the exact listed size. Its first record must identify aspect as
// Derived when the selected ratio is equivalent, or Snapped otherwise.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDeclaredSizeSelection(t *testing.T) {
	dimensionCases := []struct {
		name, ratio, resolution, selected string
		sizes                             []string
	}{
		{"ratio breaks short-edge tie", "16:9", "720p", "1280x720", []string{"1080x720", "1280x720"}},
		{"short edge outranks ratio", "16:9", "720p", "1080x720", []string{"1600x900", "1080x720"}},
		{"declaration breaks exact tie", "16:9", "720p", "1280x720", []string{"1280x720", "1280x720"}},
		{"equivalent square ratio", "2:2", "", "1024x1024", []string{"1024x1024", "1536x1024"}},
	}
	for _, dimensionCase := range dimensionCases {
		t.Run(dimensionCase.name, func(t *testing.T) {
			model := parameterModel{label: "dimensions", definitions: []params.Definition{{FlagID: params.FlagTypeSize, AllowedValues: dimensionCase.sizes}, {FlagID: params.FlagTypeAspect}, {FlagID: params.FlagTypeResolution}}}

			inputs := params.FlagInputs{params.FlagTypeAspect: dimensionCase.ratio}
			if dimensionCase.resolution != "" {
				inputs[params.FlagTypeResolution] = dimensionCase.resolution
			}

			adjusted, changes := adjustParameters(t, inputs, &model)
			if adjusted[params.FlagTypeSize] != dimensionCase.selected {
				t.Errorf("✗ selected %v, want %s", adjusted[params.FlagTypeSize], dimensionCase.selected)
			}

			requestedRatio := map[string]float64{"16:9": 16.0 / 9.0, "2:2": 1}[dimensionCase.ratio]
			width, height, _ := params.ParseDimensions(dimensionCase.selected)

			noticeKind := params.ChangeSnapped
			if requestedRatio == float64(width)/float64(height) {
				noticeKind = params.ChangeDerived
			}

			if len(changes) == 0 || changes[0].FlagID != params.FlagTypeAspect || changes[0].Type != noticeKind {
				t.Errorf("✗ ratio notice = %+v, want %s first", changes, noticeKind)
			}

			if !t.Failed() {
				t.Log("✓ fixed-size selection and ratio notice agree")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ declared inputs alone determine size")
	}
}

// TestResolutionUninterpretedLabels verifies invariant #10: Nonnumerical resolution labels.
//
// What is being tested:
// Given allowed resolution labels HD and FHD, params.Adjust must normalize padded hd to HD without
// a record. Given 100p, it must omit resolution and return one Dropped record.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolutionUninterpretedLabels(t *testing.T) {
	model := parameterModel{label: "named-resolutions", definitions: []params.Definition{{FlagID: params.FlagTypeResolution, AllowedValues: []string{"HD", "FHD"}}}}

	adjusted, changes := adjustParameters(t, params.FlagInputs{params.FlagTypeResolution: " hd "}, &model)
	if adjusted[params.FlagTypeResolution] != "HD" || len(changes) != 0 {
		t.Errorf("✗ exact declared label changed: %v %+v", adjusted, changes)
	}

	adjusted, changes = adjustParameters(t, params.FlagInputs{params.FlagTypeResolution: "100p"}, &model)
	if _, present := adjusted[params.FlagTypeResolution]; present {
		t.Errorf("✗ numerical request inferred an opaque label: %v", adjusted)
	}

	if len(changes) != 1 || changes[0].Type != params.ChangeDropped {
		t.Errorf("✗ expected one dropped notice: %+v", changes)
	}

	if !t.Failed() {
		t.Log("✓ opaque labels require declared membership")
	}
}

// parameterModel supplies the model identity and parameter declarations for adjustment.
//   - label: the model identifier used in notices
//   - definitions: the declared parameter mappings and constraints
type parameterModel struct {
	label       string
	definitions params.Definitions
}

// adjustCase specifies one parameter-adjustment input and its expected result.
//   - name: the table case name
//   - cfg: supplied parameter values
//   - model: the model's declarations
//   - want: expected request values
//   - sent: flags expected in the request
//   - changes: expected adjustment records, in order
type adjustCase struct {
	name    string
	cfg     params.FlagInputs
	model   parameterModel
	want    params.Values
	sent    []params.FlagType
	changes []params.Adjustment
}

// trioModel returns an image-model fixture with three fixed sizes and common generation parameters.
func trioModel(test testing.TB) parameterModel {
	test.Helper()

	return parameterModel{
		label: "trio-image",
		definitions: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution, ParamID: "resolution"},
			{FlagID: params.FlagTypeQuality, ParamID: "quality", AllowedValues: []string{"low", "medium", "high", "auto"}},
			{FlagID: params.FlagTypeImageN, ParamID: "n", MaxValue: params.GetSetIf(true, 10.0)},
			{FlagID: params.FlagTypeOutputFormat, ParamID: "output_format", AllowedValues: []string{"png", "jpeg", "webp"}},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 16},
			{FlagID: params.FlagTypeSize, ParamID: "size", AllowedValues: []string{"1024x1024", "1024x1536", "1536x1024"}},
		},
	}
}

// replaceParamCfg replaces or appends a parameter definition in the fixture and returns it.
// Replacement may update the supplied slice backing the fixture.
func replaceParamCfg(test testing.TB, model parameterModel, paramCfg params.Definition) parameterModel {
	test.Helper()

	for i := range model.definitions {
		if model.definitions[i].FlagID == paramCfg.FlagID {
			model.definitions[i] = paramCfg

			return model
		}
	}

	model.definitions = append(model.definitions, paramCfg)

	return model
}

// customModel returns the image-model fixture configured for custom sizes.
func customModel(test testing.TB) parameterModel {
	test.Helper()
	b := freeFormBounds(test)
	m := trioModel(test)
	m.label = "custom-size-image"

	return replaceParamCfg(test, m, params.Definition{FlagID: params.FlagTypeSize, ParamID: "size", CustomSize: &b})
}

// fixedSizeVideoModel returns the video-model fixture with a four-member size set and a
// three-member duration set.
func fixedSizeVideoModel(test testing.TB) parameterModel {
	test.Helper()

	return parameterModel{
		label: "fixed-size-video",
		definitions: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution, ParamID: "resolution"},
			{FlagID: params.FlagTypeDuration, ParamID: "seconds", AllowedValues: []string{"4", "8", "12"}},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 1},
			{FlagID: params.FlagTypeSize, ParamID: "size", AllowedValues: []string{"1280x720", "720x1280", "1792x1024", "1024x1792"}},
		},
	}
}

// restrictedSizeVideoModel returns the fixed-size video-model fixture narrowed to two sizes.
func restrictedSizeVideoModel(test testing.TB) parameterModel {
	test.Helper()
	m := fixedSizeVideoModel(test)
	m.label = "restricted-size-video"

	return replaceParamCfg(test, m, params.Definition{
		FlagID:        params.FlagTypeSize,
		ParamID:       "size",
		AllowedValues: []string{"1280x720", "720x1280"},
	})
}

// fixedAspectImageModel returns the image-model fixture with fixed aspect and resolution sets.
func fixedAspectImageModel(test testing.TB) parameterModel {
	test.Helper()

	return parameterModel{
		label: "fixed-aspect-image",
		definitions: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio", AllowedValues: []string{
				"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20", "auto",
			}},
			{FlagID: params.FlagTypeResolution, ParamID: "resolution", AllowedValues: []string{"1k", "2k"}},
			{FlagID: params.FlagTypeImageN, ParamID: "n", MaxValue: params.GetSetIf(true, 10.0)},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 5},
		},
	}
}

// singleResolutionModel returns the raw image-model fixture narrowed to one resolution.
func singleResolutionModel(test testing.TB) parameterModel {
	test.Helper()
	m := rawModel(test)
	m.label = "single-resolution-image"

	return replaceParamCfg(test, m, params.Definition{
		FlagID:        params.FlagTypeResolution,
		ParamID:       "image_size",
		AllowedValues: []string{"1K"},
	})
}

// flagTypeNumberFixture is a synthetic number-typed parameter the fixture models declare.
const flagTypeNumberFixture = "number-fixture"

// rawModel returns the image-model fixture whose aspect and resolution values are passed through
// without a fixed allowed-value set.
func rawModel(test testing.TB) parameterModel {
	test.Helper()

	return parameterModel{
		label: "raw-image",
		definitions: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution, ParamID: "image_size"},
			{FlagID: flagTypeNumberFixture, ParamID: "number_fixture"},
			{FlagID: params.FlagTypeThinkingLevel, ParamID: "thinking_level", AllowedValues: []string{"low", "high"}},
			{FlagID: params.FlagTypeThoughts},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 14},
		},
	}
}

// openVideoModel returns the video-model fixture with no declared constraint on any parameter.
func openVideoModel(test testing.TB) parameterModel {
	test.Helper()

	return parameterModel{
		label: "open-video",
		definitions: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution, ParamID: "resolution"},
			{FlagID: params.FlagTypeDuration, ParamID: "duration"},
			{FlagID: params.FlagTypeInputMedia},
		},
	}
}

// formatChoiceImageModel returns the image-model fixture declaring an output-format set.
func formatChoiceImageModel(test testing.TB) parameterModel {
	test.Helper()

	return parameterModel{
		label: "format-choice-image",
		definitions: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution, ParamID: "resolution"},
			{FlagID: params.FlagTypeQuality, ParamID: "quality"},
			{FlagID: params.FlagTypeImageN, ParamID: "n", MaxValue: params.GetSetIf(true, 10.0)},
			{FlagID: params.FlagTypeOutputFormat, ParamID: "output_format", AllowedValues: []string{"png", "jpeg", "webp"}},
			{FlagID: params.FlagTypeInputMedia},
		},
	}
}

// runAdjust asserts one adjustCase: the adjusted values, the exact sent-flag set (the map's key
// set), and the exact record slice.
func runAdjust(t *testing.T, c adjustCase) {
	t.Helper()

	gParams, paramChanges := adjustParameters(t, c.cfg, &c.model)
	if c.want == nil {
		c.want = params.Values{}
	}

	if !reflect.DeepEqual(gParams, c.want) {
		t.Errorf("✗ %s: GenParams = %+v, want %+v", c.name, gParams, c.want)
	}

	checkSentFlags(t, c.name, gParams, c.sent)

	if len(paramChanges) == 0 && len(c.changes) == 0 {
		return
	}

	if !reflect.DeepEqual(paramChanges, c.changes) {
		t.Errorf("✗ %s: changes = %+v, want %+v", c.name, paramChanges, c.changes)
	}
}

// checkSentFlags checks whether each known fixture flag appears in the adjusted values as expected.
func checkSentFlags(t *testing.T, name string, gp params.Values, sent []params.FlagType) {
	t.Helper()

	all := []params.FlagType{
		params.FlagTypeAspect, params.FlagTypeResolution, params.FlagTypeSize, params.FlagTypeQuality,
		params.FlagTypeOutputFormat, params.FlagTypeThinkingLevel, flagTypeNumberFixture, params.FlagTypeDuration,
		params.FlagTypeImageN, params.FlagTypeThoughts, params.FlagTypeInputMedia,
	}

	wantSent := map[params.FlagType]bool{}
	for _, p := range sent {
		wantSent[p] = true
	}

	for _, p := range all {
		if _, ok := gp[p]; ok != wantSent[p] {
			t.Errorf("✗ %s: sent(%s) = %v, want %v", name, p, ok, wantSent[p])
		}
	}
}

// ch constructs one expected parameter-change record for an adjustment test.
func ch(test testing.TB, p params.FlagType, k params.Change, req, used, detail string) params.Adjustment {
	test.Helper()

	return params.Adjustment{FlagID: p, Type: k, InputVal: req, WireVal: used, Comment: detail}
}

// rangedDurationVideoModel returns the video-model fixture with a duration range and fixed aspect
// and resolution sets.
func rangedDurationVideoModel(test testing.TB) parameterModel {
	test.Helper()

	return parameterModel{
		label: "ranged-duration-video",
		definitions: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio", AllowedValues: []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"}},
			{FlagID: params.FlagTypeResolution, ParamID: "resolution", AllowedValues: []string{"480p", "720p"}},
			{FlagID: params.FlagTypeDuration, ParamID: "duration", MinValue: params.GetSetIf(true, 1.0), MaxValue: params.GetSetIf(true, 15.0)},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 1},
		},
	}
}

// adjustParameters applies the fixture declarations and rejects unexpected adjustment failures.
func adjustParameters(test testing.TB, inputs params.FlagInputs, model *parameterModel) (params.Values, []params.Adjustment) {
	test.Helper()

	values, changes, err := params.Adjust(inputs, model.definitions, model.label)
	if err != nil {
		test.Errorf("✗ unexpected adjustment failure: %v", err)
	}

	return values, changes
}

// freeFormBounds supplies the configured bounds used by the adjustment cases.
func freeFormBounds(test testing.TB) params.SizeBounds {
	test.Helper()

	return params.SizeBounds{MaxRatio: 3, MaxEdge: 3840, MinPx: 655360, MaxPx: 8294400, EdgeIncrem: 16, LongEdge: 1536}
}
