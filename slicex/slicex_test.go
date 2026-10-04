package slicex

import (
	"strconv"
	"testing"

	"github.com/samuelsih/golib/assert"
)

type person struct {
	ID   int
	Name string
}

func TestPartitions(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		size int
		want [][]int
	}{
		{name: "empty", s: nil, size: 3},
		{name: "empty with zero size", s: nil, size: 0},
		{name: "exact", s: []int{1, 2, 3, 4}, size: 2, want: [][]int{{1, 2}, {3, 4}}},
		{name: "remainder", s: []int{1, 2, 3, 4, 5}, size: 2, want: [][]int{{1, 2}, {3, 4}, {5}}},
		{name: "single chunk", s: []int{1, 2}, size: 5, want: [][]int{{1, 2}}},
		{name: "size one", s: []int{1, 2}, size: 1, want: [][]int{{1}, {2}}},
		{name: "zero size", s: []int{1, 2}, size: 0, want: [][]int{{1, 2}}},
		{name: "negative size", s: []int{1, 2}, size: -1, want: [][]int{{1, 2}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Partitions(tt.s, tt.size)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name string
		s    []string
		want []string
	}{
		{name: "empty", s: nil},
		{name: "no duplicates", s: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "duplicates", s: []string{"a", "b", "a", "c", "b"}, want: []string{"a", "b", "c"}},
		{name: "empty strings", s: []string{"", "a", ""}, want: []string{"", "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unique(tt.s)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestUniqueT(t *testing.T) {
	tests := []struct {
		name string
		s    []person
		want []person
	}{
		{name: "empty", s: nil},
		{name: "struct values", s: []person{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, want: []person{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}},
		{name: "duplicate ids", s: []person{{ID: 1, Name: "a"}, {ID: 1, Name: "b"}, {ID: 2, Name: "c"}}, want: []person{{ID: 1, Name: "a"}, {ID: 2, Name: "c"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueT(tt.s, func(p person) int { return p.ID })
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		name string
		a    []int
		b    []int
		want []int
	}{
		{name: "both empty", a: nil, b: nil},
		{name: "left empty", a: nil, b: []int{1}},
		{name: "right empty", a: []int{1}, b: nil},
		{name: "duplicates", a: []int{3, 1, 2, 3}, b: []int{1, 3}, want: []int{3, 1}},
		{name: "duplicates in both", a: []int{1, 1, 2}, b: []int{1, 1, 3}, want: []int{1}},
		{name: "disjoint", a: []int{1, 2}, b: []int{3, 4}, want: []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Intersect(tt.a, tt.b)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestIntersectT(t *testing.T) {
	tests := []struct {
		name string
		a    []person
		b    []person
		want []person
	}{
		{name: "both empty", a: nil, b: nil},
		{name: "shared ids", a: []person{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}, {ID: 1, Name: "c"}}, b: []person{{ID: 1, Name: "x"}, {ID: 3, Name: "y"}}, want: []person{{ID: 1, Name: "a"}}},
		{name: "order of a", a: []person{{ID: 2, Name: "x"}, {ID: 1, Name: "y"}}, b: []person{{ID: 1, Name: "z"}, {ID: 2, Name: "w"}}, want: []person{{ID: 2, Name: "x"}, {ID: 1, Name: "y"}}},
		{name: "disjoint ids", a: []person{{ID: 1, Name: "a"}}, b: []person{{ID: 2, Name: "b"}}, want: []person{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntersectT(tt.a, tt.b, func(p person) int { return p.ID })
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestUnion(t *testing.T) {
	tests := []struct {
		name string
		a    []int
		b    []int
		want []int
	}{
		{name: "both empty", a: nil, b: nil},
		{name: "left empty", a: nil, b: []int{1, 2}, want: []int{1, 2}},
		{name: "right empty", a: []int{1, 2}, b: nil, want: []int{1, 2}},
		{name: "overlap", a: []int{1, 2, 2}, b: []int{2, 3}, want: []int{1, 2, 3}},
		{name: "duplicates in both", a: []int{1, 1}, b: []int{1, 2, 2}, want: []int{1, 2}},
		{name: "disjoint", a: []int{1}, b: []int{2}, want: []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Union(tt.a, tt.b)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestUnionT(t *testing.T) {
	tests := []struct {
		name string
		a    []person
		b    []person
		want []person
	}{
		{name: "both empty", a: nil, b: nil},
		{name: "first occurrence wins", a: []person{{ID: 1, Name: "a"}, {ID: 1, Name: "b"}}, b: []person{{ID: 1, Name: "c"}, {ID: 2, Name: "d"}}, want: []person{{ID: 1, Name: "a"}, {ID: 2, Name: "d"}}},
		{name: "a before b", a: []person{{ID: 2, Name: "x"}}, b: []person{{ID: 1, Name: "y"}}, want: []person{{ID: 2, Name: "x"}, {ID: 1, Name: "y"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnionT(tt.a, tt.b, func(p person) int { return p.ID })
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestNilKey(t *testing.T) {
	var key func(person) int

	assert.Nil(t, UniqueT([]person{{ID: 1}}, key))
	assert.Nil(t, IntersectT([]person{{ID: 1}}, []person{{ID: 1}}, key))
	assert.Nil(t, UnionT([]person{{ID: 1}}, []person{{ID: 2}}, key))
}

func TestTransform(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		want []string
	}{
		{name: "empty", s: nil},
		{name: "values", s: []int{1, 2, 3}, want: []string{"1", "2", "3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Transform(tt.s, strconv.Itoa)
			assert.Equal(t, got, tt.want)
		})
	}

	t.Run("nil fn", func(t *testing.T) {
		var fn func(int) string

		got := Transform([]int{1}, fn)
		assert.Nil(t, got)
	})
}

func TestFilter(t *testing.T) {
	even := func(v int) bool { return v%2 == 0 }

	tests := []struct {
		name string
		s    []int
		want []int
	}{
		{name: "empty", s: nil},
		{name: "none match", s: []int{1, 3, 5}, want: []int{}},
		{name: "all match", s: []int{2, 4}, want: []int{2, 4}},
		{name: "some match", s: []int{1, 2, 3, 4, 5}, want: []int{2, 4}},
		{name: "order preserved", s: []int{4, 1, 2, 3}, want: []int{4, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(tt.s, even)
			assert.Equal(t, got, tt.want)
		})
	}

	t.Run("does not mutate input", func(t *testing.T) {
		s := []int{1, 2, 3, 4}

		got := Filter(s, even)
		assert.Equal(t, got, []int{2, 4})
		assert.Equal(t, s, []int{1, 2, 3, 4})
	})

	t.Run("nil fn", func(t *testing.T) {
		var fn func(int) bool

		got := Filter([]int{1}, fn)
		assert.Nil(t, got)
	})
}

func TestFilterInPlace(t *testing.T) {
	even := func(v int) bool { return v%2 == 0 }

	tests := []struct {
		name string
		s    []int
		want []int
	}{
		{name: "empty", s: nil},
		{name: "none match", s: []int{1, 3, 5}, want: []int{}},
		{name: "all match", s: []int{2, 4}, want: []int{2, 4}},
		{name: "some match", s: []int{1, 2, 3, 4, 5}, want: []int{2, 4}},
		{name: "order preserved", s: []int{4, 1, 2, 3}, want: []int{4, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterInPlace(tt.s, even)
			assert.Equal(t, got, tt.want)
		})
	}

	t.Run("shares backing array", func(t *testing.T) {
		s := []int{1, 2, 3, 4}

		got := FilterInPlace(s, even)
		assert.Equal(t, got, []int{2, 4})

		got[0] = 99
		assert.Equal(t, s[0], 99)
	})

	t.Run("clears vacated tail", func(t *testing.T) {
		s := []string{"a", "bb", "c", "dd"}

		got := FilterInPlace(s, func(v string) bool { return len(v) == 1 })
		assert.Equal(t, got, []string{"a", "c"})
		assert.Equal(t, s[2:], []string{"", ""})
	})

	t.Run("nil fn", func(t *testing.T) {
		var fn func(int) bool

		got := FilterInPlace([]int{1}, fn)
		assert.Nil(t, got)
	})
}
