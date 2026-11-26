package bytex

import (
	"testing"

	"github.com/samuelsih/golib/assert"
)

func TestIndexAnyByte(t *testing.T) {
	tests := []struct {
		name string
		s    []byte
		set  []byte
		want int
	}{
		{name: "first delimiter", s: []byte("a,b;c"), set: []byte(";,."), want: 1},
		{name: "later delimiter", s: []byte("abc,def;ghi"), set: []byte(";,."), want: 3},
		{name: "match at start", s: []byte(",abc"), set: []byte(";,."), want: 0},
		{name: "match at end", s: []byte("abc;"), set: []byte(";,."), want: 3},
		{name: "single byte set", s: []byte("abc;def"), set: []byte(";"), want: 3},
		{name: "no match", s: []byte("abcdef"), set: []byte(";,."), want: -1},
		{name: "empty s", s: nil, set: []byte(";,."), want: -1},
		{name: "empty set", s: []byte("abc"), set: nil, want: -1},
		{name: "empty s and set", s: []byte{}, set: []byte{}, want: -1},
		{name: "non-ascii bytes", s: []byte{0x00, 0xff}, set: []byte{0xff}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, IndexAnyByte(tt.s, tt.set), tt.want)
		})
	}
}
