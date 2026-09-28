package identity

import (
	"slices"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// identity's stream resolves in topology.go with identity as its owner, and its
// SubjectAccountCreated is in XLEARN_IDENTITY's Emits (ADR-0035 §1.1 registry).
func TestIdentityTopologyAndSubjects(t *testing.T) {
	s := events.MustStream(StreamIdentity)
	if s.Owner != ServiceName {
		t.Fatalf("XLEARN_IDENTITY owner %q, code says %q", s.Owner, ServiceName)
	}
	if !slices.Equal(s.Emits, []string{store.SubjectAccountCreated}) {
		t.Fatalf("XLEARN_IDENTITY Emits %v, identity produces only %s", s.Emits, store.SubjectAccountCreated)
	}
}
