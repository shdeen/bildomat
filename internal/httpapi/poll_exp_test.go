package httpapi

// Invariants tested:
//  1. Poll timeout: When wrapped in a PollError, getPollTimeout must preserve a supplied transport
//     error for errors.Is and include the wrapper's job name in the error text.
//  2. Canceled waits: Given a canceled context and a five-second delay, wait must return
//     ErrCanceled within one second.
//  3. Completed observation at deadline: Given a probe that reports successful completion after its
//     polling context expires, Poll must return nil.
//  4. Terminal request-creation failure: Given a probe that fails request creation, Poll must
//     return ErrTransportCreate after exactly one attempt and within five seconds of starting a
//     thirty-second polling budget.

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/metadata"
)

// TestPollTimeout verifies invariant #1: Poll timeout.
//
// What is being tested:
// When wrapped in a PollError, getPollTimeout must preserve a supplied transport error for
// errors.Is and include the wrapper's job name in the error text. With no supplied cause, the
// wrapper must still name the job and must not match that transport error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollTimeout(t *testing.T) {
	last := errors.New("connection reset")

	wErr := &errs.PollError{Model: "video-model", Resource: "job-1", Cause: getPollTimeout(2*time.Second, last)}
	if !strings.Contains(wErr.Error(), "job-1") {
		t.Errorf("✗ pollTimeout error %v does not name the job", wErr)
	}

	if !errors.Is(wErr, last) {
		t.Errorf("✗ pollTimeout with a last error must carry it (errors.Is): %v", wErr)
	}

	nerr := &errs.PollError{Model: "fixture video", Resource: "job-2", Cause: getPollTimeout(2*time.Second, nil)}
	if !strings.Contains(nerr.Error(), "job-2") {
		t.Errorf("✗ pollTimeout(nil) error %v does not name the job", nerr)
	}

	if errors.Is(nerr, last) {
		t.Errorf("✗ pollTimeout(nil) must not carry an unrelated error: %v", nerr)
	}

	if !t.Failed() {
		t.Log("✓ pollTimeout names the job and carries the last transport error only when one occurred")
	}
}

// TestWaitCanceled verifies invariant #2: Canceled waits.
//
// What is being tested:
// Given a canceled context and a five-second delay, wait must return ErrCanceled within one second.
// With a live context and a one-millisecond delay, it must return nil.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWaitCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()

	err := wait(ctx, 5*time.Second)
	if err == nil || !errors.Is(err, errs.ErrCanceled) {
		t.Errorf("✗ wait(canceled ctx) = %v, want a wrapped ErrCanceled", err)
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("✗ wait blocked %v after cancellation, want a prompt return", elapsed)
	}

	if err := wait(context.Background(), time.Millisecond); err != nil {
		t.Errorf("✗ wait(live ctx, 1ms) = %v, want nil after the delay", err)
	}

	if !t.Failed() {
		t.Log("✓ wait returns promptly with ErrCanceled on cancellation and nil after the delay")
	}
}

// TestCompletedObservationAtDeadline verifies invariant #3: Completed observation at deadline.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// Given a probe that reports successful completion after its polling context expires, Poll must
// return nil. If the probe also cancels the parent context, Poll must instead return ErrCanceled.
func TestCompletedObservationAtDeadline(t *testing.T) {
	for _, completionCase := range []struct {
		name         string
		cancelParent bool
	}{{name: "budget expiry"}, {name: "parent cancellation", cancelParent: true}} {
		t.Run(completionCase.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			completionProbe := &deadlineCompletionProbe{test: t}
			if completionCase.cancelParent {
				completionProbe.cancelParent = cancel
			}

			pollError := Poll(ctx, time.Millisecond, time.Second, completionProbe)
			if completionCase.cancelParent && !errors.Is(pollError, errs.ErrCanceled) {
				t.Errorf("✗ canceled parent returned %v; require ErrCanceled", pollError)
			}

			if !completionCase.cancelParent && pollError != nil {
				t.Errorf("✗ valid completion at deadline returned %v; require success", pollError)
			}

			if !t.Failed() {
				t.Log("✓ completion respects the applicable context outcome")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ completed observations survive budget expiry and respect parent cancellation")
	}
}

// TestPollTerminalCreate verifies invariant #4: Terminal request-creation failure.
//
// What is being tested:
// Given a probe that fails request creation, Poll must return ErrTransportCreate after exactly one
// attempt and within five seconds of starting a thirty-second polling budget.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPollTerminalCreate(t *testing.T) {
	var attempts atomic.Int64

	probe := &createFailCheck{test: t, attempts: &attempts}
	start := time.Now()

	err := Poll(t.Context(), time.Millisecond, 30*time.Second, probe)
	if err == nil || !errors.Is(err, errs.ErrTransportCreate) {
		t.Errorf("✗ httpapi.Poll(create failure) = %v, want the wrapped ErrTransportCreate returned immediately", err)
	}

	if got := attempts.Load(); got != 1 {
		t.Errorf("✗ %d probe attempts after a deterministic create failure, want exactly 1", got)
	}

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("✗ Poll took %v on a create failure, want an immediate return with no deadline wait", elapsed)
	}

	if !t.Failed() {
		t.Log("✓ a deterministic request-create failure is terminal on the first attempt")
	}
}

// deadlineCompletionProbe reports completion after its polling context expires.
//   - test: the owning test
//   - cancelParent: optional cancellation applied before completion is reported
type deadlineCompletionProbe struct {
	test         *testing.T
	cancelParent context.CancelFunc
}

// Poll waits for the polling deadline, optionally cancels the parent, then reports completion.
func (probe *deadlineCompletionProbe) Poll(ctx context.Context) (bool, error) {
	probe.test.Helper()

	if probe.cancelParent != nil {
		probe.cancelParent()
	}

	<-ctx.Done()

	return true, nil
}

// createFailCheck counts polling attempts that fail before a request can be sent.
//   - test: the test receiving helper failures
//   - attempts: the shared attempt counter
type createFailCheck struct {
	test     testing.TB
	attempts *atomic.Int64
}

// Poll counts the attempt and returns a deterministic request-creation failure.
func (c *createFailCheck) Poll(ctx context.Context) (bool, error) {
	c.test.Helper()

	c.attempts.Add(1)

	_, _, err := GetAuth(ctx, "://not-a-url", AuthCredential{}, metadata.Asynchronous, nil)

	return false, err
}
