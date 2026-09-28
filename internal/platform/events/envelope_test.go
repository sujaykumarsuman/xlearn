package events

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

func readEnvelopeFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "envelope", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// fixtureSubjects are the fixture stems, one per subject v1 emits.
var fixtureSubjects = []string{
	"problem_solved", "solution_revealed_early", "attempt_logged",
	"revision_scheduled", "revision_due", "mistake_opened", "mistake_closed",
	"mock_completed", "account_created",
}

func TestV1PathSlugIsDSA(t *testing.T) {
	if V1PathSlug != course.DSASlug {
		t.Fatalf("V1PathSlug = %q, course.DSASlug = %q", V1PathSlug, course.DSASlug)
	}
}

// Every v1 fixture and its v2 twin decode to the same envelope (v1 → path_slug "dsa").
func TestDecodeEnvelopeV1AndV2Twins(t *testing.T) {
	for _, stem := range fixtureSubjects {
		t.Run(stem, func(t *testing.T) {
			v1, err := DecodeEnvelope(readEnvelopeFixture(t, stem+".v1.json"))
			if err != nil {
				t.Fatalf("v1: %v", err)
			}
			v2, err := DecodeEnvelope(readEnvelopeFixture(t, stem+".v2.json"))
			if err != nil {
				t.Fatalf("v2: %v", err)
			}
			if v1.Version != EnvelopeV1 || v2.Version != EnvelopeV2 {
				t.Fatalf("versions %d/%d", v1.Version, v2.Version)
			}
			if v1.PathSlug != "dsa" {
				t.Fatalf("v1 path_slug = %q, want dsa", v1.PathSlug)
			}
			if CourseScoped(v1.Subject) && v2.PathSlug != "dsa" {
				t.Fatalf("v2 path_slug = %q, want dsa", v2.PathSlug)
			}
			// Same event: only version (and an account-scoped v2's absent path) differ.
			v2.Version = v1.Version
			if !CourseScoped(v1.Subject) {
				v2.PathSlug = v1.PathSlug
			}
			if !reflect.DeepEqual(normalize(t, v1), normalize(t, v2)) {
				t.Fatalf("v1 %+v\nv2 %+v", v1, v2)
			}
		})
	}
}

// normalize re-marshals Data so whitespace differences in the fixtures don't matter.
func normalize(t *testing.T, e Envelope) Envelope {
	t.Helper()
	var v any
	if err := json.Unmarshal(e.Data, &v); err != nil {
		t.Fatalf("data: %v", err)
	}
	b, _ := json.Marshal(v)
	e.Data = b
	return e
}

// A course-scoped v2 event without path_slug is ErrInvalidEnvelope (dead-lettered).
func TestDecodeEnvelopeV2WithoutPathIsInvalid(t *testing.T) {
	for _, stem := range fixtureSubjects {
		if stem == "account_created" {
			continue
		}
		t.Run(stem, func(t *testing.T) {
			_, err := DecodeEnvelope(readEnvelopeFixture(t, stem+".v2-nopath.json"))
			if !errors.Is(err, ErrInvalidEnvelope) {
				t.Fatalf("err = %v, want ErrInvalidEnvelope", err)
			}
			if ErrClass(err) != ErrClassDecode {
				t.Fatalf("ErrClass = %s, want decode", ErrClass(err))
			}
		})
	}
}

// An account-scoped v2 event (identity.*) carries no path_slug and decodes.
func TestDecodeEnvelopeAccountScopedV2(t *testing.T) {
	env, err := DecodeEnvelope(readEnvelopeFixture(t, "account_created.v2.json"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.PathSlug != "" || env.Version != 2 {
		t.Fatalf("env = %+v", env)
	}
}

// A future version with extra fields still decodes: unknown fields are ignored.
func TestDecodeEnvelopeUnknownVersionExtraFields(t *testing.T) {
	env, err := DecodeEnvelope(readEnvelopeFixture(t, "problem_solved.v3-extra.json"))
	if err != nil {
		t.Fatalf("decode v3: %v", err)
	}
	if env.Version != 3 || env.PathSlug != "dsa" || env.Subject != "xlearn.practice.problem_solved" {
		t.Fatalf("env = %+v", env)
	}
	var d struct {
		ProblemID string `json:"problem_id"`
	}
	if err := json.Unmarshal(env.Data, &d); err != nil || d.ProblemID != "16" {
		t.Fatalf("data = %s (%v)", env.Data, err)
	}
}

func TestDecodeEnvelopeEdgeCases(t *testing.T) {
	// No version at all: v1, DSA.
	env, err := DecodeEnvelope(readEnvelopeFixture(t, "problem_solved.v1-noversion.json"))
	if err != nil || env.Version != 1 || env.PathSlug != "dsa" {
		t.Fatalf("no version: %+v, %v", env, err)
	}
	// A v1 envelope's payload path_slug is never trusted: v1 is DSA.
	env, err = DecodeEnvelope([]byte(`{"event_id":"e","subject":"xlearn.practice.problem_solved","version":1,"path_slug":"sql","data":{}}`))
	if err != nil || env.PathSlug != "dsa" {
		t.Fatalf("v1 with path: %+v, %v", env, err)
	}
	for name, b := range map[string]string{
		"malformed":        `not json`,
		"negative version": `{"event_id":"e","subject":"xlearn.practice.problem_solved","version":-1}`,
		"v2 no subject":    `{"event_id":"e","version":2,"path_slug":"dsa"}`,
		"wrong type":       `{"event_id":"e","version":"2"}`,
	} {
		if _, err := DecodeEnvelope([]byte(b)); !errors.Is(err, ErrInvalidEnvelope) {
			t.Errorf("%s: err = %v, want ErrInvalidEnvelope", name, err)
		}
	}
	// An unknown (unregistered) v2 subject without path_slug is not a decode error: the
	// consumer's unlisted-subject path handles it.
	if _, err := DecodeEnvelope([]byte(`{"event_id":"e","subject":"xlearn.practice.future","version":2}`)); err != nil {
		t.Fatalf("unknown subject: %v", err)
	}
}

func TestNewEnvelopeRoundTrip(t *testing.T) {
	at := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	data := map[string]any{"problem_id": "16", "outcome": "clean", "first_solve": true}

	b, err := NewEnvelope(EnvelopeV2, "e-1", "xlearn.practice.problem_solved", "acct", "dsa", at, data)
	if err != nil {
		t.Fatalf("v2: %v", err)
	}
	env, err := DecodeEnvelope(b)
	if err != nil || env.Version != 2 || env.PathSlug != "dsa" || env.EventID != "e-1" || env.AccountID != "acct" ||
		env.OccurredAt != "2026-09-21T12:00:00Z" {
		t.Fatalf("round trip: %+v, %v", env, err)
	}

	b, err = NewEnvelope(EnvelopeV1, "e-2", "xlearn.practice.problem_solved", "acct", "", at, data)
	if err != nil {
		t.Fatalf("v1: %v", err)
	}
	if strings.Contains(string(b), "path_slug") {
		t.Fatalf("a v1 envelope carries path_slug: %s", b)
	}
	if env, err := DecodeEnvelope(b); err != nil || env.PathSlug != "dsa" || env.Version != 1 {
		t.Fatalf("v1 round trip: %+v, %v", env, err)
	}

	if _, err := NewEnvelope(EnvelopeV2, "e", "xlearn.identity.account_created", "acct", "", at, map[string]any{}); err != nil {
		t.Fatalf("account-scoped v2 without path: %v", err)
	}
	for name, call := range map[string]func() error{
		"v2 course-scoped without path": func() error {
			_, err := NewEnvelope(EnvelopeV2, "e", "xlearn.review.revision_due", "acct", "", at, data)
			return err
		},
		"v1 with path": func() error {
			_, err := NewEnvelope(EnvelopeV1, "e", "xlearn.review.revision_due", "acct", "dsa", at, data)
			return err
		},
		"version 0": func() error {
			_, err := NewEnvelope(0, "e", "xlearn.review.revision_due", "acct", "", at, data)
			return err
		},
		"no event id": func() error {
			_, err := NewEnvelope(EnvelopeV2, "", "xlearn.review.revision_due", "acct", "dsa", at, data)
			return err
		},
		"over the cap": func() error {
			_, err := NewEnvelope(EnvelopeV2, "e", "xlearn.review.revision_due", "acct", "dsa", at,
				map[string]any{"blob": strings.Repeat("x", MaxEnvelopeBytes)})
			if !errors.Is(err, ErrEnvelopeTooLarge) {
				t.Errorf("over the cap: %v, want ErrEnvelopeTooLarge", err)
			}
			return err
		},
	} {
		if call() == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// The registry flag: every practice, review and assessment subject is course-scoped,
// identity's are account-scoped, and CourseScoped ⊆ Emits for every stream.
func TestCourseScopedRegistry(t *testing.T) {
	for _, s := range Streams() {
		for _, c := range s.CourseScoped {
			if !contains(s.Emits, c) {
				t.Errorf("%s: CourseScoped %s is not in Emits", s.Name, c)
			}
		}
		switch s.Owner {
		case "practice", "review", "assessment":
			if !reflect.DeepEqual(s.CourseScoped, s.Emits) {
				t.Errorf("%s: CourseScoped %v, want every emitted subject %v", s.Name, s.CourseScoped, s.Emits)
			}
		case "identity":
			if len(s.CourseScoped) != 0 {
				t.Errorf("%s: identity subjects are account-scoped, got %v", s.Name, s.CourseScoped)
			}
		}
	}
	if !CourseScoped("xlearn.practice.problem_solved") || CourseScoped("xlearn.identity.account_created") ||
		CourseScoped("xlearn.practice.unknown") {
		t.Fatal("CourseScoped lookup wrong")
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
