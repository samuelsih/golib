package stringx

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/samuelsih/golib/assert"
)

func TestBytes(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		distinct bool
	}{
		{name: "zero", n: 0},
		{name: "one", n: 1},
		{name: "sixteen", n: 16},
		{name: "large", n: 1024},
		{name: "distinct", n: 16, distinct: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Bytes(tt.n)
			assert.Equal(t, len(got), tt.n)
			assert.NotNil(t, got)

			if tt.distinct {
				assert.NotEqual(t, got, Bytes(tt.n))
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		letters  []string
		charset  string
		distinct bool
	}{
		{name: "zero length", n: 0, charset: Base62Chars},
		{name: "default alphabet", n: 64, charset: Base62Chars, distinct: true},
		{name: "single letter", n: 32, letters: []string{"a"}, charset: "a"},
		{name: "custom letters", n: 32, letters: []string{"abc"}, charset: "abc", distinct: true},
		{name: "numeric letters", n: 16, letters: []string{DecChars}, charset: DecChars, distinct: true},
		{name: "hex letters", n: 16, letters: []string{HexChars}, charset: HexChars, distinct: true},
		{name: "unicode letters", n: 32, letters: []string{"ąę"}, charset: "ąę", distinct: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := String(tt.n, tt.letters...)
			assert.Equal(t, utf8.RuneCountInString(got), tt.n)
			assert.Equal(t, strings.Trim(got, tt.charset), "")

			if tt.distinct {
				assert.NotEqual(t, got, String(tt.n, tt.letters...))
			}
		})
	}
}

func TestGenerators(t *testing.T) {
	tests := []struct {
		name    string
		gen     func(int) string
		charset string
		n       int
	}{
		{name: "base64", gen: Base64, charset: Base64Chars, n: 64},
		{name: "base64 zero", gen: Base64, charset: Base64Chars, n: 0},
		{name: "base62", gen: Base62, charset: Base62Chars, n: 64},
		{name: "base62 zero", gen: Base62, charset: Base62Chars, n: 0},
		{name: "hex", gen: Hex, charset: HexChars, n: 32},
		{name: "hex zero", gen: Hex, charset: HexChars, n: 0},
		{name: "dec", gen: Dec, charset: DecChars, n: 32},
		{name: "dec zero", gen: Dec, charset: DecChars, n: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gen(tt.n)
			assert.Equal(t, utf8.RuneCountInString(got), tt.n)
			assert.Equal(t, strings.Trim(got, tt.charset), "")
		})
	}
}

func TestPascalCase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "blank", in: " ", want: ""},
		{name: "lowercase", in: "simple", want: "Simple"},
		{name: "uppercase", in: "SIMPLE", want: "SIMPLE"},
		{name: "snake", in: "simple_test", want: "SimpleTest"},
		{name: "kebab", in: "simple-test", want: "SimpleTest"},
		{name: "spaces", in: "simple test", want: "SimpleTest"},
		{name: "dotted", in: "simple.test", want: "SimpleTest"},
		{name: "mixed separators", in: "a.b-c_d e", want: "ABCDE"},
		{name: "already pascal", in: "AlreadyPascal", want: "AlreadyPascal"},
		{name: "camel", in: "alreadyCamel", want: "AlreadyCamel"},
		{name: "acronym", in: "JSONData", want: "JSONData"},
		{name: "digits", in: "foo2bar", want: "Foo2Bar"},
		{name: "leading digits", in: "2fast2furious", want: "2Fast2Furious"},
		{name: "padded", in: "  padded  ", want: "Padded"},
		{name: "repeated separators", in: "--multiple--separators--", want: "MultipleSeparators"},
		{name: "leading and trailing underscores", in: "__leading__trailing__", want: "LeadingTrailing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, PascalCase(tt.in), tt.want)
		})
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "blank", in: " ", want: ""},
		{name: "lowercase", in: "simple", want: "simple"},
		{name: "pascal single word", in: "Simple", want: "simple"},
		{name: "uppercase", in: "SIMPLE", want: "sIMPLE"},
		{name: "snake", in: "simple_test", want: "simpleTest"},
		{name: "kebab", in: "simple-test", want: "simpleTest"},
		{name: "spaces", in: "simple test", want: "simpleTest"},
		{name: "dotted", in: "simple.test", want: "simpleTest"},
		{name: "mixed separators", in: "a.b-c_d e", want: "aBCDE"},
		{name: "already camel", in: "alreadyCamel", want: "alreadyCamel"},
		{name: "pascal", in: "AlreadyPascal", want: "alreadyPascal"},
		{name: "acronym", in: "JSONData", want: "jSONData"},
		{name: "digits", in: "foo2bar", want: "foo2Bar"},
		{name: "leading digits", in: "2fast2furious", want: "2Fast2Furious"},
		{name: "padded", in: "  padded  ", want: "padded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, CamelCase(tt.in), tt.want)
		})
	}
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "blank", in: " ", want: ""},
		{name: "lowercase", in: "simple", want: "simple"},
		{name: "uppercase", in: "SIMPLE", want: "simple"},
		{name: "pascal", in: "AlreadyPascal", want: "already_pascal"},
		{name: "camel", in: "getUserList", want: "get_user_list"},
		{name: "acronym", in: "JSONData", want: "json_data"},
		{name: "acronym with number", in: "HTTP2Server", want: "http_2_server"},
		{name: "trailing acronym", in: "userID", want: "user_id"},
		{name: "snake unchanged", in: "already_snake_test", want: "already_snake_test"},
		{name: "kebab", in: "simple-test", want: "simple_test"},
		{name: "spaces", in: "simple test", want: "simple_test"},
		{name: "dotted", in: "simple.test", want: "simple.test"},
		{name: "digits", in: "foo2Bar", want: "foo_2_bar"},
		{name: "leading digits", in: "2fast2furious", want: "2_fast_2_furious"},
		{name: "mixed separators", in: "a.b-c_d e", want: "a.b_c_d_e"},
		{name: "padded", in: "  padded  ", want: "padded"},
		{name: "leading and trailing underscores", in: "__leading__trailing__", want: "__leading__trailing__"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, SnakeCase(tt.in), tt.want)
		})
	}
}

func TestUnsafeFromBytes(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{name: "nil", in: nil, want: ""},
		{name: "empty", in: []byte{}, want: ""},
		{name: "ascii", in: []byte("mario"), want: "mario"},
		{name: "unicode", in: []byte("mario🎮"), want: "mario🎮"},
		{name: "with zero byte", in: []byte{'a', 0, 'b'}, want: "a\x00b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, UnsafeFromBytes(tt.in), tt.want)
		})
	}
}

func TestUnsafeToBytes(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantLen int
	}{
		{name: "empty", in: "", wantLen: 0},
		{name: "ascii", in: "mario", wantLen: 5},
		{name: "unicode", in: "mario🎮", wantLen: len("mario🎮")},
		{name: "with zero byte", in: "a\x00b", wantLen: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnsafeToBytes(tt.in)
			assert.Equal(t, len(got), tt.wantLen)
			assert.Equal(t, string(got), tt.in)
		})
	}
}

func TestUnsafeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "ascii", in: "mario"},
		{name: "unicode", in: "mario🎮"},
		{name: "with zero byte", in: "a\x00b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, UnsafeFromBytes(UnsafeToBytes(tt.in)), tt.in)
			assert.Equal(t, string(UnsafeToBytes(UnsafeFromBytes([]byte(tt.in)))), tt.in)
		})
	}
}
