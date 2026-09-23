package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// The request encodings a configuration may declare for input media.
//   - inputMediaPayloadJSON: input media travels inside the JSON body
//   - InputMediaPayloadForm: input media travels as multipart form parts
const (
	inputMediaPayloadJSON = "json"
	InputMediaPayloadForm = "form"
)

// Configuration section names used in validation errors.
//   - SectionImageAPI: the shared image API settings
//   - SectionVideoAPI: the shared video API settings
//   - sectionAdapterPoll: the adapter's generation polling settings
//   - sectionAdapterFilePoll: the adapter's file polling settings
const (
	SectionImageAPI        = "ImageAPI"
	SectionVideoAPI        = "VideoAPI"
	sectionAdapterPoll     = "AdapterAPI poll"
	sectionAdapterFilePoll = "AdapterAPI file poll"
)

// Supported input media payload types and layouts.
//   - inputMediaPayloads: the accepted request encodings
//   - inputMediaStyles: the accepted JSON layouts
//
//nolint:gochecknoglobals // read-only value sets, written only at package load.
var (
	inputMediaPayloads = []string{inputMediaPayloadJSON, InputMediaPayloadForm}

	inputMediaStyles = []InputMediaStyle{InputMediaParts, InputMediaNested, InputMediaSingle, InputMediaString}
)

// ProviderConfig contains the request settings consumed by generators.
//   - StringParams: the parameters sent as strings regardless of their data type
//   - ImageAPI: the optional shared image API settings
//   - VideoAPI: the optional shared video API settings
//   - AdapterAPI: the optional settings for a provider-specific adapter
type ProviderConfig struct {
	StringParams []params.FlagType `json:"stringParams,omitempty"`
	ImageAPI     *ImageAPI         `json:"imageAPI,omitempty"`
	VideoAPI     *VideoAPI         `json:"videoAPI,omitempty"`
	AdapterAPI   *AdapterAPI       `json:"adapterAPI,omitempty"`
}

// AdapterAPI contains endpoints, polling settings, status values, and fallback extensions for an
// adapter.
//   - APIBase: the base API endpoint
//   - PollInterval: seconds between generation status requests
//   - PollTimeout: the generation polling budget in seconds
//   - FilePollInterval: seconds between file status requests
//   - FilePollTimeout: the file polling budget in seconds
//   - PendingStatusText: the statuses that indicate pending work
//   - ReadyStatusText: the status that indicates completed work
//   - FailedStatusText: the statuses that indicate failed work
//   - ImageFallbackExt: the fallback extension for downloaded images
//   - VideoFallbackExt: the fallback extension for downloaded videos
type AdapterAPI struct {
	APIBase           string      `json:"apiBase"`
	PollInterval      PollSeconds `json:"pollInterval"`
	PollTimeout       PollSeconds `json:"pollTimeout"`
	FilePollInterval  PollSeconds `json:"filePollInterval"`
	FilePollTimeout   PollSeconds `json:"filePollTimeout"`
	PendingStatusText []string    `json:"pendingStatusText"`
	ReadyStatusText   string      `json:"readyStatusText"`
	FailedStatusText  []string    `json:"failedStatusText"`
	ImageFallbackExt  string      `json:"imageFallbackExt"`
	VideoFallbackExt  string      `json:"videoFallbackExt"`
}

// loadProviderConfig decodes a provider configuration document into the provider it declares and
// validates it against the parameter flag definitions.
func loadProviderConfig(flags []params.Flag, cfgName string, provConfigBytes []byte) (Provider, error) {
	prov, err := decodeProvConfig(cfgName, provConfigBytes)
	if err != nil {
		return Provider{}, err
	}

	if err := checkProvConfig(cfgName, &prov, flags); err != nil {
		return Provider{}, err
	}

	return prov, nil
}

// decodeProvConfig decodes exactly one provider document, rejecting unknown fields and trailing
// data.
func decodeProvConfig(cfgName string, b []byte) (Provider, error) {
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()

	var prov Provider
	if err := decoder.Decode(&prov); err != nil {
		return Provider{}, &errs.ConfigError{Path: cfgName, Provider: strings.TrimSuffix(cfgName, ".json"), Cause: errors.Join(errs.ErrProvConfigDecode, err)} //nolint:goconst // The JSON filename suffix is syntax, not a shared vocabulary item.
	}

	switch err := decoder.Decode(new(json.RawMessage)); {
	case err == nil:
		return Provider{}, &errs.ConfigError{Path: cfgName, Provider: strings.TrimSuffix(cfgName, ".json"), Cause: errs.ErrProvConfigTrailing}
	case !errors.Is(err, io.EOF):
		return Provider{}, &errs.ConfigError{Path: cfgName, Provider: strings.TrimSuffix(cfgName, ".json"), Cause: errors.Join(errs.ErrProvConfigDecode, err)}
	}

	return prov, nil
}

// checkModelConfig validates model identity, media kind, and nonduplicated parameter definitions.
func checkModelConfig(cfgName string, model *Model, flagsByID map[params.FlagType]params.Flag) error {
	if model.ID == "" {
		return createInvalidCfgError(cfgName, ModelIDMissing)
	}

	if model.Media != media.Image && model.Media != media.Video {
		return createInvalidCfgError(cfgName, fmt.Sprintf(ModelMediaInvalid, model.ID, string(model.Media)))
	}

	// A description is optional, since not every provider publishes one, but a name is not: the
	// model has no other human-readable identity to render.
	if model.Name == "" {
		return createInvalidCfgError(cfgName, fmt.Sprintf(ModelNameMissing, model.ID))
	}

	seenFlagIDs := map[params.FlagType]bool{}

	for i := range model.Params {
		paramCfg := &model.Params[i]
		if seenFlagIDs[paramCfg.FlagID] {
			return createInvalidCfgError(cfgName, fmt.Sprintf(ModelFlagDuplicate, model.ID, string(paramCfg.FlagID)))
		}

		seenFlagIDs[paramCfg.FlagID] = true
		if err := checkParamCfg(cfgName, model.ID, paramCfg, flagsByID); err != nil {
			return err
		}
	}

	return nil
}

// createInvalidCfgError returns a configuration validation error containing the configuration label
// and fault.
func createInvalidCfgError(cfgName, fault string) error {
	return &errs.ConfigError{Path: cfgName, Provider: strings.TrimSuffix(cfgName, ".json"), Problem: fault, Cause: errs.ErrProvConfigInvalid}
}

// checkProvConfig returns an error when a provider declares no request settings, when its models
// conflict with the parameter flags, or when its request settings are invalid.
func checkProvConfig(cfgName string, prov *Provider, flags []params.Flag) error {
	if prov.Config == nil {
		return createInvalidCfgError(cfgName, ConfigMissing)
	}

	flagsByID := map[params.FlagType]params.Flag{}
	for i := range flags {
		flagsByID[flags[i].FlagID] = flags[i]
	}

	for _, stringParam := range prov.Config.StringParams {
		if _, ok := flagsByID[stringParam]; !ok {
			return createInvalidCfgError(cfgName, fmt.Sprintf(StringParamsUnknownFlag, string(stringParam)))
		}
	}

	hasImageModels, hasVideoModels, defaultModelFound := false, false, false

	for i := range prov.Models {
		model := &prov.Models[i]
		if err := checkModelConfig(cfgName, model, flagsByID); err != nil {
			return err
		}

		if err := checkRequestPaths(cfgName, model, prov.Config); err != nil {
			return err
		}

		if err := checkModelInputMedia(cfgName, model, prov.Config); err != nil {
			return err
		}

		hasImageModels = hasImageModels || model.Media == media.Image
		hasVideoModels = hasVideoModels || model.Media == media.Video
		defaultModelFound = defaultModelFound || model.ID == prov.DefaultModel
	}

	if prov.DefaultModel != "" && !defaultModelFound {
		return createInvalidCfgError(cfgName, fmt.Sprintf(ProviderDefaultModelUnknown, prov.ID, prov.DefaultModel))
	}

	if prov.Config.AdapterAPI != nil {
		return checkAdapterAPI(cfgName, prov.Config.AdapterAPI, hasImageModels, hasVideoModels)
	}

	return checkAPIs(cfgName, prov.Config, hasImageModels, hasVideoModels)
}

// checkModelInputMedia requires an encoding description when a shared API model accepts media.
// Adapters define their own request encodings instead of declaring them in ImageAPI or VideoAPI.
func checkModelInputMedia(cfgName string, model *Model, settings *ProviderConfig) error {
	if settings.AdapterAPI != nil || !model.SupportsParam(params.FlagTypeInputMedia) {
		return nil
	}

	if model.Media == media.Image && settings.ImageAPI != nil && settings.ImageAPI.InputMediaPayloadType == "" {
		return createInvalidCfgError(cfgName, fmt.Sprintf(InputMediaPayloadTypeInvalid, SectionImageAPI, ""))
	}

	if model.Media == media.Video && settings.VideoAPI != nil && settings.VideoAPI.InputMediaPayloadType == "" {
		return createInvalidCfgError(cfgName, fmt.Sprintf(InputMediaPayloadTypeInvalid, SectionVideoAPI, ""))
	}

	return nil
}

// checkAPIs returns an error when a required API description is missing or invalid.
func checkAPIs(cfgName string, provCfg *ProviderConfig, hasImageModels, hasVideoModels bool) error {
	if hasImageModels && provCfg.ImageAPI == nil {
		return createInvalidCfgError(cfgName, ImageAPIMissing)
	}

	if hasVideoModels && provCfg.VideoAPI == nil {
		return createInvalidCfgError(cfgName, VideoAPIMissing)
	}

	if provCfg.ImageAPI != nil {
		if err := checkInputMedia(cfgName, SectionImageAPI, provCfg.ImageAPI.InputMediaPayloadType, provCfg.ImageAPI.InputMediaStyle, provCfg.ImageAPI.InputMediaProvParam, provCfg.ImageAPI.InputMediaListProvParam); err != nil {
			return err
		}
	}

	if provCfg.VideoAPI != nil {
		if err := checkVideoAPI(cfgName, provCfg.VideoAPI, hasVideoModels); err != nil {
			return err
		}
	}

	return nil
}

// checkAdapterAPI returns an error when adapter settings lack required fallback extensions or
// polling values.
func checkAdapterAPI(cfgName string, adapterAPI *AdapterAPI, hasImageModels, hasVideoModels bool) error {
	if hasImageModels && adapterAPI.ImageFallbackExt == "" {
		return createInvalidCfgError(cfgName, ImageFallbackExtMissing)
	}

	if hasVideoModels && adapterAPI.VideoFallbackExt == "" {
		return createInvalidCfgError(cfgName, VideoFallbackExtMissing)
	}

	mainPolling := hasVideoModels || len(adapterAPI.PendingStatusText) > 0 || adapterAPI.ReadyStatusText != "" || len(adapterAPI.FailedStatusText) > 0
	if err := checkPolling(cfgName, sectionAdapterPoll, adapterAPI.PollInterval, adapterAPI.PollTimeout, mainPolling); err != nil {
		return err
	}

	return checkPolling(cfgName, sectionAdapterFilePoll, adapterAPI.FilePollInterval, adapterAPI.FilePollTimeout, false)
}

// checkInputMedia rejects unsupported or incompatible encodings and missing field names.
func checkInputMedia(cfgName, api, payload string, style InputMediaStyle, singularField, pluralField string) error {
	if payload == "" && style == "" && singularField == "" && pluralField == "" {
		return nil
	}

	if !slices.Contains(inputMediaPayloads, payload) {
		return createInvalidCfgError(cfgName, fmt.Sprintf(InputMediaPayloadTypeInvalid, api, payload))
	}

	if !slices.Contains(inputMediaStyles, style) {
		return createInvalidCfgError(cfgName, fmt.Sprintf(InputMediaStyleInvalid, api, string(style)))
	}

	if (payload == InputMediaPayloadForm) != (style == InputMediaParts) {
		return createInvalidCfgError(cfgName, fmt.Sprintf(InputMediaEncodingConflict, api, payload, style))
	}

	if singularField == "" || (style == InputMediaSingle && pluralField == "") {
		return createInvalidCfgError(cfgName, fmt.Sprintf(InputMediaParamMissing, api))
	}

	return nil
}

// checkVideoAPI validates input-media encoding, complete frame fields, and required polling
// settings.
func checkVideoAPI(cfgName string, videoAPI *VideoAPI, vidDeclared bool) error {
	if err := checkInputMedia(cfgName, SectionVideoAPI, videoAPI.InputMediaPayloadType, videoAPI.InputMediaStyle, videoAPI.InputMediaProvParam, videoAPI.InputMediaListProvParam); err != nil {
		return err
	}

	if !vidDeclared {
		return nil
	}

	present, complete := videoAPI.FrameFields()
	if present && !complete {
		return createInvalidCfgError(cfgName, FrameMediaIncomplete)
	}

	return checkPolling(cfgName, SectionVideoAPI, videoAPI.PollInterval, videoAPI.PollTimeout, true)
}

// Clone returns independent copies of the request settings and their nested collections.
func (config *ProviderConfig) Clone() ProviderConfig {
	settings := *config
	settings.StringParams = slices.Clone(config.StringParams)

	if config.ImageAPI != nil {
		imageAPI := *config.ImageAPI
		imageAPI.FixedProvFields = maps.Clone(imageAPI.FixedProvFields)
		settings.ImageAPI = &imageAPI
	}

	if config.VideoAPI != nil {
		videoAPI := *config.VideoAPI
		videoAPI.ProgressStatusText = slices.Clone(videoAPI.ProgressStatusText)
		videoAPI.FailedStatusText = slices.Clone(videoAPI.FailedStatusText)
		videoAPI.URLPathSeq = slices.Clone(videoAPI.URLPathSeq)
		settings.VideoAPI = &videoAPI
	}

	if config.AdapterAPI != nil {
		adapterAPI := *config.AdapterAPI
		adapterAPI.PendingStatusText = slices.Clone(adapterAPI.PendingStatusText)
		adapterAPI.FailedStatusText = slices.Clone(adapterAPI.FailedStatusText)
		settings.AdapterAPI = &adapterAPI
	}

	return settings
}

// requestPaths lists the configurable assignments used by one model.
func (config *ProviderConfig) requestPaths(model *Model) []string {
	requestPaths := make([]string, 0, len(model.Params))
	for parameterIndex := range model.Params {
		definition := &model.Params[parameterIndex]
		if definition.ParamID != "" {
			requestPaths = append(requestPaths, definition.ParamID)
		}
	}

	if config.AdapterAPI == nil {
		if model.Media == media.Image && config.ImageAPI != nil {
			api := config.ImageAPI

			requestPaths = append(requestPaths, api.InputMediaProvParam, api.InputMediaListProvParam)
			for fieldPath := range api.FixedProvFields {
				requestPaths = append(requestPaths, fieldPath)
			}
		}

		if model.Media == media.Video && config.VideoAPI != nil {
			api := config.VideoAPI
			requestPaths = append(requestPaths, api.InputMediaProvParam, api.InputMediaListProvParam, api.FrameMediaProvParam)
		}
	}

	return requestPaths
}
