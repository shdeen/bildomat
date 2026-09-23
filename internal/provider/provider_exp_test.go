package provider

// Invariants tested:
//  1. Empty and expired responses: Given an empty image data array or an entry without artifact
//     data, descriptor generation must return ErrResponseNoData and no artifacts.
//  2. Impossible size declarations: Given contradictory edges, pixel bounds, a ratio below one, or
//     an impossible increment band, the descriptor generator's AdjustParams must return
//     ErrProvConfigInvalid naming the model and omit the size parameter.

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// TestEmptyAndExpiredResponses verifies invariant #1: Empty and expired responses.
//
// What is being tested:
// Given an empty image data array or an entry without artifact data, descriptor generation must
// return ErrResponseNoData and no artifacts. If the artifact URL returns HTTP 403, it must return
// ErrTransportStatus and no artifacts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestEmptyAndExpiredResponses(t *testing.T) {
	provCfg := urlResponseImageFixtureCfg(t)
	rasterModel := configuredModel(t, provCfg, strengthOnlyImageModelID)
	harness := newURLArtifactHTTPHarness(t)

	responseCases := []struct {
		name               string
		successBody        string
		artifactStatusCode int
		errorClass         error
	}{
		{name: "empty data array", successBody: `{"data":[]}`, errorClass: errs.ErrResponseNoData},
		{name: "entry without artifact data", successBody: `{"data":[{}]}`, errorClass: errs.ErrResponseNoData},
		{name: "expired artifact URL", artifactStatusCode: http.StatusForbidden, errorClass: errs.ErrTransportStatus},
	}

	for _, responseCase := range responseCases {
		t.Run(responseCase.name, func(t *testing.T) {
			harness.resetRequestRecording()
			harness.successBody = responseCase.successBody
			harness.artifactStatusCode = responseCase.artifactStatusCode

			imageRun := runImageGeneration(t, provCfg, rasterModel, nil, nil, harness)
			cleanupDownloadedArtifacts(t, imageRun.result.Artifacts)

			if !errors.Is(imageRun.err, responseCase.errorClass) {
				t.Errorf("✗ response error = %v, want classification %v", imageRun.err, responseCase.errorClass)
			}

			if len(imageRun.result.Artifacts) != 0 {
				t.Errorf("✗ failed response landed artifacts: %+v", imageRun.result.Artifacts)
			}

			if !t.Failed() {
				t.Log("✓ the response failure stays classified and lands no artifact")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ malformed and expired responses fail without landing artifacts")
	}
}

// TestImpossibleSizeDeclarations verifies invariant #2: Impossible size declarations.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Test kind: permanent.
// What is being tested:
// Given contradictory edges, pixel bounds, a ratio below one, or an impossible increment band, the
// descriptor generator's AdjustParams must return ErrProvConfigInvalid naming the model and omit
// the size parameter.
func TestImpossibleSizeDeclarations(t *testing.T) {
	for _, bounds := range []params.SizeBounds{{MinEdge: 128, MaxEdge: 64}, {MinPx: 10000, MaxPx: 9000}, {MaxRatio: 0.5}, {MinPx: 10000, MaxPx: 10100, EdgeIncrem: 16}} {
		model := catalog.Model{ID: "impossible-dimensions", Params: []params.Definition{{FlagID: params.FlagTypeSize, CustomSize: &bounds}}}
		providerDescription := resizingVideoFixtureCfg(t)

		generator, err := NewProvider(&providerDescription)
		if err != nil {
			t.Fatalf("💣 construct resizing provider: %v", err)
		}

		preparedGeneration, err := generator.AdjustParams(&model, params.FlagInputs{params.FlagTypeSize: "96x96"}, nil, nil)
		if !errors.Is(err, errs.ErrProvConfigInvalid) || !strings.Contains(fmt.Sprint(err), model.ID) {
			t.Errorf("✗ %+v: expected classified model error, got %v", bounds, err)
		}

		if _, present := preparedGeneration.Params[params.FlagTypeSize]; present {
			t.Errorf("✗ impossible declaration produced size %v with %+v", preparedGeneration.Params, preparedGeneration.Changes)
		}
	}

	if !t.Failed() {
		t.Log("✓ impossible dimensions fail before generation")
	}
}
