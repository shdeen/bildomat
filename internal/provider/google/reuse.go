package google

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// VeoExtendID identifies provisional, experimental Veo extension. Its selection
// syntax and the generation-record format are subject to change.
const VeoExtendID = "veo-extend"

// Veo extension accepts full and Fast targets, 720p sources, and at most 141
// seconds of source video. The request duration is always eight seconds.
const (
	veoExtensionModel            = "veo-3.1-generate-preview"
	veoExtensionFast             = "veo-3.1-fast-generate-preview"
	veoExtensionResolution       = "720p"
	veoExtensionMaxSourceSeconds = 141
	googleVideoHost              = "generativelanguage.googleapis.com"
	googleVideoScheme            = "https"
)

// reuseSourceSettings decodes restrictions known from the earlier generation.
type reuseSourceSettings struct {
	Resolution string  `json:"resolution"`
	Aspect     string  `json:"aspect-ratio"`
	Duration   float64 `json:"duration"`
}

// adjustReuse resolves an experimental Veo extension reference and adjusts its
// required output settings before parameter notices are printed. The caller
// supplies a non-nil reuse selection.
func adjustReuse(model *catalog.Model, parameterValues params.Values, inputs []media.Input, reuse *metadata.Reuse) (string, []params.Adjustment, error) {
	if len(inputs) != 0 {
		return "", nil, fmt.Errorf("%q: %w", VeoExtendID, errs.ErrReuseInputMedia)
	}

	if model == nil || (model.ID != veoExtensionModel && model.ID != veoExtensionFast) || parameterValues == nil {
		return "", nil, fmt.Errorf("%q: %w", VeoExtendID, errs.ErrReuseVideoModel)
	}

	videoURI := reuse.URI
	if reuse.Record != nil {
		var err error

		videoURI, err = recordVideoURI(reuse.Record)
		if err != nil {
			return "", nil, err
		}
	}

	if !originalVideoURI(videoURI) {
		return "", nil, fmt.Errorf("%q: %w", videoURI, errs.ErrReuseVideoURI)
	}

	var changes []params.Adjustment

	for _, required := range []struct {
		flag  params.FlagType
		value any
	}{
		{params.FlagTypeDuration, forcedDurationSeconds},
		{params.FlagTypeResolution, veoExtensionResolution},
	} {
		previous := ""

		if supplied, present := parameterValues[required.flag]; present {
			if supplied == required.value {
				continue
			}

			previous = params.FormatValue(supplied)
		}

		parameterValues[required.flag] = required.value
		changes = append(changes, params.Adjustment{
			FlagID: required.flag, Type: params.ChangeForced,
			InputVal: previous, WireVal: params.FormatValue(required.value),
		})
	}

	return videoURI, changes, nil
}

// recordVideoURI selects one distinct original video without reading local media.
func recordVideoURI(record *metadata.Record) (string, error) {
	if record.Provider != ProviderID || !strings.HasPrefix(record.Model, familyVeo+"-") {
		return "", fmt.Errorf("%q: %w", record.Provider+"/"+record.Model, errs.ErrReuseVideoModel)
	}

	if err := checkReuseSource(record.Request.Adjusted); err != nil {
		return "", err
	}

	videoURIs := responseVideoURIs(record.Responses)
	for _, retained := range record.Returns {
		if retained.Provider != ProviderID || retained.Model != record.Model {
			continue
		}

		var completed retainedOperation
		if err := json.Unmarshal(retained.Data, &completed); err != nil {
			return "", fmt.Errorf("%q: %w, %w", record.Model, errs.ErrRecordInvalid, err)
		}

		for _, video := range completed.Videos {
			if originalVideoURI(video.URI) && !slices.Contains(videoURIs, video.URI) {
				videoURIs = append(videoURIs, video.URI)
			}
		}
	}

	if len(videoURIs) > 1 {
		return "", fmt.Errorf("%q: %w", record.Model, errs.ErrReuseVideoMultiple)
	}

	if len(videoURIs) == 0 {
		return "", fmt.Errorf("%q: %w", record.Model, errs.ErrReuseVideoURI)
	}

	return videoURIs[0], nil
}

// checkReuseSource rejects restrictions contradicted by available metadata.
// Omitted fields do not prove eligibility; Google validates the original video.
func checkReuseSource(adjusted json.RawMessage) error {
	if len(adjusted) == 0 {
		return nil
	}

	var settings reuseSourceSettings
	if err := json.Unmarshal(adjusted, &settings); err != nil {
		return fmt.Errorf("%q: %w, %w", VeoExtendID, errs.ErrRecordInvalid, err)
	}

	if settings.Resolution != "" && settings.Resolution != veoExtensionResolution ||
		settings.Aspect != "" && settings.Aspect != "16:9" && settings.Aspect != "9:16" ||
		settings.Duration > veoExtensionMaxSourceSeconds || settings.Duration < 0 {
		return fmt.Errorf("%q: %w", string(adjusted), errs.ErrReuseVideoModel)
	}

	return nil
}

// responseVideoURIs extracts references from completed operation bodies. This
// also detects multiple videos when the extracted return values are incomplete.
// Non-operation responses are ignored.
func responseVideoURIs(responses []metadata.Response) []string {
	var videoURIs []string

	for responseIndex := range responses {
		var completed operation
		if json.Unmarshal(responses[responseIndex].Body, &completed) != nil || !completed.Done || completed.Error != nil || completed.Response == nil || completed.Response.GenerateVideoResponse == nil {
			continue
		}

		for _, sample := range completed.Response.GenerateVideoResponse.GeneratedSamples {
			if sample.Video != nil && originalVideoURI(sample.Video.URI) && !slices.Contains(videoURIs, sample.Video.URI) {
				videoURIs = append(videoURIs, sample.Video.URI)
			}
		}
	}

	return videoURIs
}

// originalVideoURI checks the form of a Google Files reference without replacing
// or fetching it. Only Google can confirm its provenance and current lifetime.
func originalVideoURI(videoURI string) bool {
	reference, err := url.Parse(videoURI)

	return err == nil && reference.Scheme == googleVideoScheme && reference.Hostname() == googleVideoHost && reference.User == nil && fileIDFromURI(videoURI) != "" //nolint:nilaway // url.Parse returns a non-nil URL whenever its error is nil.
}
