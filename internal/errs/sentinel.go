// Package errs defines the sentinels for bildomat's internal error contract. The
// error model is a hierarchy of categories and more specific failures. Category roots
// are the stable errors.Is targets that internal/output dispatches user
// messages on. Each root is a bare subject noun ("transport", "input media", …).
// Beneath each root sit PRECISE operation/condition sentinels, defined as
// fmt.Errorf("%w: <predicate>", <root>) so a sentinel reads as "subject: predicate"
// (e.g. "transport: request failed") and errors.Is matches BOTH the precise
// sentinel and its root. Every error site wraps the PRECISE sentinel and adds only
// the nearest granular detail (endpoint, path, id, model) as context, so the
// operation/condition lives in the sentinel rather than in repeated context text.
//
// ErrKeyMissing and ErrCanceled are separate standalone markers: neither belongs
// to a category. Credential errors retain the provider's setting locations;
// ErrCanceled is wrapped directly over the context error. Record and video-reuse
// sentinels also stand alone: their complete messages pass through the fallback
// renderer because none of the existing category messages describes those failures.
package errs

import (
	"errors"
	"fmt"
)

// Root sentinels:
//   - ErrCLI is the category root for command-line interface failures.
//   - ErrProcess is the category root for process-level failures (working dir, stdin).
//   - ErrTransport is the category root for HTTP/transport failures.
//   - ErrResponse is the category root for malformed or error provider responses.
//   - ErrInputMedia is the category root for input-media failures.
//   - ErrModelResolve is the category root for model-input resolution failures.
//   - ErrOutputFile is the category root for output-file failures.
//   - ErrProvConfig is the category root for invalid embedded provider configs.
//   - ErrParamValue is the category root for typed parameter-value parsing failures.
//   - ErrOutputPage is the category root for embedded output-page template failures.
//   - ErrSearch is the category root for list-search failures.
//   - ErrUserConfig is the category root for optional user configuration faults.
//   - ErrJSON is the category root for JSON encoding and decoding failures outside a
//     provider response.
//   - ErrKeyMissing marks an unset provider credential; the provider names the
//     environment variable and user-config key in the wrapping context.
//   - ErrCanceled marks a run canceled through the command context (Ctrl-C);
//     waits and transport surface it so cancellation interrupts long work.
var (
	ErrCLI          = errors.New("CLI")
	ErrProcess      = errors.New("process")
	ErrTransport    = errors.New("transport")
	ErrResponse     = errors.New("provider response")
	ErrInputMedia   = errors.New("input media")
	ErrModelResolve = errors.New("model resolution")
	ErrOutputFile   = errors.New("output file")
	ErrProvConfig   = errors.New("provider config")
	ErrParamValue   = errors.New("parameter value")
	ErrOutputPage   = errors.New("output page")
	ErrSearch       = errors.New("search")
	ErrUserConfig   = errors.New("user config")
	ErrJSON         = errors.New("JSON")
	ErrKeyMissing   = errors.New("is not set")
	ErrCanceled     = errors.New("generation canceled")
)

// CLI sentinels:
//   - ErrCLIFlagParse marks a command-line flag parse failure.
//   - ErrCLIPromptMissing marks a missing or empty prompt argument.
//   - ErrCLINoModelInput marks a missing --model input when the command requires one.
//   - ErrCLINoArgs marks a command that takes no arguments but was given one or more.
//   - ErrCLIOneArg marks a command that takes exactly one argument but was given none or more than one.
//   - ErrCLIOneArgMax marks a command that takes at most one argument but was given more than one.
//   - ErrCLIUnknownHelpTopic marks a help argument naming no command.
//   - ErrCLISearchPattern marks a search or exclusion term that does not compile as a regular expression under --regex.
//   - ErrCLISearchTermMissing marks a search given neither a search term nor an exclusion term.
//   - ErrCLISearchTermEmpty marks a search term or an exclusion term typed as the empty string.
//   - ErrCLIHelpCombined marks a help or version flag given beside other flags or positional arguments.
//   - ErrCLIFlagsBeforeCommand marks a flag placed before a command word.
//   - ErrCLINoDefaultModel marks an omitted --model when no configured provider carries a default model.
var (
	ErrCLIFlagParse          = fmt.Errorf("%w: flag parse failed", ErrCLI)
	ErrCLIPromptMissing      = fmt.Errorf("%w: prompt missing", ErrCLI)
	ErrCLINoModelInput       = fmt.Errorf("%w: no model input", ErrCLI)
	ErrCLINoArgs             = fmt.Errorf("%w: command takes no arguments", ErrCLI)
	ErrCLIOneArg             = fmt.Errorf("%w: command takes exactly one argument", ErrCLI)
	ErrCLIOneArgMax          = fmt.Errorf("%w: command takes only one argument", ErrCLI)
	ErrCLIUnknownHelpTopic   = fmt.Errorf("%w: no help topic for that name", ErrCLI)
	ErrCLISearchPattern      = fmt.Errorf("%w: search pattern is not a valid regular expression", ErrCLI)
	ErrCLISearchTermMissing  = fmt.Errorf("%w: search term missing", ErrCLI)
	ErrCLISearchTermEmpty    = fmt.Errorf("%w: search term empty", ErrCLI)
	ErrCLIHelpCombined       = fmt.Errorf("%w: help or version flag combined with other input", ErrCLI)
	ErrCLIFlagsBeforeCommand = fmt.Errorf("%w: flags placed before a command word", ErrCLI)
	ErrCLINoDefaultModel     = fmt.Errorf("%w: no default model among the configured providers", ErrCLI)
)

// Process sentinels:
//   - ErrProcessWorkingDir marks a failure to get the current working directory.
//   - ErrProcessReadInput marks a failure to read the prompt from stdin.
var (
	ErrProcessWorkingDir = fmt.Errorf("%w: get working dir failed", ErrProcess)
	ErrProcessReadInput  = fmt.Errorf("%w: read input failed", ErrProcess)
)

// Transport sentinels:
//   - ErrTransportMarshal marks a failed request-body marshal.
//   - ErrTransportCreate marks a failed HTTP request construction.
//   - ErrTransportRequest marks a failed HTTP request execution.
//   - ErrTransportRead marks a failed response-body read.
//   - ErrTransportSize marks a body that exceeds the read limit.
//   - ErrTransportMultipart marks a failed multipart-body assembly.
//   - ErrTransportStatus marks a non-2xx status on a raw (non-API) download.
//   - ErrTransportDownload marks a failed content download.
//   - ErrTransportTimeout marks a poll that exceeded its deadline.
//   - ErrTransportRedirectLimit marks a request that exceeded the redirect limit.
var (
	ErrTransportMarshal       = fmt.Errorf("%w: request marshal failed", ErrTransport)
	ErrTransportCreate        = fmt.Errorf("%w: request creation failed", ErrTransport)
	ErrTransportRequest       = fmt.Errorf("%w: request failed", ErrTransport)
	ErrTransportRead          = fmt.Errorf("%w: response read failed", ErrTransport)
	ErrTransportSize          = fmt.Errorf("%w: response exceeds size limit", ErrTransport)
	ErrTransportMultipart     = fmt.Errorf("%w: multipart assembly failed", ErrTransport)
	ErrTransportStatus        = fmt.Errorf("%w: unexpected response status", ErrTransport)
	ErrTransportDownload      = fmt.Errorf("%w: download failed", ErrTransport)
	ErrTransportTimeout       = fmt.Errorf("%w: timed out", ErrTransport)
	ErrTransportRedirectLimit = fmt.Errorf("%w: redirect limit reached", ErrTransport)
)

// Response sentinels:
//   - ErrResponseDecode marks a response body that could not be decoded.
//   - ErrResponseNoData marks a response missing its expected data.
//   - ErrResponseStatus marks a non-2xx provider API response.
//   - ErrResponseServer marks a provider server message extracted from an
//     error response's documented body shape; the extracted text is the
//     ProviderError message, retained separately for the renderer.
//   - ErrResponseGen marks a provider-reported generation failure.
//   - ErrResponseNoSample marks a ready response that lacks a sample.
//   - ErrResponseUnknown marks an unrecognized poll status.
//   - ErrResponseNoJobID marks a start response that lacks a job ID.
//   - ErrResponseCodeMissing marks a Kling envelope with no code field; it
//     refines ErrResponseNoData.
//   - ErrResponseCodeInvalid marks a Kling envelope code that is not an
//     integer; it refines ErrResponseDecode.
//   - ErrResponseTaskDataMissing marks a task response with no data object;
//     it refines ErrResponseNoData.
//   - ErrResponseTaskRecordInvalid marks a task record that is not an
//     object; it refines ErrResponseDecode.
//   - ErrResponseResultInvalid marks a result record that is not an object;
//     it refines ErrResponseDecode.
//   - ErrResponseResultIndexInvalid marks a result record whose index is not
//     an integer; it refines ErrResponseDecode.
//   - ErrResponseResultIndexDuplicate marks two result records sharing one
//     index; it refines ErrResponseDecode.
//   - ErrResponseOutputInvalid marks an output record that is not an object;
//     it refines ErrResponseDecode.
//   - ErrResponseJobStatusMissing marks a job response with no status; it
//     refines ErrResponseNoData.
var (
	ErrResponseDecode          = fmt.Errorf("%w: decode failed", ErrResponse)
	ErrResponseNoData          = fmt.Errorf("%w: no data", ErrResponse)
	ErrResponseStatus          = fmt.Errorf("%w: error status", ErrResponse)
	ErrResponseStatusTemporary = fmt.Errorf("%w: temporary", ErrResponseStatus)
	ErrResponseServer          = fmt.Errorf("%w: server message", ErrResponse)
	ErrResponseGen             = fmt.Errorf("%w: generation failed", ErrResponse)
	ErrResponseNoSample        = fmt.Errorf("%w: no sample in the ready result", ErrResponse)
	ErrResponseUnknown         = fmt.Errorf("%w: unexpected status", ErrResponse)
	ErrResponseNoJobID         = fmt.Errorf("%w: no job id in the start response", ErrResponse)

	ErrResponseCodeMissing          = fmt.Errorf("%w: envelope code missing", ErrResponseNoData)
	ErrResponseCodeInvalid          = fmt.Errorf("%w: envelope code invalid", ErrResponseDecode)
	ErrResponseTaskDataMissing      = fmt.Errorf("%w: task data missing", ErrResponseNoData)
	ErrResponseTaskRecordInvalid    = fmt.Errorf("%w: task record invalid", ErrResponseDecode)
	ErrResponseResultInvalid        = fmt.Errorf("%w: result record invalid", ErrResponseDecode)
	ErrResponseResultIndexInvalid   = fmt.Errorf("%w: result index invalid", ErrResponseDecode)
	ErrResponseResultIndexDuplicate = fmt.Errorf("%w: result index duplicated", ErrResponseDecode)
	ErrResponseOutputInvalid        = fmt.Errorf("%w: output record invalid", ErrResponseDecode)
	ErrResponseJobStatusMissing     = fmt.Errorf("%w: job status missing", ErrResponseNoData)
)

// InputMedia sentinels:
//   - ErrInputMediaNotFound marks a missing input-media path.
//   - ErrInputMediaRead marks a failed input-media read.
//   - ErrInputMediaMIME marks an unsupported input-media MIME type.
//   - ErrInputMediaSource marks an invalid source path or URL.
//   - ErrInputMediaTime marks an invalid frame time or incompatible frame selection.
//   - ErrInputMediaEmpty marks empty input media.
//   - ErrInputMediaSize marks a reference-image size that is not a WxH value.
//   - ErrInputMediaDecode marks a failed reference-image decode.
//   - ErrInputMediaEncode marks a failed reference-image encode.
var (
	ErrInputMediaNotFound = fmt.Errorf("%w: not found", ErrInputMedia)
	ErrInputMediaRead     = fmt.Errorf("%w: read failed", ErrInputMedia)
	ErrInputMediaMIME     = fmt.Errorf("%w: MIME not supported", ErrInputMedia)
	ErrInputMediaSource   = fmt.Errorf("%w: source invalid", ErrInputMedia)
	ErrInputMediaTime     = fmt.Errorf("%w: frame time invalid", ErrInputMedia)
	ErrInputMediaEmpty    = fmt.Errorf("%w: empty", ErrInputMedia)
	ErrInputMediaSize     = fmt.Errorf("%w: invalid size", ErrInputMedia)
	ErrInputMediaDecode   = fmt.Errorf("%w: decode failed", ErrInputMedia)
	ErrInputMediaEncode   = fmt.Errorf("%w: encode failed", ErrInputMedia)
)

// ErrInputMediaUnsendable marks input that the selected provider route cannot carry.
var ErrInputMediaUnsendable = fmt.Errorf("%w: cannot be sent by the selected route", ErrInputMedia)

// Model resolution sentinels:
//   - ErrModelResolveUnknown marks an unrecognized --model input.
//   - ErrModelResolveConflict marks a model input matching multiple catalog models.
var (
	ErrModelResolveUnknown  = fmt.Errorf("%w: unrecognized model specifier", ErrModelResolve)
	ErrModelResolveConflict = fmt.Errorf("%w: ambiguous model specifier", ErrModelResolve)
)

// Output sentinels:
//   - ErrOutputFileHome marks a failed home-directory lookup.
//   - ErrOutputFileMkdir marks a failed output-directory creation.
//   - ErrOutputFileEmpty marks an empty artifact set.
//   - ErrOutputFileWrite marks a failed artifact write.
//   - ErrOutputFileClose marks a failed file close.
//   - ErrOutputFileCreate marks a failed exclusive file create.
//   - ErrOutputFileRemove marks a failed removal of an owned temporary file.
var (
	ErrOutputFileHome   = fmt.Errorf("%w: home directory lookup failed", ErrOutputFile)
	ErrOutputFileMkdir  = fmt.Errorf("%w: directory creation failed", ErrOutputFile)
	ErrOutputFileEmpty  = fmt.Errorf("%w: no artifacts to write", ErrOutputFile)
	ErrOutputFileWrite  = fmt.Errorf("%w: write failed", ErrOutputFile)
	ErrOutputFileClose  = fmt.Errorf("%w: close failed", ErrOutputFile)
	ErrOutputFileCreate = fmt.Errorf("%w: create failed", ErrOutputFile)
	ErrOutputFileRemove = fmt.Errorf("%w: remove failed", ErrOutputFile)
)

// Provider-config sentinels
//   - ErrProvConfigDecode marks an embedded provider config that failed to decode
//     (malformed JSON or an unknown field).
//   - ErrProvConfigInvalid marks a decoded provider config with invalid content (bad
//     enum, missing medium description, invalid numeric field).
//   - ErrProvConfigDupAlias marks an alias token declared twice across the
//     catalog's healthy configs.
//   - ErrProvConfigDupModel marks a duplicate provider-model pair.
//   - ErrProvConfigNoConstructor marks a provider whose config loaded but for which
//     no constructor was registered: a wiring defect in the provider table.
//   - ErrProvConfigNotLoaded marks a provider the catalog holds no healthy config
//     for, whether its config failed to load or the provider was never registered.
//   - ErrProvConfigConstructorFailed marks a constructor that returned an interface
//     holding a nil pointer, which compares unequal to nil yet fails on the first
//     method call.
//   - ErrProvConfigNilInterface marks a constructor that returned a nil Generator.
//   - ErrProvConfigTrailing marks a provider config document with
//     trailing content after the JSON object.
//   - ErrProvConfigNoAdapterAPI marks a provider config that lacks an AdapterAPI section.
//   - ErrProvConfigParamUnplaced marks a declared parameter that the provider's adapter
//     has no request field for, so its value could not reach the request.
var (
	ErrProvConfigDecode            = fmt.Errorf("%w: decode failed", ErrProvConfig)
	ErrProvConfigInvalid           = fmt.Errorf("%w: invalid content", ErrProvConfig)
	ErrProvConfigDupAlias          = fmt.Errorf("%w: duplicate alias", ErrProvConfig)
	ErrProvConfigDupModel          = fmt.Errorf("%w: duplicate provider-model pair", ErrProvConfig)
	ErrProvConfigNoConstructor     = fmt.Errorf("%w: no constructor for provider", ErrProvConfig)
	ErrProvConfigNotLoaded         = fmt.Errorf("%w: not loaded", ErrProvConfig)
	ErrProvConfigConstructorFailed = fmt.Errorf("%w: constructor failed for provider", ErrProvConfig)
	ErrProvConfigNilInterface      = fmt.Errorf("%w: constructor returned nil interface", ErrProvConfig)
	ErrProvConfigTrailing          = fmt.Errorf("%w: trailing content after the provider config document", ErrProvConfig)

	ErrProvConfigNoAdapterAPI  = fmt.Errorf("%w: no AdapterAPI section", ErrProvConfig)
	ErrProvConfigParamUnplaced = fmt.Errorf("%w: declared parameter has no request field", ErrProvConfig)
)

// Parameter-value sentinels:
//   - ErrParamValueNumber marks text that cannot be parsed as a number.
//   - ErrParamValueInteger marks text that cannot be parsed as an integer.
//   - ErrParamValueBoolean marks text that cannot be parsed as a boolean.
//   - ErrParamValueDataType marks a data type outside the supported vocabulary.
//   - ErrParamValueTypeMismatch marks a stored parameter value whose type differs from the
//     type its flag declares.
var (
	ErrParamValueNumber       = fmt.Errorf("%w: not a number", ErrParamValue)
	ErrParamValueInteger      = fmt.Errorf("%w: not an integer", ErrParamValue)
	ErrParamValueBoolean      = fmt.Errorf("%w: not a boolean", ErrParamValue)
	ErrParamValueDataType     = fmt.Errorf("%w: unsupported data type", ErrParamValue)
	ErrParamValueTypeMismatch = fmt.Errorf("%w: stored value of another type", ErrParamValue)
)

// Output-page sentinels:
//   - ErrOutputPageParse marks an embedded template parse failure.
//   - ErrOutputPageUnknown marks a page name with no template.
//   - ErrOutputPageExecute marks an embedded template execution failure.
var (
	ErrOutputPageParse   = fmt.Errorf("%w: template parse failed", ErrOutputPage)
	ErrOutputPageUnknown = fmt.Errorf("%w: template not found", ErrOutputPage)
	ErrOutputPageExecute = fmt.Errorf("%w: template execution failed", ErrOutputPage)
)

// Search sentinels:
//   - ErrSearchTooManyResults marks a list search whose provider and model
//     results together exceed the display limit.
var (
	ErrSearchTooManyResults = fmt.Errorf("%w: too many results to display", ErrSearch)
)

// JSON sentinels:
//   - ErrJSONEncode marks a value that could not be encoded as JSON.
//   - ErrJSONDecode marks JSON data that could not be decoded into its value.
var (
	ErrJSONEncode = fmt.Errorf("%w: encode failed", ErrJSON)
	ErrJSONDecode = fmt.Errorf("%w: decode failed", ErrJSON)
)

// Experimental generation-record and reuse failures.
//   - ErrReuseSyntax rejects an incomplete key/value selection.
//   - ErrReuseProvider rejects identifiers unsupported by the selected provider.
//   - ErrRecordRead retains the cause of a failed record read.
//   - ErrRecordInvalid rejects malformed or unsupported record documents.
//   - ErrReuseVideoURI rejects a missing or unusable original Google reference.
//   - ErrReuseInputMedia rejects simultaneous reuse and input media.
//   - ErrReuseVideoModel rejects incompatible extension models or source settings.
//   - ErrReuseVideoMultiple rejects an ambiguous retained video selection.
//
//nolint:staticcheck // The owner-approved user-facing sentences retain their capitalization.
var (
	ErrReuseSyntax   = fmt.Errorf("%w: Reuse requires IDENTIFIER=VALUE", ErrCLI)
	ErrReuseProvider = fmt.Errorf("%w: Reuse is not supported by the selected provider", ErrCLI)
	//lint:ignore ST1005 Preserve the owner's approved user-facing wording.
	ErrRecordRead = errors.New("Generation record could not be read")
	//lint:ignore ST1005 Preserve the owner's approved user-facing wording.
	ErrRecordInvalid = errors.New("Generation record is invalid or uses an unsupported schema version")
	//lint:ignore ST1005 Preserve the owner's approved user-facing wording.
	ErrReuseVideoURI = errors.New("Veo extension requires an original Google video URI")
	//lint:ignore ST1005 Preserve the owner's approved user-facing wording.
	ErrReuseInputMedia = errors.New("Veo extension cannot be combined with input media")
	//lint:ignore ST1005 Preserve the owner's approved user-facing wording.
	ErrReuseVideoModel = errors.New("Veo extension requires a compatible Veo model and a 720p source")
	//lint:ignore ST1005 Preserve the owner's approved user-facing wording.
	ErrReuseVideoMultiple = errors.New("This record contains multiple videos; supply the chosen URI directly")
)

// User-config sentinels:
//   - ErrUserConfigRead marks a config file that exists but could not be read.
//   - ErrUserConfigDecode marks a config file whose content could not be
//     decoded as the expected YAML document.
//   - ErrUserConfigLocate marks a failed lookup of the config file's
//     location (the home directory could not be resolved).
//   - ErrUserConfigUnknownSetting marks a top-level key the schema does not
//     declare; the known settings still apply.
//   - ErrUserConfigUnknownProvider marks a non-empty api-keys entry naming a
//     provider the catalog does not hold; the remaining entries still apply.
var (
	ErrUserConfigRead            = fmt.Errorf("%w: file read failed", ErrUserConfig)
	ErrUserConfigDecode          = fmt.Errorf("%w: decode failed", ErrUserConfig)
	ErrUserConfigLocate          = fmt.Errorf("%w: location lookup failed", ErrUserConfig)
	ErrUserConfigUnknownSetting  = fmt.Errorf("%w: unknown setting", ErrUserConfig)
	ErrUserConfigUnknownProvider = fmt.Errorf("%w: unknown provider under api-keys", ErrUserConfig)
)
