package httpx

import (
	"strings"
	"testing"

	"github.com/samuelsih/golib/assert"
)

func TestWildcard(t *testing.T) {
	w := wildcard{"foo", "bar"}
	assert.True(t, w.match("foobar"))
	assert.True(t, w.match("foobazbar"))
	assert.False(t, w.match("foobaz"))

	w = wildcard{"foo", "oof"}
	assert.False(t, w.match("foof"))
}

func TestConvert(t *testing.T) {
	s := convert([]string{"A", "b", "C"}, strings.ToLower)
	e := []string{"a", "b", "c"}
	assert.Equal(t, s, e)
}

func TestParseHeaderList(t *testing.T) {
	h := parseHeaderList("header, second-header, THIRD-HEADER, Numb3r3d-H34d3r, Header_with_underscore Header.with.full.stop")
	e := []string{"Header", "Second-Header", "Third-Header", "Numb3r3d-H34d3r", "Header_with_underscore", "Header.with.full.stop"}
	assert.Equal(t, h, e)
}

func TestParseHeaderListEmpty(t *testing.T) {
	assert.Equal(t, parseHeaderList(""), []string{})
	assert.Equal(t, parseHeaderList(" , "), []string{})
}

func BenchmarkParseHeaderList(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		parseHeaderList("header, second-header, THIRD-HEADER")
	}
}

func BenchmarkParseHeaderListSingle(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		parseHeaderList("header")
	}
}

func BenchmarkParseHeaderListNormalized(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		parseHeaderList("Header1, Header2, Third-Header")
	}
}

func BenchmarkWildcard(b *testing.B) {
	w := wildcard{"foo", "bar"}
	b.Run("match", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			w.match("foobazbar")
		}
	})
	b.Run("too short", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			w.match("fobar")
		}
	})
}
