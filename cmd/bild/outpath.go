// File: cmd/bild/outpath.go
// The --output-path orchestration: directory selection, the extension's format
// request against the resolved model, and the decision on the saved file's extension from
// the adjusted values.

package main

import (
	"fmt"
	"strings"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// applyOutPathFormat takes the parsed output path, the output path as typed,
// the resolved model, and the supplied parameter values, and returns any
// params.Adjustment records the format request produced. It returns no records
// when the path carries no format token, or when the model does not consume the
// output format.
//
// applyOutPathFormat also writes the extension's format token into the supplied
// parameter values as the effective output format, superseding a differing
// value that the user supplied.
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

// landingExt takes the parsed output path and the run's adjusted parameter
// values, and returns the extension requested for the written artifacts: the
// output path's own supported extension, or the canonical extension of the
// adjusted output format when the path names a file without one. It returns an
// empty string when neither applies, and the parameter error when the adjusted
// format is stored with another type.
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

// applyLandingExt takes the generated artifacts and a requested file extension,
// and returns one params.Adjustment record for each distinct artifact extension
// that the request could not override.
//
// An empty request is the ordinary case, not an error: the run requested no
// particular extension — no --output-path was given, the path named a bare
// directory, or the model consumes no output format — so there is nothing to
// apply and no records to return.
//
// applyLandingExt also mutates the caller's slice in place: it overwrites the
// FileExt field of each artifact.Media element whose own extension names the
// same format class as the requested one.
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

// canonExt takes a file extension and returns its format class: lowercased,
// with the two jpeg spellings folded into one.
func canonExt(ext string) string {
	lower := strings.ToLower(ext)
	if lower == "."+media.FormatJPEG {
		return "." + media.FormatJPG
	}

	return lower
}

// resolveOutputTarget resolves one output location and applies its format request.
// The configured directory is used only when the run supplied no output path.
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
