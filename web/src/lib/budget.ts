// Study-budget constants + presentational helpers shared by the Settings budget panel
// and onboarding step 2 (both persist the same { weekday_minutes, weekend_band } shape).
import type { WeekendBand } from "./auth";

export const WEEKDAY_MIN = 30;
export const WEEKDAY_MAX = 240;
export const WEEKDAY_STEP = 15;
export const DEFAULT_WEEKDAY = 90;
export const DEFAULT_WEEKEND: WeekendBand = "3-4";

export const WEEKEND_BANDS: { value: WeekendBand; label: string }[] = [
  { value: "2", label: "2h" },
  { value: "3-4", label: "3–4h" },
  { value: "5", label: "5h+" },
];

export function weekendLabel(band: WeekendBand): string {
  return WEEKEND_BANDS.find((b) => b.value === band)?.label ?? "3–4h";
}

export function clampWeekday(m: number): number {
  return Math.max(WEEKDAY_MIN, Math.min(WEEKDAY_MAX, m));
}

/** budgetEta is the presentational "weeks to interview-ready" estimate (matches the
 *  artboard's client-side heuristic — it is display-only, never persisted). */
export function budgetEta(weekday: number, weekend: WeekendBand): string {
  if (weekday >= 120 || weekend === "5") return "14 weeks";
  if (weekday <= 60 && weekend === "2") return "22 weeks";
  return "16 weeks";
}
