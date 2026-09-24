package reflectx

import (
	"encoding"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// IsEmptyValue check if reflect value has zero value based on its type.
func IsEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Interface, reflect.Pointer, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.Struct:
		return v.IsZero()
	}

	return false
}

// GetTagValueAs parses key from sf's struct tag into T and reports whether the
// key is present. On a parse error the zero value is returned with true.
func GetTagValueAs[T any](sf reflect.StructField, key string) (T, bool) {
	var out T

	raw, ok := sf.Tag.Lookup(key)
	if !ok {
		return out, false
	}

	if u, ok := any(&out).(encoding.TextUnmarshaler); ok {
		if err := u.UnmarshalText([]byte(raw)); err != nil {
			var zero T
			return zero, true
		}

		return out, true
	}

	switch p := any(&out).(type) {
	case *string:
		*p = raw
	case *bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return out, true
		}

		*p = b
	case *int:
		n, err := strconv.ParseInt(raw, 10, strconv.IntSize)
		if err != nil {
			return out, true
		}

		*p = int(n)
	case *int8:
		n, err := strconv.ParseInt(raw, 10, 8)
		if err != nil {
			return out, true
		}

		*p = int8(n)
	case *int16:
		n, err := strconv.ParseInt(raw, 10, 16)
		if err != nil {
			return out, true
		}

		*p = int16(n)
	case *int32:
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return out, true
		}

		*p = int32(n)
	case *int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return out, true
		}

		*p = n
	case *uint:
		n, err := strconv.ParseUint(raw, 10, strconv.IntSize)
		if err != nil {
			return out, true
		}

		*p = uint(n)
	case *uint8:
		n, err := strconv.ParseUint(raw, 10, 8)
		if err != nil {
			return out, true
		}

		*p = uint8(n)
	case *uint16:
		n, err := strconv.ParseUint(raw, 10, 16)
		if err != nil {
			return out, true
		}

		*p = uint16(n)
	case *uint32:
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return out, true
		}

		*p = uint32(n)
	case *uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return out, true
		}

		*p = n
	case *uintptr:
		n, err := strconv.ParseUint(raw, 10, strconv.IntSize)
		if err != nil {
			return out, true
		}

		*p = uintptr(n)
	case *float32:
		f, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return out, true
		}

		*p = float32(f)
	case *float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return out, true
		}

		*p = f
	case *complex64:
		c, err := strconv.ParseComplex(raw, 64)
		if err != nil {
			return out, true
		}

		*p = complex64(c)
	case *complex128:
		c, err := strconv.ParseComplex(raw, 128)
		if err != nil {
			return out, true
		}

		*p = c
	case *time.Duration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return out, true
		}

		*p = d
	case *time.Location:
		loc, err := time.LoadLocation(raw)
		if err != nil {
			return out, true
		}

		*p = *loc
	case **time.Location:
		loc, err := time.LoadLocation(raw)
		if err != nil {
			return out, true
		}

		*p = loc
	case *url.URL:
		u, err := url.Parse(raw)
		if err != nil {
			return out, true
		}

		*p = *u
	case **url.URL:
		u, err := url.Parse(raw)
		if err != nil {
			return out, true
		}

		*p = u
	}

	return out, true
}

// Indirect returns t with every leading pointer level removed.
func Indirect(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t
}

// JSONField is the parsed json tag of a struct field.
type JSONField struct {
	Name string
	Omit bool
	Skip bool
}

// JSONFieldOf parses the json tag of sf following encoding/json rules. A
// missing tag keeps the field name; a "-" tag marks Skip.
func JSONFieldOf(sf reflect.StructField) JSONField {
	tag, ok := sf.Tag.Lookup("json")
	if !ok {
		return JSONField{Name: sf.Name}
	}
	if tag == "-" {
		return JSONField{Skip: true}
	}

	parts := strings.Split(tag, ",")
	field := JSONField{Name: parts[0]}
	if field.Name == "" {
		field.Name = sf.Name
	}

	for _, opt := range parts[1:] {
		if opt == "omitempty" || opt == "omitzero" {
			field.Omit = true
		}
	}

	return field
}

// WalkStruct calls fn for every field of t, dereferencing pointers and walking
// nested structs. Recursive types are cut off for each branch.
func WalkStruct(t reflect.Type, fn func(sf reflect.StructField)) {
	if t == nil || fn == nil {
		return
	}
	walkStruct(t, fn, map[reflect.Type]struct{}{})
}

func walkStruct(t reflect.Type, fn func(sf reflect.StructField), path map[reflect.Type]struct{}) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	if _, ok := path[t]; ok {
		return
	}

	path[t] = struct{}{}

	for sf := range t.Fields() {
		fn(sf)
		walkStruct(sf.Type, fn, path)
	}

	delete(path, t)
}

// WalkFields calls fn for every exported field of t, recursing into anonymous
// fields for which flatten returns true. Recursive types are cut off per branch.
func WalkFields(t reflect.Type, flatten func(reflect.StructField) bool, fn func(reflect.StructField)) {
	if fn == nil {
		return
	}

	walkFields(Indirect(t), flatten, fn, map[reflect.Type]struct{}{})
}

func walkFields(t reflect.Type, flatten func(reflect.StructField) bool, fn func(reflect.StructField), path map[reflect.Type]struct{}) {
	if t == nil || t.Kind() != reflect.Struct {
		return
	}
	if _, ok := path[t]; ok {
		return
	}

	path[t] = struct{}{}
	defer delete(path, t)

	for sf := range t.Fields() {
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}
		if sf.Anonymous && flatten != nil && flatten(sf) {
			walkFields(Indirect(sf.Type), flatten, fn, path)
			continue
		}

		fn(sf)
	}
}
