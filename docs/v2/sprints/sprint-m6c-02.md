# Sprint m6c-02 — Outline: multi-speaker fairness check → AI-proposed voice Communication; history-calibrated estimates

> **Milestone:** M6c — interviewer extras (v2.2) · **Track:** product · **Order:** 88 · **Kind:** **plan-outline card** (BP1: no prompt until v2.2 planning)
> **Prereqs:** [m6b-04](sprint-m6b-04.md) (`v2.1.0` interviewer GA live). Builds on:
> - [m6a-03](sprint-m6a-03.md): the twin fairness gate, proposal → explicit accept → `ScoreMock` once, and `scored_by`;
> - [m6a-01](sprint-m6a-01.md): the manifest's `mock.rubric.dims[].evidence` and `ai: propose | self_only | self_only_voice`;
> - [m6a-02](sprint-m6a-02.md): the brain, `store:false` and the replay fixtures;
> - [m6b-02](sprint-m6b-02.md): per-segment usage and cost, and the mandatory $ cap;
> - [spk-04](sprint-spk-04.md): S6's M11 result, the ASR fidelity on DSA terms.
>
> **Unblocks:** nothing gates on it. Siblings: [m6c-01](sprint-m6c-01.md), [m6c-03](sprint-m6c-03.md).
> **Sources:** [rollout §4](../rollout-plan.md#m5-later-2x-and-m6c-v22) (the M6c row, "the fairness check") and [§8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading) (the v2.2+ row); [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) item 7 and [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys).
> **Release action:** **outline only (v2.2).** This card merges nothing and tags nothing. Once expanded:
> - the estimator and the proposal-path code are "merge only", shipping dark in `v2.1.x` patches;
> - the fairness run is an **owner calendar event**: no merge, with results recorded through a docs PR;
> - the **voice-Communication default flip** is a T-1 manifest change. It is labelled at **whichever labelled minor follows a passed run** ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): labels mark default flips). That is the M6c GA minor (the next free minor after `v2.1.0`: `v2.2.0` indicative, or the next one if [m5-01](sprint-m5-01.md) has already taken it) if the run has passed by then, and a later 2.x minor otherwise. It **never blocks M6c GA**.
>
> **Calendar:** v2.2+, after `v2.1.0` (≈ Q1 2027) *(inferred)*. The fairness run needs the owner and about 10 speakers, so it is an event and not a sprint.
>
> **Execute with:** no prompt yet. v2.2 planning expands this card into a full plan and writes `../prompts/prompt-m6c-02.md` (one prompt, one session). Expect a share of the M6c design sprint (`ds-m6c-01`, see the entry gates), one build session, one session to prepare the event, and the event itself.

## Status

_Overall:_ ⬜ Not started. This is an **outline card**: expand it at v2.2 planning before doing any work.

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Expand this card into a full plan and prompt at v2.2 planning | X | ⬜ |
| 1a | Sketch: the multi-speaker fairness check (≥ 10 consenting speakers, ≥ 1 disfluent), run as an owner event | X · O | ⬜ |
| 1b | Sketch: AI-proposed voice Communication, a manifest flip for each passed shell × brain model × `prompt@v` | X | ⬜ |
| 1c | Sketch: history-calibrated cost and time estimates at pre-flight | X | ⬜ |

> Rows 1a–1c are sketches, not commitments. The expansion replaces them with real tasks.
> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line to match, and mirror the sprint's state into [`../status.md`](../status.md):
> - the Sprint board row ("⬜ outline" until the expansion);
> - the M6c milestone row;
> - the owner-events table, once the fairness event is booked;
> - the flag inventory's permanent kill switches, once `VOICE_COMM_AI` exists.
>
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **M6b shipped:** `v2.1.0` (interviewer GA) is live and verified ([m6b-04](sprint-m6b-04.md)). [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) is Accepted with the S6 results folded in.
- [ ] **This card has been expanded** at v2.2 planning into a full plan plus `prompt-m6c-02.md` (BP1).
- [ ] **The [m6a-03](sprint-m6a-03.md) twin gate is green** for every brain model to be checked. The multi-speaker check extends that per-model gate to voice ([t6 §6](../research/t6-realtime-interviewer.md#6-assessment) item 7). A model that failed the twin gate is `self_only` and isn't checked.
- [ ] **The S6 M11 result and the shipped shell are known** ([spk-04](sprint-spk-04.md) results note). If M11 was below 80%, timeline evidence is already dropped in voice mode, and the check design must account for that.
- [ ] **Board variants are frozen before the expansion's first UI sprint** (BP3). They are drafted in the shared M6c design sprint (e.g. `ds-m6c-01`, whose PR merges on CI green: the merge is the freeze, D40; see [m6c-01](sprint-m6c-01.md)'s entry gates):
  - the **AB27** voice-Communication proposal row: a proposed band with verified quotes, replacing "you set this";
  - the **AB24/AB29** history-based estimate line, "Based on your last N sessions".

  If that design sprint merged without these variants (e.g. this card is expanded later, after the fairness run), follow the [m5-01](sprint-m5-01.md) precedent instead: variant screenshots committed with the build PR, and the merge freezes the variant (D40; the owner may revise it later with a follow-up PR).
- [ ] **For 1a's run** (an owner event, not a sprint):
  - the owner has recruited **≥ 10 consenting speakers** across accents, **including ≥ 1 disfluent speaker**, each with a signed consent;
  - every speaker is **18+** ([t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)) and **not resident in the EU/EEA**, unless the legal review that [ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits) requires for voice has cleared it;
  - the owner is present;
  - there is a throwaway OpenAI project with an enforced **hard spend limit**, as in S6.
- [ ] **For 1c (soft):** the owner's own voice sessions provide ≥ 2 sessions of real `cost_micros`. If they don't, 1c is tested with fixtures only.
- [ ] **Parallel sessions:** before claiming an ADR number or tagging, check peers' open PRs, tags and worktrees (`gh pr list`, `git ls-remote --tags origin`, `git worktree list`, ListAgents).

## Goal

1. **Make voice Communication AI-proposable only once a multi-speaker fairness check passes** ([ADR-0032 §3](../../adr/0032-realtime-ai-mock-interviewer.md#3-what-the-ai-perceives-a-second-interviewer-sharing-the-session-owner-d29): self-scored "until a multi-speaker fairness check passes (P2)"). A pass applies only to the exact shell, brain model and review prompt that were checked. Any change reverts to self-scoring.
2. **Make the pre-flight estimate learn from the learner's own history** ([t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)): once the learner has ≥ 2 sessions, their own p50/p90 cost per active minute replaces the catalog defaults.

## Scope

**In (outline)**
- **1a:** the multi-speaker fairness check: scripts, harness, pass rule, results note and consent handling.
- **1b:** the voice Communication dimension becomes `propose` for passed combinations, with a kill switch back to `self_only_voice`, plus the AB27 debrief/proposal variant.
- **1c:** history-calibrated cost (and optionally wall-time) estimates in the AB24/AB29 pre-flight.

**Out**
- The photo, read-aloud and local recording → [m6c-01](sprint-m6c-01.md). Safari, voice-lite, the canary and AB31 → [m6c-03](sprint-m6c-03.md).
- **Scoring delivery** (pace, fillers, fluency, accent or tone). These are **never assessed** ([t6 §6](../research/t6-realtime-interviewer.md#6-assessment); EU AI Act Art. 5(1)(f)). Delivery timing stays informational only.
- **The public route:** it stays mock count only (D31) whatever `scored_by` says.
- Any platform-key use. The interviewer is BYO only, and review calls stay on the learner's key.
- Other rubric dimensions. They are already `propose` in voice mode, with timeline evidence advisory.

## Tasks

### 1 · Outline only: expand at v2.2 planning [X]

The v2.2 planning session (docs only) replaces this card with a full plan and prompt. It must:
- **Re-read what exists by then:** the Accepted ADR-0032, the S6 results note, m6a-03's twin-gate harness and its recorded tolerance, and m6b-02's segment log schema.
- **Turn the run into an owner event:** add `ev-m6c-speakers` to the owner-events table, **prepared by** the expanded sprint (scripts, harness, consent form, runbook) and executed with the owner present. Whichever prompt runs the event lists its owner-only steps (the recruiting, the throwaway OpenAI project with its hard limit, being present) under `## Before you launch (owner)` (D40).
- **Record the pass rule:** as a decisions-log entry. It needs an ADR that amends ADR-0032 §3 only if it departs from [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) item 7. Run the parallel-sessions check before numbering.
- **Add the kill switch:** a **permanent** T-2 kill switch, e.g. `VOICE_COMM_AI`. It is unset by default, so shipping it needs no infra PR. Setting it to `off` is an infra PR (the kill path), and it forces `self_only_voice`. List it with the permanent kill switches and operating modes in the status.md flag inventory: it has **no removal milestone** and doesn't count toward the ≤ 6 live non-kill flags ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)). The pass-row check stays the standing gate underneath it. ADR-0034 §2 names five permanent kill switches, so add a decisions-log line recording this sixth one; the status.md inventory is the living list.
- **Decouple the flip from M6c GA:** the Communication default flips at whichever labelled minor follows a passed run. It rides the M6c GA tag sprint (added by [m6c-01](sprint-m6c-01.md) task 1) only if the run has passed by then, and it never holds that tag.

### 1a · Sketch: multi-speaker fairness check [X · O]

Sources:
- [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) item 7: "at least 10 consenting speakers across accents, including at least one disfluent speaker, read scripted answers through the real shell, and the result must show ±1-band parity";
- the [t6 §14](../research/t6-realtime-interviewer.md#14-risks) row on ASR errors for Indian English and accents;
- the calibration-gate pattern in [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (model classes resolve to pinned IDs only through a passed calibration or acceptance row) and [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) (the gate itself).

**Scripts [X].** A fixed, public-safe set of scripted candidate answers:
- e.g. 3 short DSA-mock excerpts of known Communication quality: strong, medium and weak;
- plus 1 injected-instruction script;
- written once, with no pack material (safety invariant S1).

**Harness [X].** Throwaway, in the scratchpad, like S6's. It is never committed with any speaker data. Each speaker reads each script through the **real shipped shell** on the owner's key:
- the shell's input transcription becomes the transcript;
- the **real review call** (`xlearn.mock_review@1`, with the same brain model and `prompt@v` as production) runs **× 3 samples** and yields a Communication band;
- a **text reference** comes from running each typed script through the same review.

**Pass rule (to refine at expansion).**
- For every script, every speaker's median band is within **±1** of the text reference.
- No speaker is systematically lower. A candidate bound is mean shortfall ≤ 0.5 band.
- The injection script gets `review_flag` for every speaker.
- Record the result per (shell, brain model, `prompt@v`).

**Data handling.**
- Speakers are **not xLearn accounts**: no invites, and no change to D35's owner-only use.
- Speakers are **18+ and not EU/EEA-resident** unless the ADR-0032 §6 legal review has cleared it (entry gates). The consent form asks the speaker to confirm both.
- The consent form says:
  - their audio goes to OpenAI under the owner's org, with 30-day abuse logs and no training by default;
  - xLearn stores no audio;
  - only scrubbed transcripts and bands are kept.
- The results note uses **synthetic speaker ids**. An accent label is optional and self-described, and it is reported only in aggregate. The repo is public, so there are no names and no demographic inference.

**Cost** *(inferred)*:
- 10 speakers × 4 scripts × ~2 min ≈ 80 min of shell time ≈ $4 on GPT-Live;
- ≈ 120 review calls ≈ $7;
- ≈ $11 in total, under a $15 hard limit with the harness stopping at $12.

**Output [X].**
- A results note, `docs/v2/research/m6c-voice-fairness.md` (or a t6 addendum), via a docs PR;
- a **pass row** compiled into the coach catalog for the passed combination.

**Re-run trigger.** Any change to the shell, the brain model or the review `prompt@v` invalidates the pass for that combination, and voice Communication falls back to `self_only_voice` until a re-run passes.

### 1b · Sketch: AI-proposed voice Communication [X]

**Manifest.** In `curriculum/dsa/course.json`, and in any course with a mock rubric, `mock.rubric.dims[communication].ai` changes from `self_only_voice` to `propose`. This is a **T-1 default**. It takes effect **only when** the running (shell, brain model, `prompt@v`) has a pass row: the "presence by config" rule of [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service). The permanent T-2 kill switch (`VOICE_COMM_AI=off`) forces `self_only_voice`.

**coach** (`internal/coach/interview`, the m6a-03 proposal path):
- the proposal stops withholding a Communication band in voice mode for passed combinations;
- quotes are still verified with normalise-then-substring;
- missing evidence still gives `band:null`;
- timeline evidence stays **advisory** in voice mode;
- custom model ids still never get a proposal.

**Unchanged:** the `ai-byo` conditions ([t6 §6](../research/t6-realtime-interviewer.md#6-assessment) step 5), the explicit Accept, and the rule that there is no auto-accept for mocks.

**web.** An AB27 variant: in voice mode, the Communication row shows a proposed band with verified quotes instead of "you set this". The copy reads: *"Proposed from what you said, never how you sounded."*

**Lints and tests.**
- The never-list CI lint still passes.
- The evidence enum stays `transcript | code | judge | timeline | hints`, with no `audio` member.
- Replay-fixture tests cover:
  - a passed combination → Communication proposed;
  - an unpassed combination or the kill switch → `self_only_voice`;
  - a custom model → no proposal.

### 1c · Sketch: history-calibrated cost and time estimates [X]

Source: [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys): "Once the learner has at least 2 sessions, the p50 / p90 of their own `segment.cost_micros` per active minute replace the defaults (P2)."

**coach.** A read over the learner's **own** segment log ([m6b-02](sprint-m6b-02.md); 90-day retention; erased with the account):
- p50 and p90 µUSD per active minute, keyed by shell, model and price-table version;
- "typical" = p50 × planned minutes × time multiplier;
- "plan for" = p90 × the same;
- the default $ cap is still "plan for" × 1.5, and the learner can edit it;
- fall back to the catalog defaults below 2 sessions, or when the shell, model or prices change.

**Time (optional; decide at expansion).** The learner's median wall time against the rail, including pauses and holds, e.g. "usually takes ~52 min". t6 only specifies cost.

**web.** An AB24/AB29 variant: "Based on your last N sessions" under the estimate, keeping the "+ tax where applicable" line.

**Privacy.** Estimates are per account only: never pooled across accounts and never public.

**Tests.** Estimator unit tests for 0, 1, 2 and N sessions, a model change and a price change; a pre-flight render test.

## Acceptance criteria

- [ ] n/a (outline). Nothing here is executable until the card is expanded.

Candidate acceptance to refine at expansion (not gates today):
- [ ] The results note for the fairness run is merged. It covers ≥ 10 speakers (≥ 1 disfluent), the pass rule, the verdict per (shell, brain model, `prompt@v`), and the injection script flagged. It contains no names and no audio.
- [ ] Voice Communication is proposed **only** for passed combinations. The kill switch and any combination change revert it to `self_only_voice`, and tests cover both.
- [ ] The never-list lint is green, and the evidence enum has no `audio`/`prosody` member.
- [ ] After ≥ 2 sessions, the pre-flight estimate uses the learner's own p50/p90. Below that, the catalog defaults apply.

## Release

**Outline only (v2.2).** This card merges nothing and cuts no tag.

Once expanded:
- **code:** "merge only (ships dark in the next `v2.1.x` patch)", with the ADR-0034 §6 checklist on each patch, including **"from M6: no live interviews"**;
- **the fairness run:** an owner event. The harness is throwaway and never committed, and the results land through a docs PR;
- **the flip:** the Communication default flip rides **whichever labelled minor follows a passed run**: the M6c GA tag sprint that [m6c-01](sprint-m6c-01.md) task 1 adds (e.g. `m6c-04`) if the run has passed by then, and a later 2.x minor otherwise. It never blocks M6c GA. That tag sprint copies the [ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) release checklist plus the ADR-0035 §2 NetworkPolicy standing rule. No new in-cluster caller is expected;
- **infra:** no infra PR. The kill switch is unset by default; setting it is an infra PR, and that is only the kill path.

## Definition of Done

**This card:**
- it exists with its sources linked;
- it is listed on the [`../status.md`](../status.md) sprint board as "⬜ outline";
- no code and no prompt.

**At expansion:**
- replaced by a full plan and `../prompts/prompt-m6c-02.md`;
- `ev-m6c-speakers` added to the owner events;
- the pass rule recorded;
- the kill switch entered in the flag inventory as a permanent kill switch, with no removal milestone;
- the board variants scheduled in the shared M6c design sprint, or the m5-01 screenshot precedent planned (the build PR's merge freezes the variant).

## Risks / watch-outs

- **Recruiting is the long pole.** Ten consenting speakers need owner time, and v2 is owner-only use (D35). The check may wait until v3 brings real demand. Meanwhile voice Communication stays self-scored, which is the safe default, and M6c GA ships without the flip.
- **Small sample.** Ten speakers can't certify fairness. ±1-band parity is a floor, not a proof. Keep `self_only_voice` as the documented fallback, keep the kill switch, and re-run on any change.
- **EU AI Act Art. 5(1)(f).** The check measures **band parity only**. It never uses accent or prosody as a feature and stores no demographic inference. Voice stays off for EU/EEA accounts ([ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits)).
- **Speaker privacy on a public repo.** Only scrubbed transcripts and synthetic ids get committed. Speaker audio exists only in the owner's throwaway OpenAI project, which is deleted afterwards, as in S6.
- **Silent provider drift.** A shell or model update can change ASR quality. That's why a pass is keyed on the exact combination, and a change reverts to self-scoring until re-run.
- **Estimates drift with prices.** Per-minute prices change. Key the history on the price-table version, and never mix prices in one percentile.
