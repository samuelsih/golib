package golib

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
)

var (
	_ fmt.Stringer     = Secret[string]{}
	_ fmt.GoStringer   = Secret[string]{}
	_ json.Marshaler   = Secret[string]{}
	_ json.Unmarshaler = (*Secret[string])(nil)
)

// Secret holds a value that is redacted when formatted or marshaled.
type Secret[T any] struct {
	val T
}

// NewSecret returns a Secret wrapping value.
func NewSecret[T any](value T) Secret[T] {
	return Secret[T]{value}
}

// String returns "{REDACTED}".
func (s Secret[_]) String() string {
	return "{REDACTED}"
}

// GoString returns "{REDACTED}".
func (s Secret[_]) GoString() string {
	return s.String()
}

// MarshalJSON always encodes the secret as the string "{REDACTED}".
func (s Secret[_]) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('"')
	buf.WriteString(s.String())
	buf.WriteByte('"')
	return buf.Bytes(), nil
}

// UnmarshalJSON decodes p into the secret, leaving it unchanged when p is null.
func (s *Secret[T]) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		return nil
	}

	var value T
	if err := json.Unmarshal(p, &value); err != nil {
		return err
	}

	*s = NewSecret(value)
	return nil
}

// ExposedValue returns the wrapped value, revealing the secret.
func (s Secret[T]) ExposedValue() T {
	return s.val
}
