package output

import (
	"fmt"
	"io"
	"os"

	"github.com/shdeen/bildomat/internal/errs"
)

// WriteText writes formatted text to the supplied destination. Incomplete writes
// retain their original cause and output-write classification, including short writes.
func WriteText(destination io.Writer, format string, values ...any) error {
	content := fmt.Sprintf(format, values...)

	written, err := io.WriteString(destination, content)
	if err == nil && written != len(content) {
		err = io.ErrShortWrite
	}

	if err == nil {
		return nil
	}

	destinationName := ""
	if file, named := destination.(*os.File); named {
		destinationName = file.Name()
	}

	return errs.FileError(errs.FileOpWrite, destinationName, errs.ErrOutputFileWrite, err)
}
