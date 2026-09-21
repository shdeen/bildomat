// Package config loads the user's optional configuration file at
// <home>/.bildomat/config.yml: a default model, a default output directory,
// and per-provider API keys, read once per run. A fault never stops the
// caller; Load returns the decoded settings together with the classified
// faults, each wrapping a sentinel under errs.ErrUserConfig, for the
// caller's warning path.
package config

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/shdeen/bildomat/internal/errs"
)

// dirName is the user-scoped directory under the home directory.
const dirName = ".bildomat"

// fileName is the config file's name inside the user-scoped directory.
const fileName = "config.yml"

// yamlTagKey is the struct tag key that carries each setting's document key.
const yamlTagKey = "yaml"

// Settings holds the user config file's three optional settings. An empty
// value means unset: the loader drops empty values, so a consumer never
// sees an empty-but-present setting.
//   - DefaultModel: the model used when --model is omitted; anything the
//     --model flag accepts
//   - DefaultOutputDir: the directory used when no output location is given;
//     anything the directory portion of --output-path accepts
//   - APIKeys: API keys by provider ID, which outrank the providers'
//     environment variables
type Settings struct {
	DefaultModel     string            `yaml:"default-model"`
	DefaultOutputDir string            `yaml:"output-dir"`
	APIKeys          map[string]string `yaml:"api-keys"`
}

// declaredKeys holds the schema's top-level keys.
//
//nolint:gochecknoglobals // computed once from the Settings tags at package load, never written again.
var declaredKeys = readSettingKeys()

// readSettingKeys returns the schema's top-level keys from the Settings
// yaml tags, so the schema is stated exactly once.
func readSettingKeys() []string {
	settingsType := reflect.TypeFor[Settings]()
	keys := make([]string, 0, settingsType.NumField())

	for settingsField := range settingsType.Fields() {
		settingKey, _, _ := strings.Cut(settingsField.Tag.Get(yamlTagKey), ",")
		if settingKey != "" && settingKey != "-" {
			keys = append(keys, settingKey)
		}
	}

	return keys
}

// Load resolves the config file's location under the home directory, reads
// and decodes the file, and returns the decoded settings, the file's
// resolved path (empty when the location cannot be resolved), and the
// classified faults. A missing file or a missing .bildomat directory is
// silent: zero settings and no faults.
func Load() (Settings, string, []error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Settings{}, "", []error{&errs.ConfigError{Path: filepath.Join(dirName, fileName), Cause: errors.Join(errs.ErrUserConfigLocate, err)}}
	}

	filePath := filepath.Join(home, dirName, fileName)

	// #nosec G304 -- the path is the fixed user-config location under the resolved home directory.
	content, err := os.ReadFile(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return Settings{}, filePath, nil
	}

	if err != nil {
		return Settings{}, filePath, []error{&errs.ConfigError{Path: filePath, Cause: errors.Join(errs.ErrUserConfigRead, err)}}
	}

	settings, faults := decode(filePath, content)

	return settings, filePath, faults
}

// UnknownProviderFault returns the classified fault for a non-empty
// api-keys entry naming a provider the catalog does not hold. The caller
// supplies the config file's path and the offending provider ID.
func UnknownProviderFault(filePath, providerID string) error {
	return &errs.ConfigError{Path: filePath, Provider: providerID, Cause: errs.ErrUserConfigUnknownProvider}
}

// decode decodes one config file's content into its settings, dropping
// empty values, and returns the classified faults: a decode failure, or
// one fault per unknown top-level key. The path labels the faults; it is
// not read.
func decode(filePath string, content []byte) (Settings, []error) {
	var settings Settings
	if err := yaml.Unmarshal(content, &settings); err != nil {
		return Settings{}, []error{decodeFault(filePath, err)}
	}

	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		return Settings{}, []error{decodeFault(filePath, err)}
	}

	var faults []error

	for _, documentKey := range slices.Sorted(maps.Keys(document)) {
		if !slices.Contains(declaredKeys, documentKey) {
			faults = append(faults, &errs.ConfigError{Path: filePath, Setting: documentKey, Cause: errs.ErrUserConfigUnknownSetting})
		}
	}

	for providerID, apiKey := range settings.APIKeys {
		if apiKey == "" {
			delete(settings.APIKeys, providerID)
		}
	}

	if len(settings.APIKeys) == 0 {
		settings.APIKeys = nil
	}

	return settings, faults
}

// decodeFault classifies one failed decode of the config file.
func decodeFault(filePath string, err error) error {
	return &errs.ConfigError{Path: filePath, Cause: errors.Join(errs.ErrUserConfigDecode, err)}
}
