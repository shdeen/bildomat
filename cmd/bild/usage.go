package main

import (
	"fmt"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/urfave/cli/v3"
)

// combinedFlagError takes the long name of the help or version flag and
// returns the usage error for that flag given beside a positional argument.
// The message names the flag in its long form whichever form was typed,
// because the library does not report the typed form.
func combinedFlagError(flagName string) error {
	return &errs.UsageError{Text: fmt.Sprintf(HelpFlagCombinedForm, "--"+flagName), Cause: errs.ErrCLIHelpCombined}
}

// searchTermsError takes the parsed search command and returns the usage
// error its terms raise, or nil: the argument-count error for more than one
// positional argument, the missing-term error when neither a search term nor
// --exclude was given, and the empty-term error when either was typed as the
// empty string. Only whether the flag was set and the argument count tell a
// term typed empty from a term not given; the values alone cannot.
func searchTermsError(c *cli.Command) error {
	termCount := c.Args().Len()
	hasExcludeFlag := c.IsSet(FilterFlagExclude)

	switch {
	case termCount > 1:
		return fmt.Errorf(ArgCountForm, c.Name, errs.ErrCLIOneArgMax, termCount)
	case termCount == 0 && !hasExcludeFlag:
		return &errs.UsageError{Text: SearchTermMissing, Cause: errs.ErrCLISearchTermMissing}
	case termCount == 1 && c.Args().Get(0) == "", hasExcludeFlag && c.String(FilterFlagExclude) == "":
		return &errs.UsageError{Text: SearchTermEmpty, Cause: errs.ErrCLISearchTermEmpty}
	}

	return nil
}

// argCountError takes a command name and the argument counts required and
// received, and returns the CLI error for the required count.
func argCountError(cmdName string, expectedArgCt, actualArgCt int) error {
	if expectedArgCt == 0 {
		return fmt.Errorf(ArgCountForm, cmdName, errs.ErrCLINoArgs, actualArgCt)
	}

	return fmt.Errorf(ArgCountForm, cmdName, errs.ErrCLIOneArg, actualArgCt)
}
