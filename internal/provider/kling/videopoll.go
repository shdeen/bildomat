package kling

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/provider"
)

// The video completion state and query route. Pending and failed states are
// shared with image jobs, but the two APIs spell completion differently.
//   - videoStatusSucceeded: the video task completed
//   - tasksQueryPath: the task query path, before the query string
//   - fieldTaskIDs: the query parameter naming the tasks to report
const (
	videoStatusSucceeded = "succeeded"
	tasksQueryPath       = "/tasks?"
	fieldTaskIDs         = "task_ids"
)

// videoJobPoll checks one video-generation task and retains its completed
// result URL.
type videoJobPoll struct {
	apiBase           string
	credential        httpapi.AuthCredential
	taskID            string
	providerModelName string
	resultURL         string
	record            *metadata.Record
}

// Poll retrieves and classifies one video task response.
func (videoPoll *videoJobPoll) Poll(ctx context.Context) (taskComplete bool, pollErr error) {
	queryValues := url.Values{fieldTaskIDs: []string{videoPoll.taskID}}
	endpoint := videoPoll.apiBase + tasksQueryPath + queryValues.Encode()

	statusCode, responseBody, err := httpapi.GetAuth(ctx, endpoint, videoPoll.credential, metadata.Asynchronous, videoPoll.record)
	if err := provider.PollResponseError(videoPoll.providerModelName, statusCode, responseBody, err); err != nil {
		return false, err
	}

	return videoPoll.classifyResponse(responseBody)
}

// videoTask carries the video API's distinct status and output list.
type videoTask struct {
	Status  any             `json:"status"`
	Message any             `json:"message"`
	Outputs json.RawMessage `json:"outputs"`
}

// classifyResponse preserves the video API's first-task selection and statuses.
func (videoPoll *videoJobPoll) classifyResponse(responseBody []byte) (bool, error) {
	response, err := decodeEnvelope(responseBody, videoPoll.providerModelName+": "+VideoTaskResponseContext)
	if err != nil {
		return false, err
	}

	if err := envelopeFailure(response, videoPoll.providerModelName); err != nil {
		return false, err
	}

	var tasks []json.RawMessage
	if err := json.Unmarshal(response.Data, &tasks); err != nil || len(tasks) == 0 {
		return false, fmt.Errorf("%q, %w", videoPoll.taskID, errs.ErrResponseTaskDataMissing)
	}

	var task *videoTask
	if err := json.Unmarshal(tasks[0], &task); err != nil || task == nil {
		return false, fmt.Errorf("%q, %w", videoPoll.taskID, errs.ErrResponseTaskRecordInvalid)
	}

	status, valid := task.Status.(string)
	if !valid {
		return false, unknownStatus(videoPoll.taskID, task.Status)
	}

	switch status {
	case taskStatusSubmitted, taskStatusProcessing:
		return false, nil
	case videoStatusSucceeded:
		if err := videoPoll.retainResultURL(task.Outputs); err != nil {
			return false, err
		}

		return true, nil
	case taskStatusFailed:
		message, _ := task.Message.(string) //nolint:revive // An absent or non-string optional message intentionally becomes empty.

		return false, taskFailure(videoPoll.taskID, status, message)
	default:
		return false, unknownStatus(videoPoll.taskID, status)
	}
}

// retainResultURL validates the video API's first output before retaining it.
func (videoPoll *videoJobPoll) retainResultURL(encodedOutputs json.RawMessage) error {
	var outputs []json.RawMessage
	if err := json.Unmarshal(encodedOutputs, &outputs); err != nil || len(outputs) == 0 {
		return fmt.Errorf("%q, %w", fmt.Sprintf(ResultURLMissing, videoPoll.providerModelName), errs.ErrResponseNoData)
	}

	var output *struct {
		URL any `json:"url"`
	}
	if err := json.Unmarshal(outputs[0], &output); err != nil || output == nil {
		return fmt.Errorf("%q, %w", videoPoll.taskID, errs.ErrResponseOutputInvalid)
	}

	resultURL, valid := output.URL.(string)
	if !valid || resultURL == "" {
		return fmt.Errorf("%q, %w", fmt.Sprintf(ResultURLMissing, videoPoll.providerModelName), errs.ErrResponseNoData)
	}

	videoPoll.resultURL = resultURL

	return nil
}
