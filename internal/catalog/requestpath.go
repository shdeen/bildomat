package catalog

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/media"
)

// checkRequestPaths rejects empty segments and overlapping declared assignments.
// Sibling paths share their parent object; an assigned value cannot also be a parent.
func checkRequestPaths(cfgName string, model *Model, providerSettings *ProviderConfig) error {
	requestPaths := providerSettings.requestPaths(model)

	slices.Sort(requestPaths)

	assignedPaths := make(map[string]bool, len(requestPaths))
	for _, requestPath := range requestPaths {
		if requestPath == "" {
			continue
		}

		if slices.Contains(strings.Split(requestPath, "."), "") {
			return createInvalidCfgError(cfgName, fmt.Sprintf(RequestPathInvalid, model.ID, requestPath))
		}

		if assignedPaths[requestPath] {
			return createInvalidCfgError(cfgName, fmt.Sprintf(RequestPathConflict, model.ID, requestPath, requestPath))
		}

		for segmentEnd, character := range requestPath {
			if character != '.' {
				continue
			}

			parentPath := requestPath[:segmentEnd]
			if assignedPaths[parentPath] {
				return createInvalidCfgError(cfgName, fmt.Sprintf(RequestPathConflict, model.ID, parentPath, requestPath))
			}
		}

		assignedPaths[requestPath] = true
	}

	return nil
}

// requestPaths lists the configurable assignments used by one model.
func (providerSettings *ProviderConfig) requestPaths(model *Model) []string {
	requestPaths := make([]string, 0, len(model.Params))
	for parameterIndex := range model.Params {
		definition := &model.Params[parameterIndex]
		if definition.ParamID != "" {
			requestPaths = append(requestPaths, definition.ParamID)
		}
	}

	if providerSettings.AdapterAPI == nil {
		if model.Media == media.Image && providerSettings.ImageAPI != nil {
			api := providerSettings.ImageAPI

			requestPaths = append(requestPaths, api.InputMediaProvParam, api.InputMediaListProvParam)
			for fieldPath := range api.FixedProvFields {
				requestPaths = append(requestPaths, fieldPath)
			}
		}

		if model.Media == media.Video && providerSettings.VideoAPI != nil {
			api := providerSettings.VideoAPI
			requestPaths = append(requestPaths, api.InputMediaProvParam, api.InputMediaListProvParam, api.FrameMediaProvParam)
		}
	}

	return requestPaths
}
