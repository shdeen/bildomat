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

// The status vocabulary of the JSON outcomes, shared with the run boundary.
//   - StatusCompleted: the run produced its artifacts
//   - StatusCanceled: the user canceled the run
//   - StatusFailed: the run stopped on a failure
const (
	StatusCompleted = "completed"
	StatusCanceled  = "canceled"
	StatusFailed    = "failed"
)

// fallbackFailureOutcome is the failure outcome written when the encoder
// itself fails, spelled out because nothing else can be encoded at that point.
const fallbackFailureOutcome = `{"status":"failed","errors":[]}`

// FailureOutcome is the JSON result for a command that failed before producing a more
// specific outcome.
type FailureOutcome struct {
	Status string   `json:"status"`
	Errors []string `json:"errors"`
}

// SavedFile is one file written by a generation, in the same user-facing terms as the
// regular Saved line.
type SavedFile = artifact.SavedFile

// Adjustment is one user-facing flag adjustment made before provider submission.
type Adjustment struct {
	Flag      string `json:"flag"`
	Submitted string `json:"submitted,omitempty"`
	Used      string `json:"used,omitempty"`
	Notice    string `json:"notice"`
}

// GenerationOutcome is the complete user-facing JSON result of one generation request.
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

// NewGenerationOutcome takes a start time, prompt, and submitted flags and returns the
// initial JSON generation result.
func NewGenerationOutcome(startedAt time.Time, prompt string, flags map[string]any) *GenerationOutcome {
	return &GenerationOutcome{
		Timestamp: documentTimestamp(startedAt),
		Prompt:    prompt,
		Flags:     flags,
	}
}

// documentTimestamp takes a time and returns it as the timestamp the JSON
// document and the sidecar front matter carry: UTC, in RFC 3339 form with
// fractional seconds.
func documentTimestamp(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339Nano)
}

// AdjustmentRecords takes the parameter-flag definitions and parameter-change
// records and returns the records' user-facing JSON representation without
// provider parameter names.
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

// NoticeTexts takes the parameter-flag definitions and parameter-change
// records and returns each record's regular user-facing notice text for JSON
// result notices.
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
		errorText = UserErrorText(err, providerDisplayName, modelName, debugMode)
	}

	return FailureOutcome{Status: StatusFailed, Errors: []string{errorText}}
}

// PrintJSON writes exactly one indented JSON document to destination. A fallback
// failure document does not erase the original encoding failure. Failed delivery
// retains its write cause alongside any encoding cause.
func PrintJSON(destination io.Writer, value any) error {
	encoded, encodeErr := encodeJSON(value)
	if encodeErr != nil {
		fallback, fallbackErr := json.Marshal(FailureOutcome{
			Status: StatusFailed,
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

// encodeJSON takes an outcome or an info page and returns it as one indented
// JSON value ending in a newline, without the HTML escaping of angle brackets
// and ampersands that the encoder applies by default, or the serialization
// error.
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

// Complete records the elapsed time, final status, and generation error before
// delivery. A later delivery failure cannot alter a document already written.
func (outcome *GenerationOutcome) Complete(startedAt time.Time, providerDisplayName, modelName string, generationErr error, debugMode bool) {
	outcome.DurationMS = time.Since(startedAt).Milliseconds()

	outcome.Status = generationStatus(generationErr, outcome.Status)
	if generationErr != nil {
		outcome.Errors = append(outcome.Errors, UserErrorText(generationErr, providerDisplayName, modelName, debugMode))
	}
}

// generationStatus takes the run's error and the generation outcome's
// current status and returns the status the completed outcome carries:
// canceled for a canceled run, failed for any other error, and otherwise the
// current status, or completed when none was set.
func generationStatus(runErr error, outcomeStatus string) string {
	switch {
	case errors.Is(runErr, errs.ErrCanceled):
		return StatusCanceled
	case runErr != nil:
		return StatusFailed
	case outcomeStatus == "":
		return StatusCompleted
	default:
		return outcomeStatus
	}
}
