package concx

import (
	"context"
	"sync"
	"time"
)

// Fence lets one goroutine wait until another goroutine finishes its work.
type Fence struct {
	_       noCopy
	create  sync.Once
	release sync.Once
	wait    chan struct{}
}

// Release marks the Fence as released, waking all goroutines blocked in Wait.
// It is safe to call Release more than once.
func (f *Fence) Release() {
	f.ensureInit()
	f.release.Do(func() {
		close(f.wait)
	})
}

// Released returns a channel that is closed when Release is called.
// Receiving from the channel blocks until Release is called.
func (f *Fence) Released() <-chan struct{} {
	f.ensureInit()
	return f.wait
}

// Wait blocks until Release is called or ctx is done. It returns nil when the
// Fence is released, or ctx.Err() when the context ends first.
func (f *Fence) Wait(ctx context.Context) error {
	f.ensureInit()
	select {
	case <-f.Released():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *Fence) ensureInit() {
	f.create.Do(func() {
		f.wait = make(chan struct{})
	})
}

// ForEach applies handler to every element of values concurrently, using one
// goroutine per element, and returns once all calls have completed.
func ForEach[T any](values []T, handler func(T)) {
	if len(values) == 0 {
		return
	}

	if handler == nil {
		panic("concx: ForEach: nil handler")
	}

	var wg sync.WaitGroup
	for _, v := range values {
		wg.Go(func() {
			handler(v)
		})
	}
	wg.Wait()
}

// ForEachRes applies handler to every element of values concurrently and returns
// the results once all calls have completed.
func ForEachRes[T, U any](values []T, handler func(T) U) []U {
	n := len(values)
	if n == 0 {
		return nil
	}

	if handler == nil {
		panic("concx: ForEachRes: nil handler")
	}

	results := make([]U, n)

	var wg sync.WaitGroup
	for i, v := range values {
		wg.Go(func() {
			results[i] = handler(v)
		})
	}
	wg.Wait()

	return results
}

// Sleep waits for duration or until ctx is done, whichever happens first. It
// returns ctx.Err() if ctx ends before duration.
func Sleep(ctx context.Context, duration time.Duration) error {
	t := time.NewTicker(duration)
	defer t.Stop()

	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type noCopy struct{}

// Lock is a no-op used by the copylocks checker in go vet.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
