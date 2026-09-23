package kling

// Invariants tested:
//  1. Kling route helper branches: Given a nil model, selectCreationPath must return an empty
//     route. Given opening and closing frame markers, omniContents must return first_frame then
//     last_frame records. Given three unmarked inputs, frameContents must return exactly two
//     records.
//  2. Kling creation envelope branches: When creation returns code 1200, createTask must return
//     ErrResponseGen. For a video response with code zero and data.id, it must return that ID
//     without an error.
//  3. Declared media fields: Given one image URL and the declared_images parameter name,
//     buildRequestBody must put the URL in that field. A standard image model must use a string; an
//     omni model must use a one-element array containing an image object.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestKlingRouteHelperBranches verifies invariant #1: Kling route helper branches.
//
// What is being tested:
// Given a nil model, selectCreationPath must return an empty route. Given opening and closing frame
// markers, omniContents must return first_frame then last_frame records. Given three unmarked
// inputs, frameContents must return exactly two records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingRouteHelperBranches(t *testing.T) {
	if creationPath := selectCreationPath(nil, 0); creationPath != "" {
		t.Errorf("✗ nil model selected route %q", creationPath)
	}

	omniMedia := []media.Input{
		{URL: "https://media.example/open.png", FrameAnchor: media.FrameFirst},
		{URL: "https://media.example/close.png", FrameAnchor: media.FrameLast},
	}

	omniContents := omniContents(omniMedia)
	if len(omniContents) != 2 || mapValue(t, omniContents[0], "omni opening frame")["type"] != "first_frame" || mapValue(t, omniContents[1], "omni closing frame")["type"] != "last_frame" {
		t.Errorf("✗ anchored omni contents = %#v, want opening and closing frames", omniContents)
	}

	standardMedia := []media.Input{
		{URL: "https://media.example/a.png"},
		{URL: "https://media.example/b.png"},
		{URL: "https://media.example/c.png"},
	}

	standardContents := frameContents(standardMedia)
	if len(standardContents) != 2 {
		t.Errorf("✗ surplus standard frame contents = %#v, want two retained frame roles", standardContents)
	}

	if !t.Failed() {
		t.Log("✓ Kling route helpers cover nil models, anchored omni frames, and capped surplus frames")
	}
}

// TestKlingCreationEnvelopeBranches verifies invariant #2: Kling creation envelope branches.
//
// What is being tested:
// When creation returns code 1200, createTask must return ErrResponseGen. For a video response with
// code zero and data.id, it must return that ID without an error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestKlingCreationEnvelopeBranches(t *testing.T) {
	serverResponse := []byte(`{"code":1200,"message":"creation envelope failed","data":null}`)

	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, _ *http.Request) {
		_, _ = responseWriter.Write(serverResponse)
	}))
	defer server.Close()

	if _, creationErr := createTask(t.Context(), server.URL, httpapi.AuthCredential{}, &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: catalog.Provider{ID: ProviderID}, Model: catalog.Model{Media: media.Image}}}, map[string]any{}); !errors.Is(creationErr, errs.ErrResponseGen) {
		t.Errorf("✗ creation envelope error = %v, want generation classification", creationErr)
	}

	serverResponse = []byte(`{"code":0,"data":{"id":"video-coverage-task"}}`)

	taskID, creationErr := createTask(context.Background(), server.URL, httpapi.AuthCredential{}, &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: catalog.Provider{ID: ProviderID}, Model: catalog.Model{Media: media.Video}}}, map[string]any{})
	if creationErr != nil || taskID != "video-coverage-task" {
		t.Errorf("✗ video creation result = %q, %v; want video task ID", taskID, creationErr)
	}

	if !t.Failed() {
		t.Log("✓ Kling creation handles failed envelopes and the video task identifier field")
	}
}

// TestDeclaredMediaField verifies invariant #3: Declared media fields.
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
// What is being tested:
// Given one image URL and the declared_images parameter name, buildRequestBody must put the URL in
// that field. A standard image model must use a string; an omni model must use a one-element array
// containing an image object.
func TestDeclaredMediaField(t *testing.T) {
	for _, family := range []string{"", "omni"} {
		t.Run(family, func(t *testing.T) {
			request := &generation.Generation{
				ProvModelPair: catalog.ProvModelPair{Model: catalog.Model{
					ID: "image-model", Media: media.Image, Family: family,
					Params: params.Definitions{{FlagID: params.FlagTypeInputMedia, ParamID: "declared_images"}},
				}},
				Preparation: generation.Preparation{InputMedia: []media.Input{{URL: "https://media.example/reference.png", MIME: "image/png"}}},
			}

			_, body := buildRequestBody(request)
			if family == "" {
				if body["declared_images"] != request.InputMedia[0].URL {
					t.Errorf("✗ declared media field lost: %#v", body)
				}
			} else {
				images, valid := body["declared_images"].([]any)
				if !valid || len(images) != 1 {
					t.Errorf("✗ declared omni images lost: %#v", body)
				} else if imageFields, valid := images[0].(map[string]any); !valid || imageFields["image"] != request.InputMedia[0].URL {
					t.Errorf("✗ omni reference representation changed: %#v", images)
				}
			}

			if !t.Failed() {
				t.Log("✓ declared media field retains the appropriate image representation")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ image media follows the declared field")
	}
}
