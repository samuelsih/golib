package assert

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

// jsonNumber keeps the exact text of a JSON number so large values survive
// decoding. Equal compares it numerically against int, uint and float values.
type jsonNumber string

var unmarshalJSONNumbers = json.UnmarshalFromFunc(func(decoder *jsontext.Decoder, value *any) error {
	if decoder.PeekKind() != jsontext.KindNumber {
		return errors.ErrUnsupported
	}
	number, err := decoder.ReadValue()
	if err != nil {
		return err
	}
	*value = jsonNumber(number)
	return nil
})

var errJSONPathNotFound = errors.New("value not found")

// JSONPath returns the value at path in the JSON document data. It supports
// $.field, $[index] and combinations such as $[0].users.id.
func JSONPath[T ~string | ~[]byte](data T, path string) any {
	value, err := JSONPathE(data, path)
	if err == nil {
		return value
	}
	if errors.Is(err, errJSONPathNotFound) {
		return nil
	}
	return jsonPathError{err}
}

// JSONPathE is JSONPath with explicit error reporting.
func JSONPathE[T ~string | ~[]byte](data T, path string) (any, error) {
	steps, err := parseJSONPath(path)
	if err != nil {
		return nil, err
	}

	var root any
	if err := json.Unmarshal([]byte(data), &root, json.WithUnmarshalers(unmarshalJSONNumbers)); err != nil {
		return nil, fmt.Errorf("jsonpath %q: invalid JSON: %w", path, err)
	}

	value, ok := resolveJSONPath(root, steps)
	if !ok {
		return nil, fmt.Errorf("jsonpath %q: %w", path, errJSONPathNotFound)
	}
	return value, nil
}

// jsonPathError fails assertions with the parse error it wraps.
type jsonPathError struct {
	err error
}

func (e jsonPathError) String() string { return e.err.Error() }

func (e jsonPathError) Equal(any) bool { return false }

type jsonPathStep struct {
	key     string
	index   int
	isIndex bool
}

func parseJSONPath(path string) ([]jsonPathStep, error) {
	if path == "" || path[0] != '$' {
		return nil, fmt.Errorf(`jsonpath %q: must start with "$"`, path)
	}

	var steps []jsonPathStep
	for i := 1; i < len(path); {
		switch path[i] {
		case '.':
			i++
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			if start == i {
				return nil, fmt.Errorf("jsonpath %q: empty key", path)
			}
			steps = append(steps, jsonPathStep{key: path[start:i]})

		case '[':
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("jsonpath %q: missing ]", path)
			}
			text := strings.TrimSpace(path[i+1 : i+end])
			index, err := strconv.Atoi(text)
			if err != nil || index < 0 {
				return nil, fmt.Errorf("jsonpath %q: invalid index %q", path, text)
			}
			steps = append(steps, jsonPathStep{index: index, isIndex: true})
			i += end + 1

		default:
			return nil, fmt.Errorf("jsonpath %q: unexpected character %q", path, path[i])
		}
	}
	return steps, nil
}

func resolveJSONPath(root any, steps []jsonPathStep) (any, bool) {
	current := root
	for _, step := range steps {
		if step.isIndex {
			items, ok := current.([]any)
			if !ok || step.index >= len(items) {
				return nil, false
			}
			current = items[step.index]
			continue
		}

		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if current, ok = object[step.key]; !ok {
			return nil, false
		}
	}
	return current, true
}

// equalValues compares values with Equal: numbers match across Go numeric
// types, and slices, arrays and string-keyed maps compare element by element.
func equalValues(got, want any) bool {
	if gotNumber, ok := numberValue(got); ok {
		wantNumber, ok := numberValue(want)
		return ok && gotNumber.Cmp(wantNumber) == 0
	}

	gotValue, wantValue := reflect.ValueOf(got), reflect.ValueOf(want)
	switch gotValue.Kind() {
	case reflect.Slice, reflect.Array:
		if wantValue.Kind() != reflect.Slice && wantValue.Kind() != reflect.Array {
			return false
		}
		if gotValue.Len() != wantValue.Len() {
			return false
		}
		for i := range gotValue.Len() {
			if !equalValues(gotValue.Index(i).Interface(), wantValue.Index(i).Interface()) {
				return false
			}
		}
		return true

	case reflect.Map:
		if wantValue.Kind() != reflect.Map {
			return false
		}
		if gotValue.Type().Key().Kind() != reflect.String || wantValue.Type().Key().Kind() != reflect.String {
			return reflect.DeepEqual(got, want)
		}
		if gotValue.Len() != wantValue.Len() {
			return false
		}
		for _, key := range gotValue.MapKeys() {
			wantItem := wantValue.MapIndex(reflect.ValueOf(key.String()).Convert(wantValue.Type().Key()))
			if !wantItem.IsValid() || !equalValues(gotValue.MapIndex(key).Interface(), wantItem.Interface()) {
				return false
			}
		}
		return true
	}

	return reflect.DeepEqual(got, want)
}

// numberValue converts JSON numbers and Go integers, uints and floats to an
// exact rational, so large integers always compare exactly.
func numberValue(value any) (*big.Rat, bool) {
	if number, ok := value.(jsonNumber); ok {
		rational, ok := new(big.Rat).SetString(string(number))
		return rational, ok
	}

	valueReflect := reflect.ValueOf(value)
	switch valueReflect.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return new(big.Rat).SetInt64(valueReflect.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return new(big.Rat).SetInt(new(big.Int).SetUint64(valueReflect.Uint())), true
	case reflect.Float32, reflect.Float64:
		rational, ok := new(big.Rat).SetString(strconv.FormatFloat(valueReflect.Float(), 'g', -1, 64))
		return rational, ok
	}
	return nil, false
}
