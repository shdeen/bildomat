package terminal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
// 1. Repeated spinner completion: Calling Finish twice must return the same elapsed duration both
//    times and write no additional bytes on the second call.
// 2. Cosmetic delivery failure: When the spinner writer refuses its first write, Finish must still
//    return a nonnegative duration and return the same duration on a second call. When the spinner
//    writer accepts the first frame but rejects a later frame, Finish must return within two
//    seconds.
// 3. Context cancellation cleanup: Whether the context is canceled before or after StartSpinner,
//    the spinner must write its clear sequence within two seconds, before Finish is called.

// TestSpinnerRepeatedFinish verifies invariant #1: Repeated spinner completion.
//
// What is being tested:
// Calling Finish twice must return the same elapsed duration both times and write no additional
// bytes on the second call.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSpinnerRepeatedFinish(t *testing.T) {
	var buf bytes.Buffer

	destination := &buf

	spinner := StartSpinner(context.Background(), destination, media.Image)
	firstElapsed := spinner.Finish()
	writtenAfterFirst := buf.Len()

	repeatElapsed := spinner.Finish()
	if repeatElapsed != firstElapsed {
		t.Errorf("✗ repeated Finish = %v, want the first duration %v", repeatElapsed, firstElapsed)
	}

	if buf.Len() != writtenAfterFirst {
		t.Errorf("✗ repeated Finish wrote %d further bytes", buf.Len()-writtenAfterFirst)
	}

	if !t.Failed() {
		t.Logf("✓ a repeated Finish is inert")
	}
}

// TestSpinnerDeliveryFailure verifies invariant #2: Cosmetic delivery failure.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When the spinner writer refuses its first write, Finish must still return a nonnegative duration
// and return the same duration on a second call.
//
// Kind: permanent.
func TestSpinnerDeliveryFailure(t *testing.T) {
	spinner := StartSpinner(context.Background(), refusedWriter{test: t}, media.Video)

	elapsed := spinner.Finish()
	if elapsed < 0 {
		t.Errorf("✗ cosmetic failure returned a negative duration: %v", elapsed)
	}

	if repeated := spinner.Finish(); repeated != elapsed {
		t.Errorf("✗ repeated completion changed duration: %v versus %v", repeated, elapsed)
	}

	if !t.Failed() {
		t.Log("✓ cosmetic delivery failure permits ordinary completion")
	}
}

// TestSpinnerLaterDeliveryFailure verifies invariant #2: Cosmetic delivery failure.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When the spinner writer accepts the first frame but rejects a later frame, Finish must return
// within two seconds. A second Finish call must return the same duration without another write.
//
// Kind: permanent.
func TestSpinnerLaterDeliveryFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	destination := &laterRefusedWriter{test: t, refused: make(chan struct{})}

	spinner := StartSpinner(ctx, destination, media.Video)
	select {
	case <-destination.refused:
	case <-time.After(2 * time.Second):
		t.Error("✗ animation did not reach its refused later frame")
		cancel()
	}

	completed := make(chan time.Duration, 1)
	go func() { completed <- spinner.Finish() }()

	select {
	case elapsed := <-completed:
		writesAfterCompletion := destination.writes
		if repeated := spinner.Finish(); repeated != elapsed || destination.writes != writesAfterCompletion {
			t.Errorf("✗ repeated completion changed duration or wrote again: %v, %v", elapsed, repeated)
		}
	case <-time.After(2 * time.Second):
		t.Error("✗ refused later frame prevented completion")
	}

	if !t.Failed() {
		t.Log("✓ a refused later frame ends animation and permits completion")
	}
}

// TestSpinnerContextCancellation verifies invariant #3: Context cancellation cleanup.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Whether the context is canceled before or after StartSpinner, the spinner must write its clear
// sequence within two seconds, before Finish is called. After Finish, no further output may follow
// that sequence.
//
// Kind: permanent.
func TestSpinnerContextCancellation(t *testing.T) {
	const shutdownWait = 2 * time.Second

	for _, canceledBeforeStart := range []bool{false, true} {
		t.Run(fmt.Sprintf("canceled-before-start=%t", canceledBeforeStart), func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatalf("💣 status pipe: %v", err)
			}

			t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })

			if err := reader.SetReadDeadline(time.Now().Add(shutdownWait)); err != nil {
				t.Fatalf("💣 status read deadline: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if canceledBeforeStart {
				cancel()
			}

			spinner := StartSpinner(ctx, writer, media.Video)
			defer spinner.Finish()

			cancel()

			statusReader := bufio.NewReader(reader)

			var rendered strings.Builder
			for !strings.HasSuffix(rendered.String(), "\r\x1b[K") {
				statusByte, readErr := statusReader.ReadByte()
				if readErr != nil {
					t.Errorf("✗ cancellation did not clear the status before completion: %v", readErr)

					break
				}

				rendered.WriteByte(statusByte)
			}

			spinner.Finish()

			if err := writer.Close(); err != nil {
				t.Fatalf("💣 close completed status writer: %v", err)
			}

			remaining, err := io.ReadAll(statusReader)
			if err != nil || len(remaining) != 0 {
				t.Errorf("✗ output followed cancellation cleanup: %q, %v", remaining, err)
			}

			if !t.Failed() {
				t.Log("✓ cancellation clears and ends animation before completion")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ canceled animation cannot race essential final rendering")
	}
}

// laterRefusedWriter accepts the first frame and rejects subsequent writes.
type laterRefusedWriter struct {
	test    testing.TB
	refused chan struct{}
	writes  int
}

// Write signals the first refusal while allowing cleanup to attempt its write.
func (writer *laterRefusedWriter) Write(content []byte) (int, error) {
	writer.test.Helper()

	writer.writes++
	if writer.writes == 1 {
		return len(content), nil
	}

	if writer.writes == 2 {
		close(writer.refused)
	}

	return 0, io.ErrClosedPipe
}
