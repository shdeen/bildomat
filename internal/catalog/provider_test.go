package catalog

// Invariants tested:
//  1. Provider encoding: Given a Provider with only identity fields, JSON marshaling must emit
//     exactly those fields.
//  2. Declared parameter support: Given a model declaring only aspect and size, SupportsParam must
//     return true only for those flags among the tested set.

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestProviderEncoding verifies invariant #1: Provider encoding.
//
// What is being tested:
// Given a Provider with only identity fields, JSON marshaling must emit exactly those fields. With
// models, aggregator, and config supplied, it must include those keys. Marshaling a Model with no
// description must omit that value.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderEncoding(t *testing.T) {
	identity := encodedKeys(t, Provider{ID: "p", DisplayName: "P", APIKeyEnvVar: "P_KEY"})
	if !reflect.DeepEqual(identity, map[string]any{"id": "p", "displayName": "P", "apiKeyEnvVar": "P_KEY"}) {
		t.Errorf("✗ an identity-only provider encodes %v", identity)
	}

	full := encodedKeys(t, Provider{ID: "p", DisplayName: "P", APIKeyEnvVar: "P_KEY", Aggregator: true, Config: &ProviderConfig{StringParams: []params.FlagType{params.FlagTypeDuration}}, Models: []Model{{ID: "m", Name: "M", Media: media.Image}}})
	for _, key := range []string{"aggregator", "config", "models"} {
		if _, present := full[key]; !present {
			t.Errorf("✗ a full provider lacks %q: %v", key, full)
		}
	}

	model, _ := encodedKeys(t, Model{ID: "m", Name: "M", Media: media.Image})["description"]
	if model != nil {
		t.Errorf("✗ a model without a description encodes one: %v", model)
	}

	if !t.Failed() {
		t.Log("✓ a provider encodes its identity and only what it declares beyond it")
	}
}

// TestSupports verifies invariant #2: Declared parameter support.
//
// What is being tested:
// Given a model declaring only aspect and size, SupportsParam must return true only for those flags
// among the tested set. With no declared parameters, it must return false for every tested flag.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSupports(t *testing.T) {
	all := []params.FlagType{
		params.FlagTypeAspect, params.FlagTypeResolution, params.FlagTypeQuality, params.FlagTypeThinkingLevel,
		params.FlagTypeThoughts, params.FlagTypeDuration, params.FlagTypeImageN, params.FlagTypeOutputFormat, params.FlagTypeInputMedia, params.FlagTypeSize,
	}

	model := Model{ID: "m", Media: media.Image, Params: []params.Definition{
		{FlagID: params.FlagTypeAspect}, {FlagID: params.FlagTypeSize},
	}}
	for _, p := range all {
		want := p == params.FlagTypeAspect || p == params.FlagTypeSize
		if got := model.SupportsParam(p); got != want {
			t.Errorf("✗ SupportsParam(%s) = %v, want %v", p, got, want)
		}
	}

	empty := Model{ID: "e", Media: media.Image}
	for _, p := range all {
		if empty.SupportsParam(p) {
			t.Errorf("✗ an empty model config reports SupportsParam(%s)", p)
		}
	}

	if !t.Failed() {
		t.Log("✓ SupportsParam reflects exactly the declared param set")
	}
}

// encodedKeys decodes JSON into an object map for field-presence assertions.
func encodedKeys(t *testing.T, value any) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("💣 encoding: %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("💣 decoding the encoding: %v", err)
	}

	return document
}
