package store

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/jackc/pgx/v5/pgtype"
)

// uuidString renders a pgtype.UUID as the canonical 8-4-4-4-12 string, or "" when
// the value is NULL.
func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}

// parseUUID parses a canonical UUID string into a pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

// mustUUID parses s or panics — used only for ids this process just generated.
func mustUUID(s string) pgtype.UUID {
	u, err := parseUUID(s)
	if err != nil {
		panic("store: generated an invalid uuid: " + err.Error())
	}
	return u
}

// newUUIDv4 returns a random RFC-4122 v4 UUID string. Used for outbox event ids
// (domain-row ids are generated in Postgres via gen_random_uuid()).
func newUUIDv4() string {
	var b [16]byte
	// crypto/rand.Read never returns a short read and only errors on a broken
	// platform RNG; per stdlib guidance the error is ignored (a well-formed random
	// uuid is still produced from the buffer).
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}
