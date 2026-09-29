package course

import (
	"fmt"
	"sort"
	"sync"

	"github.com/sujaykumarsuman/xlearn/curriculum"
)

// Registry is a read-only set of course manifests keyed by slug: the courses a binary
// knows (ADR-0026 §1: compiled in, never fetched). Production code uses Embedded(); a
// test injects fixture manifests with NewRegistry (internal/course/coursetest), which is
// the only way a fixture course (zz-fixture, …) exists: fixtures are never embedded in an
// image.
type Registry struct {
	bySlug map[string]*Manifest
	slugs  []string
}

// NewRegistry builds a registry over ms (not copied: callers must not modify the map or
// its manifests afterwards).
func NewRegistry(ms map[string]*Manifest) *Registry {
	r := &Registry{bySlug: make(map[string]*Manifest, len(ms))}
	for s, m := range ms {
		r.bySlug[s] = m
		r.slugs = append(r.slugs, s)
	}
	sort.Strings(r.slugs)
	return r
}

// Lookup returns the manifest for slug, whatever its status. A nil registry knows no
// course.
func (r *Registry) Lookup(slug string) (*Manifest, bool) {
	if r == nil {
		return nil, false
	}
	m, ok := r.bySlug[slug]
	return m, ok
}

// Slugs returns every known slug, sorted.
func (r *Registry) Slugs() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.slugs...)
}

var embedded = sync.OnceValues(func() (*Registry, error) {
	ms, err := Load(curriculum.Manifests)
	if err != nil {
		return nil, err
	}
	return NewRegistry(ms), nil
})

// LoadEmbedded loads and validates the manifests compiled into this binary
// (curriculum.Manifests). Services call it at startup so a bad manifest fails the boot,
// not a request; the result is cached.
func LoadEmbedded() (*Registry, error) {
	r, err := embedded()
	if err != nil {
		return nil, fmt.Errorf("course: embedded manifests: %w", err)
	}
	return r, nil
}

// Embedded returns the compiled-in registry. It panics if the embedded manifests are
// invalid, which CI (contentlint, the course tests) rules out before any image is built.
func Embedded() *Registry {
	r, err := LoadEmbedded()
	if err != nil {
		panic(err)
	}
	return r
}

// Lookup resolves slug against the compiled-in manifests (Embedded().Lookup).
func Lookup(slug string) (*Manifest, bool) { return Embedded().Lookup(slug) }
