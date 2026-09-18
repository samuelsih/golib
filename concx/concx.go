package concx

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type ForEachOpts struct {
	// MaxGoroutines is the upper bound on the number of worker goroutines.
	MaxGoroutines int
}

// DefaultForEachOpts holds the default options for ForEach.
var DefaultForEachOpts = ForEachOpts{
	MaxGoroutines: runtime.GOMAXPROCS(0),
}

// ForEach applies handler to every element of values concurrently and returns
// once all calls have completed.
func ForEach[T any](values []T, handler func(T), opts ForEachOpts) {
	n := len(values)
	if n == 0 {
		return
	}

	if handler == nil {
		panic("concx: ForEach: nil handler")
	}

	workers := opts.MaxGoroutines
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	workers = min(workers, n)

	var cursor atomic.Int64
	task := func() {
		for i := int(cursor.Add(1)) - 1; i < n; i = int(cursor.Add(1)) - 1 {
			handler(values[i])
		}
	}

	var wg sync.WaitGroup
	for range workers {
		wg.Go(task)
	}
	wg.Wait()
}

type ForEachResOpts struct {
	// MaxGoroutines is the upper bound on the number of worker goroutines.
	MaxGoroutines int
}

// DefaultForEachResOpts holds the default options for ForEachRes.
var DefaultForEachResOpts = ForEachResOpts{
	MaxGoroutines: runtime.GOMAXPROCS(0),
}

// ForEachRes applies handler to every element of values concurrently and returns
// the results once all calls have completed.
func ForEachRes[T any](values []T, handler func(T) T, opts ForEachResOpts) []T {
	n := len(values)
	if n == 0 {
		return nil
	}

	if handler == nil {
		panic("concx: ForEachRes: nil handler")
	}

	workers := opts.MaxGoroutines
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	workers = min(workers, n)

	var (
		cursor atomic.Int64
		wg     sync.WaitGroup
	)

	results := make([]T, n)

	task := func() {
		for i := int(cursor.Add(1)) - 1; i < n; i = int(cursor.Add(1)) - 1 {
			results[i] = handler(values[i])
		}
	}

	for range workers {
		wg.Go(task)
	}
	wg.Wait()

	return results
}
