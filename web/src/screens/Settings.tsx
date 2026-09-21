import { useState } from "react";
import { Icon } from "../components/Icon";
import { BudgetFields } from "../components/BudgetFields";
import { DEFAULT_WEEKDAY, DEFAULT_WEEKEND, budgetEta, clampWeekday, weekendLabel } from "../lib/budget";
import { useMe, usePatchMe, type Account, type WeekendBand } from "../lib/auth";
import { useCoachKey, type CoachKey } from "../lib/settings";

/**
 * Settings is the D6 account surface (Settings.dc.html): Profile, Study budget and
 * Reminders are live (persisted via PATCH /me); the API-keys section is the S10 shell
 * (masked GET /coach/key read + empty state) whose store/delete land with the coach in
 * S11. Every control reuses the design-system classes (ds-field/ds-input/ds-seg/
 * ds-toggle/ds-badge) verbatim.
 */
export default function Settings() {
  const me = useMe();

  return (
    <div className="xl-content">
      <div className="xl-page-h">
        <div>
          <div className="xl-eyebrow">Account</div>
          <h1 style={{ marginTop: 6 }}>Settings</h1>
        </div>
      </div>

      {me.isLoading && <Centered>Loading your settings…</Centered>}
      {me.isError && (
        <div className="xl-panel" style={{ maxWidth: 800 }}>
          <div className="xl-panel__b" style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <Icon name="alert" style={{ color: "var(--ds-warn)" }} />
            <span style={{ flex: 1 }}>Couldn’t load your settings.</span>
            <button className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => me.refetch()}>
              Retry
            </button>
          </div>
        </div>
      )}

      {me.data && (
        <div style={{ maxWidth: 800 }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 22 }}>
            <ProfileSection account={me.data.account} />
            <BudgetSection account={me.data.account} />
            <ApiKeysSection />
            <RemindersSection account={me.data.account} />
          </div>
        </div>
      )}
    </div>
  );
}

// --- Profile ---

const TIMEZONES = [
  "UTC",
  "Asia/Kolkata",
  "Asia/Singapore",
  "Asia/Tokyo",
  "Asia/Dubai",
  "Europe/London",
  "Europe/Berlin",
  "America/New_York",
  "America/Chicago",
  "America/Los_Angeles",
  "Australia/Sydney",
];

function ProfileSection({ account }: { account: Account }) {
  const save = usePatchMe();
  const [name, setName] = useState(account.display_name);
  const [tz, setTz] = useState(account.timezone || "UTC");
  const zones = TIMEZONES.includes(tz) ? TIMEZONES : [tz, ...TIMEZONES];
  const nameValid = name.trim().length > 0;

  const onEdit = <T,>(setter: (v: T) => void) => (v: T) => {
    if (save.isSuccess || save.isError) save.reset();
    setter(v);
  };

  return (
    <Panel title="Profile">
      <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
          <span className="ds-avatar ds-avatar--lg">{initial(account)}</span>
          <div>
            <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" disabled title="Avatar upload is coming soon">
              Change avatar
            </button>
            <div style={{ fontSize: 11, color: "var(--ds-muted)", marginTop: 6 }}>PNG or JPG, up to 2 MB</div>
          </div>
        </div>

        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
          <div className="ds-field">
            <label className="ds-field__label" htmlFor="set-name">
              Full name
            </label>
            <input
              id="set-name"
              className={nameValid ? "ds-input" : "ds-input ds-input--invalid"}
              value={name}
              onChange={(e) => onEdit(setName)(e.target.value)}
            />
          </div>
          <div className="ds-field">
            <label className="ds-field__label" htmlFor="set-email">
              Email
            </label>
            <input id="set-email" className="ds-input" value={account.email ?? ""} readOnly style={{ color: "var(--ds-dim)" }} />
          </div>
        </div>

        <div className="ds-field" style={{ maxWidth: 340 }}>
          <label className="ds-field__label" htmlFor="set-tz">
            Timezone
          </label>
          <div style={{ position: "relative" }}>
            <select
              id="set-tz"
              className="ds-input"
              value={tz}
              onChange={(e) => onEdit(setTz)(e.target.value)}
              style={{ appearance: "none", paddingRight: 32 }}
            >
              {zones.map((z) => (
                <option key={z} value={z}>
                  {z}
                </option>
              ))}
            </select>
            <Icon name="chevdown" className="xl-ico--sm" style={{ position: "absolute", right: 11, top: 11, color: "var(--ds-muted)", pointerEvents: "none" }} />
          </div>
        </div>

        <SaveRow
          save={save}
          disabled={!nameValid}
          onSave={() => save.mutate({ display_name: name.trim(), timezone: tz })}
        />
      </div>
    </Panel>
  );
}

function initial(account: Account): string {
  const s = (account.display_name || account.email || "?").trim();
  return s ? s[0]!.toUpperCase() : "?";
}

// --- Study budget ---

function BudgetSection({ account }: { account: Account }) {
  const save = usePatchMe();
  const [weekday, setWeekday] = useState(clampWeekday(account.study_budget.weekday_minutes ?? DEFAULT_WEEKDAY));
  const [weekend, setWeekend] = useState<WeekendBand>(account.study_budget.weekend_band ?? DEFAULT_WEEKEND);

  const edit = <T,>(setter: (v: T) => void) => (v: T) => {
    if (save.isSuccess || save.isError) save.reset();
    setter(v);
  };

  return (
    <Panel title="Study budget" icon="clock">
      <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
        <p style={{ fontSize: 12.5, color: "var(--ds-muted)" }}>
          xLearn sizes each day’s plan to fit your budget. Reviews always come first.
        </p>

        <BudgetFields weekday={weekday} weekend={weekend} onWeekday={edit(setWeekday)} onWeekend={edit(setWeekend)} />

        <div
          style={{
            padding: "12px 14px",
            background: "var(--ds-inset)",
            borderRadius: 9,
            fontSize: 12,
            color: "var(--ds-dim)",
            display: "flex",
            alignItems: "center",
            gap: 8,
          }}
        >
          <Icon name="bulb" className="xl-ico--sm" style={{ color: "var(--ds-teal)" }} />
          <span>
            At {weekday} min/weekday + {weekendLabel(weekend)}/weekend, you’ll finish the 16-week path in ~
            <b style={{ color: "var(--ds-text)" }}>{budgetEta(weekday, weekend)}</b> at your current pace.
          </span>
        </div>

        <SaveRow save={save} onSave={() => save.mutate({ study_budget: { weekday_minutes: weekday, weekend_band: weekend } })} />
      </div>
    </Panel>
  );
}

// --- API keys (S10 shell) ---

const ADD_PROVIDERS = [
  { key: "anthropic", label: "Anthropic" },
  { key: "openai", label: "OpenAI" },
  { key: "google", label: "Google" },
] as const;

function ApiKeysSection() {
  const coach = useCoachKey();
  const keys = coach.data?.keys ?? [];
  const connected = coach.data?.connected ?? false;
  const [addOpen, setAddOpen] = useState(false);
  const [provider, setProvider] = useState<(typeof ADD_PROVIDERS)[number]["key"]>("anthropic");

  return (
    <Panel
      title="API keys · your coach"
      icon="key"
      right={
        connected ? (
          <span className="ds-badge ds-badge--ok">Connected</span>
        ) : (
          <span className="ds-badge ds-badge--warn">No key — coach off</span>
        )
      }
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 12, color: "var(--ds-muted)" }}>
          <Icon name="lock" className="xl-ico--sm" style={{ color: "var(--ds-ok)" }} />
          Used only for your coach. Stored encrypted; never shared or used for anything else.
        </div>

        {keys.map((k) => (
          <KeyRow key={k.provider} k={k} />
        ))}

        {keys.length === 0 && !coach.isLoading && (
          <div
            style={{
              padding: "13px 15px",
              background: "var(--ds-panel-2)",
              border: "1px solid var(--ds-line-2)",
              borderRadius: 10,
              fontSize: 12.5,
              color: "var(--ds-muted)",
            }}
          >
            No provider key yet — your AI coach is off. Add a key to enable Socratic hints during attempts and a
            reviewer after each solve.
          </div>
        )}

        {!addOpen && (
          <button type="button" className="ds-btn ds-btn--secondary" onClick={() => setAddOpen(true)} style={{ alignSelf: "flex-start" }}>
            <Icon name="plus" className="xl-ico--sm" /> Add a provider
          </button>
        )}

        {addOpen && (
          <div style={{ padding: 16, border: "1px dashed var(--ds-line-2)", borderRadius: 10, display: "flex", flexDirection: "column", gap: 14 }}>
            <div className="ds-field">
              <span className="ds-field__label">Provider</span>
              <div className="ds-seg" role="group" aria-label="Coach provider">
                {ADD_PROVIDERS.map((p) => (
                  <button
                    key={p.key}
                    type="button"
                    className={provider === p.key ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"}
                    aria-pressed={provider === p.key}
                    onClick={() => setProvider(p.key)}
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>
            <div className="ds-field">
              <label className="ds-field__label" htmlFor="add-key">
                API key
              </label>
              <input id="add-key" className="ds-input ds-input--mono" type="password" placeholder={keyPlaceholder(provider)} aria-label="API key" />
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" disabled title="Coach key storage arrives with the coach">
                <Icon name="check" className="xl-ico--sm" /> Add key
              </button>
              <button type="button" className="ds-btn ds-btn--ghost ds-btn--sm" onClick={() => setAddOpen(false)}>
                Cancel
              </button>
              <span style={{ fontSize: 11.5, color: "var(--ds-muted)", marginLeft: "auto" }}>Key storage arrives with the AI coach.</span>
            </div>
          </div>
        )}
      </div>
    </Panel>
  );
}

function KeyRow({ k }: { k: CoachKey }) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 14,
        padding: "13px 15px",
        background: "var(--ds-panel-2)",
        border: "1px solid var(--ds-line-2)",
        borderRadius: 10,
      }}
    >
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <b style={{ fontSize: 13.5 }}>{k.provider}</b>
          <span className="ds-chip ds-chip--xs ds-mono">{k.default_model}</span>
          {k.tested ? (
            <span className="ds-badge ds-badge--ok">Connected</span>
          ) : (
            <span className="ds-chip ds-chip--xs">Untested</span>
          )}
        </div>
        <div className="ds-mono" style={{ fontSize: 12, color: "var(--ds-muted)", marginTop: 3 }}>
          {k.masked_key}
        </div>
      </div>
      <label className={k.enabled ? "ds-toggle ds-toggle--on" : "ds-toggle"} aria-hidden="true">
        <span className="ds-toggle__track">
          <span className="ds-toggle__knob" />
        </span>
      </label>
    </div>
  );
}

function keyPlaceholder(provider: string): string {
  return provider === "anthropic" ? "sk-ant-…" : provider === "openai" ? "sk-…" : "AIza…";
}

// --- Reminders ---

function RemindersSection({ account }: { account: Account }) {
  const save = usePatchMe();
  const [dailyOn, setDailyOn] = useState(account.reminders.daily_reminder_on ?? true);
  const [dailyTime, setDailyTime] = useState(account.reminders.daily_reminder_time ?? "20:00");
  const [alertsOn, setAlertsOn] = useState(account.reminders.revision_due_alerts_on ?? true);
  const timeValid = /^([01]\d|2[0-3]):[0-5]\d$/.test(dailyTime);

  const touch = () => {
    if (save.isSuccess || save.isError) save.reset();
  };

  return (
    <Panel title="Reminders" icon="bell">
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <ToggleRow
          title="Daily study reminder"
          desc="A nudge if you haven’t started today’s plan"
          on={dailyOn}
          onToggle={() => {
            touch();
            setDailyOn((v) => !v);
          }}
        />
        {dailyOn && (
          <div className="ds-field" style={{ maxWidth: 200 }}>
            <label className="ds-field__label" htmlFor="rem-time">
              Remind me at
            </label>
            <input
              id="rem-time"
              className={timeValid ? "ds-input ds-input--mono" : "ds-input ds-input--mono ds-input--invalid"}
              value={dailyTime}
              onChange={(e) => {
                touch();
                setDailyTime(e.target.value);
              }}
              placeholder="20:00"
              inputMode="numeric"
            />
            {!timeValid && <span className="ds-field__error">Use 24-hour HH:MM (e.g. 20:00).</span>}
          </div>
        )}
        <div style={{ borderTop: "1px solid var(--ds-line)", paddingTop: 14 }}>
          <ToggleRow
            title="Revision-due alerts"
            desc="When reviews pile up past your daily budget"
            on={alertsOn}
            onToggle={() => {
              touch();
              setAlertsOn((v) => !v);
            }}
          />
        </div>

        <SaveRow
          save={save}
          disabled={!timeValid}
          onSave={() =>
            save.mutate({
              reminders: { daily_reminder_on: dailyOn, daily_reminder_time: dailyTime, revision_due_alerts_on: alertsOn },
            })
          }
        />
      </div>
    </Panel>
  );
}

function ToggleRow({ title, desc, on, onToggle }: { title: string; desc: string; on: boolean; onToggle: () => void }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
      <span style={{ flex: 1 }}>
        <b style={{ fontSize: 13 }}>{title}</b>
        <div style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>{desc}</div>
      </span>
      <button
        type="button"
        role="switch"
        aria-checked={on}
        aria-label={title}
        className={on ? "ds-toggle ds-toggle--on" : "ds-toggle"}
        onClick={onToggle}
        style={{ background: "none", border: "none", padding: 0 }}
      >
        <span className="ds-toggle__track">
          <span className="ds-toggle__knob" />
        </span>
      </button>
    </div>
  );
}

// --- shared bits ---

function Panel({
  title,
  icon,
  right,
  children,
}: {
  title: string;
  icon?: "clock" | "key" | "bell";
  right?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="xl-panel">
      <div className="xl-panel__h">
        {icon && <Icon name={icon} className="xl-ico--sm" style={{ color: "var(--ds-teal)" }} />}
        <h3>{title}</h3>
        {right && <span style={{ marginLeft: "auto" }}>{right}</span>}
      </div>
      <div className="xl-panel__b">{children}</div>
    </div>
  );
}

/** SaveRow renders the section Save button + its pending/saved/error status. */
function SaveRow({
  save,
  onSave,
  disabled,
}: {
  save: { isPending: boolean; isSuccess: boolean; isError: boolean };
  onSave: () => void;
  disabled?: boolean;
}) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
      <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" disabled={disabled || save.isPending} onClick={onSave}>
        {save.isPending ? "Saving…" : "Save"}
      </button>
      {save.isSuccess && (
        <span style={{ fontSize: 12, color: "var(--ds-ok)", display: "inline-flex", alignItems: "center", gap: 5 }}>
          <Icon name="check" className="xl-ico--sm" /> Saved
        </span>
      )}
      {save.isError && <span style={{ fontSize: 12, color: "var(--ds-err)" }}>Couldn’t save — try again.</span>}
    </div>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 10, color: "var(--ds-dim)", padding: "24px 0" }}>
      <span className="ds-spin" aria-hidden="true" />
      <span>{children}</span>
    </div>
  );
}
