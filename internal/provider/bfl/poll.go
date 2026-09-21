package bfl

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/provider"
)

// pollResp contains the current state of a generation job.
type pollResp struct {
	// ID is the generation job identifier.
	ID string `json:"id"`
	// Status is the current lifecycle status.
	Status string `json:"status"`
	// Result contains the completed media location.
	Result *pollResult `json:"result"`
	// Details contains provider-supplied failure information.
	Details json.RawMessage `json:"details"`
}

// pollResult contains the completed media location.
type pollResult struct {
	// Sample is the signed media download URL.
	Sample string `json:"sample"`
}

// pollProbe holds the values needed to inspect a generation job.
//   - cred: the credential sent with each status request
//   - adapterAPI: the status values used to classify the job
//   - id: the generation job identifier
//   - pollURL: the endpoint used to inspect the job
//   - record: optional retention of the polling requests and responses
//   - completed: validated response available after polling succeeds
type pollProbe struct {
	cred       httpapi.AuthCredential
	adapterAPI *catalog.AdapterAPI
	id         string
	pollURL    string
	record     *metadata.Record
	completed  pollResp
}

// Poll retains the validated completed response and reports completion.
func (probe *pollProbe) Poll(ctx context.Context) (complete bool, err error) {
	status, body, err := httpapi.GetAuth(ctx, probe.pollURL, probe.cred, metadata.Asynchronous, probe.record)
	if err := provider.PollResponseError(probe.id, status, body, err); err != nil {
		return false, err
	}

	var response pollResp
	if uErr := json.Unmarshal(body, &response); uErr != nil {
		return false, fmt.Errorf("%q, %w, %w", probe.id+": "+provider.PollResponseContext, errs.ErrResponseDecode, uErr)
	}

	complete, err = classifyPollStatus(probe.adapterAPI, probe.id, response)
	if err == nil && complete {
		probe.completed = response
	}

	return complete, err
}

// classifyPollStatus accepts completion only when the response contains a sample URL.
func classifyPollStatus(adapterAPI *catalog.AdapterAPI, jobID string, response pollResp) (complete bool, err error) {
	switch {
	case slices.Contains(adapterAPI.PendingStatusText, response.Status):
		return false, nil
	case response.Status == adapterAPI.ReadyStatusText:
		if response.Result == nil || response.Result.Sample == "" {
			return false, fmt.Errorf("%q, %w", jobID, errs.ErrResponseNoSample)
		}

		return true, nil
	case slices.Contains(adapterAPI.FailedStatusText, response.Status):
		diagnostic := fmt.Sprintf("%s (%s)", jobID, response.Status)
		if d := errorDetails(response.Details); d != "" {
			diagnostic += ": " + d
		}

		return false, &errs.ProviderError{Message: diagnostic, Cause: errs.ErrResponseGen}
	default:
		obs := response.Status
		if obs == "" {
			obs = provider.MissingResponseValue
		}

		diagnostic := fmt.Sprintf("%s (%s)", jobID, strconv.Quote(obs))

		return false, fmt.Errorf("%q, %w", diagnostic, errs.ErrResponseUnknown)
	}
}

// errorDetails returns encoded failure details as text, or an empty string for absent and null values.
func errorDetails(detailsJSON json.RawMessage) string {
	if len(detailsJSON) == 0 || string(detailsJSON) == "null" { //nolint:goconst // JSON null is protocol syntax, not a domain default or shared field.
		return ""
	}

	var s string
	if json.Unmarshal(detailsJSON, &s) == nil {
		return s
	}

	return string(detailsJSON)
}
