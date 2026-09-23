// Package generation owns provider-neutral generation requests and media preparation.
package generation

import (
	"context"
	"maps"
	"slices"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// Generator prepares a model's retained inputs and generates its artifacts.
//   - AdjustParams: prepare inputs, retaining completed changes even when preparation fails
//   - Generate: submit prepared inputs and collect the provider's result
type Generator interface {
	AdjustParams(model *catalog.Model, inputs params.FlagInputs, mediaInputs []media.Input, reuse *metadata.Reuse) (Preparation, error)
	Generate(ctx context.Context, run *Generation) (Result, error)
}

// Preparation owns the values and media records to be submitted to a provider. Bytes and timestamps
// referenced by InputMedia remain immutable; transformations replace them.
//   - Params: the adjusted model parameters
//   - InputMedia: the retained inputs, in submission order
//   - Changes: the ordered reasons that supplied values changed
//   - ReuseURI: the original provider reference for a follow-up request
type Preparation struct {
	Params     params.Values
	InputMedia []media.Input
	Changes    []params.Adjustment
	ReuseURI   string
}

// Clone copies parameter values, media records, and adjustment records for independent mutation.
// Media bytes and timestamps remain shared and must stay immutable.
func (preparedGeneration *Preparation) Clone() Preparation {
	clonedPreparation := *preparedGeneration

	clonedPreparation.Params = maps.Clone(preparedGeneration.Params)
	if clonedPreparation.Params == nil {
		clonedPreparation.Params = params.Values{}
	}

	for flagID, parameterValue := range clonedPreparation.Params {
		if stringValues, present := parameterValue.([]string); present {
			clonedPreparation.Params[flagID] = slices.Clone(stringValues)
		}
	}

	clonedPreparation.InputMedia = slices.Clone(preparedGeneration.InputMedia)
	clonedPreparation.Changes = slices.Clone(preparedGeneration.Changes)

	return clonedPreparation
}

// Generation contains the selected model and prepared inputs for one provider request.
//   - ProvModelPair: the selected provider and model
//   - Preparation: the adjusted parameters and retained media
//   - Prompt: the resolved generation prompt
//   - APIKey: the resolved provider credential
//   - Record: optional transaction history retained for the command to persist
type Generation struct {
	catalog.ProvModelPair
	Preparation

	Prompt string
	APIKey string
	Record *metadata.Record
}
