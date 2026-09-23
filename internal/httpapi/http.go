package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/metadata"
)

// HTTP request limits.
//   - httpTimeout: the maximum duration of an HTTP request
//   - maxRespBytes: the maximum number of bytes read from an API response
//   - mediaTypeDetectBytes: the maximum prefix length used to identify downloaded media
const (
	httpTimeout          = 8 * time.Minute
	maxRespBytes         = 64 << 20
	mediaTypeDetectBytes = 512
)

// The HTTP header and body tokens the transport layer names.
//   - headerAuthorization: the header carrying a bearer credential
//   - headerContentType: the header naming a body's media type
//   - bearerScheme: the authorization scheme word before a bearer key
//   - contentTypeJSON: the media type of a JSON request body
//   - tempDownloadPattern: the temporary download file's name pattern
const (
	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	bearerScheme        = "Bearer"
	contentTypeJSON     = "application/json"

	tempDownloadPattern = "bild-dl-*"
)

// PostJSON sends body as JSON and returns the response status and body. When record is nonnil, it
// captures the actual request and full bounded response.
func PostJSON(ctx context.Context, endpoint string, credential AuthCredential, body any, record *metadata.Record) (status int, respBody []byte, err error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, nil, fmt.Errorf("%q, %w, %w", endpoint, errs.ErrTransportMarshal, err)
	}

	return send(ctx, http.MethodPost, endpoint, credential, contentTypeJSON, b, metadata.Synchronous, record)
}

// SendBody posts a prepared body and returns the response status and bounded body. A nonnil record
// captures the request, response, and any transfer failure.
func SendBody(ctx context.Context, endpoint string, credential AuthCredential, contentType string, body []byte, record *metadata.Record) (status int, respBody []byte, err error) {
	return send(ctx, http.MethodPost, endpoint, credential, contentType, body, metadata.Synchronous, record)
}

// GetAuth retrieves an endpoint with the credential and returns its status and bounded body. A
// nonnil record captures the transaction; responseType labels the response as a poll or direct
// retrieval.
func GetAuth(ctx context.Context, endpoint string, credential AuthCredential, responseType string, record *metadata.Record) (status int, respBody []byte, err error) {
	return send(ctx, http.MethodGet, endpoint, credential, "", nil, responseType, record)
}

// send performs a request and reads its bounded body before provider decoding. It captures the
// transaction in a nonnil record, including incomplete response bytes on failure.
func send(ctx context.Context, httpMethod, endpoint string, credential AuthCredential, contentType string, body []byte, responseType string, record *metadata.Record) (status int, respBody []byte, err error) {
	headers := make(http.Header)
	if contentType != "" {
		headers.Set(headerContentType, contentType)
	}

	response, requestIndex, err := request(ctx, httpMethod, endpoint, credential, headers, body, record)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close() //nolint:errcheck // Reading or streaming reports transfer failures; closing this response only releases its connection.

	responseBody, readErr := readFirstBytes(response.Body, maxRespBytes)
	record.Receive(requestIndex, responseType, response.StatusCode, response.Header.Get(headerContentType), responseBody, readErr)

	if readErr != nil {
		return response.StatusCode, nil, requestFailure(ctx, endpoint, readErr)
	}

	return response.StatusCode, responseBody, nil
}

// request authenticates and sends an HTTP request, recording submitted requests and send failures.
// The caller must close a successful response body.
func request(ctx context.Context, method, endpoint string, credential AuthCredential, headers http.Header, body []byte, record *metadata.Record) (*http.Response, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, -1, fmt.Errorf("%q: %w, %w", endpoint, errs.ErrTransportCreate, err)
	}

	if headers != nil {
		req.Header = headers.Clone()
	}

	credential.set(req.Header)
	requestIndex := record.Begin(method, endpoint, req.Header.Get(headerContentType), body)

	response, err := client(credential, req.URL).Do(req)
	if err != nil {
		record.RequestFailed(requestIndex, err)

		if !errors.Is(ctx.Err(), context.Canceled) {
			err = errors.Join(errs.ErrTransportRequest, err)
		}

		return nil, requestIndex, requestFailure(ctx, endpoint, err)
	}

	return response, requestIndex, nil
}

// requestFailure preserves a failed request or body operation and classifies an explicit
// cancellation without losing the underlying transport or read cause.
func requestFailure(ctx context.Context, endpoint string, cause error) error {
	if contextErr := ctx.Err(); errors.Is(contextErr, context.Canceled) {
		cause = errors.Join(errs.ErrCanceled, cause, contextErr)
	}

	return fmt.Errorf("%q: %w", endpoint, cause)
}

// client limits redirects and confines the supplied credential to the original origin.
func client(credential AuthCredential, origin *url.URL) *http.Client {
	return &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("%d: %w", len(via), errs.ErrTransportRedirectLimit)
			}

			if credential.HeaderName != "" && (req.URL.Scheme != origin.Scheme || req.URL.Host != origin.Host) {
				req.Header.Del(credential.HeaderName)
			}

			return nil
		},
	}
}

// readFirstBytes returns at most byteLimit bytes and reports read or size errors. On failure it
// preserves the received prefix for optional response recording.
func readFirstBytes(reader io.Reader, byteLimit int64) ([]byte, error) {
	if byteLimit < 0 {
		return nil, fmt.Errorf("%q, %w", fmt.Sprintf(ByteLimitForm, byteLimit), errs.ErrTransportSize)
	}

	readLimit := byteLimit
	if readLimit < math.MaxInt64 {
		readLimit++
	}

	data, err := io.ReadAll(io.LimitReader(reader, readLimit))

	oversized := int64(len(data)) > byteLimit
	if oversized {
		data = data[:byteLimit] //nolint:nilaway // len(data) > the nonnegative limit proves this slice is non-nil and the bound is valid.
	}

	if err != nil {
		return data, fmt.Errorf("%q, %w, %w", fmt.Sprintf(ByteLimitForm, byteLimit), errs.ErrTransportRead, err)
	}

	if oversized {
		return data, fmt.Errorf("%q, %w", fmt.Sprintf(ByteLimitForm, byteLimit), errs.ErrTransportSize)
	}

	return data, nil
}
