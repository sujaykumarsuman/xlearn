package curriculum

import (
	"context"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

// fakeStore is an in-memory store.Store for handler unit tests (no database). Each
// lookup returns store.ErrNotFound when the key is absent so the handlers' 404
// mapping is exercised.
type fakeStore struct {
	paths    []store.Path
	phases   map[string][]store.Phase
	weeks    map[string][]store.WeekSummary
	week     map[wkKey]store.Week
	concepts map[wkKey][]store.ConceptRef
	problems map[wkKey][]store.Problem
	problem  map[string]store.Problem
	sections map[string][]store.Section
	concept  map[string]store.Concept
	pingErr  error
}

type wkKey struct {
	slug string
	n    int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		phases:   map[string][]store.Phase{},
		weeks:    map[string][]store.WeekSummary{},
		week:     map[wkKey]store.Week{},
		concepts: map[wkKey][]store.ConceptRef{},
		problems: map[wkKey][]store.Problem{},
		problem:  map[string]store.Problem{},
		sections: map[string][]store.Section{},
		concept:  map[string]store.Concept{},
	}
}

func (f *fakeStore) ListPaths(context.Context) ([]store.Path, error) { return f.paths, nil }

func (f *fakeStore) GetPath(_ context.Context, slug string) (store.Path, error) {
	for _, p := range f.paths {
		if p.Slug == slug {
			return p, nil
		}
	}
	return store.Path{}, store.ErrNotFound
}

func (f *fakeStore) ListPhases(_ context.Context, slug string) ([]store.Phase, error) {
	return f.phases[slug], nil
}

func (f *fakeStore) ListWeeks(_ context.Context, slug string) ([]store.WeekSummary, error) {
	return f.weeks[slug], nil
}

func (f *fakeStore) GetWeek(_ context.Context, slug string, n int) (store.Week, error) {
	if w, ok := f.week[wkKey{slug, n}]; ok {
		return w, nil
	}
	return store.Week{}, store.ErrNotFound
}

func (f *fakeStore) ListConceptsByWeek(_ context.Context, slug string, n int) ([]store.ConceptRef, error) {
	return f.concepts[wkKey{slug, n}], nil
}

func (f *fakeStore) ListProblemsByWeek(_ context.Context, slug string, n int) ([]store.Problem, error) {
	return f.problems[wkKey{slug, n}], nil
}

func (f *fakeStore) ListProblemsByPath(_ context.Context, slug string) ([]store.Problem, error) {
	var out []store.Problem
	for k, ps := range f.problems {
		if k.slug == slug {
			out = append(out, ps...)
		}
	}
	return out, nil
}

func (f *fakeStore) GetProblem(_ context.Context, id string) (store.Problem, error) {
	if p, ok := f.problem[id]; ok {
		return p, nil
	}
	return store.Problem{}, store.ErrNotFound
}

func (f *fakeStore) GetProblemsByIDs(_ context.Context, ids []string) ([]store.Problem, error) {
	var out []store.Problem
	for _, id := range ids {
		if p, ok := f.problem[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeStore) ListSections(_ context.Context, id string) ([]store.Section, error) {
	return f.sections[id], nil
}

func (f *fakeStore) GetConcept(_ context.Context, slug string) (store.Concept, error) {
	if c, ok := f.concept[slug]; ok {
		return c, nil
	}
	return store.Concept{}, store.ErrNotFound
}

func (f *fakeStore) CountProblems(_ context.Context, slug string) (int, error) {
	return len(f.problem), nil
}

func (f *fakeStore) SeedAll(context.Context, store.SeedContent) error { return nil }

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

var _ store.Store = (*fakeStore)(nil)
