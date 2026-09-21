package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// submitVideo starts a video job, waits for completion, and returns the downloaded artifact.
// It may resize input images and create a temporary artifact file.
func submitVideo(ctx context.Context, api *catalog.VideoAPI, apiKey string, run *generation.Generation, stringParams ...params.FlagType) ([]artifact.Media, error) {
	provModelLabel := run.Label()

	preparedGeneration, err := prepareVideoInputs(ctx, api, run)
	run.Preparation = preparedGeneration

	if err != nil {
		return nil, fmt.Errorf("%q, %w", provModelLabel, err)
	}

	credential := httpapi.Bearer(apiKey)

	run.Record.Prepared(preparedGeneration.InputMedia)

	jobID, err := startVideoJob(ctx, credential, api, provModelLabel, &run.Model, run.Prompt, run.Params, preparedGeneration.InputMedia, run.Record, stringParams...)
	if err != nil {
		return nil, err
	}

	body, err := pollVideoJob(ctx, credential, api, provModelLabel, jobID, run.Record)
	if err != nil {
		return nil, err
	}

	run.Record.ProviderFinished(nil)

	generatedMedia, err := fetchVideo(ctx, credential, api, provModelLabel, jobID, body, run.Record)
	if err != nil {
		return nil, err
	}

	return []artifact.Media{generatedMedia}, nil
}

// prepareVideoInputs retrieves each retained multipart reference and applies model policy.
// Completed media and adjustments survive a later retrieval or transformation failure.
// The caller's request remains unchanged.
func prepareVideoInputs(ctx context.Context, api *catalog.VideoAPI, run *generation.Generation) (generation.Preparation, error) {
	preparedGeneration := run.Clone()
	if api.InputMediaPayloadType != catalog.InputMediaPayloadForm {
		return preparedGeneration, nil
	}

	for mediaIndex := range preparedGeneration.InputMedia {
		downloaded, err := httpapi.DownloadInputMedia(ctx, preparedGeneration.InputMedia[mediaIndex:mediaIndex+1])
		if err != nil {
			return preparedGeneration, err
		}

		preparedGeneration.InputMedia[mediaIndex] = downloaded[0]
		if api.InputMediaMustResize {
			changes, conformErr := generation.ConformInputMedia(&run.Model, preparedGeneration.Params, preparedGeneration.InputMedia[mediaIndex:mediaIndex+1])

			preparedGeneration.Changes = append(preparedGeneration.Changes, changes...)
			if conformErr != nil {
				return preparedGeneration, conformErr
			}
		}
	}

	return preparedGeneration, nil
}

// startVideoJob sends a video job request and returns the response's job identifier.
func startVideoJob(ctx context.Context, credential httpapi.AuthCredential, api *catalog.VideoAPI, provModelLabel string, model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input, record *metadata.Record, stringParams ...params.FlagType) (string, error) {
	url := api.AsyncJobsURL + api.URLStartPath

	var (
		status int
		body   []byte
		err    error
	)

	if len(mediaInputs) > 0 && api.InputMediaPayloadType == catalog.InputMediaPayloadForm {
		ct, form, fErr := multipartBody(videoJobFields(model, prompt, parameterValues, stringParams...), api.InputMediaProvParam, mediaInputs)
		if fErr != nil {
			return "", fmt.Errorf("%q, %w", provModelLabel, fErr)
		}

		status, body, err = httpapi.SendBody(ctx, url, credential, ct, form, record)
	} else {
		requestBody, requestErr := getVideoJobBody(api, model, prompt, parameterValues, mediaInputs, stringParams...)
		if requestErr != nil {
			return "", requestErr
		}

		status, body, err = httpapi.PostJSON(ctx, url, credential, requestBody, record)
	}

	return getJobID(provModelLabel, api.JobIDField, status, body, err)
}

// getVideoJobBody returns the JSON fields for a video job request.
func getVideoJobBody(api *catalog.VideoAPI, model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input, stringParams ...params.FlagType) (map[string]any, error) {
	body := map[string]any{
		fieldModel: model.ID,
	}

	if !model.PromptIgnored {
		body[fieldPrompt] = prompt
	}

	maps.Copy(body, WireParamValues(model, parameterValues, stringParams...))

	if err := addVideoInputMedia(body, api, mediaInputs); err != nil {
		return nil, err
	}

	return body, nil
}

// addVideoInputMedia writes ordinary references and configured opening or closing frames.
func addVideoInputMedia(body map[string]any, api *catalog.VideoAPI, mediaInputs []media.Input) error {
	ordinaryInputs, frameInputs := splitFrameInputs(mediaInputs)

	if err := addInputMedia(body, api.InputMediaStyle, api.InputMediaProvParam, api.InputMediaListProvParam, ordinaryInputs); err != nil {
		return err
	}

	if len(frameInputs) == 0 {
		return nil
	}

	if _, complete := api.FrameFields(); !complete {
		return &errs.MediaError{Problem: FrameMediaUndescribed, Cause: errs.ErrInputMediaTime}
	}

	frameValues, err := frameMediaValues(api, frameInputs)
	if err != nil {
		return err
	}

	body[api.FrameMediaProvParam] = frameValues

	return nil
}

// splitFrameInputs takes media inputs and returns the inputs carrying no frame
// prefix and the inputs carrying one, each in input order.
func splitFrameInputs(mediaInputs []media.Input) (ordinaryInputs, frameInputs []media.Input) {
	ordinaryInputs = make([]media.Input, 0, len(mediaInputs))
	frameInputs = make([]media.Input, 0, len(mediaInputs))

	for _, mediaInput := range mediaInputs {
		if mediaInput.HasFrame() {
			frameInputs = append(frameInputs, mediaInput)
		} else {
			ordinaryInputs = append(ordinaryInputs, mediaInput)
		}
	}

	return ordinaryInputs, frameInputs
}

// frameMediaValues takes the video API and the frame inputs and returns one
// request object per input, carrying the media reference under its type and
// the frame role under the API's role field. It fails on a frame on a video
// input, on an anchor the frame resolution left unresolved, and on a role
// claimed by two inputs.
func frameMediaValues(api *catalog.VideoAPI, frameInputs []media.Input) ([]any, error) {
	frameValues := make([]any, 0, len(frameInputs))
	usedRoles := map[string]bool{}

	// Frame resolution ran during parameter adjustment, so every frame input
	// carries an anchor keyword here; anything else is a repository defect.
	for _, mediaInput := range frameInputs {
		if mediaInput.Kind() == media.Video {
			return nil, &errs.MediaError{Problem: fmt.Sprintf(generation.FrameOnVideo, mediaInput.Source()), Cause: errs.ErrInputMediaTime}
		}

		var frameRole string

		switch mediaInput.FrameAnchor {
		case media.FrameFirst:
			frameRole = api.FirstFrameProvValue
		case media.FrameLast:
			frameRole = api.LastFrameProvValue
		default:
			return nil, &errs.MediaError{Problem: fmt.Sprintf(FrameTimeUnresolved, mediaInput.Source()), Cause: errs.ErrInputMediaTime}
		}

		if usedRoles[frameRole] {
			return nil, &errs.MediaError{Problem: fmt.Sprintf(FrameRoleDuplicate, frameRole), Cause: errs.ErrInputMediaTime}
		}

		usedRoles[frameRole] = true

		mediaType := mediaURLType(&mediaInput)
		frameValues = append(frameValues, map[string]any{
			fieldType: mediaType, mediaType: inputMediaURLNested(&mediaInput), api.FrameRoleProvParam: frameRole,
		})
	}

	return frameValues, nil
}

// videoJobFields returns the text fields for a multipart video job request.
func videoJobFields(model *catalog.Model, prompt string, parameterValues params.Values, stringParams ...params.FlagType) map[string]string {
	fields := map[string]string{
		fieldModel: model.ID,
	}
	if !model.PromptIgnored {
		fields[fieldPrompt] = prompt
	}

	for wireKey, paramVal := range WireParamValues(model, parameterValues, stringParams...) {
		fields[wireKey] = params.FormatValue(paramVal)
	}

	return fields
}

// getJobID returns the job identifier from a successful response.
func getJobID(provModelLabel, idField string, status int, body []byte, errReceived error) (string, error) {
	if errReceived != nil {
		return "", fmt.Errorf("%q, %w", fmt.Sprintf(StartContextForm, provModelLabel), errReceived)
	}

	if status/100 != 2 {
		return "", APIErr(provModelLabel, status, body)
	}

	job, err := getRespBody(body)
	if err != nil {
		return "", fmt.Errorf("%q: %w", provModelLabel, err)
	}

	jobID, ok := job[idField].(string)
	if !ok || jobID == "" {
		return "", fmt.Errorf("%q, %w", provModelLabel, errs.ErrResponseNoJobID)
	}

	return jobID, nil
}

// videoJobProbe holds the values needed to inspect a video job.
//   - credential: the credential sent with each status request
//   - api: the video API settings used to inspect the job
//   - jobID: the video job identifier
//   - record: optional retention of every status response
type videoJobProbe struct {
	credential httpapi.AuthCredential
	api        *catalog.VideoAPI
	jobID      string
	record     *metadata.Record
	completed  []byte
}

// Poll checks the video job and returns its response body when generation is complete.
func (probe *videoJobProbe) Poll(ctx context.Context) (isDone bool, pollErr error) {
	status, body, err := httpapi.GetAuth(ctx, probe.api.AsyncJobsURL+"/"+probe.jobID, probe.credential, metadata.Asynchronous, probe.record)
	if err := PollResponseError(probe.jobID, status, body, err); err != nil {
		return false, err
	}

	_, jobDone, cErr := classifyResponse(probe.jobID, body, probe.api.ProgressStatusText, probe.api.CompletedStatusText, probe.api.FailedStatusText)
	if cErr != nil || !jobDone {
		return false, cErr
	}

	if probe.api.URLPathRequired && walkURL(body, probe.api.URLPathSeq) == "" {
		return false, fmt.Errorf("%q, %w", probe.jobID, errs.ErrResponseNoData)
	}

	probe.completed = body

	return true, nil
}

// pollVideoJob waits for a video job to complete and returns its final response body.
func pollVideoJob(ctx context.Context, credential httpapi.AuthCredential, api *catalog.VideoAPI, provModelLabel, jobID string, record *metadata.Record) ([]byte, error) {
	pollPace := api.PollInterval.Duration()
	pollBudget := api.PollTimeout.Duration()

	probe := &videoJobProbe{credential: credential, api: api, jobID: jobID, record: record}

	err := httpapi.Poll(ctx, pollPace, pollBudget, probe)
	if err != nil {
		return nil, &errs.PollError{Model: provModelLabel, Resource: jobID, Cause: err}
	}

	return probe.completed, nil
}

// fetchVideo downloads a completed video's media and returns a file-backed artifact.
func fetchVideo(ctx context.Context, credential httpapi.AuthCredential, api *catalog.VideoAPI, provModelLabel, jobID string, body []byte, record *metadata.Record) (artifact.Media, error) {
	downloadURL := ""
	if len(api.URLPathSeq) > 0 {
		downloadURL = walkURL(body, api.URLPathSeq)
		if downloadURL == "" && api.URLPathRequired {
			diagnostic := fmt.Sprintf(DownloadURLMissing, provModelLabel, jobID)

			return artifact.Media{}, fmt.Errorf("%q, %w", diagnostic, errs.ErrResponseNoData)
		}
	}

	var downloadCred httpapi.AuthCredential

	if downloadURL == "" {
		downloadURL = api.AsyncJobsURL + "/" + jobID + api.URLContentPath
		if api.AuthDownload {
			downloadCred = credential
		}
	} else if api.AuthDownload {
		downloadCred = httpapi.CredentialForURL(downloadURL, api.AsyncJobsURL, credential)
	}

	generatedMedia, err := httpapi.Fetch(ctx, downloadURL, downloadCred, api.FallbackExt, record)
	if err != nil {
		// The fetch error carries its own endpoint context; this boundary adds
		// the provider-model identity, which only the caller knows.
		return artifact.Media{}, fmt.Errorf("%q, %w", provModelLabel, err)
	}

	return generatedMedia, nil
}

// walkURL returns the string reached by traversing a polling response through path segments.
func walkURL(pollBody []byte, segments []string) string {
	var cur any
	if json.Unmarshal(pollBody, &cur) != nil {
		return ""
	}

	for _, segment := range segments {
		switch node := cur.(type) {
		case map[string]any:
			if isDigits(segment) {
				return ""
			}

			cur = node[segment]
		case []any:
			if node == nil {
				return ""
			}

			i, ok := segmentIndex(segment, len(node))
			if !ok {
				return ""
			}

			cur = node[i]
		default:
			return ""
		}
	}

	s, ok := cur.(string)
	if !ok {
		return ""
	}

	return s
}

// segmentIndex returns the list index represented by a path segment when it is within bounds.
func segmentIndex(segment string, segmentCount int) (int, bool) {
	if !isDigits(segment) {
		return 0, false
	}

	i, err := strconv.Atoi(segment)
	if err != nil || i >= segmentCount {
		return 0, false
	}

	return i, true
}

// isDigits reports whether text contains one or more ASCII digits.
func isDigits(segment string) bool {
	if segment == "" {
		return false
	}

	for _, r := range segment {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
