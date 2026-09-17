package slogx

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"testing"

	"github.com/samuelsih/golib/assert"
)

func TestError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want slog.Attr
	}{
		{
			name: "plain error",
			err:  errors.New("something went wrong"),
			want: slog.String("err", "something went wrong"),
		},
		{
			name: "wrapped error",
			err:  fmt.Errorf("request failed: %w", errors.New("connection reset")),
			want: slog.String("err", "request failed: connection reset"),
		},
		{
			name: "custom error type",
			err:  testError{},
			want: slog.String("err", "custom error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, Error("err", tt.err), tt.want)
		})
	}
}

func TestErrorAttr(t *testing.T) {
	assert.Equal(t, ErrorAttr(errors.New("something went wrong")), slog.String("error", "something went wrong"))
}

type testError struct{}

func (testError) Error() string { return "custom error" }

func TestCaller(t *testing.T) {
	assert.Equal(t, Caller("caller"), slog.String("caller", funcName(TestCaller)))
}

func TestCallerFromFunction(t *testing.T) {
	assert.Equal(t, callerFromFunction(), slog.String("caller", funcName(callerFromFunction)))
}

func callerFromFunction() slog.Attr {
	return Caller("caller")
}

type callerHolder struct{}

func (callerHolder) Caller() slog.Attr {
	return Caller("caller")
}

func TestCallerFromMethod(t *testing.T) {
	assert.Equal(t, callerHolder{}.Caller(), slog.String("caller", funcName(callerHolder.Caller)))
}

func TestCallerFromClosure(t *testing.T) {
	closure := func() slog.Attr { return Caller("caller") }
	assert.Equal(t, closure(), slog.String("caller", funcName(closure)))
}

func TestCallerAttr(t *testing.T) {
	assert.Equal(t, CallerAttr(), slog.String("caller", funcName(TestCallerAttr)))
}

func TestCallerAttrFromFunction(t *testing.T) {
	assert.Equal(t, callerAttrFromFunction(), slog.String("caller", funcName(callerAttrFromFunction)))
}

func callerAttrFromFunction() slog.Attr {
	return CallerAttr()
}

func funcName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}
