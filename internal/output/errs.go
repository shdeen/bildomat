// File: internal/output/errs.go
// Failure rendering and prompt styling use the copy catalog. Messages identify
// offending values; raw diagnostic chains require the raw-error selection.

package output

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	tmpl "github.com/shdeen/bildomat/internal/templates"
)

// compactUsageTemplateName names the parsed compact-usage template.
const compactUsageTemplateName = "compact usage"

//nolint:gochecknoglobals // parsed once at package load from the embedded copy, never written again.
var compactUsageTemplate, compactUsageTemplateErr = template.New(compactUsageTemplateName).Parse(tmpl.CompactUsageText)

type compactUsageData struct {
	Prefix string
	Error  string
}

// errorPrefix returns the ordinary error prefix with the selected styling.
func errorPrefix(styled bool) string {
	return accentedPrefix(ErrorPrefix+":", styled)
}

// accentedPrefix adds the selected diagnostic accent and the separating space.
func accentedPrefix(prefixWord string, styled bool) string {
	if styled {
		return ansiErrorAccent + prefixWord + ansiReset + " "
	}

	return prefixWord + " "
}

// promptStyleValues returns the requested prompt accents, or empty strings.
func promptStyleValues(styled bool) (steel, clay, reset string) {
	if styled {
		return ansiSteelBlue, ansiClay, ansiReset
	}

	return "", "", ""
}

// FileIsTTY reports whether a stream is a character device. An absent stream or
// failed stat cannot enable terminal presentation.
func FileIsTTY(stream *os.File) bool {
	if stream == nil {
		return false
	}

	info, err := stream.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// PrintUsageError writes the compact usage diagnostic. A malformed template
// still permits the primary message to be delivered, while retaining both causes.
func PrintUsageError(destination io.Writer, err error, debugMode, styled bool) error {
	errorText := usageErrorText(err, debugMode)
	usageData := compactUsageData{Prefix: errorPrefix(styled), Error: errorText}

	var (
		rendered  strings.Builder
		renderErr error
	)
	if compactUsageTemplateErr != nil {
		renderErr = fmt.Errorf("%q: %w, %w", compactUsageTemplateName, errs.ErrOutputPageParse, compactUsageTemplateErr)
	} else if err := compactUsageTemplate.Execute(&rendered, usageData); err != nil {
		renderErr = fmt.Errorf("%q: %w, %w", compactUsageTemplateName, errs.ErrOutputPageExecute, err)
	}

	if renderErr != nil {
		return errors.Join(renderErr, WriteText(destination, "%s%s\n", usageData.Prefix, errorText))
	}

	return WriteText(destination, "%s", rendered.String())
}

// PrintError writes a mapped failure, cancellation notice, or raw diagnostic
// chain to the selected diagnostic destination and returns any delivery failure.
func PrintError(destination io.Writer, err error, providerDisplayName, modelName string, debugMode, styled bool) error {
	if debugMode {
		return WriteText(destination, "%s\n", RawErrorChain(err))
	}

	if errors.Is(err, errs.ErrCanceled) {
		return WriteText(destination, "%s\n", ErrorMessage(err, providerDisplayName, modelName))
	}

	return WriteText(destination, "%s%s\n", errorPrefix(styled), ErrorMessage(err, providerDisplayName, modelName))
}

// PrintUserConfigWarning writes a configuration warning to the command's
// diagnostic stream in every result mode and returns any delivery failure.
func PrintUserConfigWarning(destination io.Writer, fault error, styled bool) error {
	return WriteText(destination, "%s%s\n", accentedPrefix(WarningPrefix+":", styled), configWarningMessage(fault))
}

// configWarningMessage renders the configuration path and offending setting
// from the producer's explicit fields, including through wrapping and joins.
func configWarningMessage(fault error) string {
	var detail *errs.ConfigError
	errors.As(fault, &detail)

	if detail == nil {
		detail = &errs.ConfigError{}
	}

	switch {
	case errors.Is(fault, errs.ErrUserConfigLocate):
		return ConfigLocationUnresolved
	case errors.Is(fault, errs.ErrUserConfigUnknownSetting):
		return fmt.Sprintf(ConfigUnknownSetting, oneNotice(detail.Path), oneNotice(detail.Setting))
	case errors.Is(fault, errs.ErrUserConfigUnknownProvider):
		return fmt.Sprintf(ConfigUnknownProvider, oneNotice(detail.Path), oneNotice(detail.Provider))
	default:
		return fmt.Sprintf(ConfigFileUnreadable, oneNotice(detail.Path))
	}
}

// candidateIndent is the indentation of each candidate key under the
// disambiguation heading.
const candidateIndent = "  "

// PrintAmbiguity writes the ambiguous specifier and its fully qualified
// candidates to the selected diagnostic destination.
func PrintAmbiguity(destination io.Writer, modelSpecifier string, modelMatches []catalog.ProvModelPair, styled bool) error {
	steel, clay, reset := promptStyleValues(styled)

	var rendered strings.Builder
	fmt.Fprintf(&rendered, AmbiguityHeading+"\n", steel, clay, reset, modelSpecifier)

	for matchIndex := range modelMatches {
		fmt.Fprintf(&rendered, candidateIndent+AmbiguityCandidate+"\n", clay, modelMatches[matchIndex].Provider.ID, modelMatches[matchIndex].Model.ID, reset)
	}

	return WriteText(destination, "%s", rendered.String())
}

// UserErrorText takes an error, the resolved provider and model names, and the
// raw-error selection and returns the selected user-facing or raw diagnostic text.
func UserErrorText(err error, providerDisplayName, modelName string, debugMode bool) string {
	if debugMode {
		return RawErrorChain(err)
	}

	return ErrorMessage(err, providerDisplayName, modelName)
}

// usageErrorText returns the usage explanation or the complete debug diagnostic.
func usageErrorText(err error, debugMode bool) string {
	if debugMode {
		return RawErrorChain(err)
	}

	if failure, matched := errors.AsType[*errs.UsageError](err); matched {
		return oneNotice(failure.Text)
	}

	return plainUsageText(err)
}

// plainUsageText takes a usage error and returns its chain text without the
// CLI category prefix.
func plainUsageText(err error) string {
	errorText := oneNotice(err.Error())
	errorText = strings.TrimPrefix(errorText, errs.ErrCLI.Error()+": ")

	return strings.Replace(errorText, ", "+errs.ErrCLI.Error()+": ", ": ", 1)
}

// RawErrorChain takes an error and returns its complete chain as one string, quoting text
// that contains control or replacement runes. It returns an empty string for a nil error.
func RawErrorChain(err error) string {
	if err == nil {
		return ""
	}

	return oneNotice(err.Error())
}

// ErrorMessage renders the principal failure and every affected output path.
// Typed details survive nested and joined errors; control characters are escaped.
// A nil error returns an empty string.
func ErrorMessage(err error, providerDisplayName, modelName string) string {
	if err == nil {
		return ""
	}

	notices, operationErr := outputFailureNotices(err)

	primaryNotice := operationNotice(err, oneNotice(providerDisplayName), modelName)
	if primaryNotice == "" && operationErr != nil {
		primaryNotice = operationNotice(operationErr, oneNotice(providerDisplayName), modelName)
	}

	if primaryNotice != "" {
		notices = append([]string{primaryNotice}, notices...)
	}

	return strings.Join(notices, "; ")
}

// operationNotice selects the principal operation's explanation without
// allowing a later output cleanup failure to hide it.
func operationNotice(err error, providerDisplayName, modelName string) string {
	var usageFailure *errs.UsageError
	switch {
	case errors.As(err, &usageFailure):
		return oneNotice(usageFailure.Text)
	case errors.Is(err, errs.ErrKeyMissing):
		return credentialNotice(err, providerDisplayName)
	case errors.Is(err, errs.ErrCanceled):
		return GenerationCanceled
	case errors.Is(err, errs.ErrProcessWorkingDir):
		return WorkingDirFailed
	case errors.Is(err, errs.ErrInputMedia):
		return inputMediaNotice(err)
	case errors.Is(err, errs.ErrModelResolve):
		return modelResolveNotice(err)
	case errors.Is(err, errs.ErrProvConfig):
		return provConfigNotice(err, providerDisplayName)
	case errors.Is(err, errs.ErrSearchTooManyResults):
		return SearchTooManyResults
	case errors.Is(err, errs.ErrTransport), errors.Is(err, errs.ErrResponse):
		return providerFailureNotice(err, providerDisplayName, modelName)
	case errors.Is(err, errs.ErrOutputFile):
		return ""
	default:
		return innermostMessage(err)
	}
}

// outputFailureNotices traverses joined causes and reports each output path
// once. It retains a separate operation cause for the ordinary fallback message.
// Filesystem errors from input media do not become output failures.
func outputFailureNotices(err error) ([]string, error) {
	if !errors.Is(err, errs.ErrOutputFile) {
		return nil, nil
	}

	var (
		notices      []string
		operationErr error
	)

	reportedPaths := map[string]bool{}

	pendingErrors := []error{err}
	for len(pendingErrors) > 0 {
		currentError := pendingErrors[0]
		pendingErrors = pendingErrors[1:]

		if !errors.Is(currentError, errs.ErrOutputFile) {
			if operationErr == nil {
				operationErr = currentError
			}

			continue
		}
		//nolint:errorlint // Inspect this layer only; errors.As would repeatedly find descendants and lose sibling paths.
		switch failure := currentError.(type) {
		case *os.PathError:
			if !reportedPaths[failure.Path] {
				noticeForm := OutputWriteFailed
				if errors.Is(failure, errs.ErrOutputFileRemove) {
					noticeForm = OutputCleanupFailed
				}

				notices = append(notices, fmt.Sprintf(noticeForm, oneNotice(failure.Path)))
				reportedPaths[failure.Path] = true
			}
		case interface{ Unwrap() []error }:
			pendingErrors = append(pendingErrors, failure.Unwrap()...)
		case interface{ Unwrap() error }:
			pendingErrors = append(pendingErrors, failure.Unwrap())
		}
	}

	if len(notices) == 0 {
		return []string{fmt.Sprintf(OutputWriteFailed, "")}, operationErr
	}

	return notices, operationErr
}

// credentialNotice takes a missing-credential error and provider display name and returns
// guidance for setting the API key.
func credentialNotice(err error, providerDisplayName string) string {
	statement := keyMissingStatement(err)
	if providerDisplayName == "" {
		return fmt.Sprintf(CredentialMissing, statement)
	}

	return fmt.Sprintf(CredentialMissingProvider, statement, providerDisplayName)
}

// keyMissingStatement returns credential-setting locations from any matching
// branch, excluding outer request labels and unrelated cleanup failures.
func keyMissingStatement(err error) string {
	if failure, matched := errors.AsType[*errs.CredentialError](err); matched {
		return oneNotice(failure.Error())
	}

	return APIKeyNotSet
}

// inputMediaNotice renders the exact media source or actionable media problem.
func inputMediaNotice(err error) string {
	var detail *errs.MediaError
	errors.As(err, &detail)

	if detail == nil {
		detail = &errs.MediaError{}
	}

	switch {
	case errors.Is(err, errs.ErrInputMediaNotFound):
		return fmt.Sprintf(InputMediaNotFound, oneNotice(detail.Source))
	case errors.Is(err, errs.ErrInputMediaMIME):
		return fmt.Sprintf(InputMediaFormat, oneNotice(detail.Source))
	case errors.Is(err, errs.ErrInputMediaUnsendable):
		if detail.Problem != "" {
			return oneNotice(detail.Problem)
		}

		return InputMediaInvalid
	case errors.Is(err, errs.ErrInputMediaTime):
		if detail.Problem != "" {
			return oneNotice(detail.Problem)
		}

		return FrameTimeInvalidGeneric
	}

	description := detail.Source
	if description == "" {
		description = detail.Problem
	}

	if description != "" {
		return fmt.Sprintf(InputMediaSourceInvalid, oneNotice(description))
	}

	return InputMediaInvalid
}

// modelResolveNotice identifies the unresolved model from its structured error.
func modelResolveNotice(err error) string {
	if errors.Is(err, errs.ErrModelResolveConflict) {
		return ModelAmbiguous
	}

	var detail *errs.ModelError
	if errors.As(err, &detail) && detail.Specifier != "" {
		return fmt.Sprintf(ModelUnknown, oneNotice(detail.Specifier))
	}

	return ModelUnknownGeneric
}

// provConfigNotice renders configuration ownership and its specific problem.
func provConfigNotice(err error, providerDisplayName string) string {
	owner := providerDisplayName

	var detail *errs.ConfigError
	if !errors.As(err, &detail) {
		return fmt.Sprintf(ProviderConfigBroken, owner)
	}

	if detail.Provider != "" && (detail.Path != "" || owner == "") {
		owner = detail.Provider
	}

	description := detail.Path
	if detail.Problem != "" {
		if description != "" {
			description += ": "
		}

		description += detail.Problem
	}

	if description == "" {
		return fmt.Sprintf(ProviderConfigBroken, oneNotice(owner))
	}

	return fmt.Sprintf(ProviderConfigBrokenDetail, oneNotice(owner), oneNotice(description))
}

// providerFailureNotice takes a transport or response error with the resolved
// provider and model names and returns the classified failure's message. An
// error carrying an extracted server message renders the server-error
// formula; a response with no extractable message renders the generic
// unusable-response message; transport failures keep their timeout and
// unreachable messages.
func providerFailureNotice(err error, providerDisplayName, modelName string) string {
	provName := providerDisplay(providerDisplayName)

	if errors.Is(err, errs.ErrTransportTimeout) {
		message := fmt.Sprintf(TransportTimeout, provName)

		if failure, matched := errors.AsType[*errs.PollError](err); matched {
			identityModel := failure.Model
			if identityModel == "" {
				identityModel = modelName
			}

			message += " " + fmt.Sprintf(PollIdentity, oneNotice(identityModel), oneNotice(failure.Resource))
		}

		return message
	}

	if errors.Is(err, errs.ErrResponse) {
		if serverMsg := serverMessage(err); serverMsg != "" && modelName != "" {
			return fmt.Sprintf(ProviderServerError, provName, oneNotice(modelName), serverMsg)
		}

		return fmt.Sprintf(ResponseUnusable, provName)
	}

	return fmt.Sprintf(TransportUnreachable, provName)
}

// serverMessage returns the provider's explanation from any matching cause.
func serverMessage(err error) string {
	if detail, matched := errors.AsType[*errs.ProviderError](err); matched {
		return oneNotice(detail.Message)
	}

	return ""
}

// innermostMessage returns an unclassified error's innermost message: the
// deepest cause's own text, escaped for the terminal — the most specific
// information the chain holds, without the wrapping layers the raw-error
// selection exists for.
func innermostMessage(err error) string {
	for {
		//nolint:errorlint // deliberate: each layer's unwrap capability is inspected to walk to the deepest cause; errors.As targets a concrete type, which is not the question here.
		switch chainErr := err.(type) {
		case interface{ Unwrap() error }:
			next := chainErr.Unwrap()
			if next == nil {
				return oneNotice(err.Error())
			}

			err = next
		case interface{ Unwrap() []error }:
			causes := chainErr.Unwrap()
			if len(causes) == 0 {
				return oneNotice(err.Error())
			}

			err = causes[len(causes)-1]
		default:
			return oneNotice(err.Error())
		}
	}
}
