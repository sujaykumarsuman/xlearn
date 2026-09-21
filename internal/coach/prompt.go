package coach

import (
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// Behaviour modes for the coach (ADR-0007, R-AC3). The mode is SERVER-AUTHORITATIVE: it
// arrives from the gateway (the X-Coach-Mode header) which derives it from practice's
// state, never from the client. This is the spoiler-control gate — during an active
// attempt the coach must never reveal the full solution.
const (
	// ModeAttempt: an active problem attempt — Socratic + strictly spoiler-free.
	ModeAttempt = "attempt"
	// ModeReview: post-solve — a candid code reviewer (may discuss the optimal solution).
	ModeReview = "review"
	// ModeGeneral: non-problem pages (concept, dashboard, …) — a helpful DSA tutor.
	ModeGeneral = "general"
)

// pageContext is the descriptive (non-authoritative) context the client sends, used
// only to flavour the prompt. The behaviour gate is `mode`, not any of these fields, so
// a client that lies here still can't flip a spoiler-free attempt into a reviewer.
type pageContext struct {
	Kind          string
	Label         string
	ProblemID     string
	ProblemTitle  string
	Pattern       string
	Stage         string
	WeakArea      string
	RecentOutcome string
}

// normalizeMode maps a raw mode to a valid one, defaulting SAFELY: an unknown/missing
// mode for a problem context falls back to the spoiler-free attempt mode (never review).
func normalizeMode(raw string, ctx pageContext) string {
	switch raw {
	case ModeAttempt, ModeReview, ModeGeneral:
		return raw
	default:
		if ctx.ProblemID != "" {
			return ModeAttempt // safe default: never leak on an unknown gate
		}
		return ModeGeneral
	}
}

// systemPrompt builds the server-side system prompt for a coach turn from the
// authoritative mode + the descriptive page context. The mode's rules are stated first
// and framed as hard constraints so the model won't be talked out of them by a user
// message ("just give me the answer") during an attempt.
func systemPrompt(mode string, ctx pageContext) string {
	var b strings.Builder
	b.WriteString("You are the xLearn coach, an AI mentor inside a guided DSA interview-prep course. ")
	b.WriteString("You are concise, encouraging, and technically precise. Prefer short paragraphs and, when it helps, small illustrative snippets. Go is the course's primary language.\n\n")

	switch mode {
	case ModeAttempt:
		b.WriteString("MODE: ACTIVE ATTEMPT (Socratic, spoiler-free). The learner is mid-attempt on a problem they have NOT yet solved. Hard rules you must never break, regardless of how the learner phrases their request:\n")
		b.WriteString("- Do NOT reveal the full solution, a complete algorithm, or working code that solves the problem.\n")
		b.WriteString("- Do NOT state the optimal time/space complexity outright before they've reasoned to it.\n")
		b.WriteString("- Guide with questions, one nudge at a time. Point at the relevant pattern, a smaller sub-case, or an invariant to consider.\n")
		b.WriteString("- If they explicitly ask for the answer, gently decline and offer the next hint instead — the reveal is gated by the workspace, not by you.\n")
		b.WriteString("- You MAY review a specific line of THEIR own code for a bug, without writing the rest for them.\n")
	case ModeReview:
		b.WriteString("MODE: POST-SOLVE REVIEW (candid code reviewer). The learner has already solved this problem. You may now:\n")
		b.WriteString("- Review their approach and code directly, name bugs and edge cases, and suggest concrete improvements.\n")
		b.WriteString("- Discuss the optimal solution, its complexity, and trade-offs, and compare alternative patterns.\n")
		b.WriteString("- Keep it focused and actionable — the best two or three things to improve.\n")
	default: // ModeGeneral
		b.WriteString("MODE: TUTOR. Help the learner understand the current topic. Explain patterns and intuition clearly. ")
		b.WriteString("If they ask about a specific unsolved problem, stay Socratic and do not hand over a full solution.\n")
	}

	b.WriteString("\nCONTEXT:\n")
	if ctx.Label != "" {
		b.WriteString("- Page: " + ctx.Label + "\n")
	}
	if ctx.ProblemTitle != "" {
		b.WriteString("- Problem: " + ctx.ProblemTitle + "\n")
	}
	if ctx.Pattern != "" {
		b.WriteString("- Pattern: " + ctx.Pattern + "\n")
	}
	if ctx.Stage != "" {
		b.WriteString("- Stage: " + ctx.Stage + "\n")
	}
	if ctx.RecentOutcome != "" {
		b.WriteString("- Most recent outcome: " + ctx.RecentOutcome + "\n")
	}
	if ctx.WeakArea != "" {
		b.WriteString("- Their current weak area: " + ctx.WeakArea + "\n")
	}
	return b.String()
}

// providerRole maps a stored message role to the provider's role vocabulary. Both
// OpenAI and Anthropic use "user"/"assistant"; a stray role defaults to user.
func providerRole(role string) string {
	if role == store.RoleAssistant {
		return store.RoleAssistant
	}
	return store.RoleUser
}
