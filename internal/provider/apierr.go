package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
)

// Provider error-body fields shared by supported response shapes.
const (
	respFieldDetail = "detail"
	respFieldMsg    = "msg"
)

// APIErr returns an error for a failed provider response, classified as a
// response-status failure. A server message extracted from the body's
// documented shape is retained separately from status context, so the renderer
// can surface the provider's own text. A body matching no documented shape
// keeps a bounded excerpt in
// the diagnostic context only.
func APIErr(provModelLabel string, status int, body []byte) error {
	statusContext := fmt.Sprintf(errs.StatusContextForm, provModelLabel, status)
	classification := errs.ErrResponseStatus

	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		classification = errs.ErrResponseStatusTemporary
	}

	msg := errMsg(body)
	if msg == "" {
		return fmt.Errorf("%q, %w", statusContext+": "+errs.Excerpt(body), classification)
	}

	return fmt.Errorf("%q, %w, %w", statusContext, classification,
		&errs.ProviderError{Message: msg, Cause: errs.ErrResponseServer})
}

// PollResponseError preserves both HTTP status and transport failures from one
// observation. A truncated error body must not erase a permanent HTTP status.
// Recovery is decided only by the polling loop; submissions are never retried.
func PollResponseError(identity string, status int, body []byte, transportErr error) error {
	if status != 0 && status/100 != 2 {
		return errors.Join(APIErr(identity, status, body), transportErr)
	}

	return transportErr
}

// errMsg returns the server message from an error response body's documented
// shape, or an empty string when the body matches none. OpenAI, xAI,
// OpenRouter, and Google carry the message at error.message; Google's
// streaming endpoint wraps that error document in a one-element JSON array;
// BFL carries a top-level detail, which on validation failures is an array
// of objects each carrying msg; Kling carries a top-level message. The
// nested message wins when a response contains more than one shape.
func errMsg(body []byte) string {
	var rawErrResp map[string]any
	if json.Unmarshal(body, &rawErrResp) != nil {
		var arrayWrapped []map[string]any
		if json.Unmarshal(body, &arrayWrapped) != nil || len(arrayWrapped) == 0 {
			return ""
		}

		rawErrResp = arrayWrapped[0]
	}

	if e, ok := rawErrResp["error"].(map[string]any); ok {
		if msg, ok := e["message"].(string); ok && msg != "" {
			return msg
		}
	}

	switch detail := rawErrResp[respFieldDetail].(type) {
	case string:
		if detail != "" {
			return detail
		}
	case []any:
		if message := joinDetailMessages(detail); message != "" {
			return message
		}
	}

	if message, ok := rawErrResp["message"].(string); ok {
		return message
	}

	return ""
}

// joinDetailMessages returns the msg texts of a BFL validation-failure detail
// array as one semicolon-separated text, or an empty string when no error detail
// carries one.
func joinDetailMessages(errDetails []any) string {
	var messages []string

	for _, errDetail := range errDetails {
		errDetailFields, ok := errDetail.(map[string]any)
		if !ok {
			continue
		}

		if msg, ok := errDetailFields[respFieldMsg].(string); ok && msg != "" {
			messages = append(messages, msg)
		}
	}

	return strings.Join(messages, "; ")
}
