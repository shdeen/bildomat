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
// Preparation returns completed changes even when a later operation fails.
type Generator interface {
	AdjustParams(model *catalog.Model, inputs params.FlagInputs, mediaInputs []media.Input, reuse *metadata.Reuse) (Preparation, error)
	Generate(ctx context.Context, run *Generation) (Result, error)
}

// Preparation owns the values and media records to be submitted to a provider.
// Bytes and timestamps referenced by InputMedia remain immutable; transformations replace them.
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

// Clone returns independently mutable parameter, media-record, and adjustment collections.
// Referenced bytes and timestamps retain their immutable ownership contract.
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
// Prompt and APIKey are resolved by the command. Record retains the optional transaction
// history independently of artifact ownership; the command persists it.
type Generation struct {
	catalog.ProvModelPair
	Preparation

	Prompt string
	APIKey string
	Record *metadata.Record
}
