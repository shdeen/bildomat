package generation

// Invariants tested:
//  1. Unsupported frame prefixes: Given numeric, keyword, and absent frame prefixes,
//     DropFramePrefixes must clear every frame marker and return exactly two records that pair the
//     prefixed sources with their bare URLs.
//  2. Numeric frame anchors: For an eight-second video, ResolveFrameAnchors must choose first at or
//     below four seconds and last above four seconds.

import (
	"testing"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestDropFramePrefixes verifies invariant #1: Unsupported frame prefixes.
//
// What is being tested:
// Given numeric, keyword, and absent frame prefixes, DropFramePrefixes must clear every frame
// marker and return exactly two records that pair the prefixed sources with their bare URLs.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDropFramePrefixes(t *testing.T) {
	mediaInputs := []media.Input{
		framedImage(t, "https://media.example/a.png", 3, true, ""),
		framedImage(t, "https://media.example/b.png", 0, false, media.FrameLast),
		framedImage(t, "https://media.example/c.png", 0, false, ""),
	}

	records := DropFramePrefixes(mediaInputs)

	if len(records) != 2 {
		t.Fatalf("💣 %d records, want one per dropped prefix (2): %+v", len(records), records)
	}

	if records[0].InputVal != "3:https://media.example/a.png" || records[0].WireVal != "https://media.example/a.png" {
		t.Errorf("✗ first record = %q → %q, want the prefixed and bare source", records[0].InputVal, records[0].WireVal)
	}

	if records[1].InputVal != "last:https://media.example/b.png" || records[1].WireVal != "https://media.example/b.png" {
		t.Errorf("✗ second record = %q → %q, want the prefixed and bare source", records[1].InputVal, records[1].WireVal)
	}

	for i := range mediaInputs {
		if mediaInputs[i].HasFrame() {
			t.Errorf("✗ input %d still carries a frame prefix", i)
		}
	}

	if !t.Failed() {
		t.Log("✓ prefixes drop with one conformed record each and the media stays")
	}
}

// TestResolveFrameAnchorsSnapping verifies invariant #2: Numeric frame anchors.
//
// What is being tested:
// For an eight-second video, ResolveFrameAnchors must choose first at or below four seconds and
// last above four seconds. It must clear numeric times and return one record except at zero or
// eight seconds. Without a duration, zero must select first without a record and three seconds must
// select last with one record.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolveFrameAnchorsSnapping(t *testing.T) {
	cases := []struct {
		name        string
		seconds     float64
		params      params.Values
		wantAnchor  string
		wantRecords int
	}{
		{"below half opens", 2, params.Values{params.FlagTypeDuration: 8}, media.FrameFirst, 1},
		{"exactly half opens", 4, params.Values{params.FlagTypeDuration: 8}, media.FrameFirst, 1},
		{"past half closes", 6, params.Values{params.FlagTypeDuration: 8}, media.FrameLast, 1},
		{"zero opens silently", 0, params.Values{params.FlagTypeDuration: 8}, media.FrameFirst, 0},
		{"the duration closes silently", 8, params.Values{params.FlagTypeDuration: 8}, media.FrameLast, 0},
		{"no duration, zero opens silently", 0, params.Values{}, media.FrameFirst, 0},
		{"no duration, a later time closes", 3, params.Values{}, media.FrameLast, 1},
	}

	for _, c := range cases {
		mediaInputs := []media.Input{framedImage(t, "https://media.example/a.png", c.seconds, true, "")}

		records, err := ResolveFrameAnchors(mediaInputs, c.params)
		if err != nil {
			t.Errorf("✗ %s: %v", c.name, err)

			continue
		}

		if mediaInputs[0].FrameAnchor != c.wantAnchor || len(records) != c.wantRecords {
			t.Errorf("✗ %s: anchor %q with %d records, want %q with %d",
				c.name, mediaInputs[0].FrameAnchor, len(records), c.wantAnchor, c.wantRecords)
		}

		if _, timed := mediaInputs[0].FrameTime(); timed {
			t.Errorf("✗ %s: the numeric time survives resolution", c.name)
		}
	}

	if !t.Failed() {
		t.Log("✓ numeric times snap by the half rule and exact frames snap silently")
	}
}

// framedImage builds one image input with the given URL and frame prefix state.
func framedImage(test testing.TB, url string, seconds float64, timed bool, anchor string) media.Input {
	test.Helper()

	mediaInput := media.Input{URL: url, MIME: "image/png", FrameAnchor: anchor}
	if timed {
		mediaInput.Time = new(seconds)
	}

	return mediaInput
}
