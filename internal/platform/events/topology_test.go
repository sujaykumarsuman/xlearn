package events

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// TestStreamBudget pins the storage budget (ADR-0035 §1, L20): every stream is
// bounded, and Σ MaxBytes stays within 75% of the server's 5 Gi max_file_store.
func TestStreamBudget(t *testing.T) {
	var sum int64
	for _, s := range Streams() {
		if s.MaxBytes <= 0 {
			t.Errorf("%s: MaxBytes = %d, want > 0 (unbounded streams are what N0 removes)", s.Name, s.MaxBytes)
		}
		sum += s.MaxBytes
	}
	if sum > StreamBudgetBytes {
		t.Fatalf("Σ MaxBytes = %d (%.3f GiB) > budget %d (3.75 GiB)", sum, float64(sum)/float64(GiB), StreamBudgetBytes)
	}
	// The ADR's figure: 3.375 GiB. A change here is a budget decision, not a typo.
	if want := 3*GiB + 3*GiB/8; sum != want {
		t.Errorf("Σ MaxBytes = %.3f GiB, ADR-0035 §1 says 3.375 GiB", float64(sum)/float64(GiB))
	}
}

// TestStreamTable pins the per-stream values of the ADR-0035 §1 table.
func TestStreamTable(t *testing.T) {
	want := map[string]struct {
		owner    string
		maxBytes int64
		maxAge   time.Duration
		discard  jetstream.DiscardPolicy
	}{
		StreamPractice:   {"practice", 1 * GiB, 0, jetstream.DiscardNew},
		StreamReview:     {"review", 1*GiB + 512*MiB, 0, jetstream.DiscardNew},
		StreamAssessment: {"assessment", 128 * MiB, 0, jetstream.DiscardNew},
		StreamJudge:      {"judge", 512 * MiB, 14 * 24 * time.Hour, jetstream.DiscardOld},
		StreamIdentity:   {"identity", 128 * MiB, 0, jetstream.DiscardNew},
		StreamCoach:      {"coach", 128 * MiB, 0, jetstream.DiscardNew},
	}
	got := Streams()
	if len(got) != len(want) {
		t.Fatalf("%d streams declared, want %d", len(got), len(want))
	}
	owners := map[string]bool{}
	for _, s := range got {
		w, ok := want[s.Name]
		if !ok {
			t.Errorf("unexpected stream %s", s.Name)
			continue
		}
		if s.Owner != w.owner || s.MaxBytes != w.maxBytes || s.MaxAge != w.maxAge || s.Discard != w.discard {
			t.Errorf("%s = {%s %d %s %s}, want {%s %d %s %s}", s.Name, s.Owner, s.MaxBytes, s.MaxAge, s.Discard,
				w.owner, w.maxBytes, w.maxAge, w.discard)
		}
		if owners[s.Owner] {
			t.Errorf("%s owns two streams; the ACL gives each service one own stream", s.Owner)
		}
		owners[s.Owner] = true
		if want := []string{"xlearn." + s.Owner + ".*"}; !slices.Equal(s.Subjects, want) {
			t.Errorf("%s subjects %v, want %v", s.Name, s.Subjects, want)
		}
		for _, e := range s.Emits {
			if !matchesAny(s.Subjects, e) {
				t.Errorf("%s emits %s, which its subjects %v don't capture", s.Name, e, s.Subjects)
			}
		}
		cfg := s.Config()
		if cfg.Name != s.Name || cfg.MaxBytes != s.MaxBytes || cfg.MaxAge != s.MaxAge || cfg.Discard != s.Discard ||
			cfg.Duplicates != 5*time.Minute || cfg.Storage != jetstream.FileStorage {
			t.Errorf("%s Config() = %+v", s.Name, cfg)
		}
	}
}

// TestDurablesDeclared checks each live durable is unique, points at a declared
// stream whose subjects capture its filter, and belongs to a service that has an ACL
// identity.
func TestDurablesDeclared(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range Durables() {
		key := d.Stream + "/" + d.Name
		if seen[key] {
			t.Errorf("durable %s declared twice", key)
		}
		seen[key] = true
		s, ok := LookupStream(d.Stream)
		if !ok {
			t.Errorf("durable %s: stream not declared", key)
			continue
		}
		if !slices.ContainsFunc(s.Subjects, func(sub string) bool { return filterWithin(d.Filter, sub) }) {
			t.Errorf("durable %s: filter %s is outside the stream's subjects %v", key, d.Filter, s.Subjects)
		}
		if !slices.Contains(ACLServices(), d.Service) {
			t.Errorf("durable %s: service %q owns no stream", key, d.Service)
		}
	}
	if len(seen) != 4 {
		t.Errorf("%d live durables, want the 4 of today's code (review ×2, assessment ×2)", len(seen))
	}
}

// TestSubjectRegistry is the ADR-0035 §1.1 registry: for every durable, every subject
// the stream's owner can emit that the filter captures is Handled or Ignored (never
// silently acked), the two lists are disjoint, and neither names a subject the filter
// can't receive.
func TestSubjectRegistry(t *testing.T) {
	for _, d := range Durables() {
		s, _ := LookupStream(d.Stream)
		for _, e := range s.Emits {
			if !SubjectMatches(d.Filter, e) {
				continue
			}
			h, ig := slices.Contains(d.Handles, e), slices.Contains(d.Ignores, e)
			switch {
			case h && ig:
				t.Errorf("%s/%s: %s is both handled and ignored", d.Stream, d.Name, e)
			case !h && !ig:
				t.Errorf("%s/%s: %s (emitted by %s) is neither handled nor ignored — list it in Ignores",
					d.Stream, d.Name, e, s.Owner)
			}
			if h != Handled(d.Name, e) || ig != Ignored(d.Name, e) {
				t.Errorf("%s/%s: Handled/Ignored lookups disagree with the table for %s", d.Stream, d.Name, e)
			}
		}
		for _, x := range append(slices.Clone(d.Handles), d.Ignores...) {
			if !SubjectMatches(d.Filter, x) || !slices.Contains(s.Emits, x) {
				t.Errorf("%s/%s lists %s, which %s doesn't emit through filter %s", d.Stream, d.Name, x, s.Owner, d.Filter)
			}
		}
	}
	if Handled("review", "xlearn.practice.attempt_logged") || !Ignored("review", "xlearn.practice.attempt_logged") {
		t.Error("review must ignore attempt_logged explicitly (it is assessment's)")
	}
	if Ignored("review", "xlearn.practice.unknown") || Handled("review", "xlearn.practice.unknown") {
		t.Error("an unlisted subject must be neither handled nor ignored")
	}
}

func TestSubjectMatches(t *testing.T) {
	cases := []struct {
		filter, subject string
		want            bool
	}{
		{"xlearn.practice.*", "xlearn.practice.problem_solved", true},
		{"xlearn.practice.*", "xlearn.practice.a.b", false},
		{"xlearn.practice.*", "xlearn.review.revision_due", false},
		{"xlearn.practice.>", "xlearn.practice.a.b", true},
		{"xlearn.practice.>", "xlearn.practice", false},
		{"xlearn.review.revision_due", "xlearn.review.revision_due", true},
		{"xlearn.review.revision_due", "xlearn.review.mistake_opened", false},
		{"xlearn.*.x", "xlearn.a", false},
	}
	for _, c := range cases {
		if got := SubjectMatches(c.filter, c.subject); got != c.want {
			t.Errorf("SubjectMatches(%q, %q) = %v, want %v", c.filter, c.subject, got, c.want)
		}
	}
}

func TestLookups(t *testing.T) {
	if _, ok := LookupStream("XLEARN_NOPE"); ok {
		t.Error("undeclared stream resolved")
	}
	if _, ok := LookupDurable(StreamPractice, "judge"); ok {
		t.Error("a target-shape durable must not be declared before its sprint")
	}
	s := MustStream(StreamPractice)
	s.Subjects[0] = "mutated"
	if MustStream(StreamPractice).Subjects[0] == "mutated" {
		t.Error("lookups must return copies")
	}
	defer func() {
		if recover() == nil {
			t.Error("MustStream on an undeclared stream must panic")
		}
	}()
	MustStream("XLEARN_NOPE")
}

func matchesAny(filters []string, subject string) bool {
	return slices.ContainsFunc(filters, func(f string) bool { return SubjectMatches(f, subject) })
}

// filterWithin reports whether every subject filter f can match is also captured by
// stream subject s (token-wise: a literal or * within s's * / >).
func filterWithin(f, s string) bool {
	ft, st := strings.Split(f, "."), strings.Split(s, ".")
	for i, tok := range st {
		if tok == ">" {
			return len(ft) > i
		}
		if i >= len(ft) {
			return false
		}
		if tok != "*" && tok != ft[i] {
			return false
		}
		if tok != "*" && ft[i] == ">" {
			return false
		}
	}
	return len(ft) == len(st)
}
