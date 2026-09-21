// File: cmd/bild/cli.go
// The bild command surface: the root command and its flags, the informational
// commands and their shared media filters, and the subcommand dispatch.

package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"

	"github.com/urfave/cli/v3"
)

// The command and help tokens the surface references by name.
//   - cmdNameBild: the root command
//   - cmdNameList: the listing command
//   - cmdNameInfo: the details command
//   - cmdNameSearch: the search command
//   - cmdNameHelp: the help command, the same word as the help flag
const (
	cmdNameBild   = "bild"
	cmdNameList   = "list"
	cmdNameInfo   = "info"
	cmdNameSearch = "search"
	cmdNameHelp   = HelpFlag
)

// createSharedFlags returns new instances of the three flags every
// flag-accepting command carries: the public switch that selects one JSON
// result document instead of regular command output, the hidden debug
// diagnostic switch, and the help flag. Each command places the first two
// among its ordinary flags, where its help page lists them, and the help flag
// in an alternative of its own. The library rejects a help flag typed twice.
//
// Note: the debug flag is to be officially "undocumented" and hidden from the help output.
// However, it is not "secret," and mention should not be avoided in doc comments.
func createSharedFlags() (jsonFlag, debugFlag, helpFlag cli.Flag) {
	jsonFlag = &cli.BoolFlag{
		Name:    RunFlagJSON,
		Aliases: []string{RunFlagJSONAlias},
		Local:   true,
		Usage:   JSONFlagHelp,
	}
	// Debug and its automatic record retention are experimental, provisional,
	// and subject to change. They remain outside public help and documentation.
	debugFlag = &cli.BoolFlag{Name: RunFlagDebug, Local: true, Hidden: true}
	helpFlag = &cli.BoolFlag{
		Name:     HelpFlag,
		Aliases:  []string{HelpFlagAlias},
		Local:    true,
		OnlyOnce: true,
		Usage:    HelpFlagHelp,
	}

	return jsonFlag, debugFlag, helpFlag
}

// createFlagGroup takes a command's ordinary flags and the flags that must
// each stand alone, and returns the command's one flag group: the ordinary
// flags as one alternative, then each stand-alone flag as an alternative of
// its own. The library lets flags of only one alternative be set, so it
// rejects a stand-alone flag beside any other flag. A help page lists the
// flags in this order.
func createFlagGroup(ordinaryFlags []cli.Flag, aloneFlags ...cli.Flag) []cli.MutuallyExclusiveFlags {
	alternatives := make([][]cli.Flag, 0, 1+len(aloneFlags))
	alternatives = append(alternatives, ordinaryFlags)

	for _, aloneFlag := range aloneFlags {
		alternatives = append(alternatives, []cli.Flag{aloneFlag})
	}

	return []cli.MutuallyExclusiveFlags{{Flags: alternatives}}
}

// createMediaFlags returns the info command's flag group: the shared flags
// and the image and video filters.
func createMediaFlags() []cli.MutuallyExclusiveFlags {
	jsonFlag, debugFlag, helpFlag := createSharedFlags()

	return createFlagGroup([]cli.Flag{
		jsonFlag,
		&cli.BoolFlag{
			Name:    FilterFlagImage,
			Aliases: []string{FilterFlagImageAlias},
			Usage:   InfoImageFilterHelp,
		},
		&cli.BoolFlag{
			Name:    FilterFlagVideo,
			Aliases: []string{FilterFlagVideoAlias},
			Usage:   InfoVideoFilterHelp,
		},
		debugFlag,
	}, helpFlag)
}

// createSearchFlags returns the search command's flag group: the shared
// flags, the listing controls, the regular-expression switch, and the
// exclusion flag, whose value is the exclusion term. The library has no flag
// with an optional value, so the exclusion flag always takes the next token.
func createSearchFlags() []cli.MutuallyExclusiveFlags {
	jsonFlag, debugFlag, helpFlag := createSharedFlags()

	return createFlagGroup([]cli.Flag{
		createAliasesFlag(),
		jsonFlag,
		&cli.BoolFlag{
			Name:    FilterFlagProviders,
			Aliases: []string{FilterFlagProvidersAlias},
			Usage:   SearchProvidersFlagHelp,
		},
		&cli.BoolFlag{
			Name:    FilterFlagModels,
			Aliases: []string{FilterFlagModelsAlias},
			Usage:   SearchModelsFlagHelp,
		},
		&cli.BoolFlag{
			Name:    FilterFlagImage,
			Aliases: []string{FilterFlagImageAlias},
			Usage:   SearchImageFilterHelp,
		},
		&cli.BoolFlag{
			Name:    FilterFlagVideo,
			Aliases: []string{FilterFlagVideoAlias},
			Usage:   SearchVideoFilterHelp,
		},
		&cli.BoolFlag{
			Name:    FilterFlagRegex,
			Aliases: []string{FilterFlagRegexAlias},
			Usage:   SearchRegexFlagHelp,
		},
		&cli.StringFlag{
			Name:        FilterFlagExclude,
			Aliases:     []string{FilterFlagExcludeAlias},
			Usage:       SearchExcludeFlagHelp,
			HideDefault: true,
		},
		debugFlag,
	}, helpFlag)
}

// createListingFlags returns the list command's flag group: the shared flags,
// the alias display switch, and the provider, model, and media filters.
func createListingFlags() []cli.MutuallyExclusiveFlags {
	jsonFlag, debugFlag, helpFlag := createSharedFlags()

	return createFlagGroup([]cli.Flag{
		createAliasesFlag(),
		jsonFlag,
		&cli.BoolFlag{
			Name:    FilterFlagProviders,
			Aliases: []string{FilterFlagProvidersAlias},
			Usage:   ListProvidersFlagHelp,
		},
		&cli.BoolFlag{
			Name:    FilterFlagModels,
			Aliases: []string{FilterFlagModelsAlias},
			Usage:   ListModelsFlagHelp,
		},
		&cli.BoolFlag{
			Name:    FilterFlagImage,
			Aliases: []string{FilterFlagImageAlias},
			Usage:   ListImageFilterHelp,
		},
		&cli.BoolFlag{
			Name:    FilterFlagVideo,
			Aliases: []string{FilterFlagVideoAlias},
			Usage:   ListVideoFilterHelp,
		},
		debugFlag,
	}, helpFlag)
}

// createAliasesFlag returns the switch that includes model aliases in listings.
func createAliasesFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:    ListingFlagAliases,
		Aliases: []string{ListingFlagAliasesAlias},
		Usage:   AliasesFlagHelp,
	}
}

// selectedMedia takes the list command and reports which media its filters
// select. The filters are inclusive: neither media switch, and both together,
// each select both media; one alone selects that medium.
func selectedMedia(c *cli.Command) (imageSelected, videoSelected bool) {
	return inclusivePair(c.Bool(FilterFlagImage), c.Bool(FilterFlagVideo))
}

// inclusivePair takes the two switches of one inclusive filter pair and
// returns the selection they express: both when neither or both are set, and
// otherwise the one that is.
func inclusivePair(firstSet, secondSet bool) (first, second bool) {
	if firstSet == secondSet {
		return true, true
	}

	return firstSet, secondSet
}

// createFlags takes the parameter flag records and returns the root command's
// flag group, ordered as the help page prints it: first the run's own model
// and output flags with the shared flags, then one flag for each enumerated
// parameter, sorted by long name, then the help flag and the version flag,
// each of which must stand alone. The library rejects a version flag typed
// twice. Every root flag is local: a root flag is set only when typed ahead
// of a command word, which the flags-before-command rule depends on.
func createFlags(paramFlags []params.Flag) []cli.MutuallyExclusiveFlags {
	sortedFlags := slices.Clone(paramFlags)
	slices.SortFunc(sortedFlags, func(a, b params.Flag) int {
		return strings.Compare(string(a.FlagID), string(b.FlagID))
	})

	jsonFlag, debugFlag, helpFlag := createSharedFlags()

	flags := slices.Grow([]cli.Flag{
		&cli.StringFlag{
			Name:    RunFlagModel,
			Aliases: []string{RunFlagModelAlias},
			Local:   true,
			Usage:   ModelFlagHelp,
		},
		&cli.StringFlag{
			Name:    RunFlagOutputPath,
			Aliases: []string{RunFlagOutputPathAlias},
			Local:   true,
			Usage:   OutputPathFlagHelp,
		},
		&cli.BoolFlag{
			Name:    RunFlagPrintFilename,
			Aliases: []string{RunFlagPrintFilenameAlias},
			Local:   true,
			Usage:   PrintFilenameFlagHelp,
		},
		&cli.StringFlag{
			Name:  RunFlagSaveResults,
			Local: true,
			Usage: SaveResultsFlagHelp,
		},
		jsonFlag,
		debugFlag,
		&cli.BoolFlag{Name: persistRecordFlag, Local: true, Hidden: true},
		&cli.StringFlag{Name: reuseFlag, Local: true, Hidden: true, OnlyOnce: true},
	}, len(sortedFlags))

	for i := range sortedFlags {
		flags = append(flags, paramCLIFlag(&sortedFlags[i]))
	}

	versionFlag := &cli.BoolFlag{
		Name:     VersionFlag,
		Aliases:  []string{VersionFlagAlias},
		Local:    true,
		OnlyOnce: true,
		Usage:    VersionFlagHelp,
	}

	return createFlagGroup(flags, helpFlag, versionFlag)
}

// paramCLIFlag takes one parameter flag and returns the corresponding CLI flag.
func paramCLIFlag(paramFlag *params.Flag) cli.Flag {
	flagID := string(paramFlag.FlagID)

	usage := paramFlagUsage(paramFlag)
	if paramFlag.AllowMultiple {
		return &cli.StringSliceFlag{
			Name:    flagID,
			Aliases: paramFlag.Aliases,
			Usage:   usage,
			Local:   true, HideDefault: true,
		}
	}

	switch paramFlag.DataType {
	case params.DataString:
	case params.DataNumber:
		return &cli.FloatFlag{
			Name:        flagID,
			Aliases:     paramFlag.Aliases,
			Usage:       usage,
			Local:       true,
			HideDefault: true,
		}
	case params.DataInteger:
		return &cli.IntFlag{
			Name:        flagID,
			Aliases:     paramFlag.Aliases,
			Usage:       usage,
			Local:       true,
			HideDefault: true,
		}
	case params.DataBoolean:
		return &cli.BoolFlag{
			Name:        flagID,
			Aliases:     paramFlag.Aliases,
			Usage:       usage,
			Local:       true,
			HideDefault: true,
		}
	}

	return &cli.StringFlag{
		Name:        flagID,
		Aliases:     paramFlag.Aliases,
		Usage:       usage,
		Local:       true,
		HideDefault: true,
	}
}

// paramFlagUsage takes a parameter flag and returns its help text, which
// combines the flag's description, comment, and example values. The
// provider-support note needs the catalog, so the help page renderer adds it.
func paramFlagUsage(paramFlag *params.Flag) string {
	usage := paramFlag.Description
	if paramFlag.Comment != "" {
		usage += " " + paramFlag.Comment
	}

	if len(paramFlag.ExampleValues) > 0 {
		usage += " " + fmt.Sprintf(output.ExamplesSentence, strings.Join(paramFlag.ExampleValues, ", "))
	}

	return usage
}

// newSubcommand takes the application, a subcommand's name, usage summary,
// usage form, and flag group, and returns a cli.Command carrying them
// together with the shared usage-error handler and the hook that loads the
// application. The library's own help handling is off; the command's help
// flag is in its flag group. The caller assigns the command's action.
func newSubcommand(bild *bildApp, name, summary, usageForm string, flagGroup []cli.MutuallyExclusiveFlags) *cli.Command {
	return &cli.Command{
		Name:                   name,
		Usage:                  summary,
		UsageText:              usageForm,
		HideHelp:               true,
		MutuallyExclusiveFlags: flagGroup,
		OnUsageError:           bild.cliClassifyUsageError,
		Before:                 bild.cliPrepareCommand,
	}
}

// createCommand takes the application and returns the bild command surface:
// the root command with its flags, the list, info, search, and help
// subcommands, and their page-rendering actions. It reads nothing but the
// generated parameter flag records, so it cannot fail. The library's built-in
// help and version handling is off: help and version are ordinary flags in
// each command's flag group, and the hooks and actions serve them. The
// surface declares its own help command rather than leaving the library to
// append one, because the library's carries an h alias that this surface does
// not offer. The help command accepts the debug switch and no other flag.
func createCommand(bild *bildApp) *cli.Command {
	// The library must not intercept our ordinary, mutually exclusive help flag.
	cli.HelpFlag = nil

	listCmd := newSubcommand(bild, cmdNameList, ListSummary, ListUsageForm, createListingFlags())
	listCmd.Action = bild.cliRunList

	infoCmd := newSubcommand(bild, cmdNameInfo, InfoSummary, InfoUsageForm, createMediaFlags())
	infoCmd.Action = bild.cliRunInfo

	searchCmd := newSubcommand(bild, cmdNameSearch, SearchSummary, SearchUsageForm, createSearchFlags())
	searchCmd.Action = bild.cliRunSearch

	_, helpDebugFlag, _ := createSharedFlags()
	helpCmd := &cli.Command{
		Name:         cmdNameHelp,
		Usage:        HelpSummary,
		UsageText:    HelpUsageForm,
		HideHelp:     true,
		Flags:        []cli.Flag{helpDebugFlag},
		OnUsageError: bild.cliClassifyUsageError,
		Before:       bild.cliPrepareCommand,
		Action:       bild.cliRunHelp,
	}

	return &cli.Command{
		Name:                      cmdNameBild,
		Version:                   appVersion(),
		Usage:                     "",
		UsageText:                 RootUsageForms,
		DisableSliceFlagSeparator: true,
		HideHelp:                  true,
		HideVersion:               true,
		MutuallyExclusiveFlags:    createFlags(params.Flags()),
		OnUsageError:              bild.cliClassifyUsageError,
		Before:                    bild.cliPrepareRoot,
		Action:                    bild.cliRunGenerate,
		Commands:                  []*cli.Command{listCmd, infoCmd, searchCmd, helpCmd},
	}
}
