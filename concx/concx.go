package concx

import "sync"

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
