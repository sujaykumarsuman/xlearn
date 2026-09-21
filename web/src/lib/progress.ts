// Progress data hook for the BFF (docs/architecture/api.md, GET /progress `agg`, S09).
// The gateway composes the assessment read-model projections (tiles, revision heatmap,
// per-problem mastery, mock trend, outcome mix) with the curriculum taxonomy (by-phase
// completion, by-pattern mastery). All numbers come from projections; curriculum only
// supplies the grouping (pattern names, phase week-ranges, the real problem total).
import { useQuery } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";
import type { MockTrend } from "./mock";
import type { WeakArea } from "./mistakes";

/** The four stat tiles + the first-solve outcome mix. */
export interface ProgressSummary {
  solved: number;
  total: number;
  streak: { current: number; longest: number };
  retention: { pct: number; resets: number; ladders: number };
  mock: { count: number; average: number; best: number; last: number; delta: number };
  outcomeMix: { total: number; clean: number; rough: number; assisted: number; miss: number };
}

/** One day of revision activity (solves + reviews) for the heatmap. */
export interface HeatmapDay {
  date: string; // YYYY-MM-DD
  solves: number;
  reviews: number;
}

/** One phase's completion (solved / total core problems in its week range). */
export interface PhaseCompletion {
  order: number;
  name: string;
  theme: string;
  weekFrom: number;
  weekTo: number;
  solved: number;
  total: number;
}

/** One pattern's mastery: solved / total core problems and the quality-weighted pct. */
export interface PatternMastery {
  name: string;
  solved: number;
  total: number;
  pct: number;
}

/** GET /progress (agg) payload. Sections are null when their upstream degraded. */
export interface ProgressData {
  summary: ProgressSummary;
  heatmap: { days: HeatmapDay[] } | null;
  trend: MockTrend | null;
  phases: PhaseCompletion[];
  patterns: PatternMastery[];
  weakArea: WeakArea | null;
}

/** useProgress fetches the composed Progress aggregation. */
export function useProgress() {
  return useQuery<ProgressData, ApiRequestError>({
    queryKey: ["progress"],
    queryFn: () => apiFetch<ProgressData>("/progress"),
  });
}
