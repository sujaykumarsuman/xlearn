// Dashboard "Today" data hook for the BFF (docs/architecture/api.md, GET /dashboard
// `agg`, S09). The gateway fans out to assessment (streak / solved / mock projection
// stats), review (due queue, weak-area, reminders) and curriculum (the week/problem
// taxonomy), then composes the daily plan with reviews ordered before new work (R-SR5).
import { useQuery } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";
import type { Reminder, WeakArea } from "./mistakes";
import type { DueQueue } from "./revision";

/** The quick-stats row (streak, solved, revisions due, mock best). */
export interface DashboardStats {
  streak: { current: number; longest: number };
  solved: { count: number; total: number };
  revisionsDue: number;
  mock: { best: number; last: number; count: number; target: number };
}

/** A due-review card in the daily plan (reviews lead the plan). */
export interface PlanReview {
  kind: "review";
  itemId: string;
  problemId: string;
  title: string;
  dayLabel: string;
  touchLevel: number;
  mockMode: boolean;
}

/** A new-problem card in the daily plan (follows the reviews). */
export interface PlanProblem {
  kind: "problem";
  problemId: string;
  title: string;
  difficulty: "easy" | "med" | "hard";
  pattern: string;
  status: "available" | "attempting" | "solved" | "locked";
}

export type PlanItem = PlanReview | PlanProblem;

/** One problem in the current week (for the week-progress segmented bar). */
export interface WeekProblem {
  problemId: string;
  title: string;
  status: "solved" | "attempting" | "available" | "locked";
}

/** The current-week progress panel. */
export interface DashboardWeek {
  n: number;
  title: string;
  solved: number;
  total: number;
  problems: WeekProblem[];
}

/** GET /dashboard (agg) payload. Sections degrade independently to empty/null. */
export interface DashboardData {
  stats: DashboardStats;
  plan: PlanItem[];
  week: DashboardWeek | null;
  revisions: DueQueue | null;
  weakArea: WeakArea | null;
  reminders: Reminder[];
}

/** useDashboard fetches the composed "Today" aggregation. */
export function useDashboard() {
  return useQuery<DashboardData, ApiRequestError>({
    queryKey: ["dashboard"],
    queryFn: () => apiFetch<DashboardData>("/dashboard"),
  });
}
