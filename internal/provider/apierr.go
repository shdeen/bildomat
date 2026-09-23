package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
)

// Provider error fields carry validation details.
//   - respFieldDetail: a message or list of validation failures
//   - respFieldMsg: the message within a validation failure
const (
	respFieldDetail = "detail"
	respFieldMsg    = "msg"
)

// APIErr classifies an unsuccessful HTTP response and preserves a recognized server message.
// Otherwise it includes a bounded body excerpt. Statuses 429, 502, 503, and 504 are temporary.
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

// PollResponseError combines an unsuccessful HTTP status with any transport error. A successful or
// unavailable status returns the transport error unchanged.
func PollResponseError(identity string, status int, body []byte, transportErr error) error {
	if status != 0 && status/100 != 2 {
		return errors.Join(APIErr(identity, status, body), transportErr)
	}

	return transportErr
}

// errMsg extracts a message from a provider error object or the first array element. It checks
// error.message, detail text or validation messages, then the top-level message.
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

// joinDetailMessages joins nonempty validation messages with semicolons.
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
