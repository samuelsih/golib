package sqlmigration

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/samuelsih/golib/assert"
)

const (
	firstMigration  = "migrations/1_first.sql"
	secondMigration = "migrations/2_second.sql"
	thirdMigration  = "migrations/3_third.sql"
)

var errBoom = errors.New("boom")

func newFS(files map[string]string) fstest.MapFS {
	fsys := make(fstest.MapFS, len(files))
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

func upDown(up, down string) string {
	return "-- +migrate Up\n" + up + "\n\n-- +migrate Down\n" + down + "\n"
}

func defaultFS() fstest.MapFS {
	return newFS(map[string]string{
		firstMigration:  upDown("CREATE TABLE first;", "DROP TABLE first;"),
		secondMigration: upDown("CREATE TABLE second;", "DROP TABLE second;"),
		thirdMigration:  upDown("CREATE TABLE third;", "DROP TABLE third;"),
	})
}

func appliedOf(t *testing.T, fsys fs.FS, names ...string) map[string]string {
	t.Helper()

	applied := make(map[string]string, len(names))
	for name := range slices.Values(names) {
		applied[path.Base(name)] = checksumOf(t, fsys, name)
	}
	return applied
}

func checksumOf(t *testing.T, fsys fs.FS, name string) string {
	t.Helper()

	content, err := fs.ReadFile(fsys, name)
	assert.NoError(t, err)
	return fileChecksum(content)
}

func seed(t *testing.T, db *fakeDB, fsys fs.FS, names ...string) {
	t.Helper()
	maps.Copy(db.records, appliedOf(t, fsys, names...))
}

func TestFileChecksum(t *testing.T) {
	assert.Equal(t, fileChecksum(nil), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	assert.Equal(t, fileChecksum([]byte("hello")), "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
	assert.Equal(t, len(fileChecksum([]byte("x"))), 64)
}

func TestParseMigration(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    migrationFile
	}{
		{
			name:    "up and down",
			content: "-- +migrate Up\nCREATE TABLE t;\n\n-- +migrate Down\nDROP TABLE t;\n",
			want:    migrationFile{name: "1_a.sql", up: "CREATE TABLE t;", down: "DROP TABLE t;", hasDown: true},
		},
		{
			name:    "case insensitive and spaced directives",
			content: "--+migrate UP\nCREATE 1;\n  --  +migrate  down  \nDROP 1;\n",
			want:    migrationFile{name: "1_a.sql", up: "CREATE 1;", down: "DROP 1;", hasDown: true},
		},
		{
			name:    "plain sql file",
			content: "SELECT 1;",
			want:    migrationFile{name: "1_a.sql", up: "SELECT 1;"},
		},
		{
			name:    "comment before up",
			content: "-- header\n-- +migrate Up\nSELECT 2;",
			want:    migrationFile{name: "1_a.sql", up: "SELECT 2;"},
		},
		{
			name:    "up only",
			content: "-- +migrate Up\nSELECT 3;\n",
			want:    migrationFile{name: "1_a.sql", up: "SELECT 3;"},
		},
		{
			name:    "empty down",
			content: "-- +migrate Up\nSELECT 4;\n-- +migrate Down\n",
			want:    migrationFile{name: "1_a.sql", up: "SELECT 4;", hasDown: true},
		},
		{
			name:    "empty file",
			content: "",
			want:    migrationFile{name: "1_a.sql"},
		},
		{
			name:    "windows line endings",
			content: "-- +migrate Up\r\nSELECT 5;\r\n-- +migrate Down\r\nDROP 5;\r\n",
			want:    migrationFile{name: "1_a.sql", up: "SELECT 5;", down: "DROP 5;", hasDown: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMigration("1_a.sql", []byte(tt.content))
			assert.NoError(t, err)

			tt.want.checksum = fileChecksum([]byte(tt.content))
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestParseMigrationErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"duplicate up", "-- +migrate Up\n-- +migrate Up\n", "sqlmigrate: 1_a.sql: duplicate up section"},
		{"duplicate down", "-- +migrate Up\n-- +migrate Down\n-- +migrate Down\n", "sqlmigrate: 1_a.sql: duplicate down section"},
		{"up after down", "-- +migrate Up\n-- +migrate Down\n-- +migrate Up\n", "sqlmigrate: 1_a.sql: Up section after Down section"},
		{"down before up", "-- +migrate Down\n-- +migrate Up\n", "sqlmigrate: 1_a.sql: Down section before Up section"},
		{"down only", "-- +migrate Down\n", "sqlmigrate: 1_a.sql: Down section before Up section"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMigration("1_a.sql", []byte(tt.content))
			assert.Equal(t, got, migrationFile{})
			assert.NotNil(t, err)
			if err != nil {
				assert.Equal(t, err.Error(), tt.want)
			}
		})
	}
}

func TestLoadMigrations(t *testing.T) {
	fsys := defaultFS()
	fsys["migrations/notes.txt"] = &fstest.MapFile{Data: []byte("ignore me")}
	fsys["migrations/nested/4_fourth.sql"] = &fstest.MapFile{Data: []byte(upDown("CREATE 4;", "DROP 4;"))}

	got, err := loadMigrations(fsys, "migrations")
	assert.NoError(t, err)

	want := []migrationFile{
		{
			name:     "1_first.sql",
			up:       "CREATE TABLE first;",
			down:     "DROP TABLE first;",
			hasDown:  true,
			checksum: fileChecksum([]byte(upDown("CREATE TABLE first;", "DROP TABLE first;"))),
		},
		{
			name:     "2_second.sql",
			up:       "CREATE TABLE second;",
			down:     "DROP TABLE second;",
			hasDown:  true,
			checksum: fileChecksum([]byte(upDown("CREATE TABLE second;", "DROP TABLE second;"))),
		},
		{
			name:     "3_third.sql",
			up:       "CREATE TABLE third;",
			down:     "DROP TABLE third;",
			hasDown:  true,
			checksum: fileChecksum([]byte(upDown("CREATE TABLE third;", "DROP TABLE third;"))),
		},
	}
	assert.Equal(t, got, want)
}

func TestLoadMigrationsMissingDir(t *testing.T) {
	_, err := loadMigrations(newFS(nil), "migrations")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestLoadMigrationsEmptyDir(t *testing.T) {
	got, err := loadMigrations(newFS(map[string]string{"migrations/.keep": ""}), "migrations")
	assert.NoError(t, err)
	assert.Equal(t, len(got), 0)
}

func TestLoadMigrationsParseError(t *testing.T) {
	fsys := newFS(map[string]string{"migrations/1_bad.sql": "-- +migrate Up\n-- +migrate Up\n"})

	_, err := loadMigrations(fsys, "migrations")
	assert.NotNil(t, err)
	if err != nil {
		assert.Equal(t, err.Error(), "sqlmigrate: 1_bad.sql: duplicate up section")
	}
}

func TestMarkApplied(t *testing.T) {
	fsys := defaultFS()
	migrations, err := loadMigrations(fsys, "migrations")
	assert.NoError(t, err)

	applied := appliedOf(t, fsys, firstMigration, thirdMigration)
	records := make([]AppliedMigration, 0, len(applied))
	for name, checksum := range applied {
		records = append(records, AppliedMigration{Migration: name, Checksum: checksum})
	}

	assert.NoError(t, markApplied(migrations, records, "migrations"))
	assert.True(t, migrations[0].applied)
	assert.False(t, migrations[1].applied)
	assert.True(t, migrations[2].applied)
}

func TestMarkAppliedChecksumMismatch(t *testing.T) {
	fsys := defaultFS()
	migrations, err := loadMigrations(fsys, "migrations")
	assert.NoError(t, err)

	err = markApplied(migrations, []AppliedMigration{{Migration: "2_second.sql", Checksum: "stale"}}, "migrations")
	assert.Equal(t, err, &ChecksumMismatchError{
		Migration: "2_second.sql",
		Stored:    "stale",
		File:      checksumOf(t, fsys, secondMigration),
	})
}

func TestMarkAppliedStale(t *testing.T) {
	fsys := defaultFS()
	migrations, err := loadMigrations(fsys, "migrations")
	assert.NoError(t, err)

	err = markApplied(migrations, []AppliedMigration{{Migration: "0_gone.sql", Checksum: "x"}}, "migrations")
	assert.Equal(t, err, &StaleMigrationError{Migration: "0_gone.sql", Dir: "migrations"})
}

func TestMarkAppliedStalePicksFirst(t *testing.T) {
	fsys := defaultFS()
	migrations, err := loadMigrations(fsys, "migrations")
	assert.NoError(t, err)

	records := []AppliedMigration{
		{Migration: "9_z.sql", Checksum: "x"},
		{Migration: "0_a.sql", Checksum: "y"},
	}

	err = markApplied(migrations, records, "migrations")
	assert.Equal(t, err, &StaleMigrationError{Migration: "0_a.sql", Dir: "migrations"})
}

func TestTransaction(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*fakeDB)
		fn          func(Tx) error
		wantErr     string
		wantIs      error
		wantScripts []string
		commits     int
		rollbacks   int
	}{
		{
			name:        "success",
			fn:          func(tx Tx) error { return tx.Exec(t.Context(), "SELECT 1;") },
			wantScripts: []string{"SELECT 1;"},
			commits:     1,
			rollbacks:   1,
		},
		{
			name:      "begin error",
			setup:     func(db *fakeDB) { db.beginErr = errBoom },
			fn:        func(Tx) error { return nil },
			wantErr:   "sqlmigrate: begin transaction: boom",
			wantIs:    errBoom,
			commits:   0,
			rollbacks: 0,
		},
		{
			name:      "lock error",
			setup:     func(db *fakeDB) { db.lockErr = errBoom },
			fn:        func(Tx) error { return nil },
			wantErr:   "sqlmigrate: acquire lock: boom",
			wantIs:    errBoom,
			commits:   0,
			rollbacks: 1,
		},
		{
			name:      "ensure error",
			setup:     func(db *fakeDB) { db.ensureErr = errBoom },
			fn:        func(Tx) error { return nil },
			wantErr:   "sqlmigrate: ensure migrations table: boom",
			wantIs:    errBoom,
			commits:   0,
			rollbacks: 1,
		},
		{
			name:      "fn error",
			fn:        func(Tx) error { return errBoom },
			wantErr:   "boom",
			wantIs:    errBoom,
			commits:   0,
			rollbacks: 1,
		},
		{
			name:      "commit error",
			setup:     func(db *fakeDB) { db.commitErr = errBoom },
			fn:        func(Tx) error { return nil },
			wantErr:   "sqlmigrate: commit: boom",
			wantIs:    errBoom,
			commits:   0,
			rollbacks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newFakeDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			err := transaction(t.Context(), db, tt.fn)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, err.Error(), tt.wantErr)
				}
			}
			if tt.wantIs != nil {
				assert.ErrorIs(t, err, tt.wantIs)
			}
			assert.Equal(t, db.scripts, tt.wantScripts)
			assert.Equal(t, db.commits, tt.commits)
			assert.Equal(t, db.rollbacks, tt.rollbacks)
		})
	}
}

func TestUpAppliesPending(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()

	assert.NoError(t, Up(t.Context(), db, fsys, "migrations"))

	assert.Equal(t, db.scripts, []string{
		"CREATE TABLE first;",
		"CREATE TABLE second;",
		"CREATE TABLE third;",
	})
	assert.Equal(t, db.records, appliedOf(t, fsys, firstMigration, secondMigration, thirdMigration))
	assert.Equal(t, db.commits, 1)
}

func TestUpSkipsApplied(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	seed(t, db, fsys, firstMigration)

	assert.NoError(t, Up(t.Context(), db, fsys, "migrations"))

	assert.Equal(t, db.scripts, []string{"CREATE TABLE second;", "CREATE TABLE third;"})
	assert.Equal(t, db.records, appliedOf(t, fsys, firstMigration, secondMigration, thirdMigration))
}

func TestUpAlreadyApplied(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	seed(t, db, fsys, firstMigration, secondMigration, thirdMigration)

	assert.NoError(t, Up(t.Context(), db, fsys, "migrations"))

	assert.Nil(t, db.scripts)
	assert.Equal(t, db.commits, 1)
}

func TestUpChecksumMismatch(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	db.records["1_first.sql"] = "mismatch"

	err := Up(t.Context(), db, fsys, "migrations")
	assert.Equal(t, err, &ChecksumMismatchError{
		Migration: "1_first.sql",
		Stored:    "mismatch",
		File:      checksumOf(t, fsys, firstMigration),
	})
	assert.Nil(t, db.scripts)
	assert.Equal(t, db.commits, 0)
}

func TestUpStaleRecord(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	db.records["0_gone.sql"] = "x"

	err := Up(t.Context(), db, fsys, "migrations")
	assert.Equal(t, err, &StaleMigrationError{Migration: "0_gone.sql", Dir: "migrations"})
	assert.Nil(t, db.scripts)
	assert.Equal(t, db.commits, 0)
}

func TestUpRunError(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	db.execErr = errBoom

	err := Up(t.Context(), db, fsys, "migrations")
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, err.Error(), "sqlmigrate: run 1_first.sql: boom")
	assert.Equal(t, db.commits, 0)
	assert.Equal(t, len(db.records), 0)
}

func TestUpRecordError(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	db.recordErr = errBoom

	err := Up(t.Context(), db, fsys, "migrations")
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, err.Error(), "sqlmigrate: record 1_first.sql: boom")
	assert.Equal(t, db.scripts, []string{"CREATE TABLE first;"})
	assert.Equal(t, db.commits, 0)
}

func TestUpAppliedError(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	db.appliedErr = errBoom

	err := Up(t.Context(), db, fsys, "migrations")
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, err.Error(), "sqlmigrate: load applied migrations: boom")
	assert.Equal(t, db.commits, 0)
}

func TestUpLoadError(t *testing.T) {
	db := newFakeDB()

	err := Up(t.Context(), db, newFS(nil), "migrations")
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Equal(t, db.begins, 0)
}

func TestDownRollsBackLatest(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	seed(t, db, fsys, firstMigration, secondMigration, thirdMigration)

	assert.NoError(t, Down(t.Context(), db, fsys, "migrations"))

	assert.Equal(t, db.scripts, []string{"DROP TABLE third;"})
	assert.Equal(t, db.records, appliedOf(t, fsys, firstMigration, secondMigration))
	assert.Equal(t, db.commits, 1)
}

func TestDownRollsBackHighestApplied(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	seed(t, db, fsys, firstMigration, thirdMigration)

	assert.NoError(t, Down(t.Context(), db, fsys, "migrations"))

	assert.Equal(t, db.scripts, []string{"DROP TABLE third;"})
	assert.Equal(t, db.records, appliedOf(t, fsys, firstMigration))
}

func TestDownNoAppliedMigrations(t *testing.T) {
	db := newFakeDB()

	err := Down(t.Context(), db, defaultFS(), "migrations")
	assert.ErrorIs(t, err, ErrNoAppliedMigrations)
	assert.Nil(t, db.scripts)
	assert.Equal(t, db.commits, 0)
	assert.Equal(t, db.rollbacks, 1)
}

func TestDownMissingDownSection(t *testing.T) {
	const plain = "migrations/1_plain.sql"
	fsys := newFS(map[string]string{plain: "CREATE TABLE plain;"})
	db := newFakeDB()
	seed(t, db, fsys, plain)

	err := Down(t.Context(), db, fsys, "migrations")
	assert.Equal(t, err, &MissingDownError{Migration: "1_plain.sql"})
	assert.Nil(t, db.scripts)
	assert.Equal(t, db.records, appliedOf(t, fsys, plain))
	assert.Equal(t, db.commits, 0)
}

func TestDownRunError(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	seed(t, db, fsys, firstMigration)
	db.execErr = errBoom

	err := Down(t.Context(), db, fsys, "migrations")
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, err.Error(), "sqlmigrate: run 1_first.sql down: boom")
	assert.Equal(t, db.records, appliedOf(t, fsys, firstMigration))
	assert.Equal(t, db.commits, 0)
}

func TestDownForgetError(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	seed(t, db, fsys, firstMigration)
	db.forgetErr = errBoom

	err := Down(t.Context(), db, fsys, "migrations")
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, err.Error(), "sqlmigrate: forget 1_first.sql: boom")
	assert.Equal(t, db.scripts, []string{"DROP TABLE first;"})
	assert.Equal(t, db.records, appliedOf(t, fsys, firstMigration))
	assert.Equal(t, db.commits, 0)
}

func TestDownChecksumMismatch(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	db.records["1_first.sql"] = "mismatch"

	err := Down(t.Context(), db, fsys, "migrations")
	assert.ErrorIs(t, err, ErrChecksumMismatch)
	assert.Equal(t, err, &ChecksumMismatchError{
		Migration: "1_first.sql",
		Stored:    "mismatch",
		File:      checksumOf(t, fsys, firstMigration),
	})
	assert.Equal(t, db.commits, 0)
}

func TestDownStaleRecord(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()
	db.records["0_gone.sql"] = "x"

	err := Down(t.Context(), db, fsys, "migrations")
	assert.Equal(t, err, &StaleMigrationError{Migration: "0_gone.sql", Dir: "migrations"})
	assert.Equal(t, db.commits, 0)
}

func TestUpDownRoundTrip(t *testing.T) {
	fsys := defaultFS()
	db := newFakeDB()

	assert.NoError(t, Up(t.Context(), db, fsys, "migrations"))
	assert.NoError(t, Down(t.Context(), db, fsys, "migrations"))

	assert.Equal(t, db.records, appliedOf(t, fsys, firstMigration, secondMigration))
}

// fakeDB is an in-memory DB used to exercise the runner without a database.
type fakeDB struct {
	records   map[string]string
	scripts   []string
	begins    int
	commits   int
	rollbacks int
	locked    bool

	beginErr   error
	lockErr    error
	ensureErr  error
	appliedErr error
	execErr    error
	recordErr  error
	forgetErr  error
	commitErr  error
}

func newFakeDB() *fakeDB {
	return &fakeDB{records: make(map[string]string)}
}

func (db *fakeDB) Begin(context.Context) (Tx, error) {
	db.begins++
	if db.beginErr != nil {
		return nil, db.beginErr
	}
	return &fakeTx{db: db, working: maps.Clone(db.records)}, nil
}

type fakeTx struct {
	db      *fakeDB
	working map[string]string
	closed  bool
}

func (tx *fakeTx) Lock(context.Context) error {
	if tx.db.lockErr != nil {
		return tx.db.lockErr
	}
	if tx.db.locked {
		return errors.New("migration lock already held")
	}
	tx.db.locked = true
	return nil
}

func (tx *fakeTx) EnsureMigrationsTable(context.Context) error {
	return tx.db.ensureErr
}

func (tx *fakeTx) Applied(context.Context) ([]AppliedMigration, error) {
	if tx.db.appliedErr != nil {
		return nil, tx.db.appliedErr
	}

	names := slices.Sorted(maps.Keys(tx.working))
	records := make([]AppliedMigration, 0, len(names))
	for name := range slices.Values(names) {
		records = append(records, AppliedMigration{Migration: name, Checksum: tx.working[name]})
	}
	return records, nil
}

func (tx *fakeTx) Exec(_ context.Context, script string) error {
	if tx.db.execErr != nil {
		return tx.db.execErr
	}
	tx.db.scripts = append(tx.db.scripts, script)
	return nil
}

func (tx *fakeTx) Record(_ context.Context, applied AppliedMigration) error {
	if tx.db.recordErr != nil {
		return tx.db.recordErr
	}
	tx.working[applied.Migration] = applied.Checksum
	return nil
}

func (tx *fakeTx) Forget(_ context.Context, migration string) error {
	if tx.db.forgetErr != nil {
		return tx.db.forgetErr
	}
	delete(tx.working, migration)
	return nil
}

func (tx *fakeTx) Commit(context.Context) error {
	if tx.db.commitErr != nil {
		return tx.db.commitErr
	}
	tx.db.records = tx.working
	tx.db.commits++
	tx.db.locked = false
	tx.closed = true
	return nil
}

func (tx *fakeTx) Rollback(context.Context) error {
	tx.db.rollbacks++
	tx.db.locked = false
	if tx.closed {
		return errors.New("transaction already closed")
	}
	return nil
}
