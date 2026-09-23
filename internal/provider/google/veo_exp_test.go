package google

// Invariants tested:
//  1. Observation recovery: After an unfinished operation followed by HTTP 429, 502, 503, or 504,
//     httpapi.Poll with operationProbe must issue a third GET and accept a completed video response
//     without an error.
//  2. Completed operation output: Given done=true with an empty generatedSamples array,
//     operationProbe.Poll must return false and ErrResponseNoData.
//  3. Veo duration: For arbitrary duration values and presence, adjustVeoDuration must preserve the
//     duration and return no records when there are no references and resolution is neither 1080p
//     nor 4k. Otherwise, it must set duration eight and return one Forced record with WireVal 8,
//     unless duration was already eight. It must not return an error or panic.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestGoogleOperationPollRecovery verifies invariant #1: Observation recovery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// After an unfinished operation followed by HTTP 429, 502, 503, or 504, httpapi.Poll with
// operationProbe must issue a third GET and accept a completed video response without an error.
func TestGoogleOperationPollRecovery(t *testing.T) {
	for _, statusCode := range []int{429, 502, 503, 504} {
		t.Run(strconv.Itoa(statusCode), func(t *testing.T) {
			server, observations := pollRecoveryServer(t, statusCode, `{"done":false}`, `{"done":true,"response":{"generateVideoResponse":{"generatedSamples":[{"video":{"uri":"https://result.example/video.mp4"}}]}}}`)

			poller := &operationProbe{apiBase: server.URL, name: "operations/operation-identity"}

			err := httpapi.Poll(t.Context(), time.Millisecond, time.Second, poller)
			if err != nil || *observations != 3 {
				t.Errorf("✗ completion error=%v observations=%d; want successful third observation", err, *observations)
			}

			if !t.Failed() {
				t.Log("✓ temporary observation failure recovers without resubmission")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every selected HTTP status recovers to completion")
	}
}

// TestCompletedOperationRequiresOutput verifies invariant #2: Completed operation output.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given done=true with an empty generatedSamples array, operationProbe.Poll must return false and
// ErrResponseNoData.
func TestCompletedOperationRequiresOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"done":true,"response":{"generateVideoResponse":{"generatedSamples":[]}}}`))
	}))
	defer server.Close()

	poller := &operationProbe{apiBase: server.URL, name: "operations/empty-result"}

	complete, err := poller.Poll(t.Context())
	if complete || !errors.Is(err, errs.ErrResponseNoData) {
		t.Errorf("✗ complete=%t error=%v; want terminal missing-data failure", complete, err)
	}

	if !t.Failed() {
		t.Log("✓ completed status requires usable video data")
	}
}

// FuzzVeoDuration verifies invariant #3: Veo duration.
//
// What is being tested:
// For arbitrary duration values and presence, adjustVeoDuration must preserve the duration and
// return no records when there are no references and resolution is neither 1080p nor 4k. Otherwise,
// it must set duration eight and return one Forced record with WireVal 8, unless duration was
// already eight. It must not return an error or panic.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzVeoDuration(f *testing.F) {
	for _, n := range []int{4, 6, 8, 0, -3, 30} {
		f.Add(n, true, uint8(0), uint8(0))
	}

	f.Add(5, false, uint8(1), uint8(2))
	f.Fuzz(func(t *testing.T, sec int, durSet bool, nRefs, resPick uint8) {
		gp := params.Values{}
		if durSet {
			gp[params.FlagTypeDuration] = sec
		}

		res := []string{"", "720p", "1080p", "4k"}[int(resPick)%4]
		if res != "" {
			gp[params.FlagTypeResolution] = res
		}

		inputs := make([]media.Input, int(nRefs)%4)
		for i := range inputs {
			inputs[i] = fixtureInputMedia(t, "R", "/tmp/r.png")
		}

		chs, adjustErr := adjustVeoDuration(gp, inputs)
		got := gp

		if adjustErr != nil {
			t.Errorf("✗ adjustParams returned %v, want no error", adjustErr)
		}

		checkFuzzAdjust(t, got, chs, sec, durSet, len(inputs) > 0 || res == "1080p" || res == "4k")
	})
}

// checkFuzzAdjust verifies duration presence, value, and Forced records for one combination of
// duration and forcing conditions.
func checkFuzzAdjust(t *testing.T, got params.Values, chs []params.Adjustment, sec int, durSet, trigger bool) {
	t.Helper()

	adjustedSecs, durationSent := got[params.FlagTypeDuration]
	if !trigger {
		if durationSent != durSet || (durSet && adjustedSecs != sec) || len(chs) != 0 {
			t.Errorf("✗ no trigger but the result changed: dur %v sent %v records %+v", adjustedSecs, durationSent, chs)
		}
	} else {
		if !durationSent || adjustedSecs != 8 {
			t.Errorf("✗ trigger held but duration = %v (sent %v), want a sent 8", adjustedSecs, durationSent)
		}

		if durSet && sec == 8 {
			if len(chs) != 0 {
				t.Errorf("✗ a matched value recorded: %+v", chs)
			}
		} else if len(chs) != 1 || chs[0].Type != params.ChangeForced || chs[0].WireVal != "8" {
			t.Errorf("✗ records = %+v, want one Forced to 8", chs)
		}
	}

	if !t.Failed() {
		t.Log("✓ the adjuster invariant holds")
	}
}
