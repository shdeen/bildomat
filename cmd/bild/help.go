package main

import (
	"strings"

	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/provider/config"
	"github.com/shdeen/bildomat/internal/provider/google"
	"github.com/urfave/cli/v3"
)

// Help template function names.
const (
	pageFuncCommands = "commandsText"
	pageFuncOptions  = "optionsText"
	pageFuncTips     = "tipsText"
)

// helpExampleProviderIDs supplies the provider rotation for general help examples.
//
//nolint:gochecknoglobals // Immutable provider identifiers used by help rendering.
var helpExampleProviderIDs = []string{config.IDOpenAI, google.ProviderID, config.IDXAI}

// pageFuncs takes the application and returns the functions the help page
// templates call by name: layout, command and flag descriptions, and examples.
func pageFuncs(bild *bildApp) map[string]any {
	return map[string]any{
		output.PageFuncIndent: output.IndentText,
		output.PageFuncWrap:   output.WrapDetails,
		pageFuncCommands:      commandsHelpText,
		pageFuncOptions:       bild.optionsHelpText,
		pageFuncTips:          bild.tipsHelpText,
	}
}

// commandsHelpText takes a command and returns the command section of its help
// page: one indented entry per subcommand that is not hidden, giving the
// subcommand's names padded to a common width, then its usage summary.
func commandsHelpText(cmd *cli.Command) string {
	var shown []*cli.Command

	for _, subCmd := range cmd.Commands {
		if !subCmd.Hidden {
			shown = append(shown, subCmd)
		}
	}

	padWidth := 0
	for _, subCmd := range shown {
		if width := len(strings.Join(subCmd.Names(), ", ")); width > padWidth {
			padWidth = width
		}
	}

	usages := make([]string, 0, len(shown))

	for _, subCmd := range shown {
		names := strings.Join(subCmd.Names(), ", ")
		gapChars := strings.Repeat(" ", padWidth-len(names)+2)
		usages = append(usages, output.IndentText(1, names+gapChars+subCmd.Usage))
	}

	return strings.Join(usages, "\n")
}
