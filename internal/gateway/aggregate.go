package gateway

import "encoding/json"

// Week aggregation (api.md GET /paths/{slug}/weeks/{n}, marked `agg`).
//
// The gateway stitches read-only curriculum content to the learner's five-touch /
// solve state, because services cannot cross-join (ADR-0005). Practice (S05) and
// review (S06) do not exist yet, so this sprint layers a PLACEHOLDER `userState` with
// a FROZEN shape (ADR-0013): every problem is honestly "available" with five empty
// touches, and the week rollup is `populated:false`. S05/S06 fill the same fields —
// no client change — so these field names and the 5-entry touches array must not
// drift. Nothing here is faked as done: no green dots, no "solved" chips.

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
	Solved       int              `json:"solved"`       // core problems solved (0 until practice exists)
	CoreTotal    int              `json:"coreTotal"`    // non-reinforcement problems in the week
	ByDifficulty weekByDifficulty `json:"byDifficulty"` // mix over the core problems
	Populated    bool             `json:"populated"`    // false while un-sourced (S05/S06 flip it)
}

// userState is the placeholder per-user block layered onto the week content.
type userState struct {
	Week     weekRollup              `json:"week"`
	Problems map[string]problemState `json:"problems"` // keyed by problem id
}

// aggProblem is the slice of a curriculum problem the aggregation reads to build
// the placeholder state (difficulty + reinforcement flag drive the rollup).
type aggProblem struct {
	ID              string `json:"id"`
	Difficulty      string `json:"difficulty"`
	IsReinforcement bool   `json:"is_reinforcement"`
}

// aggregateWeek merges curriculum week content with the placeholder userState. It
// preserves every field curriculum returned (week/phase/path/concepts/problems) and
// adds a single `userState` key, so the curriculum content model can evolve without
// touching the gateway.
func aggregateWeek(content []byte) ([]byte, error) {
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
	us, err := json.Marshal(placeholderUserState(problems))
	if err != nil {
		return nil, err
	}
	obj["userState"] = us
	return json.Marshal(obj)
}

// placeholderUserState builds an honest empty state for the week's problems: every
// problem "available" with five empty touches, the rollup 0-solved and unpopulated.
func placeholderUserState(problems []aggProblem) userState {
	us := userState{
		Week:     weekRollup{Solved: 0, Populated: false},
		Problems: make(map[string]problemState, len(problems)),
	}
	for _, p := range problems {
		us.Problems[p.ID] = problemState{
			Status:       "available",
			LastOutcome:  nil,
			CurrentTouch: 0,
			Touches:      emptyTouches(),
		}
		if p.IsReinforcement {
			continue // reinforcement problems are not part of the week's "core" rollup
		}
		us.Week.CoreTotal++
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
