package canon

import "sort"

// The field classification: every JSON path of course.Item is either in the contract
// (contract.go's view: a change moves contract_hash AND content_hash) or content-only (a
// change moves content_hash alone). Paths use JSON names, "[]" for the elements of a
// slice of structs; a slice of scalars is one leaf. classify_test.go walks course.Item
// by reflection and fails on any path missing here ("classify the new field in
// canon/classify.go"), and mutates every path to prove each side is honest.
//
// Decided in m3-01 (docs/v2/status.md, Decisions log): limits, languages and
// constraints[] are content-only (packlint's advisory rule 8 re-checks them against the
// pack; constraints[] is flagged for the owner to revisit); a choice's mode, a blank
// field's kind and a probe's key_source are in the contract because the pack's keys are
// read through them.

var contractPaths = set(
	"parts[].id",
	"parts[].type",
	"parts[].config.signature.mode",
	"parts[].config.signature.name",
	"parts[].config.signature.params[].type",
	"parts[].config.signature.returns",
	"parts[].config.signature.ops[].name",
	"parts[].config.signature.ops[].params[].type",
	"parts[].config.signature.ops[].returns",
	"parts[].config.harness",
	"parts[].config.checker.name",
	"parts[].config.checker.params.eps",
	"parts[].config.mode",
	"parts[].config.options[].id",
	"parts[].config.fields[].id",
	"parts[].config.fields[].kind",
	"parts[].config.palette",
	"grader[].step",
	"grader[].kind",
	"grader[].inputs",
	"grader[].rubric",
	"revision.probes[].id",
	"revision.probes[].type",
	"revision.probes[].key_source",
	"revision.probes[].config.mode",
	"revision.probes[].config.options[].id",
	"revision.probes[].config.fields[].id",
	"revision.probes[].config.fields[].kind",
	"assets",
)

var contentOnlyPaths = set(
	// identity and metadata
	"id", "course", "week_n", "sort_order", "title", "difficulty", "pattern", "role", "status",
	"provenance.origin", "provenance.inspired_by", "provenance.license", "provenance.attribution",
	"provenance.authored_by",
	"links[].kind", "links[].url",
	"review.statement", "review.hints", "review.editorial",
	"concepts",
	// parts: cadence, grading (only through the self-path gate) and required are policy
	"parts[].cadence", "parts[].grading", "parts[].required",
	// labels
	"parts[].config.signature.params[].name",
	"parts[].config.signature.ops[].params[].name",
	"parts[].config.options[].label",
	"parts[].config.fields[].label",
	// presentation, limits and public examples
	"parts[].config.languages",
	"parts[].config.constraints[].arg",
	"parts[].config.constraints[].len",
	"parts[].config.constraints[].range",
	"parts[].config.samples[].id",
	"parts[].config.samples[].ops",
	"parts[].config.samples[].args",
	"parts[].config.samples[].expected",
	"parts[].config.limits.time_ms",
	"parts[].config.limits.memory_mb",
	"parts[].config.fields[].units",
	"parts[].config.fields[].max_chars",
	"parts[].config.max_chars",
	"parts[].config.max_nodes",
	// scoring policy
	"grader[].required", "grader[].weight",
	// solution-stage facts
	"solution_facts.complexity.time", "solution_facts.complexity.space",
	// probes: placement, timing and wording
	"revision.probes[].band",
	"revision.probes[].grading",
	"revision.probes[].criterion",
	"revision.probes[].timer_s",
	"revision.probes[].prompt_md",
	"revision.probes[].config.options[].label",
	"revision.probes[].config.fields[].label",
	"revision.probes[].config.fields[].units",
	"revision.probes[].config.fields[].max_chars",
	"revision.probes[].config.max_chars",
)

// IsContractPath reports whether a course.Item JSON path (classify.go's syntax) is part
// of the contract, and whether it is classified at all.
func IsContractPath(path string) (contract, known bool) {
	if contractPaths[path] {
		return true, true
	}
	return false, contentOnlyPaths[path]
}

// ContractPaths returns the contract paths, sorted.
func ContractPaths() []string { return sortedKeys(contractPaths) }

// ContentOnlyPaths returns the content-only paths, sorted.
func ContentOnlyPaths() []string { return sortedKeys(contentOnlyPaths) }

func set(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
