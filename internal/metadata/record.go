// Package metadata stores experimental generation records. The format and reuse mechanism are
// provisional and subject to change.
package metadata

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// schemaVersion identifies the provisional record format; it is not a promise of compatibility with
// the eventual permanent format.
const schemaVersion = 1

// Record states, formats, and diagnostic paths.
//   - statusCompleted, statusCanceled, statusFailed: the local operation outcomes
//   - recordFormat: the saved document extension
//   - pendingFileBody: the response value before a binary file has a saved path
//   - requestPath, responsePath: the record locations used in persistence errors
const (
	statusCompleted = "completed"
	statusCanceled  = "canceled"
	statusFailed    = "failed"
	recordFormat    = "json"
	pendingFileBody = `{"file":null}`
	requestPath     = "request/provider-requests"
	responsePath    = "provider-responses"
)

// Response types describe the request's role, independently of HTTP timing.
//   - Synchronous: a submission or direct download response
//   - Asynchronous: a response to polling an existing operation
const (
	Synchronous  = "synchronous"
	Asynchronous = "asynchronous"
)

// Record contains the retained facts of one generation. Its unexported state owns binary copies
// independently of provider artifact cleanup.
//   - Schema: the record format version
//   - ID: the generated identifier for this run
//   - Version: the Bildomat version that produced the record
//   - Provider, Model: the selected catalog identifiers
//   - Started, Finished: the UTC execution times
//   - ElapsedMS: the elapsed execution time in milliseconds
//   - Status: the final status, including local persistence failures
//   - ProviderStatus: the provider operation status before local persistence
//   - Request: the supplied options, prepared inputs, and submitted requests
//   - Artifacts: the successfully saved generated files
//   - Responses: the provider responses in capture order
//   - Returns: the provider data retained for later requests
//   - Errors: the generation failure messages
//   - PersistenceErrors: the record capture and persistence failure messages
//
//nolint:tagliatelle // The provisional record uses the owner-approved hyphenated JSON keys.
type Record struct {
	Schema            int           `json:"schema-version"`
	ID                string        `json:"generation-id"`
	Version           string        `json:"bildomat-version"`
	Provider          string        `json:"provider"`
	Model             string        `json:"model"`
	Started           time.Time     `json:"started"`
	Finished          time.Time     `json:"finished"`
	ElapsedMS         int64         `json:"elapsed-ms"`
	Status            string        `json:"status"`
	ProviderStatus    string        `json:"provider-status,omitempty"`
	Request           Request       `json:"request"`
	Artifacts         []File        `json:"artifacts"`
	Responses         []Response    `json:"provider-responses"`
	Returns           []ReturnValue `json:"return-values,omitempty"`
	Errors            []string      `json:"errors,omitempty"`
	PersistenceErrors []string      `json:"persistence-errors,omitempty"`
	// binaries retains independent copies of captured media.
	binaries binaryStore
	// requestFields declares binary locations in submitted payloads.
	requestFields []BinaryField
	// responseFields declares binary locations in provider responses.
	responseFields []BinaryField
	// faults retains original capture and persistence causes.
	faults []error
}

// Request distinguishes the supplied options, adjusted options, prepared inputs, and exact
// submitted HTTP payloads of one generation.
//   - Prompt: the submitted prompt
//   - Supplied, Adjusted: the options before and after conformance
//   - Adjustments: the recorded parameter changes
//   - Sources: the supplied input paths or URLs
//   - Inputs: the media prepared for submission
//   - Calls: the provider requests in submission order
//
//nolint:tagliatelle // Keep the provisional record's hyphenated JSON keys.
type Request struct {
	Prompt      string          `json:"prompt"`
	Supplied    json.RawMessage `json:"supplied-parameters,omitempty"`
	Adjusted    json.RawMessage `json:"adjusted-parameters,omitempty"`
	Adjustments json.RawMessage `json:"adjustments,omitempty"`
	Sources     []string        `json:"input-sources,omitempty"`
	Inputs      []Input         `json:"prepared-inputs,omitempty"`
	Calls       []Call          `json:"provider-requests"`
}

// Input describes a prepared input; its data is retained through the submitted payload, without
// embedding the bytes a second time.
//   - Source: the input path or URL
//   - MIME: the prepared content type
//   - Time: the optional frame time in seconds
//   - Frame: the optional first or last frame selection
//
//nolint:tagliatelle // Keep the provisional record's hyphenated JSON keys.
type Input struct {
	Source string   `json:"source"`
	MIME   string   `json:"content-type,omitempty"`
	Time   *float64 `json:"time,omitempty"`
	Frame  string   `json:"frame,omitempty"`
}

// Call records an actual submitted request, excluding authentication headers.
//   - Method, Endpoint: the HTTP method and request URL
//   - ContentType: the submitted body type
//   - Payload: the retained body, with binary content represented by references
//   - Sent: the UTC request capture time
//   - Error: the failure message when no response arrived
//
//nolint:tagliatelle // Keep the provisional record's hyphenated JSON keys.
type Call struct {
	Method      string          `json:"method"`
	Endpoint    string          `json:"endpoint"`
	ContentType string          `json:"content-type,omitempty"`
	Payload     json.RawMessage `json:"payload"`
	Sent        time.Time       `json:"sent"`
	Error       string          `json:"error,omitempty"`
	// references associates payload fields with retained binary content.
	references []binaryReference
}

// Response retains one full provider body before its execution-specific decode.
//   - Type: the synchronous or asynchronous request role
//   - RequestIndex: the index of the request in Request.Calls
//   - Endpoint: the associated request URL
//   - Status: the HTTP response status
//   - ContentType: the returned body type
//   - Received: the UTC response capture time
//   - Body: the full retained body, with binary content represented by references
//   - Incomplete: whether capturing the response failed
//   - CaptureError: the response read failure message
//   - DecodeError: the JSON decode failure for a non-JSON response
//
//nolint:tagliatelle // Keep the provisional record's hyphenated JSON keys.
type Response struct {
	Type         string          `json:"type"`
	RequestIndex int             `json:"request-index"`
	Endpoint     string          `json:"endpoint"`
	Status       int             `json:"status"`
	ContentType  string          `json:"content-type,omitempty"`
	Received     time.Time       `json:"received"`
	Body         json.RawMessage `json:"full-response"`
	Incomplete   bool            `json:"incomplete,omitempty"`
	CaptureError string          `json:"capture-error,omitempty"`
	DecodeError  string          `json:"decode-error,omitempty"`
	// references associates response fields with retained binary content.
	references []binaryReference
}

// File describes generated media successfully saved to its final path.
//   - SavedFile: the final path and byte count
//   - MIME: the retained content type
//   - References: the provider download URLs whose content matches the file
//
//nolint:tagliatelle // Keep the provisional record's hyphenated JSON keys.
type File struct {
	artifact.SavedFile

	MIME       string   `json:"content-type,omitempty"`
	References []string `json:"provider-references,omitempty"`
}

// ReturnValue keeps a provider's own retained shape separate from generic facts.
//   - Provider, Model: the catalog identifiers associated with the retained data
//   - Data: the provider-specific values available for reuse
//
//nolint:tagliatelle // Keep the provisional record's hyphenated JSON keys.
type ReturnValue struct {
	Provider string          `json:"provider"`
	Model    string          `json:"model"`
	Data     json.RawMessage `json:"retained-data"`
}

// New creates the record for one generation.
func New(provider, model, version, prompt string, started time.Time) *Record {
	return &Record{
		Schema: schemaVersion, ID: rand.Text(), Version: version, Provider: provider,
		Model: model, Started: started.UTC(), Request: Request{Prompt: prompt, Calls: []Call{}},
		Responses: []Response{}, Artifacts: []File{},
	}
}

// SetFields assigns provider-declared binary fields to the record without copying the slices.
// Explicit data URIs are recognized even without a declaration.
func (record *Record) SetFields(requestFields, responseFields []BinaryField) {
	if record == nil {
		return
	}

	record.requestFields, record.responseFields = requestFields, responseFields
}

// Describe assigns supplied options and input sources to the record without copying them.
func (record *Record) Describe(supplied json.RawMessage, sources []string) {
	if record == nil {
		return
	}

	record.Request.Supplied, record.Request.Sources = supplied, sources
}

// Prepared records the inputs actually passed to the request builder and makes unchanged local
// files available for exact-content matching during persistence.
func (record *Record) Prepared(inputs []media.Input) {
	if record == nil {
		return
	}

	record.Request.Inputs = make([]Input, 0, len(inputs))
	for inputIndex := range inputs {
		input := &inputs[inputIndex]
		record.Request.Inputs = append(record.Request.Inputs, Input{Source: input.Source(), MIME: input.MIME, Time: input.Time, Frame: input.FrameAnchor})
		record.binaries.source(input)
	}
}

// Begin records the actual payload before a provider request is sent. A nil receiver leaves
// persistence disabled and returns an unused request index.
func (record *Record) Begin(method, endpoint, contentType string, body []byte) int {
	if record == nil {
		return -1
	}

	payload, references, err := record.binaries.request(body, contentType, record.requestFields)
	record.addFault(err)
	record.Request.Calls = append(record.Request.Calls, Call{
		Method: method, Endpoint: endpoint,
		ContentType: contentType, Payload: payload, Sent: time.Now().UTC(), references: references,
	})

	return len(record.Request.Calls) - 1
}

// Receive records a response before execution narrows its JSON shape.
func (record *Record) Receive(requestIndex int, responseType string, status int, contentType string, body []byte, captureErr error) {
	if record == nil {
		return
	}

	response := Response{Type: responseType, RequestIndex: requestIndex, Status: status, ContentType: contentType, Received: time.Now().UTC()}
	if requestIndex >= 0 && requestIndex < len(record.Request.Calls) {
		response.Endpoint = record.Request.Calls[requestIndex].Endpoint
	}

	if captureErr != nil {
		response.Incomplete = true
		response.CaptureError = captureErr.Error()
	}

	if !json.Valid(body) {
		response.Body, _ = json.Marshal(string(body)) //nolint:errcheck,errchkjson // A string always has a JSON representation.

		var value json.RawMessage
		if err := json.Unmarshal(body, &value); err != nil {
			response.DecodeError = err.Error()
		}
	} else {
		var err error

		response.Body, response.references, err = record.binaries.normalize(body, record.responseFields)
		record.addFault(err)
	}

	record.Responses = append(record.Responses, response)
}

// ReceiveFile retains a downloaded body independently of the artifact's temporary file. The saved
// response later references its actual final file.
func (record *Record) ReceiveFile(requestIndex, status int, contentType, path string, captureErr error) {
	if record == nil {
		return
	}

	response := Response{Type: Synchronous, RequestIndex: requestIndex, Status: status, ContentType: contentType, Received: time.Now().UTC(), Body: json.RawMessage(pendingFileBody)}
	if requestIndex >= 0 && requestIndex < len(record.Request.Calls) {
		response.Endpoint = record.Request.Calls[requestIndex].Endpoint
	}

	if captureErr != nil {
		response.Incomplete = true
		response.CaptureError = captureErr.Error()
	}

	if path != "" {
		reference, err := record.binaries.copyFile(path, contentType)
		record.addFault(err)

		reference.Path = []string{retainedFileField}
		reference.PlainPath = true
		response.references = []binaryReference{reference}
	}

	record.Responses = append(record.Responses, response)
}

// RequestFailed records a request that received no response.
func (record *Record) RequestFailed(requestIndex int, requestErr error) {
	if record == nil || requestErr == nil || requestIndex < 0 || requestIndex >= len(record.Request.Calls) {
		return
	}

	record.Request.Calls[requestIndex].Error = requestErr.Error()
}

// Retain adds provider-specific values for subsequent requests.
func (record *Record) Retain(provider, model string, values json.RawMessage) {
	if record == nil {
		return
	}

	record.Returns = append(record.Returns, ReturnValue{Provider: provider, Model: model, Data: values})
}

// ProviderFinished distinguishes provider execution from later local write errors.
func (record *Record) ProviderFinished(generationErr error) {
	if record == nil {
		return
	}

	record.ProviderStatus = completionStatus(generationErr)
}

// Save finishes the record and writes it beside the supplied final artifact paths. It also saves
// unmatched binary content and returns the record path with any persistence errors.
func (record *Record) Save(dir, stem string, files []artifact.SavedFile, generationErr error) (string, error) {
	if record == nil {
		return "", nil
	}

	record.Finished = time.Now().UTC()
	record.ElapsedMS = record.Finished.Sub(record.Started).Milliseconds()

	record.Status = completionStatus(generationErr)
	if generationErr != nil {
		record.Errors = append(record.Errors, generationErr.Error())
	}

	paths, failures, err := record.binaries.persist(dir, stem, files)
	record.addFault(err)

	for _, file := range files {
		record.Artifacts = append(record.Artifacts, File{SavedFile: file, MIME: record.binaries.fileMIMEs[file.Path], References: record.fileReferences(file.Path, paths)})
	}

	record.resolveBinaries(paths, failures)

	if len(record.faults) > 0 {
		record.Status = statusFailed
	}

	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", fmt.Errorf("%q: %w, %w", stem, errs.ErrJSONEncode, err)
	}

	saved, writeErr := artifact.Write(dir, stem+".bild", "."+recordFormat, append(encoded, '\n'))

	return saved.Path, errors.Join(writeErr, errors.Join(record.faults...))
}

// resolveBinaries supplies actual saved paths and associates each failed binary save with its
// request or response field, preserving the filesystem cause.
func (record *Record) resolveBinaries(paths map[[sha256.Size]byte]string, failures map[[sha256.Size]byte]error) {
	for index := range record.Request.Calls {
		request := &record.Request.Calls[index]
		payload, err := resolveReferences(request.Payload, request.references, paths, failures)
		request.Payload = payload

		if err != nil {
			record.addFault(fmt.Errorf("%q: %w", requestPath+"/"+strconv.Itoa(index), err))
		}
	}

	for index := range record.Responses {
		response := &record.Responses[index]
		body, err := resolveReferences(response.Body, response.references, paths, failures)
		response.Body = body

		if err != nil {
			record.addFault(fmt.Errorf("%q: %w", responsePath+"/"+strconv.Itoa(index), err))
		}
	}
}

// addFault retains each capture failure for final reporting and the saved record.
func (record *Record) addFault(err error) {
	if err == nil {
		return
	}

	record.faults = append(record.faults, err)
	record.PersistenceErrors = append(record.PersistenceErrors, err.Error())
}

// fileReferences associates a completed artifact with the provider locations whose downloaded bytes
// match it. Response order and artifact order may differ.
func (record *Record) fileReferences(filePath string, paths map[[sha256.Size]byte]string) []string {
	var references []string

	for responseIndex := range record.Responses {
		response := &record.Responses[responseIndex]
		for _, binary := range response.references {
			if binary.PlainPath && paths[binary.Digest] == filePath && response.Endpoint != "" && !slices.Contains(references, response.Endpoint) {
				references = append(references, response.Endpoint)
			}
		}
	}

	return references
}

// Read loads a generation record and validates its schema and required fields.
func Read(path string) (*Record, error) {
	// #nosec G304 -- path is the explicit record selected by the user.
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%q: %w, %w", path, errs.ErrRecordRead, err)
	}

	var record *Record
	if err := json.Unmarshal(encoded, &record); err != nil {
		return nil, fmt.Errorf("%q: %w, %w", path, errs.ErrRecordInvalid, err)
	}

	if record == nil || record.Schema != schemaVersion || record.Provider == "" || record.Model == "" || record.Responses == nil || record.Request.Calls == nil {
		return nil, fmt.Errorf("%q: %w", path, errs.ErrRecordInvalid)
	}

	return record, nil
}

// completionStatus classifies a completed operation without losing cancellation.
func completionStatus(err error) string {
	if errors.Is(err, errs.ErrCanceled) {
		return statusCanceled
	}

	if err != nil {
		return statusFailed
	}

	return statusCompleted
}
