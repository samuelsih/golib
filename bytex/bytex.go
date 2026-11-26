package bytex

import "bytes"

const (
	KB = 1_000
	MB = 1_000 * KB
	GB = 1_000 * MB
	TB = 1_000 * GB
)

const (
	KiB = 1 << 10
	MiB = 1 << 20
	GiB = 1 << 30
	TiB = 1 << 40
)

// IndexAnyByte returns the index of the first byte in s that appears in set,
// or -1 if none does. It compares raw bytes without decoding UTF-8.
func IndexAnyByte(s, set []byte) int {
	if len(s) == 0 || len(set) == 0 {
		return -1
	}

	if len(set) == 1 {
		return bytes.IndexByte(s, set[0])
	}

	var contains [256]bool
	for _, b := range set {
		contains[b] = true
	}

	for i, b := range s {
		if contains[b] {
			return i
		}
	}

	return -1
}
