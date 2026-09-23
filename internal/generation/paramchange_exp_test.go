package generation

// Invariants tested:
//  1. Input media count limits: Given four inputs and a limit of three, limitMedia must retain the
//     first three paths and return the exact Capped record for four to three.
//  2. Selected media ownership: With media limits of zero, one, and two, capInputMedia must retain
//     an input.
//  3. Completed adjustments on failure: Given impossible size bounds and excess media,
//     AdjustGeneration must return an error and exactly two completed records: the ignored aspect
//     input followed by the media cap.
//  4. Retained source arguments: For arbitrary source strings and counts, SelectSources must return
//     no sources when the model lacks media support.

import (
	"slices"
	"strconv"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestCapImgsOutcomes verifies invariant #1: Input media count limits.
//
// What is being tested:
// Given four inputs and a limit of three, limitMedia must retain the first three paths and return
// the exact Capped record for four to three. An unlimited limit or two inputs below the limit must
// retain every input and return no record.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCapImgsOutcomes(t *testing.T) {
	in := make([]media.Input, 4)
	for i := range in {
		in[i] = media.Input{Bytes: []byte{byte(i)}, Filepath: strconv.Itoa(i)}
	}

	keptImages, capRecord := limitMedia(in, 3)
	if len(keptImages) != 3 {
		t.Errorf("✗ capInputMedias(4, cap 3) kept %d, want the first 3", len(keptImages))
	}

	for i := range keptImages {
		if keptImages[i].Filepath != in[i].Filepath {
			t.Errorf("✗ keptImages[%d] = %q, want the first-N member %q", i, keptImages[i].Filepath, in[i].Filepath)
		}
	}

	want := params.Adjustment{FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped, InputVal: "4", WireVal: "3", Comment: "max 3"}
	if capRecord == nil || *capRecord != want {
		t.Errorf("✗ capInputMedias record = %+v, want the exact Capped record %+v", capRecord, want)
	}

	if keptImages, capRecord := limitMedia(in, 0); len(keptImages) != 4 || capRecord != nil {
		t.Errorf("✗ capInputMedias(unlimited) = (%d, %+v), want all 4 with no record", len(keptImages), capRecord)
	}

	if keptImages, capRecord := limitMedia(in[:2], 3); len(keptImages) != 2 || capRecord != nil {
		t.Errorf("✗ capInputMedias(under cap) = (%d, %+v), want all 2 with no record", len(keptImages), capRecord)
	}

	if !t.Failed() {
		t.Log("✓ the reference-cap outcomes hold: first-N kept, exact Capped record over cap, no record otherwise")
	}
}

// TestSelectedMediaOwnership verifies invariant #2: Selected media ownership.
//
// What is being tested:
// With media limits of zero, one, and two, capInputMedia must retain an input. Replacing its path
// and timestamp pointer must leave the caller's path, timestamp pointer, and timestamp value
// unchanged.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestSelectedMediaOwnership(t *testing.T) {
	for _, limit := range []int{0, 1, 2} {
		timestamp := 0.0
		callerMedia := []media.Input{{Filepath: "first.png", Time: &timestamp}, {Filepath: "second.png"}}
		model := &catalog.Model{Params: params.Definitions{{FlagID: params.FlagTypeInputMedia, MaxMultiple: limit}}}

		retained, _ := capInputMedia(callerMedia, model)
		if len(retained) == 0 {
			t.Errorf("✗ limit %d discarded every input", limit)

			continue
		}

		retained[0].Filepath = "changed.png"

		retained[0].Time = nil
		if callerMedia[0].Filepath != "first.png" || callerMedia[0].Time != &timestamp || timestamp != 0 {
			t.Errorf("✗ limit %d exposed caller records: %+v", limit, callerMedia[0])
		}
	}

	if !t.Failed() {
		t.Log("✓ selected records are independently owned at every cap boundary")
	}
}

// TestAdjustmentFailureRetainsChanges verifies invariant #3: Completed adjustments on failure.
//
// What is being tested:
// Given impossible size bounds and excess media, AdjustGeneration must return an error and exactly
// two completed records: the ignored aspect input followed by the media cap.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestAdjustmentFailureRetainsChanges(t *testing.T) {
	model := &catalog.Model{ID: "bounded", Params: params.Definitions{
		{FlagID: params.FlagTypeInputMedia, MaxMultiple: 1},
		{FlagID: params.FlagTypeSize, CustomSize: &params.SizeBounds{MinEdge: 100, MaxEdge: 101, EdgeIncrem: 64}},
	}}
	preparedGeneration, err := AdjustGeneration(model, params.FlagInputs{params.FlagTypeAspect: "16:9", params.FlagTypeSize: "100x100"}, []media.Input{{Filepath: "first.png"}, {Filepath: "second.png"}})
	changes := preparedGeneration.Changes

	if err == nil {
		t.Error("✗ impossible size constraints did not fail")
	}

	if len(changes) != 2 || changes[0].Type != params.ChangeIgnored || changes[0].FlagID != params.FlagTypeAspect || changes[1].Type != params.ChangeCapped {
		t.Errorf("✗ failure lost ordered completed adjustments: %+v", changes)
	}

	if !t.Failed() {
		t.Log("✓ failed scalar adjustment retains completed earlier changes")
	}
}

// FuzzSelectSources checks invariant #4: Retained source arguments.
//
// What is being tested:
// For arbitrary source strings and counts, SelectSources must return no sources when the model
// lacks media support. Otherwise it must return the original prefix up to the positive limit, or
// all sources for a nonpositive limit. Replacing a returned source must leave the caller's slice
// unchanged.
//
// Test class: Expanded.
// Test layer: Fuzzing.
// Kind: permanent.
func FuzzSelectSources(f *testing.F) {
	f.Add("photo.png", uint8(5), int8(2), true)
	f.Add("https://example.test/media", uint8(3), int8(0), true)
	f.Add("missing.png", uint8(2), int8(1), false)
	f.Fuzz(func(t *testing.T, sourceText string, suppliedCount uint8, limit int8, supported bool) {
		sourceCount := int(suppliedCount % 17)

		sourceArgs := make([]string, sourceCount)
		for sourceIndex := range sourceArgs {
			sourceArgs[sourceIndex] = strconv.Itoa(sourceIndex) + ":" + sourceText
		}

		originalSources := slices.Clone(sourceArgs)
		model := catalog.Model{}
		retainedCount := 0

		if supported {
			model.Params = []params.Definition{{FlagID: params.FlagTypeInputMedia, MaxMultiple: int(limit)}}

			retainedCount = sourceCount
			if limit > 0 {
				retainedCount = min(retainedCount, int(limit))
			}
		}

		retainedSources := SelectSources(sourceArgs, &model)
		if !slices.Equal(retainedSources, originalSources[:retainedCount]) {
			t.Errorf("✗ retained sources differ from the supported prefix: %q / %q", retainedSources, originalSources[:retainedCount])
		}

		if len(retainedSources) > 0 {
			retainedSources[0] = "replacement:" + retainedSources[0]
		}

		if !slices.Equal(sourceArgs, originalSources) {
			t.Errorf("✗ selected sources share writable caller state: %q / %q", sourceArgs, originalSources)
		}

		if !t.Failed() {
			t.Log("✓ source selection preserves the retained prefix and caller ownership")
		}
	})
}
