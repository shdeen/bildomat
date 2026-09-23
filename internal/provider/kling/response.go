package kling

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
)

// taskEnvelope preserves absent code separately from explicit JSON null.
type taskEnvelope struct {
	// Code distinguishes an absent code from null or another encoded value.
	Code json.RawMessage `json:"code"`
	// Message preserves the optional provider message and its JSON type.
	Message any `json:"message"`
	// Data contains the task payload.
	Data json.RawMessage `json:"data"`
}

// decodeEnvelope accepts additional remote fields and rejects a non-object envelope.
func decodeEnvelope(responseBody []byte, diagnosticContext string) (*taskEnvelope, error) {
	var response *taskEnvelope
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("%q, %w, %w", diagnosticContext, errs.ErrResponseDecode, err)
	}

	if response == nil {
		return nil, fmt.Errorf("%q, %w", diagnosticContext, errs.ErrResponseDecode)
	}

	return response, nil
}

// envelopeFailure returns nil for code zero and classifies missing, invalid, or nonzero response
// codes.
func envelopeFailure(response *taskEnvelope, diagnosticContext string) error {
	if len(response.Code) == 0 {
		return fmt.Errorf("%q, %w", diagnosticContext, errs.ErrResponseCodeMissing)
	}

	code, valid := exactInteger(response.Code)
	if !valid {
		return fmt.Errorf("%q, %w", diagnosticContext, errs.ErrResponseCodeInvalid)
	}

	if code == 0 {
		return nil
	}

	message, messagePresent := response.Message.(string)
	if !messagePresent || message == "" {
		message = fmt.Sprintf(EnvelopeCodeForm, diagnosticContext, code)
	}

	return &errs.ProviderError{Message: message, Cause: errs.ErrResponseGen}
}

// exactInteger converts a valid encoded JSON number to int without rounding. Decimal and exponent
// forms must be integral and fit int; oversized exponents do not allocate expanded numbers.
func exactInteger(encoded json.RawMessage) (int, bool) {
	text := string(encoded)
	if text == "" || (text[0] != '-' && (text[0] < '0' || text[0] > '9')) {
		return 0, false
	}

	mantissa, exponentText, hasExponent := strings.Cut(strings.ToLower(text), "e")
	negative := strings.HasPrefix(mantissa, "-")
	mantissa = strings.TrimPrefix(mantissa, "-")
	integerPart, fractionPart, _ := strings.Cut(mantissa, ".")

	digits := strings.TrimLeft(integerPart+fractionPart, "0")
	if digits == "" {
		return 0, true
	}

	exponent := 0

	if hasExponent {
		var err error

		exponent, err = strconv.Atoi(exponentText)
		if err != nil {
			return 0, false
		}
	}

	return integerFromDecimal(digits, len(fractionPart), exponent, negative)
}

// integerFromDecimal bounds the exponent before expanding or truncating digits. It rejects
// fractional remainders and int overflow without lossy conversion.
func integerFromDecimal(digits string, decimalPlaces, exponent int, negative bool) (int, bool) {
	// An int has at most 19 decimal digits on supported 64-bit platforms. Bound the exponent
	// before subtraction, so extreme exponents cannot overflow.
	maxDigits := len(strconv.FormatInt(int64(^uint(0)>>1), 10))
	if exponent > decimalPlaces+maxDigits || exponent < decimalPlaces-len(digits) {
		return 0, false
	}

	power := exponent - decimalPlaces
	if power < 0 {
		integerLength := len(digits) + power
		if strings.Trim(digits[integerLength:], "0") != "" {
			return 0, false
		}

		digits = digits[:integerLength]
	} else {
		if len(digits)+power > maxDigits {
			return 0, false
		}

		digits += strings.Repeat("0", power)
	}

	if negative {
		digits = "-" + digits
	}

	value, err := strconv.ParseInt(digits, 10, strconv.IntSize)

	return int(value), err == nil
}

// taskFailure returns a provider generation failure containing the task, terminal status, and
// provider message when one is present.
func taskFailure(taskID, taskStatus, failureMessage string) error {
	failureContext := fmt.Sprintf("%s (%s)", taskID, taskStatus)
	if failureMessage != "" {
		failureContext += ": " + failureMessage
	}

	return &errs.ProviderError{Message: failureContext, Cause: errs.ErrResponseGen}
}

// unknownStatus returns an unexpected-status failure containing the task and observed value.
func unknownStatus(taskID string, observedStatus any) error {
	return fmt.Errorf("%q, %w", fmt.Sprintf(errs.StatusContextForm, taskID, observedStatus), errs.ErrResponseUnknown)
}
