package cryptx

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/samuelsih/golib/assert"
	"golang.org/x/crypto/argon2"
)

func TestArgon2DefaultParams(t *testing.T) {
	assert.Equal(t, Argon2PasswordDefaultParams, ArgonEncodePasswordParams{
		Time:       3,
		Memory:     64 * 1024,
		Threads:    4,
		KeyLength:  32,
		SaltLength: 16,
	})
}

func TestArgon2EncodePassword(t *testing.T) {
	encoded, err := Argon2EncodePassword("correct horse battery staple", Argon2PasswordDefaultParams)
	assert.NoError(t, err)

	parts := strings.Split(encoded, "$")
	assert.Equal(t, len(parts), 6)
	assert.Equal(t, parts[0], "")
	assert.Equal(t, parts[1], "argon2id")
	assert.Equal(t, parts[2], fmt.Sprintf("v=%d", argon2.Version))
	assert.Equal(t, parts[3], fmt.Sprintf(
		"m=%d,t=%d,p=%d",
		Argon2PasswordDefaultParams.Memory,
		Argon2PasswordDefaultParams.Time,
		Argon2PasswordDefaultParams.Threads,
	))

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	assert.NoError(t, err)
	assert.Equal(t, len(salt), int(Argon2PasswordDefaultParams.SaltLength))

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	assert.NoError(t, err)
	assert.Equal(t, len(key), int(Argon2PasswordDefaultParams.KeyLength))
}

func TestArgon2EncodePasswordUsesRandomSalt(t *testing.T) {
	first, err := Argon2EncodePassword("password", Argon2PasswordDefaultParams)
	assert.NoError(t, err)

	second, err := Argon2EncodePassword("password", Argon2PasswordDefaultParams)
	assert.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestArgon2ComparePassword(t *testing.T) {
	encoded, err := Argon2EncodePassword("s3cret", Argon2PasswordDefaultParams)
	assert.NoError(t, err)

	assert.True(t, Argon2ComparePassword("s3cret", encoded))
	assert.False(t, Argon2ComparePassword("wrong", encoded))
	assert.False(t, Argon2ComparePassword("", encoded))
}

func TestArgon2ComparePasswordEmptyPassword(t *testing.T) {
	encoded, err := Argon2EncodePassword("", Argon2PasswordDefaultParams)
	assert.NoError(t, err)

	assert.True(t, Argon2ComparePassword("", encoded))
	assert.False(t, Argon2ComparePassword("x", encoded))
}

func TestArgon2EncodePasswordCustomParams(t *testing.T) {
	params := ArgonEncodePasswordParams{
		Time:       2,
		Memory:     16 * 1024,
		Threads:    2,
		KeyLength:  24,
		SaltLength: 20,
	}

	encoded, err := Argon2EncodePassword("password", params)
	assert.NoError(t, err)

	parts := strings.Split(encoded, "$")
	assert.Equal(t, parts[2], fmt.Sprintf("v=%d", argon2.Version))
	assert.Equal(t, parts[3], fmt.Sprintf("m=%d,t=%d,p=%d", params.Memory, params.Time, params.Threads))

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	assert.NoError(t, err)
	assert.Equal(t, len(salt), int(params.SaltLength))

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	assert.NoError(t, err)
	assert.Equal(t, len(key), int(params.KeyLength))

	assert.True(t, Argon2ComparePassword("password", encoded))
	assert.False(t, Argon2ComparePassword("wrong", encoded))
}

func TestArgon2EncodePasswordZeroFieldsUseDefaults(t *testing.T) {
	encoded, err := Argon2EncodePassword("password", ArgonEncodePasswordParams{Threads: 2})
	assert.NoError(t, err)

	parts := strings.Split(encoded, "$")
	assert.Equal(t, parts[3], fmt.Sprintf("m=%d,t=%d,p=2", ArgonMemory, ArgonTime))

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	assert.NoError(t, err)
	assert.Equal(t, len(salt), ArgonSaltLength)

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	assert.NoError(t, err)
	assert.Equal(t, len(key), ArgonKeyLength)
}

func TestArgon2ComparePasswordMalformed(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
	}{
		{name: "empty", encoded: ""},
		{name: "not a hash", encoded: "password"},
		{name: "missing leading separator", encoded: "x$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$a2V5"},
		{name: "wrong algorithm", encoded: "$argon2i$v=19$m=65536,t=3,p=4$c2FsdA$a2V5"},
		{name: "wrong version", encoded: "$argon2id$v=16$m=65536,t=3,p=4$c2FsdA$a2V5"},
		{name: "malformed version", encoded: "$argon2id$version=19$m=65536,t=3,p=4$c2FsdA$a2V5"},
		{name: "missing parameter", encoded: "$argon2id$v=19$m=65536,t=3$c2FsdA$a2V5"},
		{name: "extra parameter", encoded: "$argon2id$v=19$m=65536,t=3,p=4,x=1$c2FsdA$a2V5"},
		{name: "zero memory", encoded: "$argon2id$v=19$m=0,t=3,p=4$c2FsdA$a2V5"},
		{name: "zero time", encoded: "$argon2id$v=19$m=65536,t=0,p=4$c2FsdA$a2V5"},
		{name: "zero threads", encoded: "$argon2id$v=19$m=65536,t=3,p=0$c2FsdA$a2V5"},
		{name: "memory overflow", encoded: "$argon2id$v=19$m=4294967296,t=3,p=4$c2FsdA$a2V5"},
		{name: "threads overflow", encoded: "$argon2id$v=19$m=65536,t=3,p=256$c2FsdA$a2V5"},
		{name: "malformed memory", encoded: "$argon2id$v=19$m=many,t=3,p=4$c2FsdA$a2V5"},
		{name: "invalid salt encoding", encoded: "$argon2id$v=19$m=65536,t=3,p=4$!!!$a2V5"},
		{name: "invalid key encoding", encoded: "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$!!!"},
		{name: "empty salt", encoded: "$argon2id$v=19$m=65536,t=3,p=4$$a2V5"},
		{name: "empty key", encoded: "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$"},
		{name: "padded salt", encoded: "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA==$a2V5"},
		{name: "padded key", encoded: "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$a2V5="},
		{name: "extra segment", encoded: "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$a2V5$extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, Argon2ComparePassword("password", tt.encoded))
		})
	}
}
