package reflectx

import (
	"encoding"
	"net/url"
	"reflect"
	"strconv"
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

// GetTagValueAs returns the value of key in sf's struct tag parsed into T.
// Supports encoding.TextUnmarshaler, built-in primitives, url.URL, time.Duration and time.Location.
func GetTagValueAs[T any](sf reflect.StructField, key string) T {
	var out T

	raw, ok := sf.Tag.Lookup(key)
	if !ok {
		return out
	}

	if u, ok := any(&out).(encoding.TextUnmarshaler); ok {
		if err := u.UnmarshalText([]byte(raw)); err != nil {
			var zero T
			return zero
		}

		return out
	}

	switch p := any(&out).(type) {
	case *string:
		*p = raw
	case *bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return out
		}

		*p = b
	case *int:
		n, err := strconv.ParseInt(raw, 10, strconv.IntSize)
		if err != nil {
			return out
		}

		*p = int(n)
	case *int8:
		n, err := strconv.ParseInt(raw, 10, 8)
		if err != nil {
			return out
		}

		*p = int8(n)
	case *int16:
		n, err := strconv.ParseInt(raw, 10, 16)
		if err != nil {
			return out
		}

		*p = int16(n)
	case *int32:
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return out
		}

		*p = int32(n)
	case *int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return out
		}

		*p = n
	case *uint:
		n, err := strconv.ParseUint(raw, 10, strconv.IntSize)
		if err != nil {
			return out
		}

		*p = uint(n)
	case *uint8:
		n, err := strconv.ParseUint(raw, 10, 8)
		if err != nil {
			return out
		}

		*p = uint8(n)
	case *uint16:
		n, err := strconv.ParseUint(raw, 10, 16)
		if err != nil {
			return out
		}

		*p = uint16(n)
	case *uint32:
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return out
		}

		*p = uint32(n)
	case *uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return out
		}

		*p = n
	case *uintptr:
		n, err := strconv.ParseUint(raw, 10, strconv.IntSize)
		if err != nil {
			return out
		}

		*p = uintptr(n)
	case *float32:
		f, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return out
		}

		*p = float32(f)
	case *float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return out
		}

		*p = f
	case *complex64:
		c, err := strconv.ParseComplex(raw, 64)
		if err != nil {
			return out
		}

		*p = complex64(c)
	case *complex128:
		c, err := strconv.ParseComplex(raw, 128)
		if err != nil {
			return out
		}

		*p = c
	case *time.Duration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return out
		}

		*p = d
	case *time.Location:
		loc, err := time.LoadLocation(raw)
		if err != nil {
			return out
		}

		*p = *loc
	case **time.Location:
		loc, err := time.LoadLocation(raw)
		if err != nil {
			return out
		}

		*p = loc
	case *url.URL:
		u, err := url.Parse(raw)
		if err != nil {
			return out
		}

		*p = *u
	case **url.URL:
		u, err := url.Parse(raw)
		if err != nil {
			return out
		}

		*p = u
	}

	return out
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
