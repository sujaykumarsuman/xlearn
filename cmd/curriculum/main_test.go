package main

import (
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum"
)

// L21: the pool is pinned to 4 connections unless PG_MAX_CONNS overrides it.
func TestPoolConfigPinsMaxConns(t *testing.T) {
	db := curriculum.DBConfig{Host: "localhost", Port: "5432", Database: "xlearndb", User: "u", SSLMode: "disable", SearchPath: "curriculum"}
	t.Setenv("PG_MAX_CONNS", "")
	cfg, err := poolConfig(db)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConns != 4 || cfg.ConnConfig.RuntimeParams["search_path"] != "curriculum" {
		t.Fatalf("MaxConns = %d, search_path = %q; want 4, curriculum", cfg.MaxConns, cfg.ConnConfig.RuntimeParams["search_path"])
	}
	t.Setenv("PG_MAX_CONNS", "6")
	if cfg, _ = poolConfig(db); cfg.MaxConns != 6 {
		t.Fatalf("PG_MAX_CONNS=6: MaxConns = %d", cfg.MaxConns)
	}
}
