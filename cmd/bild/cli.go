package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"

	"github.com/urfave/cli/v3"
)

// Command names used when constructing the CLI and its help pages.
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

// createSharedFlags returns fresh JSON, hidden debug, and standalone help flags.
func createSharedFlags() (jsonFlag, debugFlag, helpFlag cli.Flag) {
	jsonFlag = &cli.BoolFlag{
		Name:    RunFlagJSON,
		Aliases: []string{RunFlagJSONAlias},
		Local:   true,
		Usage:   JSONFlagHelp,
	}
	// Debug and its automatic record retention are experimental, provisional, and subject to
	// change. They remain outside public help and documentation.
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

// createFlagGroup allows ordinaryFlags together and makes each aloneFlags member exclusive of every
// other flag. The returned group also determines the flags' order in help.
func createFlagGroup(ordinaryFlags []cli.Flag, aloneFlags ...cli.Flag) []cli.MutuallyExclusiveFlags {
	alternatives := make([][]cli.Flag, 0, 1+len(aloneFlags))
	alternatives = append(alternatives, ordinaryFlags)

	for _, aloneFlag := range aloneFlags {
		alternatives = append(alternatives, []cli.Flag{aloneFlag})
	}

	return []cli.MutuallyExclusiveFlags{{Flags: alternatives}}
}

// createMediaFlags returns the info command's flag group: the shared flags and the image and video
// filters.
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

// createSearchFlags adds search controls to the shared presentation flags. The exclusion flag
// always consumes a value, including when that value resembles a flag.
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

// createListingFlags returns the list command's flag group: the shared flags, the alias display
// switch, and the provider, model, and media filters.
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

// inclusivePair selects both alternatives when neither or both switches are set.
func inclusivePair(firstSet, secondSet bool) (first, second bool) {
	if firstSet == secondSet {
		return true, true
	}

	return firstSet, secondSet
}

// createFlags builds root flags in help-page order, sorting parameter flags by name. Root flags are
// local so their position relative to a subcommand remains observable.
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

// paramCLIFlag converts a parameter definition into a typed, local CLI flag.
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

// paramFlagUsage combines a parameter description, guidance, and examples. The help renderer adds
// provider support after the catalog loads.
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

// newSubcommand attaches shared preparation and usage-error handling. The caller supplies its
// action; bild handles help instead of the CLI library.
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

// createCommand builds the CLI without loading configuration or providers. It disables the
// library's global help flag so bild can enforce its own help and version rules.
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
