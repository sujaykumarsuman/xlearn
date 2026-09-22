// Path enrollment (F002): starting a path is an explicit, server-backed action.
// POST /paths/{slug}/start enrolls (idempotent); enrollments ride GET /me, so the
// Catalog and Dashboard read started-state from the ["me"] cache without a second call.
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";
import type { Me, PathEnrollment } from "./auth";

/** POST /paths/{slug}/start response. */
export interface StartPathResponse {
  enrollment: PathEnrollment;
}

/** enrollmentFor returns the caller's enrollment on a path, or null if not started. */
export function enrollmentFor(me: Me | undefined, slug: string): PathEnrollment | null {
  return me?.enrollments?.find((e) => e.path_slug === slug) ?? null;
}

/** currentDay is the 1-based day number since the path was started (learner-local
 *  midnights). Day 1 is the start day. Returns null when not started / undated. */
export function currentDay(enrollment: PathEnrollment | null): number | null {
  if (!enrollment) return null;
  const started = new Date(enrollment.started_at);
  if (Number.isNaN(started.getTime())) return null;
  const startMidnight = new Date(started.getFullYear(), started.getMonth(), started.getDate());
  const now = new Date();
  const nowMidnight = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const days = Math.floor((nowMidnight.getTime() - startMidnight.getTime()) / 86_400_000);
  return Math.max(1, days + 1);
}

/** useStartPath enrolls the caller in a path and refreshes ["me"] so the Catalog +
 *  Dashboard flip to the started view. Idempotent server-side. */
export function useStartPath() {
  const qc = useQueryClient();
  return useMutation<StartPathResponse, ApiRequestError, string>({
    mutationFn: (slug: string) =>
      apiFetch<StartPathResponse>(`/paths/${encodeURIComponent(slug)}/start`, { method: "POST" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });
}
