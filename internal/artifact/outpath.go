package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// Location contains the parsed components of an output path.
//   - Dir: the directory portion of the path
//   - Stem: the filename without its extension
//   - Ext: the supported media extension as supplied
//   - Format: the image format represented by Ext
type Location struct {
	Dir    string
	Stem   string
	Ext    string
	Format string
}

// ParseOutPath returns the directory, filename stem, extension, and image format represented by an output path.
func ParseOutPath(outPath string) Location {
	if outPath == "" {
		return Location{}
	}

	if os.IsPathSeparator(outPath[len(outPath)-1]) {
		return Location{Dir: outPath}
	}

	base := filepath.Base(outPath)
	if base == "." || base == ".." || base == "~" {
		return Location{Dir: outPath}
	}

	dir := filepath.Dir(outPath)
	if dir == "." && !strings.HasPrefix(outPath, "./") {
		dir = ""
	}

	ext := filepath.Ext(base)
	if ext == "" && pathIsExistingDirectory(outPath) {
		return Location{Dir: outPath}
	}

	stem := strings.TrimSuffix(base, ext)

	format, supported := outPathFormat(ext)
	if !supported {
		return Location{Dir: dir, Stem: stem}
	}

	return Location{Dir: dir, Stem: stem, Ext: ext, Format: format}
}

// pathIsExistingDirectory checks a path using the shared home expansion rules.
func pathIsExistingDirectory(path string) bool {
	statPath, err := ExpandHome(path)
	if err != nil {
		return false
	}

	fileInfo, err := os.Stat(statPath)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

// outPathFormat returns the image format represented by an extension and whether the extension is supported.
func outPathFormat(ext string) (string, bool) {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case media.FormatPNG:
		return media.FormatPNG, true
	case media.FormatJPG, media.FormatJPEG:
		return media.FormatJPEG, true
	case media.FormatWebP:
		return media.FormatWebP, true
	case media.FormatMP4:
		return "", true
	}

	return "", false
}

// FormatExt returns the canonical file extension for format, or an empty string when format is unsupported.
func FormatExt(format string) string {
	switch strings.ToLower(format) {
	case media.FormatPNG:
		return "." + media.FormatPNG
	case media.FormatJPG, media.FormatJPEG:
		return "." + media.FormatJPG
	case media.FormatWebP:
		return "." + media.FormatWebP
	}

	return ""
}

// ExpandHome takes a path and returns it with a leading tilde expanded to the user's home
// directory. It returns the path unchanged when no expansion is needed and an error when
// the home directory is unavailable.
func ExpandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errs.FileError(errs.FileOpExpand, path, errs.ErrOutputFileHome, err)
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}

// resolveOutputDir takes an output-directory path, expands a leading tilde, resolves a
// relative path against the working directory, and returns the resulting absolute path. It
// creates the directory and missing parents and returns an error if expansion, resolution,
// or creation fails.
func resolveOutputDir(userInputDirPath string) (string, error) {
	expandedDir, err := ExpandHome(userInputDirPath)
	if err != nil {
		return "", err
	}

	outDir, err := filepath.Abs(expandedDir)
	if err != nil {
		return "", fmt.Errorf("%q: %w, %w", expandedDir, errs.ErrProcessWorkingDir, err)
	}

	if err := CreateDir(outDir); err != nil {
		return "", err
	}

	return outDir, nil
}

// CreateDir creates an already resolved output directory and its missing parents.
// It may be called again after a long generation without resolving the path again.
func CreateDir(path string) error {
	// #nosec G301 -- generated output directories follow the invoking user's umask.
	if err := os.MkdirAll(path, 0o755); err != nil {
		return errs.FileError(errs.FileOpMkdir, path, errs.ErrOutputFileMkdir, err)
	}

	return nil
}

// Resolve makes the location's directory absolute, creates it, and sanitizes
// the requested stem. An empty stem remains empty for command format precedence.
func (location *Location) Resolve() error {
	directory, err := resolveOutputDir(location.Dir)
	if err != nil {
		return err
	}

	location.Dir = directory
	location.Stem = sanitizeFilename(location.Stem)

	return nil
}

// sanitizeFilename takes filename text and returns it with slashes and control characters replaced by hyphens.
func sanitizeFilename(text string) string {
	var builder strings.Builder

	for _, runeValue := range text {
		if runeValue == '/' || runeValue == '\\' || runeValue < 0x20 || runeValue == 0x7f {
			builder.WriteRune('-')

			continue
		}

		builder.WriteRune(runeValue)
	}

	return builder.String()
}
