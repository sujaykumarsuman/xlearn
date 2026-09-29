package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

// The pure rule table of withhold() (m1-06 task 1), one row per state × surface.
func TestWithholdRuleTable(t *testing.T) {
	stages := func(s ...string) map[string]bool {
		m := map[string]bool{stageAttempt: true}
		for _, st := range s {
			m[st] = true
		}
		return m
	}
	var (
		none      = withheld{}
		answer    = withheld{Pattern: true, Concepts: true, SolutionFacts: true}
		v1Blind   = withheld{SolutionFacts: true, HintStage: true, SolutionStage: true}
		v1Hint    = withheld{SolutionFacts: true, SolutionStage: true}
		gatedAtt  = withholdAll
		gatedHint = withheld{SolutionFacts: true, SolutionStage: true}
		gatedSol  = withheld{}
	)
	openAtt := itemState{OpenAttempt: true, Unlocked: stages()}
	openHint := itemState{OpenAttempt: true, Unlocked: stages(stageHint)}
	openSol := itemState{OpenAttempt: true, Unlocked: stages(stageHint, stageSolution)}
	resolveSolved := itemState{OpenAttempt: true, Solved: true, Unlocked: stages()} // a re-attempt of a solved item
	due := itemState{DueTouch: true, Solved: true, Unlocked: stages(stageHint, stageSolution)}
	liveTouch := itemState{LiveTouch: true, Solved: true, Unlocked: stages(stageHint, stageSolution)}
	blind := itemState{Solved: true, Unlocked: stages()}
	solvedHint := itemState{Solved: true, Unlocked: stages(stageHint)}
	solvedAll := itemState{Solved: true, Unlocked: stages(stageHint, stageSolution)}
	fresh := itemState{Unlocked: stages()}
	unknown := failClosedState()

	cases := []struct {
		name string
		s    itemState
		sf   surface
		want withheld
	}{
		{"list/open attempt", openAtt, surfaceList, answer},
		{"list/due touch", due, surfaceList, answer},
		{"list/live touch", liveTouch, surfaceList, answer},
		{"list/solved blind not due", blind, surfaceList, none},
		{"list/never attempted", fresh, surfaceList, none},
		{"list/unknown", unknown, surfaceList, answer},

		{"workspace/due touch", due, surfaceWorkspace, withholdAll},
		{"workspace/live touch", liveTouch, surfaceWorkspace, withholdAll},
		{"workspace/open attempt, attempt stage", openAtt, surfaceWorkspace, gatedAtt},
		{"workspace/open attempt, hint stage", openHint, surfaceWorkspace, gatedHint},
		{"workspace/open attempt, solution stage", openSol, surfaceWorkspace, gatedSol},
		{"workspace/re-attempt of a solved item", resolveSolved, surfaceWorkspace, gatedAtt},
		{"workspace/never solved", fresh, surfaceWorkspace, gatedAtt},
		{"workspace/solved blind (v1 chip)", blind, surfaceWorkspace, v1Blind},
		{"workspace/solved at hint (v1)", solvedHint, surfaceWorkspace, v1Hint},
		{"workspace/solved, all stages (v1)", solvedAll, surfaceWorkspace, none},
		{"workspace/unknown", unknown, surfaceWorkspace, withholdAll},

		{"arena/open attempt", openAtt, surfaceArena, withholdAll},
		{"arena/due touch", due, surfaceArena, withholdAll},
		{"arena/solved blind", blind, surfaceArena, none},
		{"arena/never attempted", fresh, surfaceArena, none},

		{"coach/open attempt", openAtt, surfaceCoach, answer},
		{"coach/due touch", due, surfaceCoach, answer},
		{"coach/never solved", fresh, surfaceCoach, answer},
		{"coach/solved, not live", blind, surfaceCoach, none},

		{"unknown surface fails closed", blind, surface(99), withholdAll},
	}
	for _, c := range cases {
		if got := withhold(c.s, c.sf); got != c.want {
			t.Errorf("%s: withhold = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestLivePredicate(t *testing.T) {
	for _, c := range []struct {
		s    itemState
		want bool
	}{
		{itemState{}, false},
		{itemState{Solved: true}, false},
		{itemState{OpenAttempt: true}, true},
		{itemState{DueTouch: true}, true},
		{itemState{LiveTouch: true}, true},
		{failClosedState(), true},
	} {
		if got := live(c.s); got != c.want {
			t.Errorf("live(%+v) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestPracticeItemState(t *testing.T) {
	solvedAt := "2026-09-21T00:00:00Z"
	for _, c := range []struct {
		name string
		p    practiceItem
		want itemState
	}{
		{"no row", practiceItem{}, itemState{Unlocked: map[string]bool{stageAttempt: true}}},
		{"attempting", practiceItem{Status: "attempting"}, itemState{OpenAttempt: true, Unlocked: map[string]bool{stageAttempt: true}}},
		{"timer only", practiceItem{Status: "available", Timer: json.RawMessage(`{"kind":"attempt"}`)}, itemState{OpenAttempt: true, Unlocked: map[string]bool{stageAttempt: true}}},
		{"null timer", practiceItem{Status: "available", Timer: json.RawMessage(`null`)}, itemState{Unlocked: map[string]bool{stageAttempt: true}}},
		{"solved blind", practiceItem{Status: "solved", FirstSolvedAt: &solvedAt, UnlockedStages: []string{"attempt"}}, itemState{Solved: true, Unlocked: map[string]bool{stageAttempt: true}}},
		{"re-attempt after a solve", practiceItem{Status: "attempting", FirstSolvedAt: &solvedAt, UnlockedStages: []string{"attempt", "hint"}},
			itemState{OpenAttempt: true, Solved: true, Unlocked: map[string]bool{stageAttempt: true, stageHint: true}}},
	} {
		got := c.p.state()
		if got.OpenAttempt != c.want.OpenAttempt || got.Solved != c.want.Solved || got.DueTouch || got.LiveTouch || len(got.Unlocked) != len(c.want.Unlocked) {
			t.Errorf("%s: state = %+v, want %+v", c.name, got, c.want)
		}
		for st := range c.want.Unlocked {
			if !got.Unlocked[st] {
				t.Errorf("%s: stage %s not unlocked", c.name, st)
			}
		}
	}
}

func TestParseDueSet(t *testing.T) {
	due, ok := parseDueSet([]byte(`{"items":[{"problemId":"2","due":true},{"problemId":"3","due":false},{"problemId":"","due":true}],"dueCount":1}`))
	if !ok || len(due) != 1 || !due["2"] {
		t.Fatalf("parseDueSet = %v, %v", due, ok)
	}
	if _, ok := parseDueSet([]byte(`not json`)); ok {
		t.Fatal("parseDueSet accepted malformed JSON")
	}
}

func TestWithholdListArrayStripsEntryAndJoin(t *testing.T) {
	states := map[string]itemState{"1": {OpenAttempt: true}, "3": {Solved: true}}
	arr := json.RawMessage(`[
		{"id":"m-1","problemId":"1","pattern":"P1","problem":{"id":"1","pattern":"P1","concepts":["C1"],"title":"T1"}},
		{"id":"m-3","problemId":"3","pattern":"P3","problem":{"id":"3","pattern":"P3","concepts":["C3"]}},
		{"id":"9","pattern":"P9"},
		{"kind":"review","problemId":"1","title":"T1"}
	]`)
	out := string(withholdListArray(arr, states))
	for _, gone := range []string{`"P1"`, `"C1"`, `"P9"`} { // item 9 was never resolved: fail closed
		if strings.Contains(out, gone) {
			t.Errorf("withheld value %s survived: %s", gone, out)
		}
	}
	for _, kept := range []string{`"P3"`, `"C3"`, `"T1"`} {
		if !strings.Contains(out, kept) {
			t.Errorf("value %s over-withheld: %s", kept, out)
		}
	}
}

func TestStripItemObjectLeavesNonObjects(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `"x"`} {
		if got := string(stripItemObject(json.RawMessage(raw), withholdAll)); got != raw {
			t.Errorf("stripItemObject(%s) = %s", raw, got)
		}
	}
}

func TestWithholdProblemBodyDropsFactsBlock(t *testing.T) {
	content := []byte(`{"problem":{"id":"5","pattern":"P","solution_facts":{"x":1}},"sections":[
		{"stage":"attempt","kind":"summary"},{"stage":"solution","kind":"code"},{"stage":"solution","kind":"solution_facts","solution_facts":{"x":1}}]}`)
	all := map[string]bool{stageAttempt: true, stageHint: true, stageSolution: true}
	// Facts withheld on their own (the solution stage stays): the facts block goes.
	out, err := withholdProblemBody(content, json.RawMessage(`{}`), all, withheld{SolutionFacts: true})
	if err != nil {
		t.Fatal(err)
	}
	if s := string(out); strings.Contains(s, `"solution_facts"`) || !strings.Contains(s, `"kind":"code"`) || !strings.Contains(s, `"pattern":"P"`) {
		t.Errorf("facts not dropped cleanly: %s", s)
	}
}
