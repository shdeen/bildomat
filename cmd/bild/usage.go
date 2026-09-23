package main

import (
	"fmt"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/urfave/cli/v3"
)

// combinedFlagError rejects help or version beside a positional argument. It names the long flag
// because the parser does not retain the typed spelling.
func combinedFlagError(flagName string) error {
	return &errs.UsageError{Text: fmt.Sprintf(HelpFlagCombinedForm, "--"+flagName), Cause: errs.ErrCLIHelpCombined}
}

// searchTermsError validates search-term presence and count. An explicitly empty term is invalid,
// distinct from an omitted term.
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

// argCountError reports a violation of a command's zero- or one-argument requirement.
func argCountError(cmdName string, expectedArgCt, actualArgCt int) error {
	if expectedArgCt == 0 {
		return fmt.Errorf(ArgCountForm, cmdName, errs.ErrCLINoArgs, actualArgCt)
	}

	return fmt.Errorf(ArgCountForm, cmdName, errs.ErrCLIOneArg, actualArgCt)
}
