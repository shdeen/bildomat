package main

import (
	"strings"

	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/provider/config"
	"github.com/shdeen/bildomat/internal/provider/google"
	"github.com/urfave/cli/v3"
)

// Help template callbacks for the command, option, and tips sections.
const (
	pageFuncCommands = "commandsText"
	pageFuncOptions  = "optionsText"
	pageFuncTips     = "tipsText"
)

// helpExampleProviderIDs supplies the provider rotation for general help examples.
//
//nolint:gochecknoglobals // Immutable provider identifiers used by help rendering.
var helpExampleProviderIDs = []string{config.IDOpenAI, google.ProviderID, config.IDXAI}

// pageFuncs supplies the template callbacks for this application's help pages.
func pageFuncs(bild *bildApp) map[string]any {
	return map[string]any{
		output.PageFuncIndent: output.IndentText,
		output.PageFuncWrap:   output.WrapDetails,
		pageFuncCommands:      commandsHelpText,
		pageFuncOptions:       bild.optionsHelpText,
		pageFuncTips:          bild.tipsHelpText,
	}
}

// commandsHelpText formats visible subcommands with aligned names and usage summaries.
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
