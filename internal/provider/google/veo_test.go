package google

// Invariants tested:
//  1. Veo input media shapes: Given opening and closing image URLs, requestBody must put their URIs
//     and MIME types in image and lastFrame. Given local MP4 bytes, it must put base64 encodedVideo
//     and video/mp4 encoding in video. Given a remote MP4, it must put the unchanged uri and
//     encoding in video.
//  2. Veo adjust: Without reference images or high resolution, adjustVeoDuration must preserve
//     duration four, duration eight, or an absent duration, and return no records. Reference
//     images, 1080p, or 4k must set duration eight and return exactly one Forced record when the
//     supplied value changes or was absent. It must leave resolution unchanged and return no error.
//  3. Original video reuse: Given an original Veo URI, duration eight, and resolution 720p,
//     requestBody must create a video object containing only that unchanged URI, including its
//     query string. It must put the parameters in durationSeconds and resolution.

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestVeoInputMediaShapes verifies invariant #1: Veo input media shapes.
//
// What is being tested:
// Given opening and closing image URLs, requestBody must put their URIs and MIME types in image and
// lastFrame. Given local MP4 bytes, it must put base64 encodedVideo and video/mp4 encoding in
// video. Given a remote MP4, it must put the unchanged uri and encoding in video.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVeoInputMediaShapes(t *testing.T) {
	model := catalog.Model{Media: media.Video, Params: []params.Definition{
		{FlagID: params.FlagTypeInputMedia, MaxMultiple: 3},
		{FlagID: params.FlagTypeDuration, ParamID: "durationSeconds"},
	}}
	parameterValues := params.Values{params.FlagTypeDuration: 8}

	frameInputs := []media.Input{
		{URL: "https://media.example/open.png", MIME: "image/png", FrameAnchor: media.FrameFirst},
		{URL: "https://media.example/close.webp", MIME: "image/webp", FrameAnchor: media.FrameLast},
	}
	body := requestBody(&model, "animate", parameterValues, frameInputs, "")
	instances := body["instances"].([]any)
	instance := instances[0].(map[string]any)
	wantOpening := map[string]any{"gcsUri": frameInputs[0].URL, "mimeType": "image/png"}
	wantClosing := map[string]any{"gcsUri": frameInputs[1].URL, "mimeType": "image/webp"}

	if !reflect.DeepEqual(instance["image"], wantOpening) || !reflect.DeepEqual(instance["lastFrame"], wantClosing) {
		t.Errorf("✗ frame instance = %#v, want image %#v and lastFrame %#v", instance, wantOpening, wantClosing)
	}

	videoInput := []media.Input{{Bytes: []byte("MP4"), MIME: "video/mp4", Filepath: "/tmp/source.mp4"}}
	body = requestBody(&model, "continue", parameterValues, videoInput, "")
	instances = body["instances"].([]any)
	instance = instances[0].(map[string]any)
	wantVideo := map[string]any{"encodedVideo": base64.StdEncoding.EncodeToString([]byte("MP4")), "encoding": "video/mp4"}

	if !reflect.DeepEqual(instance["video"], wantVideo) {
		t.Errorf("✗ video = %#v, want %#v", instance["video"], wantVideo)
	}

	remoteVideo := []media.Input{{URL: "https://media.example/source.mp4", MIME: "video/mp4"}}
	body = requestBody(&model, "continue", parameterValues, remoteVideo, "")
	instances = body["instances"].([]any)
	instance = instances[0].(map[string]any)
	wantRemoteVideo := map[string]any{"uri": remoteVideo[0].URL, "encoding": "video/mp4"}

	if !reflect.DeepEqual(instance["video"], wantRemoteVideo) {
		t.Errorf("✗ remote video = %#v, want %#v", instance["video"], wantRemoteVideo)
	}
}

// TestVeoAdjust verifies invariant #2: Veo adjust.
//
// What is being tested:
// Without reference images or high resolution, adjustVeoDuration must preserve duration four,
// duration eight, or an absent duration, and return no records. Reference images, 1080p, or 4k must
// set duration eight and return exactly one Forced record when the supplied value changes or was
// absent. It must leave resolution unchanged and return no error.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVeoAdjust(t *testing.T) {
	cases := []adjCase{
		{"none requested 4", 0, "", 4, true, 4, true, nil},
		{"none requested 8", 0, "", 8, true, 8, true, nil},
		{"none unrequested", 0, "", 0, false, 0, false, nil},
		{"720p requested 4", 0, "720p", 4, true, 4, true, nil},
		{"refs requested 4", 1, "", 4, true, 8, true, forced(t, "4", ReasonReferenceImages)},
		{"refs requested 8", 1, "", 8, true, 8, true, nil},
		{"refs unrequested", 1, "", 0, false, 8, true, forced(t, "", ReasonReferenceImages)},
		{"1080p requested 4", 0, "1080p", 4, true, 8, true, forced(t, "4", "1080p")},
		{"1080p requested 8", 0, "1080p", 8, true, 8, true, nil},
		{"1080p unrequested", 0, "1080p", 0, false, 8, true, forced(t, "", "1080p")},
		{"4k requested 4", 0, "4k", 4, true, 8, true, forced(t, "4", "4k")},
		{"4k requested 8", 0, "4k", 8, true, 8, true, nil},
		{"4k unrequested", 0, "4k", 0, false, 8, true, forced(t, "", "4k")},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkAdjust(t, c)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the forcing truth table holds across every trigger and request combination")
	}
}

// TestVeoReuseRequest verifies invariant #3: Original video reuse.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Given an original Veo URI, duration eight, and resolution 720p, requestBody must create a video
// object containing only that unchanged URI, including its query string. It must put the parameters
// in durationSeconds and resolution.
// Kind: permanent.
func TestVeoReuseRequest(t *testing.T) {
	model := mustModel(t, "veo-3.1-generate-preview")
	videoURI := "https://generativelanguage.googleapis.com/v1beta/files/original:download?alt=media&token=a=b"
	parameterValues := params.Values{params.FlagTypeDuration: 8, params.FlagTypeResolution: "720p"}
	body := requestBody(&model, "continue", parameterValues, nil, videoURI)

	instance := body["instances"].([]any)[0].(map[string]any)
	if !reflect.DeepEqual(instance["video"], map[string]any{"uri": videoURI}) {
		t.Errorf("✗ submitted reference %#v", instance["video"])
	}

	wireParams := body["parameters"].(map[string]any)
	if wireParams["durationSeconds"] != 8 || wireParams["resolution"] != "720p" {
		t.Errorf("✗ extension parameters %#v", wireParams)
	}

	if !t.Failed() {
		t.Log("✓ extension submits the complete original URI without encoding")
	}
}

// adjCase is one duration-forcing truth-table case.
type adjCase struct {
	name     string
	refs     int
	res      string
	dur      int
	durSet   bool
	wantDur  int
	wantSent bool
	want     []params.Adjustment
}

// forced creates the one expected Forced record.
func forced(test testing.TB, requested, detail string) []params.Adjustment {
	test.Helper()

	return []params.Adjustment{{
		FlagID: params.FlagTypeDuration, Type: params.ChangeForced,
		InputVal: requested, WireVal: "8", Comment: detail,
	}}
}

// checkAdjust drives one truth-table case through the adjuster.
func checkAdjust(t *testing.T, c adjCase) {
	t.Helper()

	gp := params.Values{}
	if c.durSet {
		gp[params.FlagTypeDuration] = c.dur
	}

	if c.res != "" {
		gp[params.FlagTypeResolution] = c.res
	}

	inputs := make([]media.Input, 0, c.refs)
	for range c.refs {
		inputs = append(inputs, fixtureInputMedia(t, "R", "/tmp/r.png"))
	}

	chs, adjustErr := adjustVeoDuration(gp, inputs)
	got := gp

	if adjustErr != nil {
		t.Errorf("✗ adjustParams returned %v, want no error", adjustErr)
	}

	adjustedSecs, durationSent := got[params.FlagTypeDuration]
	if durationSent != c.wantSent {
		t.Errorf("✗ duration sent = %v, want %v", durationSent, c.wantSent)
	}

	if c.wantSent && adjustedSecs != c.wantDur {
		t.Errorf("✗ duration = %v, want %d", adjustedSecs, c.wantDur)
	}

	adjustedRes, _ := got[params.FlagTypeResolution].(string)
	if adjustedRes != c.res {
		t.Errorf("✗ the adjuster touched the resolution: %q → %q", c.res, adjustedRes)
	}

	if !reflect.DeepEqual(chs, c.want) {
		t.Errorf("✗ records = %+v, want %+v", chs, c.want)
	}

	if !t.Failed() {
		t.Logf("✓ %s → duration %v (sent %v) with the exact record set", c.name, adjustedSecs, c.wantSent)
	}
}
