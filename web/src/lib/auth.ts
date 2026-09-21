// Auth data hooks for the BFF (docs/architecture/api.md): GET /me drives route
// gating and the account menu; POST /onboarding/step persists step 1; POST
// /auth/logout revokes the session. OAuth sign-in is a top-level form POST to
// /auth/{provider}/start (302 to the provider), so it is a plain URL, not a fetch.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { API_BASE, ApiRequestError, apiFetch } from "./api";

/** Onboarding state (the 3-step first-run flow). */
export interface Onboarding {
  path_chosen: string | null;
  budget_set: boolean;
  key_added: boolean;
  completed: boolean;
}

/** The weekend "hours per day" band (Settings segmented control). */
export type WeekendBand = "2" | "3-4" | "5";

/** Study budget (account.study_budget_json). Fields are optional because a fresh
 *  account carries an empty {} until onboarding step 2 / Settings sets it. */
export interface StudyBudget {
  weekday_minutes?: number;
  weekend_band?: WeekendBand;
}

/** Reminder preferences (account.reminders_json). Optional for the same reason. */
export interface Reminders {
  daily_reminder_on?: boolean;
  daily_reminder_time?: string; // "HH:MM" local
  revision_due_alerts_on?: boolean;
}

/** The current account. study_budget / reminders are the learner's own prefs, returned
 *  only on the JWT-gated /me so the Settings form can round-trip them. */
export interface Account {
  id: string;
  display_name: string;
  email?: string;
  timezone: string;
  study_budget: StudyBudget;
  reminders: Reminders;
  created_at: string;
}

/** GET /me payload: account + onboarding. */
export interface Me {
  account: Account;
  onboarding: Onboarding;
}

/** useMe fetches the current session's account + onboarding. A 401 surfaces as an
 *  ApiRequestError with isUnauthenticated — callers redirect to OAuth. */
export function useMe() {
  return useQuery<Me, ApiRequestError>({
    queryKey: ["me"],
    queryFn: () => apiFetch<Me>("/me"),
    retry: false,
    staleTime: 30_000,
  });
}

/** oauthStartAction is the form action that begins OAuth for a provider. The SPA
 *  submits a top-level POST so the browser follows the 302 to the provider. v1
 *  ships GitHub only; Google (ADR-0006) is deferred. */
export function oauthStartAction(provider: "github"): string {
  return `${API_BASE}/auth/${provider}/start`;
}

/** useSetOnboardingPath persists the chosen path (onboarding step 1). */
export function useSetOnboardingPath() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (path: string) =>
      apiFetch<{ onboarding: Onboarding }>("/onboarding/step", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ step: "path", path_chosen: path }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });
}

/** useSetOnboardingBudget persists the study budget (onboarding step 2). */
export function useSetOnboardingBudget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (budget: StudyBudget) =>
      apiFetch<{ onboarding: Onboarding }>("/onboarding/step", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ step: "budget", study_budget: budget }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });
}

/** useCompleteOnboarding marks onboarding finished (step 3 · Finish / Skip). It never
 *  sets key_added — the coach key store lands in S11. */
export function useCompleteOnboarding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch<{ onboarding: Onboarding }>("/onboarding/step", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ step: "finish" }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });
}

/** The editable fields of the current account (PATCH /me). Any subset may be sent. */
export interface MePatch {
  display_name?: string;
  timezone?: string;
  study_budget?: StudyBudget;
  reminders?: Reminders;
}

/** usePatchMe saves profile / study budget / timezone / reminders (Settings). The
 *  response is the full {account, onboarding} so the ["me"] cache updates in place. */
export function usePatchMe() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (patch: MePatch) =>
      apiFetch<Me>("/me", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      }),
    onSuccess: (me) => qc.setQueryData(["me"], me),
  });
}

/** useLogout revokes the server session and clears the cookie. */
export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiFetch<void>("/auth/logout", { method: "POST" }),
    onSuccess: () => qc.clear(),
  });
}
