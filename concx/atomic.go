package concx

import "sync/atomic"

// Atomic is a type-safe wrapper around atomic.Value.
type Atomic[T any] struct {
	val atomic.Value
}

// NewAtomic returns a new Atomic holding val.
func NewAtomic[T any](val T) *Atomic[T] {
	v := Atomic[T]{}
	v.val.Store(val)
	return &v
}

// Load returns the current value and panics if the Atomic was never stored to.
func (v *Atomic[T]) Load() T {
	value, ok := v.TryLoad()
	if !ok {
		panic("atomic.Value is a nil")
	}
	return value
}

// TryLoad returns the current value and true, or the zero value and false if the Atomic was never stored to.
func (v *Atomic[T]) TryLoad() (out T, ok bool) {
	value := v.val.Load()
	if value == nil {
		return out, false
	}
	return value.(T), true
}

// Store atomically replaces the current value with value.
func (v *Atomic[T]) Store(value T) {
	v.val.Store(value)
}

// Swap stores new and returns the previous value.
func (v *Atomic[T]) Swap(new T) T {
	return v.val.Swap(new).(T)
}

// CompareAndSwap stores new and returns true only if the current value equals old.
func (v *Atomic[T]) CompareAndSwap(old, new T) bool {
	return v.val.CompareAndSwap(old, new)
}
