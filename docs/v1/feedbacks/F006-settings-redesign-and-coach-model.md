# F006 — Settings redesign + choose your coach's model

## Feedback

Round 4, on the **Settings** screen:

1. **Center the content** — it ran full-bleed and felt unanchored.
2. **Coach = pick a brain.** The API-key section should offer **two providers — Anthropic
   and OpenAI, each with its logo** — let you **choose a model from a curated list** (the
   ones we recommend for best results), and **name** the coach (default name = the model's
   own name), so the account picks a model *as* their coach.
3. **A better-designed page** — "it's very plain and simple; use modern, minimal components."

## Decision

Iterated with the owner via a **design-canvas mockup** (five interactive artboards) before
building. The design landed on **more than one coach**:

- **Connect one key PER PROVIDER** (Anthropic *and* OpenAI), each with its own model and
  name, and **one marked the default** — the model the coach answers with. (Superseded the
  first-round "one coach" plan.)
- **Curated model list** (owner's pick, current families) — a **Custom…** escape hatch for any
  exact id:
  - Anthropic — **Opus 5**, **Opus 4.8**, **Sonnet 5**
  - OpenAI — **GPT-5.6 Sol** (deepest), **GPT-5.6 Terra** (balanced), **GPT-5.6 Luna** (fastest)
  - ids: `claude-opus-5` · `claude-opus-4-8` · `claude-sonnet-5` · `gpt-5.6-sol` ·
    `gpt-5.6-terra` · `gpt-5.6-luna` (OpenAI ids confirmed via web search).
- **Header quick-switch.** A compact icon **pill switch** (Anthropic/OpenAI) + the active
  provider's **model dropdown**, in the top bar, so the coach's brain can change from any
  screen (same default state Settings edits; only connected providers appear).
- **Change model/name/default without re-pasting the key.** `PUT /coach/key` gained four
  provider-keyed modes: store/replace, **set default**, toggle enabled, and meta update.
- **Two-column, pill/circle layout.** A section **rail** (sticky) + a column of cards;
  fully-rounded (pill) controls and circular tiles per the owner's shape preference. Reuses
  `theme.css` tokens (dark theme kept).

## Scope / changes

- **coach service**
  - migrations: `00002_key_name.sql` (`name`), `00003_multi_provider_keys.sql` (drop
    `UNIQUE(account_id)` → `UNIQUE(account_id, provider)`, add `is_default`, backfill).
  - `store/queries/api_key_config.sql` — per-provider CRUD: `ListApiKeyConfigs`,
    `GetApiKeyConfig` (account+provider), `GetDefaultApiKeyConfig`, `CountApiKeyConfigs`,
    `UpsertApiKeyConfig` (+`is_default`), `UpdateApiKeyMeta`, `SetApiKeyEnabled`,
    `SetDefaultProvider`, `PromoteEarliestDefault`, `DeleteApiKeyConfig`.
  - regenerated sqlc; `store.go` — `KeyConfig.IsDefault`, `ListKeys`/`GetKey(provider)`/
    `GetDefaultKey`/`SetDefault`/`DeleteKey(provider)` maintaining the "exactly one default"
    invariant (first key defaults; delete promotes a survivor).
  - `handlers.go` — `GET /keys` returns `keys[]` + `default_provider`; `PUT /keys` four
    provider-keyed modes; `DELETE /keys?provider=`; chat uses the **default** key + disables
    the right provider on auth failure. `providers.go` default-model fallbacks refreshed.
  - tests: `TestMultiProviderAndDefault`, `TestSetDefaultUnknownProvider404`, updated CRUD +
    integration (`multi-provider keys, default, and delete-promotes`).
- **gateway** — `deleteKey` forwards `?provider=`; empty-state includes `default_provider`.
- **docs** — `openapi.yaml` `CoachKey.is_default`, `CoachKeyList.default_provider`,
  `CoachKeyRequest` 4 modes + `default`, DELETE `provider` param; drift check green.
- **web**
  - `lib/settings.ts` — `is_default`/`default_provider`, `default` on the body,
    provider-scoped delete, and the shared `COACH_MODELS`/`COACH_PROVIDERS`/`coachModelLabel`.
  - `components/ProviderLogo.tsx` (new), `components/CoachModelSwitcher.tsx` (new header
    quick-switch), mounted in `Topbar.tsx`; `Coach.tsx` keys usability off the default.
  - `screens/Settings.tsx` — rewritten: section **rail** + cards; the **profile folded into
    the rail**, styled like a **GitHub profile** — a large avatar (with a change badge), the
    name, an **Edit-profile** button, then meta rows (**local time** computed from the timezone,
    and email); editing opens a compact form (name + timezone, email read-only). Visual-only —
    no new account fields (bio/location/website deferred). `CoachSection` with a per-provider
    `ProviderPanel` (connect/edit, model pills + Custom, name, key, set-default, remove);
    pill/circle shapes.
  - `styles/app.css` — `.xl-settings__grid`, `.xl-set-rail`/`.xl-rail`, `.xl-prov*`,
    `.xl-mpill`, `.xl-defbadge`/`.xl-setdef`, the `.xl-coachsw`/`.xl-pillsw`/`.xl-modelbtn`/
    `.xl-menu` header switch, and the in-`.xl-settings` pill/circle shape pass (theme.css
    untouched).
  - `screens/Settings.test.tsx` — rewritten to the multi-key UI (9 tests).

## Status

✅ **Done — in local review.** Code-complete and green (Go build/vet/test, `sqlc diff`, web
typecheck/lint, 83 web tests). Rebuilt on the local docker-compose stack for the owner to
review at `http://localhost:8080/xlearn` → Settings + the header quick-switch. Land-and-sync
only after an explicit go-ahead — it ships as `v1.2.0` (coach schema migration included).

## Notes

- Provider marks are simple nominative-use brand glyphs (no wordmark lockups).
- The curated list is easy to extend: `COACH_MODELS[provider]` in `lib/settings.ts`
  (shared by Settings + the header switch).
- Coach keys remain **envelope-encrypted**; the meta/default/toggle paths never read or
  re-seal the key. The chat answers with the **default** provider's key + model.
