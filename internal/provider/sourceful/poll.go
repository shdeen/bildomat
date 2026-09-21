package sourceful

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/provider"
)

// jobPoll retrieves and classifies one Sourceful job response.
type jobPoll struct {
	adapterAPI        *catalog.AdapterAPI
	apiCredential     httpapi.AuthCredential
	jobID             string
	providerModelName string
	resultMIME        string
	resultURL         string
	record            *metadata.Record
}

type jobData struct {
	Job    json.RawMessage `json:"job"`
	Result json.RawMessage `json:"result"`
}

type jobRecord struct {
	Status           any             `json:"status"`
	LastErrorMessage any             `json:"lastErrorMessage"`
	Result           json.RawMessage `json:"result"`
}

type resultOutput struct {
	URL  any `json:"url"`
	MIME any `json:"mimeType"`
}

// Poll retains the completed output only after the response passes validation.
func (sourcefulPoll *jobPoll) Poll(ctx context.Context) (jobDone bool, err error) {
	pollEndpoint := strings.TrimRight(sourcefulPoll.adapterAPI.APIBase, "/") + pollRoute + url.PathEscape(sourcefulPoll.jobID)

	status, body, err := httpapi.GetAuth(ctx, pollEndpoint, sourcefulPoll.apiCredential, metadata.Asynchronous, sourcefulPoll.record)
	if err := provider.PollResponseError(sourcefulPoll.providerModelName, status, body, err); err != nil {
		return false, err
	}

	return sourcefulPoll.classifyJobResponse(body)
}

// classifyJobResponse preserves status and output presence distinctions.
func (sourcefulPoll *jobPoll) classifyJobResponse(responseBody []byte) (bool, error) {
	var response responseEnvelope
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return false, fmt.Errorf("%q, %w, %w", sourcefulPoll.providerModelName, errs.ErrResponseDecode, err)
	}

	var (
		data jobData
		job  jobRecord
	)
	if json.Unmarshal(response.Data, &data) != nil || json.Unmarshal(data.Job, &job) != nil {
		return false, fmt.Errorf("%q, %w", sourcefulPoll.jobID, errs.ErrResponseJobStatusMissing)
	}

	jobStatus, statusPresent := job.Status.(string)
	if !statusPresent {
		return false, fmt.Errorf("%q, %w", sourcefulPoll.jobID, errs.ErrResponseJobStatusMissing)
	}

	switch {
	case slices.Contains(sourcefulPoll.adapterAPI.PendingStatusText, jobStatus):
		return false, nil
	case jobStatus == sourcefulPoll.adapterAPI.ReadyStatusText:
		return sourcefulPoll.completedResult(job.Result, data.Result)
	case slices.Contains(sourcefulPoll.adapterAPI.FailedStatusText, jobStatus):
		failureMessage, _ := job.LastErrorMessage.(string) //nolint:revive // An absent or non-string optional message intentionally becomes empty.

		failureContext := fmt.Sprintf("%s (%s)", sourcefulPoll.jobID, jobStatus)
		if failureMessage != "" {
			failureContext += ": " + failureMessage
		}

		return false, &errs.ProviderError{Message: failureContext, Cause: errs.ErrResponseGen}
	default:
		return false, fmt.Errorf("%q, %w", fmt.Sprintf(JobStatusUnknownForm, sourcefulPoll.jobID, jobStatus), errs.ErrResponseUnknown)
	}
}

// completedResult resolves each optional field independently. A present string,
// including an empty string, wins over the alternate response location.
func (sourcefulPoll *jobPoll) completedResult(primary, alternate json.RawMessage) (bool, error) {
	primaryOutput := decodeResultOutput(primary)
	alternateOutput := decodeResultOutput(alternate)

	resultURL, urlPresent := primaryOutput.URL.(string)
	if !urlPresent {
		resultURL, urlPresent = alternateOutput.URL.(string)
	}

	if !urlPresent || resultURL == "" {
		return false, fmt.Errorf("%q, %w", fmt.Sprintf(ResultURLMissing, sourcefulPoll.providerModelName), errs.ErrResponseNoData)
	}

	resultMIME, mimePresent := primaryOutput.MIME.(string)
	if !mimePresent {
		resultMIME, _ = alternateOutput.MIME.(string) //nolint:revive // An absent or non-string optional MIME value intentionally becomes empty.
	}

	sourcefulPoll.resultURL, sourcefulPoll.resultMIME = resultURL, resultMIME

	return true, nil
}

// decodeResultOutput treats an absent or incompatible optional output location
// as unavailable, allowing the other published location to supply its fields.
func decodeResultOutput(encodedResult json.RawMessage) resultOutput {
	var result struct {
		Output json.RawMessage `json:"output"`
	}
	if json.Unmarshal(encodedResult, &result) != nil {
		return resultOutput{}
	}

	var output resultOutput
	if json.Unmarshal(result.Output, &output) != nil {
		return resultOutput{}
	}

	return output
}
