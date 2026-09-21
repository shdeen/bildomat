package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// The request encodings a configuration may declare for input media.
//   - InputMediaPayloadJSON: input media travels inside the JSON body
//   - InputMediaPayloadForm: input media travels as multipart form parts
const (
	InputMediaPayloadJSON = "json"
	InputMediaPayloadForm = "form"
)

// The configuration section names the validation faults cite.
//   - SectionImageAPI: the image API section, which provider packages also cite
//   - sectionVideoAPI: the video API section
//   - sectionAdapterPoll: the adapter's generation polling settings
//   - sectionAdapterFilePoll: the adapter's file polling settings
const (
	SectionImageAPI        = "ImageAPI"
	sectionVideoAPI        = "VideoAPI"
	sectionAdapterPoll     = "AdapterAPI poll"
	sectionAdapterFilePoll = "AdapterAPI file poll"
)

// Supported input media payload types and layouts.
//   - inputMediaPayloads: the accepted request encodings
//   - inputMediaStyles: the accepted JSON layouts
//
//nolint:gochecknoglobals // read-only value sets, written only at package load.
var (
	inputMediaPayloads = []string{InputMediaPayloadJSON, InputMediaPayloadForm}

	inputMediaStyles = []InputMediaStyle{InputMediaParts, InputMediaNested, InputMediaSingle, InputMediaString}
)

// ProviderConfig is a provider's request settings: the part of its
// configuration document the generators read and no user-facing document
// carries. The registered, undecoded form of the whole document is
// Source.
//   - StringParams: the parameters sent as strings whatever their data type
//   - ImageAPI: the optional image generation API description
//   - VideoAPI: the optional video generation API description
//   - AdapterAPI: the optional settings for a provider-specific adapter
type ProviderConfig struct {
	StringParams []params.FlagType `json:"stringParams,omitempty"`
	ImageAPI     *ImageAPI         `json:"imageAPI,omitempty"`
	VideoAPI     *VideoAPI         `json:"videoAPI,omitempty"`
	AdapterAPI   *AdapterAPI       `json:"adapterAPI,omitempty"`
}

// AdapterAPI contains endpoints, polling settings, status values, and fallback extensions for an adapter.
//   - APIBase: the base API endpoint
//   - PollInterval: the number of seconds between generation status requests
//   - PollTimeout: the maximum number of seconds allowed for generation polling
//   - FilePollInterval: the number of seconds between file status requests
//   - FilePollTimeout: the maximum number of seconds allowed for file polling
//   - PendingStatusText: the status that indicates pending work
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

// loadProviderConfig decodes a provider configuration document into the
// provider it declares and validates it against the parameter flag definitions.
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

// decodeProvConfig decodes one provider configuration document from encoded data.
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

// checkModelConfig returns an error when model has an invalid media value or parameter configuration.
func checkModelConfig(cfgName string, model *Model, flagsByID map[params.FlagType]params.Flag) error {
	if model.ID == "" {
		return createInvalidCfgError(cfgName, ModelIDMissing)
	}

	if model.Media != media.Image && model.Media != media.Video {
		return createInvalidCfgError(cfgName, fmt.Sprintf(ModelMediaInvalid, model.ID, string(model.Media)))
	}

	// A description is optional, since not every provider publishes one, but a
	// name is not: the model has no other human-readable identity to render.
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

// createInvalidCfgError returns a configuration validation error containing the configuration label and fault.
func createInvalidCfgError(cfgName, fault string) error {
	return &errs.ConfigError{Path: cfgName, Provider: strings.TrimSuffix(cfgName, ".json"), Problem: fault, Cause: errs.ErrProvConfigInvalid}
}

// checkProvConfig returns an error when a provider declares no request
// settings, when its models conflict with the parameter flags, or when its
// request settings are invalid.
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
		return createInvalidCfgError(cfgName, fmt.Sprintf(InputMediaPayloadTypeInvalid, sectionVideoAPI, ""))
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

// checkAdapterAPI returns an error when adapter settings lack required fallback extensions or polling values.
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

// checkVideoAPI returns an error when video API settings contain unsupported image settings or invalid polling values.
func checkVideoAPI(cfgName string, videoAPI *VideoAPI, vidDeclared bool) error {
	if err := checkInputMedia(cfgName, sectionVideoAPI, videoAPI.InputMediaPayloadType, videoAPI.InputMediaStyle, videoAPI.InputMediaProvParam, videoAPI.InputMediaListProvParam); err != nil {
		return err
	}

	if !vidDeclared {
		return nil
	}

	present, complete := videoAPI.FrameFields()
	if present && !complete {
		return createInvalidCfgError(cfgName, FrameMediaIncomplete)
	}

	return checkPolling(cfgName, sectionVideoAPI, videoAPI.PollInterval, videoAPI.PollTimeout, true)
}
