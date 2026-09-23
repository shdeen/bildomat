package bfl

// Invariants tested:
//  1. BFL flux 3 video requests: For FLUX.3 Video and duration eight, requestBody must select t2v
//     without media, i2v for images, and v2v for a video. It must encode untimed keyframes as
//     strings, fill missing times in timed keyframes, and send the video URL in start_video. It
//     must return ErrInputMedia for mixed images and video, and ErrInputMediaTime for inconsistent,
//     reversed, or out-of-range frame times.

import (
	"encoding/base64"
	"errors"
	"reflect"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestBFLFlux3VideoRequests verifies invariant #1: BFL flux 3 video requests.
//
// What is being tested:
// For FLUX.3 Video and duration eight, requestBody must select t2v without media, i2v for images,
// and v2v for a video. It must encode untimed keyframes as strings, fill missing times in timed
// keyframes, and send the video URL in start_video. It must return ErrInputMedia for mixed images
// and video, and ErrInputMediaTime for inconsistent, reversed, or out-of-range frame times.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestBFLFlux3VideoRequests(t *testing.T) {
	provCfg := shippedBFLCfg(t)
	model := shippedBFLModel(t, provCfg, "flux-3-video")
	parameterValues := params.Values{params.FlagTypeDuration: 8}

	tests := []struct {
		name      string
		inputs    []media.Input
		wantMode  string
		wantField string
		wantValue any
	}{
		{name: "text", wantMode: "t2v"},
		{
			name: "untimed images",
			inputs: []media.Input{
				{Bytes: []byte("OPEN"), MIME: "image/png", Filepath: "/tmp/open.png"},
				{URL: "https://media.example/close.webp", MIME: "image/webp"},
			},
			wantMode:  "i2v",
			wantField: "keyframes",
			wantValue: []any{base64.StdEncoding.EncodeToString([]byte("OPEN")), "https://media.example/close.webp"},
		},
		{
			name: "partial timed images",
			inputs: []media.Input{
				{URL: "https://media.example/open.png", MIME: "image/png", Time: new(0.0)},
				{URL: "https://media.example/close.png", MIME: "image/png"},
			},
			wantMode:  "i2v",
			wantField: "keyframes",
			wantValue: []any{[]any{float64(0), "https://media.example/open.png"}, []any{float64(8), "https://media.example/close.png"}},
		},
		{
			name: "three evenly spaced keyframes",
			inputs: []media.Input{
				{URL: "https://media.example/open.png", MIME: "image/png", Time: new(0.0)},
				{URL: "https://media.example/middle.png", MIME: "image/png"},
				{URL: "https://media.example/close.png", MIME: "image/png", Time: new(8.0)},
			},
			wantMode:  "i2v",
			wantField: "keyframes",
			wantValue: []any{
				[]any{float64(0), "https://media.example/open.png"},
				[]any{float64(4), "https://media.example/middle.png"},
				[]any{float64(8), "https://media.example/close.png"},
			},
		},
		{
			name: "untimed opening endpoint",
			inputs: []media.Input{
				{URL: "https://media.example/open.png", MIME: "image/png"},
				{URL: "https://media.example/middle.png", MIME: "image/png", Time: new(3.0)},
				{URL: "https://media.example/close.png", MIME: "image/png", Time: new(8.0)},
			},
			wantMode:  "i2v",
			wantField: "keyframes",
			wantValue: []any{
				[]any{float64(0), "https://media.example/open.png"},
				[]any{float64(3), "https://media.example/middle.png"},
				[]any{float64(8), "https://media.example/close.png"},
			},
		},
		{
			name: "untimed closing endpoint",
			inputs: []media.Input{
				{URL: "https://media.example/open.png", MIME: "image/png", Time: new(0.0)},
				{URL: "https://media.example/middle.png", MIME: "image/png", Time: new(5.0)},
				{URL: "https://media.example/close.png", MIME: "image/png"},
			},
			wantMode:  "i2v",
			wantField: "keyframes",
			wantValue: []any{
				[]any{float64(0), "https://media.example/open.png"},
				[]any{float64(5), "https://media.example/middle.png"},
				[]any{float64(8), "https://media.example/close.png"},
			},
		},
		{
			name:      "video continuation",
			inputs:    []media.Input{{URL: "https://media.example/source.mp4", MIME: "video/mp4"}},
			wantMode:  "v2v",
			wantField: "start_video",
			wantValue: "https://media.example/source.mp4",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			body, err := requestBody(&model, testPrompt, parameterValues, testCase.inputs)
			if err != nil {
				t.Fatalf("💣 requestBody: %v", err)
			}

			if body["mode"] != testCase.wantMode {
				t.Errorf("✗ mode = %#v, want %q", body["mode"], testCase.wantMode)
			}

			if body["duration"] != 8 {
				t.Errorf("✗ duration = %#v, want numeric 8", body["duration"])
			}

			if testCase.wantField != "" && !reflect.DeepEqual(body[testCase.wantField], testCase.wantValue) {
				t.Errorf("✗ %s = %#v, want %#v", testCase.wantField, body[testCase.wantField], testCase.wantValue)
			}
		})
	}

	mixedInputs := []media.Input{
		{URL: "https://media.example/open.png", MIME: "image/png"},
		{URL: "https://media.example/source.mp4", MIME: "video/mp4"},
	}
	if _, err := requestBody(&model, testPrompt, parameterValues, mixedInputs); err == nil || !errors.Is(err, errs.ErrInputMedia) {
		t.Errorf("✗ mixed image/video error = %v, want input-media rejection", err)
	}

	invalidKeyframes := [][]media.Input{
		{
			{URL: "https://media.example/open.png", MIME: "image/png", Time: new(1.0)},
			{URL: "https://media.example/middle.png", MIME: "image/png"},
			{URL: "https://media.example/close.png", MIME: "image/png", Time: new(8.0)},
		},
		{
			{URL: "https://media.example/late.png", MIME: "image/png", Time: new(8.0)},
			{URL: "https://media.example/early.png", MIME: "image/png", Time: new(0.0)},
		},
		{
			{URL: "https://media.example/open.png", MIME: "image/png", Time: new(0.0)},
			{URL: "https://media.example/late.png", MIME: "image/png", Time: new(9.0)},
		},
	}
	for invalidIndex, invalidInputs := range invalidKeyframes {
		if _, err := requestBody(&model, testPrompt, parameterValues, invalidInputs); err == nil || !errors.Is(err, errs.ErrInputMediaTime) {
			t.Errorf("✗ invalid keyframe case %d error = %v, want frame-time rejection", invalidIndex+1, err)
		}
	}
}
