package generation

import (
	"bytes"
	"errors"
	"fmt"
	"image"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// dimensionsForm renders a width and height as a dimension value.
const dimensionsForm = "%dx%d"

// ConformInputMedia applies the model's image dimensions to owned parameter and media records.
// It returns only actual changes, including completed changes before a later failure.
// An unresolved first image defers size adoption until its bytes become available.
//
//nolint:funlen // Keep sequential size adoption and completed-change preservation visible in one loop.
func ConformInputMedia(model *catalog.Model, parameterValues params.Values, mediaInputs []media.Input) ([]params.Adjustment, error) {
	requestedSize, err := params.Value[string](parameterValues, params.FlagTypeSize)
	if err != nil {
		return nil, err
	}

	var changes []params.Adjustment

	for mediaIndex := range mediaInputs {
		mediaInput := &mediaInputs[mediaIndex]
		if mediaInput.Kind() == media.Video {
			continue
		}

		if mediaInput.URL != "" {
			if requestedSize == "" {
				return changes, nil
			}

			continue
		}

		dimensions, _, decodeErr := image.DecodeConfig(bytes.NewReader(mediaInput.Bytes))
		if decodeErr != nil {
			return changes, &errs.MediaError{Source: mediaInput.Filepath, Cause: errors.Join(errs.ErrInputMediaDecode, decodeErr)}
		}

		if requestedSize == "" {
			sizeDefinition, _ := model.Param(params.FlagTypeSize)

			requestedSize = params.PickSize(sizeDefinition.AllowedValues, dimensions.Width >= dimensions.Height, 0,
				params.GetSetIf(true, float64(dimensions.Width)/float64(dimensions.Height)))
			if requestedSize == "" {
				return changes, nil
			}

			parameterValues[params.FlagTypeSize] = requestedSize
			changes = append(changes, params.Adjustment{FlagID: params.FlagTypeSize, Type: params.ChangeDerived, WireVal: requestedSize, Comment: ReasonAdoptedFromReference})
		}

		changed, resizeErr := conformImage(mediaInput, dimensions, requestedSize)
		if resizeErr != nil {
			return changes, resizeErr
		}

		if !changed {
			continue
		}

		changes = append(changes, params.Adjustment{
			FlagID: params.FlagTypeInputMedia, Type: params.ChangeConformed,
			InputVal: fmt.Sprintf(dimensionsForm, dimensions.Width, dimensions.Height), WireVal: requestedSize,
			Comment: fmt.Sprintf(ReasonMatchedToRequest, model.ID),
		})
	}

	return changes, nil
}

// conformImage resizes an owned image record only when its decoded dimensions differ.
func conformImage(imageInput *media.Input, dimensions image.Config, requestedSize string) (bool, error) {
	width, height, valid := params.ParseDimensions(requestedSize)
	if !valid {
		return false, &errs.MediaError{Problem: requestedSize, Cause: errs.ErrInputMediaSize}
	}

	if dimensions.Width == width && dimensions.Height == height {
		return false, nil
	}

	resized, err := media.Resize(width, height, []media.Input{*imageInput})
	if err != nil {
		return false, err
	}

	*imageInput = resized[0] //nolint:nilaway // Resize received exactly one image and returned no error, so it returns exactly one replacement.

	return true, nil
}
