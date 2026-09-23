package provider

import (
	"fmt"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
)

// CheckOwnedPaths rejects configured assignments that overlap fields an adapter must construct. The
// adapter supplies its own protocol paths; generic configured-path consistency is checked when the
// catalog is decoded.
func CheckOwnedPaths(providerID string, model *catalog.Model, ownedPaths ...string) error {
	for parameterIndex := range model.Params {
		definition := &model.Params[parameterIndex]

		configuredPath := definition.ParamID
		if configuredPath == "" {
			continue
		}

		for _, ownedPath := range ownedPaths {
			if configuredPath == ownedPath || strings.HasPrefix(configuredPath, ownedPath+".") || strings.HasPrefix(ownedPath, configuredPath+".") {
				return &errs.ConfigError{
					Provider: providerID, Setting: string(definition.FlagID),
					Problem: fmt.Sprintf(catalog.RequestPathConflict, model.ID, configuredPath, ownedPath),
					Cause:   errs.ErrProvConfigParamUnplaced,
				}
			}
		}
	}

	return nil
}
