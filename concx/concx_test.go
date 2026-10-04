package concx

import (
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
