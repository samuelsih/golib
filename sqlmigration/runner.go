package sqlmigration

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

var directivePattern = regexp.MustCompile(`(?i)^--\s*\+migrate\s+(up|down)$`)

// Up applies every pending migration in a single transaction.
func Up(ctx context.Context, db DB, fsys fs.FS, dir string) error {
	migrations, err := loadMigrations(fsys, dir)
	if err != nil {
		return err
	}

	return transaction(ctx, db, func(tx Tx) error {
		records, err := tx.Applied(ctx)
		if err != nil {
			return fmt.Errorf("sqlmigrate: load applied migrations: %w", err)
		}
		if err := markApplied(migrations, records, dir); err != nil {
			return err
		}

		for m := range slices.Values(migrations) {
			if m.applied {
				continue
			}
			if err := tx.Exec(ctx, m.up); err != nil {
				return fmt.Errorf("sqlmigrate: run %s: %w", m.name, err)
			}
			if err := tx.Record(ctx, AppliedMigration{Migration: m.name, Checksum: m.checksum}); err != nil {
				return fmt.Errorf("sqlmigrate: record %s: %w", m.name, err)
			}
		}
		return nil
	})
}

// Down rolls back the most recently applied migration in a single transaction.
// It returns ErrNoAppliedMigrations when nothing was applied and ErrMissingDown
// when the latest migration has no Down section.
func Down(ctx context.Context, db DB, fsys fs.FS, dir string) error {
	migrations, err := loadMigrations(fsys, dir)
	if err != nil {
		return err
	}

	return transaction(ctx, db, func(tx Tx) error {
		records, err := tx.Applied(ctx)
		if err != nil {
			return fmt.Errorf("sqlmigrate: load applied migrations: %w", err)
		}
		if len(records) == 0 {
			return ErrNoAppliedMigrations
		}
		if err := markApplied(migrations, records, dir); err != nil {
			return err
		}

		var last migrationFile
		for _, m := range slices.Backward(migrations) {
			if m.applied {
				last = m
				break
			}
		}
		if !last.hasDown {
			return &MissingDownError{Migration: last.name}
		}
		if err := tx.Exec(ctx, last.down); err != nil {
			return fmt.Errorf("sqlmigrate: run %s down: %w", last.name, err)
		}
		if err := tx.Forget(ctx, last.name); err != nil {
			return fmt.Errorf("sqlmigrate: forget %s: %w", last.name, err)
		}
		return nil
	})
}

// transaction opens a locked transaction and commits it when fn succeeds.
func transaction(ctx context.Context, db DB, fn func(Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("sqlmigrate: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tx.Lock(ctx); err != nil {
		return fmt.Errorf("sqlmigrate: acquire lock: %w", err)
	}
	if err := tx.EnsureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("sqlmigrate: ensure migrations table: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("sqlmigrate: commit: %w", err)
	}
	return nil
}

// markApplied verifies the applied records against the migration files, marks
// matching files, and fails when a checksum changed or a record has no file.
func markApplied(migrations []migrationFile, records []AppliedMigration, dir string) error {
	slices.SortFunc(records, func(a, b AppliedMigration) int {
		return cmp.Compare(a.Migration, b.Migration)
	})

	for record := range slices.Values(records) {
		i, ok := slices.BinarySearchFunc(migrations, record.Migration, func(m migrationFile, name string) int {
			return cmp.Compare(m.name, name)
		})
		if !ok {
			return &StaleMigrationError{Migration: record.Migration, Dir: dir}
		}
		if migrations[i].checksum != record.Checksum {
			return &ChecksumMismatchError{Migration: record.Migration, Stored: record.Checksum, File: migrations[i].checksum}
		}
		migrations[i].applied = true
	}
	return nil
}

// migrationFile is one parsed .sql migration file.
type migrationFile struct {
	name     string
	up       string
	down     string
	hasDown  bool
	checksum string
	applied  bool
}

// parseMigration splits a migration file into Up and Down sections. A file
// without directives is treated as a plain Up-only migration.
func parseMigration(name string, content []byte) (migrationFile, error) {
	m := migrationFile{name: name, checksum: fileChecksum(content)}

	var up, down strings.Builder
	var section string

	for line := range strings.Lines(string(content)) {
		match := directivePattern.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			switch section {
			case "up":
				up.WriteString(line)
			case "down":
				down.WriteString(line)
			}
			continue
		}

		next := strings.ToLower(match[1])
		switch {
		case next == section:
			return migrationFile{}, fmt.Errorf("sqlmigrate: %s: duplicate %s section", name, next)
		case next == "up" && section == "down":
			return migrationFile{}, fmt.Errorf("sqlmigrate: %s: Up section after Down section", name)
		case next == "down" && section == "":
			return migrationFile{}, fmt.Errorf("sqlmigrate: %s: Down section before Up section", name)
		}
		section = next
	}

	if section == "" {
		m.up = string(content)
		return m, nil
	}
	m.up = strings.TrimSpace(up.String())
	m.down = strings.TrimSpace(down.String())
	m.hasDown = section == "down"
	return m, nil
}

// loadMigrations reads every .sql file in dir sorted by name.
func loadMigrations(fsys fs.FS, dir string) ([]migrationFile, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("sqlmigrate: read migrations from %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)

	migrations := make([]migrationFile, 0, len(names))
	for name := range slices.Values(names) {
		content, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("sqlmigrate: read %s: %w", name, err)
		}
		m, err := parseMigration(name, content)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}
	return migrations, nil
}

func fileChecksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
