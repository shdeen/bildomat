package output

import (
	"fmt"
	"io"
	"time"

	"github.com/shdeen/bildomat/internal/media"
)

// File: internal/output/progress.go Generation status, elapsed duration, file size, and completion
// formatting.

// tenthsPerMinute is the number of tenths of a second in one minute; the counter measures elapsed
// time in tenths and divides it into minutes.
const tenthsPerMinute = 600

// Decimal units used in saved-file reports.
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

// ElapsedText formats elapsed time as seconds with one decimal place, truncating fractional tenths.
// It adds minutes at one minute and hours at one hour, as in 23.6s, 1m 23.6s, and 1h 2m 12.3s.
// Negative durations render as zero.
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

// fileSizeText takes a byte count and returns its human-friendly form: bytes under 1000, and
// otherwise 1000-based KB, MB, or GB with one decimal place. A negative count renders as zero.
func fileSizeText(byteCount int64) string {
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

// PrintGenerationCompleted writes the completion message with the supplied elapsed time and returns
// any write failure.
func PrintGenerationCompleted(destination io.Writer, elapsedText string) error {
	return WriteText(destination, GenerationCompleted+"\n", elapsedText)
}

// GenerationStatusText formats one status frame with the requested styling. It performs no cursor
// movement or animation.
func GenerationStatusText(frame rune, mediaKind media.Kind, elapsed time.Duration, styled bool) string {
	style := pageStyleValues(styled)

	return fmt.Sprintf(GenerationStatus, string(frame), string(mediaKind), style.dim, ElapsedText(elapsed), style.reset)
}
