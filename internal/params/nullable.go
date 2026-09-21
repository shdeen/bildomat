package params

import (
	"encoding/json"
	"fmt"

	"github.com/shdeen/bildomat/internal/errs"
)

// nullJSON is the JSON representation of an absent parameter value.
const nullJSON = "null"

// Nullable stores a value and whether it was explicitly set.
type Nullable[T any] struct {
	// value contains the stored value.
	value T

	// isSet reports whether the value was explicitly set.
	isSet bool
}

// GetSetIf takes a condition and value and returns a set Nullable when the condition is
// true. Otherwise, it returns an unset Nullable.
func GetSetIf[T any](provided bool, val T) Nullable[T] {
	if provided {
		return Nullable[T]{value: val, isSet: true}
	}

	return Nullable[T]{}
}

// ValOr takes a default value and returns the stored value when set. Otherwise, it returns
// the default value.
func (n Nullable[T]) ValOr(defaultVal T) T {
	if n.isSet {
		return n.value
	}

	return defaultVal
}

// ValIf returns the stored value and whether it was set.
func (n Nullable[T]) ValIf() (T, bool) {
	return n.value, n.isSet
}

// MarshalJSON encodes the stored value, or the JSON null token when none was set.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.isSet {
		return []byte(nullJSON), nil
	}

	encoded, err := json.Marshal(n.value)
	if err != nil {
		return nil, fmt.Errorf("%q, %w, %w", fmt.Sprintf("%T", n.value), errs.ErrJSONEncode, err)
	}

	return encoded, nil
}

// UnmarshalJSON takes encoded data and decodes it into n. It clears n for null data and
// returns an error when a non-null value cannot be decoded.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if string(data) == nullJSON {
		*n = Nullable[T]{}

		return nil
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("%q, %w, %w", errs.Excerpt(data), errs.ErrJSONDecode, err)
	}

	*n = Nullable[T]{value: v, isSet: true}

	return nil
}
