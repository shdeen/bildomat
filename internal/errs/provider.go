package errs

import "fmt"

// ProviderError carries the provider's explanation independently of outer request labels and other
// failures in the same error chain.
//   - Message: the provider-supplied failure explanation
//   - Cause: the response failure classification and original error
type ProviderError struct {
	Message string
	Cause   error
}

// Error returns the provider's explanation and classified cause.
func (failure *ProviderError) Error() string {
	return fmt.Sprintf("%q, %s", failure.Message, failure.Cause)
}

// Unwrap retains the response classification and original cause.
func (failure *ProviderError) Unwrap() error { return failure.Cause }
