package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	// Register the pgx stdlib driver under the "pgx" name for the database/sql
	// handle goose needs (the app itself uses pgxpool, not database/sql).
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// migrationsFS embeds the plain-SQL goose migrations (ADR-0005). fs.Sub trims the
// "migrations/" prefix so goose sees the files at the filesystem root.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrationTable keeps goose's own version table inside schema identity — the only
// schema the xlearn_identity role is granted on — rather than the default public.
const migrationTable = "identity.goose_db_version"

// Migrate applies all pending migrations on startup inside a Postgres advisory
// session lock (ADR-0005), so parallel replicas don't race. It opens a short-lived
// database/sql handle (goose's API), runs migrations, and closes it; the service
// then uses a pgxpool for queries. The caller refuses to serve on a non-nil error.
func Migrate(ctx context.Context, dsn string, logger *slog.Logger) error {
	return migrate(ctx, dsn, logger, 0)
}

// MigrateTo applies pending migrations up to and including version, under the same
// advisory lock. Rehearsals only (m1-02's backfill test builds a v1.5.2-shaped schema
// with it, seeds it, then runs Migrate; m1-08's contract rehearsal reuses it) — services
// always call Migrate.
func MigrateTo(ctx context.Context, dsn string, version int64, logger *slog.Logger) error {
	if version <= 0 {
		return fmt.Errorf("migrate to version %d: want a version > 0", version)
	}
	return migrate(ctx, dsn, logger, version)
}

// migrate applies pending migrations — all of them when to is 0, else up to to.
func migrate(ctx context.Context, dsn string, logger *slog.Logger, to int64) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration db: %w", err)
	}
	defer db.Close()

	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("sub migrations fs: %w", err)
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("new session locker: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres, db, sub,
		goose.WithSessionLocker(locker),
		goose.WithTableName(migrationTable),
	)
	if err != nil {
		return fmt.Errorf("new goose provider: %w", err)
	}

	var results []*goose.MigrationResult
	if to > 0 {
		results, err = provider.UpTo(ctx, to)
	} else {
		results, err = provider.Up(ctx)
	}
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		logger.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path)
	}
	return nil
}
