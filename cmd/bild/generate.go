// File: cmd/bild/generate.go
// The stages of one generation, from the model input to the written files,
// as the generate command runs them in order.

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

// adjustRequest preserves completed preparation and records early changes, including on failure.
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

// recordPreparation saves final descriptive facts separately from the captured provider payloads.
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

// writeRunResults recreates the output directory, writes media and any requested
// thoughts sidecar, and retains every saved path for final reporting. Returned files
// contain only media because record persistence must not treat a sidecar as media.
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
	// Artifact names are final here, after extension and collision handling.
	// Later naming changes must preserve this stem and these paths for the
	// generation record; retained-only files never enter artifact reporting.
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

// cleanupGeneration joins owned cleanup with the return error. The error pointer
// lets the early defer preserve failures from stages that return before persistence.
func cleanupGeneration(artifacts []artifact.Media, generationErr *error) { //nolint:gocritic // A deferred cleanup must update the named return error before it leaves the function.
	*generationErr = errors.Join(*generationErr, artifact.Cleanup(artifacts))
}
