package config

import (
	"os"
	"strconv"
	"strings"
)

// DefaultPGMaxConns pins each DB-owning service's pgxpool (L21, ADR-0035 §4): 4 per
// service, so Σ stays ≤ ~40 of Postgres' 100 connections. pgx's own default is
// max(4, NumCPU) — 4 on today's 4-vCPU node, but it would double on a KVM 8. judge
// uses 8 (m3-05); the gateway has no pool.
const DefaultPGMaxConns int32 = 4

// PGMaxConns returns the pool size: PG_MAX_CONNS when it is a positive integer, else
// def. An invalid value falls back to def rather than failing a pod over a tuning knob.
func PGMaxConns(def int32) int32 {
	v := strings.TrimSpace(os.Getenv("PG_MAX_CONNS"))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil || n <= 0 {
		return def
	}
	return int32(n)
}
