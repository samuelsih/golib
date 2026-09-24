package oas

import (
	"encoding/json/v2"
	"slices"
	"testing"

	"github.com/guregu/null/v6"
	"github.com/samuelsih/golib/assert"
)

func TestRefTMarshal(t *testing.T) {
	tests := []struct {
		name  string
		input RefT[Schema]
		want  string
	}{
		{
			name:  "inline value",
			input: RefT[Schema]{Value: &Schema{Type: Strings{"object"}}},
			want:  `{"type":"object"}`,
		},
		{
			name:  "reference",
			input: RefT[Schema]{Ref: &Reference{Ref: "#/components/schemas/User"}},
			want:  `{"$ref":"#/components/schemas/User"}`,
		},
		{
			name:  "empty",
			input: RefT[Schema]{},
			want:  `null`,
		},
	}

	for tt := range slices.Values(tests) {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, string(got), tt.want)
		})
	}
}

func TestRefTUnmarshal(t *testing.T) {
	t.Run("reference", func(t *testing.T) {
		var got RefT[Schema]

		err := json.Unmarshal([]byte(`{"$ref":"#/components/schemas/User"}`), &got)
		assert.NoError(t, err)
		assert.NotNil(t, got.Ref)
		assert.Nil(t, got.Value)
		assert.Equal(t, got.Ref.Ref, "#/components/schemas/User")
	})

	t.Run("inline value", func(t *testing.T) {
		var got RefT[Schema]

		err := json.Unmarshal([]byte(`{"type":"object"}`), &got)
		assert.NoError(t, err)
		assert.Nil(t, got.Ref)
		assert.NotNil(t, got.Value)
		assert.Equal(t, got.Value.Type, Strings{"object"})
	})

	t.Run("null", func(t *testing.T) {
		var got RefT[Schema]

		err := json.Unmarshal([]byte(`null`), &got)
		assert.NoError(t, err)
		assert.Nil(t, got.Ref)
		assert.Nil(t, got.Value)
	})
}

func TestBoolOrSchemaMarshal(t *testing.T) {
	tests := []struct {
		name  string
		input BoolOrSchema
		want  string
	}{
		{
			name:  "true",
			input: BoolOrSchema{Bool: null.BoolFrom(true)},
			want:  `true`,
		},
		{
			name:  "false",
			input: BoolOrSchema{Bool: null.BoolFrom(false)},
			want:  `false`,
		},
		{
			name:  "schema",
			input: BoolOrSchema{Schema: RefT[Schema]{Ref: &Reference{Ref: "#/components/schemas/User"}}},
			want:  `{"$ref":"#/components/schemas/User"}`,
		},
		{
			name:  "empty",
			input: BoolOrSchema{},
			want:  `null`,
		},
	}

	for tt := range slices.Values(tests) {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, string(got), tt.want)
		})
	}
}

func TestBoolOrSchemaUnmarshal(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		var got BoolOrSchema

		err := json.Unmarshal([]byte(`false`), &got)
		assert.NoError(t, err)
		assert.True(t, got.Bool.Valid)
		assert.False(t, got.Bool.Bool)
		assert.Nil(t, got.Schema.Value)
	})

	t.Run("schema", func(t *testing.T) {
		var got BoolOrSchema

		err := json.Unmarshal([]byte(`{"type":"object"}`), &got)
		assert.NoError(t, err)
		assert.False(t, got.Bool.Valid)
		assert.NotNil(t, got.Schema.Value)
		assert.Equal(t, got.Schema.Value.Type, Strings{"object"})
	})

	t.Run("null", func(t *testing.T) {
		var got BoolOrSchema

		err := json.Unmarshal([]byte(`null`), &got)
		assert.NoError(t, err)
		assert.False(t, got.Bool.Valid)
		assert.Nil(t, got.Schema.Value)
	})
}

func TestStringsMarshal(t *testing.T) {
	tests := []struct {
		name  string
		input Strings
		want  string
	}{
		{name: "single", input: Strings{"object"}, want: `"object"`},
		{name: "multiple", input: Strings{"string", "null"}, want: `["string","null"]`},
		{name: "empty", input: Strings{}, want: `[]`},
	}

	for tt := range slices.Values(tests) {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, string(got), tt.want)
		})
	}
}

func TestStringsUnmarshal(t *testing.T) {
	t.Run("single string", func(t *testing.T) {
		var got Strings

		err := json.Unmarshal([]byte(`"string"`), &got)
		assert.NoError(t, err)
		assert.Equal(t, got, Strings{"string"})
	})

	t.Run("array", func(t *testing.T) {
		var got Strings

		err := json.Unmarshal([]byte(`["string","null"]`), &got)
		assert.NoError(t, err)
		assert.Equal(t, got, Strings{"string", "null"})
	})

	t.Run("null", func(t *testing.T) {
		var got Strings

		err := json.Unmarshal([]byte(`null`), &got)
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestUnionOmitZero(t *testing.T) {
	tests := []struct {
		name  string
		input unionHolder
		want  string
	}{
		{
			name:  "empty",
			input: unionHolder{},
			want:  `{}`,
		},
		{
			name: "set",
			input: unionHolder{
				Ref:     RefT[Schema]{Ref: &Reference{Ref: "#/x"}},
				Bool:    BoolOrSchema{Bool: null.BoolFrom(false)},
				Strings: Strings{"object"},
			},
			want: `{"ref":{"$ref":"#/x"},"bool":false,"strings":"object"}`,
		},
	}

	for tt := range slices.Values(tests) {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, string(got), tt.want)
		})
	}
}

type unionHolder struct {
	Ref     RefT[Schema] `json:"ref,omitzero"`
	Bool    BoolOrSchema `json:"bool,omitzero"`
	Strings Strings      `json:"strings,omitempty"`
}
