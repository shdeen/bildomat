package errs

import "fmt"

// PollError identifies the existing remote resource whose observation failed. Model and Resource
// remain available to ordinary output independently of transport diagnostics in Cause.
//   - Model: the model associated with the remote job
//   - Resource: the existing job identifier or polling URL
//   - Cause: the polling failure classification and original error
type PollError struct {
	Model    string
	Resource string
	Cause    error
}

// Error returns the operation identity and its classified cause.
func (failure *PollError) Error() string {
	return fmt.Sprintf("%q, %q, %s", failure.Model, failure.Resource, failure.Cause)
}

// Unwrap preserves the timeout, response, or transport cause.
func (failure *PollError) Unwrap() error { return failure.Cause }
