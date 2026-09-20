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

/** usePaths fetches every path + status for the Catalog. */
export function usePaths() {
  return useQuery<PathsResponse, ApiRequestError>({
    queryKey: ["paths"],
    queryFn: () => apiFetch<PathsResponse>("/paths"),
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
