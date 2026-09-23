package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"
)

// adjustRequest prepares provider inputs and appends their changes to outcome. It also updates
// record, when present, even when preparation returns partial results and an error.
func adjustRequest(generator generation.Generator, pair *catalog.ProvModelPair, inputs params.FlagInputs, inputMedia []media.Input, pathChanges []params.Adjustment, paramFlags []params.Flag, outcome *output.GenerationOutcome, record *metadata.Record, reuse *metadata.Reuse) (generation.Preparation, error) {
	preparedGeneration, err := generator.AdjustParams(&pair.Model, inputs, inputMedia, reuse)

	preparedGeneration.Changes = slices.Concat(pathChanges, preparedGeneration.Changes)
	if err != nil && len(preparedGeneration.Params) == 0 && len(preparedGeneration.InputMedia) == 0 && len(preparedGeneration.Changes) == 0 {
		return preparedGeneration, err
	}

	recordErr := recordPreparation(&preparedGeneration, record)
	outcome.Adjustments = append(outcome.Adjustments, output.AdjustmentRecords(paramFlags, preparedGeneration.Changes, runFlagNames)...)

	return preparedGeneration, errors.Join(err, recordErr)
}

// recordPreparation copies adjusted inputs and changes into the optional in-memory record.
//
//nolint:nilaway // Callers pass the address of an owned Preparation value, which cannot be nil.
func recordPreparation(preparedGeneration *generation.Preparation, record *metadata.Record) error {
	if record == nil {
		return nil
	}

	adjusted, err := json.Marshal(preparedGeneration.Params)
	if err != nil {
		return fmt.Errorf("%q: %w, %w", record.Model, errs.ErrJSONEncode, err)
	}

	changes, err := json.Marshal(preparedGeneration.Changes)
	if err != nil {
		return fmt.Errorf("%q: %w, %w", record.Model, errs.ErrJSONEncode, err)
	}

	record.Request.Adjusted, record.Request.Adjustments = adjusted, changes
	record.Prepared(preparedGeneration.InputMedia)

	return nil
}

// writeRunResults creates the output directory if missing and saves media and any requested
// thoughts sidecar. It adds every saved file to outcome but returns only media files for record
// persistence.
//
//nolint:nilaway // Callers pass the address of an owned Result value, which cannot be nil.
func writeRunResults(result *generation.Result, outDir, stem, extOverride string, run *generation.Generation, paramFlags []params.Flag, outcome *output.GenerationOutcome) (finalStem string, completedFiles []output.SavedFile, resultErr error) {
	if err := artifact.CreateDir(outDir); err != nil {
		return stem, nil, err
	}

	landingChanges := applyLandingExt(result.Artifacts, extOverride)
	outcome.Notices = append(outcome.Notices, output.NoticeTexts(paramFlags, landingChanges, runFlagNames)...)

	thoughtsRequested, err := params.Value[bool](run.Params, params.FlagTypeThoughts)
	if err != nil {
		return stem, nil, err
	}

	resolvedStem, savedFiles, err := artifact.WriteMedia(outDir, stem, run.Model.Media, result.Artifacts, thoughtsRequested)
	// Extension and collision handling have determined the final artifact names. Record
	// persistence must use this stem and these paths without reporting the record as an
	// artifact.
	outcome.Artifacts = append(outcome.Artifacts, savedFiles...)

	if len(savedFiles) != len(result.Artifacts) || !thoughtsRequested {
		return resolvedStem, savedFiles, err
	}

	content := output.RenderSidecar(run.Prompt, run.Model.ID, paramFlags, run.Params, run.InputMedia, result.Thoughts)

	sidecar, sidecarErr := artifact.Write(outDir, resolvedStem, artifact.SidecarExt, content)
	if sidecarErr != nil {
		return resolvedStem, savedFiles, errors.Join(err, sidecarErr)
	}

	outcome.Artifacts = append(outcome.Artifacts, sidecar)

	return resolvedStem, savedFiles, err
}

// cleanupGeneration removes owned temporary artifacts and joins failures into generationErr.
func cleanupGeneration(artifacts []artifact.Media, generationErr *error) { //nolint:gocritic // A deferred cleanup must update the named return error before it leaves the function.
	*generationErr = errors.Join(*generationErr, artifact.Cleanup(artifacts))
}
