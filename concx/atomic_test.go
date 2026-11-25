package concx

import (
	"sync"
	"testing"

	"github.com/samuelsih/golib/assert"
)

func TestNewAtomicLoad(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		a := NewAtomic(42)
		assert.Equal(t, a.Load(), 42)
	})

	t.Run("string", func(t *testing.T) {
		a := NewAtomic("hello")
		assert.Equal(t, a.Load(), "hello")
	})

	t.Run("struct", func(t *testing.T) {
		type point struct{ X, Y int }
		a := NewAtomic(point{X: 1, Y: 2})
		assert.Equal(t, a.Load(), point{X: 1, Y: 2})
	})

	t.Run("nil pointer", func(t *testing.T) {
		a := NewAtomic[*int](nil)
		assert.Nil(t, a.Load())
	})
}

func TestAtomicTryLoad(t *testing.T) {
	t.Run("initialized", func(t *testing.T) {
		a := NewAtomic(42)

		v, ok := a.TryLoad()

		assert.True(t, ok)
		assert.Equal(t, v, 42)
	})

	t.Run("uninitialized returns zero value", func(t *testing.T) {
		var a Atomic[int]

		v, ok := a.TryLoad()

		assert.False(t, ok)
		assert.Equal(t, v, 0)
	})

	t.Run("zero value after store", func(t *testing.T) {
		var a Atomic[string]
		a.Store("hello")

		v, ok := a.TryLoad()

		assert.True(t, ok)
		assert.Equal(t, v, "hello")
	})

	t.Run("stored nil pointer is loaded", func(t *testing.T) {
		a := NewAtomic[*int](nil)

		v, ok := a.TryLoad()

		assert.True(t, ok)
		assert.Nil(t, v)
	})
}

func TestAtomicStore(t *testing.T) {
	a := NewAtomic(1)

	a.Store(2)
	assert.Equal(t, a.Load(), 2)

	a.Store(3)
	assert.Equal(t, a.Load(), 3)
}

func TestAtomicSwap(t *testing.T) {
	a := NewAtomic("old")

	old := a.Swap("new")

	assert.Equal(t, old, "old")
	assert.Equal(t, a.Load(), "new")

	old = a.Swap("newest")
	assert.Equal(t, old, "new")
	assert.Equal(t, a.Load(), "newest")
}

func TestAtomicCompareAndSwap(t *testing.T) {
	t.Run("matching old value swaps", func(t *testing.T) {
		a := NewAtomic(1)

		swapped := a.CompareAndSwap(1, 2)

		assert.True(t, swapped)
		assert.Equal(t, a.Load(), 2)
	})

	t.Run("different old value does not swap", func(t *testing.T) {
		a := NewAtomic(1)

		swapped := a.CompareAndSwap(99, 2)

		assert.False(t, swapped)
		assert.Equal(t, a.Load(), 1)
	})

	t.Run("struct value", func(t *testing.T) {
		type point struct{ X, Y int }
		a := NewAtomic(point{X: 1, Y: 2})

		assert.True(t, a.CompareAndSwap(point{X: 1, Y: 2}, point{X: 3, Y: 4}))
		assert.Equal(t, a.Load(), point{X: 3, Y: 4})
		assert.False(t, a.CompareAndSwap(point{X: 1, Y: 2}, point{X: 5, Y: 6}))
		assert.Equal(t, a.Load(), point{X: 3, Y: 4})
	})
}

func TestAtomicConcurrent(t *testing.T) {
	t.Run("compare and swap counter", func(t *testing.T) {
		const (
			goroutines = 8
			increments = 1000
		)

		counter := NewAtomic(0)

		var wg sync.WaitGroup
		for range goroutines {
			wg.Go(func() {
				for range increments {
					for {
						old := counter.Load()
						if counter.CompareAndSwap(old, old+1) {
							break
						}
					}
				}
			})
		}
		wg.Wait()

		assert.Equal(t, counter.Load(), goroutines*increments)
	})

	t.Run("store and load", func(t *testing.T) {
		const iterations = 1000

		a := NewAtomic(0)
		done := make(chan struct{})

		var wg sync.WaitGroup
		wg.Go(func() {
			defer close(done)
			for i := range iterations {
				a.Store(i)
			}
		})

		wg.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
					v := a.Load()
					assert.True(t, v >= 0 && v < iterations)
				}
			}
		})
		wg.Wait()
	})
}
