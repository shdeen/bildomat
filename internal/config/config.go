// Package config loads optional settings from <home>/.bildomat/config.yml. It returns settings and
// classified faults so the caller can warn about configuration errors without stopping the run.
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

// Settings holds optional defaults and API keys. Empty values mean unset; Load removes empty API
// keys.
//   - DefaultModel: the model used when --model is omitted; anything the --model flag accepts
//   - DefaultOutputDir: the directory used when no output location is given; anything the directory
//     portion of --output-path accepts
//   - APIKeys: API keys by provider ID, which outrank the providers' environment variables
type Settings struct {
	DefaultModel     string            `yaml:"default-model"`
	DefaultOutputDir string            `yaml:"output-dir"`
	APIKeys          map[string]string `yaml:"api-keys"`
}

// declaredKeys holds the schema's top-level keys.
//
//nolint:gochecknoglobals // computed once from the Settings tags at package load, never written again.
var declaredKeys = readSettingKeys()

// readSettingKeys returns the top-level configuration keys declared by Settings.
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

// Load reads the optional configuration file and returns settings, its path, and classified faults.
// A missing file returns empty settings without a fault; an unresolved home returns an empty path.
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

// UnknownProviderFault describes an api-keys provider absent from the catalog.
func UnknownProviderFault(filePath, providerID string) error {
	return &errs.ConfigError{Path: filePath, Provider: providerID, Cause: errs.ErrUserConfigUnknownProvider}
}

// decode parses configuration bytes, drops empty API keys, and reports malformed YAML or unknown
// top-level settings. The supplied path labels faults without being read.
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
