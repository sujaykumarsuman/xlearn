package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// account_created is a v2 envelope (m1-03), account-scoped: no path_slug, and v1's data
// fields and occurred_at unchanged.
func TestAccountCreatedEnvelopeIsV2AccountScoped(t *testing.T) {
	at := time.Date(2026, 10, 12, 9, 30, 0, 123456789, time.FixedZone("IST", 5*3600+1800))
	b, err := marshalAccountCreated("0e8a5d4c-1f2b-4c3d-8e9f-0a1b2c3d4e5f", "6a1b2c3d-4e5f-4a6b-8c7d-000000000001", "github", "Ada", at)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	env, err := events.DecodeEnvelope(b)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Version != events.EnvelopeV2 || env.PathSlug != "" || env.Subject != SubjectAccountCreated ||
		env.EventID != "0e8a5d4c-1f2b-4c3d-8e9f-0a1b2c3d4e5f" || env.AccountID != "6a1b2c3d-4e5f-4a6b-8c7d-000000000001" ||
		env.OccurredAt != "2026-10-12T04:00:00.123456789Z" {
		t.Fatalf("envelope %+v", env)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["path_slug"]; ok {
		t.Fatalf("account_created carries path_slug: %s", b)
	}
	var data map[string]string
	if err := json.Unmarshal(env.Data, &data); err != nil || len(data) != 2 || data["provider"] != "github" || data["display_name"] != "Ada" {
		t.Fatalf("data %s (%v), want v1's provider + display_name", env.Data, err)
	}
}
