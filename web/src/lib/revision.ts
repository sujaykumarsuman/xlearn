// Revision data hooks for the BFF (docs/architecture/api.md, Revision & mistakes).
// The due queue is a gateway aggregation (review's five-touch queue + curriculum
// problem metadata); the score submit auto-scores a re-solve → advance or reset.
import { useQuery } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";

/** The curriculum problem metadata the gateway enriches each due item with (null when
 *  curriculum can't resolve the id — the screen falls back to the bare id). */
export interface RevisionProblem {
  id: string;
  title: string;
  difficulty: "easy" | "med" | "hard";
  pattern: string;
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

/** GET /revision/due payload (Revision screen, BFF agg). */
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
 *  moved (advance on a pass, reset-to-Day-1 on a fail). */
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
}

/** The five touches (level 1..5 → Day label + short dot label). */
export const TOUCH_DAYS = [1, 3, 7, 21, 45] as const;
export const TOUCH_DOT_LABELS = ["D1", "D3", "D7", "D21", "D45"] as const;

/** dayLabelFor renders a touch level as its human day ("Day 1" … "Day 45"). */
export function dayLabelFor(level: number): string {
  const d = TOUCH_DAYS[level - 1];
  return d ? `Day ${d}` : "";
}

/** useDueRevision fetches the learner's prioritised revision queue (Revision screen). */
export function useDueRevision() {
  return useQuery<DueQueue, ApiRequestError>({
    queryKey: ["revision", "due"],
    queryFn: () => apiFetch<DueQueue>("/revision/due"),
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
