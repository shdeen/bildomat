package output

// File: internal/output/prompt.go
// The two prompts that read a reply from standard input: the confirmation of
// a one-word generation prompt, and the request for a replacement model input
// after an ambiguous one. Each asks only when standard input is a terminal.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
)

// cancelReplyExpression matches the trimmed replies that cancel the one-word
// confirmation: n or N, alone or followed by a lowercase o.
const cancelReplyExpression = `^[nN]o?$`

// cancelReplyPattern is cancelReplyExpression compiled once at startup.
var cancelReplyPattern = regexp.MustCompile(cancelReplyExpression)

// ConfirmOneWordPrompt asks whether to submit a one-word prompt when interaction
// is enabled. It returns cancellation or the classified prompt/read failure.
func ConfirmOneWordPrompt(destination io.Writer, prompt string, interactive, styled bool) (canceled bool, err error) {
	if !interactive || len(strings.Fields(prompt)) != 1 {
		return false, nil
	}

	steel, clay, reset := promptStyleValues(styled)
	if err := WriteText(destination, SingleWordConfirmation+" ", steel, clay, reset, prompt); err != nil {
		return false, err
	}

	reply, err := readReply()
	if err != nil {
		return false, err
	}

	return cancelReplyPattern.MatchString(reply), nil
}

// RepromptModel asks for a replacement model when interaction is enabled.
// It reports whether it asked and preserves prompt or input failures.
func RepromptModel(destination io.Writer, interactive, styled bool) (reply string, asked bool, err error) {
	if !interactive {
		return "", false, nil
	}

	steel, _, reset := promptStyleValues(styled)
	if err := WriteText(destination, Reprompt+" ", steel, reset); err != nil {
		return "", true, err
	}

	reply, err = readReply()

	return reply, true, err
}

// readReply reads one line from standard input, one byte at a time so that
// nothing past the line is consumed, and returns it trimmed. A stream that
// closes before the newline returns what was read. A failed read returns the
// process read error.
func readReply() (string, error) {
	var line []byte

	next := make([]byte, 1)

	for {
		n, err := os.Stdin.Read(next)
		if n == 1 {
			if next[0] == '\n' {
				break
			}

			line = append(line, next[0])
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return "", fmt.Errorf("%w, %w", errs.ErrProcessReadInput, err)
		}
	}

	return strings.TrimSpace(string(line)), nil
}
