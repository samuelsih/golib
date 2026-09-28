package pgx

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/samuelsih/golib/sqlmigration"
)

// Transactor is implemented by *pgxpool.Pool and *pgx.Conn.
type Transactor interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Adapter implements sqlmigration.DB on top of a pgx pool or connection.
type Adapter struct {
	db Transactor
}

// New returns an Adapter backed by db, which may be a *pgxpool.Pool or a
// *pgx.Conn.
func New(db Transactor) *Adapter {
	return &Adapter{db: db}
}

func (a *Adapter) Begin(ctx context.Context) (sqlmigration.Tx, error) {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &transaction{tx: tx}, nil
}

type transaction struct {
	tx pgx.Tx
}

func (t *transaction) Lock(ctx context.Context) error {
	_, err := t.tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('sqlmigration'))`)
	return err
}

func (t *transaction) EnsureMigrationsTable(ctx context.Context) error {
	_, err := t.tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS sql_migrations (
	id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	migration VARCHAR(255) NOT NULL UNIQUE,
	checksum CHAR(64) NOT NULL,
	applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
	return err
}

func (t *transaction) Applied(ctx context.Context) ([]sqlmigration.AppliedMigration, error) {
	rows, err := t.tx.Query(ctx, `SELECT migration, checksum FROM sql_migrations ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applied []sqlmigration.AppliedMigration
	for rows.Next() {
		var record sqlmigration.AppliedMigration
		if err := rows.Scan(&record.Migration, &record.Checksum); err != nil {
			return nil, err
		}
		applied = append(applied, record)
	}
	return applied, rows.Err()
}

func (t *transaction) Exec(ctx context.Context, script string) error {
	_, err := t.tx.Exec(ctx, script)
	return err
}

func (t *transaction) Record(ctx context.Context, applied sqlmigration.AppliedMigration) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO sql_migrations (migration, checksum) VALUES ($1, $2)`, applied.Migration, applied.Checksum)
	return err
}

func (t *transaction) Forget(ctx context.Context, migration string) error {
	_, err := t.tx.Exec(ctx, `DELETE FROM sql_migrations WHERE migration = $1`, migration)
	return err
}

func (t *transaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *transaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}
