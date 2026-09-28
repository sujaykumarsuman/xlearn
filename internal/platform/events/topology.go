package events

import (
	"slices"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// This file is the single source of truth for the JetStream topology (ADR-0035 §1,
// L20): every stream's owner, subjects and limits, and every live durable consumer.
// It is plain data with no I/O. The publisher builds its StreamConfig from it
// (ensureStream), the consumer refuses a (stream, durable, filter) it doesn't declare,
// the NATS authorization block is rendered from it (acl.go), and the tests pin the
// storage budget and the subject registry.
//
// The v2 target shape (ADR-0035 §1) adds the durables below. They are NOT declared
// yet: no code subscribes to them. Each is declared here by the sprint that adds its
// Subscribe call, which re-renders the ACL golden and merges the infra ACL PR before
// its tag (ADR-0035 §2 standing rule):
//
//	stream             consumer (service)                          sprint
//	XLEARN_IDENTITY    practice, review, assessment, coach (erase)  l-01
//	XLEARN_PRACTICE    identity (erase acks)                        l-01
//	XLEARN_REVIEW      identity (erase acks)                        l-01
//	XLEARN_ASSESSMENT  identity (erase acks)                        l-01
//	XLEARN_COACH       identity (erase acks)                        l-01
//	XLEARN_IDENTITY    judge (erase)                                m3-05
//	XLEARN_JUDGE       identity (erase acks)                        m3-05
//	XLEARN_JUDGE       practice (evaluation_completed)              m3-08
//	XLEARN_PRACTICE    judge (analyzer)                             m4-03
//	XLEARN_JUDGE       review (evaluation_analyzed)                 m4-03

// Stream names (events.md). Each service's own constants alias these.
const (
	StreamPractice   = "XLEARN_PRACTICE"
	StreamReview     = "XLEARN_REVIEW"
	StreamAssessment = "XLEARN_ASSESSMENT"
	StreamJudge      = "XLEARN_JUDGE"
	StreamIdentity   = "XLEARN_IDENTITY"
	StreamCoach      = "XLEARN_COACH"
)

// Storage units for the stream limits.
const (
	KiB int64 = 1 << 10
	MiB int64 = 1 << 20
	GiB int64 = 1 << 30
)

// StreamBudgetBytes is the ceiling for Σ MaxBytes over every stream: 75% of the NATS
// server's max_file_store (5 Gi, ../infra infrastructure/messaging/release.yaml), so
// the file store keeps headroom for compaction and the PVC resize trigger (ADR-0035 §1).
const StreamBudgetBytes = 3*GiB + 3*GiB/4

// Stream is one JetStream stream. Owner is its only publisher; Emits lists every
// concrete subject Owner can publish (the subject registry's producer side).
type Stream struct {
	Name     string                  // XLEARN_PRACTICE
	Owner    string                  // practice (the only publisher; ACL + registry key)
	Subjects []string                // captured subjects, e.g. xlearn.practice.*
	Emits    []string                // every concrete subject Owner can publish (registry)
	MaxBytes int64                   // > 0; Σ ≤ StreamBudgetBytes
	MaxAge   time.Duration           // 0 = unlimited
	Discard  jetstream.DiscardPolicy // New | Old
}

// Durable is one live durable pull consumer. Handles ∪ Ignores must cover every
// subject the stream's owner emits that Filter captures (the subject registry's
// consumer side); a subject in neither is logged at ERROR by the handler.
type Durable struct {
	Stream, Name, Filter, Service string
	Handles, Ignores              []string
}

// streams is the v2 stream table (ADR-0035 §1). XLEARN_JUDGE and XLEARN_COACH are
// declared (budgeted and rendered into the ACL) but nothing publishes to them until
// m3-05 and l-01. Σ MaxBytes = 3.375 GiB.
var streams = []Stream{
	{
		Name:     StreamPractice,
		Owner:    "practice",
		Subjects: []string{"xlearn.practice.*"},
		Emits: []string{
			"xlearn.practice.problem_solved",
			"xlearn.practice.attempt_logged",
			"xlearn.practice.solution_revealed_early",
		},
		MaxBytes: 1 * GiB,
		Discard:  jetstream.DiscardNew,
	},
	{
		Name:     StreamReview,
		Owner:    "review",
		Subjects: []string{"xlearn.review.*"},
		Emits: []string{
			"xlearn.review.revision_scheduled",
			"xlearn.review.revision_due",
			"xlearn.review.mistake_opened",
			"xlearn.review.mistake_closed",
		},
		MaxBytes: 1*GiB + 512*MiB,
		Discard:  jetstream.DiscardNew,
	},
	{
		Name:     StreamAssessment,
		Owner:    "assessment",
		Subjects: []string{"xlearn.assessment.*"},
		Emits:    []string{"xlearn.assessment.mock_completed"},
		MaxBytes: 128 * MiB,
		Discard:  jetstream.DiscardNew,
	},
	{
		Name:     StreamJudge,
		Owner:    "judge",
		Subjects: []string{"xlearn.judge.*"},
		Emits:    nil, // declared only: judge's producer lands in m3-05
		MaxBytes: 512 * MiB,
		MaxAge:   14 * 24 * time.Hour,
		Discard:  jetstream.DiscardOld,
	},
	{
		Name:     StreamIdentity,
		Owner:    "identity",
		Subjects: []string{"xlearn.identity.*"},
		Emits:    []string{"xlearn.identity.account_created"},
		MaxBytes: 128 * MiB,
		Discard:  jetstream.DiscardNew,
	},
	{
		Name:     StreamCoach,
		Owner:    "coach",
		Subjects: []string{"xlearn.coach.*"},
		Emits:    nil, // declared only: coach's erase acknowledgement lands in l-01
		MaxBytes: 128 * MiB,
		Discard:  jetstream.DiscardNew,
	},
}

// durables is the table of LIVE durable consumers, matching today's Subscribe calls
// (review ×2, assessment ×2). The target-shape durables are listed in the file comment.
var durables = []Durable{
	{
		Stream: StreamPractice, Name: "review", Filter: "xlearn.practice.*", Service: "review",
		Handles: []string{"xlearn.practice.problem_solved", "xlearn.practice.solution_revealed_early"},
		Ignores: []string{"xlearn.practice.attempt_logged"}, // assessment's; review has no use for it
	},
	{
		Stream: StreamPractice, Name: "assessment", Filter: "xlearn.practice.*", Service: "assessment",
		Handles: []string{
			"xlearn.practice.problem_solved",
			"xlearn.practice.attempt_logged",
			"xlearn.practice.solution_revealed_early",
		},
	},
	{
		Stream: StreamReview, Name: "notifications", Filter: "xlearn.review.revision_due", Service: "review",
		Handles: []string{"xlearn.review.revision_due"},
	},
	{
		Stream: StreamReview, Name: "assessment", Filter: "xlearn.review.*", Service: "assessment",
		Handles: []string{
			"xlearn.review.revision_scheduled",
			"xlearn.review.revision_due",
			"xlearn.review.mistake_opened",
			"xlearn.review.mistake_closed",
		},
	},
}

// Streams returns a copy of the stream table.
func Streams() []Stream {
	out := make([]Stream, len(streams))
	for i, s := range streams {
		out[i] = s.clone()
	}
	return out
}

// Durables returns a copy of the live durable table.
func Durables() []Durable {
	out := make([]Durable, len(durables))
	for i, d := range durables {
		out[i] = d.clone()
	}
	return out
}

// LookupStream returns the declared stream named name.
func LookupStream(name string) (Stream, bool) {
	for _, s := range streams {
		if s.Name == name {
			return s.clone(), true
		}
	}
	return Stream{}, false
}

// MustStream returns the declared stream named name, panicking when it isn't declared.
// For package-level vars derived from the table (a typo fails at init, not in prod).
func MustStream(name string) Stream {
	s, ok := LookupStream(name)
	if !ok {
		panic("events: stream " + name + " is not declared in topology.go")
	}
	return s
}

// LookupDurable returns the live durable named name on stream.
func LookupDurable(stream, name string) (Durable, bool) {
	for _, d := range durables {
		if d.Stream == stream && d.Name == name {
			return d.clone(), true
		}
	}
	return Durable{}, false
}

// Handled reports whether the durable named durable lists subject under Handles. The
// durable is resolved by name and by the filter that captures subject (assessment's
// durable name is the same on two streams).
func Handled(durable, subject string) bool {
	d, ok := durableFor(durable, subject)
	return ok && slices.Contains(d.Handles, subject)
}

// Ignored reports whether the durable named durable lists subject under Ignores: a
// captured subject the handler deliberately acks without acting on. A consumer's
// default branch calls it and logs ERROR when it returns false, so no subject is ever
// acked silently (ADR-0035 §1.1).
func Ignored(durable, subject string) bool {
	d, ok := durableFor(durable, subject)
	return ok && slices.Contains(d.Ignores, subject)
}

func durableFor(name, subject string) (Durable, bool) {
	for _, d := range durables {
		if d.Name == name && SubjectMatches(d.Filter, subject) {
			return d, true
		}
	}
	return Durable{}, false
}

// Config builds the JetStream stream config from the table. The dedupe window stays
// 5 min for every stream (the relay publishes with the event id as Nats-Msg-Id).
func (s Stream) Config() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:       s.Name,
		Subjects:   slices.Clone(s.Subjects),
		Storage:    jetstream.FileStorage,
		Duplicates: dedupeWindow,
		MaxBytes:   s.MaxBytes,
		MaxAge:     s.MaxAge,
		Discard:    s.Discard,
	}
}

func (s Stream) clone() Stream {
	s.Subjects = slices.Clone(s.Subjects)
	s.Emits = slices.Clone(s.Emits)
	return s
}

func (d Durable) clone() Durable {
	d.Handles = slices.Clone(d.Handles)
	d.Ignores = slices.Clone(d.Ignores)
	return d
}

// SubjectMatches reports whether the NATS subject filter (with * and > wildcards)
// matches the concrete subject.
func SubjectMatches(filter, subject string) bool {
	ft := strings.Split(filter, ".")
	st := strings.Split(subject, ".")
	for i, f := range ft {
		if f == ">" {
			return len(st) > i
		}
		if i >= len(st) {
			return false
		}
		if f != "*" && f != st[i] {
			return false
		}
	}
	return len(ft) == len(st)
}
