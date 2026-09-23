// Command bild generates and edits images and videos through supported AI providers.
package main

import (
	"context"
	"errors"
	"os"
	"reflect"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/provider"
	"github.com/shdeen/bildomat/internal/provider/bfl"
	"github.com/shdeen/bildomat/internal/provider/config"
	"github.com/shdeen/bildomat/internal/provider/google"
	"github.com/shdeen/bildomat/internal/provider/kling"
	"github.com/shdeen/bildomat/internal/provider/sourceful"
)

// devVersion is the version reported when the build supplies no version.
const devVersion = "0.0.0-dev"

// Release metadata that the build script injects with -ldflags "-X".
//   - AppName: the application name, which no code reads
//   - AppVersion: the version that the command reports, or devVersion when the build injects
//     nothing; see appVersion
//   - AppBuildDate: the date of the build, which no code reads
//
//nolint:gochecknoglobals // set by the build script through -ldflags "-X", which writes only to a package-level variable.
var (
	AppName, AppVersion, AppBuildDate string
)

// providerRegistration binds a provider's description to its executable operations.
//   - ProviderID: the canonical catalog identifier
//   - ConfigBytes: the embedded provider description
//   - New: the adapter constructor
//   - ReuseIDs: the provider's accepted experimental reuse operations
type providerRegistration struct {
	ProviderID  string
	ConfigBytes []byte
	New         func(*catalog.Provider) (generation.Generator, error)
	ReuseIDs    []string
}

// providerRegistrations returns every provider registration in catalog load order.
func providerRegistrations() []providerRegistration {
	return []providerRegistration{
		{ProviderID: config.IDOpenAI, ConfigBytes: config.JSONOpenAI, New: provider.NewProvider},
		{ProviderID: config.IDXAI, ConfigBytes: config.JSONXAI, New: provider.NewProvider},
		{ProviderID: config.IDOpenRouter, ConfigBytes: config.JSONOpenRouter, New: provider.NewProvider},
		{ProviderID: google.ProviderID, ConfigBytes: google.ConfigJSON, New: google.NewProvider, ReuseIDs: []string{google.VeoExtendID}},
		{ProviderID: bfl.ProviderID, ConfigBytes: bfl.ConfigJSON, New: bfl.NewProvider},
		{ProviderID: sourceful.ProviderID, ConfigBytes: sourceful.ConfigJSON, New: sourceful.NewProvider},
		{ProviderID: config.IDRecraft, ConfigBytes: config.JSONRecraft, New: provider.NewProvider},
		{ProviderID: kling.ProviderID, ConfigBytes: kling.ConfigJSON, New: kling.NewProvider},
	}
}

// main runs the command and exits with its classified status.
func main() {
	command := createCommand(&bildApp{})
	runErr := command.Run(context.Background(), os.Args)
	os.Exit(runExitCode(runErr))
}

// runExitCode returns 0 for success, 2 for usage errors, and 1 for other failures.
func runExitCode(runErr error) int {
	switch {
	case runErr == nil:
		return 0
	case errors.Is(runErr, errs.ErrCLI):
		return 2
	default:
		return 1
	}
}

// appVersion returns the injected release version, falling back to devVersion.
func appVersion() string {
	if AppVersion == "" {
		return devVersion
	}

	return AppVersion
}

// newGenerator constructs a registered, loaded provider and rejects unusable constructors.
func newGenerator(loadedCatalog *catalog.Catalog, providerID string, registrations []providerRegistration) (generation.Generator, error) {
	description, loaded := loadedCatalog.Provider(providerID)
	if !loaded {
		return nil, &errs.ConfigError{Provider: providerID, Cause: errs.ErrProvConfigNotLoaded}
	}

	for _, registration := range registrations {
		if registration.ProviderID != providerID {
			continue
		}

		if registration.New == nil {
			return nil, &errs.ConfigError{Provider: providerID, Cause: errs.ErrProvConfigNoConstructor}
		}

		generator, err := registration.New(&description)
		if err != nil {
			return nil, err
		}

		if generator == nil {
			return nil, &errs.ConfigError{Provider: providerID, Cause: errs.ErrProvConfigNilInterface}
		}

		if value := reflect.ValueOf(generator); value.Kind() == reflect.Pointer && value.IsNil() {
			return nil, &errs.ConfigError{Provider: providerID, Cause: errs.ErrProvConfigConstructorFailed}
		}

		return generator, nil
	}

	return nil, &errs.ConfigError{Provider: providerID, Cause: errs.ErrProvConfigNoConstructor}
}
