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
