package slicex

// Partitions splits s into consecutive chunks of at most size elements.
// The final chunk may contain fewer than size elements.
//
// If size is not positive, len(s) is used, producing a single chunk.
func Partitions[T any](s []T, size int) [][]T {
	if size <= 0 {
		size = len(s)
	}

	n := len(s)
	if n == 0 {
		return nil
	}

	chunks := make([][]T, 0, (n+size-1)/size)
	for start := 0; start < n; start += size {
		end := min(start+size, n)

		chunk := make([]T, end-start)
		copy(chunk, s[start:end])
		chunks = append(chunks, chunk)
	}

	return chunks
}

// Unique returns a new slice containing the first occurrence of every distinct
// value in s, preserving order.
func Unique[T comparable](s []T) []T {
	return UniqueT(s, func(v T) T { return v })
}

// UniqueT returns a new slice containing the first occurrence of every distinct
// value in s, where distinctness is determined by key, preserving order.
//
// It returns nil if key is nil.
func UniqueT[T any, K comparable](s []T, key func(T) K) []T {
	if len(s) == 0 || key == nil {
		return nil
	}

	seen := make(map[K]struct{}, len(s))
	out := make([]T, 0, len(s))
	for _, v := range s {
		k := key(v)
		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		out = append(out, v)
	}

	return out
}

// Intersect returns the distinct elements that appear in both a and b, in the
// order they first appear in a.
func Intersect[T comparable](a, b []T) []T {
	return IntersectT(a, b, func(v T) T { return v })
}

// IntersectT returns the distinct elements that appear in both a and b, where
// distinctness is determined by key, in the order they first appear in a.
//
// It returns nil if key is nil.
func IntersectT[T any, K comparable](a, b []T, key func(T) K) []T {
	if len(a) == 0 || len(b) == 0 || key == nil {
		return nil
	}

	inB := make(map[K]struct{}, len(b))
	for _, v := range b {
		inB[key(v)] = struct{}{}
	}

	out := make([]T, 0, min(len(a), len(b)))
	for _, v := range a {
		k := key(v)
		if _, ok := inB[k]; !ok {
			continue
		}

		out = append(out, v)
		delete(inB, k)
	}

	return out
}

// Union returns the distinct elements from a and b, taking elements from a
// first and then elements of b that do not already appear.
func Union[T comparable](a, b []T) []T {
	return UnionT(a, b, func(v T) T { return v })
}

// UnionT returns the distinct elements from a and b, where distinctness is
// determined by key, taking elements from a first and then elements of b whose
// key does not already appear.
//
// It returns nil if key is nil.
func UnionT[T any, K comparable](a, b []T, key func(T) K) []T {
	if (len(a) == 0 && len(b) == 0) || key == nil {
		return nil
	}

	seen := make(map[K]struct{}, len(a)+len(b))
	out := make([]T, 0, len(a)+len(b))
	for _, v := range a {
		k := key(v)
		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		out = append(out, v)
	}

	for _, v := range b {
		k := key(v)
		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		out = append(out, v)
	}

	return out
}

// Transform applies fn to every element of s and returns a new slice holding
// the results, preserving order. It returns nil if fn is nil.
func Transform[T, U any](s []T, fn func(T) U) []U {
	if len(s) == 0 || fn == nil {
		return nil
	}

	out := make([]U, len(s))
	for i, v := range s {
		out[i] = fn(v)
	}

	return out
}

// Filter returns a new slice containing the elements of s for which keep
// reports true, preserving order. It returns nil if keep is nil.
func Filter[T any](s []T, keep func(T) bool) []T {
	if len(s) == 0 || keep == nil {
		return nil
	}

	out := make([]T, 0, len(s))
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}

	return out
}

// FilterInPlace returns the retained elements of s, reusing its backing array
// and clearing the vacated tail; s must not be used afterwards.
func FilterInPlace[T any](s []T, keep func(T) bool) []T {
	if len(s) == 0 || keep == nil {
		return nil
	}

	out := s[:0]
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}

	clear(s[len(out):])

	return out
}
