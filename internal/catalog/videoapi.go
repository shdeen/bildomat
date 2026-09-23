package catalog

// VideoAPI describes the endpoints, wire format, and polling settings for video generation.
//   - AsyncJobsURL: the base endpoint for video jobs
//   - URLStartPath: the path used to start a video job
//   - JobIDField: the response field containing the job identifier
//   - ProgressStatusText: the statuses that indicate a running job
//   - CompletedStatusText: the status that indicates a completed job
//   - FailedStatusText: the statuses that indicate a failed job
//   - URLPathSeq: the response path leading to the video download URL
//   - URLPathRequired: whether a completed response must contain a download URL
//   - URLContentPath: the path used to download video content when the response has no URL
//   - AuthDownload: whether same-origin downloads receive the provider credential
//   - InputMediaPayloadType: the request encoding for input media
//   - InputMediaListProvParam: the array field for multiple single-or-array references
//   - InputMediaProvParam: the request field for ordinary input media
//   - InputMediaStyle: the JSON layout for ordinary input media
//   - InputMediaMustResize: whether local input images must match the requested dimensions
//   - FrameMediaProvParam: the request field for timed frame images
//   - FrameRoleProvParam: the field naming a frame image's role
//   - FirstFrameProvValue: the role value for an opening frame
//   - LastFrameProvValue: the role value for a closing frame
//   - FallbackExt: the file extension used when the response does not identify its media format
//   - PollInterval: the number of seconds between job status requests
//   - PollTimeout: the maximum number of seconds allowed for job polling
type VideoAPI struct {
	AsyncJobsURL            string          `json:"asyncJobsURL"`
	URLStartPath            string          `json:"urlStartPath"`
	JobIDField              string          `json:"jobIDField"`
	ProgressStatusText      []string        `json:"progressStatusText"`
	CompletedStatusText     string          `json:"completedStatusText"`
	FailedStatusText        []string        `json:"failedStatusText"`
	URLPathSeq              []string        `json:"urlPathSeq"`
	URLPathRequired         bool            `json:"urlPathRequired"`
	URLContentPath          string          `json:"urlContentPath"`
	AuthDownload            bool            `json:"authDownload"`
	InputMediaPayloadType   string          `json:"inputMediaPayloadType"`
	InputMediaProvParam     string          `json:"inputMediaProvParam"`
	InputMediaListProvParam string          `json:"inputMediaListProvParam,omitempty"`
	InputMediaStyle         InputMediaStyle `json:"inputMediaStyle"`
	InputMediaMustResize    bool            `json:"inputMediaMustResize"`
	FrameMediaProvParam     string          `json:"frameMediaProvParam"`
	FrameRoleProvParam      string          `json:"frameRoleProvParam"`
	FirstFrameProvValue     string          `json:"firstFrameProvValue"`
	LastFrameProvValue      string          `json:"lastFrameProvValue"`
	FallbackExt             string          `json:"fallbackExt"`
	PollInterval            PollSeconds     `json:"pollInterval"`
	PollTimeout             PollSeconds     `json:"pollTimeout"`
}

// FrameFields reports whether any frame field is present and whether all frame fields are present.
func (api *VideoAPI) FrameFields() (present, complete bool) {
	fields := []string{api.FrameMediaProvParam, api.FrameRoleProvParam, api.FirstFrameProvValue, api.LastFrameProvValue}
	count := 0

	for _, field := range fields {
		if field != "" {
			count++
		}
	}

	return count != 0, count == len(fields)
}
