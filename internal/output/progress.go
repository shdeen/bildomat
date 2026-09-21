package output

import (
	"fmt"
	"io"
	"time"

	"github.com/shdeen/bildomat/internal/media"
)

// File: internal/output/progress.go
// The terminal generation display: the animated spinner status line, the
// elapsed-counter and file-size text forms, the completion report, and the
// styled saved-file report.

// spinnerFrameRunes spells the braille animation frames in display order.
const spinnerFrameRunes = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"

// spinnerFrames holds the animation frames as runes.
//
//nolint:gochecknoglobals // decoded once at package load from spinnerFrameRunes; a rune slice cannot be a constant.
var spinnerFrames = []rune(spinnerFrameRunes)

// frameInterval is the delay between spinner frame advances.
const frameInterval = 100 * time.Millisecond

// tenthsPerMinute is the number of tenths of a second in one minute; the
// counter measures elapsed time in tenths and divides it into minutes.
const tenthsPerMinute = 600

// The file-size units. The scale between adjacent units is a choice, not a
// property of bytes, so it is declared once and the unit sizes derive from it.
//   - sizeUnitScale: the factor between adjacent units
//   - bytesPerKB: the bytes in one kilobyte
//   - bytesPerMB: the bytes in one megabyte
//   - bytesPerGB: the bytes in one gigabyte
const (
	sizeUnitScale = 1_000
	bytesPerKB    = sizeUnitScale
	bytesPerMB    = sizeUnitScale * bytesPerKB
	bytesPerGB    = sizeUnitScale * bytesPerMB
)

// Spinner renders the terminal generation status line — a braille frame, the
// word Generating, the medium of the run, and a dimmed elapsed counter —
// animated until Finish.
type Spinner struct {
	destination  io.Writer
	startedAt    time.Time
	ticker       *time.Ticker
	done         chan struct{}
	animated     chan struct{}
	media        media.Kind
	frameIndex   int
	finished     bool
	finalElapsed time.Duration
}

// StartSpinner starts a cosmetic status display on destination. A write failure
// stops the animation; essential result delivery has its own error contract.
func StartSpinner(destination io.Writer, mediaKind media.Kind) *Spinner {
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
	go spinner.animate()

	return spinner
}

// Finish stops and joins the animation before clearing its status line. Repeated
// calls return the original duration without writing again.
func (spinner *Spinner) Finish() time.Duration {
	if spinner.finished {
		return spinner.finalElapsed
	}

	spinner.finished = true
	close(spinner.done)
	<-spinner.animated
	spinner.ticker.Stop()
	// Erasing a cosmetic status line must not change the generation outcome.
	_ = WriteText(spinner.destination, "%s", ansiEraseStatus) //nolint:errcheck // Clearing cosmetic animation cannot change generation status.
	spinner.finalElapsed = time.Since(spinner.startedAt)

	return spinner.finalElapsed
}

// animate advances frames until completion or the first cosmetic write failure.
func (spinner *Spinner) animate() {
	defer close(spinner.animated)
	defer spinner.ticker.Stop()

	for {
		select {
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
	return WriteText(spinner.destination, "\r"+GenerationStatus, string(spinnerFrames[spinner.frameIndex%len(spinnerFrames)]), string(spinner.media), ansiDim, ElapsedText(time.Since(spinner.startedAt)), ansiReset)
}

// ElapsedText takes an elapsed duration and returns its counter form:
// truncated one-decimal seconds (`23.6s`), prefixed by minutes from one
// minute on (`1m 23.6s`) and by hours from one hour on (`1h 2m 12.3s`). A
// negative duration renders as zero.
func ElapsedText(elapsed time.Duration) string {
	if elapsed < 0 {
		elapsed = 0
	}

	tenths := elapsed.Milliseconds() / 100
	secondsText := fmt.Sprintf(CounterSecondsForm, (tenths/10)%60, tenths%10)
	totalMinutes := tenths / tenthsPerMinute

	switch {
	case totalMinutes == 0:
		return secondsText
	case totalMinutes < 60:
		return fmt.Sprintf(CounterMinutesForm, totalMinutes, secondsText)
	}

	return fmt.Sprintf(CounterHoursForm, totalMinutes/60, totalMinutes%60, secondsText)
}

// FileSizeText takes a byte count and returns its human-friendly form: bytes
// under 1000, and otherwise 1000-based KB, MB, or GB with one decimal place.
// A negative count renders as zero.
func FileSizeText(byteCount int64) string {
	if byteCount < 0 {
		byteCount = 0
	}

	switch {
	case byteCount < bytesPerKB:
		return fmt.Sprintf(FileSizeBytesForm, byteCount)
	case byteCount < bytesPerMB:
		return fmt.Sprintf(FileSizeKBForm, float64(byteCount)/bytesPerKB)
	case byteCount < bytesPerGB:
		return fmt.Sprintf(FileSizeMBForm, float64(byteCount)/bytesPerMB)
	default:
		return fmt.Sprintf(FileSizeGBForm, float64(byteCount)/bytesPerGB)
	}
}

// PrintGenerationCompleted writes the provider-generation duration and returns
// any failure to deliver this essential summary.
func PrintGenerationCompleted(destination io.Writer, elapsedText string) error {
	return WriteText(destination, GenerationCompleted+"\n", elapsedText)
}
