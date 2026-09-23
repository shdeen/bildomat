package kling

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/provider"
)

// The task states shared by image and video jobs, and the image completion state.
//   - taskStatusSubmitted: the task is queued
//   - taskStatusProcessing: the task is running
//   - taskStatusFailed: the task failed
//   - imageStatusSucceeded: the image task completed, as the image API spells it
const (
	taskStatusSubmitted  = "submitted"
	taskStatusProcessing = "processing"
	taskStatusFailed     = "failed"
	imageStatusSucceeded = "succeed"
)

// imageJobPoll tracks one Kling image task.
//   - apiBase: the provider API base URL
//   - credential: authentication for status requests
//   - taskID: the provider task identifier
//   - providerModelName: the provider/model label used in errors
//   - queryPath: the task status path
//   - resultURLs: completed image URLs in provider index order
//   - record: optional request and response retention
type imageJobPoll struct {
	apiBase           string
	credential        httpapi.AuthCredential
	taskID            string
	providerModelName string
	queryPath         string
	resultURLs        []string
	record            *metadata.Record
}

// Poll retrieves and classifies one image task response.
func (imagePoll *imageJobPoll) Poll(ctx context.Context) (taskComplete bool, pollErr error) {
	endpoint := imagePoll.apiBase + imagePoll.queryPath

	statusCode, responseBody, err := httpapi.GetAuth(ctx, endpoint, imagePoll.credential, metadata.Asynchronous, imagePoll.record)
	if err := provider.PollResponseError(imagePoll.providerModelName, statusCode, responseBody, err); err != nil {
		return false, err
	}

	return imagePoll.classifyResponse(responseBody)
}

// imageTask carries the image API's distinct status and result fields.
type imageTask struct {
	// Status preserves the reported task state and its JSON type.
	Status any `json:"task_status"`
	// Message preserves the optional task failure message.
	Message any `json:"task_status_msg"`
	// Result contains the encoded image results.
	Result json.RawMessage `json:"task_result"`
}

// imageResponse preserves the declared index until its exact value is validated.
type imageResponse struct {
	// Index identifies the image's position in the provider result.
	Index json.RawMessage `json:"index"`
	// URL contains the image download location.
	URL any `json:"url"`
}

// imageResult contains the validated facts needed to order and download an image.
type imageResult struct {
	// Index identifies the image's position in the provider result.
	Index int
	// URL contains the image download location.
	URL string
}

// classifyResponse reports task completion and stores ordered output URLs in the poller. It
// validates the fields required by the reported task state.
func (imagePoll *imageJobPoll) classifyResponse(responseBody []byte) (bool, error) {
	response, err := decodeEnvelope(responseBody, imagePoll.providerModelName+": "+ImageTaskResponseContext)
	if err != nil {
		return false, err
	}

	if err := envelopeFailure(response, imagePoll.providerModelName); err != nil {
		return false, err
	}

	var task *imageTask
	if err := json.Unmarshal(response.Data, &task); err != nil || task == nil {
		return false, fmt.Errorf("%q, %w", imagePoll.taskID, errs.ErrResponseTaskDataMissing)
	}

	status, valid := task.Status.(string)
	if !valid {
		return false, unknownStatus(imagePoll.taskID, task.Status)
	}

	switch status {
	case taskStatusSubmitted, taskStatusProcessing:
		return false, nil
	case taskStatusFailed:
		message, _ := task.Message.(string) //nolint:revive // An absent or non-string optional message intentionally becomes empty.

		return false, taskFailure(imagePoll.taskID, status, message)
	case imageStatusSucceeded:
		if err := imagePoll.retainResultURLs(task.Result); err != nil {
			return false, err
		}

		return true, nil
	default:
		return false, unknownStatus(imagePoll.taskID, status)
	}
}

// retainResultURLs validates image URLs and unique indexes, then stores URLs in index order in the
// poller. Invalid results leave its retained URLs unchanged.
func (imagePoll *imageJobPoll) retainResultURLs(encodedResult json.RawMessage) error {
	var result struct {
		Images []json.RawMessage `json:"images"`
	}
	if err := json.Unmarshal(encodedResult, &result); err != nil || len(result.Images) == 0 {
		return fmt.Errorf("%q, %w", fmt.Sprintf(ResultURLMissing, imagePoll.providerModelName), errs.ErrResponseNoData)
	}

	images := make([]imageResult, 0, len(result.Images))

	usedIndexes := make(map[int]bool, len(result.Images))
	for _, encodedImage := range result.Images {
		var image *imageResponse
		if err := json.Unmarshal(encodedImage, &image); err != nil || image == nil {
			return fmt.Errorf("%q, %w", imagePoll.taskID, errs.ErrResponseResultInvalid)
		}

		index, valid := exactInteger(image.Index)
		if !valid {
			return fmt.Errorf("%q, %w", imagePoll.taskID, errs.ErrResponseResultIndexInvalid)
		}

		resultURL, valid := image.URL.(string)
		if !valid || resultURL == "" {
			return fmt.Errorf("%q, %w", fmt.Sprintf(ResultURLMissing, imagePoll.providerModelName), errs.ErrResponseNoData)
		}

		if usedIndexes[index] {
			return fmt.Errorf("%q, %w", imagePoll.taskID, errs.ErrResponseResultIndexDuplicate)
		}

		usedIndexes[index] = true
		images = append(images, imageResult{Index: index, URL: resultURL})
	}

	slices.SortFunc(images, compareImageIndexes)

	resultURLs := make([]string, len(images))
	for imageIndex := range images {
		resultURLs[imageIndex] = images[imageIndex].URL
	}

	imagePoll.resultURLs = resultURLs

	return nil
}

// compareImageIndexes compares images by their provider-assigned index.
func compareImageIndexes(firstImage, secondImage imageResult) int {
	return cmp.Compare(firstImage.Index, secondImage.Index)
}
