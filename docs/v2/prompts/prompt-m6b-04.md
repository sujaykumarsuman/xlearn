# Prompt — Sprint m6b-04 · Fake-media e2e + interviewer GA flip → v2.1.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root. **The owner must be present for about 45 minutes, in one sitting** (the run, the snapshot right before the tag, the post-tag look).
> **Plan:** [`../sprints/sprint-m6b-04.md`](../sprints/sprint-m6b-04.md)   ·   **Milestone:** M6b (release sprint: the M6b exit + interviewer GA, M6a + M6b)   ·   **Prereqs:** [m6b-03](../sprints/sprint-m6b-03.md), [m6a-06](../sprints/sprint-m6a-06.md), [mi-13](../sprints/sprint-mi-13.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions, GitOps, land-and-sync.
- **The plan:** [`../sprints/sprint-m6b-04.md`](../sprints/sprint-m6b-04.md). The run's step table, the no-media checks, the flip's keep-list, the snapshot step and the release checklist are there.
- **The runbook** `docs/runbooks/voice-fake-media-e2e.md` ([m6b-03](../sprints/sprint-m6b-03.md) wrote it), and m6b-03's PR, for the media-audit verb, the `qa_slice` toggle and the pre-flight order.
- **The M6a sprints this run and the flip rest on** (and their merged PRs, which win where they differ):
  - [m6a-01](../sprints/sprint-m6a-01.md): the gate this sprint flips, **`interviewAudience = "cohort"` → `"all"`**; the start rule (`setup —start→ preflight`, ≤ 2 a day); `qa_slice` (owner-only, never scored); retention (`abandoned` → `ended_at + 30 d`);
  - [m6a-03](../sprints/sprint-m6a-03.md): `ScoreMock` for `format=voice`, `self_only_voice`, 409 `qa_session_unscored`;
  - [m6a-05](../sprints/sprint-m6a-05.md): `MockRoute` (200 → Mock-v2, 404 → v1), the routes and `MockRoute.test.tsx`.
- **Research, t6:**
  - [§3](../research/t6-realtime-interviewer.md#3-architecture--media-path): the media path, the sideband's transient audio copies, the abuse caps and the onboarding advice.
  - [§4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes): the FSM, starts and non-terminal states.
  - [§5](../research/t6-realtime-interviewer.md#5-the-coding-round): the rail and the 0.33× slice.
  - [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys): the 45-minute cost figures the run's ≈ $0.7–1.2 comes from.
  - [§9](../research/t6-realtime-interviewer.md#9-phased-plan): P1, the fake-media run.
  - [§10](../research/t6-realtime-interviewer.md#10-the-smallest-spike): the S6 thresholds M1, M3b, M5 and M9 used as reference bands.
  - [§14](../research/t6-realtime-interviewer.md#14-risks).
- **ADRs:**
  - [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) (§2 no media on the node; §6 caps, consent, availability);
  - [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md): §1.1 (labels: `v2.1.0` = interviewer GA; the minor moves only at a GA flip), §1.4 (no range change), §2 (T-1 flip, flag removal), §4.1–§4.2 (rollback, R-d), **§4.3 (snapshot before any GA tag)**, **§6 (release checklist)**;
  - [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L19).
- **Rollout:** [`../rollout-plan.md`](../rollout-plan.md) §3 (M6b exit), [§7](../rollout-plan.md#7-indicative-tag-timeline) (`v2.1.0`), [§8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading) (what v2.1.0 contains). The [feasibility decisions log](../feasibility.md#decisions-log-newest-first): D28, D29, D32, D34, D35.
- **Code:**
  - the interviewer cohort gate: `interviewAudience` in `internal/gateway` (the `/api/interviews/*` proxy) and mi-13's `/segments` gate; any other `role ∈ {owner, tester}` check on interviewer routes (grep for them; the owner-only `qa_slice` check stays);
  - `web/src/screens/mock/MockRoute*` (no web flag: it follows the gateway);
  - `.release-line`; `.github/workflows/deploy.yml`;
  - coach's admin CLI (`interviews --live`, `interviews media-audit`).
- **Infra (read only, unless S6 M7 failed):** `../infra/apps/image-automation.yaml` (the ranges, and `xlearn-coach-interview`'s exact pin), `../infra/apps/xlearn-coach.yaml`.

## Context

- **What is live.** [m6a-06](../sprints/sprint-m6a-06.md) shipped M6a, the text interviewer, dark in a `v2.0.x` patch and recorded its exit. [m6b-03](../sprints/sprint-m6b-03.md) shipped M6b, voice, dark in the next patch: [m6b-01](../sprints/sprint-m6b-01.md)'s broker and sideband, [m6b-02](../sprints/sprint-m6b-02.md)'s robustness and caps, and the AB29/AB30 UI. Both sit behind the interviewer's T-3 cohort gate (`interviewAudience = "cohort"`).
- **Voice is unproven on prod.** Only the owner's free pre-flight look (m6b-03 task 7) has touched it. This sprint runs it first, with the owner present:
  - a **manual Chrome fake-media run** on the owner's own OpenAI key, which is "the learner's key" in owner-only v2 (D35);
  - a looped synthetic candidate clip and m6a-01's owner-only **`qa_slice`** (0.33× rail), ≈ $0.7–1.2 under a $3.00 cap;
  - the run ends at `proposed`: a `qa_slice` interview is never scored (409 `qa_session_unscored`), so the owner abandons it. Score-once rests on m6a-06's text exit and m6a-03's tests;
  - proof on the node that **no media was stored or relayed**: the M6b exit.
- **Only then** does the interviewer's **T-1 default flip** merge, turning text and voice on for every account, and **`v2.1.0` — interviewer GA** is tagged, right after the owner's manual snapshot (a GA tag, ADR-0034 §4.3).
- **No range change.** `.release-line` is already `2` and the ranges are `<3.0.0`.
- **The flip is invisible today**, since every v2 account is the owner or a tester. Prove it in compose with a `learner`-role account.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] m6a-06 patch and m6b-03 patch (M6b dark) live on prod (healthz ≥ m6b-03's `v2.0.N`)
- [ ] Fake-media run passed against that patch (tasks 1–3) before the flip PR merges. This guards the merge in step 7.
- [ ] No live interviews at tag time
- [ ] The M6a exit is recorded; the twin fairness gate (m6a-03), `store:false` and the replay suite (m6a-02) are green
- [ ] MI-16 done ([mi-13](../sprints/sprint-mi-13.md)); ADR-0032 Accepted
- [ ] `coach admin interviews media-audit`, the owner's `qa_slice` toggle and `docs/runbooks/voice-fake-media-e2e.md` are live, and m6b-03's task 7 (the owner's pre-flight look) is ✅
- [ ] The privacy notice covers the interviewer: "covered", or the owner has merged the notice PR (step 1 tags a patch if no tag carries it). Still open → stop; the owner decides
- [ ] The owner is booked (≈ 45 min, one sitting) and confirms, himself:
  - an OpenAI `interview` key with `voice_shell`;
  - in a dedicated project with a **hard spend limit** leaving ≥ $4 of headroom (the $3.00 cap + $1);
  - a non-EU/EEA region;
  - no non-terminal interview and **no starts today (UTC)**;
  - he can take a Hostinger manual snapshot right before the tag; the last weekly image is ≤ 7 days old.

  **You never see or handle the key.**
- [ ] The run host is the owner's Mac, with Chrome stable (≥ the S6-tested major), `say` and `afconvert`
- [ ] `.release-line` = `2`; every `xlearn-*` range `>=1.0.0 <3.0.0`; `git ls-remote --tags origin 'refs/tags/v2.1*'` is empty
- [ ] Only if S6 M7 failed: `coach-interview` is live on m6b-03's patch
- [ ] [m5-01](../sprints/sprint-m5-01.md)'s state is known: merged untagged → it rides `v2.1.0`; open → it waits until after the tag (≥ `v2.2.0`)
- [ ] No open peer PR on the interviewer gates, and no peer about to tag, a `v2.0.x` patch included (`gh pr list`, `git ls-remote --tags origin`, `git worktree list`, ListAgents)

## Do this (in order)

1. **(X + H) Pre-run** (plan task 1):
   - **Only if the owner merged a privacy-notice PR that no tag carries yet:** tag the next free `v2.0.N` with the release checklist (no live interviews), verify it, and have the owner re-accept the notice before the run. Record it.
   - `coach admin interviews --live` is empty; healthz reports m6b-03's patch or later; `host-verify --cluster` is green; the owner confirms his key, project, limit and no starts today.
   - Know the start rule: a start is `setup → preflight` (after consent and the cap, before the voice check), ≤ 2 a day, so there is one retry at most.
   - Build a **≈ 3-minute looped clip** in the scratchpad: generic DSA utterances, one 45 s silence and a stretch of thinking aloud, via `say … [[slnc 45000]] …` → `afconvert -f WAVE -d LEI16@48000 -c 1`. Every `getUserMedia` restarts it, so don't pace it to the rail. Never commit it.
   - Launch Chrome on a throwaway `--user-data-dir` with `--remote-debugging-port=9333 --use-fake-device-for-media-stream --use-file-for-fake-audio-capture=<clip>.wav` (no `%noloop`; keep the real permission prompt). Open `chrome://webrtc-internals` in a **separate window** beside the interview window, so the interview tab stays visible. Record the Chrome version.
2. **(X) Open the flip PR early** (step 5's content), so it is green before the owner arrives. Don't merge it.
3. **(O + X) Fake-media run on prod** (plan task 2 and its step table):
   - The **owner** signs in (**you never type credentials**), ticks every consent box himself, allows the mic, sets and confirms the **$3.00** cap, clicks Start, and submits and abandons on the review screen.
   - **You** drive the rest over the DevTools protocol (a scratchpad-only script against the loopback-bound debugging port; never committed), or read the steps out for the owner to click.
   - The steps: setup (voice, QA slice, Standard, 1×) → consent → mic permission and meter → estimate and cap → `start` → the real voice check → Start interview → the AI disclosure and captions → self-view on/off → paste code → Run → PTT via the F4 toggle (≈ 1 min) and back → Hold 60 s → Talk → wrap-up and the spoken debrief → Finish → **Check your transcript (no edits)** → proposal (voice Communication `self_only_voice` with quotes) → submit → **409 `qa_session_unscored`** → the owner abandons → the webrtc-internals selected pair.
   - Record every measurement in the plan's table.
   - **Any failure → stop**: no flip; file it; fix it in a `v2.0.x` patch; re-run (one retry today at most).
4. **(H) No media on the node** (plan task 3):
   - `coach admin interviews media-audit <id>` reports `clean`;
   - the coach and gateway (and `coach-interview`) log greps for SDP, audio-event and base64 markers since the run start return `0`;
   - coach's pod has no PVC and a read-only rootfs, and `find` finds no new files in its `emptyDir` volumes since the run start;
   - `ufw` allows TCP only, and the WebRTC pair was browser ↔ OpenAI, never the VPS IP;
   - `--live` is empty; the interview is `abandoned` (never `scored`); its `purge_at` is `ended_at + 30 d`.
   - **Cost:** the segment-log µUSD is within ±25% of the owner's OpenAI usage page (he reads it; reconcile within 24 h if it lags) and ≤ the $3.00 cap.
   - Then delete the profile, the clip and the dump.
5. **(X) The T-1 flip** (plan task 4), on branch `feat/m6b-04-interviewer-ga`:
   - Flip m6a-01's **`interviewAudience` from `"cohort"` to `"all"`**, mi-13's `/segments` cohort gate with it, and any other owner/tester check on interviewer routes. The web follows through `MockRoute`.
   - **Keep:** `aud=coach`, the Chrome/Edge gate, coach's EU/EEA 403, the OpenAI `voice_shell` requirement, voice consent, L19 and the $ cap, the owner-only `qa_slice`, and the classic mock.
   - Tests:
     - a compose `learner` account (region `IN`) can create a text interview and reach the voice pre-flight;
     - with region `DE` it still gets the notice and the 403;
     - it can't set `qa_slice`;
     - anonymous → 401;
     - the cohort-only tests inverted (the gateway's and `MockRoute.test.tsx`);
     - all suites green; `sqlc diff` clean (no migration); OpenAPI drift green.
   - Write the release notes' behaviour-change list (plus M5 if it rode).
   - Remove the interviewer cohort-gate row(s) from status.md's flag inventory.
   - Fix any runbook drift found in steps 1–4.
6. **(X) CI.** `go test ./...`, `go vet`, lint, the web suite, `sqlc diff`, the e2e lane. Everything must be green.
7. **(X) Merge the flip PR, only now**, with the owner still present. First tell the peer sessions (ListAgents, then a message to each) that `main` is about to carry the interviewer GA flip and that no `v2.0.x` may be cut from it. Merging deploys nothing.
8. **(O + X) Snapshot, then tag `v2.1.0`** straight after the merge:
   - Run the release checklist in the plan's **Release** section: the parallel-sessions check; `v2.1*` still free and no `v2.0.x` on or after the flip commit; major = `.release-line`; ranges already `<3.0.0` (no range change); `host-verify --cluster` green and the host settled; **`coach admin interviews --live` empty right before the tag**.
   - **The owner takes the Hostinger manual snapshot** (owner event `ev-snap-v2.1.0`; a GA tag, ADR-0034 §4.3), after checking the last weekly image is ≤ 7 days old. Record its time and mark the event ✅. If he waives it, record that as his explicit decision in the decisions log.
   - Tag the merge commit `v2.1.0`, with the release title `v2.1.0 — interviewer GA` and step 5's notes.
9. **(H + O) Verify:**
   - healthz reports `v2.1.0`; the images; every `xlearn-*` ImagePolicy's latest = `2.1.0` (except `xlearn-coach-interview` if M7 failed, until step 10); HelmReleases Ready;
   - smoke-test login, dashboard and coach;
   - the owner opens Mock-v2: the text/voice choice renders and the voice pre-flight works up to the free mic check; he stops before `start` and abandons that `setup` interview (no paid session);
   - `--live` is empty; `media-audit` runs.
10. **(I) Only if S6 M7 failed:** with no live interviews, open **its own `../infra` PR** moving `xlearn-coach-interview`'s exact pin to `2.1.0`, merge it once green, and re-check that policy and the Deployment.
11. **(X) Record** (plan task 7), in a docs PR:
    - status.md: M6a ✅, M6b ✅; `M6a + M6b → v2.1.0 → floor unchanged → snapshot <time>` (or the owner's waiver); owner event `ev-snap-v2.1.0` ✅; the notice patch if step 1 cut one; the flag inventory; the run record (numbers and verdicts only, the end state `abandoned` after the 409); the decisions log; the rollout §7 row; next steps (v2.2 planning expands m6c-01…03; M5 ≥ `v2.2.0` unless it rode).
    - The "shipped in `v2.1.0` (date)" lines in ADR-0032's status and PRD V6.

## Constraints

- **The owner's key and consent are his:**
  - never type credentials, never handle or display the key, and never tick consent boxes;
  - never start a voice session without the owner present and agreeing;
  - the $3.00 cap and his project's hard limit bound the spend.
- **The QA session is never scored.** Don't work around the 409; the owner abandons it.
- **Nothing from the run is committed** except the record's numbers and verdicts. The clip, the Chrome profile, webrtc dumps (IP addresses), transcripts and screenshots of the transcript stay in the scratchpad and are deleted.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the flip changes defaults only. No new service, route, caller, migration or event. Coach keeps the key and every provider call.
- **Gates that are not the cohort stay:**
  - browser, EU/EEA, `voice_shell`, consent, L19, $ cap, `aud=coach`;
  - `qa_slice` stays owner-only;
  - `SIGNUP_MODE` stays `closed`, and no invite is minted: the opening is v3 (D35).
- **Release discipline:**
  - `v2.1.0` is reserved for this flip (ADR-0034 §1.1): never cut it for anything else, never cut a `v2.0.x` from the flip commit, and everything after it is `v2.1.x`;
  - the owner's snapshot right before the tag (a GA tag, ADR-0034 §4.3 and §6), or his recorded waiver;
  - **no live interviews** at merge-to-tag time;
  - check peers' tags, PRs, worktrees and ListAgents before tagging or claiming an ADR number;
  - never move or re-push a tag;
  - infra PRs (the conditional `coach-interview` bump only) are their own tasks, never folded into the tag.
- **GitOps only:** no hand `kubectl apply`/`edit`/`rollout`. That is why a mid-call restart isn't tested on prod. `ssh vps` is for reads, `find`, `ufw status` and `host-verify.sh`; `kubectl exec` only runs `coach admin`; `kubectl logs` is a read.
- **D34:** no alerting, opscheck, interview counters, Flux Alert or push channel. The verification is by looking.
- **Memory-sum rule:** no new pod, so nothing to re-check beyond `host-verify --cluster` green.
- **goose/sqlc/outbox:** unchanged (`sqlc diff` clean). Consumers before producers and ACL PRs: n/a.

## Deliverables

- Only if needed: the privacy-notice `v2.0.N` patch, tagged and verified before the run.
- The fake-media run on prod, recorded: the step results, timings, the transcript check, the 409 and the abandon, the media-audit verdict, the log/volume/network verdicts, and cost vs cap vs the provider's figure.
- The GA flip PR (`interviewAudience` → `"all"` and mi-13's gate; tests; release notes; flag-inventory removal; runbook fixes), merged after the run.
- The owner's snapshot, then tag `v2.1.0 — interviewer GA`, deployed and verified. The conditional `coach-interview` ImagePolicy PR only if M7 failed.
- A docs PR: the status.md records and the ADR-0032 and PRD shipped lines.

## Update status

- This sprint's Status table in [`../sprints/sprint-m6b-04.md`](../sprints/sprint-m6b-04.md) (🔄 / ✅ / ⛔ per task; _Overall_).
- [`../status.md`](../status.md):
  - the **Sprint board** row;
  - the **Milestones** rows **M6a ✅** and **M6b ✅** (labelled `v2.1.0`);
  - **milestone → tag → floor → snapshot:** `M6a + M6b (interviewer GA) → v2.1.0 → floor unchanged → snapshot <time>` (or the owner's recorded waiver);
  - the **flag inventory:** the interviewer cohort-gate rows (`interviewAudience`) removed;
  - the **run record** (date, Chrome, shell, interview id, `qa_slice`, timings, verdicts, the end state, cost);
  - the **decisions log:** the flip, the snapshot or its waiver, rollback R-b/R-c/R-d, lessons;
  - the rollout §7 `v2.1.0` row → done;
  - "next": v2.2 planning for [m6c-01](../sprints/sprint-m6c-01.md)…[m6c-03](../sprints/sprint-m6c-03.md), and [m5-01](../sprints/sprint-m5-01.md)'s label.
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md)'s status line and the [PRD](../../prd/xlearn-v2-prd.md) V6 row: "shipped in `v2.1.0` (date)". No new ADR expected; claim a number only after the parallel-sessions check.

## Done when (acceptance)

- [ ] **M6b exit.** A voice mock ran end to end on production, against m6b-03's dark patch, on the owner's own OpenAI key and the owner-only `qa_slice`: pre-flight (consent and cap before the voice check) → live → Run echo → push-to-talk → Hold/Talk reseed → debrief → transcript check → AI proposal (voice Communication `self_only_voice`, with quotes) → submit refused with 409 `qa_session_unscored` → `abandoned`, with no scored mock added. There was **no media on the node**:
  - media-audit reports `clean`;
  - the log greps return 0;
  - coach's volumes hold no media files;
  - the WebRTC pair was browser ↔ OpenAI only.
- [ ] Spend ≤ the $3.00 cap, and the segment-log cost is within ±25% of the provider's usage (reconciled within 24 h)
- [ ] The GA flip PR (`interviewAudience` → `"all"`, plus mi-13's `/segments` gate) merged only after the run. In compose, a `learner`-role account gets the interviewer, and the browser, EU/EEA, key, consent, L19, $-cap and `qa_slice` gates are unchanged. The cohort flag rows are removed.
- [ ] **`v2.1.0` verified:** tagged with the release checklist (no live interviews, no range change, the owner's snapshot taken right before it or its waiver recorded), no `v2.0.x` cut from the flip commit, deployed by Flux, healthz, images, policies and HelmReleases checked, smoke test green
- [ ] status.md shows M6a ✅, M6b ✅, the `v2.1.0` row with its snapshot, and the run record

Ship at session end per AGENT.md land-and-sync, with **this sprint's release action: tag `v2.1.0`**:
1. Merge the flip PR only after the fake-media run passed, with CI green and the peers told.
2. The owner takes the snapshot; with `coach admin interviews --live` empty, tag `v2.1.0` after the release checklist.
3. Let Flux deploy, then verify live.
4. Only if S6 M7 failed: merge the `coach-interview` ImagePolicy bump in `../infra` (its own PR).
5. Merge the record docs PR.
6. `git checkout main && git pull` in every repo touched.
