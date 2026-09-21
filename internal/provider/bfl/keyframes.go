package bfl

import (
	"fmt"
	"math"
	"slices"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// buildKeyframes returns an all-string or all-tuple keyframe array: the media
// values alone when no image carries a frame time, and otherwise time and
// media value pairs, with the missing times filled from the duration and every
// time checked for order and range.
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

// keyframeTimes takes the image inputs and returns each input's frame time and
// whether the input carries one, in input order.
func keyframeTimes(imageInputs []media.Input) (times []float64, timeSet []bool) {
	times = make([]float64, len(imageInputs))
	timeSet = make([]bool, len(imageInputs))

	for inputIndex, mediaInput := range imageInputs {
		times[inputIndex], timeSet[inputIndex] = mediaInput.FrameTime()
	}

	return times, timeSet
}

// untimedKeyframes takes image inputs carrying no frame times and returns the
// keyframe array of their media values alone.
func untimedKeyframes(imageInputs []media.Input) []any {
	keyframes := make([]any, 0, len(imageInputs))
	for _, mediaInput := range imageInputs {
		keyframes = append(keyframes, mediaInput.URLOrBase64())
	}

	return keyframes
}

// fillKeyframeTimes takes the frame times, their set flags, and the duration,
// and fills the missing times in place: an unset first time becomes zero and
// an unset last time becomes the duration; when a time is still unset, missing
// time takes its even share of the duration. Supplied values remain unchanged;
// disagreement beyond arithmetic rounding fails.
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
			// One rounding step in division and one in multiplication bound
			// the difference between equivalent evenly spaced positions.
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

// checkKeyframeTimes takes the frame times and the duration and fails on a time
// below zero, above the duration, or not later than the time before it.
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

// timedKeyframes takes the image inputs and their frame times and returns the
// keyframe array of time and media value pairs.
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

// resolveFluxFrameAnchors resolves the first and last anchor keywords for the
// arbitrary-time keyframe request, mutating the media inputs in place. With a
// duration set, the anchors become the zero and duration keyframe times. With
// none set, they reorder the media so the opening image leads and the closing
// image trails, which the keyframe array's own position inference then honors.
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

// compareFrameOrder orders media inputs by frame anchor: the opening image
// first, unanchored media between, and the closing image last.
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
