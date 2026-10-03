package terminal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/output"
)

// Invariants tested:
// 1. Spinner rendering and cleanup: StartSpinner followed by Finish after at least one frame
//    interval must write the first Generating frame with a dimmed 0.0s counter, end with the
//    line-clearing sequence, and return a positive duration.
// 2. Spinner frame advancement: After StartSpinner, the output must contain the second spinner
//    frame within two seconds.
// 3. Spinner medium word: For image and video runs, StartSpinner must write a first frame
//    containing the selected medium, the dimmed starting counter, and the format specified by
//    GenerationStatus.

// TestSpinnerRendersAndErases verifies invariant #1: Spinner rendering and cleanup.
//
// What is being tested:
// StartSpinner followed by Finish after at least one frame interval must write the first
// Generating frame with a dimmed 0.0s counter, end with the line-clearing sequence, and return a
// positive duration. The wait keeps the duration positive on a clock as coarse as the Windows one.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSpinnerRendersAndErases(t *testing.T) {
	var buf bytes.Buffer

	destination := &buf

	spinner := StartSpinner(context.Background(), destination, media.Image)

	time.Sleep(frameInterval)

	elapsed := spinner.Finish()

	rendered := buf.String()
	if !strings.Contains(rendered, "⠋ Generating") {
		t.Errorf("✗ the immediate first frame is missing: %q", rendered)
	}

	if !strings.Contains(rendered, "\x1b[2m0.0s") {
		t.Errorf("✗ the dimmed starting counter is missing: %q", rendered)
	}

	if !strings.HasSuffix(rendered, "\r\x1b[K") {
		t.Errorf("✗ the status line is not erased at the finish: %q", rendered)
	}

	if elapsed <= 0 {
		t.Errorf("✗ Finish elapsed = %v, want a positive duration", elapsed)
	}

	if !t.Failed() {
		t.Logf("✓ the spinner renders immediately and erases its line at the finish")
	}
}

// TestSpinnerAnimatesFrames verifies invariant #2: Spinner frame advancement.
//
// What is being tested:
// After StartSpinner, the output must contain the second spinner frame within two seconds.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSpinnerAnimatesFrames(t *testing.T) {
	pipeReadEnd, pipeWriteEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 os.Pipe: %v", err)
	}

	t.Cleanup(func() { _ = pipeReadEnd.Close(); _ = pipeWriteEnd.Close() })

	spinner := StartSpinner(context.Background(), pipeWriteEnd, media.Image)
	defer spinner.Finish()

	// The frames arrive on a channel so that the wait is bounded by a timer rather than by a pipe
	// read deadline, which Windows pipes do not support.
	frameRunes := make(chan rune)
	stopReading := make(chan struct{})

	t.Cleanup(func() { close(stopReading) })

	go readFrameRunes(bufio.NewReader(pipeReadEnd), frameRunes, stopReading)

	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case streamRune, open := <-frameRunes:
			if !open {
				t.Fatalf("💣 the frame stream closed before an advanced frame")
			}

			if streamRune != '⠙' {
				continue
			}

			if !t.Failed() {
				t.Logf("✓ the spinner advances its frames while running")
			}

			return
		case <-deadline.C:
			t.Errorf("✗ no advanced frame within two seconds")

			return
		}
	}
}

// TestSpinnerStatusNamesMedium verifies invariant #3: Spinner medium word.
//
// What is being tested:
// For image and video runs, StartSpinner must write a first frame containing the selected medium,
// the dimmed starting counter, and the format specified by GenerationStatus.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSpinnerStatusNamesMedium(t *testing.T) {
	for _, media := range []media.Kind{media.Video, media.Image} {
		var buf bytes.Buffer

		destination := &buf

		spinner := StartSpinner(context.Background(), destination, media)
		spinner.Finish()

		wantFirstFrame := fmt.Sprintf(output.GenerationStatus, string(spinnerFrames[0]), string(media), "\x1b[2m", output.ElapsedText(0), "\x1b[0m")
		if !strings.Contains(buf.String(), wantFirstFrame) {
			t.Errorf("✗ the %s status line is missing %q in %q", media, wantFirstFrame, buf.String())
		}
	}

	if !t.Failed() {
		t.Log("✓ the status line names the medium after the word Generating")
	}
}

// readFrameRunes forwards each rune of the frame stream until the stream ends or stopReading
// closes, then closes frameRunes.
func readFrameRunes(frameStream *bufio.Reader, frameRunes chan<- rune, stopReading <-chan struct{}) {
	defer close(frameRunes)

	for {
		streamRune, _, err := frameStream.ReadRune()
		if err != nil {
			return
		}

		select {
		case frameRunes <- streamRune:
		case <-stopReading:
			return
		}
	}
}
