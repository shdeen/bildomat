package errs

import "fmt"

// MediaError identifies an input source or a specific media problem without
// requiring the renderer to recover either from a formatted diagnostic.
type MediaError struct {
	Source  string
	Problem string
	Cause   error
}

// Error returns the source and media diagnostic with its original cause.
func (failure *MediaError) Error() string {
	description := failure.Source
	if failure.Problem != "" {
		if description != "" {
			description += ": "
		}

		description += failure.Problem
	}

	return fmt.Sprintf("%q: %s", description, failure.Cause)
}

// Unwrap retains the media classification and original cause.
func (failure *MediaError) Unwrap() error { return failure.Cause }
