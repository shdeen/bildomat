package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// Status values used by JSON generation outcomes.
//   - statusCompleted: the run produced its artifacts
//   - StatusCanceled: the user canceled the run
//   - statusFailed: the run stopped on a failure
const (
	statusCompleted = "completed"
	StatusCanceled  = "canceled"
	statusFailed    = "failed"
)

// fallbackFailureOutcome is the literal result used if encoding the failure document also fails.
const fallbackFailureOutcome = `{"status":"failed","errors":[]}`

// FailureOutcome is the JSON result for a command that failed before producing a more specific
// outcome.
//   - Status: the failed command status
//   - Errors: the rendered command failure messages
type FailureOutcome struct {
	Status string   `json:"status"`
	Errors []string `json:"errors"`
}

// SavedFile is one file written by a generation, in the same user-facing terms as the regular Saved
// line.
type SavedFile = artifact.SavedFile

// Adjustment is one user-facing flag adjustment made before provider submission.
//   - Flag: the permanent parameter flag identifier
//   - Submitted, Used: the original and adjusted values in display form
//   - Notice: the rendered explanation of the change
type Adjustment struct {
	Flag      string `json:"flag"`
	Submitted string `json:"submitted,omitempty"`
	Used      string `json:"used,omitempty"`
	Notice    string `json:"notice"`
}

// GenerationOutcome is the complete user-facing JSON result of one generation request.
//   - Timestamp: the UTC start time in RFC 3339 format
//   - DurationMS: the elapsed run time in milliseconds
//   - Status: the completed, canceled, or failed result
//   - Provider, Model: the selected catalog identifiers
//   - Prompt: the submitted prompt
//   - Flags: the submitted values keyed by permanent flag identifier
//   - Adjustments: the changes made before submission
//   - Artifacts: the successfully saved files
//   - Notices: the rendered warnings and adjustment explanations
//   - Errors: the rendered run failure messages
type GenerationOutcome struct {
	Timestamp   string         `json:"timestamp"`
	DurationMS  int64          `json:"durationMs"`
	Status      string         `json:"status"`
	Provider    string         `json:"provider,omitempty"`
	Model       string         `json:"model,omitempty"`
	Prompt      string         `json:"prompt"`
	Flags       map[string]any `json:"flags"`
	Adjustments []Adjustment   `json:"adjustments,omitempty"`
	Artifacts   []SavedFile    `json:"artifacts,omitempty"`
	Notices     []string       `json:"notices,omitempty"`
	Errors      []string       `json:"errors,omitempty"`
}

// NewGenerationOutcome initializes a generation result with a UTC start time. The result retains
// the supplied flag map without copying it.
func NewGenerationOutcome(startedAt time.Time, prompt string, flags map[string]any) *GenerationOutcome {
	return &GenerationOutcome{
		Timestamp: documentTimestamp(startedAt),
		Prompt:    prompt,
		Flags:     flags,
	}
}

// documentTimestamp returns a UTC timestamp in RFC 3339 format with available fractional seconds.
func documentTimestamp(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339Nano)
}

// AdjustmentRecords formats parameter changes as user-facing JSON adjustment records.
func AdjustmentRecords(paramFlags []params.Flag, paramChanges []params.Adjustment, flagNames map[params.FlagType]string) []Adjustment {
	adjustments := make([]Adjustment, 0, len(paramChanges))
	for i := range paramChanges {
		paramChange := &paramChanges[i]
		adjustments = append(adjustments, Adjustment{
			Flag:      string(paramChange.FlagID),
			Submitted: paramChange.InputVal,
			Used:      paramChange.WireVal,
			Notice:    formatNotice(paramFlags, paramChange, flagNames),
		})
	}

	return adjustments
}

// NoticeTexts formats parameter changes as user-facing notice strings.
func NoticeTexts(paramFlags []params.Flag, paramChanges []params.Adjustment, flagNames map[params.FlagType]string) []string {
	notices := make([]string, 0, len(paramChanges))
	for i := range paramChanges {
		notices = append(notices, formatNotice(paramFlags, &paramChanges[i], flagNames))
	}

	return notices
}

// NewFailureOutcome returns the failure document for a command or usage error.
func NewFailureOutcome(err error, providerDisplayName, modelName string, debugMode, usageError bool) FailureOutcome {
	var errorText string
	if usageError {
		errorText = usageErrorText(err, debugMode)
	} else {
		errorText = userErrorText(err, providerDisplayName, modelName, debugMode)
	}

	return FailureOutcome{Status: statusFailed, Errors: []string{errorText}}
}

// PrintJSON writes one JSON document to destination, indenting successfully encoded values. If
// encoding fails, it writes a failure document and returns the original encoding error. A write
// failure is returned together with any encoding error.
func PrintJSON(destination io.Writer, value any) error {
	encoded, encodeErr := encodeJSON(value)
	if encodeErr != nil {
		fallback, fallbackErr := json.Marshal(FailureOutcome{
			Status: statusFailed,
			Errors: []string{encodeErr.Error()},
		})
		if fallbackErr != nil {
			encoded = []byte(fallbackFailureOutcome)
		} else {
			encoded = fallback
		}

		encoded = append(encoded, '\n')
	}

	return errors.Join(encodeErr, WriteText(destination, "%s", encoded))
}

// encodeJSON returns indented JSON followed by a newline, without HTML escaping.
func encodeJSON(value any) ([]byte, error) {
	var encoded bytes.Buffer

	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("%q, %w, %w", fmt.Sprintf("%T", value), errs.ErrJSONEncode, err)
	}

	return encoded.Bytes(), nil
}

// Complete records the elapsed time and final status, and appends a formatted generation error when
// one is present.
func (outcome *GenerationOutcome) Complete(startedAt time.Time, providerDisplayName, modelName string, generationErr error, debugMode bool) {
	outcome.DurationMS = time.Since(startedAt).Milliseconds()

	outcome.Status = generationStatus(generationErr, outcome.Status)
	if generationErr != nil {
		outcome.Errors = append(outcome.Errors, userErrorText(generationErr, providerDisplayName, modelName, debugMode))
	}
}

// generationStatus returns canceled or failed for an error. Otherwise it preserves an existing
// status or supplies completed.
func generationStatus(runErr error, outcomeStatus string) string {
	switch {
	case errors.Is(runErr, errs.ErrCanceled):
		return StatusCanceled
	case runErr != nil:
		return statusFailed
	case outcomeStatus == "":
		return statusCompleted
	default:
		return outcomeStatus
	}
}
