package generation

// Invariants tested:
//  1. Request carries media: After resolving a video model, passing its ProvModelPair in a
//     Generation request must let the probe generator read media.Video from request.Model.Media.

import (
	"context"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestRequestCarriesMedia verifies invariant #1: Request carries media.
//
// What is being tested:
// After resolving a video model, passing its ProvModelPair in a Generation request must let the
// probe generator read media.Video from request.Model.Media.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestRequestCarriesMedia(t *testing.T) {
	probe := &mediaProbe{test: t}

	loadedCatalog, loadErr := catalog.LoadCatalog(params.Flags(),
		catalog.Source{ProviderID: "prov", ConfigBytes: []byte(`{
		  "id": "prov", "displayName": "Prov", "apiKeyEnvVar": "PROV_API_KEY",
		  "models": [{"id": "m-vid", "name": "m-vid", "media": "video", "params": []}],
		  "config": {
		  "videoAPI": {
		    "asyncJobsURL": "https://example.test/vid", "jobIDField": "id",
		    "progressStatusText": ["pending"], "completedStatusText": "done", "failedStatusText": ["failed"],
		    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
		    "fallbackExt": ".mp4", "pollInterval": 5, "pollTimeout": 600
		  }
		  }
}`)},
	)
	if loadErr != nil {
		t.Fatalf("💣 fixture catalog failed: %v", loadErr)
	}

	binds, err := loadedCatalog.ResolveModelInput("prov/m-vid")
	if err != nil || len(binds) != 1 {
		t.Fatalf("💣 fixture resolution failed (cannot exercise the contract): (%+v, %v)", binds, err)
	}

	if _, err := probe.Generate(context.Background(), Generation{ProvModelPair: binds[0]}); err != nil {
		t.Errorf("✗ probe Generate errored: %v", err)
	}

	if probe.observedKind != media.Video {
		t.Errorf("✗ generator observed media %q through the request, want video with no table lookup", probe.observedKind)
	}

	if !t.Failed() {
		t.Log("✓ a generator observes the resolved media directly on the request")
	}
}

// mediaProbe records the media kind submitted to Generate.
//   - test: the owning test
//   - observedKind: the output kind observed in the request
type mediaProbe struct {
	test         testing.TB
	observedKind media.Kind
}

// Generate records the media type carried by a generation request.
func (probe *mediaProbe) Generate(_ context.Context, request Generation) (Result, error) {
	probe.test.Helper()

	probe.observedKind = request.Model.Media

	return Result{}, nil
}
