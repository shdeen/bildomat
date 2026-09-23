package terminal

import (
	"context"
	"io"
	"time"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/output"
)

// ansiEraseStatus returns to the beginning of the status line and clears it.
const ansiEraseStatus = "\r\x1b[K"

// spinnerFrameRunes spells the braille animation frames in display order.
const spinnerFrameRunes = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"

// spinnerFrames holds the animation frames as runes.
//
//nolint:gochecknoglobals // decoded once at package load from spinnerFrameRunes; a rune slice cannot be a constant.
var spinnerFrames = []rune(spinnerFrameRunes)

// frameInterval is the delay between spinner frame advances.
const frameInterval = 100 * time.Millisecond

// Spinner animates a generation status until completion, cancellation, or a write failure.
type Spinner struct {
	// destination receives status frames and the final erase sequence.
	destination io.Writer
	// startedAt anchors the elapsed counter.
	startedAt time.Time
	// ticker schedules frame advances.
	ticker *time.Ticker
	// done requests animation shutdown when closed.
	done chan struct{}
	// animated closes after animation and its cleanup finish.
	animated chan struct{}
	// media selects the image or video status label.
	media media.Kind
	// frameIndex selects the current animation frame modulo the frame count.
	frameIndex int
	// finished records that Finish has already returned a duration.
	finished bool
	// finalElapsed preserves the duration returned by repeated Finish calls.
	finalElapsed time.Duration
}

// StartSpinner renders the first status frame and starts animation on destination. A write failure
// stops the animation without affecting generation.
func StartSpinner(ctx context.Context, destination io.Writer, mediaKind media.Kind) *Spinner {
	spinner := &Spinner{
		destination: destination,
		startedAt:   time.Now(),
		ticker:      time.NewTicker(frameInterval),
		done:        make(chan struct{}),
		animated:    make(chan struct{}),
		media:       mediaKind,
	}

	if err := spinner.render(); err != nil {
		spinner.ticker.Stop()
		close(spinner.animated)

		return spinner
	}
	go spinner.animate(ctx)

	return spinner
}

// Finish stops and waits for animation, returning the elapsed duration. Repeated calls return the
// same duration without writing again.
func (spinner *Spinner) Finish() time.Duration {
	if spinner.finished {
		return spinner.finalElapsed
	}

	spinner.finished = true
	close(spinner.done)
	<-spinner.animated
	spinner.finalElapsed = time.Since(spinner.startedAt)

	return spinner.finalElapsed
}

// animate advances frames until completion, cancellation, or a write failure. It then attempts to
// clear the status line.
func (spinner *Spinner) animate(ctx context.Context) {
	defer close(spinner.animated)
	defer spinner.ticker.Stop()
	defer output.WriteText(spinner.destination, "%s", ansiEraseStatus) //nolint:errcheck // Cosmetic cleanup cannot replace the generation outcome.

	for {
		select {
		case <-ctx.Done():
			return
		case <-spinner.done:
			return
		case <-spinner.ticker.C:
			spinner.frameIndex++
			if err := spinner.render(); err != nil {
				return
			}
		}
	}
}

// render replaces the status line with one frame and reports delivery failure.
func (spinner *Spinner) render() error {
	return output.WriteText(spinner.destination, "\r%s", output.GenerationStatusText(spinnerFrames[spinner.frameIndex%len(spinnerFrames)], spinner.media, time.Since(spinner.startedAt), true))
}
