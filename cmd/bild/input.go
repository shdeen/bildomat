package main

import (
	"math"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
	"github.com/urfave/cli/v3"
)

// RunFlags contains generation values that are independent of model params.
//   - Model: the model specifier supplied by the user
//   - OutPath: the requested output path
//   - Prompt: the generation prompt
//   - PersistRecord: experimental opt-in retention of provider transactions
//   - Reuse: experimental selection of retained provider data
type RunFlags struct {
	Model         params.Nullable[string]
	OutPath       params.Nullable[string]
	Prompt        params.Nullable[string]
	PersistRecord bool
	Reuse         params.Nullable[string]
}

// createRunInputs takes the root command and returns the run-surface inputs
// (RunFlags) that it carries: the model, output path, and prompt. The
// prompt is the first argument without its surrounding whitespace. A flag
// that the user did not pass stays unset, which is distinct from being set
// to its zero value.
func createRunInputs(cmd *cli.Command) RunFlags {
	return RunFlags{
		Model:         params.GetSetIf(cmd.IsSet(RunFlagModel), cmd.String(RunFlagModel)),
		OutPath:       params.GetSetIf(cmd.IsSet(RunFlagOutputPath), cmd.String(RunFlagOutputPath)),
		Prompt:        params.GetSetIf(true, strings.TrimSpace(cmd.Args().Get(0))),
		PersistRecord: cmd.Bool(persistRecordFlag) || cmd.Bool(RunFlagDebug),
		Reuse:         params.GetSetIf(cmd.IsSet(reuseFlag), cmd.String(reuseFlag)),
	}
}

// createParamInputs takes a command and the parameter-flag enumeration, and
// returns the parameter values (params.FlagInputs) that the user supplied, keyed
// by flag ID. Presence means supplied, so an explicit zero value is preserved.
func createParamInputs(cmd *cli.Command, paramFlags []params.Flag) params.FlagInputs {
	paramFlagInputs := params.FlagInputs{}

	for i := range paramFlags {
		paramFlag := &paramFlags[i]

		flagID := string(paramFlag.FlagID)
		if !cmd.IsSet(flagID) {
			continue
		}

		paramFlagInputs[paramFlag.FlagID] = parsedFlagValue(cmd, paramFlag, flagID)
	}

	return paramFlagInputs
}

// getGenFlagsInput takes root generation inputs and parameter inputs and returns
// only the user-facing flags explicitly supplied on the command line.
func getGenFlagsInput(runFlags *RunFlags, paramInputs params.FlagInputs) map[string]any {
	flags := make(map[string]any, len(paramInputs)+2)

	if model, supplied := runFlags.Model.ValIf(); supplied {
		flags[RunFlagModel] = model
	}

	if outputPath, supplied := runFlags.OutPath.ValIf(); supplied {
		flags[RunFlagOutputPath] = outputPath
	}

	for flagName, flagValue := range paramInputs {
		if repeatedValues, ok := flagValue.([]string); ok {
			flagValue = slices.Clone(repeatedValues)
		}

		// A nonfinite number has no JSON form; its drop is reported among the
		// adjustments, so the record leaves it out rather than fail to encode.
		if number, ok := flagValue.(float64); ok && (math.IsNaN(number) || math.IsInf(number, 0)) {
			continue
		}

		flags[string(flagName)] = flagValue
	}

	return flags
}

// parsedFlagValue takes a command and one parameter flag, and returns that
// flag's value in the type that the command parsed it into: a string slice for
// a repeatable flag, and otherwise the string, number, integer, or boolean that
// the flag's declared data type names.
func parsedFlagValue(cmd *cli.Command, paramFlag *params.Flag, flagID string) any {
	if paramFlag.AllowMultiple {
		values := cmd.StringSlice(flagID)
		if paramFlag.FlagID == params.FlagTypeInputMedia {
			return inputMediaFlagValues(values)
		}

		return values
	}

	switch paramFlag.DataType {
	case params.DataString:
	case params.DataNumber:
		return cmd.Float(flagID)
	case params.DataInteger:
		return cmd.Int(flagID)
	case params.DataBoolean:
		return cmd.Bool(flagID)
	}

	return cmd.String(flagID)
}

// inputMediaFlagValues expands comma-separated local source lists while
// preserving complete HTTP(S) sources. A frame prefix may precede a URL.
func inputMediaFlagValues(values []string) []string {
	var sources []string

	for _, value := range values {
		if media.IsURLSource(value) {
			sources = append(sources, value)

			continue
		}

		sources = append(sources, strings.Split(value, ",")...)
	}

	return sources
}
