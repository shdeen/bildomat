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

// createRunInputs captures generation controls and trims the prompt. Optional string flags retain
// the distinction between omitted and explicitly empty.
func createRunInputs(cmd *cli.Command) RunFlags {
	return RunFlags{
		Model:         params.GetSetIf(cmd.IsSet(RunFlagModel), cmd.String(RunFlagModel)),
		OutPath:       params.GetSetIf(cmd.IsSet(RunFlagOutputPath), cmd.String(RunFlagOutputPath)),
		Prompt:        params.GetSetIf(true, strings.TrimSpace(cmd.Args().Get(0))),
		PersistRecord: cmd.Bool(persistRecordFlag) || cmd.Bool(RunFlagDebug),
		Reuse:         params.GetSetIf(cmd.IsSet(reuseFlag), cmd.String(reuseFlag)),
	}
}

// createParamInputs collects supplied parameter flags, preserving explicit zero values.
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

// getGenFlagsInput copies supplied model, output-path, and parameter flags for reporting. It omits
// nonfinite numbers, which cannot be encoded as JSON.
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

		// JSON cannot encode nonfinite numbers. Omit them from submitted flags; parameter
		// adjustment reports why they were rejected.
		if number, ok := flagValue.(float64); ok && (math.IsNaN(number) || math.IsInf(number, 0)) {
			continue
		}

		flags[string(flagName)] = flagValue
	}

	return flags
}

// parsedFlagValue reads a flag in its declared type and expands local input-media lists.
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

// inputMediaFlagValues expands comma-separated local source lists while preserving complete HTTP(S)
// sources. A frame prefix may precede a URL.
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
