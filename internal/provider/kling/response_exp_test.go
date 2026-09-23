package kling

// Invariants tested:
//  1. Kling envelope branches: decodeEnvelope must return ErrResponseDecode for a null document.
//     envelopeFailure must return ErrResponseNoData for a missing code, ErrResponseDecode for a
//     nonnumeric code, and ErrResponseGen for code 1200 without a message. exactInteger must reject
//     math.MaxFloat64.
//  2. Exact numerical envelope: For an arbitrary int64 coefficient and decimal exponent,
//     envelopeFailure must agree with an independent rational-number calculation. It must return
//     ErrResponseCodeInvalid for fractions and values outside the platform's int range, no error
//     for zero, and ErrResponseGen for other representable integers.

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestKlingEnvelopeBranches verifies invariant #1: Kling envelope branches.
//
// What is being tested:
// decodeEnvelope must return ErrResponseDecode for a null document. envelopeFailure must return
// ErrResponseNoData for a missing code, ErrResponseDecode for a nonnumeric code, and ErrResponseGen
// for code 1200 without a message. exactInteger must reject math.MaxFloat64.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestKlingEnvelopeBranches(t *testing.T) {
	if _, decodeErr := decodeEnvelope([]byte("null"), "null envelope"); !errors.Is(decodeErr, errs.ErrResponseDecode) {
		t.Errorf("✗ null envelope error = %v, want decode classification", decodeErr)
	}

	if envelopeErr := envelopeFailure(&taskEnvelope{}, "missing code"); !errors.Is(envelopeErr, errs.ErrResponseNoData) {
		t.Errorf("✗ missing code error = %v, want no-data classification", envelopeErr)
	}

	if envelopeErr := envelopeFailure(&taskEnvelope{Code: json.RawMessage(`"invalid"`)}, "invalid code"); !errors.Is(envelopeErr, errs.ErrResponseDecode) {
		t.Errorf("✗ invalid code error = %v, want decode classification", envelopeErr)
	}

	if envelopeErr := envelopeFailure(&taskEnvelope{Code: json.RawMessage(`1200`)}, "code-only failure"); !errors.Is(envelopeErr, errs.ErrResponseGen) {
		t.Errorf("✗ code-only error = %v, want generation classification", envelopeErr)
	}

	if _, integerValid := exactInteger(json.RawMessage(strconv.FormatFloat(math.MaxFloat64, 'g', -1, 64))); integerValid {
		t.Errorf("✗ maximum float converted to an integer")
	}

	if !t.Failed() {
		t.Log("✓ Kling envelope boundaries and integer conversion limits return classified outcomes")
	}
}

// FuzzResponseCode verifies invariant #2: Exact numerical envelope.
// What is being tested:
// For an arbitrary int64 coefficient and decimal exponent, envelopeFailure must agree with an
// independent rational-number calculation. It must return ErrResponseCodeInvalid for fractions and
// values outside the platform's int range, no error for zero, and ErrResponseGen for other
// representable integers.
// Test class: Expanded.
// Test layer: Fuzzing.
// Kind: permanent.
func FuzzResponseCode(f *testing.F) {
	for _, integral := range []int64{0, 1, -1, math.MaxInt64, math.MinInt64, 9007199254740993} {
		f.Add(integral, int8(0))
		f.Add(integral, int8(-1))
		f.Add(integral, int8(1))
	}

	f.Fuzz(func(t *testing.T, integral int64, scale int8) {
		exponent := int64(scale % 20)
		number := fmt.Sprintf("%de%d", integral, exponent)

		envelope, err := decodeEnvelope([]byte(`{"code":`+number+`}`), "numeric response")
		if err != nil {
			t.Fatalf("💣 setup failed: %v", err)
		}

		failure := envelopeFailure(envelope, "numeric response")
		rational := new(big.Rat).SetInt64(integral)

		magnitude := exponent
		if magnitude < 0 {
			magnitude = -magnitude
		}

		power := new(big.Int).Exp(big.NewInt(10), big.NewInt(magnitude), nil)

		factor := new(big.Rat).SetInt(power)
		if exponent < 0 {
			rational.Quo(rational, factor)
		} else {
			rational.Mul(rational, factor)
		}

		isInteger := rational.IsInt() && rational.Num().IsInt64()
		if isInteger {
			isInteger = int64(int(rational.Num().Int64())) == rational.Num().Int64()
		}

		switch {
		case !isInteger:
			if !errors.Is(failure, errs.ErrResponseCodeInvalid) {
				t.Errorf("✗ %s: error=%v; want invalid numerical code", number, failure)
			}
		case rational.Sign() == 0:
			if failure != nil {
				t.Errorf("✗ %s: error=%v; want successful zero code", number, failure)
			}
		default:
			if !errors.Is(failure, errs.ErrResponseGen) {
				t.Errorf("✗ %s: error=%v; want provider-reported failure", number, failure)
			}
		}

		if !t.Failed() {
			t.Log("✓ exact response code agrees with the rational oracle")
		}
	})
}
