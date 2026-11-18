package concx

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/samuelsih/golib/assert"
)

func TestForEach(t *testing.T) {
	const n = 100

	values := make([]int, n)
	counts := make([]atomic.Int64, n)
	for i := range values {
		values[i] = i
	}

	ForEach(values, func(v int) {
		counts[v].Add(1)
	})

	for i := range counts {
		assert.Equal(t, counts[i].Load(), int64(1))
	}
}

func TestForEachEmpty(t *testing.T) {
	var called atomic.Bool
	handler := func(int) { called.Store(true) }

	ForEach([]int{}, handler)
	assert.False(t, called.Load())

	ForEach[int](nil, handler)
	assert.False(t, called.Load())
}

func TestForEachMutates(t *testing.T) {
	values := []int{1, 2, 3, 4}
	ptrs := make([]*int, len(values))
	for i := range values {
		ptrs[i] = &values[i]
	}

	ForEach(ptrs, func(v *int) { *v *= 2 })

	assert.Equal(t, values, []int{2, 4, 6, 8})
}

func TestForEachConcurrent(t *testing.T) {
	const n = 8

	values := make([]int, n)
	release := make(chan struct{})
	var (
		entered   sync.WaitGroup
		closeOnce sync.Once
	)
	entered.Add(n)

	unblock := func() { closeOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)

	go func() {
		entered.Wait()
		unblock()
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		ForEach(values, func(int) {
			entered.Done()
			<-release
		})
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("ForEach did not run all handlers concurrently")
	}
}

func TestForEachNilHandlerPanics(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()

	ForEach([]int{1}, nil)
}

func TestForEachRes(t *testing.T) {
	const n = 100

	values := make([]int, n)
	counts := make([]atomic.Int64, n)
	for i := range values {
		values[i] = i
	}

	results := ForEachRes(values, func(v int) int {
		counts[v].Add(1)
		return v * v
	})

	assert.Equal(t, len(results), n)
	for i := range results {
		assert.Equal(t, counts[i].Load(), int64(1))
		assert.Equal(t, results[i], i*i)
	}
}

func TestForEachResEmpty(t *testing.T) {
	var called atomic.Bool
	handler := func(v int) int {
		called.Store(true)
		return v
	}

	results := ForEachRes([]int{}, handler)
	assert.Nil(t, results)
	assert.False(t, called.Load())

	results = ForEachRes[int, int](nil, handler)
	assert.Nil(t, results)
	assert.False(t, called.Load())
}

func TestForEachResTypeChange(t *testing.T) {
	results := ForEachRes(
		[]string{"go", "is", "fun"},
		strings.ToUpper,
	)

	assert.Equal(t, results, []string{"GO", "IS", "FUN"})
}

func TestForEachResConcurrent(t *testing.T) {
	const n = 8

	values := make([]int, n)
	for i := range values {
		values[i] = i
	}

	release := make(chan struct{})
	var (
		entered   sync.WaitGroup
		closeOnce sync.Once
	)
	entered.Add(n)

	unblock := func() { closeOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)

	go func() {
		entered.Wait()
		unblock()
	}()

	done := make(chan []int, 1)
	go func() {
		done <- ForEachRes(values, func(v int) int {
			entered.Done()
			<-release
			return v * 2
		})
	}()

	select {
	case results := <-done:
		for i := range results {
			assert.Equal(t, results[i], i*2)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ForEachRes did not run all handlers concurrently")
	}
}

func TestForEachResNilHandlerPanics(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()

	ForEachRes[int, int]([]int{1}, nil)
}

func TestFence(t *testing.T) {
	t.Run("wait after release", func(t *testing.T) {
		var f Fence
		f.Release()

		assert.NoError(t, f.Wait(t.Context()))
	})

	t.Run("wait blocks until release", func(t *testing.T) {
		var f Fence

		done := make(chan error, 1)
		go func() { done <- f.Wait(t.Context()) }()

		select {
		case err := <-done:
			t.Fatalf("Wait returned before Release: %v", err)
		case <-time.After(10 * time.Millisecond):
		}

		f.Release()

		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Wait did not return after Release")
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		var f Fence

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		assert.ErrorIs(t, f.Wait(ctx), context.Canceled)
	})

	t.Run("released channel", func(t *testing.T) {
		var f Fence
		ch := f.Released()

		select {
		case <-ch:
			t.Fatal("Released channel closed before Release")
		default:
		}

		f.Release()

		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("Released channel not closed after Release")
		}
	})
}

func TestSleep(t *testing.T) {
	t.Run("completes after duration", func(t *testing.T) {
		const duration = 20 * time.Millisecond

		start := time.Now()
		err := Sleep(t.Context(), duration)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, elapsed >= duration)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		start := time.Now()
		err := Sleep(ctx, time.Hour)
		elapsed := time.Since(start)

		assert.ErrorIs(t, err, context.Canceled)
		assert.True(t, elapsed < time.Second)
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		err := Sleep(ctx, time.Hour)

		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
