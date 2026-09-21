package output

import "io"

// PrintSavedFile writes one completed file's path and size to destination.
// Styling is selected by the command, independently of the file's facts.
func PrintSavedFile(destination io.Writer, savedFile SavedFile, styled bool) error {
	if styled {
		return WriteText(destination, SavedReportStyled+"\n", ansiClay, savedFile.Path, ansiReset, ansiDim, FileSizeText(savedFile.Bytes))
	}

	return WriteText(destination, SavedReport+"\n", savedFile.Path, savedFile.Bytes)
}
