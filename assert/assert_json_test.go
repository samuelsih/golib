package assert

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

const sampleJSON = `{
	"id": "foo",
	"active": true,
	"count": 3,
	"price": 9.99,
	"none": null,
	"tags": ["a", "b"],
	"users": [
		{"id": 1, "name": "mario"},
		{"id": 2, "name": "luigi"}
	],
	"meta": {"nested": {"deep": "value"}}
}`

func TestJSONPath(t *testing.T) {
	data := []byte(sampleJSON)

	Equal(t, JSONPath(data, "$.id"), "foo")
	Equal(t, JSONPath(data, "$.active"), true)
	Equal(t, JSONPath(data, "$.meta.nested.deep"), "value")
	Equal(t, JSONPath(data, "$.tags[0]"), "a")
	Equal(t, JSONPath(data, "$.tags[1]"), "b")
	Equal(t, JSONPath(data, "$.users[0].name"), "mario")
	Equal(t, JSONPath(data, "$.users[1].id"), 2)

	NotNil(t, JSONPath(data, "$.tags"))
	NotNil(t, JSONPath(data, "$.meta"))
	Nil(t, JSONPath(data, "$.none"))
	Nil(t, JSONPath(data, "$.missing"))
	Nil(t, JSONPath(data, "$.users[9]"))
	Nil(t, JSONPath(data, "$.id.child"))
	Nil(t, JSONPath(data, "$.tags.name"))
}

func TestJSONPathRootValues(t *testing.T) {
	Equal(t, JSONPath(`{"a": {"b": 1}}`, "$"), map[string]any{"a": map[string]any{"b": 1}})
	Equal(t, JSONPath(`[1, 2]`, "$"), []int{1, 2})
	Equal(t, JSONPath(`"hello"`, "$"), "hello")
	Equal(t, JSONPath(`7`, "$"), 7)
	Nil(t, JSONPath(`null`, "$"))
	Nil(t, JSONPath(`null`, "$.a"))
}

func TestJSONPathNestedArrays(t *testing.T) {
	data := []byte(`[[1, 2], [3, 4]]`)

	Equal(t, JSONPath(data, "$[0][1]"), 2)
	Equal(t, JSONPath(data, "$[1][0]"), 3)
	Nil(t, JSONPath(data, "$[2][0]"))
}

func TestJSONPathRootArray(t *testing.T) {
	data := []byte(`[{"users": {"a": {"b": {"c": "deep"}}}}, {"id": 2}]`)

	Equal(t, JSONPath(data, "$[0].users.a.b.c"), "deep")
	Equal(t, JSONPath(data, "$[1].id"), 2)
	Equal(t, JSONPath(data, "$[0].users"), map[string]any{
		"a": map[string]any{"b": map[string]any{"c": "deep"}},
	})
	Nil(t, JSONPath(data, "$[2]"))
	Nil(t, JSONPath(data, "$[0].users.a.b.c.d"))
}

func TestJSONPathInputTypes(t *testing.T) {
	Equal(t, JSONPath(`{"id": "foo"}`, "$.id"), "foo")
	Equal(t, JSONPath([]byte(`{"id": "foo"}`), "$.id"), "foo")
	Equal(t, JSONPath(jsontext.Value(`{"id": "foo"}`), "$.id"), "foo")
}

func TestJSONPathKeyCharacters(t *testing.T) {
	data := []byte(`{"user-id": 7, "user name": "mario", "héllo": {"wörld": true}}`)

	Equal(t, JSONPath(data, "$.user-id"), 7)
	Equal(t, JSONPath(data, "$.user name"), "mario")
	Equal(t, JSONPath(data, "$.héllo.wörld"), true)
}

func TestJSONPathDeepEquality(t *testing.T) {
	data := []byte(`{
		"user": {"id": 1, "name": "mario", "roles": ["admin", "user"]},
		"users": [{"id": 1}, {"id": 2}],
		"tags": ["a", "b"]
	}`)

	Equal(t, JSONPath(data, "$.user"), map[string]any{
		"id":    1,
		"name":  "mario",
		"roles": []string{"admin", "user"},
	})
	Equal(t, JSONPath(data, "$.users"), []map[string]any{{"id": 1}, {"id": 2}})
	Equal(t, JSONPath(data, "$.users"), []any{map[string]int{"id": 1}, map[string]int{"id": 2}})
	Equal(t, JSONPath(data, "$.tags"), []string{"a", "b"})

	NotEqual(t, JSONPath(data, "$.tags"), []string{"a"})
	NotEqual(t, JSONPath(data, "$.user"), map[string]any{"id": 1, "name": "luigi", "roles": []string{"admin", "user"}})
}

func TestJSONPathNumbers(t *testing.T) {
	data := []byte(`{
		"int": 42,
		"negative": -7,
		"float": 0.1,
		"exponent": 1e2,
		"max": 9223372036854775807,
		"maxUint": 18446744073709551615
	}`)

	Equal(t, JSONPath(data, "$.int"), 42)
	Equal(t, JSONPath(data, "$.int"), int32(42))
	Equal(t, JSONPath(data, "$.int"), uint(42))
	Equal(t, JSONPath(data, "$.int"), 42.0)
	Equal(t, JSONPath(data, "$.negative"), -7)
	Equal(t, JSONPath(data, "$.float"), 0.1)
	Equal(t, JSONPath(data, "$.exponent"), 100)
	Equal(t, JSONPath(data, "$.max"), int64(math.MaxInt64))
	Equal(t, JSONPath(data, "$.maxUint"), uint64(math.MaxUint64))

	NotEqual(t, JSONPath(data, "$.int"), 43)
	NotEqual(t, JSONPath(data, "$.int"), "42")
	NotEqual(t, JSONPath(data, "$.max"), int64(math.MaxInt64-1))
	NotEqual(t, JSONPath(data, "$.maxUint"), uint64(math.MaxUint64-1))
}

func TestJSONPathExactNumbers(t *testing.T) {
	data := []byte(`{
		"huge": 123456789012345678901234567890,
		"tiny": 1e400,
		"decimal": 0.10
	}`)

	Equal(t, JSONPath(data, "$.huge"), jsonNumber("123456789012345678901234567890"))
	Equal(t, JSONPath(data, "$.huge"), jsonNumber("1.23456789012345678901234567890e29"))
	Equal(t, JSONPath(data, "$.tiny"), jsonNumber("1e400"))
	Equal(t, JSONPath(data, "$.decimal"), jsonNumber("0.1"))
	NotEqual(t, JSONPath(data, "$.tiny"), jsonNumber("2e400"))
	NotEqual(t, JSONPath(data, "$.huge"), int64(math.MaxInt64))
}

func TestJSONPathMissing(t *testing.T) {
	data := []byte(sampleJSON)

	paths := []string{
		"$.missing",
		"$.missing.deep",
		"$.id.missing",
		"$.users[9]",
		"$.users[0].missing",
		"$.id[0]",
		"$.users.name",
		"$.count[0]",
		"$.active[0]",
		"$.meta[0]",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			if got := JSONPath(data, path); got != nil {
				t.Errorf("JSONPath(%q) = %#v; want nil", path, got)
			}
			if _, err := JSONPathE(data, path); !errors.Is(err, errJSONPathNotFound) {
				t.Errorf("JSONPathE(%q) error = %v; want errJSONPathNotFound", path, err)
			}
		})
	}
}

func TestJSONPathMalformedPath(t *testing.T) {
	data := []byte(`{"id": 1}`)

	tests := []struct{ name, path string }{
		{name: "empty", path: ""},
		{name: "no root", path: "id"},
		{name: "root only dot", path: "$."},
		{name: "double dot", path: "$..id"},
		{name: "trailing dot", path: "$.id."},
		{name: "unclosed bracket", path: "$[0"},
		{name: "empty bracket", path: "$[]"},
		{name: "non numeric index", path: "$[x]"},
		{name: "negative index", path: "$[-1]"},
		{name: "junk after bracket", path: "$[0]x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSONPath(data, tt.path); !isJSONPathError(got) {
				t.Errorf("JSONPath(%q) = %#v; want jsonPathError", tt.path, got)
			}
			if _, err := JSONPathE(data, tt.path); err == nil {
				t.Errorf("JSONPathE(%q) error = nil; want non-nil", tt.path)
			}
		})
	}
}

func TestJSONPathMalformedDocument(t *testing.T) {
	tests := []struct{ name, doc string }{
		{name: "empty", doc: ""},
		{name: "whitespace", doc: "   "},
		{name: "garbage", doc: "not json"},
		{name: "truncated", doc: `{"id": 1`},
		{name: "trailing garbage", doc: `{"id": 1} x`},
		{name: "two values", doc: `{"id": 1} {"id": 2}`},
		{name: "duplicate names", doc: `{"id": 1, "id": 2}`},
		{name: "invalid utf8", doc: "{\"id\": \"\xff\xfe\"}"},
		{name: "nan", doc: `{"id": NaN}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSONPath([]byte(tt.doc), "$.id"); !isJSONPathError(got) {
				t.Errorf("JSONPath(%q) = %#v; want jsonPathError", tt.doc, got)
			}
		})
	}
}

func TestJSONPathE(t *testing.T) {
	value, err := JSONPathE([]byte(sampleJSON), "$.users[1].name")
	NoError(t, err)
	Equal(t, value, "luigi")

	_, err = JSONPathE([]byte(sampleJSON), "$.missing")
	if !errors.Is(err, errJSONPathNotFound) {
		t.Fatalf("error = %v; want errJSONPathNotFound", err)
	}
	if !strings.Contains(err.Error(), "$.missing") {
		t.Errorf("error = %v; want it to mention the path", err)
	}

	_, err = JSONPathE([]byte(`{"id": 1`), "$.id")
	if err == nil || !strings.Contains(err.Error(), "$.id") {
		t.Errorf("error = %v; want it to mention the path", err)
	}
}

func TestJSONPathErrorValue(t *testing.T) {
	got := JSONPath([]byte(`{"id": 1`), "$.id")
	if !isJSONPathError(got) {
		t.Fatalf("got %#v (%T); want jsonPathError", got, got)
	}
	if got == nil {
		t.Fatal("got nil; want non-nil error value")
	}

	if message := fmt.Sprintf("%v", got); !strings.Contains(message, "jsonpath") {
		t.Errorf("formatted error value = %q; want it to contain the jsonpath error", message)
	}
	if isEqual(got, "foo") {
		t.Errorf("isEqual(error value, %q) = true; want false", "foo")
	}
}

func TestEqualJSONValues(t *testing.T) {
	var got any = jsonNumber("42")

	Equal(t, got, 42)
	Equal(t, got, int64(42))
	Equal(t, got, uint(42))
	Equal(t, got, 42.0)
	NotEqual(t, got, "42")
	NotEqual(t, got, 43)

	var document any = []any{map[string]any{"id": jsonNumber("1")}}
	Equal(t, document, []map[string]any{{"id": 1}})
	NotEqual(t, document, []map[string]any{{"id": 2}})
}

func TestEqualValues(t *testing.T) {
	tests := []struct {
		name      string
		got       any
		want      any
		wantEqual bool
	}{
		{name: "json number and int", got: jsonNumber("42"), want: 42, wantEqual: true},
		{name: "json number and float", got: jsonNumber("0.1"), want: 0.1, wantEqual: true},
		{name: "json number and string", got: jsonNumber("42"), want: "42", wantEqual: false},
		{name: "json numbers with different spellings", got: jsonNumber("1e2"), want: jsonNumber("100"), wantEqual: true},
		{name: "decimal json numbers", got: jsonNumber("0.10"), want: jsonNumber("0.1"), wantEqual: true},
		{name: "huge json numbers equal", got: jsonNumber("1e400"), want: jsonNumber("1e400"), wantEqual: true},
		{name: "huge json numbers differ", got: jsonNumber("1e400"), want: jsonNumber("2e400"), wantEqual: false},
		{name: "integer and float", got: 1, want: 1.0, wantEqual: true},
		{name: "different numbers", got: 1, want: 2, wantEqual: false},
		{name: "large exact integers", got: jsonNumber("9223372036854775807"), want: int64(math.MaxInt64), wantEqual: true},
		{name: "large integers differ by one", got: jsonNumber("9223372036854775807"), want: int64(math.MaxInt64 - 1), wantEqual: false},
		{name: "uint64 max", got: jsonNumber("18446744073709551615"), want: uint64(math.MaxUint64), wantEqual: true},
		{name: "nested slices", got: []any{jsonNumber("1"), "two"}, want: []any{1, "two"}, wantEqual: true},
		{name: "different slice lengths", got: []any{jsonNumber("1")}, want: []any{1, 2}, wantEqual: false},
		{name: "nested maps", got: map[string]any{"a": []any{jsonNumber("1")}}, want: map[string]any{"a": []int{1}}, wantEqual: true},
		{name: "map missing key", got: map[string]any{"a": 1}, want: map[string]any{"b": 1}, wantEqual: false},
		{name: "map and slice", got: map[string]any{}, want: []any{}, wantEqual: false},
		{name: "slice and string", got: []any{}, want: "x", wantEqual: false},
		{name: "non string map keys", got: map[int]string{1: "a"}, want: map[int]string{1: "a"}, wantEqual: true},
		{name: "float32 and float64", got: float32(0.5), want: 0.5, wantEqual: true},
		{name: "nil map values", got: map[string]any{"a": nil}, want: map[string]any{"a": nil}, wantEqual: true},
		{name: "nil and slice", got: nil, want: []any{}, wantEqual: false},
		{name: "string and number", got: "1", want: 1, wantEqual: false},
		{name: "same structs equal", got: byID{id: 1, name: "mario"}, want: byID{id: 1, name: "mario"}, wantEqual: true},
		{name: "same structs differ", got: byID{id: 1, name: "mario"}, want: byID{id: 1, name: "luigi"}, wantEqual: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := equalValues(tt.got, tt.want); got != tt.wantEqual {
				t.Errorf("equalValues(%#v, %#v) = %v; want %v", tt.got, tt.want, got, tt.wantEqual)
			}
		})
	}
}

func isJSONPathError(value any) bool {
	_, ok := value.(jsonPathError)
	return ok
}
