package params

import (
	"encoding/json"
	"fmt"

	"github.com/shdeen/bildomat/internal/errs"
)

// nullJSON is the JSON representation of an absent parameter value.
const nullJSON = "null"

// Nullable distinguishes an explicitly supplied value from absence.
//   - value: the stored value, including its type's zero value
//   - isSet: whether a value was explicitly supplied
type Nullable[T any] struct {
	value T

	isSet bool
}

// GetSetIf returns a set Nullable only when provided is true.
func GetSetIf[T any](provided bool, val T) Nullable[T] {
	if provided {
		return Nullable[T]{value: val, isSet: true}
	}

	return Nullable[T]{}
}

// ValOr returns the stored value or defaultVal when unset.
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

// UnmarshalJSON replaces n with the decoded value, or clears it for null. An invalid value returns
// a decoding error without changing n.
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
