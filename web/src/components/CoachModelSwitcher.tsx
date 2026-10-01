import { Fragment, useId, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  COACH_MODEL_ID,
  COACH_PROVIDERS,
  coachCapabilityTags,
  coachFeatureDefault,
  coachModelLabel,
  coachPriceCaption,
  coachPriceLine,
  catalogModelsFor,
  providerLabel,
  usePopoverDismiss,
  useCoachKey,
  useCoachModels,
  usePutCoachKey,
  type CoachModel,
  type ProviderId,
} from "../lib/settings";
import { Icon } from "./Icon";
import { ProviderLogo } from "./ProviderLogo";

/**
 * CoachModelSwitcher is the header quick-switch (AB01 F12): a compact icon pill toggle of
 * the connected providers + a dropdown of the active provider's models, drawn from the
 * SERVER catalog (GET /coach/models). Switching either sets the account's `coach` feature
 * default — the same state Settings edits — so the coach's brain can change from any
 * screen without opening Settings. Renders nothing until a coach is connected.
 */
export function CoachModelSwitcher() {
  const coach = useCoachKey();
  const catalog = useCoachModels();
  const put = usePutCoachKey();
  const keys = coach.data?.keys ?? [];
  const current = coachFeatureDefault(coach.data, "coach");
  const active = current?.provider as ProviderId | undefined;

  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  usePopoverDismiss(ref, open, () => setOpen(false));

  if (keys.length === 0 || !active || !current) return null; // nothing to switch until a coach is connected

  const connectedProviders = COACH_PROVIDERS.filter((p) => keys.some((k) => k.provider === p.id));

  return (
    <div className="xl-coachsw" ref={ref}>
      {connectedProviders.length > 1 && (
        <div className="xl-pillsw" role="group" aria-label="Coach provider">
          {connectedProviders.map((p) => (
            <button
              key={p.id}
              type="button"
              className={active === p.id ? "xl-pillsw__btn xl-pillsw__btn--on" : "xl-pillsw__btn"}
              aria-label={p.label}
              aria-pressed={active === p.id}
              disabled={put.isPending}
              onClick={() => active !== p.id && put.mutate({ provider: p.id, default: true, feature: "coach" })}
            >
              <ProviderLogo provider={p.id} size={16} bare />
            </button>
          ))}
        </div>
      )}

      <div style={{ position: "relative" }}>
        <button type="button" className="xl-modelbtn" aria-haspopup="listbox" aria-expanded={open} aria-label="Coach model" onClick={() => setOpen((o) => !o)}>
          <span className="xl-modelbtn__lg">
            <ProviderLogo provider={active} size={13} bare />
          </span>
          <span>{coachModelLabel(current.model, catalog.data?.models)}</span>
          <Icon name="chevdown" className="xl-ico--sm" style={{ color: "var(--ds-muted)" }} />
        </button>

        {open && (
          <CoachModelMenu
            groups={[{ provider: active, models: catalogModelsFor(catalog.data?.models, active, current.model) }]}
            asOf={catalog.data?.as_of}
            selected={current.model}
            customProvider={active}
            onPick={(provider, model) => {
              if (model !== current.model) put.mutate({ provider, default: true, feature: "coach", default_model: model });
              setOpen(false);
            }}
            onManage={() => setOpen(false)}
          />
        )}
      </div>
    </div>
  );
}

/** One provider's section of the catalog list: its head line and its models. */
export interface CoachModelGroup {
  provider: string;
  models: CoachModel[];
}

/**
 * CoachModelMenu is the catalog list itself (AB01 F12) — the model rows with their
 * capability tags, dated prices and Recommended badge, plus the "Custom model id…" escape
 * hatch. It is shared by the header quick-switch and Settings' per-feature "Change", which
 * is why it takes groups (Settings may list two connected providers at once) and reports
 * BOTH provider and model id to onPick.
 *
 * There is deliberately no base-URL field: a learner picks a model id, never an endpoint.
 *
 * Dismissal (Esc, outside click) belongs to the caller, which owns the trigger button —
 * handling it here would close the menu on the very pointerdown that is meant to toggle it.
 */
export function CoachModelMenu({
  groups,
  asOf,
  selected,
  customProvider,
  onPick,
  onManage,
  failing,
  ariaLabel = "Coach model",
}: {
  groups: CoachModelGroup[];
  /** The catalog's `as_of`; absent when the catalog could not be read, which also drops the
   *  dated price caption — an undated price is worse than none. */
  asOf?: string;
  selected: string;
  /** The provider a custom id is stored against (a custom id is not in the catalog, so it
   *  cannot name its own provider). */
  customProvider: string;
  onPick: (provider: string, model: string) => void;
  /** Shows the "Manage in Settings…" row; omitted inside Settings itself. */
  onManage?: () => void;
  /** A model the key may not use (F11) — it stays listed and selected, marked, with no
   *  price and no tags. */
  failing?: string;
  ariaLabel?: string;
}) {
  const [customOpen, setCustomOpen] = useState(false);
  const [custom, setCustom] = useState("");
  const fieldId = useId();
  const errorId = `${fieldId}-error`;
  const typed = custom.trim();
  const valid = COACH_MODEL_ID.test(typed);
  const invalid = typed !== "" && !valid;

  return (
    <div className="xl-menu" role="listbox" aria-label={ariaLabel}>
      {groups.map((g) => (
        <Fragment key={g.provider}>
          <div className="xl-menu__head">
            {providerLabel(g.provider)} · models
            {asOf && <span>{coachPriceCaption(asOf)}</span>}
          </div>
          {g.models.map((m) => {
            const on = selected === m.id;
            const cantUse = failing === m.id;
            return (
              <button
                key={`${g.provider}:${m.id}`}
                type="button"
                role="option"
                aria-selected={on}
                className={on ? "xl-mitem xl-mitem--on" : "xl-mitem"}
                onClick={() => onPick(g.provider, m.id)}
              >
                <span className="xl-mitem__main">
                  <span className="xl-mitem__label">
                    {m.label}
                    {m.recommended && <span className="ds-badge ds-badge--ok">Recommended</span>}
                  </span>
                  <span className="xl-mitem__meta">
                    {cantUse ? (
                      <span className="xl-mitem__err">This key can't use it</span>
                    ) : (
                      <>
                        {coachCapabilityTags(m).map((t) => (
                          <span key={t} className="xl-tag">
                            {t}
                          </span>
                        ))}
                        {m.price ? (
                          <span className="xl-price">{coachPriceLine(m.price)}</span>
                        ) : (
                          <span className="ds-chip ds-chip--warn ds-chip--sm">cost unknown</span>
                        )}
                      </>
                    )}
                  </span>
                  {m.covered_model && <span className="xl-mitem__hint">Your provider keeps these chats 30 days</span>}
                </span>
                {on && <Icon name="check" className="xl-ico--sm" style={{ color: "var(--ds-teal)" }} />}
              </button>
            );
          })}
        </Fragment>
      ))}

      <div className="xl-menu__sep" />
      <button type="button" role="option" aria-selected={false} className="xl-mitem xl-mitem--link" onClick={() => setCustomOpen((o) => !o)}>
        <Icon name="plus" className="xl-ico--sm" /> Custom model id…
      </button>
      {customOpen && (
        <div className="xl-menu__custom">
          <label className="ds-field__label" htmlFor={fieldId}>
            Model id
          </label>
          <div className="xl-menu__customrow">
            <input
              id={fieldId}
              className={invalid ? "ds-input ds-input--mono ds-input--invalid" : "ds-input ds-input--mono"}
              value={custom}
              aria-invalid={invalid || undefined}
              aria-describedby={invalid ? errorId : undefined}
              autoComplete="off"
              spellCheck={false}
              onChange={(e) => setCustom(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && valid) onPick(customProvider, typed);
              }}
            />
            <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" aria-disabled={!valid} disabled={!valid} onClick={() => onPick(customProvider, typed)}>
              Use
            </button>
          </div>
          {invalid ? (
            <span className="xl-mitem__err" id={errorId}>
              That isn't a model id. Use letters, digits and . _ : - (no URLs).
            </span>
          ) : (
            <span className="xl-mitem__meta">
              <span className="ds-chip ds-chip--warn ds-chip--sm">cost unknown</span> Not in the catalog, so this month's estimate leaves it
              out.
            </span>
          )}
        </div>
      )}

      {onManage && (
        <>
          <div className="xl-menu__sep" />
          <Link to="/settings?tab=coach" className="xl-mitem xl-mitem--link" onClick={onManage}>
            Manage in Settings…
          </Link>
        </>
      )}
    </div>
  );
}
