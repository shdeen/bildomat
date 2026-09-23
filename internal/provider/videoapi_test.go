package provider

// Invariants tested:
//  1. Video job body media references: Given image, video, first-frame, and last-frame URLs,
//     getVideoJobBody must put ordinary references in the configured nested list and frames in the
//     configured frame list, preserving URL kinds, order, and role values.
//  2. Submitted request ownership: Given a PNG reference and an empty caller parameter map,
//     prepareVideoInputs with resizing enabled must succeed without adding parameters to that map
//     or changing the caller's original image bytes.
//  3. Video lifecycle: Given the three video API layouts, submitVideo must POST and poll with
//     authentication at the declared routes, apply the expected authentication to the content or
//     returned download URL, and return one file-backed MP4 containing the served bytes.
//  4. Video duration: Given duration 4, submitVideo must send exactly model, prompt, and the
//     declared duration field in its JSON start request: the string "4" under seconds or the number
//     4 under duration.
//  5. Video reference encoding: Given no reference or one image reference, submitVideo must omit
//     the reference field or encode it under the configured field as a single image_url object or
//     nested reference list, with no extra request keys.
//  6. Video submission lifecycle: After adjusting and conforming a 1600x1000 image, submitVideo
//     must POST one 1280x720 PNG with size 1280x720 and return one file-backed artifact with an
//     extension.
//  7. Video polling interval: Given one working response followed by completion and PollInterval
//     set to one second, submitVideo must succeed and separate the first two polls by at least one
//     second and less than five seconds.
//  8. Video URL traversal: Given a completed response without its required URL, submitVideo must
//     return ErrResponseNoData containing the job ID.
//  9. Origin-scoped video authentication: Given a cross-origin result URL or a download redirect to
//     another origin, submitVideo must fetch the served MP4 bytes without sending Authorization to
//     the destination server.
//  10. URL traversal: Given the listed object and array paths, walkURL must return the expected
//      terminal string.
//  11. Omitted video duration field: Given duration 4 but no duration ParamID in the model,
//      submitVideo must complete successfully and send only model and prompt in the JSON start
//      body.
//  12. Multipart video duration: Given duration 4 and an image reference that selects multipart
//      submission, submitVideo must succeed and send exactly one seconds field with value "4".
//  13. Video form URL reference: Given a URL image reference and empty caller parameters,
//      submitVideo must download the image once without Authorization, upload one resized 1280x720
//      file with size 1280x720, and leave the caller parameter map empty.
//  14. Video wire params: Given aspect 16:9 and resolution 480p, submitVideo must send exactly
//      model, prompt, aspect_ratio, and resolution in its JSON start body, preserving both
//      parameter values.
//  15. Video prompt consumption: Given empty and nonempty prompts, getVideoJobBody and
//      videoJobFields must omit prompt for PromptIgnored models and otherwise include its exact
//      supplied value.
//  16. Loaded provider request lifecycle: Given a video provider loaded from JSON and duration 5
//      adjusted to 4, submitVideo must succeed, return one artifact, and send exactly model,
//      prompt, and numeric seconds 4 in the start body.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestVideoJobBodyMediaReferences verifies invariant #1: Video job body media references.
//
// What is being tested:
// Given image, video, first-frame, and last-frame URLs, getVideoJobBody must put ordinary
// references in the configured nested list and frames in the configured frame list, preserving URL
// kinds, order, and role values.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVideoJobBodyMediaReferences(t *testing.T) {
	api := &catalog.VideoAPI{
		InputMediaProvParam: "input_references",
		InputMediaStyle:     catalog.InputMediaNested,
		FrameMediaProvParam: "frame_images",
		FrameRoleProvParam:  "frame_type",
		FirstFrameProvValue: "first_frame",
		LastFrameProvValue:  "last_frame",
	}
	model := &catalog.Model{Params: []params.Definition{{FlagID: params.FlagTypeDuration, ParamID: "duration"}}}
	mediaInputs := []media.Input{
		{URL: "https://media.example/reference.png", MIME: "image/png"},
		{URL: "https://media.example/continuation.mp4", MIME: "video/mp4"},
		{URL: "https://media.example/open.jpg", MIME: "image/jpeg", FrameAnchor: media.FrameFirst},
		{URL: "https://media.example/close.webp", MIME: "image/webp", FrameAnchor: media.FrameLast},
	}

	body, err := getVideoJobBody(api, model, "prompt", params.Values{params.FlagTypeDuration: 8}, mediaInputs)
	if err != nil {
		t.Fatalf("💣 videoJobBody: %v", err)
	}

	wantReferences := []any{
		map[string]any{"type": "image_url", "image_url": map[string]string{"url": mediaInputs[0].URL}},
		map[string]any{"type": "video_url", "video_url": map[string]string{"url": mediaInputs[1].URL}},
	}
	if !reflect.DeepEqual(body["input_references"], wantReferences) {
		t.Errorf("✗ input_references = %#v, want %#v", body["input_references"], wantReferences)
	}

	wantFrames := []any{
		map[string]any{"type": "image_url", "image_url": map[string]string{"url": mediaInputs[2].URL}, "frame_type": "first_frame"},
		map[string]any{"type": "image_url", "image_url": map[string]string{"url": mediaInputs[3].URL}, "frame_type": "last_frame"},
	}
	if !reflect.DeepEqual(body["frame_images"], wantFrames) {
		t.Errorf("✗ frame_images = %#v, want %#v", body["frame_images"], wantFrames)
	}
}

// TestVideoPreparationRequestIsolation verifies invariant #2: Submitted request ownership.
//
// What is being tested:
// Given a PNG reference and an empty caller parameter map, prepareVideoInputs with resizing enabled
// must succeed without adding parameters to that map or changing the caller's original image bytes.
//
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
func TestVideoPreparationRequestIsolation(t *testing.T) {
	original := pngRef(t, 640, 360)
	originalBytes := bytes.Clone(original.Bytes)
	request := &generation.Generation{ProvModelPair: catalog.ProvModelPair{Model: conformVideoModel(t)}, Preparation: generation.Preparation{Params: params.Values{}, InputMedia: []media.Input{original}}, APIKey: os.Getenv((catalog.ProvModelPair{Model: conformVideoModel(t)}).Provider.APIKeyEnvVar)}

	_, err := prepareVideoInputs(context.Background(), &catalog.VideoAPI{InputMediaPayloadType: catalog.InputMediaPayloadForm, InputMediaMustResize: true}, request)
	if err != nil {
		t.Errorf("✗ preparation failed: %v", err)
	}

	if len(request.Params) != 0 {
		t.Errorf("✗ preparation mutated request parameters: %v", request.Params)
	}

	if !bytes.Equal(request.InputMedia[0].Bytes, originalBytes) {
		t.Error("✗ preparation replaced caller bytes")
	}

	if !t.Failed() {
		t.Log("✓ submission preparation preserves the caller request")
	}
}

// TestVidLifecycle verifies invariant #3: Video lifecycle.
//
// What is being tested:
// Given the three video API layouts, submitVideo must POST and poll with authentication at the
// declared routes, apply the expected authentication to the content or returned download URL, and
// return one file-backed MP4 containing the served bytes.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidLifecycle(t *testing.T) {
	video := []byte("VIDEO-CONTENT-BYTES")

	cases := []struct {
		name, layout, id  string
		pollBody          string
		startPath, dlPath string
		dlAuth            string
	}{
		{
			"content endpoint authenticated", contentEndpointVideo, "vid-content-1",
			`{"status":"completed"}`, "/videos", "/videos/vid-content-1/content", "Bearer k",
		},
		{
			"bare returned url", bareURLVideo, "vid-bare-1",
			`{"status":"done","video":{"url":"http://%HOST%/dl/bare"}}`, "/videos/generations", "/dl/bare", "",
		},
		{
			"same-origin url authenticated", sameOriginVideo, "vid-origin-1",
			`{"status":"completed","unsigned_urls":["http://%HOST%/dl/origin"]}`, "/videos", "/dl/origin", "Bearer k",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := fixtureProvider(t, c.layout)
			m := fixtureModel(t, d)
			recorder, srv := vidSrv(t, d.Config.VideoAPI.JobIDField, c.id, c.pollBody, video)

			artifacts, err := vidGo(t, d, m, vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
			if err != nil {
				t.Errorf("✗ submit failed: %v", err)

				return
			}

			vidFile(t, c.name, artifacts, video)

			start := recorder.byPath(t, c.startPath)
			if start.reqMethod != http.MethodPost || start.header.Get("Authorization") != "Bearer k" {
				t.Errorf("✗ start = %s with auth %q, want an authenticated POST at %s", start.reqMethod, start.header.Get("Authorization"), c.startPath)
			}

			poll := recorder.byPath(t, "/videos/"+c.id)
			if poll.reqMethod != http.MethodGet || poll.header.Get("Authorization") != "Bearer k" {
				t.Errorf("✗ poll = %s with auth %q, want an authenticated GET at /videos/%s", poll.reqMethod, poll.header.Get("Authorization"), c.id)
			}

			dl := recorder.byPath(t, c.dlPath)
			if got := dl.header.Get("Authorization"); got != c.dlAuth {
				t.Errorf("✗ download auth = %q, want %q at %s", got, c.dlAuth, c.dlPath)
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the three fixture layouts drive start, poll, and download routes with the expected credential behavior")
	}
}

// TestVidDuration verifies invariant #4: Video duration.
//
// What is being tested:
// Given duration 4, submitVideo must send exactly model, prompt, and the declared duration field in
// its JSON start request: the string "4" under seconds or the number 4 under duration.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidDuration(t *testing.T) {
	video := []byte("V")
	gp := params.Values{params.FlagTypeDuration: 4}

	t.Run("seconds sent as a string", func(t *testing.T) {
		d := fixtureProvider(t, contentEndpointVideo)
		recorder, srv := vidSrv(t, "id", "vid-d1", `{"status":"completed"}`, video)

		artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), gp, nil)
		if err != nil {
			t.Errorf("✗ submit failed: %v", err)

			return
		}

		for _, a := range artifacts {
			rmArtifact(t, a)
		}

		body := jsonKeys(t, recorder.byPath(t, "/videos").body, "model", "prompt", "seconds")
		if body["seconds"] != "4" {
			t.Errorf("✗ seconds = %#v, want the string \"4\"", body["seconds"])
		}

		if !t.Failed() {
			t.Log("✓ seconds sent as a string")
		}
	})
	t.Run("duration sent as an integer", func(t *testing.T) {
		d := fixtureProvider(t, bareURLVideo)
		recorder, srv := vidSrv(t, "request_id", "vid-d2", `{"status":"done","video":{"url":"http://%HOST%/dl/d2"}}`, video)

		artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), gp, nil)
		if err != nil {
			t.Errorf("✗ submit failed: %v", err)

			return
		}

		for _, a := range artifacts {
			rmArtifact(t, a)
		}

		body := jsonKeys(t, recorder.byPath(t, "/videos/generations").body, "model", "prompt", "duration")
		if body["duration"] != float64(4) {
			t.Errorf("✗ duration = %#v, want the integer 4", body["duration"])
		}

		if !t.Failed() {
			t.Log("✓ duration sent as an integer")
		}
	})

	if !t.Failed() {
		t.Log("✓ duration is sent under each description's key and type: \"seconds\":\"4\" and \"duration\":4")
	}
}

// TestVidRefEncode verifies invariant #5: Video reference encoding.
//
// What is being tested:
// Given no reference or one image reference, submitVideo must omit the reference field or encode it
// under the configured field as a single image_url object or nested reference list, with no extra
// request keys.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidRefEncode(t *testing.T) {
	refs := makeInputMedia(t, 1)

	const (
		bareURLPoll    = `{"status":"done","video":{"url":"http://%HOST%/dl/r1"}}`
		sameOriginPoll = `{"status":"completed","unsigned_urls":["http://%HOST%/dl/r2"]}`
	)

	refStart(t, bareURLVideo, "vid-r1", bareURLPoll, "/videos/generations", "image", nil)

	body := refStart(t, bareURLVideo, "vid-r1", bareURLPoll, "/videos/generations", "image", refs)
	if body != nil && !reflect.DeepEqual(body["image"], inputMediaObject(t, refs[0], false)) {
		t.Errorf("✗ image = %#v, want the single image_url object", body["image"])
	}

	refStart(t, sameOriginVideo, "vid-r2", sameOriginPoll, "/videos", "input_references", nil)

	body = refStart(t, sameOriginVideo, "vid-r2", sameOriginPoll, "/videos", "input_references", refs)
	if body != nil && !reflect.DeepEqual(body["input_references"], inputMediaList(t, refs, true)) {
		t.Errorf("✗ input_references = %#v, want the nested reference list", body["input_references"])
	}

	if !t.Failed() {
		t.Log("✓ video references are sent under each description's own provider parameter and style, absent at zero references")
	}
}

// TestVidSubmitRun verifies invariant #6: Video submission lifecycle.
//
// What is being tested:
// After adjusting and conforming a 1600x1000 image, submitVideo must POST one 1280x720 PNG with
// size 1280x720 and return one file-backed artifact with an extension. The preparation must retain
// Conformed and Derived records; moving the artifact to the output fixture must preserve the served
// video bytes and remove the temporary file.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidSubmitRun(t *testing.T) {
	video := []byte("FIXTURE-VIDEO-PAYLOAD")
	d := fixtureProvider(t, contentEndpointVideo)
	m := fixtureModel(t, d)
	recorder, srv := vidSrv(t, "id", "vid-run-1", `{"status":"completed"}`, video)
	api := vidAPI(t, d, srv.URL+"/videos")

	// Follow parameter adjustment with media conformance, submission, and artifact writing.
	inputs := []media.Input{pngRef(t, 1600, 1000)}

	prepared, adjustmentErr := generation.AdjustGeneration(&m, params.FlagInputs{}, inputs)
	gp, records := prepared.Params, prepared.Changes

	if adjustmentErr != nil {
		t.Fatalf("💣 parameter adjustment failed; cannot conform a nil parameter map: %v", adjustmentErr)
	}

	keptImages := prepared.InputMedia

	conformRecords, err := generation.ConformInputMedia(&m, gp, keptImages)
	if err != nil {
		t.Fatalf("💣 the resize decision failed: %v", err)
	}

	records = append(records, conformRecords...)
	res := generation.Result{}
	run := imgReq(t, d, m, "a prompt", gp, keptImages)

	res.Artifacts, err = submitVideo(t.Context(), &api, "fixture-key", &run)
	if err != nil {
		t.Fatalf("💣 run failed: %v", err)
	}

	if len(res.Artifacts) != 1 || res.Artifacts[0].TmpPath == "" {
		t.Fatalf("💣 artifacts = %+v, want one file-backed artifact", res.Artifacts)
	}

	checkCarriage(t, "converted run", res.Artifacts[0])
	checkRunStart(t, recorder.byPath(t, "/videos"))
	checkRunChanges(t, records, gp)
	checkRunWrite(t, res, video)

	if !t.Failed() {
		t.Log("✓ the converted order transmits the fitted reference and delivers a consumable file-backed video with the decision's records")
	}
}

// TestVidPollPace verifies invariant #7: Video polling interval.
//
// What is being tested:
// Given one working response followed by completion and PollInterval set to one second, submitVideo
// must succeed and separate the first two polls by at least one second and less than five seconds.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidPollPace(t *testing.T) {
	var (
		mu        sync.Mutex
		pollTimes []time.Time
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			fmt.Fprint(w, `{"id":"job-pace"}`)
		case strings.HasSuffix(r.URL.Path, "/content"):
			_, _ = w.Write([]byte("V"))
		default:
			mu.Lock()

			pollTimes = append(pollTimes, time.Now())
			n := len(pollTimes)
			mu.Unlock()

			if n == 1 {
				fmt.Fprint(w, `{"status":"working"}`)

				return
			}

			fmt.Fprint(w, `{"status":"done"}`)
		}
	}))
	t.Cleanup(srv.Close)
	d := fixtureProvider(t, contentEndpointVideo)

	artifacts, err := vidGo(t, d, fixtureModel(t, d), boundsAPI(t, srv.URL+"/v", 60), params.Values{}, nil)
	if err != nil {
		t.Errorf("✗ submit failed: %v", err)

		return
	}

	for _, a := range artifacts {
		rmArtifact(t, a)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(pollTimes) < 2 {
		t.Errorf("✗ %d polls recorded, want 2 — nothing to measure", len(pollTimes))

		return
	}

	if gap := pollTimes[1].Sub(pollTimes[0]); gap < time.Second || gap >= 5*time.Second {
		t.Errorf("✗ inter-poll interval = %v, want within [1s, 5s) for PollInterval 1", gap)
	}

	if !t.Failed() {
		t.Log("✓ PollInterval 1 yields an observed inter-poll interval within [1s, 5s)")
	}
}

// TestVidURLWalks verifies invariant #8: Video URL traversal.
//
// What is being tested:
// Given a completed response without its required URL, submitVideo must return ErrResponseNoData
// containing the job ID. When the URL is optional, it must use the configured content path and
// query, send an authenticated GET, and return the served MP4 bytes.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidURLWalks(t *testing.T) {
	t.Run("required miss errors naming the id", func(t *testing.T) {
		d := fixtureProvider(t, bareURLVideo)
		_, srv := vidSrv(t, "request_id", "vid-w1", `{"status":"done"}`, nil)

		_, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
		if err == nil || !errors.Is(err, errs.ErrResponseNoData) || !strings.Contains(err.Error(), "vid-w1") {
			t.Errorf("✗ err = %v, want ErrResponseNoData naming vid-w1", err)
		}

		if !t.Failed() {
			t.Log("✓ required miss errors naming the id")
		}
	})
	t.Run("non-required miss falls back to the content endpoint", func(t *testing.T) {
		video := []byte("FALLBACK-VIDEO")
		d := fixtureProvider(t, sameOriginVideo)
		recorder, srv := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				fmt.Fprint(w, `{"id":"vid-w2"}`)
			case r.URL.RequestURI() == "/videos/vid-w2/content?index=0":
				_, _ = w.Write(video)
			case strings.HasSuffix(r.URL.Path, "/content"):
				w.WriteHeader(http.StatusNotFound)
			default:
				fmt.Fprint(w, `{"status":"completed"}`)
			}
		})

		artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
		if err != nil {
			t.Errorf("✗ submit failed: %v", err)

			return
		}

		vidFile(t, "fallback", artifacts, video)

		fb := recorder.byPath(t, "/videos/vid-w2/content")
		if fb.reqMethod != http.MethodGet {
			t.Errorf("✗ the content-endpoint fallback was not requested")
		}

		if got := fb.header.Get("Authorization"); got != "Bearer k" {
			t.Errorf("✗ fallback content-endpoint auth = %q, want the bearer credential (AuthDownload on the provider's own endpoint)", got)
		}

		if !t.Failed() {
			t.Log("✓ non-required miss falls back to the content endpoint")
		}
	})

	if !t.Failed() {
		t.Log("✓ a required traversal miss errors naming the id; a non-required miss falls back to URLContentPath verbatim")
	}
}

// TestVidOriginAuth verifies invariant #9: Origin-scoped video authentication.
//
// What is being tested:
// Given a cross-origin result URL or a download redirect to another origin, submitVideo must fetch
// the served MP4 bytes without sending Authorization to the destination server.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidOriginAuth(t *testing.T) {
	video := []byte("CROSS-ORIGIN-VIDEO")

	t.Run("cross-origin returned url bare", func(t *testing.T) {
		farRec, far := recSrv(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(video) })
		d := fixtureProvider(t, sameOriginVideo)
		pollBody := fmt.Sprintf(`{"status":"completed","unsigned_urls":[%q]}`, far.URL+"/dl/x")
		_, srv := vidSrv(t, "id", "vid-o1", pollBody, nil)

		artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
		if err != nil {
			t.Errorf("✗ submit failed: %v", err)

			return
		}

		vidFile(t, "cross-origin", artifacts, video)

		if got := farRec.byPath(t, "/dl/x").header.Get("Authorization"); got != "" {
			t.Errorf("✗ the cross-origin host observed the credential %q — it must be fetched bare", got)
		}

		if !t.Failed() {
			t.Log("✓ cross-origin returned url bare")
		}
	})
	t.Run("cross-origin redirect target sees no credential", func(t *testing.T) {
		farRec, far := recSrv(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(video) })
		d := fixtureProvider(t, contentEndpointVideo)
		_, srv := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				fmt.Fprint(w, `{"id":"vid-o2"}`)
			case strings.HasSuffix(r.URL.Path, "/content"):
				http.Redirect(w, r, far.URL+"/land/v", http.StatusFound)
			default:
				fmt.Fprint(w, `{"status":"completed"}`)
			}
		})

		artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, nil)
		if err != nil {
			t.Errorf("✗ submit failed: %v", err)

			return
		}

		vidFile(t, "redirect", artifacts, video)

		land := farRec.byPath(t, "/land/v")
		if got := land.header.Get("Authorization"); got != "" {
			t.Errorf("✗ the redirect target observed the credential %q", got)
		}

		if !t.Failed() {
			t.Log("✓ cross-origin redirect target sees no credential")
		}
	})

	if !t.Failed() {
		t.Log("✓ no credential leaves the provider's API origin on returned URLs or redirects")
	}
}

// TestWalkTable verifies invariant #10: URL traversal.
//
// What is being tested:
// Given the listed object and array paths, walkURL must return the expected terminal string.
// Invalid JSON, absent paths, wrong container types, invalid indices, and non-string terminal
// values must return an empty string.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWalkTable(t *testing.T) {
	cases := []struct {
		name string
		body string
		segs []string
		want string
	}{
		{"object traversal", `{"video":{"url":"u1"}}`, []string{"video", "url"}, "u1"},
		{"list first element", `{"unsigned_urls":["a","b"]}`, []string{"unsigned_urls", "0"}, "a"},
		{"list second", `{"unsigned_urls":["a","b"]}`, []string{"unsigned_urls", "1"}, "b"},
		{"digits segment on an object", `{"0":"x"}`, []string{"0"}, ""},
		{"non-digit segment on a list", `{"list":["a"]}`, []string{"list", "x"}, ""},
		{"index out of range", `{"list":["a"]}`, []string{"list", "2"}, ""},
		{"empty segment", `{"list":["a"]}`, []string{"list", ""}, ""},
		{"non-string terminal", `{"video":{"url":123}}`, []string{"video", "url"}, ""},
		{"scalar mid-traversal", `{"video":5}`, []string{"video", "url"}, ""},
		{"not json", `not json`, []string{"a"}, ""},
		{"no segments", `{"a":"b"}`, nil, ""},
	}
	for _, c := range cases {
		if got := walkURL([]byte(c.body), c.segs); got != c.want {
			t.Errorf("✗ %s: walkURL = %q, want %q", c.name, got, c.want)
		}
	}

	if !t.Failed() {
		t.Log("✓ the URLPathSeq traversal yields the declared string and misses on every type or bounds mismatch")
	}
}

// TestVidNoDurKey verifies invariant #11: Omitted video duration field.
//
// What is being tested:
// Given duration 4 but no duration ParamID in the model, submitVideo must complete successfully and
// send only model and prompt in the JSON start body.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidNoDurKey(t *testing.T) {
	gp := params.Values{params.FlagTypeDuration: 4}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			// The exact key set: a duration must not leak under ANY key when the
			// description declares no duration key.
			body, _ := io.ReadAll(r.Body)
			jsonKeys(t, body, "model", "prompt")
			fmt.Fprint(w, `{"id":"vid-nd"}`)
		case strings.HasSuffix(r.URL.Path, "/content"):
			_, _ = w.Write([]byte("V"))
		default:
			fmt.Fprint(w, `{"status":"done"}`)
		}
	}))
	t.Cleanup(srv.Close)
	api := boundsAPI(t, srv.URL+"/v", 60)
	d := fixtureProvider(t, contentEndpointVideo)
	// Remove the model's duration parameter name so the request has no duration field.
	model := fixtureModel(t, d)
	for i := range model.Params {
		if model.Params[i].FlagID == params.FlagTypeDuration {
			model.Params[i].ParamID = ""
		}
	}

	artifacts, err := vidGo(t, d, model, api, gp, nil)
	if err != nil {
		t.Errorf("✗ submit failed: %v", err)

		return
	}

	for _, a := range artifacts {
		rmArtifact(t, a)
	}

	if !t.Failed() {
		t.Log("✓ a sent duration stays out of the request when the model declares no duration wire key")
	}
}

// TestVidFormDur verifies invariant #12: Multipart video duration.
//
// What is being tested:
// Given duration 4 and an image reference that selects multipart submission, submitVideo must
// succeed and send exactly one seconds field with value "4".
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidFormDur(t *testing.T) {
	gp := params.Values{params.FlagTypeDuration: 4, params.FlagTypeSize: "1280x720"}
	d := fixtureProvider(t, contentEndpointVideo)
	recorder, srv := vidSrv(t, "id", "vid-fd", `{"status":"completed"}`, []byte("V"))

	artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), gp, []media.Input{pngRef(t, 1600, 1000)})
	if err != nil {
		t.Errorf("✗ submit failed: %v", err)

		return
	}

	for _, a := range artifacts {
		rmArtifact(t, a)
	}

	form := parseForm(t, recorder.byPath(t, "/videos"))
	if got := form.Value["seconds"]; len(got) != 1 || got[0] != "4" {
		t.Errorf("✗ form seconds = %v, want the stringified 4 on the multipart start", got)
	}

	if !t.Failed() {
		t.Log("✓ the duration is sent in the multipart start as its stringified form field")
	}
}

// TestVidFormURLReference verifies invariant #13: Video form URL reference.
//
// What is being tested:
// Given a URL image reference and empty caller parameters, submitVideo must download the image once
// without Authorization, upload one resized 1280x720 file with size 1280x720, and leave the caller
// parameter map empty.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidFormURLReference(t *testing.T) {
	sourceImage := pngRef(t, 1600, 1000)
	mediaRecorder, mediaServer := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(sourceImage.Bytes)
	})
	mediaInput := media.Input{
		URL:  mediaServer.URL + "/video-reference.png",
		MIME: "image/png",
	}

	d := fixtureProvider(t, contentEndpointVideo)
	recorder, lifecycleServer := vidSrv(t, "id", "vid-url", `{"status":"completed"}`, []byte("V"))
	parameterValues := params.Values{}

	artifacts, err := vidGo(
		t,
		d,
		fixtureModel(t, d),
		vidAPI(t, d, lifecycleServer.URL+"/videos"),
		parameterValues,
		[]media.Input{mediaInput},
	)
	if err != nil {
		t.Errorf("✗ submit failed: %v", err)

		return
	}

	for _, generatedMedia := range artifacts {
		rmArtifact(t, generatedMedia)
	}

	form := parseForm(t, recorder.byPath(t, "/videos"))

	parts := form.File["input_reference"]
	if len(parts) != 1 {
		t.Fatalf("💣 %d input-reference parts, want 1", len(parts))
	}

	decodedImage, _, decodeErr := image.Decode(bytes.NewReader(partBytes(t, parts[0])))
	if decodeErr != nil {
		t.Errorf("✗ decode uploaded reference: %v", decodeErr)
	} else if bounds := decodedImage.Bounds(); bounds.Dx() != 1280 || bounds.Dy() != 720 {
		t.Errorf("✗ uploaded reference bounds = %dx%d, want 1280x720", bounds.Dx(), bounds.Dy())
	}

	if submittedSize := form.Value["size"]; len(submittedSize) != 1 || submittedSize[0] != "1280x720" {
		t.Errorf("✗ adopted request size = %v, want 1280x720", submittedSize)
	}

	if len(parameterValues) != 0 {
		t.Errorf("✗ submission changed caller parameters: %v", parameterValues)
	}

	if mediaRecorder.count() != 1 {
		t.Errorf("✗ media download count = %d, want 1", mediaRecorder.count())
	} else if authorization := mediaRecorder.last(t).header.Get("Authorization"); authorization != "" {
		t.Errorf("✗ media download carried Authorization %q, want none", authorization)
	}

	if !t.Failed() {
		t.Log("✓ a URL video reference is downloaded bare, resized, and uploaded as a multipart file")
	}
}

// TestVidWireParams verifies invariant #14: Video wire params.
//
// What is being tested:
// Given aspect 16:9 and resolution 480p, submitVideo must send exactly model, prompt, aspect_ratio,
// and resolution in its JSON start body, preserving both parameter values.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestVidWireParams(t *testing.T) {
	gp := params.Values{params.FlagTypeAspect: "16:9", params.FlagTypeResolution: "480p"}
	d := fixtureProvider(t, bareURLVideo)
	recorder, srv := vidSrv(t, "request_id", "vid-wp", `{"status":"done","video":{"url":"http://%HOST%/dl/wp"}}`, []byte("V"))

	artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), gp, nil)
	if err != nil {
		t.Errorf("✗ submit failed: %v", err)

		return
	}

	for _, a := range artifacts {
		rmArtifact(t, a)
	}

	body := jsonKeys(t, recorder.byPath(t, "/videos/generations").body, "model", "prompt", "aspect_ratio", "resolution")
	if body["aspect_ratio"] != "16:9" || body["resolution"] != "480p" {
		t.Errorf("✗ request params = %v/%v, want 16:9 and 480p under their provider parameter names", body["aspect_ratio"], body["resolution"])
	}

	if !t.Failed() {
		t.Log("✓ aspect and resolution are sent in the JSON video start under their provider parameter names")
	}
}

// TestVideoPromptConsumption verifies invariant #15: Video prompt consumption.
//
// What is being tested:
// Given empty and nonempty prompts, getVideoJobBody and videoJobFields must omit prompt for
// PromptIgnored models and otherwise include its exact supplied value.
//
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
func TestVideoPromptConsumption(t *testing.T) {
	for _, ignored := range []bool{true, false} {
		for _, prompt := range []string{"", "a drifting boat"} {
			model := &catalog.Model{ID: "video", Media: media.Video, PromptIgnored: ignored}

			body, err := getVideoJobBody(&catalog.VideoAPI{}, model, prompt, nil, nil)
			if err != nil {
				t.Errorf("✗ JSON construction failed: %v", err)

				continue
			}

			jsonPrompt, jsonPresent := body["prompt"]

			formPrompt, formPresent := videoJobFields(model, prompt, nil)["prompt"]
			if jsonPresent == ignored || formPresent == ignored {
				t.Errorf("✗ prompt presence: ignored=%v JSON=%v form=%v", ignored, body, videoJobFields(model, prompt, nil))
			}

			if !ignored && (jsonPrompt != prompt || formPrompt != prompt) {
				t.Errorf("✗ consumed prompt changed: JSON=%v form=%q", jsonPrompt, formPrompt)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ both video encodings honor model prompt consumption")
	}
}

// TestLoadComposedRun verifies invariant #16: Loaded provider request lifecycle.
//
// What is being tested:
// Given a video provider loaded from JSON and duration 5 adjusted to 4, submitVideo must succeed,
// return one artifact, and send exactly model, prompt, and numeric seconds 4 in the start body.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLoadComposedRun(t *testing.T) {
	t.Setenv("LOADFLOW_KEY", "k")

	var srvURL string

	recorder, srv := recSrv(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			fmt.Fprint(w, `{"id":"j1"}`)
		case r.URL.Path == "/media":
			w.Header().Set("Content-Type", "video/mp4")
			fmt.Fprint(w, "VID")
		default:
			fmt.Fprintf(w, `{"status":"done","url":%q}`, srvURL+"/media")
		}
	})
	srvURL = srv.URL
	provCfgJSON := fmt.Sprintf(`{
		"id": "loadflow", "displayName": "Load Flow", "apiKeyEnvVar": "LOADFLOW_KEY",
		"models": [{"id": "vid-x", "name": "Fixture Video X", "media": "video", "params": [
			{"paramID": "seconds", "flagID": "duration", "allowedValues": ["4", "8"]}
		]}],
		"config": {
		"videoAPI": {
			"asyncJobsURL": %q, "jobIDField": "id",
			"progressStatusText": ["pending"], "completedStatusText": "done", "failedStatusText": ["failed"],
			"urlPathSeq": ["url"],
			"inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
			"fallbackExt": ".mp4", "pollInterval": 1, "pollTimeout": 5
		}
		}
}`, srv.URL)

	provCfg := loadTestProvider(t, "loadflow", []byte(provCfgJSON))

	gp, _, adjustmentErr := params.Adjust(params.FlagInputs{params.FlagTypeDuration: 5}, provCfg.Models[0].Params, provCfg.Models[0].ID)
	if adjustmentErr != nil {
		t.Errorf("✗ parameter adjustment failed: %v", adjustmentErr)
	}

	res := generation.Result{}
	run := generation.Generation{
		ProvModelPair: catalog.ProvModelPair{Provider: provCfg.Identity(), Model: provCfg.Models[0]},
		Prompt:        "p",
		Preparation:   generation.Preparation{Params: gp},
	}

	artifacts, err := submitVideo(t.Context(), provCfg.Config.VideoAPI, "k", &run)
	res.Artifacts = artifacts

	if err != nil {
		t.Fatalf("💣 the loaded provider listing's run failed: %v", err)
	}

	for _, a := range res.Artifacts {
		rmArtifact(t, a)
	}

	start := jsonKeys(t, recorder.byPath(t, "/").body, "model", "prompt", "seconds")
	if v, ok := start["seconds"].(float64); !ok || v != 4 {
		t.Errorf("✗ the start body's seconds = %v, want the adjusted 4 under the declared duration parameter name", start["seconds"])
	}

	if len(res.Artifacts) != 1 {
		t.Errorf("✗ %d artifacts, want the one downloaded video", len(res.Artifacts))
	}

	if !t.Failed() {
		t.Log("✓ a Load-composed provider listing runs its declared lifecycle with the duration parameter name carried into the request and into the sent report")
	}
}

// byPath returns the first captured request with the given path; fatal when none arrived, since no
// request assertion after it could run.
func (c *capture) byPath(t *testing.T, reqPath string) recReq {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.reqs {
		if r.reqPath == reqPath {
			return r
		}
	}

	t.Fatalf("💣 no request for %s reached the harness — nothing to inspect", reqPath)

	return recReq{}
}

// rmArtifact removes an artifact's temporary file if present.
func rmArtifact(t *testing.T, generatedMedia artifact.Media) {
	t.Helper()

	if generatedMedia.TmpPath != "" {
		_ = os.Remove(generatedMedia.TmpPath)
	}
}

// checkCarriage checks that an artifact has inline bytes or a temporary file, exclusively, and a
// nonempty extension.
func checkCarriage(t *testing.T, tag string, generatedMedia artifact.Media) {
	t.Helper()

	if (len(generatedMedia.Data) > 0) == (generatedMedia.TmpPath != "") {
		t.Errorf("✗ %s: artifact carriage not exclusive: %d inline bytes, SrcPath %q", tag, len(generatedMedia.Data), generatedMedia.TmpPath)
	}

	if generatedMedia.FileExt == "" {
		t.Errorf("✗ %s: artifact has no extension", tag)
	}
}

// conformVideoModel returns the video-model fixture the conformance tests fit references to: two
// fixed sizes, one landscape and one portrait, and a single reference.
func conformVideoModel(t *testing.T) catalog.Model {
	t.Helper()

	return catalog.Model{
		ID: "restricted-size-video", Media: media.Video,
		Params: []params.Definition{
			{FlagID: params.FlagTypeAspect, ParamID: "aspect_ratio"},
			{FlagID: params.FlagTypeResolution, ParamID: "resolution"},
			{FlagID: params.FlagTypeDuration, ParamID: "seconds", AllowedValues: []string{"4", "8", "12", "16", "20"}},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 1},
			{FlagID: params.FlagTypeSize, ParamID: "size", AllowedValues: []string{"1280x720", "720x1280"}},
		},
	}
}

// vidAPI copies a provider config video description with its job endpoint pointed at the harness.
func vidAPI(t *testing.T, d catalog.Provider, jobsURL string) catalog.VideoAPI {
	t.Helper()

	if d.Config.VideoAPI == nil {
		t.Fatalf("💣 provider config has no VideoAPI")
	}

	api := *d.Config.VideoAPI
	api.AsyncJobsURL = jobsURL

	return api
}

// vidSrv creates a recording lifecycle harness: a POST answers the id body, a path under /dl/ or
// ending in /content serves the video bytes, and anything else serves pollBody with %HOST% replaced
// by the harness host (so returned URLs are absolute and same-origin).
func vidSrv(t *testing.T, idField, id, pollBody string, video []byte) (*capture, *httptest.Server) {
	t.Helper()

	idBody := fmt.Sprintf(`{%q:%q}`, idField, id)

	return recSrv(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			fmt.Fprint(w, idBody)
		case strings.HasPrefix(r.URL.Path, "/dl/") || strings.HasSuffix(r.URL.Path, "/content"):
			_, _ = w.Write(video)
		default:
			//nolint:gosec // pollBody and r.Host are controlled by this test server.
			fmt.Fprint(w, strings.ReplaceAll(pollBody, "%HOST%", r.Host))
		}
	})
}

// vidGo submits a video request using the supplied local API settings and returns its artifacts.
func vidGo(t *testing.T, d catalog.Provider, m catalog.Model, api catalog.VideoAPI, gp params.Values, inputs []media.Input) ([]artifact.Media, error) {
	t.Helper()

	run := imgReq(t, d, m, "a prompt", gp, inputs)

	return submitVideo(t.Context(), &api, "k", &run, d.Config.StringParams...)
}

// vidFile checks that one file-backed MP4 artifact contains the served video bytes, then removes
// it.
func vidFile(t *testing.T, tag string, artifacts []artifact.Media, video []byte) {
	t.Helper()

	if len(artifacts) != 1 {
		t.Errorf("✗ %s: %d artifacts, want 1", tag, len(artifacts))

		return
	}

	generatedMedia := artifacts[0]
	defer rmArtifact(t, generatedMedia)

	checkCarriage(t, tag, generatedMedia)

	if generatedMedia.TmpPath == "" || len(generatedMedia.Data) != 0 {
		t.Errorf("✗ %s: artifact = (%d inline bytes, %q), want file-backed", tag, len(generatedMedia.Data), generatedMedia.TmpPath)

		return
	}

	if generatedMedia.FileExt != ".mp4" {
		t.Errorf("✗ %s: FileExt = %q, want .mp4", tag, generatedMedia.FileExt)
	}

	got, err := os.ReadFile(generatedMedia.TmpPath)
	if err != nil {
		t.Errorf("✗ %s: read the streamed temp file: %v", tag, err)

		return
	}

	if !bytes.Equal(got, video) {
		t.Errorf("✗ %s: streamed bytes differ from the served video (%d vs %d bytes)", tag, len(got), len(video))
	}
}

// boundsAPI returns video settings with a one-second poll interval and the supplied timeout.
func boundsAPI(test testing.TB, jobsURL string, timeoutSecs int) catalog.VideoAPI {
	test.Helper()

	return catalog.VideoAPI{
		AsyncJobsURL: jobsURL, JobIDField: "id", ProgressStatusText: []string{"working"}, CompletedStatusText: "done",
		FailedStatusText: []string{"failed"}, URLContentPath: "/content",
		InputMediaPayloadType: "json", InputMediaStyle: catalog.InputMediaSingle, FallbackExt: ".mp4",
		PollInterval: 1, PollTimeout: catalog.PollSeconds(timeoutSecs),
	}
}

// refStart drives one video start with the given references against a fresh harness and returns the
// recorded start body under the exact expected key set (the reference field present only when a
// reference was sent).
func refStart(t *testing.T, layout, id, pollBody, startPath, refField string, refs []media.Input) map[string]any {
	t.Helper()
	d := fixtureProvider(t, layout)
	recorder, srv := vidSrv(t, d.Config.VideoAPI.JobIDField, id, pollBody, []byte("V"))

	artifacts, err := vidGo(t, d, fixtureModel(t, d), vidAPI(t, d, srv.URL+"/videos"), params.Values{}, refs)
	if err != nil {
		t.Errorf("✗ submit with %d refs failed: %v", len(refs), err)

		return nil
	}

	for _, a := range artifacts {
		rmArtifact(t, a)
	}

	keys := []string{"model", "prompt"}
	if len(refs) > 0 {
		keys = append(keys, refField)
	}

	return jsonKeys(t, recorder.byPath(t, startPath).body, keys...)
}

// checkRunChanges requires Conformed input-media and Derived size records, with 1280x720 in both
// the Derived record and the adjusted parameters.
func checkRunChanges(t *testing.T, records []params.Adjustment, gp params.Values) {
	t.Helper()

	if c := pickKind(t, records, params.ChangeConformed); c == nil || c.FlagID != params.FlagTypeInputMedia {
		t.Errorf("✗ records = %+v, want a Conformed input-media record", records)
	}

	if c := pickKind(t, records, params.ChangeDerived); c == nil || c.FlagID != params.FlagTypeSize || c.WireVal != "1280x720" {
		t.Errorf("✗ records = %+v, want the adopted size as a Derived record", records)
	}

	sentSize, _ := gp[params.FlagTypeSize].(string)
	if sentSize != "1280x720" {
		t.Errorf("✗ Sent size = %q, want the adopted 1280x720 applied to the sent values", sentSize)
	}
}

// writeRunArtifact claims an output path, moves the artifact there, and returns its stem. It
// reproduces file-backed output here because importing output would create a cycle.
func writeRunArtifact(t *testing.T, dir string, generatedMedia artifact.Media) string {
	t.Helper()

	stem := "bild-image-01"
	dst := filepath.Join(dir, stem+generatedMedia.FileExt)

	// #nosec G302 G304 -- dst is test-owned; the helper mirrors user-readable artifact permissions.
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		t.Fatalf("💣 claim the artifact path: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("💣 close the claimed path: %v", err)
	}

	if err := os.Rename(generatedMedia.TmpPath, dst); err != nil {
		t.Fatalf("💣 move the artifact into place: %v", err)
	}

	return stem
}

// checkRunWrite moves the artifact through writeRunArtifact, checks the saved video bytes, and
// requires the temporary file to be gone.
func checkRunWrite(t *testing.T, res generation.Result, video []byte) {
	t.Helper()
	outDir := t.TempDir()
	stem := writeRunArtifact(t, outDir, res.Artifacts[0])

	// #nosec G304 -- outDir is test-owned and stem is the returned output name.
	saved, err := os.ReadFile(filepath.Join(outDir, stem+".mp4"))
	if err != nil {
		t.Fatalf("💣 read the written artifact: %v", err)
	}

	if !bytes.Equal(saved, video) {
		t.Errorf("✗ written bytes differ from the served video")
	}

	if _, err := os.Stat(res.Artifacts[0].TmpPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("✗ the streamed temp file survived WriteArtifacts: %v", err)
	}
}

// checkRunStart parses the recorded multipart start request as form data and asserts the fitted
// reference was actually sent: exactly one input_reference file part whose content decodes as the
// conformed 1280x720 PNG, beside the adopted size form field.
func checkRunStart(t *testing.T, start recReq) {
	t.Helper()

	if start.reqMethod != http.MethodPost {
		t.Errorf("✗ start = %s, want a POST", start.reqMethod)
	}

	form := parseForm(t, start)
	if got := form.Value["size"]; len(got) != 1 || got[0] != "1280x720" {
		t.Errorf("✗ start form size = %v, want the adopted 1280x720 sent", got)
	}

	parts := form.File["input_reference"]
	if len(parts) != 1 {
		t.Errorf("✗ %d input_reference file parts, want 1 — the fitted reference must be sent in the start body", len(parts))

		return
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(partBytes(t, parts[0])))
	if err != nil || format != "png" || cfg.Width != 1280 || cfg.Height != 720 {
		t.Errorf("✗ input_reference part = %s %dx%d (%v), want the conformed png 1280x720", format, cfg.Width, cfg.Height, err)
	}
}

// pngRef creates a decodable in-memory PNG reference image of the given dimensions.
func pngRef(t *testing.T, w, h int) media.Input {
	t.Helper()

	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("💣 encode fixture PNG: %v", err)
	}

	return media.Input{Bytes: b.Bytes(), MIME: "image/png", Filepath: fmt.Sprintf("/in/ref-%dx%d.png", w, h)}
}

// pickKind returns the first record of a kind, or nil.
func pickKind(test testing.TB, paramChanges []params.Adjustment, kind params.Change) *params.Adjustment {
	test.Helper()

	for i := range paramChanges {
		if paramChanges[i].Type == kind {
			return &paramChanges[i]
		}
	}

	return nil
}
