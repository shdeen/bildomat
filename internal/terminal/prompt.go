package terminal

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/output"
)

// ConfirmOneWordPrompt asks whether to submit a one-word prompt when interaction is enabled. It
// reads one reply from input and treats n and no as cancellation, regardless of case. Prompt and
// input failures retain their original causes.
func ConfirmOneWordPrompt(input io.Reader, destination io.Writer, prompt string, interactive, styled bool) (canceled bool, err error) {
	if !interactive || len(strings.Fields(prompt)) != 1 {
		return false, nil
	}

	if err := output.PrintSingleWordPromptConfirmation(destination, prompt, styled); err != nil {
		return false, err
	}

	reply, err := readReply(input)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(reply, "n") || strings.EqualFold(reply, "no"), nil
}

// RepromptModel asks for a replacement model when interaction is enabled. It reports whether it
// asked and preserves prompt or input failures.
func RepromptModel(input io.Reader, destination io.Writer, interactive, styled bool) (reply string, asked bool, err error) {
	if !interactive {
		return "", false, nil
	}

	if err := output.PrintReprompt(destination, styled); err != nil {
		return "", true, err
	}

	reply, err = readReply(input)

	return reply, true, err
}

// readReply reads and trims one line without consuming bytes from a later reply. EOF returns the
// available reply; other read failures retain their cause.
func readReply(input io.Reader) (string, error) {
	var replyBytes []byte

	nextByte := make([]byte, 1)

	for {
		count, err := input.Read(nextByte)
		if count == 1 {
			if nextByte[0] == '\n' {
				break
			}

			replyBytes = append(replyBytes, nextByte[0])
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return "", fmt.Errorf("%w, %w", errs.ErrProcessReadInput, err)
		}
	}

	return strings.TrimSpace(string(replyBytes)), nil
}
