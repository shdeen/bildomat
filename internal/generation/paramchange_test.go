package generation

// Invariants tested:
//  1. Size-only aspect notice: Given an undeclared aspect input, AdjustGeneration must return one
//     Ignored aspect record and derive no size.
//  2. Undeclared sizing inputs: Given aspect and resolution inputs that the model does not declare,
//     AdjustGeneration must succeed, omit the size value, and return exactly two adjustment
//     records.
//  3. Input media limits: Given a model without media support, capInputMedia must return no images
//     or records.
//  4. Generation adjustment composition: Given an unsupported quality input, four images with a
//     limit of three, and duration five with allowed values four and eight, AdjustGeneration must
//     return only duration four in Params.
//  5. Ignored parameters from model configuration: Given declared aspect and quality inputs
//     alongside four undeclared inputs, ignoredParamRecords must include each undeclared flag and
//     exclude aspect and quality.
//  6. No ignored records for absent params: Given an empty FlagInputs map, ignoredParamRecords must
//     return no records.
//  7. Inapplicable parameter records: Given undeclared numeric, thinking-level, and thoughts
//     inputs, ignoredParamRecords must include an Ignored record for each with the expected model
//     comment.
//  8. Exhaustive ignored parameter records: Given every supplied flag and a model that consumes
//     none, ignoredParamRecords must return exactly one Ignored record per flag in sorted order
//     with the expected model comment.
//  9. False include-thoughts value: Given an explicit false thoughts value and a model that does
//     not consume it, ignoredParamRecords must return no records.
//  10. Input-media and size ignored records: Given an input path and size, ignoredParamRecords must
//      include both flags when the model does not consume them, with Ignored types and the expected
//      model comment.

import (
	"fmt"
	"slices"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestSizeOnlyModelAspectNotice verifies invariant #1: Size-only aspect notice.
//
// What is being tested:
// Given an undeclared aspect input, AdjustGeneration must return one Ignored aspect record and
// derive no size. It must retain an explicit size and return only one notice each for undeclared
// aspect and resolution inputs. When the model declares aspect and custom size bounds, it must
// derive a size without marking aspect ignored.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSizeOnlyModelAspectNotice(t *testing.T) {
	sizeOnlyModel := catalog.Model{
		ID: "size-only-image", Media: media.Image,
		Params: []params.Definition{{FlagID: params.FlagTypeSize, CustomSize: &params.SizeBounds{MinEdge: 64, MaxPx: 4194304}}},
	}

	sizeOnlyPreparation, adjustmentErr := AdjustGeneration(&sizeOnlyModel, params.FlagInputs{params.FlagTypeAspect: "16:9"}, nil)
	parameterValues, paramChanges := sizeOnlyPreparation.Params, sizeOnlyPreparation.Changes

	if adjustmentErr != nil {
		t.Errorf("✗ unexpected adjustment failure: %v", adjustmentErr)
	}

	if adjustedSize, ok := parameterValues[params.FlagTypeSize]; ok {
		t.Errorf("✗ a size was derived from an aspect ratio the model does not declare: %q", adjustedSize)
	}

	aspectRecords := aspectChangeRecords(t, paramChanges)
	if len(aspectRecords) != 1 || aspectRecords[0].Type != params.ChangeIgnored {
		t.Errorf("✗ aspect records = %+v, want exactly one Ignored record", aspectRecords)
	}

	explicitPreparation, explicitErr := AdjustGeneration(&sizeOnlyModel, params.FlagInputs{
		params.FlagTypeSize: "1024x1024", params.FlagTypeAspect: "16:9", params.FlagTypeResolution: "2K",
	}, nil)
	if explicitErr != nil || explicitPreparation.Params[params.FlagTypeSize] != "1024x1024" {
		t.Errorf("✗ valid explicit size failed: %+v, %v", explicitPreparation, explicitErr)
	}

	if len(explicitPreparation.Changes) != 2 || explicitPreparation.Changes[0].FlagID != params.FlagTypeAspect || explicitPreparation.Changes[1].FlagID != params.FlagTypeResolution {
		t.Errorf("✗ explicit size duplicated unsupported sizing notices: %+v", explicitPreparation.Changes)
	}

	bounds := params.SizeBounds{MaxRatio: 3, MaxEdge: 3840, MinPx: 655360, MaxPx: 8294400, EdgeIncrem: 16, LongEdge: 1536}
	aspectDerivingModel := catalog.Model{
		ID: "aspect-deriving-image", Media: media.Image,
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect},
			{FlagID: params.FlagTypeResolution},
			{FlagID: params.FlagTypeSize, CustomSize: &bounds},
		},
	}

	aspectPreparation, adjustmentErr := AdjustGeneration(&aspectDerivingModel, params.FlagInputs{params.FlagTypeAspect: "16:9"}, nil)
	parameterValues, paramChanges = aspectPreparation.Params, aspectPreparation.Changes

	if adjustmentErr != nil {
		t.Errorf("✗ unexpected adjustment failure: %v", adjustmentErr)
	}

	if _, ok := parameterValues[params.FlagTypeSize]; !ok {
		t.Errorf("✗ no size was derived on the model declaring the aspect flag: %+v", paramChanges)
	}

	for _, record := range aspectChangeRecords(t, paramChanges) {
		if record.Type == params.ChangeIgnored {
			t.Errorf("✗ the declared aspect flag was recorded as ignored: %+v", record)
		}
	}

	if !t.Failed() {
		t.Log("✓ a size-only model records one ignored aspect notice, and an aspect-declaring model still derives its size")
	}
}

// TestUndeclaredSizingInputs verifies invariant #2: Undeclared sizing inputs.
//
// What is being tested:
// Given aspect and resolution inputs that the model does not declare, AdjustGeneration must
// succeed, omit the size value, and return exactly two adjustment records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestUndeclaredSizingInputs(t *testing.T) {
	model := catalog.Model{ID: "size-only", Params: []params.Definition{{FlagID: params.FlagTypeSize, AllowedValues: []string{"1024x1024"}}}}

	preparedGeneration, adjustmentErr := AdjustGeneration(&model, params.FlagInputs{params.FlagTypeAspect: "1:1", params.FlagTypeResolution: "720p"}, nil)
	adjusted, changes := preparedGeneration.Params, preparedGeneration.Changes

	if adjustmentErr != nil {
		t.Errorf("✗ unexpected adjustment error: %v", adjustmentErr)
	}

	if _, present := adjusted[params.FlagTypeSize]; present {
		t.Errorf("✗ undeclared inputs manufactured a size: %v", adjusted)
	}

	if len(changes) != 2 {
		t.Errorf("✗ expected only the two not-consumed notices: %+v", changes)
	}

	if !t.Failed() {
		t.Log("✓ undeclared sizing inputs retain their notices without deriving size")
	}
}

// TestCapInputMediaStates verifies invariant #3: Input media limits.
//
// What is being tested:
// Given a model without media support, capInputMedia must return no images or records. With a limit
// of three, it must preserve the first three of four images and return the exact Capped record.
// With no maximum, it must keep all seven inputs and return no record.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestCapInputMediaStates(t *testing.T) {
	t.Run("no entry keeps none", func(t *testing.T) {
		model := catalog.Model{ID: "m-plain", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}}

		keptImages, records := capInputMedia(capImgs(t, 2), &model)
		if len(keptImages) != 0 || len(records) != 0 {
			t.Errorf("✗ an unsupported-reference model kept %d image(s) with %d record(s), want none and none", len(keptImages), len(records))
		}

		if !t.Failed() {
			t.Log("✓ no entry keeps none")
		}
	})
	t.Run("a declared maximum keeps the first N with the exact record", func(t *testing.T) {
		model := catalog.Model{ID: "m-cap", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeInputMedia, MaxMultiple: 3}}}
		supplied := capImgs(t, 4)

		keptImages, records := capInputMedia(supplied, &model)
		if len(keptImages) != 3 {
			t.Fatalf("💣 kept %d image(s), want the first 3", len(keptImages))
		}

		for i := range 3 {
			if string(keptImages[i].Bytes) != string(supplied[i].Bytes) || keptImages[i].Filepath != supplied[i].Filepath {
				t.Errorf("✗ kept image %d differs from the first-N set: %+v", i, keptImages[i])
			}
		}

		want := params.Adjustment{FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped, InputVal: "4", WireVal: "3", Comment: "max 3"}
		if len(records) != 1 || records[0] != want {
			t.Errorf("✗ cap records = %+v, want exactly %+v", records, want)
		}

		if !t.Failed() {
			t.Log("✓ a declared maximum keeps the first N with the exact record")
		}
	})
	t.Run("no declared maximum keeps everything", func(t *testing.T) {
		model := catalog.Model{ID: "m-open", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeInputMedia}}}

		keptImages, records := capInputMedia(capImgs(t, 7), &model)
		if len(keptImages) != 7 || len(records) != 0 {
			t.Errorf("✗ an unlimited-reference model kept %d image(s) with %d record(s), want all 7 and none", len(keptImages), len(records))
		}

		if !t.Failed() {
			t.Log("✓ no declared maximum keeps everything")
		}
	})

	if !t.Failed() {
		t.Log("✓ the cap application keeps none, the first N with the exact Capped record, or everything, per the declared maximum")
	}
}

// TestAdjustGenerationComposition verifies invariant #4: Generation adjustment composition.
//
// What is being tested:
// Given an unsupported quality input, four images with a limit of three, and duration five with
// allowed values four and eight, AdjustGeneration must return only duration four in Params. It must
// return exactly the expected Ignored quality, Capped media, and Snapped duration records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAdjustGenerationComposition(t *testing.T) {
	model := catalog.Model{
		ID: "m-vid", Media: media.Video,
		Params: []params.Definition{
			{FlagID: params.FlagTypeDuration, AllowedValues: []string{"4", "8"}},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 3},
		},
	}
	inputs := params.FlagInputs{params.FlagTypeDuration: 5, params.FlagTypeQuality: "high"}

	preparedGeneration, adjustmentErr := AdjustGeneration(&model, inputs, capImgs(t, 4))
	gp, records := preparedGeneration.Params, preparedGeneration.Changes

	if adjustmentErr != nil {
		t.Errorf("✗ generation adjustment failed: %v", adjustmentErr)
	}

	if len(gp) != 1 || gp[params.FlagTypeDuration] != 4 {
		t.Errorf("✗ adjusted values = %+v, want the snapped duration alone", gp)
	}

	wantRecords := []params.Adjustment{
		{FlagID: params.FlagTypeQuality, Type: params.ChangeIgnored, Comment: fmt.Sprintf(ReasonNotConsumed, "m-vid")},
		{FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped, InputVal: "4", WireVal: "3", Comment: "max 3"},
		{FlagID: params.FlagTypeDuration, Type: params.ChangeSnapped, InputVal: "5", WireVal: "4"},
	}
	if len(records) != len(wantRecords) {
		t.Errorf("✗ %d record(s) %+v, want %d", len(records), records, len(wantRecords))
	}

	for _, want := range wantRecords {
		found := false

		for _, record := range records {
			if record == want {
				found = true
			}
		}

		if !found {
			t.Errorf("✗ the composed record set is missing %+v: %+v", want, records)
		}
	}

	if !t.Failed() {
		t.Log("✓ the one shared call returns the adjusted values with the ignored, cap, and adjustment records together")
	}
}

// TestIgnoredRecordsFromConfig verifies invariant #5: Ignored parameters from model configuration.
//
// What is being tested:
// Given declared aspect and quality inputs alongside four undeclared inputs, ignoredParamRecords
// must include each undeclared flag and exclude aspect and quality. Each returned record must have
// type Ignored and the expected comment naming the model.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestIgnoredRecordsFromConfig(t *testing.T) {
	model := catalog.Model{
		ID: "m-img", Media: media.Image,
		Params: []params.Definition{{FlagID: params.FlagTypeAspect}, {FlagID: params.FlagTypeResolution}, {FlagID: params.FlagTypeQuality}},
	}
	cfg := params.FlagInputs{
		params.FlagTypeAspect:        "16:9",
		params.FlagTypeQuality:       "high",
		flagTypeNumberFixture:        0.7,
		params.FlagTypeThinkingLevel: "high",
		params.FlagTypeThoughts:      true,
		params.FlagTypeDuration:      8,
	}

	ignored := ignoredParamsOf(t, ignoredParamRecords(cfg, &model), "m-img")
	for _, want := range []params.FlagType{flagTypeNumberFixture, params.FlagTypeThinkingLevel, params.FlagTypeThoughts, params.FlagTypeDuration} {
		if !slices.Contains(ignored, want) {
			t.Errorf("✗ ignored set is missing %s: %v", want, ignored)
		}
	}

	for _, consumed := range []params.FlagType{params.FlagTypeAspect, params.FlagTypeQuality} {
		if slices.Contains(ignored, consumed) {
			t.Errorf("✗ declared-consumed %s reported as ignored: %v", consumed, ignored)
		}
	}

	if !t.Failed() {
		t.Log("✓ supplied-but-unconsumed params derive from the model's declared param set; consumed params are excluded")
	}
}

// TestIgnoredRecordsEmptyWhenUnset verifies invariant #6: No ignored records for absent params.
//
// What is being tested:
// Given an empty FlagInputs map, ignoredParamRecords must return no records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestIgnoredRecordsEmptyWhenUnset(t *testing.T) {
	model := catalog.Model{ID: "m", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}}
	if got := ignoredParamRecords(params.FlagInputs{}, &model); len(got) != 0 {
		t.Errorf("✗ nothing was supplied yet params were reported ignored: %v", got)
	}

	if !t.Failed() {
		t.Log("✓ no supplied params, no ignored set")
	}
}

// TestIgnoredRecordsInapplicable verifies invariant #7: Inapplicable parameter records.
//
// What is being tested:
// Given undeclared numeric, thinking-level, and thoughts inputs, ignoredParamRecords must include
// an Ignored record for each with the expected model comment. Given no supplied inputs, it must
// return no records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestIgnoredRecordsInapplicable(t *testing.T) {
	supplied := params.FlagInputs{
		flagTypeNumberFixture:        0.7,
		params.FlagTypeThinkingLevel: "high",
		params.FlagTypeThoughts:      true,
	}
	model := catalog.Model{
		ID: "fixture-image", Media: media.Image,
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect},
			{FlagID: params.FlagTypeResolution},
			{FlagID: params.FlagTypeQuality},
			{FlagID: params.FlagTypeImageN},
			{FlagID: params.FlagTypeOutputFormat},
		},
	}

	ignored := ignoredParamsOf(t, ignoredParamRecords(supplied, &model), "fixture-image")
	for _, want := range []params.FlagType{flagTypeNumberFixture, params.FlagTypeThinkingLevel, params.FlagTypeThoughts} {
		if !slices.Contains(ignored, want) {
			t.Errorf("✗ ignored set is missing %s: %v", want, ignored)
		}
	}

	if got := ignoredParamRecords(params.FlagInputs{}, &model); len(got) != 0 {
		t.Errorf("✗ unset flags should produce an empty ignored set, got %v", got)
	}

	if !t.Failed() {
		t.Log("✓ the inapplicable-flag set derives from declared params; unset flags stay out of it")
	}
}

// TestIgnoredRecordsExhaustive verifies invariant #8: Exhaustive ignored parameter records.
//
// What is being tested:
// Given every supplied flag and a model that consumes none, ignoredParamRecords must return exactly
// one Ignored record per flag in sorted order with the expected model comment. A model that
// consumes every flag must produce no records. Supplying each flag alone must produce only that
// flag's record.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestIgnoredRecordsExhaustive(t *testing.T) {
	none := catalog.Model{ID: "m-none", Media: media.Image}
	if got := ignoredParamsOf(t, ignoredParamRecords(supplyAll(t), &none), "m-none"); !slices.Equal(got, allParams(t)) {
		t.Errorf("✗ ignored on a nothing-consuming model = %v, want exactly %v", got, allParams(t))
	}

	allCfgs := make([]params.Definition, 0, len(allParams(t)))
	for _, param := range allParams(t) {
		allCfgs = append(allCfgs, params.Definition{FlagID: param})
	}

	all := catalog.Model{ID: "m-all", Media: media.Image, Params: allCfgs}
	if got := ignoredParamRecords(supplyAll(t), &all); len(got) != 0 {
		t.Errorf("✗ ignored on an everything-consuming model = %v, want empty", got)
	}

	// Each parameter alone: supplied on a nothing-consuming model it is exactly the one ignored
	// entry, so every supplied flag's wiring is individually proven.
	singles := []struct {
		param params.FlagType
		cfg   params.FlagInputs
	}{
		{params.FlagTypeAspect, params.FlagInputs{params.FlagTypeAspect: "16:9"}},
		{params.FlagTypeResolution, params.FlagInputs{params.FlagTypeResolution: "1080p"}},
		{params.FlagTypeSize, params.FlagInputs{params.FlagTypeSize: "1024x1024"}},
		{params.FlagTypeQuality, params.FlagInputs{params.FlagTypeQuality: "high"}},
		{params.FlagTypeThinkingLevel, params.FlagInputs{params.FlagTypeThinkingLevel: "low"}},
		{params.FlagTypeThoughts, params.FlagInputs{params.FlagTypeThoughts: true}},
		{params.FlagTypeDuration, params.FlagInputs{params.FlagTypeDuration: 8}},
		{params.FlagTypeImageN, params.FlagInputs{params.FlagTypeImageN: 2}},
		{params.FlagTypeOutputFormat, params.FlagInputs{params.FlagTypeOutputFormat: "png"}},
		{params.FlagTypeInputMedia, params.FlagInputs{params.FlagTypeInputMedia: []string{"a.png"}}},
	}
	for _, s := range singles {
		if got := ignoredParamsOf(t, ignoredParamRecords(s.cfg, &none), "m-none"); !slices.Equal(got, []params.FlagType{s.param}) {
			t.Errorf("✗ ignored with only %s supplied = %v, want exactly [%s]", s.param, got, s.param)
		}
	}

	if !t.Failed() {
		t.Log("✓ every supplied flag reports when unconsumed, exactly once, deterministically ordered, and never when consumed")
	}
}

// TestIgnoredRecordsThoughtsFalse verifies invariant #9: False include-thoughts value.
//
// What is being tested:
// Given an explicit false thoughts value and a model that does not consume it, ignoredParamRecords
// must return no records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestIgnoredRecordsThoughtsFalse(t *testing.T) {
	none := catalog.Model{ID: "m-none", Media: media.Image}
	if got := ignoredParamRecords(params.FlagInputs{params.FlagTypeThoughts: false}, &none); len(got) != 0 {
		t.Errorf("✗ an explicit false include-thoughts reported ignored: %v", got)
	}

	if !t.Failed() {
		t.Log("✓ include-thoughts counts as supplied only when true")
	}
}

// TestIgnoredRecordsNewMembers verifies invariant #10: Input-media and size ignored records.
//
// What is being tested:
// Given an input path and size, ignoredParamRecords must include both flags when the model does not
// consume them, with Ignored types and the expected model comment. It must return no records when
// the model consumes both or when neither input was supplied.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestIgnoredRecordsNewMembers(t *testing.T) {
	nonConsuming := catalog.Model{ID: "m-plain", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}}
	cfg := params.FlagInputs{
		params.FlagTypeInputMedia: []string{"a.png"},
		params.FlagTypeSize:       "1024x1024",
	}

	ignored := ignoredParamsOf(t, ignoredParamRecords(cfg, &nonConsuming), "m-plain")
	for _, want := range []params.FlagType{params.FlagTypeInputMedia, params.FlagTypeSize} {
		if !slices.Contains(ignored, want) {
			t.Errorf("✗ ignored set is missing %s: %v", want, ignored)
		}
	}

	consuming := catalog.Model{ID: "m-refs", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeInputMedia}, {FlagID: params.FlagTypeSize}}}
	if got := ignoredParamRecords(cfg, &consuming); len(got) != 0 {
		t.Errorf("✗ consumed input-media/size reported as ignored: %v", got)
	}

	if got := ignoredParamRecords(params.FlagInputs{}, &nonConsuming); len(got) != 0 {
		t.Errorf("✗ no inputs and no size supplied, yet the ignored set is %v", got)
	}

	if !t.Failed() {
		t.Log("✓ input-media counts as supplied only when any path was given; size participates; consumed params never report")
	}
}

// aspectChangeRecords returns the change records that concern the aspect-ratio flag.
func aspectChangeRecords(test testing.TB, paramChanges []params.Adjustment) []params.Adjustment {
	test.Helper()

	var aspectRecords []params.Adjustment

	for _, record := range paramChanges {
		if record.FlagID == params.FlagTypeAspect {
			aspectRecords = append(aspectRecords, record)
		}
	}

	return aspectRecords
}

// flagTypeNumberFixture is the numeric parameter used by the generator fixtures.
const flagTypeNumberFixture = "number-fixture"

// ignoredParamsOf extracts the ignored parameter set from the records, asserting every record
// carries the Ignored type and the not-consumed comment naming the model.
func ignoredParamsOf(t *testing.T, records []params.Adjustment, modelID string) []params.FlagType {
	t.Helper()

	parameterValues := make([]params.FlagType, 0, len(records))
	for _, record := range records {
		if record.Type != params.ChangeIgnored || record.Comment != fmt.Sprintf(ReasonNotConsumed, modelID) {
			t.Errorf("✗ malformed ignored record: %+v", record)
		}

		parameterValues = append(parameterValues, record.FlagID)
	}

	return parameterValues
}

// allParams lists the typed parameter constants in flag-name order.
func allParams(test testing.TB) []params.FlagType {
	test.Helper()

	parameterValues := []params.FlagType{
		params.FlagTypeAspect, params.FlagTypeResolution, params.FlagTypeSize, params.FlagTypeQuality, params.FlagTypeThinkingLevel,
		params.FlagTypeThoughts, params.FlagTypeDuration, params.FlagTypeImageN, params.FlagTypeOutputFormat, params.FlagTypeInputMedia,
	}
	slices.Sort(parameterValues)

	return parameterValues
}

// supplyAll supplies a value for each typed parameter constant.
func supplyAll(test testing.TB) params.FlagInputs {
	test.Helper()

	return params.FlagInputs{
		params.FlagTypeAspect:        "16:9",
		params.FlagTypeResolution:    "1080p",
		params.FlagTypeSize:          "1024x1024",
		params.FlagTypeQuality:       "high",
		params.FlagTypeThinkingLevel: "low",
		params.FlagTypeThoughts:      true,
		params.FlagTypeDuration:      8,
		params.FlagTypeImageN:        2,
		params.FlagTypeOutputFormat:  "png",
		params.FlagTypeInputMedia:    []string{"a.png"},
	}
}

// capImgs creates n fixture input images with distinct bytes and paths.
func capImgs(test testing.TB, n int) []media.Input {
	test.Helper()

	imgs := make([]media.Input, 0, n)
	for i := range n {
		imgs = append(imgs, media.Input{
			Bytes:    []byte(fmt.Sprintf("IMG-%d", i)),
			MIME:     "image/png",
			Filepath: fmt.Sprintf("/tmp/ref-%d.png", i),
		})
	}

	return imgs
}
