package artifact

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// genStemPrefix opens the filename stem used when no stem is requested; the
// medium of the run completes it, as in bild-image and bild-video.
const genStemPrefix = "bild"

// SidecarExt is the Markdown extension shared by sidecar persistence and collision checks.
const SidecarExt = ".md"

// firstSuffixNumber is the first two-digit suffix a taken name receives.
const firstSuffixNumber = 2

// DefaultStem takes the medium of the run and returns the filename stem used
// when no stem is requested.
func DefaultStem(mediaKind media.Kind) string {
	return genStemPrefix + "-" + string(mediaKind)
}

// WriteMedia exclusively saves a generation under an available filename stem.
// It returns every successfully closed destination, even when later writing or
// source cleanup fails. Destination failures stop further writes. Cleanup runs
// after writing, so a failed source removal cannot prevent another artifact.
func WriteMedia(dir, requestedStem string, mediaKind media.Kind, artifacts []Media, writesSidecar bool) (finalStem string, completedFiles []SavedFile, writeErr error) {
	if len(artifacts) == 0 {
		return "", nil, errs.ErrOutputFileEmpty
	}

	stem, firstFile, err := resolveStem(dir, requestedStem, mediaKind, artifacts, writesSidecar)
	if err != nil {
		return "", nil, errors.Join(err, Cleanup(artifacts))
	}

	savedFiles := make([]SavedFile, 0, len(artifacts))
	for artifactIndex, generated := range artifacts {
		var savedFile SavedFile
		if artifactIndex == 0 {
			savedFile, err = writeToReservedFile(firstFile, artifactPath(dir, stem, artifacts, artifactIndex), generated)
		} else {
			savedFile, err = writeArtifact(dir, artifactName(stem, artifacts, artifactIndex), generated)
		}

		writeErr = errors.Join(writeErr, err)

		if savedFile.Path == "" {
			break
		}

		savedFiles = append(savedFiles, savedFile)
	}

	return stem, savedFiles, errors.Join(writeErr, Cleanup(artifacts))
}

// writeToReservedFile writes generated media to an exclusively claimed destination.
// It returns a fact only after that destination has closed successfully.
func writeToReservedFile(file *os.File, dstPath string, generated Media) (SavedFile, error) {
	if generated.TmpPath == "" {
		return commitFile(file, dstPath, generated.Data)
	}

	return copyFile(file, dstPath, generated.TmpPath)
}

// writeArtifact claims the first free destination for generated media and writes
// it, returning the completed-file fact and any error.
func writeArtifact(dir, name string, generated Media) (SavedFile, error) {
	file, path, err := claimFile(dir, name, generated.FileExt)
	if err != nil {
		return SavedFile{}, err
	}

	return writeToReservedFile(file, path, generated)
}

// suffixedName takes a filename stem and a suffix number and returns the stem with the
// two-digit suffix appended.
func suffixedName(name string, suffixNumber int) string {
	return fmt.Sprintf("%s-%02d", name, suffixNumber)
}

// artifactName takes a stem, artifact collection, and index and returns that artifact's
// filename without its extension: the bare stem for a single artifact, and the stem with
// the artifact's index for a batch.
func artifactName(stem string, artifacts []Media, i int) string {
	if len(artifacts) > 1 {
		return fmt.Sprintf("%s-%d", stem, i)
	}

	return stem
}

// artifactPath takes a directory, stem, artifact collection, and index and returns the
// destination path for that artifact.
func artifactPath(dir, stem string, artifacts []Media, i int) string {
	return filepath.Join(dir, artifactName(stem, artifacts, i)+artifacts[i].FileExt)
}

// resolveStem takes a directory, requested stem, the medium of the run, artifacts, and
// whether the run will write a sidecar, and returns an available stem with an open file for
// the first artifact: the requested stem, or the medium's default stem when none was
// requested, taken bare when free and otherwise under the first free suffix. It creates that
// file exclusively and returns an error when the path cannot be created for a reason other
// than a name conflict.
func resolveStem(dir, requestedStem string, mediaKind media.Kind, artifacts []Media, writesSidecar bool) (stem string, file *os.File, err error) {
	if requestedStem == "" {
		requestedStem = DefaultStem(mediaKind)
	}

	for {
		stem = uniqueStem(dir, requestedStem, artifacts, writesSidecar)

		claimPath := artifactPath(dir, stem, artifacts, 0)

		file, err = openFile(claimPath)
		if err == nil {
			return stem, file, nil
		}

		if !errors.Is(err, fs.ErrExist) {
			return stem, nil, errs.FileError(errs.FileOpCreate, claimPath, errs.ErrOutputFileCreate, err)
		}
	}
}

// uniqueStem takes a directory, requested stem, and files the run will write. It returns
// the requested stem when every destination is available or the first available
// two-digit variant beginning at 02.
func uniqueStem(dir, stem string, artifacts []Media, writesSidecar bool) string {
	if !hasDestinationClash(dir, stem, artifacts, writesSidecar) {
		return stem
	}

	for i := firstSuffixNumber; ; i++ {
		candidateStem := suffixedName(stem, i)
		if !hasDestinationClash(dir, candidateStem, artifacts, writesSidecar) {
			return candidateStem
		}
	}
}

// hasDestinationClash reports whether any exact path the run will write already exists.
func hasDestinationClash(dir, stem string, artifacts []Media, writesSidecar bool) bool {
	for artifactIndex := range artifacts {
		if fileExists(artifactPath(dir, stem, artifacts, artifactIndex)) {
			return true
		}
	}

	return writesSidecar && fileExists(filepath.Join(dir, stem+SidecarExt))
}

// fileExists reports whether a file, directory, or symbolic link occupies a path.
func fileExists(path string) bool {
	_, err := os.Lstat(path)

	return err == nil
}
