package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// The v2 event envelope (sprint m1-02, M1a; t0 §5, ADR-0026 §3, ADR-0034 §3).
//
// v2 adds one field to the events.md envelope: path_slug, the course a course-scoped
// event belongs to. The envelope is APPEND-ONLY and every decoder keeps reading v1 and
// v2 FOREVER. Consumers ship before producers: in v1.6.0 every consumer decodes both
// versions while every producer still emits v1; producers switch to NewEnvelope with
// EnvelopeV2 one tag later (m1-03).
//
// Decode rule (DecodeEnvelope):
//   - version 1, or no version: a v1 envelope. It predates courses, so it is DSA:
//     PathSlug = V1PathSlug ("dsa"), whatever the payload holds.
//   - version >= 2: the known fields are read and unknown fields ignored (a v3 envelope
//     with extra fields still decodes). A course-scoped subject (CourseScoped: every
//     practice.*, review.* and assessment.* subject) MUST carry path_slug; one without
//     it is ErrInvalidEnvelope, which the consumer dead-letters on its first delivery
//     (event_dead_letter row + ERROR log; D34, no alert). Account-scoped subjects
//     (identity.*) carry no path_slug.
//   - malformed JSON, a negative version, or a v2+ envelope without a subject is
//     ErrInvalidEnvelope too: no redelivery can fix it.

// Envelope versions.
const (
	EnvelopeV1 = 1
	EnvelopeV2 = 2
)

// V1PathSlug is the course every v1 envelope belongs to: v1 is DSA-only. It equals
// course.DSASlug (pinned by a test; the platform package doesn't import course).
const V1PathSlug = "dsa"

// ErrInvalidEnvelope marks an envelope no redelivery can fix. The consumer dead-letters
// it on its FIRST delivery (dispatch) instead of spending the ~8 h redelivery budget,
// and ErrClass records it as "decode".
var ErrInvalidEnvelope = errors.New("events: invalid envelope")

// Envelope is the decoded event envelope (v1 or v2). Fields are append-only: never
// remove, rename or repurpose one.
type Envelope struct {
	EventID    string          `json:"event_id"`
	Subject    string          `json:"subject"`
	OccurredAt string          `json:"occurred_at"` // RFC 3339 UTC; consumers parse leniently
	Version    int             `json:"version"`
	AccountID  string          `json:"account_id"`
	PathSlug   string          `json:"path_slug,omitempty"` // v2: the course (course-scoped subjects)
	Data       json.RawMessage `json:"data"`
}

// NewEnvelope marshals an event envelope and checks it against MaxEnvelopeBytes (L20).
// A version-1 envelope never carries path_slug (v1 has no such field, and a v1 reader
// assumes DSA), so pathSlug must be empty for it. A v2+ envelope on a course-scoped
// subject must carry pathSlug. data is marshalled as the envelope's data object.
//
// Producers adopt it in m1-03 (replacing marshalEnvelope in practice, review and
// assessment and identity's marshalAccountCreated); v1.6.0 producers still emit v1.
func NewEnvelope(version int, eventID, subject, accountID, pathSlug string, occurredAt time.Time, data any) ([]byte, error) {
	switch {
	case version < EnvelopeV1:
		return nil, fmt.Errorf("events: envelope version %d < 1", version)
	case version == EnvelopeV1 && pathSlug != "":
		return nil, fmt.Errorf("events: a v1 envelope carries no path_slug (subject %s)", subject)
	case version >= EnvelopeV2 && pathSlug == "" && CourseScoped(subject):
		return nil, fmt.Errorf("events: course-scoped subject %s needs path_slug in a v%d envelope", subject, version)
	case eventID == "" || subject == "":
		return nil, errors.New("events: envelope needs event_id and subject")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("events: marshal %s data: %w", subject, err)
	}
	b, err := json.Marshal(Envelope{
		EventID:    eventID,
		Subject:    subject,
		OccurredAt: occurredAt.UTC().Format(time.RFC3339Nano),
		Version:    version,
		AccountID:  accountID,
		PathSlug:   pathSlug,
		Data:       raw,
	})
	if err != nil {
		return nil, fmt.Errorf("events: marshal %s envelope: %w", subject, err)
	}
	if err := CheckEnvelope(b); err != nil {
		return nil, err
	}
	return b, nil
}

// DecodeEnvelope decodes a v1 or v2 (or later) envelope under the decode rule above.
// Every error wraps ErrInvalidEnvelope.
func DecodeEnvelope(b []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w: %w", ErrInvalidEnvelope, err)
	}
	switch {
	case env.Version < 0:
		return Envelope{}, fmt.Errorf("%w: version %d", ErrInvalidEnvelope, env.Version)
	case env.Version <= EnvelopeV1:
		// v1 (or a v1 producer that omitted version): DSA, never a payload value.
		env.Version = EnvelopeV1
		env.PathSlug = V1PathSlug
	case env.Subject == "":
		return Envelope{}, fmt.Errorf("%w: v%d envelope without subject", ErrInvalidEnvelope, env.Version)
	case env.PathSlug == "" && CourseScoped(env.Subject):
		return Envelope{}, fmt.Errorf("%w: v%d %s without path_slug", ErrInvalidEnvelope, env.Version, env.Subject)
	}
	return env, nil
}
