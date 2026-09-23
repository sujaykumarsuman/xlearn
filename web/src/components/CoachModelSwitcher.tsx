import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { COACH_MODELS, COACH_PROVIDERS, coachModelLabel, providerLabel, useCoachKey, usePutCoachKey, type ProviderId } from "../lib/settings";
import { Icon } from "./Icon";
import { ProviderLogo } from "./ProviderLogo";

/**
 * CoachModelSwitcher is the header quick-switch (F006): a compact icon pill toggle of the
 * connected providers + a dropdown of the active provider's models. Switching either sets
 * the account DEFAULT (the same state Settings edits), so the coach's brain can change from
 * any screen without opening Settings. Renders nothing until a coach is connected.
 */
export function CoachModelSwitcher() {
  const coach = useCoachKey();
  const put = usePutCoachKey();
  const keys = coach.data?.keys ?? [];
  const defaultKey = keys.find((k) => k.is_default);
  const active = defaultKey?.provider as ProviderId | undefined;

  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    const onPointer = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [open]);

  if (keys.length === 0 || !active) return null; // nothing to switch until a coach is connected

  const connectedProviders = COACH_PROVIDERS.filter((p) => keys.some((k) => k.provider === p.id));
  const activeModels = COACH_MODELS[active];

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
              onClick={() => active !== p.id && put.mutate({ provider: p.id, default: true })}
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
          <span>{coachModelLabel(defaultKey!.default_model)}</span>
          <Icon name="chevdown" className="xl-ico--sm" style={{ color: "var(--ds-muted)" }} />
        </button>

        {open && (
          <div className="xl-menu" role="listbox">
            <div className="xl-menu__head">{providerLabel(active)} · models</div>
            {activeModels.map((m) => {
              const on = defaultKey!.default_model === m.id;
              return (
                <button
                  key={m.id}
                  type="button"
                  role="option"
                  aria-selected={on}
                  className={on ? "xl-mitem xl-mitem--on" : "xl-mitem"}
                  onClick={() => {
                    if (!on) put.mutate({ provider: active, default_model: m.id, name: defaultKey!.name || coachModelLabel(m.id) });
                    setOpen(false);
                  }}
                >
                  <span style={{ flex: 1 }}>
                    <span className="xl-mitem__label">{m.label}</span>
                    <span className="xl-mitem__hint">{m.hint}</span>
                  </span>
                  {on && <Icon name="check" className="xl-ico--sm" style={{ color: "var(--ds-teal)" }} />}
                </button>
              );
            })}
            <div className="xl-menu__sep" />
            <Link to="/settings?tab=coach" className="xl-mitem xl-mitem--link" onClick={() => setOpen(false)}>
              Manage in Settings…
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
