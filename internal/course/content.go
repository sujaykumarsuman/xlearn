package course

// The per-course content files beside the manifest (sprint m1-09, ADR-0027 §2, t1 §3.2):
// `courses/<slug>/{phases,weeks,concepts}.json`, the item sidecars, and the repo-wide
// `ids.lock.json`. Each JSON shape has a schema in curriculum/_schema/ (schema_test.go keeps
// the two in lockstep). The rows carry no course field: the directory is the course.

// Content file names inside a course directory and at the curriculum root.
const (
	PhasesFile   = "phases.json"
	WeeksFile    = "weeks.json"
	ConceptsFile = "concepts.json"
	ItemFile     = "item.json"
	LockFile     = "ids.lock.json"
)

// PathRow is one row of curriculum/paths.json: a course's Catalog card (the path.* row).
// Every row has a manifest with the same slug, title and status (Load checks it).
type PathRow struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Summary      string `json:"summary"`
	ProblemTotal int    `json:"problem_total"`
	WeekTotal    int    `json:"week_total"`
	SortOrder    int    `json:"sort_order"`
}

// Phase is one row of phases.json: a contiguous week range with a name and theme.
type Phase struct {
	Order    int    `json:"order"`
	Name     string `json:"name"`
	Theme    string `json:"theme"`
	WeekFrom int    `json:"week_from"`
	WeekTo   int    `json:"week_to"`
}

// Week is one row of weeks.json.
type Week struct {
	N      int    `json:"n"`
	Title  string `json:"title"`
	Thesis string `json:"thesis"`
}

// Concept is one row of concepts.json. Its bodies are sidecars: concepts/<slug>.md,
// concepts/<slug>.when.md and concepts/_code/<slug>.<lang>.snip.
type Concept struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	// Weeks are the week numbers the concept is read in.
	Weeks []int `json:"weeks"`
}

// ResolvedConcept is a concept row with its sidecars inlined.
type ResolvedConcept struct {
	Concept
	BodyMD      string
	WhenToUseMD string
	// Templates maps a language (the extension before .snip) to its code template.
	Templates map[string]string
}

// IDsLock is curriculum/ids.lock.json: every item id ever published, with its course
// and status, and every immutable asset `id@v` with its digest. It is append-only: an
// id never disappears or changes course, and an `id@v` never changes bytes (contentlint
// checks both against the previous release tag).
type IDsLock struct {
	Items  map[string]LockEntry `json:"items"`
	Assets map[string]string    `json:"assets"`
}

// LockEntry is one item's lock row.
type LockEntry struct {
	Course string `json:"course"`
	Status string `json:"status"`
}

// Section stages in serving order.
var SectionStages = []string{"attempt", "hint", "solution"}

// Section is one stage-scoped content section of an item, read from a sidecar:
// sections/<stage>/<NN>-<kind>.md (prose; Language "") or _code/<stage>-<NN>.<lang>.snip
// (kind "code"; Language from the extension before .snip). Bodies are byte-exact.
type Section struct {
	Stage    string `json:"stage"`
	Order    int    `json:"order"`
	Kind     string `json:"kind"`
	Language string `json:"language,omitempty"`
	BodyMD   string `json:"body_md,omitempty"`
	Code     string `json:"code,omitempty"`
}

// ResolvedItem is an item with its sidecars inlined: item.json plus every section, in
// canonical order (stage attempt → hint → solution, then order, then language). It is
// what the seed writes and what canon.ContentHash hashes.
type ResolvedItem struct {
	Item     Item      `json:"item"`
	Sections []Section `json:"sections"`
}

// StageRank orders stages for serving and hashing; an unknown stage sorts last.
func StageRank(stage string) int {
	for i, s := range SectionStages {
		if s == stage {
			return i
		}
	}
	return len(SectionStages)
}
