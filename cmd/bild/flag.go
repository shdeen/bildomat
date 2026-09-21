package main

import (
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"
)

// Run-surface, filter, and help flag names: the command flags declared
// outside the parameter enumeration, each with the one-letter alias it
// carries. A flag's dashed spelling is never stored; the site that needs one
// prefixes the name with DoubleDash and the alias with Dash.
//   - RunFlagModel: the root command's model specifier
//   - RunFlagOutputPath: the root command's output path
//   - RunFlagPrintFilename: the switch that prints only the saved file paths
//   - RunFlagSaveResults: the file that receives the regular results output
//   - RunFlagJSON: the switch selecting one JSON result document
//   - RunFlagDebug: the hidden flag rendering the internal error chain
//   - FilterFlagImage: the image filter of the list, info, and search commands
//   - FilterFlagVideo: the video filter of the list, info, and search commands
//   - FilterFlagProviders: the providers filter of the list and search commands
//   - FilterFlagModels: the models filter of the list and search commands
//   - FilterFlagRegex: the search command's regular-expression switch
//   - FilterFlagExclude: the search command's flag whose value is the exclusion term
//   - ListingFlagAliases: whether list and search display model aliases
//   - HelpFlag: the help flag and the help command, one word for both
//   - VersionFlag: the root command's version flag
//   - RunFlagModelAlias: the model specifier's alias
//   - RunFlagOutputPathAlias: the output path's alias
//   - RunFlagPrintFilenameAlias: the print-filename switch's alias
//   - RunFlagJSONAlias: the JSON switch's alias
//   - FilterFlagImageAlias: the image filter's alias
//   - FilterFlagVideoAlias: the video filter's alias
//   - FilterFlagProvidersAlias: the providers filter's alias
//   - FilterFlagModelsAlias: the models filter's alias
//   - FilterFlagRegexAlias: the regular-expression switch's alias
//   - FilterFlagExcludeAlias: the exclusion flag's alias
//   - ListingFlagAliasesAlias: the model-alias display switch's alias
//   - HelpFlagAlias: the help flag's alias
//   - VersionFlagAlias: the version flag's alias
const (
	RunFlagModel         = "model"
	RunFlagOutputPath    = "output-path"
	RunFlagPrintFilename = "print-filename"
	RunFlagSaveResults   = "save-results"
	RunFlagJSON          = "json"
	RunFlagDebug         = "debug"

	FilterFlagImage     = "image"
	FilterFlagVideo     = "video"
	FilterFlagProviders = "providers"
	FilterFlagModels    = "models"
	FilterFlagRegex     = "regex"
	FilterFlagExclude   = "exclude"
	ListingFlagAliases  = "aliases"

	HelpFlag    = "help"
	VersionFlag = "version"

	RunFlagModelAlias         = "m"
	RunFlagOutputPathAlias    = "o"
	RunFlagPrintFilenameAlias = "p"
	RunFlagJSONAlias          = "j"
	FilterFlagImageAlias      = "i"
	FilterFlagVideoAlias      = "v"
	FilterFlagProvidersAlias  = "p"
	FilterFlagModelsAlias     = "m"
	FilterFlagRegexAlias      = "r"
	FilterFlagExcludeAlias    = "x"
	ListingFlagAliasesAlias   = "a"
	HelpFlagAlias             = "h"
	VersionFlagAlias          = "v"
)

// Presentation inputs for flags owned by the command.
//
//nolint:gochecknoglobals // These flag descriptions are immutable after package initialization.
var (
	runFlagNames         = map[params.FlagType]string{RunFlagOutputPath: output.OutPathDisplayName}
	runFlagHints         = map[string]string{RunFlagModel: "model", RunFlagOutputPath: "path", RunFlagSaveResults: "path", FilterFlagExclude: "term"} //nolint:goconst // The value hint and flag identifier have independent meanings despite equal spelling.
	mediaFilterFlagNames = map[media.Kind]string{media.Image: FilterFlagImage, media.Video: FilterFlagVideo}
)
