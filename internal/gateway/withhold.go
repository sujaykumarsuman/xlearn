package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// This file is the gateway's single answer-withholding rule (m1-06; ADR-0027 §1, t1 §3.1
// and §10, t0 §7, t4 §6.4/§8, t5 §9). An item is LIVE while the learner has an open
// counted attempt on it or a due (or, from M2a, live) touch; while it is live, the
// answer-bearing fields — the pattern (topic), concepts, solution facts and the hint and
// solution stages — are withheld on every surface that shows the item. One pure function
// (withhold) decides per surface; every item route applies it through the same helpers
// here, and withhold_routes_test.go proves no route skips it.
//
// Withholding is a nudge, not secrecy: the repository is public. It must still be
// consistent across surfaces and fail closed (an unknown state hides rather than leaks).
//
// Caching: the Week and Today aggregates are cached per account (aggCache, 15 s TTL) with
// the withholding already applied. The learner's own attempt-start/outcome and revision
// score writes invalidate the account, so a new open attempt is withheld on the next read;
// a touch that turns due through review's sweep is picked up within the TTL — acceptable
// for a nudge.

// Content stages (practice's unlocked-stage names; curriculum's section `stage`).
const (
	stageAttempt  = "attempt"
	stageHint     = "hint"
	stageSolution = "solution"
)

// sectionKindSolutionFacts is the solution-stage block curriculum serves an item's
// solution_facts in (never top-level, never on lists; t1 §4).
const sectionKindSolutionFacts = "solution_facts"

// itemState is what withholding needs to know about one item for one learner.
type itemState struct {
	// OpenAttempt: a counted attempt is in progress (practice status "attempting").
	OpenAttempt bool
	// DueTouch: a revision touch on the item is due now (review's due queue).
	DueTouch bool
	// LiveTouch is reserved for M2a's touch attempts (m2-01/m2-04); always false today.
	LiveTouch bool
	// Solved: the item has been solved at least once (status solved or firstSolvedAt).
	Solved bool
	// Unlocked are the content stages of the latest attempt (the workspace's stage gate).
	Unlocked map[string]bool
}

// live reports whether an item is live: an open counted attempt, a due touch or a live
// touch. It is the predicate m1-07's coach mode gate builds on.
func live(s itemState) bool { return s.OpenAttempt || s.DueTouch || s.LiveTouch }

// failClosedState is the state an item gets when an upstream that decides liveness could
// not answer: the most restrictive live state, so every surface hides rather than leaks.
func failClosedState() itemState { return itemState{OpenAttempt: true, DueTouch: true} }

// surface is where an item is shown; each has its own withholding rule.
type surface int

const (
	// surfaceList: the problem index, the week view, Today, the revision due queue,
	// mistakes/weak-area enrichment (the curriculum join and review's own pattern), the
	// revision score result, and concept pages' item chips (which come from the week view).
	surfaceList surface = iota + 1
	// surfaceWorkspace: the problem workspace in the course (GET /problems/{id}).
	surfaceWorkspace
	// surfaceArena: the Problems arena study view (GET /problems/{id}?practice=1).
	surfaceArena
	// surfaceCoach: the problem context the gateway composes for coach.
	surfaceCoach
)

// withheld says which answer-bearing fields a surface must not show for an item.
type withheld struct {
	Pattern, Concepts, SolutionFacts, HintStage, SolutionStage bool
}

// withholdAll withholds every answer-bearing field.
var withholdAll = withheld{Pattern: true, Concepts: true, SolutionFacts: true, HintStage: true, SolutionStage: true}

// withhold is the rule table (sprint m1-06, task 1):
//
//	lists     live → pattern, concepts, facts
//	workspace due/live touch → everything; open attempt or never solved → stage-gated
//	          (pattern + concepts from the hint stage, facts + solution from the solution
//	          stage); solved and not live → as v1 (pattern + concepts shown, hint/solution/
//	          facts by unlocked stage — a problem solved blind keeps its chip)
//	arena     live → the attempt stage only, no pattern/concepts/facts; else the full view
//	coach     live or never solved → pattern, concepts, facts stripped
//
// An unknown surface withholds everything (fail closed).
func withhold(s itemState, sf surface) withheld {
	switch sf {
	case surfaceList:
		if live(s) {
			return withheld{Pattern: true, Concepts: true, SolutionFacts: true}
		}
		return withheld{}
	case surfaceWorkspace:
		if s.DueTouch || s.LiveTouch {
			return withholdAll
		}
		hint, solution := s.Unlocked[stageHint], s.Unlocked[stageSolution]
		w := withheld{SolutionFacts: !solution, HintStage: !hint, SolutionStage: !solution}
		if s.OpenAttempt || !s.Solved {
			// t4 §8: in the attempt shell the chip appears only from the hint stage (the
			// solution stage is past it: stages unlock in order).
			fromHint := hint || solution
			w.Pattern, w.Concepts = !fromHint, !fromHint
		}
		return w
	case surfaceArena:
		if live(s) {
			return withholdAll
		}
		return withheld{}
	case surfaceCoach:
		if live(s) || !s.Solved {
			return withheld{Pattern: true, Concepts: true, SolutionFacts: true}
		}
		return withheld{}
	}
	return withholdAll
}

// --- item states per request ---

// practiceItem is the slice of a practice state (GET /state/{id}, or one entry of the
// light GET /state?ids=) that withholding reads.
type practiceItem struct {
	Status         string          `json:"status"`
	UnlockedStages []string        `json:"unlockedStages"`
	FirstSolvedAt  *string         `json:"firstSolvedAt"`
	Timer          json.RawMessage `json:"timer"`
}

// state derives the item's withholding state from practice alone (DueTouch comes from
// review). An absent practice row is a never-attempted item: not live, not solved.
func (p practiceItem) state() itemState {
	s := itemState{
		OpenAttempt: p.Status == "attempting" || (len(p.Timer) > 0 && string(p.Timer) != "null"),
		Solved:      p.Status == "solved" || (p.FirstSolvedAt != nil && *p.FirstSolvedAt != ""),
		Unlocked:    map[string]bool{stageAttempt: true},
	}
	for _, st := range p.UnlockedStages {
		s.Unlocked[st] = true
	}
	return s
}

// parsePracticeItems parses a practice GET /state?ids= body ({"states":{id:{…}}}).
func parsePracticeItems(body []byte) (map[string]practiceItem, bool) {
	var env struct {
		States map[string]practiceItem `json:"states"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false
	}
	if env.States == nil {
		env.States = map[string]practiceItem{}
	}
	return env.States, true
}

// parsePracticeItem parses a practice GET /state/{id} body ({"state":{…}}).
func parsePracticeItem(body []byte) (practiceItem, bool) {
	var env struct {
		State *practiceItem `json:"state"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.State == nil {
		return practiceItem{}, false
	}
	return *env.State, true
}

// parseDueSet reads the problem ids with a touch due NOW from a review GET
// /revisions/due body (raw or enriched: {"items":[{"problemId","due"}…]}).
func parseDueSet(body []byte) (map[string]bool, bool) {
	var env struct {
		Items []struct {
			ProblemID string `json:"problemId"`
			Due       bool   `json:"due"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false
	}
	out := make(map[string]bool, len(env.Items))
	for _, it := range env.Items {
		if it.Due && it.ProblemID != "" {
			out[it.ProblemID] = true
		}
	}
	return out, true
}

// stateInputs are upstream answers a handler already holds, so itemStates does not ask
// again. A "known" source is used as is (ok or not); an unknown one is fetched.
type stateInputs struct {
	practice      map[string]practiceItem
	practiceKnown bool
	practiceOK    bool
	due           map[string]bool
	dueKnown      bool
	dueOK         bool
}

// itemStates resolves the withholding state of ids (items of course slug) for the
// account: one parallel fan-out under aggCallTimeout — practice GET /state?ids= and review
// GET /revisions/due — skipping any source the handler already fetched (in). An upstream
// that isn't configured has nothing to report. If a configured upstream fails, every
// requested item fails closed (failClosedState) and one WARN is logged; complete is then
// false (callers don't cache a fail-closed composition).
func (g *Gateway) itemStates(ctx context.Context, accountID, slug string, ids []string, in stateInputs) (states map[string]itemState, complete bool) {
	ids = dedupeIDs(ids)
	var wg sync.WaitGroup
	if !in.practiceKnown {
		in.practiceKnown = true
		wg.Add(1)
		go func() {
			defer wg.Done()
			in.practice, in.practiceOK = g.fetchPracticeItems(ctx, accountID, ids)
		}()
	}
	if !in.dueKnown {
		in.dueKnown = true
		wg.Add(1)
		go func() {
			defer wg.Done()
			in.due, in.dueOK = g.fetchDueSet(ctx, accountID, slug)
		}()
	}
	wg.Wait()
	return g.combineStates(ids, in)
}

// combineStates builds each id's state from the two sources, failing closed when either
// could not answer.
func (g *Gateway) combineStates(ids []string, in stateInputs) (map[string]itemState, bool) {
	out := make(map[string]itemState, len(ids))
	if !in.practiceOK || !in.dueOK {
		g.log.Warn("withhold: item state unavailable; withholding as live (fail closed)",
			"practice_ok", in.practiceOK, "review_ok", in.dueOK, "items", len(ids))
		for _, id := range ids {
			out[id] = failClosedState()
		}
		return out, false
	}
	for _, id := range ids {
		s := in.practice[id].state()
		s.DueTouch = in.due[id]
		out[id] = s
	}
	return out, true
}

// fetchPracticeItems asks practice for the light states of ids. Not configured, or no
// ids, is an empty answer; a mint, transport, status or parse failure is ok=false.
func (g *Gateway) fetchPracticeItems(ctx context.Context, accountID string, ids []string) (map[string]practiceItem, bool) {
	if g.practice == nil || len(ids) == 0 {
		return map[string]practiceItem{}, true
	}
	token, ok := g.mintForPractice(accountID)
	if !ok {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(ctx, aggCallTimeout)
	defer cancel()
	body, status, err := g.practice.get(ctx, token, "/state?ids="+url.QueryEscape(strings.Join(ids, ",")))
	if err != nil || status != http.StatusOK {
		return nil, false
	}
	return parsePracticeItems(body)
}

// fetchDueSet asks review which of the course's items have a touch due now. Not
// configured is an empty answer; any failure is ok=false.
func (g *Gateway) fetchDueSet(ctx context.Context, accountID, slug string) (map[string]bool, bool) {
	if g.review == nil {
		return map[string]bool{}, true
	}
	token, ok := g.mintQuiet(accountID, g.audReview)
	if !ok {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(ctx, aggCallTimeout)
	defer cancel()
	body, status, err := g.review.get(ctx, token, withPath("/revisions/due", slug))
	if err != nil || status != http.StatusOK {
		return nil, false
	}
	return parseDueSet(body)
}

// stateOf is an item's state, failing closed for an id the resolver wasn't asked about.
func stateOf(states map[string]itemState, id string) itemState {
	if s, ok := states[id]; ok {
		return s
	}
	return failClosedState()
}

// --- applying the rule to composed JSON ---

// stripItemObject removes the withheld answer-bearing keys (pattern, concepts,
// solution_facts) from one item object. Anything that isn't a JSON object is returned
// unchanged.
func stripItemObject(raw json.RawMessage, w withheld) json.RawMessage {
	if !w.Pattern && !w.Concepts && !w.SolutionFacts {
		return raw
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return raw
	}
	changed := false
	del := func(on bool, key string) {
		if _, ok := obj[key]; on && ok {
			delete(obj, key)
			changed = true
		}
	}
	del(w.Pattern, "pattern")
	del(w.Concepts, "concepts")
	del(w.SolutionFacts, "solution_facts")
	if !changed {
		return raw
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return out
}

// elementItemID is the item id of a composed list element: its `problemId` (due items,
// mistake entries, plan cards) or else its `id` (a curriculum problem object).
func elementItemID(el map[string]json.RawMessage) string {
	var id string
	if raw, ok := el["problemId"]; ok && json.Unmarshal(raw, &id) == nil && id != "" {
		return id
	}
	if raw, ok := el["id"]; ok {
		_ = json.Unmarshal(raw, &id)
	}
	return id
}

// withholdListArray applies the list rule to every element of a JSON array: the
// element's own answer-bearing keys (a curriculum problem, a plan card, review's
// mistake_entry.pattern) and those of its nested `problem` object (the curriculum join).
func withholdListArray(arr json.RawMessage, states map[string]itemState) json.RawMessage {
	var els []map[string]json.RawMessage
	if err := json.Unmarshal(arr, &els); err != nil {
		return arr
	}
	for i, el := range els {
		if el == nil {
			continue
		}
		w := withhold(stateOf(states, elementItemID(el)), surfaceList)
		if !w.Pattern && !w.Concepts && !w.SolutionFacts {
			continue
		}
		if w.Pattern {
			delete(el, "pattern")
		}
		if w.Concepts {
			delete(el, "concepts")
		}
		if w.SolutionFacts {
			delete(el, "solution_facts")
		}
		if p, ok := el["problem"]; ok {
			els[i]["problem"] = stripItemObject(p, w)
		}
	}
	out, err := json.Marshal(els)
	if err != nil {
		return arr
	}
	return out
}

// withholdListBody applies the list rule to the array under arrayKey of a JSON object
// body. A body that isn't an object, or lacks the key, is returned unchanged.
func withholdListBody(body []byte, arrayKey string, states map[string]itemState) []byte {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(body, &env); err != nil || env == nil {
		return body
	}
	arr, ok := env[arrayKey]
	if !ok || len(arr) == 0 || string(arr) == "null" {
		return body
	}
	env[arrayKey] = withholdListArray(arr, states)
	out, err := json.Marshal(env)
	if err != nil {
		return body
	}
	return out
}

// listItemIDs collects the item ids of the elements under arrayKey of a JSON object body
// (see elementItemID).
func listItemIDs(body []byte, arrayKey string) []string {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(body, &env); err != nil {
		return nil
	}
	var els []map[string]json.RawMessage
	if err := json.Unmarshal(env[arrayKey], &els); err != nil {
		return nil
	}
	ids := make([]string, 0, len(els))
	for _, el := range els {
		if id := elementItemID(el); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// visibleStages is the stage set a workspace/arena response may deliver: the unlocked
// stages minus the withheld ones (the attempt stage — the statement — is always shown).
func visibleStages(unlocked map[string]bool, w withheld) map[string]bool {
	return map[string]bool{
		stageAttempt:  true,
		stageHint:     unlocked[stageHint] && !w.HintStage,
		stageSolution: unlocked[stageSolution] && !w.SolutionStage,
	}
}

// withholdProblemBody composes a problem (workspace or arena) body: it keeps only the
// sections of the visible stages (aggregateProblem's stage filter), drops the
// solution-facts block when facts are withheld, embeds the state, and strips the problem
// object's withheld keys.
func withholdProblemBody(content []byte, stateRaw json.RawMessage, unlocked map[string]bool, w withheld) ([]byte, error) {
	merged, err := aggregateProblem(content, stateRaw, visibleStages(unlocked, w))
	if err != nil {
		return nil, err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(merged, &obj); err != nil {
		return nil, err
	}
	if p, ok := obj["problem"]; ok {
		obj["problem"] = stripItemObject(p, w)
	}
	if raw, ok := obj["sections"]; ok && w.SolutionFacts {
		var sections []json.RawMessage
		if err := json.Unmarshal(raw, &sections); err != nil {
			return nil, err
		}
		kept := make([]json.RawMessage, 0, len(sections))
		for _, sec := range sections {
			var meta struct {
				Kind string `json:"kind"`
			}
			if json.Unmarshal(sec, &meta) == nil && meta.Kind == sectionKindSolutionFacts {
				continue
			}
			kept = append(kept, sec)
		}
		if obj["sections"], err = json.Marshal(kept); err != nil {
			return nil, err
		}
	}
	return json.Marshal(obj)
}

// --- route policy (task 2) ---

// withholdMode says whether a route applies withhold() or is exempt from it.
type withholdMode int

const (
	// withholdApplied: the route shows item data and composes it through withhold().
	withholdApplied withholdMode = iota + 1
	// withholdExempt: the route carries no per-item answer field (Reason says why).
	withholdExempt
)

// withholdPolicy is every apiRoute's declaration (withhold_routes_test.go checks that
// every route has one and that every exempt reason is reviewed there).
type withholdPolicy struct {
	Mode   withholdMode
	Reason string
}

// withholdApplies marks a route that composes item data through withhold().
var withholdApplies = withholdPolicy{Mode: withholdApplied}

// exempt marks a route that carries no per-item answer field, with the reviewed reason.
func exempt(reason string) withholdPolicy {
	return withholdPolicy{Mode: withholdExempt, Reason: reason}
}
