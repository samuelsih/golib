package assert

import (
	"errors"
	"reflect"
	"testing"
)

func Equal(t *testing.T, got, want any) {
	t.Helper()
	if !isEqual(got, want) {
		t.Errorf("got: %v; want: %v", got, want)
	}
}

func NotEqual(t *testing.T, got, want any) {
	t.Helper()
	if isEqual(got, want) {
		t.Errorf("got: %v; expected values to be different", got)
	}
}

func True(t *testing.T, got bool) {
	t.Helper()
	if !got {
		t.Errorf("got: false; want: true")
	}
}

func False(t *testing.T, got bool) {
	t.Helper()
	if got {
		t.Errorf("got: true; want: false")
	}
}

func Nil(t *testing.T, got any) {
	t.Helper()
	if !isNil(got) {
		t.Errorf("got: %v; want: nil", got)
	}
}

func NotNil(t *testing.T, got any) {
	t.Helper()
	if isNil(got) {
		t.Errorf("got: nil; want: non-nil")
	}
}

func ErrorIs(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Errorf("got: %v; want: %v", got, want)
	}
}

func ErrorAs(t *testing.T, got error, target any) {
	t.Helper()
	if got == nil {
		t.Errorf("got: nil; want assignable to: %T", target)
		return
	}

	if !errors.As(got, target) {
		t.Errorf("got: %v; want assignable to: %T", got, target)
	}
}

func ErrorAsType[E error](t *testing.T, got error) E {
	t.Helper()
	target, ok := errors.AsType[E](got)
	if !ok {
		var zero E
		t.Errorf("got: %v; want assignable to: %T", got, zero)
	}
	return target
}

func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("got: %v; want nil-error", err)
	}
}

func isEqual(got, want any) bool {
	gotNil, wantNil := isNil(got), isNil(want)
	if gotNil || wantNil {
		return gotNil && wantNil
	}
	if equal, ok := callEqualMethod(got, want); ok {
		return equal
	}
	return equalValues(got, want)
}

// callEqualMethod invokes got's Equal method when its parameter can hold
// want, letting types customize their own comparison.
func callEqualMethod(got, want any) (equal, ok bool) {
	method := reflect.ValueOf(got).MethodByName("Equal")
	if !method.IsValid() {
		return false, false
	}

	methodType := method.Type()
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 || methodType.Out(0).Kind() != reflect.Bool {
		return false, false
	}

	wantValue := reflect.ValueOf(want)
	if !wantValue.IsValid() || !wantValue.Type().AssignableTo(methodType.In(0)) {
		return false, false
	}

	return method.Call([]reflect.Value{wantValue})[0].Bool(), true
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return rv.IsNil()
	}
	return false
}
