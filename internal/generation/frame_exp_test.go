package generation

// Invariants tested:
//  1. Keyword frame anchors: Given first and last keywords, ResolveFrameAnchors must preserve their
//     anchors and return no records.
//  2. Frame anchor pairs and conflicts: Given two numeric times, ResolveFrameAnchors must assign
//     first and last in chronological order and return two records.

import (
	"errors"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestResolveFrameAnchorsKeywords verifies invariant #1: Keyword frame anchors.
//
// What is being tested:
// Given first and last keywords, ResolveFrameAnchors must preserve their anchors and return no
// records. Given two first keywords, it must return ErrInputMediaTime and name both sources in the
// error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestResolveFrameAnchorsKeywords(t *testing.T) {
	mediaInputs := []media.Input{
		framedImage(t, "https://media.example/tail.png", 0, false, media.FrameLast),
		framedImage(t, "https://media.example/lead.png", 0, false, media.FrameFirst),
	}

	records, err := ResolveFrameAnchors(mediaInputs, params.Values{})
	if err != nil || len(records) != 0 {
		t.Errorf("✗ keywords = (%v, %v), want direct anchors with no records", records, err)
	}

	if mediaInputs[0].FrameAnchor != media.FrameLast || mediaInputs[1].FrameAnchor != media.FrameFirst {
		t.Errorf("✗ anchors = (%q, %q), want (last, first)", mediaInputs[0].FrameAnchor, mediaInputs[1].FrameAnchor)
	}

	duplicated := []media.Input{
		framedImage(t, "https://media.example/one.png", 0, false, media.FrameFirst),
		framedImage(t, "https://media.example/two.png", 0, false, media.FrameFirst),
	}

	_, err = ResolveFrameAnchors(duplicated, params.Values{})
	if !errors.Is(err, errs.ErrInputMediaTime) {
		t.Fatalf("💣 duplicate anchors = %v, want the frame-time rejection", err)
	}

	for _, requiredName := range []string{"one.png", "two.png"} {
		if !strings.Contains(err.Error(), requiredName) {
			t.Errorf("✗ the duplicate-anchor error does not name %q: %v", requiredName, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ keywords resolve directly and duplicates fail naming both sources")
	}
}

// TestResolveFrameAnchorsPairsAndConflicts verifies invariant #2: Frame anchor pairs and conflicts.
//
// What is being tested:
// Given two numeric times, ResolveFrameAnchors must assign first and last in chronological order
// and return two records. Beside a first keyword, a numeric time must take last. Equal times, three
// timed images, and a timed video must each return ErrInputMediaTime.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestResolveFrameAnchorsPairsAndConflicts(t *testing.T) {
	pair := []media.Input{
		framedImage(t, "https://media.example/six.png", 6, true, ""),
		framedImage(t, "https://media.example/five.png", 5, true, ""),
	}

	records, err := ResolveFrameAnchors(pair, params.Values{params.FlagTypeDuration: 8})
	if err != nil || len(records) != 2 {
		t.Fatalf("💣 numeric pair = (%v, %v), want two snap records", records, err)
	}

	if pair[0].FrameAnchor != media.FrameLast || pair[1].FrameAnchor != media.FrameFirst {
		t.Errorf("✗ pair anchors = (%q, %q), want numeric order (last, first)", pair[0].FrameAnchor, pair[1].FrameAnchor)
	}

	mixed := []media.Input{
		framedImage(t, "https://media.example/lead.png", 0, false, media.FrameFirst),
		framedImage(t, "https://media.example/two.png", 2, true, ""),
	}

	if _, err := ResolveFrameAnchors(mixed, params.Values{params.FlagTypeDuration: 8}); err != nil {
		t.Fatalf("💣 keyword plus numeric = %v, want the free frame filled", err)
	}

	if mixed[1].FrameAnchor != media.FrameLast {
		t.Errorf("✗ the numeric time beside a first keyword = %q, want the free closing frame", mixed[1].FrameAnchor)
	}

	equalTimes := []media.Input{
		framedImage(t, "https://media.example/a.png", 4, true, ""),
		framedImage(t, "https://media.example/b.png", 4, true, ""),
	}

	if _, err := ResolveFrameAnchors(equalTimes, params.Values{params.FlagTypeDuration: 8}); !errors.Is(err, errs.ErrInputMediaTime) {
		t.Errorf("✗ equal times = %v, want the frame-time rejection", err)
	}

	crowded := []media.Input{
		framedImage(t, "https://media.example/a.png", 1, true, ""),
		framedImage(t, "https://media.example/b.png", 2, true, ""),
		framedImage(t, "https://media.example/c.png", 3, true, ""),
	}

	if _, err := ResolveFrameAnchors(crowded, params.Values{params.FlagTypeDuration: 8}); !errors.Is(err, errs.ErrInputMediaTime) {
		t.Errorf("✗ three timed inputs = %v, want the frame-time rejection", err)
	}

	timedVideo := []media.Input{{URL: "https://media.example/clip.mp4", MIME: "video/mp4", Time: new(2.0)}}
	if _, err := ResolveFrameAnchors(timedVideo, params.Values{}); !errors.Is(err, errs.ErrInputMediaTime) {
		t.Errorf("✗ a timed video = %v, want the frame-time rejection", err)
	}

	if !t.Failed() {
		t.Log("✓ pairs order by time, free frames fill, and unresolvable requests fail")
	}
}
