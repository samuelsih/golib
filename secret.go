package golib

import (
	"bytes"
	"encoding"
	"encoding/json/v2"
	"fmt"
)

var (
	_ fmt.Stringer             = Secret[string]{}
	_ fmt.GoStringer           = Secret[string]{}
	_ json.Marshaler           = Secret[string]{}
	_ encoding.TextUnmarshaler = (*Secret[string])(nil)
	_ json.Unmarshaler         = (*Secret[string])(nil)
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

// UnmarshalText decodes text into the secret, delegating to the wrapped type's encoding.TextUnmarshaler or assigning string and []byte values.
func (s *Secret[T]) UnmarshalText(text []byte) error {
	value := new(T)

	if unmarshaler, ok := any(value).(encoding.TextUnmarshaler); ok {
		if err := unmarshaler.UnmarshalText(text); err != nil {
			return err
		}
		*s = NewSecret(*value)
		return nil
	}

	switch v := any(value).(type) {
	case *string:
		*v = string(text)
	case *[]byte:
		*v = bytes.Clone(text)
	default:
		return fmt.Errorf("golib: cannot unmarshal text into Secret[%T]", *value)
	}

	*s = NewSecret(*value)
	return nil
}

// Value returns the wrapped value, revealing the secret.
func (s Secret[T]) Value() T {
	return s.val
}
