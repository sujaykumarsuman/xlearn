package gateway

import "encoding/json"

// BFF aggregation (api.md `agg`). The gateway stitches read-only curriculum content
// to the learner's practice/review state, because services cannot cross-join
// (ADR-0005).
//
// Week (GET /paths/{slug}/weeks/{n}): curriculum content + a `userState` block whose
// shape is FROZEN (ADR-0013). S05 fills the solve fields from practice
// (status/lastOutcome, the rollup's solved count, and `populated`); the five-touch
// `touches` stay neutral until review lands (S06) — same fields, filled in place.
// When practice is unavailable the block degrades honestly to the placeholder
// (every problem "available", 0 solved, `populated:false`) rather than guessing.
//
// Problem (GET /problems/{id}): curriculum content with its sections filtered to the
// learner's UNLOCKED stages (R-PF1 — locked hint/solution content is never delivered)
// plus the practice `state` (status, stage, unlocked stages, active timer).

// touchCount is the five spaced-repetition touches — levels 1..5 = Day 1/3/7/21/45.
const touchCount = 5

// touchState is one of the five revision touches for a problem.
type touchState struct {
	Level   int     `json:"level"`   // 1..5 (Day 1/3/7/21/45)
	DueDate *string `json:"dueDate"` // ISO-8601 when scheduled; nil until review runs
	Result  string  `json:"result"`  // "none" until a touch is attempted (then pass/fail)
}

// problemState is a learner's per-problem practice + revision state.
type problemState struct {
	Status       string       `json:"status"`       // locked|available|attempting|solved
	LastOutcome  *string      `json:"lastOutcome"`  // clean|rough|assisted|miss; nil until attempted
	CurrentTouch int          `json:"currentTouch"` // 0..5 revision touches completed
	Touches      []touchState `json:"touches"`      // always 5 entries
}

// weekByDifficulty is the difficulty mix of a week's core problems.
type weekByDifficulty struct {
	Easy int `json:"easy"`
	Med  int `json:"med"`
	Hard int `json:"hard"`
}

// weekRollup is the "Week N progress" meter state.
type weekRollup struct {
	Solved       int              `json:"solved"`       // core problems solved (from practice)
	CoreTotal    int              `json:"coreTotal"`    // non-reinforcement problems in the week
	ByDifficulty weekByDifficulty `json:"byDifficulty"` // mix over the core problems
	Populated    bool             `json:"populated"`    // true once practice sourced the solve state
}

// userState is the per-user block layered onto the week content.
type userState struct {
	Week     weekRollup              `json:"week"`
	Problems map[string]problemState `json:"problems"` // keyed by problem id
}

// aggProblem is the slice of a curriculum problem the aggregation reads to build the
// state (difficulty + reinforcement flag drive the rollup).
type aggProblem struct {
	ID              string `json:"id"`
	Difficulty      string `json:"difficulty"`
	IsReinforcement bool   `json:"is_reinforcement"`
}

// practiceProblemState is the slice of practice state the week rollup needs.
type practiceProblemState struct {
	Status       string
	LastOutcome  *string
	CurrentTouch int
}

// aggregateWeek merges curriculum week content with the userState block. states holds
// the learner's practice states keyed by problem id (a subset — untouched problems
// are absent); populated reports whether practice actually sourced them (false ⇒ the
// honest placeholder). Every field curriculum returned is preserved; a single
// `userState` key is added.
func aggregateWeek(content []byte, states map[string]practiceProblemState, populated bool) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(content, &obj); err != nil {
		return nil, err
	}
	var problems []aggProblem
	if raw, ok := obj["problems"]; ok {
		if err := json.Unmarshal(raw, &problems); err != nil {
			return nil, err
		}
	}
	us, err := json.Marshal(buildUserState(problems, states, populated))
	if err != nil {
		return nil, err
	}
	obj["userState"] = us
	return json.Marshal(obj)
}

// buildUserState layers the practice states onto the week's problems. A problem with
// no practice row is honestly "available"; the rollup counts solved core problems.
func buildUserState(problems []aggProblem, states map[string]practiceProblemState, populated bool) userState {
	us := userState{
		Week:     weekRollup{Populated: populated},
		Problems: make(map[string]problemState, len(problems)),
	}
	for _, p := range problems {
		status := "available"
		var lastOutcome *string
		currentTouch := 0
		if ps, ok := states[p.ID]; ok {
			if ps.Status != "" {
				status = ps.Status
			}
			lastOutcome = ps.LastOutcome
			currentTouch = ps.CurrentTouch
		}
		us.Problems[p.ID] = problemState{
			Status:       status,
			LastOutcome:  lastOutcome,
			CurrentTouch: currentTouch,
			Touches:      emptyTouches(), // review (S06) fills the five-touch schedule
		}
		if p.IsReinforcement {
			continue // reinforcement problems are not part of the week's "core" rollup
		}
		us.Week.CoreTotal++
		if status == "solved" {
			us.Week.Solved++
		}
		switch p.Difficulty {
		case "easy":
			us.Week.ByDifficulty.Easy++
		case "med":
			us.Week.ByDifficulty.Med++
		case "hard":
			us.Week.ByDifficulty.Hard++
		}
	}
	return us
}

// emptyTouches returns the five neutral touches (levels 1..5, no due date, no result).
func emptyTouches() []touchState {
	t := make([]touchState, touchCount)
	for i := range t {
		t[i] = touchState{Level: i + 1, DueDate: nil, Result: "none"}
	}
	return t
}

// aggregateProblem merges curriculum problem content with the practice state: it
// filters `sections` to the unlocked stages (R-PF1) and adds a `state` key. stateRaw
// is the practice state JSON object to embed; unlocked is the set of stages whose
// sections may be delivered.
func aggregateProblem(content []byte, stateRaw json.RawMessage, unlocked map[string]bool) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(content, &obj); err != nil {
		return nil, err
	}
	if raw, ok := obj["sections"]; ok {
		var sections []json.RawMessage
		if err := json.Unmarshal(raw, &sections); err != nil {
			return nil, err
		}
		kept := make([]json.RawMessage, 0, len(sections))
		for _, sec := range sections {
			var meta struct {
				Stage string `json:"stage"`
			}
			if err := json.Unmarshal(sec, &meta); err != nil {
				return nil, err
			}
			if unlocked[meta.Stage] {
				kept = append(kept, sec)
			}
		}
		filtered, err := json.Marshal(kept)
		if err != nil {
			return nil, err
		}
		obj["sections"] = filtered
	}
	obj["state"] = stateRaw
	return json.Marshal(obj)
}
