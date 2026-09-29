package store

import (
	"context"
	"database/sql"
	"fmt"
)

// MigrateDownTo rolls coach's schema back to version with the same goose provider
// Migrate uses. Test-only (services never migrate down; ADR-0034 §3): m1-03's data
// migration test steps below 00005 to seed v1 rows, then runs 00005 through goose again.
func MigrateDownTo(ctx context.Context, dsn string, version int64) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration db: %w", err)
	}
	defer db.Close()
	provider, err := newProvider(db)
	if err != nil {
		return err
	}
	if _, err := provider.DownTo(ctx, version); err != nil {
		return fmt.Errorf("migrate down to %d: %w", version, err)
	}
	return nil
}
