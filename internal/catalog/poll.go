package catalog

import (
	"fmt"
	"time"
)

// PollSeconds is an interval or timeout expressed in whole seconds in a provider description.
type PollSeconds int

// Duration returns the configured number of seconds as a duration.
func (seconds PollSeconds) Duration() time.Duration { return time.Duration(seconds) * time.Second }

// checkPolling requires a complete positive pair when the operation polls. An unused optional
// operation may omit both values.
func checkPolling(cfgName, section string, interval, timeout PollSeconds, required bool) error {
	if !required && interval == 0 && timeout == 0 {
		return nil
	}

	if interval <= 0 {
		if section == SectionVideoAPI {
			return createInvalidCfgError(cfgName, fmt.Sprintf(PollIntervalNotPositive, interval))
		}

		return createInvalidCfgError(cfgName, fmt.Sprintf(PollIntervalInvalid, section, interval))
	}

	if timeout <= 0 {
		if section == SectionVideoAPI {
			return createInvalidCfgError(cfgName, fmt.Sprintf(PollBudgetNotPositive, timeout))
		}

		return createInvalidCfgError(cfgName, fmt.Sprintf(PollTimeoutInvalid, section, timeout))
	}

	return nil
}
