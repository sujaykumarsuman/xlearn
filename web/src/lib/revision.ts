// Revision data hooks for the BFF (docs/architecture/api.md, Revision & mistakes).
// The due queue is a course-scoped gateway aggregation (review's five-touch queue +
// curriculum problem metadata, GET /paths/{slug}/revision/due since m1-03); the score
// submit (by the item's global id) auto-scores a re-solve → advance or reset.
import { useQuery } from "@tanstack/react-query";
import { ApiRequestError, apiFetch, coursePathApi } from "./api";
import type { CourseViewBand } from "./curriculum";

/** The curriculum problem metadata the gateway enriches each due item with (null when
 *  curriculum can't resolve the id — the screen falls back to the bare id). `pattern` is
 *  absent whenever the item is live — a due touch (so every due row), an open attempt,
 *  or never solved: the gateway withholds it (m1-06 `withhold()`), and the screen renders
 *  the pattern chip only when the payload carries it. */
export interface RevisionProblem {
  id: string;
  title: string;
  difficulty: "easy" | "med" | "hard";
  pattern?: string;
  week_n: number;
}

/** One entry in the prioritised revision queue. `touchLevel` 1..5 = Day 1/3/7/21/45;
 *  `due` is true for an overdue review (vs an upcoming "coming up" touch); `mockMode`
 *  marks the Day 21 / Day 45 stricter touches. */
export interface DueItem {
  itemId: string;
  problemId: string;
  touchLevel: number;
  dayLabel: string;
  dueDate: string;
  due: boolean;
  mockMode: boolean;
  status: "pending" | "passed" | "failed";
  problem: RevisionProblem | null;
}

/** GET /paths/{slug}/revision/due payload (Revision screen, BFF agg). */
export interface DueQueue {
  items: DueItem[];
  dueCount: number;
}

/** The three auto-score inputs a re-solve submits (R-SR2). */
export interface ScoreInput {
  namedPatternSecs: number;
  solvedInTimer: boolean;
  statedComplexity: boolean;
}

/** POST /revision/{itemId}/score response: the auto-score result + where the schedule
 *  moved (advance on a pass, reset-to-Day-1 on a fail). `problem` is the same curriculum
 *  problem the due items carry, composed after the touch concluded — so it normally
 *  carries `pattern` (the AB03 reveal); it is absent if the item is still live some other
 *  way, and a pre-m1-06 gateway omits `problem` altogether. */
export interface ScoreResult {
  itemId: string;
  problemId: string;
  touchLevel: number;
  autoPass: boolean;
  status: "passed" | "failed";
  mockMode: boolean;
  reset: boolean;
  nextTouchLevel: number; // 0 = ladder complete (Day 45 passed)
  nextDayLabel: string;
  nextDueDate: string | null;
  problem?: RevisionProblem | null;
}

/** The five touches (level 1..5 → Day label + short dot label). */
export const TOUCH_DAYS = [1, 3, 7, 21, 45] as const;
export const TOUCH_DOT_LABELS = ["D1", "D3", "D7", "D21", "D45"] as const;

/** dayLabelFor renders a touch level as its human day ("Day 1" … "Day 45"). */
export function dayLabelFor(level: number): string {
  const d = TOUCH_DAYS[level - 1];
  return d ? `Day ${d}` : "";
}

/** bandFor returns the course's revision band covering a ladder level (AB03 format
 *  badges), or undefined when the course view carries no band for it. */
export function bandFor(bands: readonly CourseViewBand[] | undefined, level: number): CourseViewBand | undefined {
  return bands?.find((b) => b.levels.includes(level));
}

/** bandTimerSecs is a timed band's re-solve timer in seconds, or undefined for an
 *  untimed band (or no band). */
export function bandTimerSecs(band: CourseViewBand | undefined): number | undefined {
  return band?.timer_s && band.timer_s > 0 ? band.timer_s : undefined;
}

/** bandDuration renders a band's duration for its badge: a timed band's timer as mm:ss
 *  ("20:00"), an untimed band's estimate as "~N min", or "" when it has neither. Nothing
 *  here is a number in code — both come from the manifest band. */
export function bandDuration(band: CourseViewBand): string {
  const t = bandTimerSecs(band);
  if (t !== undefined) return formatMMSS(t);
  if (band.est_minutes && band.est_minutes > 0) return `~${band.est_minutes} min`;
  return "";
}

/** formatMMSS renders seconds as m:ss ("20:00", "1:05"). */
export function formatMMSS(secs: number): string {
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

/** nextReviewDate is the queue's next upcoming touch: the earliest due date among its
 *  not-due items, or null when the payload carries none (AB03-F6). */
export function nextReviewDate(items: readonly DueItem[]): Date | null {
  let next: Date | null = null;
  for (const it of items) {
    if (it.due) continue;
    const d = new Date(it.dueDate);
    if (Number.isNaN(d.getTime())) continue;
    if (!next || d < next) next = d;
  }
  return next;
}

/** formatNextReview renders the next review date learner-local, like AB03-F6's
 *  "Thu, Oct 8". */
export function formatNextReview(d: Date): string {
  return d.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" });
}

/** useDueRevision fetches the learner's prioritised revision queue for a course (the
 *  Revision screen, and the sidebar's live badge). */
export function useDueRevision(slug: string, enabled = true) {
  return useQuery<DueQueue, ApiRequestError>({
    queryKey: ["revision", "due", slug],
    queryFn: () => apiFetch<DueQueue>(coursePathApi(slug, "/revision/due")),
    enabled: enabled && slug !== "",
  });
}

/** scoreRevision submits a re-solve for auto-scoring (POST /revision/{itemId}/score). */
export function scoreRevision(itemId: string, input: ScoreInput): Promise<ScoreResult> {
  return apiFetch<ScoreResult>(`/revision/${encodeURIComponent(itemId)}/score`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}
