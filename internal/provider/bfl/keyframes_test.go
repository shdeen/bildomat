package bfl

// Invariants tested:
//  1. Resolve flux frame anchors: Given duration eight, resolveFluxFrameAnchors must replace first
//     and last markers with times zero and eight without reordering the inputs. Without a duration,
//     it must order the opening, middle, and closing images and remove all frame markers.

import (
	"testing"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestResolveFluxFrameAnchors verifies invariant #1: Resolve flux frame anchors.
//
// What is being tested:
// Given duration eight, resolveFluxFrameAnchors must replace first and last markers with times zero
// and eight without reordering the inputs. Without a duration, it must order the opening, middle,
// and closing images and remove all frame markers.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolveFluxFrameAnchors(t *testing.T) {
	withDuration := []media.Input{
		{URL: "https://media.example/tail.png", MIME: "image/png", FrameAnchor: media.FrameLast},
		{URL: "https://media.example/lead.png", MIME: "image/png", FrameAnchor: media.FrameFirst},
	}

	resolveFluxFrameAnchors(withDuration, params.Values{params.FlagTypeDuration: 8})

	tailSeconds, tailSet := withDuration[0].FrameTime()
	leadSeconds, leadSet := withDuration[1].FrameTime()

	if !tailSet || tailSeconds != 8 || !leadSet || leadSeconds != 0 {
		t.Errorf("✗ anchored times = (%v %v, %v %v), want last at 8 and first at 0",
			tailSeconds, tailSet, leadSeconds, leadSet)
	}

	if withDuration[0].FrameAnchor != "" || withDuration[1].FrameAnchor != "" {
		t.Errorf("✗ anchors survive resolution with a duration set")
	}

	noDuration := []media.Input{
		{URL: "https://media.example/tail.png", MIME: "image/png", FrameAnchor: media.FrameLast},
		{URL: "https://media.example/mid.png", MIME: "image/png"},
		{URL: "https://media.example/lead.png", MIME: "image/png", FrameAnchor: media.FrameFirst},
	}

	resolveFluxFrameAnchors(noDuration, params.Values{})

	wantOrder := []string{"https://media.example/lead.png", "https://media.example/mid.png", "https://media.example/tail.png"}
	for i, wantURL := range wantOrder {
		if noDuration[i].URL != wantURL {
			t.Errorf("✗ position %d = %q, want %q", i, noDuration[i].URL, wantURL)
		}

		if noDuration[i].HasFrame() {
			t.Errorf("✗ position %d still carries a frame prefix", i)
		}
	}

	if !t.Failed() {
		t.Log("✓ anchors become exact keyframe times with a duration and ordering without one")
	}
}
