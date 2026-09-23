package bfl

// Invariants tested:
//  1. Keyframe arithmetic precision: For eight frames across 29 seconds, fillKeyframeTimes must
//     preserve the assigned endpoints and interior time, then assign the remaining frames at evenly
//     spaced times. It must accept an interior time one representable float above 29/7 and return
//     ErrInputMediaTime for a 0.0001-second displacement.

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestKeyframeRounding verifies invariant #1: Keyframe arithmetic precision.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// For eight frames across 29 seconds, fillKeyframeTimes must preserve the assigned endpoints and
// interior time, then assign the remaining frames at evenly spaced times. It must accept an
// interior time one representable float above 29/7 and return ErrInputMediaTime for a 0.0001-second
// displacement.
func TestKeyframeRounding(t *testing.T) {
	for _, timeOffset := range []float64{0, math.Nextafter(29.0/7, math.Inf(1)) - 29.0/7, 0.0001} {
		t.Run(fmt.Sprint(timeOffset), func(t *testing.T) {
			times := make([]float64, 8)
			timeSet := make([]bool, 8)
			times[0], times[7] = 0, 29
			timeSet[0], timeSet[7] = true, true
			times[1], timeSet[1] = 29.0/7+timeOffset, true
			assignedInterior := times[1]

			err := fillKeyframeTimes(times, timeSet, 29)
			if timeOffset == 0.0001 {
				if !errors.Is(err, errs.ErrInputMediaTime) {
					t.Errorf("✗ inconsistent supplied time returned %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("✗ arithmetic rounding rejected: %v", err)
				}

				if times[0] != 0 || times[7] != 29 || times[1] != assignedInterior {
					t.Errorf("✗ supplied times changed: %v", times)
				}

				for frameIndex := 2; frameIndex < 7; frameIndex++ {
					expectedTime := float64(frameIndex) * (29.0 / 7)
					if !timeSet[frameIndex] || math.Abs(times[frameIndex]-expectedTime) > 2*(math.Nextafter(expectedTime, math.Inf(1))-expectedTime) {
						t.Errorf("✗ frame %d time=%v assigned=%t; want evenly spaced position %v", frameIndex, times[frameIndex], timeSet[frameIndex], expectedTime)
					}
				}
			}

			if !t.Failed() {
				t.Log("✓ interpolation distinguishes arithmetic rounding from inconsistent timing")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ assigned keyframe times survive interpolation exactly")
	}
}
