package cryptx

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	ArgonSaltLength = 16
	ArgonKeyLength  = 32
	ArgonTime       = 3
	ArgonMemory     = 64 * 1024
	ArgonThreads    = 4
)

// Argon2PasswordDefaultParams contains the default parameters for Argon2EncodePassword.
var Argon2PasswordDefaultParams = ArgonEncodePasswordParams{
	Time:       ArgonTime,
	Memory:     ArgonMemory,
	Threads:    ArgonThreads,
	KeyLength:  ArgonKeyLength,
	SaltLength: ArgonSaltLength,
}

// ArgonEncodePasswordParams configures Argon2EncodePassword; zero fields use defaults.
type ArgonEncodePasswordParams struct {
	Time       uint32 // Number of passes over memory.
	Memory     uint32 // Memory in KiB.
	Threads    uint8  // Degree of parallelism.
	KeyLength  uint32 // Derived key length in bytes.
	SaltLength uint32 // Random salt length in bytes.
}

// Argon2EncodePassword hashes password with Argon2id using params.
func Argon2EncodePassword(password string, params ArgonEncodePasswordParams) (string, error) {
	params = params.withDefaults()

	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("cryptx: generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, params.KeyLength)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Time,
		params.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Argon2ComparePassword reports whether password matches the encoded Argon2id hash.
func Argon2ComparePassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false
	}

	version, ok := parseField[uint32](parts[2], "v=", 32)
	if !ok || version != argon2.Version {
		return false
	}

	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false
	}

	memory, ok := parseField[uint32](params[0], "m=", 32)
	if !ok || memory == 0 {
		return false
	}

	timeCost, ok := parseField[uint32](params[1], "t=", 32)
	if !ok || timeCost == 0 {
		return false
	}

	threads, ok := parseField[uint8](params[2], "p=", 8)
	if !ok || threads == 0 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return false
	}

	computed := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(key)))

	return subtle.ConstantTimeCompare(key, computed) == 1
}

func (p ArgonEncodePasswordParams) withDefaults() ArgonEncodePasswordParams {
	if p.Time == 0 {
		p.Time = ArgonTime
	}

	if p.Memory == 0 {
		p.Memory = ArgonMemory
	}

	if p.Threads == 0 {
		p.Threads = ArgonThreads
	}

	if p.KeyLength == 0 {
		p.KeyLength = ArgonKeyLength
	}

	if p.SaltLength == 0 {
		p.SaltLength = ArgonSaltLength
	}

	return p
}

func parseField[T uint8 | uint32](field, prefix string, bits int) (T, bool) {
	value, found := strings.CutPrefix(field, prefix)
	if !found {
		return 0, false
	}

	parsed, err := strconv.ParseUint(value, 10, bits)
	if err != nil {
		return 0, false
	}

	return T(parsed), true
}
