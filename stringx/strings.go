package stringx

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"strings"
	"unsafe"
)

const (
	Base64Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+/"
	Base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	HexChars    = "0123456789abcdef"
	DecChars    = "0123456789"
)

// list of default letters that can be used to make a random string when calling String
// function with no letters provided.
//
// source: https://github.com/thanhpk/randstr
var defaultRandomLetters = []rune(Base62Chars)

// Bytes generates n random bytes.
//
// source: https://github.com/thanhpk/randstr
func Bytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

// Base64 generates a random Base64 string with length of n.
//
// Example: X02+jDDF/exDoqPg9/aXlzbUCN93GIQ5
//
// source: https://github.com/thanhpk/randstr
func Base64(n int) string { return String(n, Base64Chars) }

// Base62 generates a random Base62 string with length of n.
//
// Example: 1BsNqB61o4ztSqLC6labKGNf4MYy352X
//
// source: https://github.com/thanhpk/randstr
func Base62(n int) string { return String(n, Base62Chars) }

// Dec generates a random decimal number string with length of n
//
// Example: 37110235710860781655802098192113
//
// source: https://github.com/thanhpk/randstr
func Dec(n int) string { return String(n, DecChars) }

// Hex generates a random Hexadecimal string with length of n
//
// Example: 67aab2d956bd7cc621af22cfb169cba8
//
// source: https://github.com/thanhpk/randstr
func Hex(n int) string { return String(n, HexChars) }

// String generates a random string using only letters provided in the letters parameter.
//
// If user omits letters parameter, this function will use Base62Chars instead.
//
// source: https://github.com/thanhpk/randstr
func String(n int, letters ...string) string {
	var letterRunes []rune
	if len(letters) == 0 {
		letterRunes = defaultRandomLetters
	} else {
		letterRunes = []rune(letters[0])
	}

	var bb bytes.Buffer
	bb.Grow(n)
	l := int64(len(letterRunes))
	// on each loop, generate one random rune and append to output
	for range n {
		bb.WriteRune(letterRunes[int64(binary.BigEndian.Uint32(Bytes(4)))%l])
	}
	return bb.String()
}

// PascalCase convert to PascalCase.
//
// source: https://github.com/iancoleman/strcase
func PascalCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := true
	for i, v := range []byte(s) {
		isCap := v >= 'A' && v <= 'Z'
		isLow := v >= 'a' && v <= 'z'
		if capNext {
			if isLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if isCap {
				v += 'a'
				v -= 'A'
			}
		}
		if isCap || isLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
	}
	return n.String()
}

// CamelCase convert to camelCase.
//
// source: https://github.com/iancoleman/strcase
func CamelCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := false
	for i, v := range []byte(s) {
		isCap := v >= 'A' && v <= 'Z'
		isLow := v >= 'a' && v <= 'z'
		if capNext {
			if isLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if isCap {
				v += 'a'
				v -= 'A'
			}
		}
		if isCap || isLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
	}
	return n.String()
}

// SnakeCase convert to snake case.
//
// source: https://github.com/iancoleman/strcase
func SnakeCase(s string) string {
	s = strings.TrimSpace(s)
	n := strings.Builder{}
	const delimiter = '_'
	ignore := byte(0)
	n.Grow(len(s) + 2) // nominal 2 bytes of extra space for inserted delimiters
	for i, v := range []byte(s) {
		isCap := v >= 'A' && v <= 'Z'
		isLow := v >= 'a' && v <= 'z'
		if isCap {
			v += 'a'
			v -= 'A'
		}

		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		if i+1 < len(s) {
			next := s[i+1]
			vIsNum := v >= '0' && v <= '9'
			nextIsCap := next >= 'A' && next <= 'Z'
			nextIsLow := next >= 'a' && next <= 'z'
			nextIsNum := next >= '0' && next <= '9'
			// add underscore if next letter case type is changed
			if (isCap && (nextIsLow || nextIsNum)) || (isLow && (nextIsCap || nextIsNum)) || (vIsNum && (nextIsCap || nextIsLow)) {
				if prevIgnore := ignore > 0 && i > 0 && s[i-1] == ignore; !prevIgnore {
					if isCap && nextIsLow {
						if prevIsCap := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; prevIsCap {
							n.WriteByte(delimiter)
						}
					}
					n.WriteByte(v)
					if isLow || vIsNum || nextIsNum {
						n.WriteByte(delimiter)
					}
					continue
				}
			}
		}

		if (v == ' ' || v == '_' || v == '-') && v != ignore {
			// replace space/underscore/hyphen with delimiter
			n.WriteByte(delimiter)
		} else {
			n.WriteByte(v)
		}
	}

	return n.String()
}

const (
	MaskFromBegin = 1
	MaskUntilEnd  = -1
)

// Mask replaces runes at 1-based positions startAt through endAt (inclusive) with
// the repeated replacement rune. Use MaskEnd as endAt to mask through the last rune.
func Mask(target string, replacement rune, startAt, endAt int) string {
	if target == "" || replacement == 0 {
		return target
	}

	runes := []rune(target)
	if startAt < 1 {
		startAt = 1
	}
	if endAt <= MaskUntilEnd {
		endAt = len(runes)
	}
	if endAt > len(runes) {
		endAt = len(runes)
	}
	if startAt > endAt || startAt > len(runes) {
		return target
	}

	masked := endAt - startAt + 1
	repl := string(replacement)

	var b strings.Builder
	b.Grow(len(target) + masked*len(repl))
	b.WriteString(string(runes[:startAt-1]))
	b.WriteString(strings.Repeat(repl, masked))
	b.WriteString(string(runes[endAt:]))
	return b.String()
}

// UnsafeFromBytes returns a string pointer without allocation.
// Dont ever mutate the []byte if []byte still used.
func UnsafeFromBytes(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// UnsafeToBytes returns a byte pointer without allocation.
// Dont ever mutate the returned []byte if string still used.
func UnsafeToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
