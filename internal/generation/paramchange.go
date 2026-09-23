package generation

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// limitMedia copies the retained prefix of input media and records any reduction. A nonpositive
// limit retains every input.
func limitMedia(mediaInputs []media.Input, limit int) ([]media.Input, *params.Adjustment) {
	if limit <= 0 || len(mediaInputs) <= limit {
		return slices.Clone(mediaInputs), nil
	}

	change := inputCapRecord(len(mediaInputs), limit)

	return slices.Clone(mediaInputs[:limit]), &change
}

// inputCapRecord describes the reduction from supplied source count to the model limit.
func inputCapRecord(inputCount, limit int) params.Adjustment {
	return params.Adjustment{
		FlagID: params.FlagTypeInputMedia, Type: params.ChangeCapped,
		InputVal: strconv.Itoa(inputCount), WireVal: strconv.Itoa(limit),
		Comment: fmt.Sprintf(params.BoundMaxForm, strconv.Itoa(limit)),
	}
}

// SelectSources returns only the arguments a model retains, before any source is opened.
func SelectSources(sources []string, model *catalog.Model) []string {
	definition, supported := model.Param(params.FlagTypeInputMedia)
	if !supported {
		return nil
	}

	if definition.MaxMultiple > 0 && len(sources) > definition.MaxMultiple {
		sources = sources[:definition.MaxMultiple]
	}

	return slices.Clone(sources)
}

// AdjustGeneration selects owned media records and applies common scalar adjustments once.
// Completed records survive any later scalar failure, in their established presentation order.
func AdjustGeneration(model *catalog.Model, inputs params.FlagInputs, mediaInputs []media.Input) (Preparation, error) {
	preparedGeneration := Preparation{Changes: ignoredParamRecords(inputs, model)}
	retainedMedia, capRecords := capInputMedia(mediaInputs, model)
	preparedGeneration.InputMedia = retainedMedia

	if sources, supplied := inputs[params.FlagTypeInputMedia].([]string); supplied {
		if definition, supported := model.Param(params.FlagTypeInputMedia); supported && definition.MaxMultiple > 0 && len(sources) > definition.MaxMultiple {
			capRecords = []params.Adjustment{inputCapRecord(len(sources), definition.MaxMultiple)}
		}
	}

	preparedGeneration.Changes = append(preparedGeneration.Changes, capRecords...)
	adjusted, scalarChanges, err := params.Adjust(inputs, model.Params, model.ID)
	preparedGeneration.Params = adjusted
	preparedGeneration.Changes = append(preparedGeneration.Changes, scalarChanges...)

	return preparedGeneration, err
}

// ignoredParamRecords takes supplied parameter values and a model and returns one ignored record
// for each meaningful parameter that the model does not accept. Records are ordered by parameter
// name.
func ignoredParamRecords(inputs params.FlagInputs, model *catalog.Model) []params.Adjustment {
	suppliedParams := make([]params.FlagType, 0, len(inputs))
	for param := range inputs {
		suppliedParams = append(suppliedParams, param)
	}

	slices.Sort(suppliedParams)

	var changeRecords []params.Adjustment

	for _, param := range suppliedParams {
		if !inputs.Supplied(param) || model.SupportsParam(param) {
			continue
		}

		changeRecords = append(changeRecords, params.Adjustment{FlagID: param, Type: params.ChangeIgnored, Comment: fmt.Sprintf(ReasonNotConsumed, model.ID)})
	}

	return changeRecords
}

// capInputMedia copies the media accepted by the model and records any count limit. It returns no
// media when the model does not accept inputs.
func capInputMedia(mediaInputs []media.Input, model *catalog.Model) ([]media.Input, []params.Adjustment) {
	inputMediaConfig, ok := model.Param(params.FlagTypeInputMedia)
	if !ok {
		return nil, nil
	}

	keptMedia, capRecord := limitMedia(mediaInputs, inputMediaConfig.MaxMultiple)
	if capRecord != nil {
		return keptMedia, []params.Adjustment{*capRecord}
	}

	return keptMedia, nil
}
