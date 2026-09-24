# F010 — Coach: onboarding key save, provider-error handling, cut-off replies

## Feedback

Three confirmed v1 coach/onboarding bugs, verified on `main@2681862` (2026-09-24):

1. **The onboarding API-key step threw the key away.** The `onb-key` input in `Auth.tsx` was
   uncontrolled, so the key was never sent, while the copy said "Stored encrypted with the coach". It
   also offered a **Google** provider the backend doesn't support (the coach DB CHECK and
   `providerFor` allow only `anthropic` and `openai`). `identity.onboarding.key_added` was a dead
   flag: nothing ever set it.
2. **Quota errors disabled the learner's key.** `providers.go` mapped OpenAI `insufficient_quota`
   (and any `permission_error`) to `ErrProviderAuth`, which flips `enabled=false`. A learner who ran
   out of credit had to re-enter a valid key after topping up.
3. **Anthropic replies were cut short.** `maxProviderTokens = 1024` was sent to Anthropic only.
   Current models (e.g. `claude-sonnet-5`) think at high effort by default, and thinking counts
   toward `max_tokens`. `stop_reason` was never read, so a cut-off reply looked complete.

## Decision

- **Onboarding saves the key or says why not.** Finish calls the existing `PUT /coach/key` (the same
  body Settings sends: provider + key + that provider's first curated model + its label) and marks
  onboarding finished **only after the save succeeds**. A failed save shows the server's reason plus
  "check it and try again, or skip for now", and **Skip** completes onboarding without the key.
  Finish with no key typed just completes. Google is gone; the step reuses `COACH_PROVIDERS`.
- **`key_added` is removed, not set.** Whether a coach key is connected is coach's data
  (`GET /coach/key` → `connected`). A copy in identity would go stale as soon as a key is deleted
  in Settings, and nothing read it. It is dropped from `GET /me` (`onboarding`), the store type, the
  SPA type and the OpenAPI schema. The **column stays** for now: the old identity pods' generated
  queries name it, so dropping it in the same release would break them mid-rollout. A later
  migration can drop it.
- **Typed provider errors.** Every upstream rejection is a `*ProviderError{Kind}`:
  - `auth`: **only** HTTP 401, OpenAI `invalid_api_key`, or an `authentication_error` type. Disables
    the key (unchanged 409 `provider_auth`).
  - `limited`: out of credit or quota, billing (402), spend limits (Anthropic's 400 "You have
    reached your specified [workspace] API usage limits" and its 429 spend cap; OpenAI's 429
    `insufficient_quota` / `credit_balance_exhausted` / `*_spend_limit_exceeded`), rate limits (429),
    and model/region permission errors (403, `model_not_found`). **The key stays enabled.** The
    learner gets "your provider account is out of credit or limited — top up and retry" as a 429
    `provider_limited` before streaming, or an SSE `error` frame mid-stream.
  - `unavailable`: everything else (5xx, overloaded, a malformed request). Unchanged 502
    `provider_error`.

  The provider's free-text message is read only to classify (Anthropic's spend-limit 400 can't be
  told apart from a bad request otherwise) and is never logged or returned. OpenAI's invalid-key
  message echoes a partially masked key.
- **Bigger cap, explicit low effort, marked cut-offs.**
  - The cap is now **4096** on both providers: Anthropic `max_tokens`, and OpenAI
    `max_completion_tokens` (the deprecated `max_tokens` is rejected by reasoning models).
  - The coach asks for **low effort**: Anthropic `output_config: {"effort": "low"}` (GA, no beta
    header) and OpenAI `reasoning_effort: "low"`. Each is sent only to models documented to accept
    it, because a learner can type any model id in Settings and an unsupported model answers the
    field with a 400:
    - Anthropic: the effort page's model list, e.g. sonnet-5, opus-5, opus-4-8; not Haiku 4.5.
    - OpenAI: the gpt-5 and gpt-6 reasoning families, not gpt-4o or the `-chat` variants.
  - The providers now read `stop_reason` (Anthropic `message_delta`) and `finish_reason` (OpenAI).
    A reply is **truncated** when it hit the cap (`max_tokens` /
    `model_context_window_exceeded` / `length`) or the stream ended before its terminal frame.
  - A truncated reply gets a visible note, "⚠️ This reply was cut off at the length limit — ask me
    to continue." It is streamed live and persisted with the reply, so a reload matches and the
    next turn's history tells the model its last answer was incomplete. The `done` frame carries
    `truncated`.

Parameter names, error types and model support were checked against the current docs on 2026-09-24:
- Anthropic: `platform.claude.com/docs/en/build-with-claude/effort`, `/handling-stop-reasons`,
  `/api/errors`, `/api/rate-limits`.
- OpenAI: `developers.openai.com/api/reference` (chat completions, streaming events) and
  `/api/docs/guides/error-codes`.

## Scope / changes

- **coach**: `providers.go` — `ProviderError` / `ProviderErrorKind` + `ErrProviderLimited`;
  `classifyOpenAI` / `classifyAnthropic`; error bodies parsed (not echoed) for classification;
  `Stream` returns a `StreamResult{StopReason, Truncated}`; cap 4096; model-gated effort params.
  `handlers.go` — `finishChat` gains the `limited` branch (key kept, 429 / SSE `provider_limited`)
  and the truncation note. Tests: `providers_test.go` (status + in-stream classification tables for
  both providers, effort gating, truncation), `handlers_test.go` (limited keeps the key pre- and
  mid-stream, truncated reply marked + persisted).
- **identity**: `key_added` dropped from `store.Onboarding`, `onboardingJSON`, the
  `CompleteOnboarding` query comment (+ regenerated sqlc). Column retained.
- **web**: `Auth.tsx` `StepCoach` — controlled key input, PUT-then-finish, error + skip path, no
  Google. `lib/settings.ts` — `CoachChatError.isProviderLimited`, server error message kept.
  `Coach.tsx` — the top-up message for `provider_limited` (no bounce to Settings). `lib/auth.ts` —
  `key_added` dropped. Tests: `Auth.test.tsx` (providers, save → finish order + body, failed save
  keeps the step and Skip completes, no-key Finish skips the PUT), `Coach.test.tsx` (limited
  message).
- **docs**: `openapi.yaml` (`Onboarding` minus `key_added`; `/coach/chat` 409/429 + the `done` /
  `error` frame codes), `data-model.md`.

No infra change and no migration.

## Status

🚀 **Shipped (`v1.5.1`)**. Reviewed on the local docker-compose stack (`DEV_AUTH=1`) against a fake
provider, then shipped on the owner's go-ahead. Release path: merged (xlearn#49, `fff530e`) → `v1.5.1`
tag → deploy built all seven `1.5.1` images → Flux deployed.

**Verified live:**
- every service logs `v1.5.1`;
- `/api/v1/healthz` returns 200;
- `/api/v1/me`, `/api/v1/coach/key` and `POST /api/v1/coach/chat` are session-gated (401);
- the prod SPA bundle carries the new onboarding and `provider_limited` copy, and the Google key
  placeholder is gone.

**Green throughout:** `go vet`, `go test -race ./...`, `sqlc diff`, web typecheck / lint / 103 tests /
build, and CI (go, web, e2e).

## Notes

- A 429 `provider_limited` means the **provider** limited the learner's account, not that xLearn
  rate-limited them.
- Follow-up: a migration to drop `identity.onboarding.key_added` once this release is fully rolled
  out.
