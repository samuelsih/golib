package sqlmigration

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/samuelsih/golib/assert"
)

const wantTemplate = "-- +migrate Up\n\n\n-- +migrate Down\n"

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		migration string
		suffix    string
	}{
		{name: "plain name", migration: "add_users", suffix: "_add_users.sql"},
		{name: "already has extension", migration: "add_users.sql", suffix: "_add_users.sql"},
		{name: "trims whitespace", migration: "  add_users  ", suffix: "_add_users.sql"},
		{name: "keeps inner spaces", migration: "add users", suffix: "_add users.sql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "migrations")

			filename, err := New(dir, tt.migration)
			assert.NoError(t, err)
			assert.True(t, strings.HasSuffix(filename, tt.suffix))

			content, err := os.ReadFile(filename)
			assert.NoError(t, err)
			assert.Equal(t, string(content), wantTemplate)
		})
	}
}

func TestNewTimestampPrefix(t *testing.T) {
	filename, err := New(t.TempDir(), "init")
	assert.NoError(t, err)

	stamp := regexp.MustCompile(`^\d{14}_init\.sql$`)
	assert.True(t, stamp.MatchString(filepath.Base(filename)))
}

func TestNewEmptyName(t *testing.T) {
	_, err := New(t.TempDir(), "   ")
	assert.NotNil(t, err)
	if err != nil {
		assert.Equal(t, err.Error(), "sqlmigrate: migration name is empty")
	}
}

func TestNewMkdirError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	assert.NoError(t, os.WriteFile(blocker, nil, 0o600))

	_, err := New(blocker, "init")
	assert.NotNil(t, err)
}

func TestChecksumMismatchError(t *testing.T) {
	err := &ChecksumMismatchError{Migration: "1_init.sql", Stored: "old", File: "new"}

	assert.ErrorIs(t, err, ErrChecksumMismatch)
	assert.False(t, errors.Is(err, ErrStaleMigration))
	assert.False(t, errors.Is(err, ErrMissingDown))
	assert.Equal(t, err.Error(), "sqlmigrate: 1_init.sql was already applied with a different checksum (stored old, file new)")

	got := assert.ErrorAsType[*ChecksumMismatchError](t, err)
	assert.Equal(t, got.Migration, "1_init.sql")
	assert.Equal(t, got.Stored, "old")
	assert.Equal(t, got.File, "new")
}

func TestStaleMigrationError(t *testing.T) {
	err := &StaleMigrationError{Migration: "1_init.sql", Dir: "migrations"}

	assert.ErrorIs(t, err, ErrStaleMigration)
	assert.False(t, errors.Is(err, ErrChecksumMismatch))
	assert.False(t, errors.Is(err, ErrMissingDown))
	assert.Equal(t, err.Error(), "sqlmigrate: 1_init.sql is recorded as applied but missing from migrations")

	got := assert.ErrorAsType[*StaleMigrationError](t, err)
	assert.Equal(t, got.Migration, "1_init.sql")
	assert.Equal(t, got.Dir, "migrations")
}

func TestMissingDownError(t *testing.T) {
	err := &MissingDownError{Migration: "1_init.sql"}

	assert.ErrorIs(t, err, ErrMissingDown)
	assert.False(t, errors.Is(err, ErrChecksumMismatch))
	assert.False(t, errors.Is(err, ErrStaleMigration))
	assert.Equal(t, err.Error(), "sqlmigrate: 1_init.sql has no -- +migrate Down section")

	got := assert.ErrorAsType[*MissingDownError](t, err)
	assert.Equal(t, got.Migration, "1_init.sql")
}

func TestErrNoAppliedMigrations(t *testing.T) {
	assert.False(t, errors.Is(ErrNoAppliedMigrations, ErrMissingDown))
	assert.Equal(t, ErrNoAppliedMigrations.Error(), "sqlmigrate: no applied migrations to roll back")
}
