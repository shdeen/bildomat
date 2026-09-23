package errs

import "fmt"

// CredentialError names the environment variable and optional provider entry under api-keys that
// can supply a missing credential. It never holds a secret.
//   - EnvVar: the credential environment-variable name
//   - ProviderID: the optional provider key under api-keys
type CredentialError struct {
	EnvVar     string
	ProviderID string
}

// Error returns the available credential-setting locations.
func (failure *CredentialError) Error() string {
	if failure.ProviderID == "" {
		return fmt.Sprintf("%s %s", failure.EnvVar, ErrKeyMissing)
	}

	return fmt.Sprintf(APIKeyMissingForm, failure.EnvVar, ErrKeyMissing, failure.ProviderID)
}

// Unwrap identifies a missing credential.
func (*CredentialError) Unwrap() error { return ErrKeyMissing }
