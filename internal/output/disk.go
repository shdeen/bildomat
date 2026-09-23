package output

import "io"

// PrintSavedFile writes one completed file's path and size to destination. Styled output uses a
// human-readable size; plain output reports the byte count.
func PrintSavedFile(destination io.Writer, savedFile SavedFile, styled bool) error {
	if styled {
		return WriteText(destination, SavedReportStyled+"\n", ansiClay, savedFile.Path, ansiReset, ansiDim, fileSizeText(savedFile.Bytes))
	}

	return WriteText(destination, SavedReport+"\n", savedFile.Path, savedFile.Bytes)
}
