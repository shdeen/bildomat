package output

import "io"

// PrintSingleWordPromptConfirmation renders the confirmation using the selected diagnostic styling
// and returns any failure to deliver it.
func PrintSingleWordPromptConfirmation(destination io.Writer, prompt string, styled bool) error {
	steel, clay, reset := promptStyleValues(styled)

	return WriteText(destination, SingleWordConfirmation+" ", steel, clay, reset, prompt)
}

// PrintReprompt renders the replacement-model question with the selected styling.
func PrintReprompt(destination io.Writer, styled bool) error {
	steel, _, reset := promptStyleValues(styled)

	return WriteText(destination, Reprompt+" ", steel, reset)
}
