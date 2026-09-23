package bfl

import (
	"fmt"
	"math"
	"slices"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// buildKeyframes returns media strings when no frame has an explicit time. Otherwise it fills
// missing times and returns validated time/media pairs.
func buildKeyframes(imageInputs []media.Input, parameterValues params.Values) ([]any, error) {
	times, timeSet := keyframeTimes(imageInputs)
	if !slices.Contains(timeSet, true) {
		return untimedKeyframes(imageInputs), nil
	}

	durationSeconds, durationSet := parseDuration(parameterValues)
	if !durationSet || durationSeconds < 0 {
		return nil, &errs.MediaError{Problem: KeyframesNeedDuration, Cause: errs.ErrInputMediaTime}
	}

	if err := fillKeyframeTimes(times, timeSet, durationSeconds); err != nil {
		return nil, err
	}

	if err := checkKeyframeTimes(times, durationSeconds); err != nil {
		return nil, err
	}

	return timedKeyframes(imageInputs, times), nil
}

// keyframeTimes takes the image inputs and returns each input's frame time and whether the input
// carries one, in input order.
func keyframeTimes(imageInputs []media.Input) (times []float64, timeSet []bool) {
	times = make([]float64, len(imageInputs))
	timeSet = make([]bool, len(imageInputs))

	for inputIndex, mediaInput := range imageInputs {
		times[inputIndex], timeSet[inputIndex] = mediaInput.FrameTime()
	}

	return times, timeSet
}

// untimedKeyframes takes image inputs carrying no frame times and returns the keyframe array of
// their media values alone.
func untimedKeyframes(imageInputs []media.Input) []any {
	keyframes := make([]any, 0, len(imageInputs))
	for _, mediaInput := range imageInputs {
		keyframes = append(keyframes, mediaInput.URLOrBase64())
	}

	return keyframes
}

// fillKeyframeTimes updates the supplied times and presence flags. Missing endpoints become zero
// and duration; missing interior times require evenly spaced positions. Supplied times remain
// unchanged, and inconsistent positions return an error.
func fillKeyframeTimes(times []float64, timeSet []bool, durationSeconds float64) error {
	lastIndex := len(times) - 1

	if !timeSet[0] {
		times[0], timeSet[0] = 0, true
	}

	if !timeSet[lastIndex] {
		times[lastIndex], timeSet[lastIndex] = durationSeconds, true
	}

	if !slices.Contains(timeSet, false) {
		return nil
	}

	interval := durationSeconds / float64(lastIndex)
	for inputIndex := range times {
		positionTime := float64(inputIndex) * interval
		if timeSet[inputIndex] {
			// One rounding step in division and one in multiplication bound the
			// difference between equivalent evenly spaced positions.
			tolerance := float64(inputIndex)*(math.Nextafter(interval, math.Inf(1))-interval) + (math.Nextafter(positionTime, math.Inf(1)) - positionTime)
			if math.Abs(times[inputIndex]-positionTime) > tolerance {
				return &errs.MediaError{Problem: fmt.Sprintf(KeyframePositionConflict, inputIndex+1, times[inputIndex], positionTime), Cause: errs.ErrInputMediaTime}
			}

			continue
		}

		times[inputIndex], timeSet[inputIndex] = positionTime, true
	}

	return nil
}

// checkKeyframeTimes takes the frame times and the duration and fails on a time below zero, above
// the duration, or not later than the time before it.
func checkKeyframeTimes(times []float64, durationSeconds float64) error {
	previousTime := -1.0
	for inputIndex, keyframeTime := range times {
		if keyframeTime < 0 || (inputIndex > 0 && keyframeTime <= previousTime) || keyframeTime > durationSeconds {
			return &errs.MediaError{Problem: fmt.Sprintf(KeyframeTimeForm, inputIndex+1, keyframeTime), Cause: errs.ErrInputMediaTime}
		}

		previousTime = keyframeTime
	}

	return nil
}

// timedKeyframes takes the image inputs and their frame times and returns the keyframe array of
// time and media value pairs.
func timedKeyframes(imageInputs []media.Input, times []float64) []any {
	keyframes := make([]any, 0, len(imageInputs))
	for inputIndex, mediaInput := range imageInputs {
		keyframes = append(keyframes, []any{times[inputIndex], mediaInput.URLOrBase64()})
	}

	return keyframes
}

// parseDuration returns duration as seconds when it is numeric.
func parseDuration(parameterValues params.Values) (float64, bool) {
	switch durationValue := parameterValues[params.FlagTypeDuration].(type) {
	case int:
		return float64(durationValue), true
	case float64:
		return durationValue, true
	}

	return 0, false
}

// resolveFluxFrameAnchors updates the supplied media in place and clears its frame anchors. With a
// duration, anchors become zero and duration timestamps; otherwise opening and closing images move
// to the beginning and end.
func resolveFluxFrameAnchors(mediaInputs []media.Input, parameterValues params.Values) {
	durationSeconds, durationSet := parseDuration(parameterValues)
	if durationSet && durationSeconds >= 0 {
		for i := range mediaInputs {
			switch mediaInputs[i].FrameAnchor {
			case media.FrameFirst:
				openingTime := 0.0
				mediaInputs[i].Time = &openingTime
			case media.FrameLast:
				mediaInputs[i].Time = &durationSeconds
			default:
				continue
			}

			mediaInputs[i].FrameAnchor = ""
		}

		return
	}

	slices.SortStableFunc(mediaInputs, compareFrameOrder)

	for i := range mediaInputs {
		mediaInputs[i].FrameAnchor = ""
	}
}

// compareFrameOrder orders media inputs by frame anchor: the opening image first, unanchored media
// between, and the closing image last.
//
//nolint:gocritic // slices.SortStableFunc requires a comparator taking its element type by value.
func compareFrameOrder(a, b media.Input) int {
	return frameOrderRank(a.FrameAnchor) - frameOrderRank(b.FrameAnchor)
}

// frameOrderRank maps a frame anchor to its position rank in the keyframe order.
func frameOrderRank(frameAnchor string) int {
	switch frameAnchor {
	case media.FrameFirst:
		return 0
	case media.FrameLast:
		return 2
	}

	return 1
}
