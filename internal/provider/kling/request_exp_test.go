package kling

// Invariants tested:
//  1. Kling route selection: For arbitrary model IDs and input counts, selectCreationPath must
//     return /v1/images/generations or /v1/images/omni-image for images. For videos, the route must
//     begin with /text-to-video/, /image-to-video/, or /omni-video/. Any other medium must produce
//     an empty route. The call must not panic.

import (
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
)

// FuzzKlingRouteSelection verifies invariant #1: Kling route selection.
//
// What is being tested:
// For arbitrary model IDs and input counts, selectCreationPath must return /v1/images/generations
// or /v1/images/omni-image for images. For videos, the route must begin with /text-to-video/,
// /image-to-video/, or /omni-video/. Any other medium must produce an empty route. The call must
// not panic.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzKlingRouteSelection(f *testing.F) {
	providerConfig := loadKlingTestProvider(f)

	for modelIndex := range providerConfig.Models {
		configuredModel := &providerConfig.Models[modelIndex]
		f.Add(configuredModel.ID, string(configuredModel.Media), 0)
		f.Add(configuredModel.ID, string(configuredModel.Media), 1)
	}

	f.Add("", "", -1)

	f.Fuzz(func(t *testing.T, modelID, mediumName string, mediaCount int) {
		configuredModel := catalog.Model{ID: modelID, Media: media.Kind(mediumName)}
		creationPath := selectCreationPath(&configuredModel, mediaCount)

		switch configuredModel.Media {
		case media.Image:
			if creationPath != "/v1/images/generations" && creationPath != "/v1/images/omni-image" {
				t.Errorf("✗ image route = %q, want a documented image route", creationPath)
			}
		case media.Video:
			if !strings.HasPrefix(creationPath, "/text-to-video/") &&
				!strings.HasPrefix(creationPath, "/image-to-video/") &&
				!strings.HasPrefix(creationPath, "/omni-video/") {
				t.Errorf("✗ video route = %q, want a documented video route", creationPath)
			}
		default:
			if creationPath != "" {
				t.Errorf("✗ unsupported medium %q selected route %q", configuredModel.Media, creationPath)
			}
		}

		if !t.Failed() {
			t.Log("✓ the model and media count select a documented route or no route")
		}
	})
}
