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

/** The current account. */
export interface Account {
  id: string;
  display_name: string;
  email?: string;
  timezone: string;
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

/** useLogout revokes the server session and clears the cookie. */
export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiFetch<void>("/auth/logout", { method: "POST" }),
    onSuccess: () => qc.clear(),
  });
}
