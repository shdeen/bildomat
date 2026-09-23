package main

import (
	"fmt"
	"strings"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// applyOutPathFormat applies the path's format to userInputs when the model supports it. It reports
// an adjustment if that format supersedes an explicitly supplied value.
func applyOutPathFormat(parts artifact.Location, rawOutPath string, model *catalog.Model, userInputs params.FlagInputs) []params.Adjustment {
	if parts.Format == "" || !model.SupportsParam(params.FlagTypeOutputFormat) {
		return nil
	}

	var records []params.Adjustment
	if explicit, supplied := userInputs[params.FlagTypeOutputFormat].(string); supplied &&
		!strings.EqualFold(strings.TrimSpace(explicit), parts.Format) {
		records = append(records, params.Adjustment{
			FlagID: params.FlagTypeOutputFormat, Type: params.ChangeIgnored,
			Comment: fmt.Sprintf(ReasonSupersededByOutputPath, rawOutPath),
		})
	}

	userInputs[params.FlagTypeOutputFormat] = parts.Format

	return records
}

// landingExt prefers the path's extension, then the adjusted format for a named file. An unnamed
// output or unavailable format leaves the extension unspecified.
func landingExt(parts artifact.Location, adjusted params.Values) (string, error) {
	if parts.Ext != "" {
		return parts.Ext, nil
	}

	if parts.Stem == "" {
		return "", nil
	}

	format, err := params.Value[string](adjusted, params.FlagTypeOutputFormat)
	if err != nil {
		return "", err
	}

	return artifact.FormatExt(format), nil
}

// applyLandingExt changes artifact extensions in place only within the same format. It reports each
// distinct incompatible extension once; an empty request changes nothing.
func applyLandingExt(artifacts []artifact.Media, requestedExt string) []params.Adjustment {
	if requestedExt == "" {
		return nil
	}

	var records []params.Adjustment

	recordedExts := map[string]bool{}

	for i := range artifacts {
		truthfulExt := artifacts[i].FileExt
		if canonExt(requestedExt) == canonExt(truthfulExt) {
			artifacts[i].FileExt = requestedExt

			continue
		}

		if !recordedExts[truthfulExt] {
			recordedExts[truthfulExt] = true
			records = append(records, params.Adjustment{
				FlagID: RunFlagOutputPath, Type: params.ChangeDerived,
				InputVal: requestedExt, WireVal: truthfulExt,
				Comment: ReasonContentOwnFormat,
			})
		}
	}

	return records
}

// canonExt normalizes extension case and treats .jpeg and .jpg as the same format.
func canonExt(ext string) string {
	lower := strings.ToLower(ext)
	if lower == "."+media.FormatJPEG {
		return "." + media.FormatJPG
	}

	return lower
}

// resolveOutputTarget resolves the output directory and applies the output path's format to
// userInputs. It uses the configured directory only when the run omitted an output path.
func resolveOutputTarget(genInputs *RunFlags, model *catalog.Model, userInputs params.FlagInputs, defaultOutDir string) (artifact.Location, []params.Adjustment, error) {
	rawOutPath, outPathGiven := genInputs.OutPath.ValIf()

	location := artifact.ParseOutPath(rawOutPath)
	if !outPathGiven {
		location.Dir = defaultOutDir
	}

	if err := location.Resolve(); err != nil {
		return artifact.Location{}, nil, err
	}

	return location, applyOutPathFormat(location, rawOutPath, model, userInputs), nil
}
