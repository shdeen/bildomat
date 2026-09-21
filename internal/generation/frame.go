package generation

import (
	"fmt"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// DropFramePrefixes removes the frame prefix from every media input, for a
// model that accepts none, and returns one conformed-change record per
// dropped prefix. The media itself stays in the request.
func DropFramePrefixes(mediaInputs []media.Input) []params.Adjustment {
	var records []params.Adjustment

	for i := range mediaInputs {
		mediaInput := &mediaInputs[i]
		if !mediaInput.HasFrame() {
			continue
		}

		records = append(records, params.Adjustment{
			FlagID: params.FlagTypeInputMedia, Type: params.ChangeConformed,
			InputVal: mediaInput.Source(),
			WireVal:  mediaInput.SourceName(),
			Comment:  ReasonNoFramePrefix,
		})

		mediaInput.Time = nil
		mediaInput.FrameAnchor = ""
	}

	return records
}

// ResolveFrameAnchors reconciles frame prefixes for a model that accepts only
// the opening and closing frames, mutating the media inputs in place so every
// frame request ends as an anchor keyword with no numeric time.
//
// Anchor keywords take their frames directly. Numeric times fill the frames
// the keywords left free: two times take the opening and closing frames in
// numeric order; one time takes the only free frame, or, with both free,
// snaps at or below half the duration to the opening frame and past it to the
// closing frame (with no duration set, zero opens and any later time closes).
// Each true snap returns one snapped-change record.
//
// A frame prefix on a video input, the same anchor on two inputs, equal
// times, and more timed inputs than free frames all fail, naming the
// offending sources.
func ResolveFrameAnchors(mediaInputs []media.Input, parameterValues params.Values) ([]params.Adjustment, error) {
	if len(mediaInputs) == 0 {
		return nil, nil
	}

	durationSeconds, durationSet := numericSeconds(parameterValues[params.FlagTypeDuration])

	anchorOwners, numericIndexes, err := classifyFrameRequests(mediaInputs)
	if err != nil {
		return nil, err
	}

	firstFree, lastFree, freeCount := freeFrames(anchorOwners)
	if len(numericIndexes) > freeCount {
		return nil, &errs.MediaError{Problem: fmt.Sprintf(FrameCountExceeded, len(numericIndexes)+len(anchorOwners)), Cause: errs.ErrInputMediaTime}
	}

	return fillFreeFrames(mediaInputs, numericIndexes, firstFree, lastFree, durationSeconds, durationSet)
}

// classifyFrameRequests takes the media inputs and returns the source that
// claims each anchor keyword, keyed by the keyword, and the indexes of the
// inputs carrying a numeric frame time, in input order. It fails on a frame
// prefix on a video input and on an anchor keyword claimed by two inputs,
// naming the offending sources.
func classifyFrameRequests(mediaInputs []media.Input) (anchorOwners map[string]string, numericIndexes []int, err error) {
	anchorOwners = map[string]string{}

	for i := range mediaInputs {
		mediaInput := &mediaInputs[i]
		if !mediaInput.HasFrame() {
			continue
		}

		if mediaInput.Kind() == media.Video {
			return nil, nil, &errs.MediaError{Problem: fmt.Sprintf(FrameOnVideo, mediaInput.Source()), Cause: errs.ErrInputMediaTime}
		}

		if mediaInput.FrameAnchor != "" {
			if owner := anchorOwners[mediaInput.FrameAnchor]; owner != "" {
				return nil, nil, &errs.MediaError{Problem: fmt.Sprintf(FrameAnchorConflict, owner, mediaInput.Source(), mediaInput.FrameAnchor), Cause: errs.ErrInputMediaTime}
			}

			anchorOwners[mediaInput.FrameAnchor] = mediaInput.Source()

			continue
		}

		numericIndexes = append(numericIndexes, i)
	}

	return anchorOwners, numericIndexes, nil
}

// freeFrames takes the sources claiming each anchor keyword and reports
// whether the opening frame and the closing frame are free of a keyword, and
// how many of the two are.
func freeFrames(anchorOwners map[string]string) (firstFree, lastFree bool, freeCount int) {
	firstFree = anchorOwners[media.FrameFirst] == ""
	lastFree = anchorOwners[media.FrameLast] == ""

	if firstFree {
		freeCount++
	}

	if lastFree {
		freeCount++
	}

	return firstFree, lastFree, freeCount
}

// fillFreeFrames moves the numeric-timed media inputs onto the frames the
// anchor keywords left free: two times take the opening and closing frames in
// numeric order, one time takes the only free frame or snaps by the half
// rule, and equal times fail naming both sources.
func fillFreeFrames(mediaInputs []media.Input, numericIndexes []int, firstFree, lastFree bool, durationSeconds float64, durationSet bool) ([]params.Adjustment, error) {
	switch len(numericIndexes) {
	case 0:
		return nil, nil
	case 1:
		mediaInput := &mediaInputs[numericIndexes[0]]
		seconds, _ := mediaInput.FrameTime()

		anchor := media.FrameFirst

		switch {
		case firstFree && lastFree:
			if pastHalfDuration(seconds, durationSeconds, durationSet) {
				anchor = media.FrameLast
			}
		case lastFree:
			anchor = media.FrameLast
		}

		return anchorNumericInput(mediaInput, seconds, anchor, durationSeconds, durationSet), nil
	}

	openingInput := &mediaInputs[numericIndexes[0]]
	closingInput := &mediaInputs[numericIndexes[1]]
	openingSeconds, _ := openingInput.FrameTime()
	closingSeconds, _ := closingInput.FrameTime()

	if openingSeconds == closingSeconds {
		return nil, &errs.MediaError{Problem: fmt.Sprintf(FrameTimeShared, openingInput.Source(), closingInput.Source(), openingSeconds), Cause: errs.ErrInputMediaTime}
	}

	if openingSeconds > closingSeconds {
		openingInput, closingInput = closingInput, openingInput
		openingSeconds, closingSeconds = closingSeconds, openingSeconds
	}

	records := anchorNumericInput(openingInput, openingSeconds, media.FrameFirst, durationSeconds, durationSet)

	return append(records, anchorNumericInput(closingInput, closingSeconds, media.FrameLast, durationSeconds, durationSet)...), nil
}

// pastHalfDuration reports whether a frame time falls past half the duration.
// With no duration set, any time later than zero counts as past.
func pastHalfDuration(seconds, durationSeconds float64, durationSet bool) bool {
	if !durationSet {
		return seconds > 0
	}

	return seconds > durationSeconds/2
}

// anchorNumericInput moves one numeric-timed media input onto an anchor frame
// and returns one snapped-change record, or none when the time already names
// that frame exactly (zero for the opening frame, the duration for the
// closing one).
func anchorNumericInput(mediaInput *media.Input, seconds float64, anchor string, durationSeconds float64, durationSet bool) []params.Adjustment {
	requestedSource := mediaInput.Source()
	mediaInput.Time = nil
	mediaInput.FrameAnchor = anchor

	exactOpening := anchor == media.FrameFirst && seconds == 0
	exactClosing := anchor == media.FrameLast && durationSet && seconds == durationSeconds

	if exactOpening || exactClosing {
		return nil
	}

	return []params.Adjustment{{
		FlagID: params.FlagTypeInputMedia, Type: params.ChangeSnapped,
		InputVal: requestedSource,
		WireVal:  mediaInput.Source(),
		Comment:  fmt.Sprintf(ReasonFrameTimeSnapped, seconds, anchor),
	}}
}
