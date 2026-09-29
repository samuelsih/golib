// Command sqlmigrate runs SQL migrations from the ./migrations directory.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samuelsih/golib/sqlmigration"
	pgxadapter "github.com/samuelsih/golib/sqlmigration/pgx"
)

const (
	defaultMigrationsDir = "migrations"
	defaultDriver        = "postgres"

	usage = `Usage: sqlmigrate <command> [flags]

Commands:
  new <name>  create a migration file
  up          apply all pending migrations
  down        roll back the latest applied migration

Flags:
  --database-driver string  database driver (default "postgres")
  --database-url string     database connection string
  --migrations-dir string   migrations directory (default "migrations")

Environment:
  DATABASE_DRIVER  database driver (default "postgres")
  DATABASE_URL     database connection string
  MIGRATIONS_DIR   migrations directory (default "migrations")

Flags take precedence over environment variables.`
)

var commands = map[string]func(context.Context, sqlmigration.DB, fs.FS, string) error{
	"up":   sqlmigration.Up,
	"down": sqlmigration.Down,
}

var drivers = map[string]func(context.Context, string) (sqlmigration.DB, func(), error){
	"postgres": func(ctx context.Context, url string) (sqlmigration.DB, func(), error) {
		pool, err := pgxpool.New(ctx, url)
		if err != nil {
			return nil, nil, err
		}
		return pgxadapter.New(pool), pool.Close, nil
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if args[0] == "new" {
		return newMigration(args[1:])
	}

	migrate, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown command %q", args[0])
	}
	return runMigrate(args[0], migrate, args[1:])
}

func runMigrate(name string, migrate func(context.Context, sqlmigration.DB, fs.FS, string) error, args []string) error {
	flags := newFlagSet(name)
	url := flags.String("database-url", "", "database connection string")
	driver := flags.String("database-driver", "", "database driver")
	dir := flags.String("migrations-dir", "", "migrations directory")

	if err := flags.Parse(args); err != nil {
		return parseError(err)
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("usage: sqlmigrate %s [--database-url url] [--database-driver driver] [--migrations-dir dir]", name)
	}

	driverName := setting(*driver, "DATABASE_DRIVER", defaultDriver)
	migrationsDir := setting(*dir, "MIGRATIONS_DIR", defaultMigrationsDir)
	databaseURL := setting(*url, "DATABASE_URL", "")
	if databaseURL == "" {
		return errors.New("--database-url or DATABASE_URL is required")
	}
	if _, err := os.Stat(migrationsDir); err != nil { //nolint:gosec // the directory is chosen by the operator running the CLI
		return fmt.Errorf("migrations directory: %w", err)
	}

	ctx := context.Background()
	db, closeDB, err := openDB(ctx, driverName, databaseURL)
	if err != nil {
		return err
	}
	defer closeDB()

	return migrate(ctx, db, os.DirFS(migrationsDir), ".")
}

func newMigration(args []string) error {
	flags := newFlagSet("new")
	dir := flags.String("migrations-dir", "", "migrations directory")

	if err := flags.Parse(args); err != nil {
		return parseError(err)
	}
	if flags.NArg() != 1 {
		return errors.New("usage: sqlmigrate new [--migrations-dir dir] <name>")
	}

	filename, err := sqlmigration.New(setting(*dir, "MIGRATIONS_DIR", defaultMigrationsDir), flags.Arg(0))
	if err != nil {
		return err
	}
	fmt.Println(filename)
	return nil
}

func openDB(ctx context.Context, driver, url string) (sqlmigration.DB, func(), error) {
	open, ok := drivers[driver]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported database driver %q", driver)
	}
	return open(ctx, url)
}

func newFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func parseError(err error) error {
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, usage)
		return nil
	}
	return err
}

// setting resolves a value in flag > environment > default order.
func setting(flagValue, env, fallback string) string {
	if flagValue != "" {
		return flagValue
	}
	if value := os.Getenv(env); value != "" {
		return value
	}
	return fallback
}
