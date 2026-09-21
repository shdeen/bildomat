package httpapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
)

// jobPoll checks the state of an asynchronous job.
type jobPoll interface {
	// Poll retains a valid completed result and reports completion or failure.
	Poll(ctx context.Context) (jobDone bool, err error)
}

// Poll checks a job at fixed intervals until it completes, the context ends, or the polling budget expires.
// The caller may read its poller's completed result only after Poll succeeds.
func Poll(ctx context.Context, pace, pollBudget time.Duration, poller jobPoll) error {
	dctx, cancel := context.WithTimeout(ctx, pollBudget)
	defer cancel()

	var lastErr error

	for {
		if ctx.Err() != nil {
			return fmt.Errorf("%w, %w", errs.ErrCanceled, ctx.Err())
		}

		if dctx.Err() != nil {
			return getPollTimeout(pollBudget, lastErr)
		}

		jobDone, err := poller.Poll(dctx)

		switch {
		case ctx.Err() != nil:
			return fmt.Errorf("%w, %w", errs.ErrCanceled, ctx.Err())
		case err == nil && jobDone:
			return nil
		case dctx.Err() != nil:
			return getPollTimeout(pollBudget, errors.Join(lastErr, err))
		case err != nil && !retryablePollError(err):
			return err
		case err != nil:
			lastErr = err
		}

		if wErr := wait(dctx, pace); wErr != nil {
			if ctx.Err() != nil {
				return wErr
			}
			// Only the loop's own deadline interrupted the wait.
			return getPollTimeout(pollBudget, lastErr)
		}
	}
}

// retryablePollError gives permanent classifications precedence over a
// simultaneous response-body read failure.
func retryablePollError(err error) bool {
	if errors.Is(err, errs.ErrCanceled) || errors.Is(err, errs.ErrTransportSize) || errors.Is(err, errs.ErrResponseDecode) {
		return false
	}

	if errors.Is(err, errs.ErrResponseStatus) {
		return errors.Is(err, errs.ErrResponseStatusTemporary)
	}

	return errors.Is(err, errs.ErrTransportRequest) || errors.Is(err, errs.ErrTransportRead)
}

// wait blocks for d or returns an error when ctx ends first.
func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("%w, %w", errs.ErrCanceled, ctx.Err())
	case <-timer.C:
		return nil
	}
}

// getPollTimeout preserves the last failed observation across pending replies.
func getPollTimeout(timeout time.Duration, last error) error {
	detail := fmt.Sprintf(TimeoutAfterForm, timeout)

	if last != nil {
		return fmt.Errorf("%q, %w, %w", detail, errs.ErrTransportTimeout, last)
	}

	return fmt.Errorf("%q, %w", detail, errs.ErrTransportTimeout)
}
