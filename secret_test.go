package golib

import (
	"encoding"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/samuelsih/golib/assert"
)

type secretPayload struct {
	User string         `json:"user"`
	Pass Secret[string] `json:"pass"`
}

var errSecretText = errors.New("secret text unmarshal failed")

type upperSecret string

func (u *upperSecret) UnmarshalText(text []byte) error {
	*u = upperSecret(strings.ToUpper(string(text)))
	return nil
}

type failingSecret struct {
	value string
}

func (f *failingSecret) UnmarshalText(text []byte) error {
	f.value = string(text)
	return errSecretText
}

func TestSecretExposedValue(t *testing.T) {
	assert.Equal(t, NewSecret("hello").Value(), "hello")
	assert.Equal(t, NewSecret(42).Value(), 42)
	assert.Equal(t, NewSecret([]int{1, 2}).Value(), []int{1, 2})

	var zero Secret[int]
	assert.Equal(t, zero.Value(), 0)
}

func TestSecretRedactsFormatting(t *testing.T) {
	s := NewSecret("hunter2")

	assert.Equal(t, s.String(), "{REDACTED}")
	assert.Equal(t, s.GoString(), "{REDACTED}")

	for _, format := range []string{"%v", "%+v", "%s", "%#v"} {
		assert.Equal(t, fmt.Sprintf(format, s), "{REDACTED}")
	}
}

func TestSecretDoesNotLeakInStruct(t *testing.T) {
	got := fmt.Sprintf("%+v", secretPayload{User: "sam", Pass: NewSecret("hunter2")})

	assert.Equal(t, got, "{User:sam Pass:{REDACTED}}")
	assert.False(t, strings.Contains(got, "hunter2"))
}

func TestSecretMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		secret any
	}{
		{name: "string", secret: NewSecret("value")},
		{name: "int", secret: NewSecret(99)},
		{name: "slice", secret: NewSecret([]int{1, 2})},
		{name: "pointer", secret: NewSecret(&secretPayload{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.secret)
			assert.NoError(t, err)
			assert.Equal(t, string(got), `"{REDACTED}"`)
		})
	}
}

func TestSecretMarshalJSONInStruct(t *testing.T) {
	got, err := json.Marshal(secretPayload{User: "sam", Pass: NewSecret("hunter2")})
	assert.NoError(t, err)
	assert.Equal(t, string(got), `{"user":"sam","pass":"{REDACTED}"}`)
}

func TestSecretUnmarshalJSON(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		var s Secret[string]
		assert.NoError(t, json.Unmarshal([]byte(`"hello"`), &s))
		assert.Equal(t, s.Value(), "hello")
	})

	t.Run("int", func(t *testing.T) {
		var s Secret[int]
		assert.NoError(t, json.Unmarshal([]byte(`42`), &s))
		assert.Equal(t, s.Value(), 42)
	})

	t.Run("slice", func(t *testing.T) {
		var s Secret[[]string]
		assert.NoError(t, json.Unmarshal([]byte(`["a","b"]`), &s))
		assert.Equal(t, s.Value(), []string{"a", "b"})
	})

	t.Run("nested struct", func(t *testing.T) {
		var s Secret[secretPayload]
		assert.NoError(t, json.Unmarshal([]byte(`{"user":"sam","pass":"hunter2"}`), &s))
		assert.Equal(t, s.Value().User, "sam")
		assert.Equal(t, s.Value().Pass.Value(), "hunter2")
	})
}

func TestSecretUnmarshalJSONNull(t *testing.T) {
	s := NewSecret("keep")
	assert.NoError(t, json.Unmarshal([]byte(`null`), &s))
	assert.Equal(t, s.Value(), "keep")
}

func TestSecretUnmarshalJSONInvalid(t *testing.T) {
	t.Run("wrong type", func(t *testing.T) {
		var s Secret[int]
		assert.NotNil(t, json.Unmarshal([]byte(`"nope"`), &s))
	})

	t.Run("malformed", func(t *testing.T) {
		var s Secret[string]
		assert.NotNil(t, json.Unmarshal([]byte(`{`), &s))
	})
}

func TestSecretUnmarshalText(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		var s Secret[string]
		assert.NoError(t, s.UnmarshalText([]byte("hunter2")))
		assert.Equal(t, s.Value(), "hunter2")
	})

	t.Run("empty text", func(t *testing.T) {
		var s Secret[string]
		assert.NoError(t, s.UnmarshalText(nil))
		assert.Equal(t, s.Value(), "")
	})

	t.Run("TextUnmarshaler interface", func(t *testing.T) {
		var s Secret[string]

		var u encoding.TextUnmarshaler = &s
		assert.NoError(t, u.UnmarshalText([]byte("via interface")))
		assert.Equal(t, s.Value(), "via interface")
	})

	t.Run("bytes", func(t *testing.T) {
		var s Secret[[]byte]

		text := []byte("hunter2")
		assert.NoError(t, s.UnmarshalText(text))
		assert.Equal(t, s.Value(), []byte("hunter2"))

		text[0] = 'X'
		assert.Equal(t, s.Value(), []byte("hunter2"))
	})

	t.Run("text unmarshaler", func(t *testing.T) {
		var s Secret[upperSecret]
		assert.NoError(t, s.UnmarshalText([]byte("hunter2")))
		assert.Equal(t, s.Value(), upperSecret("HUNTER2"))
	})

	t.Run("error propagates and keeps value", func(t *testing.T) {
		s := NewSecret(failingSecret{value: "keep"})

		err := s.UnmarshalText([]byte("new"))
		assert.ErrorIs(t, err, errSecretText)
		assert.Equal(t, s.Value().value, "keep")
	})

	t.Run("unsupported type", func(t *testing.T) {
		var s Secret[int]
		assert.NotNil(t, s.UnmarshalText([]byte("42")))
	})
}

func TestSecretMarshalUnmarshalIsRedacted(t *testing.T) {
	data, err := json.Marshal(NewSecret("hunter2"))
	assert.NoError(t, err)

	var decoded Secret[string]
	assert.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, decoded.Value(), "{REDACTED}")
}

func TestSecretJSONV1Compatibility(t *testing.T) {
	got, err := jsonv1.Marshal(NewSecret("hunter2"))
	assert.NoError(t, err)
	assert.Equal(t, string(got), `"{REDACTED}"`)

	got, err = jsonv1.Marshal(secretPayload{User: "sam", Pass: NewSecret("hunter2")})
	assert.NoError(t, err)
	assert.Equal(t, string(got), `{"user":"sam","pass":"{REDACTED}"}`)

	var decoded Secret[string]
	assert.NoError(t, jsonv1.Unmarshal([]byte(`"hello"`), &decoded))
	assert.Equal(t, decoded.Value(), "hello")

	keep := NewSecret("keep")
	assert.NoError(t, jsonv1.Unmarshal([]byte(`null`), &keep))
	assert.Equal(t, keep.Value(), "keep")
}
