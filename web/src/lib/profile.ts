// Public user dashboard data (F009 / ADR-0024): GET /u/{username} — the UNAUTHENTICATED
// LeetCode-style profile at /xlearn/<username>. It returns only non-PII aggregates composed
// in the gateway (per-course completion + pattern mastery, account-wide solved/streak/mock,
// and one activity heatmap merged across courses). Reuses the Progress taxonomy types so the
// public views render identically to the authed screen.
import { useQuery } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";
import type { HeatmapDay, PatternMastery, PhaseCompletion } from "./progress";

/** The public identity of a profile (never any PII). */
export interface PublicProfileUser {
  username: string;
  displayName: string;
  joinedAt: string; // RFC3339
  /** Coarse UTC-offset band (e.g. "UTC+05:30"), "" when unknown — never the IANA zone (F009). */
  region: string;
}

/** Account-wide totals (streak is one shared streak, LeetCode-style). */
export interface PublicProfileTotals {
  solved: number;
  streak: { current: number; longest: number };
}

/** Account-wide mock-interview stats. */
export interface PublicProfileMock {
  count: number;
  best: number;
  average: number;
}

/** Per-course stats: completion + the same completion-by-phase / pattern-mastery views the
 *  authed Progress screen uses. */
export interface PublicProfileCourse {
  slug: string;
  title: string;
  solved: number;
  total: number;
  pct: number;
  phases: PhaseCompletion[];
  patterns: PatternMastery[];
}

/** GET /u/{username} payload. */
export interface PublicProfile {
  user: PublicProfileUser;
  totals: PublicProfileTotals;
  mock: PublicProfileMock;
  heatmap: { days: HeatmapDay[] } | null;
  courses: PublicProfileCourse[];
}

/** usePublicProfile fetches a public dashboard. A 404 (no such user) surfaces as an
 *  ApiRequestError with status 404 — the screen renders a friendly "no such profile". No
 *  retry on a 4xx (a missing/typo'd username won't become valid on retry). */
export function usePublicProfile(username: string) {
  return useQuery<PublicProfile, ApiRequestError>({
    queryKey: ["public-profile", username],
    queryFn: () => apiFetch<PublicProfile>(`/u/${encodeURIComponent(username)}`),
    enabled: username !== "",
    retry: (count, err) => err.status >= 500 && count < 2,
  });
}
