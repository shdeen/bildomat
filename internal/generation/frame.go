package generation

import (
	"fmt"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// DropFramePrefixes clears frame markers in the supplied media slice and records each removal. The
// underlying media and bare sources remain unchanged.
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

// ResolveFrameAnchors replaces numeric frame times in the supplied media with first/last anchors
// and records actual snaps. It rejects frame markers on video, conflicting anchors or times, and
// requests exceeding the two available frames.
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

// classifyFrameRequests returns the sources claiming each anchor and the indexes of numeric times.
// It rejects frame markers on video and duplicate anchor claims.
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

// freeFrames reports which first/last anchors remain unclaimed and how many are available.
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

// fillFreeFrames assigns numeric times to unclaimed anchors in the supplied media slice. Two times
// sort chronologically; one takes the sole free anchor or snaps around half the duration. Equal
// times fail with both sources identified.
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

// pastHalfDuration reports whether a frame time falls past half the duration. With no duration set,
// any time later than zero counts as past.
func pastHalfDuration(seconds, durationSeconds float64, durationSet bool) bool {
	if !durationSet {
		return seconds > 0
	}

	return seconds > durationSeconds/2
}

// anchorNumericInput replaces one numeric frame time with its selected anchor. It returns a snap
// record unless the time exactly denotes the opening or known closing time.
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
