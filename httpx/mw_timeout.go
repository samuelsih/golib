package httpx

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type (
	timeoutCtx       struct{}
	resetTimeoutFunc func(time.Duration)
	cancelFunc       func()
)

var (
	ErrTimeoutExceeded = errors.New("timeout exceeded")
	timeoutCtxKey      = timeoutCtx{}
)

// MiddlewareTimeout is a middleware that cancels ctx after a given timeout.
func MiddlewareTimeout(duration time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			ctx := r.Context()

			if reset, ok := ctx.Value(timeoutCtxKey).(resetTimeoutFunc); ok {
				reset(duration)
				return next(w, r)
			}

			ctx, cancel, reset := withTimeout(ctx, duration)
			ctx = context.WithValue(ctx, timeoutCtxKey, reset)
			defer cancel()

			err := next(w, r.WithContext(ctx))
			if err == nil {
				err = ctx.Err()
			}

			return err
		}
	}
}

type timeoutStateContext struct {
	context.Context
	// Deadline is written only during the synchronous middleware chain,
	// before the handler runs; read only afterwards.
	// Single-goroutine by construction, so no synchronization is needed.
	deadline time.Time
}

// Err implements context.Context.Err() interface.
func (ctx *timeoutStateContext) Err() error {
	if errors.Is(context.Cause(ctx.Context), ErrTimeoutExceeded) {
		return context.DeadlineExceeded
	}

	return ctx.Context.Err()
}

// Err implements context.Context.Deadline() interface.
func (ctx *timeoutStateContext) Deadline() (time.Time, bool) {
	parent, ok := ctx.Context.Deadline()

	if ok && parent.Before(ctx.deadline) {
		return parent, true
	}

	return ctx.deadline, true
}

func withTimeout(parent context.Context, timeout time.Duration) (context.Context, cancelFunc, resetTimeoutFunc) {
	parent, parentCancel := context.WithCancelCause(parent)

	ctx := &timeoutStateContext{
		Context:  parent,
		deadline: time.Now().Add(timeout),
	}

	// timeout
	timer := time.AfterFunc(timeout, func() {
		parentCancel(ErrTimeoutExceeded)
	})

	// normal cancellation
	cancel := func() {
		timer.Stop()
		parentCancel(nil)
	}

	// changes the timeout
	reset := func(timeout time.Duration) {
		timer.Reset(timeout)
		ctx.deadline = time.Now().Add(timeout)
	}

	return ctx, cancel, reset
}
