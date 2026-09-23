package catalog

import (
	"fmt"
	"slices"
	"strings"
)

// checkRequestPaths rejects empty segments and overlapping declared assignments. Sibling paths
// share their parent object; an assigned value cannot also be a parent.
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
