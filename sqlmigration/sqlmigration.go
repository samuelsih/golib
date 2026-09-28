package sqlmigration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

const migrationTemplate = "-- +migrate Up\n\n\n-- +migrate Down\n"

// DB starts migration transactions.
type DB interface {
	Begin(ctx context.Context) (Tx, error)
}

// AppliedMigration is a migration recorded as applied in the bookkeeping
// table.
type AppliedMigration struct {
	Migration string
	Checksum  string
}

// Tx is a single migration transaction. Every statement of one run executes in
// the same transaction, so a failure rolls the whole run back.
type Tx interface {
	// Lock blocks until the migration lock is held, serializing concurrent
	// runs. Implementations may use advisory locks, table locks, or any other
	// mechanism the database provides.
	Lock(ctx context.Context) error

	// EnsureMigrationsTable creates the bookkeeping table when it is missing.
	EnsureMigrationsTable(ctx context.Context) error

	// Applied returns the migrations recorded as applied.
	Applied(ctx context.Context) ([]AppliedMigration, error)

	// Exec runs one migration section, which may contain multiple statements.
	Exec(ctx context.Context, script string) error

	// Record stores an applied migration.
	Record(ctx context.Context, applied AppliedMigration) error

	// Forget removes a migration record after its Down section ran.
	Forget(ctx context.Context, migration string) error

	// Commit persists the transaction.
	Commit(ctx context.Context) error

	// Rollback aborts the transaction. Calling it after Commit may return an
	// error, which callers using it as deferred cleanup can ignore.
	Rollback(ctx context.Context) error
}

var (
	// ErrChecksumMismatch reports that a migration file changed after it was
	// applied.
	ErrChecksumMismatch = errors.New("sqlmigrate: migration already applied with a different checksum")

	// ErrStaleMigration reports a migration recorded as applied that has no
	// matching file.
	ErrStaleMigration = errors.New("sqlmigrate: applied migration missing from migration files")

	// ErrNoAppliedMigrations reports a Down call with nothing to roll back.
	ErrNoAppliedMigrations = errors.New("sqlmigrate: no applied migrations to roll back")

	// ErrMissingDown reports a Down call on a migration without a Down section.
	ErrMissingDown = errors.New("sqlmigrate: applied migration has no Down section")
)

// ChecksumMismatchError reports a migration file whose content no longer
// matches the checksum recorded in the database.
type ChecksumMismatchError struct {
	Migration string
	Stored    string
	File      string
}

func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf("sqlmigrate: %s was already applied with a different checksum (stored %s, file %s)", e.Migration, e.Stored, e.File)
}

func (e *ChecksumMismatchError) Is(target error) bool {
	return target == ErrChecksumMismatch
}

// StaleMigrationError reports a migration recorded as applied whose file is
// missing from the migration directory.
type StaleMigrationError struct {
	Migration string
	Dir       string
}

func (e *StaleMigrationError) Error() string {
	return fmt.Sprintf("sqlmigrate: %s is recorded as applied but missing from %s", e.Migration, e.Dir)
}

func (e *StaleMigrationError) Is(target error) bool {
	return target == ErrStaleMigration
}

// MissingDownError reports a rollback of a migration without a Down section.
type MissingDownError struct {
	Migration string
}

func (e *MissingDownError) Error() string {
	return fmt.Sprintf("sqlmigrate: %s has no -- +migrate Down section", e.Migration)
}

func (e *MissingDownError) Is(target error) bool {
	return target == ErrMissingDown
}

// New creates an empty migration file in dir and returns its path. name becomes
// the file suffix after a UTC timestamp, and ".sql" is appended when missing.
func New(dir, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("sqlmigrate: migration name is empty")
	}
	if !strings.HasSuffix(name, ".sql") {
		name += ".sql"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	filename := time.Now().UTC().Format("20060102150405") + "_" + name
	fullPath := path.Join(dir, filename)

	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := file.WriteString(migrationTemplate); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return fullPath, nil
}
