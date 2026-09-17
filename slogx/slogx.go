package slogx

import (
	"log/slog"
	"runtime"
)

const (
	callerKey = "caller"
	errorKey  = "error"
)

// Caller returns the function name of its caller under the given key.
func Caller(key string) slog.Attr {
	return callerAttr(key)
}

// CallerAttr calls Caller function and uses "caller" as the key.
func CallerAttr() slog.Attr {
	return callerAttr(callerKey)
}

// Error returns the error message under the given key.
func Error(key string, value error) slog.Attr {
	return slog.String(key, value.Error())
}

// ErrorAttr calls Error function and uses "error" as the key.
func ErrorAttr(value error) slog.Attr {
	return Error(errorKey, value)
}

// callerAttr captures the frame that called Caller or CallerAttr. The skip
// accounts for runtime.Callers, callerAttr itself, and the exported wrapper.
func callerAttr(key string) slog.Attr {
	var pc [1]uintptr
	_ = runtime.Callers(3, pc[:])
	frame, _ := runtime.CallersFrames(pc[:]).Next()
	return slog.String(key, frame.Function)
}
