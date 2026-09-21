// Mistake-journal, weak-area and dashboard data hooks for the BFF
// (docs/architecture/api.md, Revision & mistakes + Dashboard). The journal and
// weak-area are gateway aggregations (review's bare-id entries + curriculum problem
// metadata); create/edit proxy to review.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiRequestError, apiFetch } from "./api";

/** The eight mistake categories (R-MJ2), in canonical order. `key` is the stored enum
 *  value; `label` is the display text; `badge` is the artboard badge class. */
export const MISTAKE_CATEGORIES = [
  { key: "misread", label: "Misread", badge: "ds-chip ds-chip--sm" },
  { key: "wrong_pattern", label: "Wrong pattern", badge: "ds-chip ds-chip--sm" },
  { key: "right_pattern_wrong_state", label: "Right pattern, wrong state", badge: "ds-badge ds-badge--violet" },
  { key: "off_by_one", label: "Off-by-one / boundary", badge: "ds-badge ds-badge--warn" },
  { key: "language_bug", label: "Language bug", badge: "ds-chip ds-chip--sm" },
  { key: "complexity_misjudged", label: "Complexity misjudged", badge: "ds-chip ds-chip--sm" },
  { key: "communication", label: "Communication", badge: "ds-badge ds-badge--info" },
  { key: "time_management", label: "Time management", badge: "ds-chip ds-chip--sm" },
] as const;

export type MistakeCategory = (typeof MISTAKE_CATEGORIES)[number]["key"];

const CATEGORY_LABEL: Record<string, string> = Object.fromEntries(
  MISTAKE_CATEGORIES.map((c) => [c.key, c.label]),
);
const CATEGORY_BADGE: Record<string, string> = Object.fromEntries(
  MISTAKE_CATEGORIES.map((c) => [c.key, c.badge]),
);

/** categoryLabel renders a stored category enum as its display text ("" → Uncategorised). */
export function categoryLabel(key: string): string {
  return key === "" ? "Uncategorised" : (CATEGORY_LABEL[key] ?? key);
}

/** categoryBadgeClass returns the artboard badge class for a category. */
export function categoryBadgeClass(key: string): string {
  return CATEGORY_BADGE[key] ?? "ds-chip ds-chip--sm";
}

/** The curriculum problem metadata the gateway enriches each entry with (null when
 *  curriculum can't resolve the id — the screen falls back to the bare id). */
export interface MistakeProblem {
  id: string;
  title: string;
  difficulty: "easy" | "med" | "hard";
  pattern: string;
  week_n: number;
}

/** One journal entry (R-MJ1). category "" = uncategorised; revisitDate null when the
 *  entry has no scheduled revision (a below-clean entry with no ladder). */
export interface Mistake {
  id: string;
  problemId: string;
  pattern: string;
  mistake: string;
  rootCause: string;
  insight: string;
  category: string;
  status: "open" | "closed";
  revisitCount: number;
  revisitDate: string | null;
  createdAt: string;
  problem: MistakeProblem | null;
}

/** GET /mistakes payload. openCount/closedCount are over the full journal;
 *  closeThreshold is the "n/2" denominator (R-MJ4). */
export interface MistakesResponse {
  mistakes: Mistake[];
  openCount: number;
  closedCount: number;
  closeThreshold: number;
  categories: string[];
}

/** GET /weak-area payload (R-MJ3). topCategory "" when no weak area this week. */
export interface WeakArea {
  weekOf?: string;
  topCategory: string;
  topCount: number;
  counts: Record<string, number>;
  entries: Mistake[];
}

/** One in-app reminder (Dashboard). */
export interface Reminder {
  id: string;
  kind: string;
  dueAt: string;
}

/** useMistakes fetches the full journal (the screen filters client-side). */
export function useMistakes() {
  return useQuery<MistakesResponse, ApiRequestError>({
    queryKey: ["mistakes"],
    queryFn: () => apiFetch<MistakesResponse>("/mistakes"),
  });
}

/** useWeakArea fetches the current weekly weak-area banner. */
export function useWeakArea() {
  return useQuery<WeakArea, ApiRequestError>({
    queryKey: ["weak-area"],
    queryFn: () => apiFetch<WeakArea>("/weak-area"),
  });
}

/** The editable fields of a journal entry (PATCH /mistakes/{id}). */
export interface MistakePatch {
  category?: string;
  rootCause?: string;
  insight?: string;
  mistake?: string;
  status?: "open" | "closed";
}

/** usePatchMistake edits an entry (root cause / insight / category / status) and
 *  refreshes the journal + weak-area on success. */
export function usePatchMistake() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: MistakePatch }) =>
      apiFetch<Mistake>(`/mistakes/${encodeURIComponent(id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["mistakes"] });
      qc.invalidateQueries({ queryKey: ["weak-area"] });
    },
  });
}
