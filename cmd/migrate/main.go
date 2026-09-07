// Command migrate applies the SQL migrations in db/migrations to the configured
// PostgreSQL database.
//
// Usage:
//
//	migrate [flags] <command> [args]
//
// Commands:
//
//	up [N]        apply all pending migrations, or the next N
//	down [N]      roll back the last N migrations (default 1)
//	version       print the current schema version and dirty flag
//	force <V>     record version V and clear the dirty flag, without running SQL
//	status        list every migration and whether it has been applied
//
// Flags:
//
//	-path <dir>   directory holding NNNNNN_name.{up,down}.sql files
//	              (default: $KERP_MIGRATIONS_PATH, else the first of
//	              /app/db/migrations, /app/migrations, db/migrations that exists)
//	-timeout <d>  overall timeout (default 5m)
//	-yes          required by "down" when N would roll back more than one step
//
// The database connection comes from the same configuration the API uses
// (config/app.yaml plus KERP_DATABASE_* environment variables), so no separate
// DSN needs to be supplied in deployment.
//
// State is kept in the schema_migrations table, in the layout golang-migrate
// uses (a single row: version bigint primary key, dirty boolean), so the
// golang-migrate CLI can be pointed at the same database if that is ever
// preferred. Each migration runs inside its own transaction, and a session-level
// advisory lock keeps two instances from migrating at the same time — which is
// what happens when a deployment starts more than one API replica.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/saintgo7/saas-kerp/internal/config"
	"github.com/saintgo7/saas-kerp/internal/database"
)

// advisoryLockID is an arbitrary but fixed identifier for the migration lock.
const advisoryLockID int64 = 6045271849302

// migrationFilePattern matches "000012_tax_invoices.up.sql".
var migrationFilePattern = regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

type migration struct {
	version  int64
	name     string
	upPath   string
	downPath string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		usage()
		return errors.New("no command given")
	}

	command := os.Args[1]
	if command == "-h" || command == "--help" || command == "help" {
		usage()
		return nil
	}

	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	path := fs.String("path", defaultMigrationsPath(), "directory containing the migration files")
	timeout := fs.Duration("timeout", 5*time.Minute, "overall timeout")
	confirm := fs.Bool("yes", false, "confirm a multi-step down migration")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}
	args := fs.Args()

	migrations, err := loadMigrations(*path)
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		return fmt.Errorf("no migration files found in %s", *path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	db, closeDB, err := openDB()
	if err != nil {
		return err
	}
	defer closeDB()

	// Hold the migration lock on one dedicated session for the whole run.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire a connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return fmt.Errorf("failed to acquire the migration lock: %w", err)
	}
	defer func() {
		// Best effort: the lock is released with the session in any case.
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", advisoryLockID)
	}()

	if err := ensureVersionTable(ctx, conn); err != nil {
		return err
	}

	version, dirty, err := currentVersion(ctx, conn)
	if err != nil {
		return err
	}

	switch command {
	case "version":
		printVersion(version, dirty)
		return nil

	case "status":
		printVersion(version, dirty)
		fmt.Println()
		for _, m := range migrations {
			state := "pending"
			if m.version <= version {
				state = "applied"
			}
			fmt.Printf("  %06d  %-40s %s\n", m.version, m.name, state)
		}
		return nil

	case "force":
		if len(args) != 1 {
			return errors.New("force requires exactly one version argument")
		}
		target, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version %q: %w", args[0], err)
		}
		if err := setVersion(ctx, conn, target, false); err != nil {
			return err
		}
		fmt.Printf("forced version to %d and cleared the dirty flag\n", target)
		return nil

	case "up":
		if dirty {
			return dirtyError(version)
		}
		limit, err := stepArg(args, 0)
		if err != nil {
			return err
		}
		return migrateUp(ctx, conn, migrations, version, limit)

	case "down":
		if dirty {
			return dirtyError(version)
		}
		limit, err := stepArg(args, 1)
		if err != nil {
			return err
		}
		if limit != 1 && !*confirm {
			return fmt.Errorf("rolling back %d migrations destroys data; re-run with -yes to confirm", limit)
		}
		return migrateDown(ctx, conn, migrations, version, limit)

	default:
		usage()
		return fmt.Errorf("unknown command %q", command)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: migrate [flags] <command> [args]

commands:
  up [N]      apply all pending migrations, or the next N
  down [N]    roll back the last N migrations (default 1; N>1 needs -yes)
  version     print the current schema version and dirty flag
  force <V>   record version V and clear the dirty flag, running no SQL
  status      list every migration and whether it has been applied

flags:
  -path <dir>    migration directory (default $KERP_MIGRATIONS_PATH, else the first of
                 /app/db/migrations, /app/migrations, db/migrations that exists)
  -timeout <d>   overall timeout (default 5m)
  -yes           confirm a multi-step down migration

The database connection is read from the API configuration
(config/app.yaml and the KERP_DATABASE_* environment variables).
`)
}

// defaultMigrationsPath locates the migration directory without depending on the
// working directory: `docker compose exec` need not run with the image's
// WORKDIR, and a relative path would then resolve to nothing.
//
// deployments/docker/Dockerfile.api copies the files to /app/db/migrations.
func defaultMigrationsPath() string {
	if p := os.Getenv("KERP_MIGRATIONS_PATH"); p != "" {
		return p
	}

	for _, candidate := range []string{
		"/app/db/migrations", // container image layout
		"/app/migrations",
		"db/migrations", // repository checkout
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return "db/migrations"
}

// openDB builds the connection from the API configuration so that migrations
// and the API can never disagree about which database they are talking to.
func openDB() (*sql.DB, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	dbCfg := cfg.Database
	// The per-request tenant session has no meaning here, and migrations must be
	// able to see and change every row.
	dbCfg.TenantGUC = false

	gormDB, err := database.NewPostgresDB(&dbCfg, nil)
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, err
	}

	return sqlDB, func() { _ = sqlDB.Close() }, nil
}

func stepArg(args []string, def int) (int, error) {
	if len(args) == 0 {
		return def, nil
	}
	if len(args) > 1 {
		return 0, fmt.Errorf("unexpected arguments: %s", strings.Join(args[1:], " "))
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid step count %q", args[0])
	}
	return n, nil
}

func dirtyError(version int64) error {
	return fmt.Errorf(
		"database is marked dirty at version %d: a previous migration failed halfway.\n"+
			"Inspect the schema, finish or undo that migration by hand, then run: migrate force %d",
		version, version)
}

func printVersion(version int64, dirty bool) {
	if version == 0 {
		fmt.Println("version: 0 (no migrations applied)")
	} else {
		fmt.Printf("version: %d\n", version)
	}
	fmt.Printf("dirty:   %t\n", dirty)
}

// loadMigrations reads the migration directory and pairs up/down files.
func loadMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory %s: %w", dir, err)
	}

	byVersion := make(map[int64]*migration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid version in %s: %w", entry.Name(), err)
		}
		if version <= 0 {
			return nil, fmt.Errorf("migration versions start at 1, got %d in %s", version, entry.Name())
		}

		m, ok := byVersion[version]
		if !ok {
			m = &migration{version: version, name: matches[2]}
			byVersion[version] = m
		}
		full := filepath.Join(dir, entry.Name())
		if matches[3] == "up" {
			m.upPath = full
		} else {
			m.downPath = full
		}
	}

	out := make([]migration, 0, len(byVersion))
	for _, m := range byVersion {
		if m.upPath == "" {
			return nil, fmt.Errorf("migration %06d (%s) has no .up.sql file", m.version, m.name)
		}
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func ensureVersionTable(ctx context.Context, conn *sql.Conn) error {
	const stmt = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT NOT NULL PRIMARY KEY,
		dirty   BOOLEAN NOT NULL
	)`
	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create schema_migrations: %w", err)
	}
	return nil
}

func currentVersion(ctx context.Context, conn *sql.Conn) (int64, bool, error) {
	var version int64
	var dirty bool
	err := conn.QueryRowContext(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("failed to read schema_migrations: %w", err)
	}
	return version, dirty, nil
}

// setVersion replaces the single row in schema_migrations, matching the layout
// golang-migrate maintains.
func setVersion(ctx context.Context, conn *sql.Conn, version int64, dirty bool) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM schema_migrations"); err != nil {
		return err
	}
	if version > 0 {
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, dirty) VALUES ($1, $2)", version, dirty); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func migrateUp(ctx context.Context, conn *sql.Conn, migrations []migration, current int64, limit int) error {
	pending := make([]migration, 0, len(migrations))
	for _, m := range migrations {
		if m.version > current {
			pending = append(pending, m)
		}
	}
	if limit > 0 && limit < len(pending) {
		pending = pending[:limit]
	}

	if len(pending) == 0 {
		fmt.Printf("no pending migrations (version %d)\n", current)
		return nil
	}

	restore := current
	for _, m := range pending {
		if err := applyMigration(ctx, conn, m, m.upPath, m.version, restore); err != nil {
			return err
		}
		fmt.Printf("applied  %06d  %s\n", m.version, m.name)
		restore = m.version
	}

	fmt.Printf("done: version %d\n", pending[len(pending)-1].version)
	return nil
}

func migrateDown(ctx context.Context, conn *sql.Conn, migrations []migration, current int64, limit int) error {
	applied := make([]migration, 0, len(migrations))
	for _, m := range migrations {
		if m.version <= current {
			applied = append(applied, m)
		}
	}
	if len(applied) == 0 {
		fmt.Println("nothing to roll back (version 0)")
		return nil
	}
	if limit > len(applied) {
		limit = len(applied)
	}

	// Roll back newest first.
	for i := 0; i < limit; i++ {
		m := applied[len(applied)-1-i]
		if m.downPath == "" {
			return fmt.Errorf("migration %06d (%s) has no .down.sql file; cannot roll back", m.version, m.name)
		}

		// After undoing version V, the schema is at the previous version.
		target := int64(0)
		if idx := len(applied) - 2 - i; idx >= 0 {
			target = applied[idx].version
		}

		if err := applyMigration(ctx, conn, m, m.downPath, target, m.version); err != nil {
			return err
		}
		fmt.Printf("reverted %06d  %s\n", m.version, m.name)
	}

	version, _, err := currentVersion(ctx, conn)
	if err != nil {
		return err
	}
	fmt.Printf("done: version %d\n", version)
	return nil
}

// applyMigration runs one migration file inside a transaction and records
// newVersion on success.
//
// The dirty flag is written before the SQL runs and cleared with the new version
// afterwards. If the SQL fails, its transaction rolls back and the marker is
// restored to restoreVersion with dirty cleared, because nothing was applied. If
// the process dies between the two writes, or the commit outcome is unknown, the
// dirty flag stays set and the next run refuses to continue and names the
// version that needs attention, rather than silently skipping or re-running a
// half-applied migration.
func applyMigration(ctx context.Context, conn *sql.Conn, m migration, file string, newVersion, restoreVersion int64) error {
	body, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", file, err)
	}
	if strings.TrimSpace(string(body)) == "" {
		return fmt.Errorf("%s is empty", file)
	}

	if err := setVersion(ctx, conn, m.version, true); err != nil {
		return fmt.Errorf("failed to mark migration %06d dirty: %w", m.version, err)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		_ = tx.Rollback()
		// Nothing was applied, so put the marker back where it was. Use a context
		// without the run's cancellation: a timeout must not leave the database
		// flagged dirty when the schema is in fact untouched.
		if restoreErr := setVersion(context.WithoutCancel(ctx), conn, restoreVersion, false); restoreErr != nil {
			return fmt.Errorf("migration %06d (%s) failed: %w (and the dirty flag could not be cleared: %v)",
				m.version, filepath.Base(file), err, restoreErr)
		}
		return fmt.Errorf("migration %06d (%s) failed: %w", m.version, filepath.Base(file), err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration %06d (%s) failed to commit: %w", m.version, filepath.Base(file), err)
	}

	if err := setVersion(ctx, conn, newVersion, false); err != nil {
		return fmt.Errorf("migration %06d applied but the version could not be recorded: %w", m.version, err)
	}
	return nil
}
