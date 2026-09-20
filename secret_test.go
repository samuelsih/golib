package golib

import (
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	"github.com/samuelsih/golib/assert"
)

type secretPayload struct {
	User string         `json:"user"`
	Pass Secret[string] `json:"pass"`
}

func TestSecretExposedValue(t *testing.T) {
	assert.Equal(t, NewSecret("hello").ExposedValue(), "hello")
	assert.Equal(t, NewSecret(42).ExposedValue(), 42)
	assert.Equal(t, NewSecret([]int{1, 2}).ExposedValue(), []int{1, 2})

	var zero Secret[int]
	assert.Equal(t, zero.ExposedValue(), 0)
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
		assert.Equal(t, s.ExposedValue(), "hello")
	})

	t.Run("int", func(t *testing.T) {
		var s Secret[int]
		assert.NoError(t, json.Unmarshal([]byte(`42`), &s))
		assert.Equal(t, s.ExposedValue(), 42)
	})

	t.Run("slice", func(t *testing.T) {
		var s Secret[[]string]
		assert.NoError(t, json.Unmarshal([]byte(`["a","b"]`), &s))
		assert.Equal(t, s.ExposedValue(), []string{"a", "b"})
	})

	t.Run("nested struct", func(t *testing.T) {
		var s Secret[secretPayload]
		assert.NoError(t, json.Unmarshal([]byte(`{"user":"sam","pass":"hunter2"}`), &s))
		assert.Equal(t, s.ExposedValue().User, "sam")
		assert.Equal(t, s.ExposedValue().Pass.ExposedValue(), "hunter2")
	})
}

func TestSecretUnmarshalJSONNull(t *testing.T) {
	s := NewSecret("keep")
	assert.NoError(t, json.Unmarshal([]byte(`null`), &s))
	assert.Equal(t, s.ExposedValue(), "keep")
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

func TestSecretMarshalUnmarshalIsRedacted(t *testing.T) {
	data, err := json.Marshal(NewSecret("hunter2"))
	assert.NoError(t, err)

	var decoded Secret[string]
	assert.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, decoded.ExposedValue(), "{REDACTED}")
}
