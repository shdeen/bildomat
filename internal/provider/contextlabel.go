package provider

// File: internal/provider/contextlabel.go
// The response labels that more than one provider adapter appends, after a
// colon and a space, to the model or job named in a failed-decode context.
//   - CreationResponseContext: the response to a request that creates a job or task
//   - PollResponseContext: the response to a poll of a running job or task
const (
	CreationResponseContext = "creation response"
	PollResponseContext     = "poll response"
)
