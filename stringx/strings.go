package stringx

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
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
