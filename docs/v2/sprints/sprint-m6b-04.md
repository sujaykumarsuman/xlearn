# Sprint m6b-04 — Fake-media e2e + interviewer GA flip → v2.1.0

> **Milestone:** M6b — voice, one shell. This is the **release sprint**: the M6b exit plus the **interviewer GA** (M6a + M6b; D28, D32). · **Track:** product · **Order:** 85
> **Prereqs:**
> - [m6b-03](sprint-m6b-03.md): the M6b dark patch is live, with the `coach admin interviews media-audit` verb, m6a-01's owner-only `qa_slice` in the setup UI and the runbook (its owner pre-flight look may still be an open pending-smoke note: task 2's run closes it);
> - [m6a-06](sprint-m6a-06.md): the M6a dark patch is live and the M6a exit is recorded;
> - [m6a-01](sprint-m6a-01.md) / [m6a-03](sprint-m6a-03.md): the `interviewAudience` gate this sprint flips, the start rule, `qa_slice` and its 409 `qa_session_unscored`;
> - [mi-13](sprint-mi-13.md): MI-16.
>
> **Unblocks:**
> - [m6c-01](sprint-m6c-01.md), [m6c-02](sprint-m6c-02.md), [m6c-03](sprint-m6c-03.md): the v2.2 outline cards, which go to v2.2 planning;
> - [m5-01](sprint-m5-01.md)'s label: M5 rides `v2.1.0` only if it merged before this tag; otherwise it waits for ≥ `v2.2.0`.
>
> **Release action:** **tag `v2.1.0`**. This is the interviewer GA, the T-1 default flip that [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme) reserves the minor for, so the GA-tag snapshot applies (the owner takes it before launch; task 5). Two conditional extras: **only if** m6b-03's privacy-notice update isn't live yet (normally n/a: m6b-03 merges it before its tag, D40), a `v2.0.N` patch that makes it live before the run (task 1); **only if** S6 M7 failed, a `coach-interview` ImagePolicy bump as its own infra PR (task 6).
> **Calendar:** ≈ Q1 2027, once [m6b-03](sprint-m6b-03.md)'s patch is live. **Before launch, the owner** takes the Hostinger manual snapshot (`ev-snap-v2.1.0`, 1-day retention) and **is present for the session, about 45 minutes in one sitting** (the prompt's before-launch block, D40): the fake-media run (task 2, on his own key, ≈ $0.7–1.2) and the post-tag look (task 5, not a gate). Have the flip PR open and green before the run, so merge → tag follow it directly, on the snapshot's day.
> **Execute with:** [`../prompts/prompt-m6b-04.md`](../prompts/prompt-m6b-04.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Pre-run: checks (and, only if needed, the privacy-notice patch), looped synthetic candidate clip, throwaway Chrome profile | X + H | ⬜ |
| 2 | Fake-media run on prod against m6b-03's dark patch (the owner present, a before-launch item; on his key), to `proposed` → refused → abandoned | O + X | ⬜ |
| 3 | No media on the node, plus the cost reconciliation | H | ⬜ |
| 4 | GA flip PR: `interviewAudience` `"cohort"` → `"all"` (T-1); release notes; flag inventory | X | ⬜ |
| 5 | The owner's snapshot (`ev-snap-v2.1.0`, before launch), then tag `v2.1.0` and verify | O + X + H | ⬜ |
| 6 | **Only if S6 M7 failed:** the `coach-interview` ImagePolicy bump (its own PR in `../infra`) | I | ⬜ |
| 7 | Record: M6a and M6b exits, the tag row, flags, the run record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line to match. Mirror the sprint's state into [`../status.md`](../status.md): the Sprint board row, the Milestones rows M6a and M6b, the milestone → tag → floor → snapshot row, the flag inventory, owner event `ev-snap-v2.1.0`, and the pending-smoke notes (m6b-03's owner pre-flight look, closed by task 2).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] m6a-06 patch and m6b-03 patch (M6b dark) live on prod: `/xlearn/api/v1/healthz` reports m6b-03's `v2.0.N` or later
- [ ] Fake-media run passed against that patch (tasks 1–3 here) before the flip PR merges. This gate guards task 4's merge, not the start of this sprint.
- [ ] No live interviews at tag time (`coach admin interviews --live` empty)
- [ ] **The M6a exit is recorded** ([m6a-06](sprint-m6a-06.md)): a 45-minute text mock survived a pause and resume and scored once. Also green:
  - the twin fairness gate ([m6a-03](sprint-m6a-03.md));
  - the `store:false` acceptance test and the replay suite ([m6a-02](sprint-m6a-02.md)).
- [ ] **MI-16 done** ([mi-13](sprint-mi-13.md)), and [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) is Accepted ([ds-m6a-01](sprint-ds-m6a-01.md))
- [ ] **[m6b-03](sprint-m6b-03.md)'s preparation is live**, and its task 7 is ✅ (its patch verified; the owner's pre-flight look may still be an open pending-smoke note, which task 2's run closes):
  - `coach admin interviews media-audit`;
  - m6a-01's owner-only **`qa_slice`**, settable from the owner's setup UI ("QA slice (0.33×)");
  - `docs/runbooks/voice-fake-media-e2e.md`.
- [ ] **The privacy notice covers the interviewer** (m6b-03's check). Either it was already "covered", or m6b-03's notice update (merged as drafted, D40) is live. If an update is still unmerged or untagged, task 1 merges it (it lands as drafted) and makes it live with a patch before the run.
- [ ] **The owner's before-launch items** (the prompt's `## Before you launch (owner)` block; launching attests them, D40; the agent never sees or handles the key). If one turns out to be missing, the session lands what doesn't depend on it and records ⛔ in status.md; it never waits. He is present for the session (about 45 minutes, one sitting), and he has confirmed, himself:
  - his `interview` default key is an OpenAI key with `voice_shell`;
  - the key sits in a dedicated project with a **hard spend limit** leaving ≥ $4 of headroom: the run's $3.00 cap plus $1, [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)'s "keep available" rule (onboarding advice, [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path));
  - his `account.region` is outside the EU/EEA;
  - he has no non-terminal interview and **no starts today (UTC)**, so a failure leaves one retry (L19: 2 starts a day);
  - he took a Hostinger manual snapshot in hPanel right before launch (`ev-snap-v2.1.0`, 1-day retention, so the tag follows the same day), after checking that the last weekly image is ≤ 7 days old.
- [ ] **The run host** is the owner's Mac, with Chrome stable (the S6-tested major or newer) plus `say` and `afconvert`
- [ ] **Release line:** `.release-line` = `2`, and every `xlearn-*` ImagePolicy is `>=1.0.0 <3.0.0`, so `v2.1.0` needs no range change. No `v2.1.*` tag exists: `git ls-remote --tags origin 'refs/tags/v2.1*'` is empty.
- [ ] **Only if S6 M7 failed:** `coach-interview` is live on m6b-03's patch
- [ ] **[m5-01](sprint-m5-01.md)'s state is known.** If it merged untagged, it rides `v2.1.0` and the release notes list it. If it is still open, it doesn't merge until `v2.1.0` is tagged, and then takes ≥ `v2.2.0` ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)).
- [ ] **Parallel sessions:** check with `gh pr list`, `git ls-remote --tags origin`, `git worktree list` and ListAgents that:
  - no open peer PR touches the interviewer gates (the gateway's interview proxy, `web/src/screens/mock/**`, `internal/coach/interview`);
  - no peer is about to tag, a `v2.0.x` patch included (task 4 explains why).

## Goal

**Prove the M6b exit on production**, against [m6b-03](sprint-m6b-03.md)'s dark patch: a voice mock runs on the learner's key with **no media on the node**. In owner-only v2 (D35), "the learner's key" is the owner's own OpenAI key. The run follows the [t6 §9](../research/t6-realtime-interviewer.md#9-phased-plan) P1 QA recipe:
- a manual Chrome fake-media end-to-end run;
- a synthetic recorded clip;
- the 0.33× slice rail (m6a-01's owner-only `qa_slice`);
- ≈ $0.7–1.2 on the owner's key (0.33 × [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)'s 45-minute figure, plus the debrief and the voice check).

A `qa_slice` interview reaches `finished` and an AI proposal but is **never scored** (m6a-01; [m6a-03](sprint-m6a-03.md) answers submit with 409 `qa_session_unscored`). So the run ends at `proposed`: it checks the voice review up to the proposal and the refusal, and the owner abandons the session. Score-once is already proven by m6a-06's text exit and m6a-03's tests (its format matrix covers `voice`).

Then **flip the interviewer defaults on for every account (T-1)**, text and voice, and cut **`v2.1.0` — the interviewer GA** (M6a + M6b; [rollout §7](../rollout-plan.md#7-indicative-tag-timeline)).

The flip is invisible today. Every v2 account is the owner or a tester (ga-01's check), and both are already in the cohort. What changes is that the interviewer becomes a **default**, not a dogfood, ahead of the v3 opening.

## Scope

**In**
- Only if needed: m6b-03's privacy-notice update made live with a `v2.0.N` patch before the run, its PR merged first if still open (X + H).
- The fake-media run on prod, with the owner present, attested before launch (O + X).
- The no-media-on-node checks and the cost reconciliation (H).
- The T-1 flip PR, the release notes and the flag-inventory update (X).
- The owner's manual snapshot (before launch), then tagging `v2.1.0`, verifying it and recording it (O + X + H).
- Only if S6 M7 failed: the `coach-interview` ImagePolicy bump, its own infra PR (I).

**Out**
- **M6c extras** go to [m6c-01](sprint-m6c-01.md), [m6c-02](sprint-m6c-02.md) and [m6c-03](sprint-m6c-03.md) (outline only; expanded at v2.2 planning).
- **M5 `evaluator_only`** goes to [m5-01](sprint-m5-01.md), at ≥ `v2.2.0`, unless it merged before this sprint.
- **Scoring a voice mock on prod** is out: the QA slice can't be scored (m6a-01/m6a-03), and a full-length paid voice mock isn't worth the spend. Score-once rests on m6a-06's text exit and m6a-03's tests.
- **A coach restart during a live call on prod** is out. A hand `kubectl rollout` is forbidden. Make-before-break is covered by S6 M7 and m6b-02's compose tests.
- **The $-cap stop on prod** is out. It is covered by [m6b-02](sprint-m6b-02.md)'s tests, and L19 allows only 2 starts a day, so none is spent on it.
- **The opening** is v3 (D35). `SIGNUP_MODE` stays `closed` and no invite is minted.
- **Alerting, opscheck and interview counters** are out (D34).

## Tasks

### 1 · Pre-run: checks, synthetic candidate clip, throwaway Chrome [X + H]

Follow `docs/runbooks/voice-fake-media-e2e.md` ([m6b-03](sprint-m6b-03.md)). Fix any drift in the runbook in task 4's PR.

**Only if m6b-03's privacy-notice update isn't live yet** (normally n/a: m6b-03 merges it before its tag, D40). It is a code PR (the `/xlearn/privacy` page and the notice version, [l-05](sprint-l-05.md)), so it needs a tag to go live:
- if the PR is still open, merge it on CI green: it lands as drafted (D40), and the owner may revise it later with a content PR;
- if no tag carries it yet, tag the **next free `v2.0.N`** from `main` with the release checklist below (no live interviews; title `v2.0.N — privacy notice (interviewer)`), before the run and before the flip PR merges;
- every account, the owner included, then re-accepts the notice (AB19 F12, renotice) on next sign-in: the owner does that when he signs in for task 2;
- record the patch in status.md.

**Pre-checks (agent):**
- `ssh vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews --live'` is empty;
- healthz reports m6b-03's `v2.0.N` or later (or the notice patch);
- `host-verify --cluster` is green;
- the owner's key, project, hard limit, region and no-starts-today are before-launch items, attested by launching. Verify what the product shows once he signs in (task 2 step 1): the setup's `voice` block must show voice available. A `reason` there means an item is missing: handle it like a failed run (task 2), never a wait.

**The start rule** ([m6a-01](sprint-m6a-01.md) task 2): a start is `setup —start→ preflight`, and it counts toward the ≤ 2 starts a day. In m6b-03's pre-flight order it happens after consent and the cap, before the voice check. A failure before it costs nothing. A failed voice check that returns the interview to `setup` (`preflight_fail`) needs a second start; a failure after it means abandoning that interview (1 non-terminal interview, L19) and starting a new one. Either way there is **one retry today at most**. Write this into the runbook if m6b-03 didn't.

**The synthetic candidate clip.** Make **one cycle of about 3 minutes** that Chrome loops. Every `getUserMedia` (the mic check, the voice check, Start, Talk after Hold) restarts the fake file, so a track paced to the ≈ 15-minute slice rail ([t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) × 0.33: ≈ 1:39 Clarify · 1:39 Brute force · 2:38 Plan · 4:57 Code · 2:19 Trace + edges · 1:39 Complexity) can't hold its pacing. A loop also means the input never runs dry before the rail and the ≤ 3-minute debrief end.
- Mock-v2 picks the item server-side, so use **generic DSA utterances**: clarifying questions, a brute force with its cost, a hash-map plan, edge cases, complexity.
- Include one 45 s silence and a stretch of thinking aloud while typing.
- Build it with `say -v <voice> -r 180 -f candidate.txt -o candidate.aiff` (`[[slnc 45000]]` makes the silence), then `afconvert -f WAVE -d LEI16@48000 -c 1 candidate.aiff candidate.wav`.
- The voice is synthetic, so no one's real voice is used. The files stay in the session scratchpad and are **never committed**.

**Chrome.** Launch it on a throwaway profile:

```sh
open -na "Google Chrome" --args --user-data-dir="$SCRATCH/chrome-voice-e2e" --no-first-run \
  --remote-debugging-port=9333 \
  --use-fake-device-for-media-stream \
  --use-file-for-fake-audio-capture="$SCRATCH/candidate.wav" \
  https://projects.sujaykumar.dev/xlearn/
```

- `--remote-debugging-port` is what task 2's DevTools-protocol driving uses. Chrome binds it to 127.0.0.1; don't add `--remote-debugging-address`.
- No `%noloop` suffix: the clip loops.
- The fake device also supplies a synthetic camera for the self-view step.
- Leave the permission prompt real (no `--use-fake-ui-for-media-stream`), so the prompt's UX is exercised.
- Before Start, open `chrome://webrtc-internals` in a **separate window** placed beside the interview window, never over it. The interview tab must stay visible (`document.hidden` false): a hidden tab plus silence reaches m6a-01's idle interrupt.
- Record the Chrome version.

### 2 · Fake-media run on prod (owner present) [O + X]

This runs against **production on m6b-03's dark patch**, in the owner's cohort account. The owner's presence is a before-launch item (D40), as for S6: the agent prompts him at each of his steps. If he turns out not to be there when the run needs him, don't wait: record ⛔ in status.md ("fake-media run: owner not present"), close the flip PR unmerged (its branch stays), land any runbook fix in a docs PR, and a re-run is a fresh launch of the prompt (a new snapshot included).

**Who does what:**
- **The owner** signs in (the agent never types credentials), allows the mic, ticks every consent box himself, sets and confirms the $ cap, and clicks **Start**. He also clicks submit and abandon on the review screen.
- **The agent** drives the rest where it can: over the DevTools protocol on the throwaway profile (`--remote-debugging-port`, bound to 127.0.0.1), from a scratchpad-only script that is never committed. Where it can't, it reads each step out for the owner to click.
- **Either way, the agent records** every observation.

| # | Step | Pass / record |
|---|---|---|
| 1 | Mock-v2 → setup (AB29 F1): format **voice**, mode Standard, multiplier 1×, **QA slice (0.33×)** | the owner-only slice option shows; the interview is created in `setup` |
| 2 | Voice consent (AB29 F2): the owner ticks every box himself → [Continue] | [Continue] is enabled only once every box is ticked |
| 3 | Mic (F3–F4): the owner allows the mic prompt; the level meter moves with the fake clip | each frame as frozen |
| 4 | Estimate and cap (F6): the estimate with its GST line; the owner sets the cap to **$3.00** and confirms → the pre-flight starts (`setup → preflight`: the day's first start) | the cap is accepted; record the figures AB29 showed. $3.00 is ≈ 2.5× the expected spend, so the 85% wrap ($2.55) can't fire before the rail ends |
| 5 | Real voice check (F5, ≈ $0.02, a `purpose: "preflight"` segment) | passes; captions appear; the segment closes |
| 6 | Ready (F12): the owner clicks **Start interview** | record the `/segments` time from the DevTools Network panel (≤ 3 s: S6 M1's band) and the time to the first interviewer audio after Start |
| 7 | Live (AB30): the AI disclosure is spoken and captioned; captions arrive for both sides (the candidate's marked "approximate"); self-view on (synthetic camera), then off | the camera stops when toggled off |
| 8 | Paste a short function into the editor, then click **Run** once | the interviewer refers to the new code within a turn (D29: 2–3 s cadence); the Run echo shows on the HUD and in the interviewer's next turn |
| 9 | Switch to push-to-talk with AB30 F4's mid-call toggle (`voice-mode`), hold-to-talk for about 1 minute, then switch back to Standard | no model speech in reply to unheld audio; a reply ≤ 2.5 s after release (S6 M3b) |
| 10 | **Hold voice while I code** → 60 s → **Talk** | the clock keeps running while held; Talk opens a new segment (a free reseed) with a one-line welcome-back; time to first audio ≤ 3 s (S6 M5) |
| 11 | Let the rail run to wrap-up → the spoken debrief (≤ 3 min, captioned) → Finish | no score is spoken |
| 12 | **Check your transcript** (AB30 F17): the candidate lines are editable and the interviewer's aren't. **Make no edit**, so the proposal stays AI-eligible → [Continue to scores] | the screen renders as frozen; no `transcript_edited` caveat |
| 13 | Proposal (AB27): voice-mode Communication is `self_only_voice`, with the AI's quotes, and the other dimensions carry AI proposals. The owner submits → **409 `qa_session_unscored`** (if m6a-06's screen withholds submit for a QA session, record that instead). Then the owner **abandons** the interview | the interview is `abandoned`; Progress shows no new scored mock (the abandoned row is listed, never charted); the public profile's mock count is unchanged (D31) |
| 14 | In `chrome://webrtc-internals`, find the selected candidate pair | the remote address is an OpenAI address, **never the VPS's IP**, and there is no relay through our host. Save the dump to the scratchpad only (it holds IP addresses) |

**If any step fails: stop the run — no flip** (a gate failure, not a review). File the issue, fix it in a `v2.0.x` patch, and re-run: the start rule (task 1) leaves at most one retry today. If no retry is left, don't wait: close the flip PR unmerged, record ⛔ in status.md naming the issue (and the fix's patch, if it landed), and a re-run is a fresh launch of the prompt, with its before-launch items (a new snapshot included).

### 3 · No media on the node, plus the cost reconciliation [H]

Run these right after the run, in the same session, and paste the verdicts (not the raw output) into the run record.

**Database.** `ssh vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews media-audit <id>'` must report `clean`:
- no binary column outside the key envelope;
- 0 SDP markers;
- 0 base64 runs;
- the turns stored as text only.

**Logs.** Run this, and the same for `deploy/xlearn-gateway` (and `deploy/xlearn-coach-interview` if M7 failed). Each count must be `0`:

```sh
ssh vps 'sudo k3s kubectl logs -n xlearn deploy/xlearn-coach --since=90m' \
  | grep -cE 'a=candidate|a=fingerprint|(^|[^a-z])v=0|output_audio|input_audio|[A-Za-z0-9+/]{1024,}'
```

**Volumes.** Coach's pod spec must have no `persistentVolumeClaim` and must have `readOnlyRootFilesystem: true`. For each `emptyDir`, run `ssh vps "sudo find /var/lib/kubelet/pods/<coach-pod-uid>/volumes/kubernetes.io~empty-dir -type f -newermt '<run start>' -printf '%s %p\n'"`. It must list nothing, or only files you can name as non-media. Get the pod uid with `k3s kubectl get pod -n xlearn -l app.kubernetes.io/instance=xlearn-coach -o jsonpath='{.items[*].metadata.uid}'`.

**Network:**
- `ssh vps 'sudo ufw status'` allows TCP 22/80/443 only, so there's no UDP path.
- Together with task 2's step 14, that shows **no RTP touched the node**.
- Coach's only provider traffic is the outbound sideband WSS on 443. Its transient audio copies are dropped unparsed, which the consent discloses ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)).

**After the run:**
- `coach admin interviews --live` is empty and the sideband lease is released;
- the interview is `abandoned`, never `scored`;
- its `purge_at` is `ended_at + 30 d` (m6a-01's retention for an abandoned interview).

**Cost:**
- The media-audit µUSD sum for the interview must be within **±25%** of the OpenAI usage page for the owner's project over the run window (S6 M9's band). The owner reads the usage page.
- Total spend must be ≤ the $3.00 cap; expect ≈ $0.7–1.2. Record the gap to t6 §9's older "≈ $0.3 per run" figure.
- The voice minutes count toward the 75-minute daily cap.
- The provider page can lag. If spend is under the cap, the tag doesn't wait for it: reconcile within 24 h and record the result.

**Cleanup:** delete the throwaway profile, the clip and the webrtc dump. Only the run record is committed: numbers and verdicts, with no transcript and no IP addresses.

### 4 · GA flip PR [X]

Branch `feat/m6b-04-interviewer-ga`. The PR may open any time (open it before the run), but it **merges only after tasks 2–3 pass**. Merging deploys nothing, because deploys are tag-only.

**The T-1 flip.** It is one code default and the checks that read it:
- [m6a-01](sprint-m6a-01.md)'s **`interviewAudience = "cohort"` → `"all"`**, the gate on the gateway's `/api/interviews/*` proxy;
- [mi-13](sprint-mi-13.md)'s cohort gate on `POST /api/interviews/{id}/segments` (the same helper, or its own check: flip it too);
- any other `role ∈ {owner, tester}` check on interviewer routes in `internal/gateway` or `internal/coach/interview` (grep for them). **The owner-only `qa_slice` check stays.**

The web needs no flag of its own: m6a-05's `MockRoute` shows Mock-v2 on the gateway's 200 and v1 on its 404, so it follows the flip. That makes every account reach Mock-v2 with the text interviewer and voice: the dark-launch path's step 3. Removing the flag rows is step 4 ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)).

**What stays exactly as it is** (none of these is a cohort gate):
- `aud=coach`;
- the Chrome/Edge gate;
- the EU/EEA gate (coach's 403 `voice_unavailable_region`);
- the OpenAI `voice_shell` key requirement;
- voice consent at its current version;
- L19: 1 non-terminal interview, ≤ 2 starts a day, ≤ 3 live voice interviews platform-wide, ≤ 75 voice minutes a day, the mandatory $ cap, and the snapshot and Run rates;
- the **owner-only `qa_slice`** (a QA tool, never flipped);
- the classic self-scored mock ([t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes): unchanged).

**Tests:**
- in compose, an account with role `learner` (accepted, region `IN`) can create a text interview and reach the voice pre-flight;
- the same account with region `DE` still gets the notice and coach's 403;
- the same account can't set `qa_slice`;
- anonymous requests still get 401;
- the former cohort-only tests are inverted (the gateway's gate tests and `MockRoute.test.tsx`);
- the unit, integration and e2e suites are green;
- `sqlc diff` is clean (the flip is code-only, with no migration);
- the OpenAPI drift test is green (no shape change).

**Release notes** (in the PR, reused for the GitHub release). The behaviour-change list:
- the AI mock interviewer is on for every account: text on any `interview` key; voice on Chrome/Edge with an OpenAI key, outside the EU/EEA;
- the failsafes: a 5-minute top-up grace, a pause of at most 24 h, the resume brief, and `incomplete` after 24 h;
- an explicit accept before any score;
- transcripts deleted 30 days after scoring by default, with a 12-month opt-in;
- no audio or video ever stored;
- the L19 caps;
- the public profile shows the mock count only (D31, since M1b);
- M5, if [m5-01](sprint-m5-01.md) merged untagged before this PR.

**Status in the same PR:** remove the interviewer's T-3 cohort-gate row(s) (`interviewAudience`) from status.md's flag inventory (removal milestone `v2.1.0`). No new flag, and the kill-switch list is unchanged: the interviewer has no kill switch, and rollback is R-b or R-c.

**Merging makes `main` a GA commit.** From the flip merge until the `v2.1.0` tag, a `v2.0.x` patch cut from `main` would ship the GA flip under a patch label, which ADR-0034 §1.1 forbids ("the minor moves only at a GA flip"). So:
- before merging, tell the peer sessions (ListAgents, then a message to each) that `main` is about to carry the interviewer GA flip and that no `v2.0.x` may be cut from it;
- merge only when the tag can follow at once (task 5), on the day of the owner's before-launch snapshot;
- if a `v2.0.x` tag appears on or after the flip commit, stop and report. Never move or re-push a tag.

### 5 · The owner's snapshot (before launch), then tag `v2.1.0` and verify [O + X + H]

Before the tag, right after the flip merge:
- Run the release checklist below.
- Right before the tag, **`coach admin interviews --live` must be empty**.
- Confirm `git ls-remote --tags origin 'refs/tags/v2.1*'` is still empty, and that no `v2.0.x` tag points at or after the flip commit.
- Confirm the ranges are already `<3.0.0` (`ssh vps 'sudo k3s kubectl get imagepolicy -n flux-system -o yaml' | grep range`). There is **no range change**: the "none" row of [ADR-0034 §1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure).
- `host-verify --cluster` is green and the host has settled.
- **Snapshot (O, before launch; owner event `ev-snap-v2.1.0`).** `v2.1.0` is a GA flip, the labelled release ADR-0034 §1.1 reserves the minor for, and [ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule) and §6 say to take a Hostinger manual snapshot "right before any contract, erase or GA tag". So the owner took it in hPanel right before launching this prompt (a before-launch item, D40), after confirming there that the last weekly image is ≤ 7 days old. Launching attests it, and that attestation confirms the checklist's "the snapshot is taken" line. Record its time (the time he gave, else the launch time). It is kept for 1 day and is the R-d cover for the day, so the tag follows the same day; a re-run on a later day needs a fresh snapshot before its launch. If the owner said in the session that he waived it, record that as an **explicit owner decision** in status.md's decisions log. Don't reinterpret the rule.

Tag `v2.1.0` on the merged flip commit, straight after the merge (on the snapshot's day). The release title is `v2.1.0 — interviewer GA`, and the notes come from task 4.

After the tag, by looking (D34):
- healthz reports `v2.1.0`;
- the images are the new ones;
- every `xlearn-*` ImagePolicy's latest is `2.1.0`, and the HelmReleases are Ready. **Exception** (only if S6 M7 failed): `xlearn-coach-interview` is pinned to an exact version (m6b-02), so it lags until task 6's PR merges; re-check it then;
- smoke-test login, the dashboard and coach, and the owner's Mock-v2 look: the text/voice choice renders, and the voice pre-flight works up to the free mic check. He stops before `start` and abandons that `setup` interview. No paid session is needed, because task 2 proved the voice path. The owner is present (a before-launch item), so he does both right after the tag; the agent never signs in. If he can't, record them as a pending-smoke note in status.md and carry on: neither gates _Overall_ ✅;
- `coach admin interviews --live` is empty, and `media-audit` still runs.

**Rollback:**
- **R-b:** narrow every `xlearn-*` policy to `!=2.1.0`, which returns to the M6b dark patch. There's no migration in 2.1.0 and the floor is unchanged.
- **R-c:** revert the flip and tag `v2.1.1`.
- **R-d:** the snapshot, within its day ([ADR-0034 §4.2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button)); it loses every write since.
- **R-a:** not available. The interviewer has no kill switch, and ADR-0034 §2's permanent list doesn't include one.

### 6 · Only if S6 M7 failed: the `coach-interview` ImagePolicy bump [I]

If S6 recorded M7 pass, mark this ✅ "n/a — M7 passed". Otherwise, after the tag and with `coach admin interviews --live` empty, open **its own PR in `../infra`** that moves `xlearn-coach-interview`'s exact pin to `2.1.0`, per m6b-02's runbook (`docs/runbooks/interviewer.md` § "coach-interview (M7 fallback)"). Without it, the interview role keeps the pre-flip code. Merge it when its checks are green; Flux rolls `xlearn-coach-interview`. Verify with `k3s kubectl get deploy -n xlearn` and re-check that policy's latest. GitOps only: never `kubectl apply`.

### 7 · Record [X]

In a docs PR after the tag, update `docs/v2/status.md`:
- the Sprint board;
- **Milestones: M6a ✅** (m6a-06's exit evidence, labelled in `v2.1.0`) and **M6b ✅**;
- **milestone → tag → floor → snapshot:** `M6a + M6b (interviewer GA) → v2.1.0 → floor unchanged → snapshot <hPanel time, taken before launch>` (or "waived by the owner", with the decisions-log entry);
- owner event `ev-snap-v2.1.0` ✅ (with the snapshot time, or "waived");
- the pending-smoke notes: m6b-03's owner pre-flight look ✅ (task 2 walked the same pre-flight), and the post-tag look if it is still open (task 5);
- the privacy-notice patch, if task 1 cut one;
- the flag inventory, per task 4;
- the **run record:**
  - date, Chrome version, shell;
  - interview id, `qa_slice`, duration, the end state (`abandoned` after 409 `qa_session_unscored`);
  - the `/segments` time and the time to first audio;
  - the PTT reply time and the reseed time to first audio;
  - the media-audit verdict, the log-grep and volume results, the WebRTC pair verdict;
  - cost vs cap vs the provider's figure;
  - any issues filed;
- the **decisions log:** the GA flip, the snapshot (or its waiver), the rollback plan, anything learned;
- the rollout §7 `v2.1.0` row marked done;
- the "next" line: v2.2 planning expands [m6c-01](sprint-m6c-01.md)…[m6c-03](sprint-m6c-03.md), and [m5-01](sprint-m5-01.md) goes to ≥ `v2.2.0` unless it rode `v2.1.0`.

Also add "shipped in `v2.1.0` (date)" to [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md)'s status line and to the [PRD](../../prd/xlearn-v2-prd.md) V6 row.

## Acceptance criteria

- [ ] **M6b exit.** A voice mock ran end to end on production, against m6b-03's dark patch, on the owner's own OpenAI key and the owner-only `qa_slice`: pre-flight (consent and cap before the voice check) → live → Run echo → push-to-talk → Hold/Talk reseed → debrief → transcript check → AI proposal (voice Communication `self_only_voice`, with quotes) → submit refused with 409 `qa_session_unscored` → `abandoned`, with no scored mock added. There was **no media on the node**:
  - media-audit reports `clean`;
  - the log greps return 0;
  - coach's volumes hold no media files;
  - the WebRTC pair was browser ↔ OpenAI only.
- [ ] Spend ≤ the $3.00 cap, and the segment-log cost is within ±25% of the provider's usage (reconciled within 24 h)
- [ ] The GA flip PR (`interviewAudience` → `"all"`, plus mi-13's `/segments` gate) merged only after the run. In compose, a `learner`-role account gets the interviewer, and the browser, EU/EEA, key, consent, L19, $-cap and `qa_slice` gates are unchanged. The cohort flag rows are removed.
- [ ] **`v2.1.0` verified:** tagged with the release checklist (no live interviews, no range change, the owner's before-launch snapshot attested for the tag's day, or its waiver recorded), no `v2.0.x` cut from the flip commit, deployed by Flux, healthz, images, policies and HelmReleases checked, smoke test green (or recorded as a pending-smoke note if the owner couldn't run it)
- [ ] status.md shows M6a ✅, M6b ✅, the `v2.1.0` row with its snapshot, and the run record

## Release

**Tag `v2.1.0` (interviewer GA).** [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme) says: "From `v2.0.0` on, the minor moves only at a GA flip". This is that flip.
- Never cut `v2.1.0` for anything else, M5 included.
- Never cut a `v2.0.x` from the flip commit or after it.
- Everything after it is a `v2.1.x` patch until the next GA flip.

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist), verbatim, plus the ADR-0035 §2 standing rule):

- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

**For this tag:**
- **Version:** `v2.1.0` exactly. It's the next free minor, reserved by ADR-0034 §1.1, and its major `2` equals `.release-line`.
- **ACL PRs:** n/a, no stream or consumer.
- **New service:** n/a.
- **Contract or erase:** n/a.
- **GA tag: yes.** `v2.1.0` is the interviewer GA flip, so the line applies in full: `host-verify --cluster` green, the host settled, and the owner's manual snapshot taken before launch, on the tag's day (task 5; a before-launch item, D40). There is no range change (ADR-0034 §1.4's "none" row), so no infra PR precedes it.
- **Live interviews:** none, checked right before the tag.
- **Standing rule:** no new caller.
- **Flags:** the interviewer's cohort-gate rows (`interviewAudience`) are removed.
- **Policies:** if S6 M7 failed, `xlearn-coach-interview` is excluded from the "latest equals the tag" line until task 6's PR merges.

The interviewer is the M6 milestone, so the record is **M6a + M6b → `v2.1.0` → floor unchanged → snapshot**. Rollback is R-b to the M6b dark patch, R-c revert + `v2.1.1`, or R-d within the snapshot's day.

## Definition of Done

- The fake-media run passed and was recorded, with no media on the node.
- CI is green on the flip PR.
- The owner's snapshot was taken before launch, on the tag's day (or its waiver recorded as his decision).
- `v2.1.0` is tagged after the checklist, deployed by Flux (no hand `kubectl`) and verified live.
- If S6 M7 failed, task 6's infra PR is merged and `xlearn-coach-interview` runs `2.1.0`.
- The M6a and M6b exits are recorded.
- The flag inventory is updated.
- Statuses are updated (this file and [`../status.md`](../status.md)).
- The ADR-0032 and PRD shipped lines are added.
- Local `main` is synced in every repo touched.

## Risks / watch-outs

- **Provider spend on the owner's key.** Expect ≈ $0.7–1.2 (0.33 × t6 §8's 45-minute figure, plus the debrief and the voice check), not t6 §9's "≈ $0.3". It is capped by the mandatory $ cap ($3.00 for the run) and his project's hard limit. The agent never starts a voice session without the owner present and consenting.
- **A cap set too low changes the run.** At $1.50 on GPT-Live, the 85% wrap ($1.275) could fire before the rail ends and skip steps 10–11. Keep the cap at ≈ 2.5× the expected spend.
- **Fake audio isn't a conversation.** The synthetic utterances won't match the interviewer's questions. The run proves the media path, captions, the director following the editor, the Run echo, PTT, Hold/Talk, the finish, the transcript check and the proposal. It doesn't prove interview quality: S6 M14 (naturalness) and m6a-02's replay suite cover that.
- **The QA session is never scored**, by design (m6a-01's `qa_slice`). The run ends at `proposed` with a 409 and an abandon; score-once rests on m6a-06's text exit and m6a-03's tests. Don't try to score it another way.
- **L19's 2 starts a day** leave a single retry. m6a-01 counts a start at `setup → preflight`, which comes after consent and the cap and before the voice check. Launch only on a day with no starts yet (a before-launch item).
- **The fake clip restarts on every `getUserMedia`**, and a hidden interview tab plus silence reaches the idle interrupt. Loop the clip, and keep webrtc-internals in its own window beside the interview.
- **Flipping before the run defeats the point.** The flip PR merges only after tasks 2–3; the entry gate says so.
- **A peer tags `v2.1.0` first** (for example M5 on its own). ADR-0034 §1.1 forbids it. Check right before tagging; if it happened, stop and report. Never move or re-push a tag.
- **A peer cuts a `v2.0.x` patch after the flip merge.** It would ship the GA flip under a patch label. Announce the merge to the peers first, tag `v2.1.0` straight after it, and if a `v2.0.x` appears on or after the flip commit, stop and report.
- **Forgetting the snapshot** removes R-d for the GA tag (ADR-0034 §4.3). It is the owner's hPanel step, right before launch (a before-launch item), on the tag's day.
- **Tagging mid-interview rolls coach** (and `coach-interview`, if M7 failed). Check `--live` right before the tag.
- **The flip is invisible in v2**, because every account is in the cohort. Prove it in compose with a `learner`-role account, not on prod.
- **Chrome flags drift between versions** (fake-device looping, `--remote-debugging-port` rules). The runbook records the Chrome version used.
- **The throwaway profile holds a session cookie, and the webrtc dump holds IP addresses.** Delete both; never commit them.
- **If S6 M7 failed**, `coach-interview`'s exact pin moves only by task 6's infra PR. Forgetting it leaves the interview role on the pre-flip code.
