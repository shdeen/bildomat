package errs

import "fmt"

// UsageError retains an actionable command-line explanation and its cause.
// Text is product copy supplied by the command that rejected the input.
type UsageError struct {
	Text  string
	Cause error
}

// Error returns the command explanation and diagnostic cause.
func (failure *UsageError) Error() string { return fmt.Sprintf("%q, %s", failure.Text, failure.Cause) }

// Unwrap retains the usage classification and original parser failure.
func (failure *UsageError) Unwrap() error { return failure.Cause }
