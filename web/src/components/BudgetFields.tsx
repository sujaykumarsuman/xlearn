// Shared study-budget picker (weekday minutes stepper + weekend hours band). Used by
// the Settings "Study budget" panel and onboarding step 2 so both stay visually in sync
// with the artboards. Constants + helpers live in ../lib/budget.
import { Icon } from "./Icon";
import type { WeekendBand } from "../lib/auth";
import { WEEKDAY_MAX, WEEKDAY_MIN, WEEKDAY_STEP, WEEKEND_BANDS, clampWeekday } from "../lib/budget";

export function BudgetFields({
  weekday,
  weekend,
  onWeekday,
  onWeekend,
}: {
  weekday: number;
  weekend: WeekendBand;
  onWeekday: (m: number) => void;
  onWeekend: (b: WeekendBand) => void;
}) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <div>
        <div className="ds-field__label" style={{ marginBottom: 8 }}>
          Weekday · minutes/day
        </div>
        <div style={{ display: "flex", alignItems: "center", width: 160 }}>
          <button
            type="button"
            className="ds-btn ds-btn--secondary"
            aria-label="Fewer weekday minutes"
            disabled={weekday <= WEEKDAY_MIN}
            onClick={() => onWeekday(clampWeekday(weekday - WEEKDAY_STEP))}
            style={{ borderRadius: "9px 0 0 9px", width: 40, padding: 0 }}
          >
            <Icon name="minus" className="xl-ico--sm" />
          </button>
          <div
            className="ds-input"
            aria-live="polite"
            style={{
              borderRadius: 0,
              borderLeft: "none",
              borderRight: "none",
              textAlign: "center",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontFamily: "var(--ds-font-mono)",
              fontWeight: 600,
            }}
          >
            {weekday}
          </div>
          <button
            type="button"
            className="ds-btn ds-btn--secondary"
            aria-label="More weekday minutes"
            disabled={weekday >= WEEKDAY_MAX}
            onClick={() => onWeekday(clampWeekday(weekday + WEEKDAY_STEP))}
            style={{ borderRadius: "0 9px 9px 0", width: 40, padding: 0 }}
          >
            <Icon name="plus" className="xl-ico--sm" />
          </button>
        </div>
      </div>

      <div>
        <div className="ds-field__label" style={{ marginBottom: 8 }}>
          Weekend · hours/day
        </div>
        <div className="ds-seg" role="group" aria-label="Weekend hours per day">
          {WEEKEND_BANDS.map((b) => (
            <button
              key={b.value}
              type="button"
              className={weekend === b.value ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"}
              aria-pressed={weekend === b.value}
              onClick={() => onWeekend(b.value)}
            >
              {b.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
