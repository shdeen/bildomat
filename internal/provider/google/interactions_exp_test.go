package google

// Invariants tested:
//  1. Unplaced parameter refusal: Given a supplied quality parameter whose declaration has no
//     request path, interactionImageBody must return a nil body and ErrProvConfigParamUnplaced
//     naming the quality flag.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestInteractionBodyRefusesUnplacedParam verifies invariant #1: Unplaced parameter refusal.
//
// What is being tested:
// Given a supplied quality parameter whose declaration has no request path, interactionImageBody
// must return a nil body and ErrProvConfigParamUnplaced naming the quality flag.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestInteractionBodyRefusesUnplacedParam(t *testing.T) {
	model := catalog.Model{ID: "synthetic-interactions-image", Media: media.Image, Params: []params.Definition{
		{FlagID: params.FlagTypeQuality},
		{FlagID: params.FlagTypeThoughts},
	}}
	parameterValues := params.Values{params.FlagTypeQuality: "high", params.FlagTypeThoughts: true}

	body, err := interactionImageBody(&model, "a prompt", parameterValues, nil)
	if !errors.Is(err, errs.ErrProvConfigParamUnplaced) {
		t.Errorf("✗ err = %v, want the unplaced-parameter error", err)
	}

	if body != nil {
		t.Errorf("✗ a body was built despite the unplaced parameter: %v", body)
	}

	if !strings.Contains(fmt.Sprint(err), string(params.FlagTypeQuality)) {
		t.Errorf("✗ the error does not name the unplaced flag: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a declared parameter with no request path stops the request before it is sent")
	}
}
