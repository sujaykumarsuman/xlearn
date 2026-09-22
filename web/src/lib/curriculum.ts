// Curriculum data hooks for the BFF (docs/architecture/api.md, curriculum-read).
// Read-only content: paths (Catalog), a path's phases + weeks (Roadmap), and — for
// later sprints — a week's content, a problem, a concept. All fetched from
// /xlearn/api with cookie credentials via apiFetch.
import { useQuery } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";

/** A learning path (Catalog card + Roadmap header). */
export interface Path {
  slug: string;
  title: string;
  status: "active" | "coming_soon";
  summary: string;
  problem_total: number;
  week_total: number;
}

/** A phase groups a contiguous week range within a path. */
export interface Phase {
  order: number;
  name: string;
  theme: string;
  week_from: number;
  week_to: number;
}

/** A week with the difficulty mix of its seeded problems (Roadmap rail). */
export interface WeekSummary {
  n: number;
  title: string;
  thesis: string;
  easy: number;
  med: number;
  hard: number;
  total: number;
}

/** GET /paths payload (Catalog). */
export interface PathsResponse {
  paths: Path[];
}

/** GET /paths/{slug} payload (Roadmap). */
export interface PathDetail {
  path: Path;
  phases: Phase[];
  weeks: WeekSummary[];
}

/** A week's content header (Week screen). */
export interface Week {
  n: number;
  title: string;
  thesis: string;
}

/** A concept reference in a week's concept grid. */
export interface ConceptRef {
  slug: string;
  title: string;
}

/** A problem row (difficulty drives the UI tokens; pattern → chip). */
export interface Problem {
  id: string;
  path_slug: string;
  week_n: number;
  title: string;
  difficulty: "easy" | "med" | "hard";
  pattern: string;
  leetcode_url: string;
  neetcode_url: string;
  is_reinforcement: boolean;
}

/** The slim path context a week view needs (drives the eyebrow). */
export interface WeekPath {
  slug: string;
  title: string;
  problem_total: number;
  week_total: number;
}

/** The phase a week belongs to (order + name → eyebrow "Phase X <name>"). */
export interface WeekPhase {
  order: number;
  name: string;
  theme: string;
  week_from: number;
  week_to: number;
}

/** One of the five spaced-repetition touches for a problem (Day 1/3/7/21/45). */
export interface Touch {
  level: number;
  dueDate: string | null;
  /** "none" until a touch is attempted; then pass/fail/due/mock. */
  result: "none" | "pass" | "fail" | "due" | "mock";
}

/** A learner's per-problem practice + revision state. */
export interface ProblemState {
  status: "locked" | "available" | "attempting" | "solved";
  lastOutcome: "clean" | "rough" | "assisted" | "miss" | null;
  currentTouch: number;
  touches: Touch[];
}

/** The "Week N progress" rollup. `populated:false` until S05/S06 source it. */
export interface WeekRollup {
  solved: number;
  coreTotal: number;
  byDifficulty: { easy: number; med: number; hard: number };
  populated: boolean;
}

/** The gateway-aggregated per-user state layered onto the week content (ADR-0013). */
export interface UserState {
  week: WeekRollup;
  problems: Record<string, ProblemState>;
}

/** GET /paths/{slug}/weeks/{n} payload (Week, BFF `agg`): curriculum content + the
 *  placeholder per-user five-touch/solve state. */
export interface WeekAggregate {
  week: Week;
  phase?: WeekPhase;
  path: WeekPath;
  concepts: ConceptRef[];
  problems: Problem[];
  userState: UserState;
}

/** One stage-scoped content section of a problem (only UNLOCKED stages are delivered
 *  — the gateway filters by the learner's practice state, R-PF1). */
export interface ProblemSection {
  stage: "attempt" | "hint" | "solution";
  kind: string;
  order: number;
  body_md: string;
  code: string;
}

/** The server-authoritative countdown for the active stage (attempt 15m / hint 10m).
 *  `remainingSeconds` is computed server-side at response time; the HUD re-syncs on
 *  poll rather than trusting a standalone local clock (R-PF3). */
export interface PracticeTimer {
  kind: "attempt" | "hint";
  deadlineAt: string;
  remainingSeconds: number;
  expired: boolean;
}

/** The learner's practice state for one problem (from the Problem BFF agg). */
export interface PracticeState {
  problemId: string;
  status: "locked" | "available" | "attempting" | "solved";
  /** Deepest content stage reached; "" before an attempt starts. */
  stageReached: "" | "attempt" | "hint" | "solution";
  /** The content stages the learner may see (always includes "attempt"). */
  unlockedStages: ("attempt" | "hint" | "solution")[];
  currentTouch: number;
  lastOutcome: "clean" | "rough" | "assisted" | "miss" | null;
  firstSolvedAt: string | null;
  revealedEarly: boolean;
  timer: PracticeTimer | null;
}

/** The curriculum gate on the Problem workspace (review round 2): must be enrolled to
 *  solve; `scheduled` is false when the problem is ahead of the frontier week (attempt
 *  freely, but it won't count toward the course). */
export interface ProblemGate {
  enrolled: boolean;
  scheduled: boolean;
  currentWeek: number; // the learner's frontier week (0 when not enrolled)
  problemWeek: number;
}

/** GET /problems/{id} payload (Problem, BFF `agg`): curriculum content limited to the
 *  unlocked stages + the practice state + active timer + the curriculum gate. */
export interface ProblemAggregate {
  problem: Problem;
  sections: ProblemSection[];
  state: PracticeState;
  gate?: ProblemGate;
}

/** The reveal penalty acknowledgement (R-PF2) returned when the solution is revealed
 *  before the attempt timer elapses. */
export interface RevealPenalty {
  owedAttempt: boolean;
  dueInDays: number;
  message: string;
}

/** POST /problems/{id}/reveal response. */
export interface RevealResponse {
  revealed: "hint" | "solution";
  penalty: RevealPenalty | null;
  state: PracticeState;
}

/** POST /problems/{id}/attempt/start and /outcome response. */
export interface StateResponse {
  state: PracticeState;
}

/** The outcome response when a solve is NOT counted toward the course: either ahead of
 *  the frontier week (scheduledWeek/currentWeek set) or a pure practice-arena run
 *  (`practice:true`). Not recorded, so the workspace shows a note, not a solved state. */
export interface AheadOutcome {
  counted: false;
  scheduledWeek?: number;
  currentWeek?: number;
  practice?: boolean;
}

/** logOutcome's response: a normal state update, or the ahead-of-schedule ack. */
export type OutcomeResponse = StateResponse | AheadOutcome;

/** isAhead narrows an OutcomeResponse to the ahead-of-schedule (not counted) ack. */
export function isAhead(r: OutcomeResponse): r is AheadOutcome {
  return (r as AheadOutcome).counted === false;
}

/** An outcome the learner logs for a solved problem (R-OL1). */
export type Outcome = "clean" | "rough" | "assisted" | "miss";

/** A full concept/pattern reading + code template (Concept screen). */
export interface Concept {
  slug: string;
  path_slug: string;
  title: string;
  body_md: string;
  when_to_use_md: string;
  code_template: string;
}

/** GET /concepts/{slug} payload (Concept). */
export interface ConceptDetail {
  concept: Concept;
}

/** useWeek fetches a week's content + placeholder five-touch/solve state (Week). */
export function useWeek(slug: string, n: number) {
  return useQuery<WeekAggregate, ApiRequestError>({
    queryKey: ["week", slug, n],
    queryFn: () => apiFetch<WeekAggregate>(`/paths/${encodeURIComponent(slug)}/weeks/${n}`),
    enabled: slug !== "" && Number.isFinite(n) && n > 0,
  });
}

/** useConcept fetches a concept's reading + code template (Concept). */
export function useConcept(slug: string) {
  return useQuery<ConceptDetail, ApiRequestError>({
    queryKey: ["concept", slug],
    queryFn: () => apiFetch<ConceptDetail>(`/concepts/${encodeURIComponent(slug)}`),
    enabled: slug !== "",
  });
}

/** patternMatchesConcept links a problem's free-text `pattern` to a concept by
 *  comparing alphanumeric-normalized forms (so "Sliding Window" ↔ "sliding-window",
 *  "Prefix Sum" ↔ "prefix-sums"). Used to surface a concept's practice problems from
 *  the week it was opened from — real problems only, never fabricated links. */
export function patternMatchesConcept(pattern: string, conceptSlugOrTitle: string): boolean {
  const norm = (s: string) => s.toLowerCase().replace(/[^a-z0-9]/g, "").replace(/s$/, "");
  const a = norm(pattern);
  const b = norm(conceptSlugOrTitle);
  if (!a || !b) return false;
  return a.includes(b) || b.includes(a);
}

/** useProblem fetches the Problem workspace aggregate. Course mode: content limited to the
 *  learner's unlocked stages + practice state + timer. Practice mode (`?practice=1`, the
 *  Problems arena): ALL stages for study, a default no-timer state, and no practice state
 *  touched. Cached separately by mode so the two section sets don't clobber each other. */
export function useProblem(id: string, practice = false) {
  return useQuery<ProblemAggregate, ApiRequestError>({
    queryKey: ["problem", id, practice ? "practice" : "course"],
    queryFn: () =>
      apiFetch<ProblemAggregate>(`/problems/${encodeURIComponent(id)}${practice ? "?practice=1" : ""}`),
    enabled: id !== "",
  });
}

/** `?practice=1` marks a write as a Problems-arena run (open to everyone, never counts). */
const practiceQuery = (practice: boolean) => (practice ? "?practice=1" : "");

/** startAttempt starts/resumes the attempt (POST /problems/{id}/attempt/start). */
export function startAttempt(id: string, practice = false): Promise<StateResponse> {
  return apiFetch<StateResponse>(`/problems/${encodeURIComponent(id)}/attempt/start${practiceQuery(practice)}`, {
    method: "POST",
  });
}

/** revealNext unlocks the next content stage (POST /problems/{id}/reveal); the
 *  response carries the penalty ack when the solution is revealed early. */
export function revealNext(id: string, practice = false): Promise<RevealResponse> {
  return apiFetch<RevealResponse>(`/problems/${encodeURIComponent(id)}/reveal${practiceQuery(practice)}`, {
    method: "POST",
  });
}

/** logOutcome logs the outcome (POST /problems/{id}/outcome). Returns counted:false when
 *  the solve doesn't count — ahead of the frontier week, or a practice-arena run. */
export function logOutcome(id: string, outcome: Outcome, practice = false): Promise<OutcomeResponse> {
  return apiFetch<OutcomeResponse>(`/problems/${encodeURIComponent(id)}/outcome${practiceQuery(practice)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ outcome }),
  });
}

/** usePaths fetches every path + status for the Catalog. */
export function usePaths() {
  return useQuery<PathsResponse, ApiRequestError>({
    queryKey: ["paths"],
    queryFn: () => apiFetch<PathsResponse>("/paths"),
  });
}

/** GET /paths/{slug}/problems payload (the Problems arena). */
export interface PathProblemsResponse {
  problems: Problem[];
}

/** usePathProblems fetches the whole problem index for a path — the Problems arena where
 *  you can browse + attempt any problem (review round 2). */
export function usePathProblems(slug: string) {
  return useQuery<PathProblemsResponse, ApiRequestError>({
    queryKey: ["path-problems", slug],
    queryFn: () => apiFetch<PathProblemsResponse>(`/paths/${encodeURIComponent(slug)}/problems`),
    enabled: slug !== "",
  });
}

/** usePath fetches a path's phases, weeks and totals for the Roadmap. */
export function usePath(slug: string) {
  return useQuery<PathDetail, ApiRequestError>({
    queryKey: ["path", slug],
    queryFn: () => apiFetch<PathDetail>(`/paths/${encodeURIComponent(slug)}`),
    enabled: slug !== "",
  });
}
