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

// migrationTable keeps goose's own version table inside schema coach — the only schema
// the xlearn_coach role is granted on — rather than the default public.
const migrationTable = "coach.goose_db_version"

// Migrate applies all pending migrations on startup inside a Postgres advisory session
// lock (ADR-0005), so parallel replicas don't race. It opens a short-lived database/sql
// handle (goose's API), runs migrations, and closes it; the service then uses a pgxpool
// for queries. The caller refuses to serve on a non-nil error.
func Migrate(ctx context.Context, dsn string, logger *slog.Logger) error {
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

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		logger.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path)
	}
	return nil
}
