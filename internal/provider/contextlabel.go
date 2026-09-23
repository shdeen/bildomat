package provider

// Response labels provide context for failed decoding.
//   - CreationResponseContext: the response that creates a job or task
//   - PollResponseContext: a status response for a running job or task
const (
	CreationResponseContext = "creation response"
	PollResponseContext     = "poll response"
)
