package errs

import "fmt"

// ModelError retains the model specifier that could not be resolved.
//   - Specifier: the unresolved model input
//   - Cause: the resolution failure classification
type ModelError struct {
	Specifier string
	Cause     error
}

// Error returns the rejected model specifier and its classification.
func (failure *ModelError) Error() string {
	return fmt.Sprintf("%q, %s", failure.Specifier, failure.Cause)
}

// Unwrap retains the model resolution classification.
func (failure *ModelError) Unwrap() error { return failure.Cause }
