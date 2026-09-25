package assert

import (
	"errors"
	"fmt"
	"testing"
	"unsafe"
)

type byID struct {
	id   int
	name string
}

func (a byID) Equal(b byID) bool {
	return a.id == b.id
}

type ptrEqual struct {
	id int
}

func (a *ptrEqual) Equal(b *ptrEqual) bool {
	return a.id == b.id
}

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func TestIsEqual(t *testing.T) {
	var nilPtr *int

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{name: "both nil", got: isEqual(nil, nil), want: true},
		{name: "equal typed nil pointers", got: isEqual(nilPtr, (*int)(nil)), want: true},
		{name: "typed nil and allocated pointer", got: isEqual(nilPtr, new(int)), want: false},
		{name: "nil and empty slice", got: isEqual([]int(nil), []int{}), want: false},
		{name: "equal slices", got: isEqual([]int{1, 2}, []int{1, 2}), want: true},
		{name: "different slices", got: isEqual([]int{1, 2}, []int{1, 3}), want: false},
		{name: "equal maps", got: isEqual(map[string]int{"a": 1}, map[string]int{"a": 1}), want: true},
		{name: "different maps", got: isEqual(map[string]int{"a": 1}, map[string]int{"b": 1}), want: false},
		{name: "Equal method ignores other fields", got: isEqual(byID{id: 1, name: "mario"}, byID{id: 1, name: "luigi"}), want: true},
		{name: "Equal method rejects different id", got: isEqual(byID{id: 1, name: "mario"}, byID{id: 2, name: "mario"}), want: false},
		{name: "typed nil got with Equal method", got: isEqual((*ptrEqual)(nil), &ptrEqual{id: 1}), want: false},
		{name: "typed nil want with Equal method", got: isEqual(&ptrEqual{id: 1}, (*ptrEqual)(nil)), want: false},
		{name: "equal pointers with Equal method", got: isEqual(&ptrEqual{id: 1}, &ptrEqual{id: 1}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got: %v; want: %v", tt.got, tt.want)
			}
		})
	}
}

func TestIsNil(t *testing.T) {
	var (
		ptr         *int
		fn          func()
		slice       []int
		m           map[string]int
		ch          chan int
		err         error
		typedNilErr error = (*customError)(nil)
	)

	tests := []struct {
		name string
		got  any
		want bool
	}{
		{name: "nil interface", got: nil, want: true},
		{name: "typed nil pointer", got: ptr, want: true},
		{name: "typed nil func", got: fn, want: true},
		{name: "typed nil slice", got: slice, want: true},
		{name: "typed nil map", got: m, want: true},
		{name: "typed nil chan", got: ch, want: true},
		{name: "typed nil error", got: err, want: true},
		{name: "error wrapping typed nil", got: typedNilErr, want: true},
		{name: "nil unsafe pointer", got: unsafe.Pointer(nil), want: true},
		{name: "non-nil int", got: 0, want: false},
		{name: "non-nil string", got: "", want: false},
		{name: "non-nil struct", got: struct{}{}, want: false},
		{name: "non-nil pointer", got: new(int), want: false},
		{name: "non-nil func", got: func() {}, want: false},
		{name: "non-nil slice", got: []int{}, want: false},
		{name: "non-nil map", got: map[string]int{}, want: false},
		{name: "non-nil chan", got: make(chan int), want: false},
		{name: "non-nil unsafe pointer", got: unsafe.Pointer(new(int)), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNil(tt.got); got != tt.want {
				t.Errorf("got: %v; want: %v", got, tt.want)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	Equal(t, 42, 42)
	Equal(t, "mario", "mario")
	Equal(t, []int{1, 2}, []int{1, 2})
	Equal(t, (*int)(nil), (*int)(nil))
	Equal(t, byID{id: 1, name: "mario"}, byID{id: 1, name: "luigi"})
}

func TestNotEqual(t *testing.T) {
	NotEqual(t, 42, 43)
	NotEqual(t, "mario", "luigi")
	NotEqual(t, byID{id: 1, name: "mario"}, byID{id: 2, name: "mario"})
}

func TestTrue(t *testing.T) {
	True(t, true)
}

func TestFalse(t *testing.T) {
	False(t, false)
}

func TestNil(t *testing.T) {
	var (
		ptr   *int
		fn    func()
		slice []int
		err   error = (*customError)(nil)
	)

	Nil(t, nil)
	Nil(t, ptr)
	Nil(t, fn)
	Nil(t, slice)
	Nil(t, err)
}

func TestNotNil(t *testing.T) {
	var err error = &customError{msg: "boom"}

	NotNil(t, 0)
	NotNil(t, "")
	NotNil(t, new(int))
	NotNil(t, []int{})
	NotNil(t, map[string]int{})
	NotNil(t, err)
}

func TestErrorIs(t *testing.T) {
	sentinel := errors.New("sentinel")

	ErrorIs(t, sentinel, sentinel)
	ErrorIs(t, fmt.Errorf("wrap: %w", sentinel), sentinel)
	ErrorIs(t, errors.Join(errors.New("other"), sentinel), sentinel)
}

func TestErrorAs(t *testing.T) {
	var target *customError

	ErrorAs(t, fmt.Errorf("wrap: %w", &customError{msg: "boom"}), &target)
	if target == nil || target.msg != "boom" {
		t.Errorf("got: %v; want msg: boom", target)
	}
}

func TestErrorAsType(t *testing.T) {
	target := ErrorAsType[*customError](t, fmt.Errorf("wrap: %w", &customError{msg: "boom"}))
	if target == nil || target.msg != "boom" {
		t.Errorf("got: %v; want msg: boom", target)
	}
}

func TestNoError(t *testing.T) {
	NoError(t, nil)
}
