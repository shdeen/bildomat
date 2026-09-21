package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/shdeen/bildomat/internal/errs"
)

// The polling response fields and the decode failure context.
//   - pollFieldDetails: the response field carrying failure details
//   - pollFieldStatus: the response field carrying job status
//   - pollFieldError: the response field carrying a provider error
//   - bodyDecodeContext: the failed body decode's chain context
const (
	pollFieldStatus   = "status"
	pollFieldError    = "error"
	pollFieldDetails  = "details"
	bodyDecodeContext = "json decode"
)

// classifyResponse returns whether a job response indicates continued polling, completion, or failure.
func classifyResponse(jobID string, body []byte, runningWords []string, doneWord string, failedWords []string) (keepPolling, jobDone bool, err error) {
	pollResp, err := getRespBody(body)
	if err != nil {
		return false, false, fmt.Errorf(JobDecodeErrorForm, jobID, errors.Join(errs.ErrResponseDecode, err))
	}

	statusWord, ok := pollResp[pollFieldStatus].(string)

	pollErr, hasErrorField := pollResp[pollFieldError]
	isPollErr := hasErrorField && pollErr != nil

	switch {
	case isPollErr:
		return false, false, getGenErr(jobID, statusWord, pollFailureMsg(pollResp))
	case !ok:
		return false, false, getUnknownErr(jobID, pollResp[pollFieldStatus], body)
	case slices.Contains(runningWords, statusWord):
		return true, false, nil
	case statusWord == doneWord:
		return false, true, nil
	case slices.Contains(failedWords, statusWord):
		return false, false, getGenErr(jobID, statusWord, pollFailureMsg(pollResp))
	default:
		return false, false, getUnknownErr(jobID, pollResp[pollFieldStatus], body)
	}
}

// pollFailureMsg returns the failure message carried by a polling response.
func pollFailureMsg(pollBody map[string]any) string {
	switch apiError := pollBody[pollFieldError].(type) {
	case map[string]any:
		if msg, ok := apiError["message"].(string); ok && msg != "" {
			return msg
		}
	case string:
		if apiError != "" {
			return apiError
		}
	}

	if d, ok := pollBody[pollFieldDetails].(string); ok && d != "" {
		return d
	}

	return ""
}

// getGenErr returns a generation error containing the job identifier, status, and optional message.
func getGenErr(jobID, statusWord, msg string) error {
	diagnostic := fmt.Sprintf("%s (%s)", jobID, statusWord)
	if msg != "" {
		diagnostic += ": " + msg
	}

	return &errs.ProviderError{Message: diagnostic, Cause: errs.ErrResponseGen}
}

// getUnknownErr returns an error describing an unrecognized job status.
func getUnknownErr(jobID string, observed any, body []byte) error {
	obs := MissingResponseValue

	switch v := observed.(type) {
	case nil: // absent, or a JSON null
	case string:
		obs = strconv.Quote(v)
	default:
		obs = fmt.Sprintf("%v", v)
	}

	diagnostic := fmt.Sprintf(UnknownStatusForm, jobID, obs, errs.Excerpt(body))

	return fmt.Errorf("%q, %w", diagnostic, errs.ErrResponseUnknown)
}

// getRespBody returns a json response body decoded as a string-keyed map, or an error when decoding fails.
func getRespBody(body []byte) (map[string]any, error) {
	var jsonBody map[string]any

	err := json.Unmarshal(body, &jsonBody)
	if err != nil {
		return nil, fmt.Errorf("%q: %w, %w", bodyDecodeContext, errs.ErrResponseDecode, err)
	}

	return jsonBody, nil
}
