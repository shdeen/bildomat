package errs

import (
	"errors"
	"fmt"
)

// ConfigError identifies a configuration source, its owner or setting, and
// the specific problem. Cause preserves its classification and original error.
type ConfigError struct {
	Path     string
	Provider string
	Setting  string
	Problem  string
	Cause    error
}

// Error returns the complete configuration diagnostic.
func (failure *ConfigError) Error() string {
	description := failure.Path
	switch {
	case errors.Is(failure.Cause, ErrUserConfigUnknownSetting):
		description += ": " + failure.Setting
	case errors.Is(failure.Cause, ErrUserConfigUnknownProvider):
		description += ": " + failure.Provider
	case failure.Problem != "":
		if description != "" {
			description += ": "
		}

		description += failure.Problem
	case description == "":
		description = failure.Provider
	}

	return fmt.Sprintf("%q: %s", description, failure.Cause)
}

// Unwrap retains the configuration classification and original cause.
func (failure *ConfigError) Unwrap() error { return failure.Cause }
