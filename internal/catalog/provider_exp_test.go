package catalog

// Invariants tested:
//  1. Model clone isolation: Given a model with aliases, allowed values, and custom size bounds,
//     Model.clone must return an equal value with a separate CustomSize pointer.

import (
	"reflect"
	"testing"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestModelClone verifies invariant #1: Model clone isolation.
//
// What is being tested:
// Given a model with aliases, allowed values, and custom size bounds, Model.clone must return an
// equal value with a separate CustomSize pointer. Mutating the clone's aliases, allowed values, and
// bounds must leave the original equal to the fixture.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestModelClone(t *testing.T) {
	orig := fullModel(t)

	cloned := orig.clone()
	if !reflect.DeepEqual(cloned, orig) {
		t.Fatalf("💣 clone does not carry the declared data (cannot exercise mutation): %+v", cloned)
	}

	clonedSizeCfg, _ := cloned.Param(params.FlagTypeSize)

	origSizeCfg, _ := orig.Param(params.FlagTypeSize)
	if clonedSizeCfg.CustomSize == origSizeCfg.CustomSize {
		t.Errorf("✗ clone shares the CustomSize pointer with the original")
	}

	corruptModel(t, &cloned)

	if want := fullModel(t); !reflect.DeepEqual(orig, want) {
		t.Errorf("✗ mutating the clone reached the original:\n got %+v\nwant %+v", orig, want)
	}

	if !t.Failed() {
		t.Log("✓ a clone shares no mutable state with its original, new fields and CustomSize included")
	}
}

// fullModel returns a model with aliases, allowed values, and size bounds for clone-isolation
// checks.
func fullModel(test testing.TB) Model {
	test.Helper()

	return Model{
		ID: "m-full", Media: media.Image, Aliases: []string{"em"},
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio", AllowedValues: []string{"16:9", "21:9"}},
			{FlagID: params.FlagTypeDuration, ParamID: "duration", MinValue: params.GetSetIf(true, 2.0), MaxValue: params.GetSetIf(true, 10.0)},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 3},
			{FlagID: params.FlagTypeSize, ParamID: "size", CustomSize: &params.SizeBounds{MaxRatio: 2, MinEdge: 256, MaxEdge: 4096, MinPx: 65536, MaxPx: 1 << 24, EdgeIncrem: 64, LongEdge: 2048}},
		},
	}
}

// corruptModel mutates aliases, allowed values, and size bounds to expose shared storage.
func corruptModel(test testing.TB, model *Model) {
	test.Helper()

	for i := range model.Aliases {
		model.Aliases[i] = "corrupted"
	}

	for i := range model.Params {
		for j := range model.Params[i].AllowedValues {
			model.Params[i].AllowedValues[j] = "corrupted"
		}

		if model.Params[i].CustomSize != nil {
			*model.Params[i].CustomSize = params.SizeBounds{MaxRatio: -1, MinEdge: -1, MaxEdge: -1, MinPx: -1, MaxPx: -1, EdgeIncrem: -1, LongEdge: -1}
		}
	}
}
