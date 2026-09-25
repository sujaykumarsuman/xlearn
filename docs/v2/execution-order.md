# xLearn v2 — prompt execution order

> **One page to see what runs when, and what each prompt brings.** The 89 sprint prompts are clubbed into
> **19 workstreams** in **7 families**, so work that is split across several prompts reads as one thing.
> Order numbers match [build-plan.md](build-plan.md) (the static plan); live state is in [status.md](status.md);
> each prompt is self-contained in [`prompts/`](prompts/). Dates are **indicative** (BP4 calendar, 2026-09-25).

**How to read it**

- **D41: the four spikes run first** (W0, Sep 25–26), before prompt 1, so every design yes/no is answered before
  the build starts; their order numbers are unchanged.
- **Run prompts in numbered order.** Inside a week, prompts in different **lanes** (infra · product · design ·
  content/spike) can run as parallel sessions (up to about three), as long as their entry gates pass. Check
  peers' open PRs first; parallel prompts in the same service collide on migrations.
- **Every prompt ends shipped** (D40): land-and-sync, owner approval pre-granted. A **tag** prompt deploys; a
  merge-only prompt ships dark in the next tag. Owner-only steps are listed under each prompt's
  `## Before you launch (owner)`.
- **Legend** (diagrams): blue = build · amber = infra · violet = design boards · pink dashed = spike (throwaway)
  · green pill = a release tag · grey dashed = shared with another workstream / v2.2 outline.

## 1. The big picture

Seven families of work, in the order they land. Arrows are the hand-offs that matter; design boards feed every
family that has screens, always one milestone ahead.

```mermaid
flowchart TB
    F["<b>🧱 Foundations</b><br/>18 prompts · Sep 25 – Oct 30<br/>→ v1.8.0"]
    E["<b>⚙️ Learning engine</b><br/>5 prompts · late Oct – Nov<br/>→ v1.10.0"]
    J["<b>⚖️ Judge</b><br/>24 prompts · Oct – Dec<br/>→ v1.15.0"]
    A["<b>✨ Platform AI</b><br/>9 prompts · Oct spike · Dec build<br/>→ v1.16.0"]
    G["<b>🚪 Learner gate & GA</b><br/>7 prompts · Nov – Jan<br/>→ v2.0.0"]
    I["<b>🎙️ Interviewer (v2.1)</b><br/>12 prompts · Q1 2027<br/>→ v2.1.0"]
    L["<b>⏭️ Later</b><br/>4 prompts · H2 2027 · v2.2"]
    D["<b>🎨 Design boards</b><br/>10 prompts · one milestone ahead"]:::design
    F --> E --> J --> A --> G --> I --> L
    F --> J
    J --> G
    D -.-> E
    D -.-> J
    D -.-> G
    D -.-> I
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
```

Families overlap in time (the judge's runner and eval packs start in October, next to the foundations): see the
timeline in §2. The workstreams inside each family:

| Family | Workstream | Prompts (in order) | What it brings |
|---|---|---|---|
| Foundations | 🛡️ **Cluster guardrails & fences** | [`mi-01`](prompts/prompt-mi-01.md) [`mi-02`](prompts/prompt-mi-02.md) [`mi-03`](prompts/prompt-mi-03.md) [`mi-04`](prompts/prompt-mi-04.md) [`mi-08`](prompts/prompt-mi-08.md) [`mi-11`](prompts/prompt-mi-11.md) | Data-loss guards, chart knobs, NetworkPolicy fences, consoles off xLearn's origin, limit hygiene |
| Foundations | 📨 **NATS auth & event plumbing** | [`mi-05`](prompts/prompt-mi-05.md) [`mi-06`](prompts/prompt-mi-06.md) | Per-service nkeys + ACLs, dead letters, identity on NATS — precondition for erase and the judge · shares `mi-11` |
| Foundations | 🧭 **Curriculum spine (multi-course)** | [`m1-01`](prompts/prompt-m1-01.md) [`m1-09`](prompts/prompt-m1-09.md) [`m1-02`](prompts/prompt-m1-02.md) [`m1-03`](prompts/prompt-m1-03.md) [`m1-08`](prompts/prompt-m1-08.md) | Course manifest, frozen item schema, path_slug everywhere, course routing — no behaviour change · shares `p-02` |
| Foundations | 🔐 **Security floor** | [`m1-04`](prompts/prompt-m1-04.md) [`m1-05`](prompts/prompt-m1-05.md) [`m1-06`](prompts/prompt-m1-06.md) | Roles in the DB, admin CLI, revocable sessions, typed limits, withhold(), safe Markdown |
| Foundations | 💬 **Coach v2** | [`m1-10`](prompts/prompt-m1-10.md) [`m1-07`](prompts/prompt-m1-07.md) | Per-feature keys, provider hygiene, assist capture, BYO caps |
| Learning engine | 🔁 **Touch engine & projections** | [`m2-01`](prompts/prompt-m2-01.md) [`m2-02`](prompts/prompt-m2-02.md) [`m2-04`](prompts/prompt-m2-04.md) [`m2-05`](prompts/prompt-m2-05.md) | Revision touches become real timed attempts, projections v2, the D2 ladder rule, Today in minutes |
| Learning engine | 🌐 **Public profile v2** | [`m2-03`](prompts/prompt-m2-03.md) | public-read authority, visibility toggles, count-only mocks, judge-checked % · shares `m1-05`, `m3-10` |
| Judge | 📦 **Eval packs & authoring tooling** | [`mi-07`](prompts/prompt-mi-07.md) [`m3-01`](prompts/prompt-m3-01.md) [`m3-02`](prompts/prompt-m3-02.md) | Private eval-pack repo, packlint + pre-push fingerprint hook, the pack pipeline |
| Judge | 🧪 **Sandbox & runner** | [`mi-14`](prompts/prompt-mi-14.md) [`spk-01`](prompts/prompt-spk-01.md) [`spk-02`](prompts/prompt-spk-02.md) [`mi-09`](prompts/prompt-mi-09.md) [`m3-03`](prompts/prompt-m3-03.md) [`m3-04`](prompts/prompt-m3-04.md) [`m3-15`](prompts/prompt-m3-15.md) [`mi-10`](prompts/prompt-mi-10.md) | Spike-proven jail, host sandbox block, Go/C++/Python runner, runner-v1.0.0 dark on prod · shares `p-01` |
| Judge | ⚖️ **Judge service** | [`m3-05`](prompts/prompt-m3-05.md) [`m3-06`](prompts/prompt-m3-06.md) [`m3-14`](prompts/prompt-m3-14.md) [`m3-07`](prompts/prompt-m3-07.md) [`m3-08`](prompts/prompt-m3-08.md) [`m3-09`](prompts/prompt-m3-09.md) [`m3-10`](prompts/prompt-m3-10.md) | The 8th service: queue, graders, admission, learner API; practice/review/assessment consume its evidence · shares `l-01` |
| Judge | 🖥️ **Judge UI** | [`m3-11`](prompts/prompt-m3-11.md) [`m3-12`](prompts/prompt-m3-12.md) [`m3-13`](prompts/prompt-m3-13.md) | Workspace-Code ★, results dock, Problems/Arena, degradation badges — judge on for the owner |
| Judge | 🧵 **Pilot course: go-concurrency** | [`p-01`](prompts/prompt-p-01.md) [`p-02`](prompts/prompt-p-02.md) [`p-03`](prompts/prompt-p-03.md) | go-race profile, the 2nd course as manifest + content, quiz + race verdict |
| Platform AI | ✨ **Platform AI** | [`spk-03`](prompts/prompt-spk-03.md) [`mi-12`](prompts/prompt-mi-12.md) [`m4-01`](prompts/prompt-m4-01.md) [`m4-02`](prompts/prompt-m4-02.md) [`m4-03`](prompts/prompt-m4-03.md) [`m4-04`](prompts/prompt-m4-04.md) [`m4-05`](prompts/prompt-m4-05.md) [`m4-06`](prompts/prompt-m4-06.md) [`m4-07`](prompts/prompt-m4-07.md) | WIF credential, shared LLM layer, Scorer + ledger, analyzer, provisional grades, consents |
| Learner gate & GA | 🚪 **Learner gate (built now, opened in v3)** | [`l-01`](prompts/prompt-l-01.md) [`l-02`](prompts/prompt-l-02.md) [`l-03`](prompts/prompt-l-03.md) [`l-05`](prompts/prompt-l-05.md) [`l-04`](prompts/prompt-l-04.md) | Erase end to end, invite-only admission, acceptance step + notice, production rehearsal · shares `m3-05`, `mi-04` |
| Learner gate & GA | 🏁 **v2.0 GA** | [`ga-01`](prompts/prompt-ga-01.md) [`ga-02`](prompts/prompt-ga-02.md) | The owner-facing default flip: judge + platform AI on, pilot active → v2.0.0 |
| Interviewer (v2.1) | 🎙️ **Interviewer: text (M6a)** | [`spk-04`](prompts/prompt-spk-04.md) [`m6a-01`](prompts/prompt-m6a-01.md) [`m6a-02`](prompts/prompt-m6a-02.md) [`m6a-03`](prompts/prompt-m6a-03.md) [`m6a-04`](prompts/prompt-m6a-04.md) [`m6a-05`](prompts/prompt-m6a-05.md) [`m6a-06`](prompts/prompt-m6a-06.md) | Interview core with failsafes, text brain, one honest mock score, coding rounds, UI |
| Interviewer (v2.1) | 🔊 **Interviewer: voice (M6b)** | [`mi-13`](prompts/prompt-mi-13.md) [`m6b-01`](prompts/prompt-m6b-01.md) [`m6b-02`](prompts/prompt-m6b-02.md) [`m6b-03`](prompts/prompt-m6b-03.md) [`m6b-04`](prompts/prompt-m6b-04.md) | Voice shell with no media on the node, robustness + cost cap, voice UI → v2.1.0 |
| Cross-cutting | 🎨 **Design boards** | [`ds-m1-01`](prompts/prompt-ds-m1-01.md) [`ds-m2-01`](prompts/prompt-ds-m2-01.md) [`ds-l-01`](prompts/prompt-ds-l-01.md) [`ds-m3-01`](prompts/prompt-ds-m3-01.md) [`ds-m3-02`](prompts/prompt-ds-m3-02.md) [`ds-p-01`](prompts/prompt-ds-p-01.md) [`ds-m4-01`](prompts/prompt-ds-m4-01.md) [`ds-m6a-01`](prompts/prompt-ds-m6a-01.md) [`ds-m6a-02`](prompts/prompt-ds-m6a-02.md) [`ds-m6b-01`](prompts/prompt-ds-m6b-01.md) | Agent-drafted artboards, one milestone ahead; the merge is the freeze (D40) |
| Later | ⏭️ **Later** | [`m5-01`](prompts/prompt-m5-01.md) [`m6c-01`](sprints/sprint-m6c-01.md) [`m6c-02`](sprints/sprint-m6c-02.md) [`m6c-03`](sprints/sprint-m6c-03.md) | DSA evaluator-only (M5) once every item is packed; v2.2 interviewer extras |

## 2. When: the v2.0 timeline

One row per workstream, **Sep 25 → mid-Jan (v2.0)**. A bar is a run of consecutive weeks with that workstream's
prompts in flight; the number on it is how many prompts. Diamonds are owner calendar dates and key tags. v2.1
(Q1 2027) is the interviewer: text, then voice, after GA.

```mermaid
---
displayMode: compact
config:
  gantt:
    leftPadding: 230
    fontSize: 12
    sectionFontSize: 12
---
gantt
    title xLearn v2.0 — workstreams over time (indicative)
    dateFormat YYYY-MM-DD
    axisFormat %d %b
    tickInterval 2week
    section Owner calendar
    H0 :milestone, ev0, 2026-09-25, 0d
    spikes :crit, ev1, 2026-09-25, 2d
    window :milestone, ev2, 2026-10-24, 0d
    section Cluster guardrails and fences
    4 :t1, 2026-09-28, 2026-10-10
    1 :t2, 2026-10-17, 2026-10-24
    1 :t3, 2026-11-09, 2026-11-21
    section NATS auth and event plumbing
    2 :t4, 2026-10-05, 2026-10-17
    section Curriculum spine (multi-course)
    4 :t5, 2026-09-28, 2026-10-17
    1 :t6, 2026-10-24, 2026-11-02
    section Security floor
    3 :t7, 2026-10-12, 2026-10-17
    section Coach v2
    2 :t8, 2026-10-12, 2026-10-24
    section Touch engine and projections
    4 :t9, 2026-10-24, 2026-11-09
    section Public profile v2
    1 :t10, 2026-10-24, 2026-11-02
    section Eval packs and authoring tooling
    3 :t11, 2026-09-28, 2026-10-10
    section Sandbox and runner
    2 :t12, 2026-09-25, 2026-09-27
    1 :t13, 2026-10-05, 2026-10-10
    5 :t14, 2026-10-17, 2026-11-09
    section Judge service
    7 :t15, 2026-11-02, 2026-11-21
    section Judge UI
    3 :t16, 2026-11-21, 2026-12-01
    section Pilot course - go-concurrency
    3 :t17, 2026-12-01, 2026-12-24
    section Platform AI
    1 :t18, 2026-09-25, 2026-09-27
    8 :t19, 2026-12-01, 2026-12-24
    section Learner gate (built now, opened in v3)
    3 :t20, 2026-11-02, 2026-11-21
    2 :t21, 2026-12-01, 2026-12-24
    section v2.0 GA
    2 :t22, 2026-12-28, 2027-01-16
    section Design boards
    3 :t23, 2026-09-28, 2026-10-03
    2 :t24, 2026-10-17, 2026-11-02
    2 :t25, 2026-11-09, 2026-11-21
    section Key tags
    v1.6.0 :milestone, t26, 2026-10-09, 0d
    v1.10.0 :milestone, t27, 2026-11-08, 0d
    v1.14.0 :milestone, t28, 2026-11-30, 0d
    v1.17.0 :milestone, t29, 2026-12-23, 0d
    v2.0.0 :milestone, t30, 2027-01-15, 0d
```

## 3. The release train

Only a tag deploys (1.x minors until GA; ADR-0034). Every prompt between two tags merges dark and ships with the
next one; infra-only prompts deploy through their own GitOps PRs. Each pill shows the tag, what it carries and
the prompt that cuts it.

```mermaid
flowchart TB
    subgraph ph0["M1 · October"]
        direction LR
        r0_0(["<b>v1.6.0</b><br/>M1a expand<br/><i>m1-02</i>"]):::tag
        r0_1(["<b>v1.7.0</b><br/>M1b: routing, security, coach<br/><i>m1-07</i>"]):::tag
        r0_2(["<b>v1.8.0</b><br/>M1c contract<br/><i>m1-08</i>"]):::tag
        r0_0 --> r0_1
        r0_1 --> r0_2
    end
    subgraph ph1["M2 + erase · late Oct – early Nov"]
        direction LR
        r1_0(["<b>v1.9.0</b><br/>M2 consumers<br/><i>m2-02</i>"]):::tag
        r1_1(["<b>v1.10.0</b><br/>Touch UI, D2, public-read<br/><i>m2-05</i>"]):::tag
        r1_2(["<b>v1.11.0</b><br/>erase consumers<br/><i>l-01</i>"]):::tag
        r1_3(["<b>v1.12.0</b><br/>erase producer<br/><i>l-02</i>"]):::tag
        r1_0 --> r1_1
        r1_1 --> r1_2
        r1_2 --> r1_3
    end
    subgraph ph2["Runner + judge · Oct – Nov"]
        direction LR
        r2_0(["<b>runner-v1.0.0</b><br/>runner: Go/C++/Python<br/><i>m3-15</i>"]):::tag
        r2_1(["<b>v1.13.0</b><br/>judge dark + evalpack 1.0<br/><i>m3-07</i>"]):::tag
        r2_2(["<b>v1.14.0</b><br/>judge on for the owner<br/><i>m3-13</i>"]):::tag
        r2_0 --> r2_1
        r2_1 --> r2_2
    end
    subgraph ph3["Pilot, AI, gate · December"]
        direction LR
        r3_0(["<b>runner-v1.1.0</b><br/>go-race profile<br/><i>p-01</i>"]):::tag
        r3_1(["<b>v1.15.0</b><br/>pilot course (preview)<br/><i>p-03</i>"]):::tag
        r3_2(["<b>v1.16.0</b><br/>platform AI on<br/><i>m4-07</i>"]):::tag
        r3_3(["<b>v1.17.0</b><br/>invite front door + erase<br/><i>l-04</i>"]):::tag
        r3_0 --> r3_1
        r3_1 --> r3_2
        r3_2 --> r3_3
    end
    subgraph ph4["GA · late Dec – Jan"]
        direction LR
        r4_0(["<b>v2.0.0</b><br/>owner-facing GA<br/><i>ga-02</i>"]):::tag
    end
    subgraph ph5["v2.1 · Q1 2027"]
        direction LR
        r5_0(["<b>v2.0.x</b><br/>interviewer patches, dark<br/><i>m6a-06 · mi-13 · m6b-03</i>"]):::tag
        r5_1(["<b>v2.1.0</b><br/>interviewer GA<br/><i>m6b-04</i>"]):::tag
        r5_0 --> r5_1
    end
    ph0 --> ph1
    ph1 --> ph2
    ph2 --> ph3
    ph3 --> ph4
    ph4 --> ph5
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
```

## 4. What each workstream brings, prompt by prompt

One diagram per family: each column is a workstream, its prompts top to bottom in run order (`#` = order
number). A green pill is the tag the prompt above it cuts. A grey dashed box at the bottom of a column belongs
to another workstream but carries part of this one's work.

### Foundations · Sep 25 – Oct 30

```mermaid
flowchart TB
    subgraph coach_g["💬 Coach v2"]
        direction TB
        m110["<b>22 · m1-10</b><br/>coach keys + providers"]:::build
        m107["<b>27 · m1-07</b><br/>coach assist + caps"]:::build
        m107_tag(["v1.7.0"]):::tag
        m110 --> m107
        m107 --> m107_tag
    end
    subgraph sec_g["🔐 Security floor"]
        direction TB
        m104["<b>21 · m1-04</b><br/>roles, sessions, admin CLI"]:::build
        m105["<b>23 · m1-05</b><br/>limits + public floor"]:::build
        m106["<b>24 · m1-06</b><br/>withhold() + Markdown"]:::build
        m104 --> m105
        m105 --> m106
    end
    subgraph spine_g["🧭 Curriculum spine (multi-course)"]
        direction TB
        m101["<b>4 · m1-01</b><br/>manifest + item schema"]:::build
        m109["<b>8 · m1-09</b><br/>converter, loader, expand"]:::build
        m102["<b>11 · m1-02</b><br/>path_slug + v2 envelope"]:::build
        m102_tag(["v1.6.0"]):::tag
        m103["<b>20 · m1-03</b><br/>course resolution"]:::build
        m108["<b>29 · m1-08</b><br/>M1 contract"]:::build
        m108_tag(["v1.8.0"]):::tag
        m101 --> m109
        m109 --> m102
        m102 --> m102_tag
        m102_tag --> m103
        m103 --> m108
        m108 --> m108_tag
        p02_sh_spine["58 · p-02<br/>go-concurrency course"]:::shared
        m108_tag ~~~ p02_sh_spine
    end
    subgraph nats_g["📨 NATS auth & event plumbing"]
        direction TB
        mi05["<b>10 · mi-05</b><br/>N0: topology, dead letters"]:::build
        mi06["<b>19 · mi-06</b><br/>NATS auth N1–N3"]:::infra
        mi05 --> mi06
        mi11_sh_nats["52 · mi-11<br/>egress, PG limits, N4"]:::shared
        mi06 ~~~ mi11_sh_nats
    end
    subgraph guard_g["🛡️ Cluster guardrails & fences"]
        direction TB
        mi01["<b>1 · mi-01</b><br/>DB/NATS prune guards + chart 0.3.0"]:::infra
        mi02["<b>3 · mi-02</b><br/>host-verify checks"]:::infra
        mi03["<b>13 · mi-03</b><br/>NetworkPolicy fences"]:::infra
        mi04["<b>15 · mi-04</b><br/>consoles → ops."]:::infra
        mi08["<b>25 · mi-08</b><br/>limit hygiene, PSA"]:::infra
        mi11["<b>52 · mi-11</b><br/>egress, PG limits, N4"]:::infra
        mi01 --> mi02
        mi02 --> mi03
        mi03 --> mi04
        mi04 --> mi08
        mi08 --> mi11
    end
    classDef build fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef infra fill:#1f2937,stroke:#f59e0b,color:#e5e7eb
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
    classDef spike fill:#1f2937,stroke:#f472b6,color:#e5e7eb,stroke-dasharray:4 3
    classDef outline fill:#111827,stroke:#6b7280,color:#9ca3af,stroke-dasharray:2 2
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef shared fill:#111827,stroke:#9ca3af,color:#9ca3af,stroke-dasharray:3 3
```

**🛡️ Cluster guardrails & fences** — Data-loss guards, chart knobs, NetworkPolicy fences, consoles off xLearn's origin, limit hygiene

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 1 | [`mi-01`](prompts/prompt-mi-01.md) | Data-loss guards on DB + NATS, chart 0.3.0 knobs (no pod rolls) | infra PRs |
| 3 | [`mi-02`](prompts/prompt-mi-02.md) | host-verify --cluster: on-demand cluster checks (the only ops tool) | infra PRs |
| 13 | [`mi-03`](prompts/prompt-mi-03.md) | NetworkPolicy fences: databases, messaging, xlearn ingress | infra PRs |
| 15 | [`mi-04`](prompts/prompt-mi-04.md) | Admin consoles move to ops.sujaykumar.dev | infra PRs |
| 25 | [`mi-08`](prompts/prompt-mi-08.md) | Limit hygiene (≈ −3 GiB), PSA labels, SA tokens off | infra PRs |
| 52 | [`mi-11`](prompts/prompt-mi-11.md) | Track B finish: egress, PG connection limits, Renovate, N4 | infra PRs |

**📨 NATS auth & event plumbing** — Per-service nkeys + ACLs, dead letters, identity on NATS — precondition for erase and the judge

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 10 | [`mi-05`](prompts/prompt-mi-05.md) | N0 (dark): NATS topology, dead letters, identity on NATS | merge (dark until next tag) |
| 19 | [`mi-06`](prompts/prompt-mi-06.md) | NATS auth live: nkey users + ACLs, legacy closed (N1–N3) | infra PRs |
| — | also [`mi-11`](prompts/prompt-mi-11.md) | this prompt carries part of this workstream | — |

**🧭 Curriculum spine (multi-course)** — Course manifest, frozen item schema, path_slug everywhere, course routing — no behaviour change

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 4 | [`m1-01`](prompts/prompt-m1-01.md) | Prod-parity compose, course manifest + golden, frozen item schema | merge (dark until next tag) |
| 8 | [`m1-09`](prompts/prompt-m1-09.md) | Curriculum converter, loader + guards, expand migration, content CI | merge (dark until next tag) |
| 11 | [`m1-02`](prompts/prompt-m1-02.md) | path_slug everywhere + v2 envelope consumers | **tag v1.6.0** |
| 20 | [`m1-03`](prompts/prompt-m1-03.md) | Course resolution end to end; DSA stays pixel-identical | merge (dark until next tag) |
| 29 | [`m1-08`](prompts/prompt-m1-08.md) | M1 contract: drop the v1 columns (snapshot first) | **tag v1.8.0** |
| — | also [`p-02`](prompts/prompt-p-02.md) | this prompt carries part of this workstream | — |

**🔐 Security floor** — Roles in the DB, admin CLI, revocable sessions, typed limits, withhold(), safe Markdown

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 21 | [`m1-04`](prompts/prompt-m1-04.md) | Roles in DB, revocable sessions, admin CLI, CSP, DEV_AUTH guard | merge (dark until next tag) |
| 23 | [`m1-05`](prompts/prompt-m1-05.md) | Typed 429/413 limits + public-profile floor (count-only mocks) | merge (dark until next tag) |
| 24 | [`m1-06`](prompts/prompt-m1-06.md) | withhold() on every route, safe Markdown, revision v2 | merge (dark until next tag) |

**💬 Coach v2** — Per-feature keys, provider hygiene, assist capture, BYO caps

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 22 | [`m1-10`](prompts/prompt-m1-10.md) | Coach keys: per-feature defaults, catalog, AEAD keyring, typed errors | merge (dark until next tag) |
| 27 | [`m1-07`](prompts/prompt-m1-07.md) | Coach assist capture, mode gate, BYO caps | **tag v1.7.0** |

### Learning engine · late Oct – Nov

```mermaid
flowchart TB
    subgraph public_g["🌐 Public profile v2"]
        direction TB
        m203["<b>36 · m2-03</b><br/>public-read + visibility"]:::build
        m105_sh_public["23 · m1-05<br/>limits + public floor"]:::shared
        m203 ~~~ m105_sh_public
        m310_sh_public["50 · m3-10<br/>review/assessment on judge"]:::shared
        m203 ~~~ m310_sh_public
    end
    subgraph touch_g["🔁 Touch engine & projections"]
        direction TB
        m201["<b>33 · m2-01</b><br/>touch attempts"]:::build
        m202["<b>34 · m2-02</b><br/>consumers + projections v2"]:::build
        m202_tag(["v1.9.0"]):::tag
        m204["<b>37 · m2-04</b><br/>Touch UI★ + Today"]:::build
        m205["<b>38 · m2-05</b><br/>producers on + D2"]:::build
        m205_tag(["v1.10.0"]):::tag
        m201 --> m202
        m202 --> m202_tag
        m202_tag --> m204
        m204 --> m205
        m205 --> m205_tag
    end
    classDef build fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef infra fill:#1f2937,stroke:#f59e0b,color:#e5e7eb
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
    classDef spike fill:#1f2937,stroke:#f472b6,color:#e5e7eb,stroke-dasharray:4 3
    classDef outline fill:#111827,stroke:#6b7280,color:#9ca3af,stroke-dasharray:2 2
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef shared fill:#111827,stroke:#9ca3af,color:#9ca3af,stroke-dasharray:3 3
```

**🔁 Touch engine & projections** — Revision touches become real timed attempts, projections v2, the D2 ladder rule, Today in minutes

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 33 | [`m2-01`](prompts/prompt-m2-01.md) | Touches become real attempts + review consumer | merge (dark until next tag) |
| 34 | [`m2-02`](prompts/prompt-m2-02.md) | M2 consumers + projections v2 (producers idle) | **tag v1.9.0** |
| 37 | [`m2-04`](prompts/prompt-m2-04.md) | Touch UI ★ + cross-course Today/agenda in minutes | merge (dark until next tag) |
| 38 | [`m2-05`](prompts/prompt-m2-05.md) | Producers on, backfill, replay, D2 ladder rule | **tag v1.10.0** |

**🌐 Public profile v2** — public-read authority, visibility toggles, count-only mocks, judge-checked %

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 36 | [`m2-03`](prompts/prompt-m2-03.md) | public-read, /public/stats, visibility toggles, profile v2 | merge (dark until next tag) |
| — | also [`m1-05`](prompts/prompt-m1-05.md), [`m3-10`](prompts/prompt-m3-10.md) | these prompts carry part of this workstream | — |

### Judge · Oct – Dec

```mermaid
flowchart TB
    subgraph pilot_g["🧵 Pilot course: go-concurrency"]
        direction TB
        p01["<b>57 · p-01</b><br/>go-race profile"]:::build
        p01_tag(["runner-v1.1.0"]):::tag
        p02["<b>58 · p-02</b><br/>go-concurrency course"]:::build
        p03["<b>59 · p-03</b><br/>quiz + race verdict UI"]:::build
        p03_tag(["v1.15.0"]):::tag
        p01 --> p01_tag
        p01_tag --> p02
        p02 --> p03
        p03 --> p03_tag
    end
    subgraph judgeui_g["🖥️ Judge UI"]
        direction TB
        m311["<b>53 · m3-11</b><br/>Workspace-Code UI★"]:::build
        m312["<b>54 · m3-12</b><br/>results dock, Problems, Arena"]:::build
        m313["<b>55 · m3-13</b><br/>progress deltas + exit"]:::build
        m313_tag(["v1.14.0"]):::tag
        m311 --> m312
        m312 --> m313
        m313 --> m313_tag
    end
    subgraph judge_g["⚖️ Judge service"]
        direction TB
        m305["<b>42 · m3-05</b><br/>judge skeleton + loader"]:::build
        m306["<b>43 · m3-06</b><br/>queue, graders, contexts"]:::build
        m314["<b>44 · m3-14</b><br/>admission + learner API"]:::build
        m307["<b>46 · m3-07</b><br/>judge on prod (dark)"]:::build
        m307_tag(["v1.13.0"]):::tag
        m308["<b>48 · m3-08</b><br/>practice ← judge; 45:00"]:::build
        m309["<b>49 · m3-09</b><br/>gateway judge BFF"]:::build
        m310["<b>50 · m3-10</b><br/>review/assessment on judge"]:::build
        m305 --> m306
        m306 --> m314
        m314 --> m307
        m307 --> m307_tag
        m307_tag --> m308
        m308 --> m309
        m309 --> m310
        l01_sh_judge["39 · l-01<br/>erase consumers"]:::shared
        m310 ~~~ l01_sh_judge
    end
    subgraph runner_g["🧪 Sandbox & runner"]
        direction TB
        mi14["<b>12 · mi-14</b><br/>runner namespace, default-deny"]:::infra
        spk01["<b>16 · spk-01</b><br/>spike: sandbox P0–P2"]:::spike
        spk02["<b>17 · spk-02</b><br/>spike: amd64 + image volume"]:::spike
        mi09["<b>26 · mi-09</b><br/>host-window prep (Oct 24)"]:::infra
        m303["<b>30 · m3-03</b><br/>runner core: jail, cgroups"]:::build
        m304["<b>31 · m3-04</b><br/>Go/C++/Python profiles"]:::build
        m315["<b>32 · m3-15</b><br/>runner release"]:::build
        m315_tag(["runner-v1.0.0"]):::tag
        mi10["<b>41 · mi-10</b><br/>runner dark on prod"]:::infra
        mi14 --> spk01
        spk01 --> spk02
        spk02 --> mi09
        mi09 --> m303
        m303 --> m304
        m304 --> m315
        m315 --> m315_tag
        m315_tag --> mi10
        p01_sh_runner["57 · p-01<br/>go-race profile"]:::shared
        mi10 ~~~ p01_sh_runner
    end
    subgraph packs_g["📦 Eval packs & authoring tooling"]
        direction TB
        mi07["<b>5 · mi-07</b><br/>eval-pack repo + GHCR"]:::infra
        m301["<b>9 · m3-01</b><br/>packlint + fingerprint hook"]:::build
        m302["<b>14 · m3-02</b><br/>eval-pack pipeline"]:::build
        mi07 --> m301
        m301 --> m302
    end
    classDef build fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef infra fill:#1f2937,stroke:#f59e0b,color:#e5e7eb
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
    classDef spike fill:#1f2937,stroke:#f472b6,color:#e5e7eb,stroke-dasharray:4 3
    classDef outline fill:#111827,stroke:#6b7280,color:#9ca3af,stroke-dasharray:2 2
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef shared fill:#111827,stroke:#9ca3af,color:#9ca3af,stroke-dasharray:3 3
```

**📦 Eval packs & authoring tooling** — Private eval-pack repo, packlint + pre-push fingerprint hook, the pack pipeline

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 5 | [`mi-07`](prompts/prompt-mi-07.md) | Private eval-pack repo, machine user, GHCR, pull secret | infra PRs |
| 9 | [`m3-01`](prompts/prompt-m3-01.md) | Authoring tooling: packlint, contract_hash, pre-push fingerprint hook | merge (dark until next tag) |
| 14 | [`m3-02`](prompts/prompt-m3-02.md) | Eval-pack pipeline: validation gates, data image, fixture pack | merge (dark until next tag) |

**🧪 Sandbox & runner** — Spike-proven jail, host sandbox block, Go/C++/Python runner, runner-v1.0.0 dark on prod

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 12 | [`mi-14`](prompts/prompt-mi-14.md) | Empty default-deny xlearn-runner namespace + admission policy | infra PRs |
| 16 | [`spk-01`](prompts/prompt-spk-01.md) | Spike: sandbox mechanism P0–P2 (throwaway) | results docs only |
| 17 | [`spk-02`](prompts/prompt-spk-02.md) | Spike: amd64 replay + eval-pack image volume (throwaway) | results docs only |
| 26 | [`mi-09`](prompts/prompt-mi-09.md) | Host-window prep: sandbox host block, kubelet reservation (Sat Oct 24) | infra PRs |
| 30 | [`m3-03`](prompts/prompt-m3-03.md) | Runner core: supervisor, jail, per-case cgroups, job API | merge (dark until next tag) |
| 31 | [`m3-04`](prompts/prompt-m3-04.md) | Runner profiles Go/C++/Python + harness codecs | merge (dark until next tag) |
| 32 | [`m3-15`](prompts/prompt-m3-15.md) | Reproducible runner image + acceptance suite | **tag runner-v1.0.0** |
| 41 | [`mi-10`](prompts/prompt-mi-10.md) | runner-v1.0.0 dark on production + acceptance suite | infra PRs |
| — | also [`p-01`](prompts/prompt-p-01.md) | this prompt carries part of this workstream | — |

**⚖️ Judge service** — The 8th service: queue, graders, admission, learner API; practice/review/assessment consume its evidence

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 42 | [`m3-05`](prompts/prompt-m3-05.md) | judge service: skeleton, schema, pack loader, erase consumer | merge (dark until next tag) |
| 43 | [`m3-06`](prompts/prompt-m3-06.md) | judge core: queue, runner lane, graders, contexts | merge (dark until next tag) |
| 44 | [`m3-14`](prompts/prompt-m3-14.md) | judge admission control, learner API, drafts, arena history | merge (dark until next tag) |
| 46 | [`m3-07`](prompts/prompt-m3-07.md) | judge on prod (dark): evalpack v1.0.0, ACL, HelmRelease | **tag v1.13.0** |
| 48 | [`m3-08`](prompts/prompt-m3-08.md) | practice consumes judge evidence; the 45:00 method | merge (dark until next tag) |
| 49 | [`m3-09`](prompts/prompt-m3-09.md) | Gateway judge BFF, DTO deny-list, typed limits | merge (dark until next tag) |
| 50 | [`m3-10`](prompts/prompt-m3-10.md) | review + assessment on judge signals (pre-fill, judge-checked %) | merge (dark until next tag) |
| — | also [`l-01`](prompts/prompt-l-01.md) | this prompt carries part of this workstream | — |

**🖥️ Judge UI** — Workspace-Code ★, results dock, Problems/Arena, degradation badges — judge on for the owner

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 53 | [`m3-11`](prompts/prompt-m3-11.md) | Workspace-Code UI ★ + lazy CodeMirror | merge (dark until next tag) |
| 54 | [`m3-12`](prompts/prompt-m3-12.md) | Results dock, Problems, Arena, degradation badges | merge (dark until next tag) |
| 55 | [`m3-13`](prompts/prompt-m3-13.md) | Progress deltas + M3 exit tests; judge on for the owner | **tag v1.14.0** |

**🧵 Pilot course: go-concurrency** — go-race profile, the 2nd course as manifest + content, quiz + race verdict

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 57 | [`p-01`](prompts/prompt-p-01.md) | go-race runner profile | **tag runner-v1.1.0** |
| 58 | [`p-02`](prompts/prompt-p-02.md) | go-concurrency course as manifest (preview) + multi-course nav | merge (dark until next tag) |
| 59 | [`p-03`](prompts/prompt-p-03.md) | Quiz + race verdict UI | **tag v1.15.0** |

### Platform AI · Oct spike · Dec build

```mermaid
flowchart TB
    subgraph ai_g["✨ Platform AI"]
        direction TB
        spk03["<b>18 · spk-03</b><br/>spike: WIF credential"]:::spike
        mi12["<b>56 · mi-12</b><br/>M4 gates: egress, secret"]:::infra
        m401["<b>60 · m4-01</b><br/>LLM layer + WIF"]:::build
        m402["<b>61 · m4-02</b><br/>Scorer, ledger, caps"]:::build
        m403["<b>62 · m4-03</b><br/>analyzer"]:::build
        m404["<b>63 · m4-04</b><br/>provisional grades, dispute"]:::build
        m405["<b>64 · m4-05</b><br/>allowance + consents"]:::build
        m406["<b>65 · m4-06</b><br/>AI UI★"]:::build
        m407["<b>66 · m4-07</b><br/>canary + caps sizing"]:::build
        m407_tag(["v1.16.0"]):::tag
        spk03 --> mi12
        mi12 --> m401
        m401 --> m402
        m402 --> m403
        m403 --> m404
        m404 --> m405
        m405 --> m406
        m406 --> m407
        m407 --> m407_tag
    end
    classDef build fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef infra fill:#1f2937,stroke:#f59e0b,color:#e5e7eb
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
    classDef spike fill:#1f2937,stroke:#f472b6,color:#e5e7eb,stroke-dasharray:4 3
    classDef outline fill:#111827,stroke:#6b7280,color:#9ca3af,stroke-dasharray:2 2
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef shared fill:#111827,stroke:#9ca3af,color:#9ca3af,stroke-dasharray:3 3
```

**✨ Platform AI** — WIF credential, shared LLM layer, Scorer + ledger, analyzer, provisional grades, consents

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 18 | [`spk-03`](prompts/prompt-spk-03.md) | Spike: WIF credential for platform AI (throwaway) | results docs only |
| 56 | [`mi-12`](prompts/prompt-mi-12.md) | M4 gates: judge 443 egress, LLM secret, provider runbook | infra PRs |
| 60 | [`m4-01`](prompts/prompt-m4-01.md) | Shared LLM layer + WIF auth + retention policy | merge (dark until next tag) |
| 61 | [`m4-02`](prompts/prompt-m4-02.md) | judge AI engine: Scorer, ledger, llm lane, caps | merge (dark until next tag) |
| 62 | [`m4-03`](prompts/prompt-m4-03.md) | Analyzer on concluded attempts + acceptance-set harness | merge (dark until next tag) |
| 63 | [`m4-04`](prompts/prompt-m4-04.md) | Provisional grades, dispute, re-grade, honor claims | merge (dark until next tag) |
| 64 | [`m4-05`](prompts/prompt-m4-05.md) | AI allowance + consents | merge (dark until next tag) |
| 65 | [`m4-06`](prompts/prompt-m4-06.md) | AI UI ★: suggestion/dispute, pointer notes, allowance | merge (dark until next tag) |
| 66 | [`m4-07`](prompts/prompt-m4-07.md) | Canary + caps sizing; platform AI on for the owner | **tag v1.16.0** |

### Learner gate & GA · Nov – Jan

```mermaid
flowchart TB
    subgraph ga_g["🏁 v2.0 GA"]
        direction TB
        ga01["<b>69 · ga-01</b><br/>GA PR + rc rehearsal"]:::build
        ga02["<b>70 · ga-02</b><br/>cut v2.0.0"]:::build
        ga02_tag(["v2.0.0"]):::tag
        ga01 --> ga02
        ga02 --> ga02_tag
    end
    subgraph gate_g["🚪 Learner gate (built now, opened in v3)"]
        direction TB
        l01["<b>39 · l-01</b><br/>erase consumers"]:::build
        l01_tag(["v1.11.0"]):::tag
        l02["<b>40 · l-02</b><br/>erase producer + UI"]:::build
        l02_tag(["v1.12.0"]):::tag
        l03["<b>47 · l-03</b><br/>invite backend (inert)"]:::build
        l05["<b>67 · l-05</b><br/>acceptance★ + notice"]:::build
        l04["<b>68 · l-04</b><br/>web erase + rehearsal"]:::build
        l04_tag(["v1.17.0"]):::tag
        l01 --> l01_tag
        l01_tag --> l02
        l02 --> l02_tag
        l02_tag --> l03
        l03 --> l05
        l05 --> l04
        l04 --> l04_tag
        m305_sh_gate["42 · m3-05<br/>judge skeleton + loader"]:::shared
        l04_tag ~~~ m305_sh_gate
        mi04_sh_gate["15 · mi-04<br/>consoles → ops."]:::shared
        l04_tag ~~~ mi04_sh_gate
    end
    classDef build fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef infra fill:#1f2937,stroke:#f59e0b,color:#e5e7eb
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
    classDef spike fill:#1f2937,stroke:#f472b6,color:#e5e7eb,stroke-dasharray:4 3
    classDef outline fill:#111827,stroke:#6b7280,color:#9ca3af,stroke-dasharray:2 2
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef shared fill:#111827,stroke:#9ca3af,color:#9ca3af,stroke-dasharray:3 3
```

**🚪 Learner gate (built now, opened in v3)** — Erase end to end, invite-only admission, acceptance step + notice, production rehearsal

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 39 | [`l-01`](prompts/prompt-l-01.md) | Erase consumers in every service | **tag v1.11.0** |
| 40 | [`l-02`](prompts/prompt-l-02.md) | Erase producer + DELETE /api/me (testers only) | **tag v1.12.0** |
| 47 | [`l-03`](prompts/prompt-l-03.md) | Invite admission backend (inert while signup is closed) | merge (dark until next tag) |
| 67 | [`l-05`](prompts/prompt-l-05.md) | Invite front door: acceptance step ★, privacy notice, AI consents | merge (dark until next tag) |
| 68 | [`l-04`](prompts/prompt-l-04.md) | Web erase for non-owners + invite rehearsal on prod | **tag v1.17.0** |
| — | also [`m3-05`](prompts/prompt-m3-05.md), [`mi-04`](prompts/prompt-mi-04.md) | these prompts carry part of this workstream | — |

**🏁 v2.0 GA** — The owner-facing default flip: judge + platform AI on, pilot active → v2.0.0

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 69 | [`ga-01`](prompts/prompt-ga-01.md) | GA PR: .release-line = 2, default flips, rc rehearsal | merge (dark until next tag) |
| 70 | [`ga-02`](prompts/prompt-ga-02.md) | Widen ranges, snapshot, tag, verify | **tag v2.0.0** |

### Interviewer (v2.1) · Q1 2027

```mermaid
flowchart TB
    subgraph ivvoice_g["🔊 Interviewer: voice (M6b)"]
        direction TB
        mi13["<b>81 · mi-13</b><br/>voice platform gates"]:::infra
        mi13_tag(["v2.0.x"]):::tag
        m6b01["<b>82 · m6b-01</b><br/>voice shell + SDP broker"]:::build
        m6b02["<b>83 · m6b-02</b><br/>voice robustness + cost cap"]:::build
        m6b03["<b>84 · m6b-03</b><br/>voice UI + gates"]:::build
        m6b03_tag(["v2.0.x"]):::tag
        m6b04["<b>85 · m6b-04</b><br/>voice e2e + flip"]:::build
        m6b04_tag(["v2.1.0"]):::tag
        mi13 --> mi13_tag
        mi13_tag --> m6b01
        m6b01 --> m6b02
        m6b02 --> m6b03
        m6b03 --> m6b03_tag
        m6b03_tag --> m6b04
        m6b04 --> m6b04_tag
    end
    subgraph ivtext_g["🎙️ Interviewer: text (M6a)"]
        direction TB
        spk04["<b>71 · spk-04</b><br/>spike S6: voice shell"]:::spike
        m6a01["<b>74 · m6a-01</b><br/>interview core + failsafes"]:::build
        m6a02["<b>75 · m6a-02</b><br/>text brain"]:::build
        m6a03["<b>76 · m6a-03</b><br/>scoring: accept → ScoreMock"]:::build
        m6a04["<b>77 · m6a-04</b><br/>coding rounds"]:::build
        m6a05["<b>78 · m6a-05</b><br/>interviewer UI 1"]:::build
        m6a06["<b>79 · m6a-06</b><br/>interviewer UI 2 + exit"]:::build
        m6a06_tag(["v2.0.x"]):::tag
        spk04 --> m6a01
        m6a01 --> m6a02
        m6a02 --> m6a03
        m6a03 --> m6a04
        m6a04 --> m6a05
        m6a05 --> m6a06
        m6a06 --> m6a06_tag
    end
    classDef build fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef infra fill:#1f2937,stroke:#f59e0b,color:#e5e7eb
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
    classDef spike fill:#1f2937,stroke:#f472b6,color:#e5e7eb,stroke-dasharray:4 3
    classDef outline fill:#111827,stroke:#6b7280,color:#9ca3af,stroke-dasharray:2 2
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef shared fill:#111827,stroke:#9ca3af,color:#9ca3af,stroke-dasharray:3 3
```

**🎙️ Interviewer: text (M6a)** — Interview core with failsafes, text brain, one honest mock score, coding rounds, UI

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 71 | [`spk-04`](prompts/prompt-spk-04.md) | Spike S6: voice-shell bake-off (owner present) | results docs only |
| 74 | [`m6a-01`](prompts/prompt-m6a-01.md) | Interview core: state machine, failsafes, caps | merge (dark until next tag) |
| 75 | [`m6a-02`](prompts/prompt-m6a-02.md) | Text brain on the learner's key + replay tests | merge (dark until next tag) |
| 76 | [`m6a-03`](prompts/prompt-m6a-03.md) | Mock scoring: proposal → accept → ScoreMock once | merge (dark until next tag) |
| 77 | [`m6a-04`](prompts/prompt-m6a-04.md) | Coding rounds via judge mock + editor interview mode | merge (dark until next tag) |
| 78 | [`m6a-05`](prompts/prompt-m6a-05.md) | Interviewer UI part 1: setup, consent, text HUD | merge (dark until next tag) |
| 79 | [`m6a-06`](prompts/prompt-m6a-06.md) | Interviewer UI part 2 + M6a exit (dark) | **tag v2.0.x** |

**🔊 Interviewer: voice (M6b)** — Voice shell with no media on the node, robustness + cost cap, voice UI → v2.1.0

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 81 | [`mi-13`](prompts/prompt-mi-13.md) | Voice platform gates: Permissions-Policy, SDP route, egress | **tag v2.0.x** |
| 82 | [`m6b-01`](prompts/prompt-m6b-01.md) | Voice shell + SDP broker + sideband (no media on the node) | merge (dark until next tag) |
| 83 | [`m6b-02`](prompts/prompt-m6b-02.md) | Voice robustness: lease, drain, cost cap, push-to-talk | merge (dark until next tag) |
| 84 | [`m6b-03`](prompts/prompt-m6b-03.md) | Voice UI + browser/EU gates (dark) | **tag v2.0.x** |
| 85 | [`m6b-04`](prompts/prompt-m6b-04.md) | Fake-media e2e + interviewer flip | **tag v2.1.0** |

### Cross-cutting · all along

Boards are drafted one milestone ahead and frozen by their own merge (D40). They are the only prompts that
produce no product code; each row names the prompts that build those screens.

| # | Design prompt | Boards | Week | Built by |
|---|---|---|---|---|
| 2 | [`ds-m1-01`](prompts/prompt-ds-m1-01.md) | Boards AB01–AB03: coach states, course nav, revision v2 | W1 | [`m1-03`](prompts/prompt-m1-03.md), [`m1-06`](prompts/prompt-m1-06.md), [`m1-10`](prompts/prompt-m1-10.md), [`m1-07`](prompts/prompt-m1-07.md) |
| 6 | [`ds-m2-01`](prompts/prompt-ds-m2-01.md) | Boards AB04★ Touch, AB05 catalog/agenda, AB06 profile, AB22 | W1 | [`m2-01`](prompts/prompt-m2-01.md), [`m2-03`](prompts/prompt-m2-03.md), [`m2-04`](prompts/prompt-m2-04.md) |
| 7 | [`ds-l-01`](prompts/prompt-ds-l-01.md) | Boards AB19★ invite acceptance, AB20 notice, AB21 erase | W1 | [`l-02`](prompts/prompt-l-02.md), [`l-05`](prompts/prompt-l-05.md) |
| 28 | [`ds-m3-01`](prompts/prompt-ds-m3-01.md) | Boards AB07★ workspace, AB08 results dock, AB11 badges | W4 | [`m3-11`](prompts/prompt-m3-11.md), [`m3-12`](prompts/prompt-m3-12.md), [`m5-01`](prompts/prompt-m5-01.md) |
| 35 | [`ds-m3-02`](prompts/prompt-ds-m3-02.md) | Boards AB09 Problems, AB10 Arena, AB12 progress deltas | W5 | [`m3-12`](prompts/prompt-m3-12.md), [`m3-13`](prompts/prompt-m3-13.md) |
| 45 | [`ds-p-01`](prompts/prompt-ds-p-01.md) | Boards AB14 quiz, AB15 race view; AB02/AB05 full fidelity | W7 | [`p-02`](prompts/prompt-p-02.md), [`p-03`](prompts/prompt-p-03.md) |
| 51 | [`ds-m4-01`](prompts/prompt-ds-m4-01.md) | Boards AB16★ AI suggestion/dispute, AB17, AB18 | W7 | [`m4-06`](prompts/prompt-m4-06.md) |
| 72 | [`ds-m6a-01`](prompts/prompt-ds-m6a-01.md) | Accept ADR-0032 + boards AB13, AB24, AB25 | W11 | [`m6a-05`](prompts/prompt-m6a-05.md) |
| 73 | [`ds-m6a-02`](prompts/prompt-ds-m6a-02.md) | Boards AB26 grace/pause, AB27 debrief, AB28 a11y | W11 | [`m6a-06`](prompts/prompt-m6a-06.md) |
| 80 | [`ds-m6b-01`](prompts/prompt-ds-m6b-01.md) | Boards AB29 voice pre-flight, AB30★ voice HUD | W11 | [`m6b-03`](prompts/prompt-m6b-03.md) |

### Later · H2 2027 · v2.2

```mermaid
flowchart TB
    subgraph later_g["⏭️ Later"]
        direction TB
        m501["<b>86 · m5-01</b><br/>DSA evaluator-only"]:::build
        m501_tag(["≥ v2.2.0"]):::tag
        m6c01["<b>87 · m6c-01</b><br/>v2.2 outline: extras"]:::outline
        m6c02["<b>88 · m6c-02</b><br/>v2.2 outline: fairness"]:::outline
        m6c03["<b>89 · m6c-03</b><br/>v2.2 outline: Safari, canvas"]:::outline
        m501 --> m501_tag
        m501_tag --> m6c01
        m6c01 --> m6c02
        m6c02 --> m6c03
    end
    classDef build fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef infra fill:#1f2937,stroke:#f59e0b,color:#e5e7eb
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
    classDef spike fill:#1f2937,stroke:#f472b6,color:#e5e7eb,stroke-dasharray:4 3
    classDef outline fill:#111827,stroke:#6b7280,color:#9ca3af,stroke-dasharray:2 2
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef shared fill:#111827,stroke:#9ca3af,color:#9ca3af,stroke-dasharray:3 3
```

**⏭️ Later** — DSA evaluator-only (M5) once every item is packed; v2.2 interviewer extras

| # | Prompt | Brings | Ships |
|---|---|---|---|
| 86 | [`m5-01`](prompts/prompt-m5-01.md) | DSA evaluator-only flip (once every item is packed) | **tag ≥ v2.2.0** |
| 87 | [`m6c-01`](sprints/sprint-m6c-01.md) | v2.2 outline: show-your-work photo, read-aloud, local download | outline only |
| 88 | [`m6c-02`](sprints/sprint-m6c-02.md) | v2.2 outline: fairness check → AI voice Communication | outline only |
| 89 | [`m6c-03`](sprints/sprint-m6c-03.md) | v2.2 outline: Safari, voice-lite, canvas board (SD course) | outline only |

## 5. The run list, week by week

The numbered order to launch prompts. Lanes in the same week can run side by side; the 🚩 column is the tag a
prompt cuts.

### W0 · Spike weekend (D41: spikes first) (Sep 25 – 26)

Lanes — **spike**: `spk-01`, `spk-02`, `spk-03`, `spk-04`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 16 | [`spk-01`](prompts/prompt-spk-01.md) | 🧪 Sandbox & runner | Spike: sandbox mechanism P0–P2 (throwaway) |  |
| 17 | [`spk-02`](prompts/prompt-spk-02.md) | 🧪 Sandbox & runner | Spike: amd64 replay + eval-pack image volume (throwaway) |  |
| 18 | [`spk-03`](prompts/prompt-spk-03.md) | ✨ Platform AI | Spike: WIF credential for platform AI (throwaway) |  |
| 71 | [`spk-04`](prompts/prompt-spk-04.md) | 🎙️ Interviewer: text (M6a) | Spike S6: voice-shell bake-off (owner present) |  |

### W1 · Week 1 (Sep 28 – Oct 2)

Lanes — **infra**: `mi-01`, `mi-02`, `mi-07` · **design**: `ds-m1-01`, `ds-m2-01`, `ds-l-01` · **product**: `m1-01`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 1 | [`mi-01`](prompts/prompt-mi-01.md) | 🛡️ Cluster guardrails & fences | Data-loss guards on DB + NATS, chart 0.3.0 knobs (no pod rolls) |  |
| 2 | [`ds-m1-01`](prompts/prompt-ds-m1-01.md) | 🎨 Design boards | Boards AB01–AB03: coach states, course nav, revision v2 |  |
| 3 | [`mi-02`](prompts/prompt-mi-02.md) | 🛡️ Cluster guardrails & fences | host-verify --cluster: on-demand cluster checks (the only ops tool) |  |
| 4 | [`m1-01`](prompts/prompt-m1-01.md) | 🧭 Curriculum spine (multi-course) | Prod-parity compose, course manifest + golden, frozen item schema |  |
| 5 | [`mi-07`](prompts/prompt-mi-07.md) | 📦 Eval packs & authoring tooling | Private eval-pack repo, machine user, GHCR, pull secret |  |
| 6 | [`ds-m2-01`](prompts/prompt-ds-m2-01.md) | 🎨 Design boards | Boards AB04★ Touch, AB05 catalog/agenda, AB06 profile, AB22 |  |
| 7 | [`ds-l-01`](prompts/prompt-ds-l-01.md) | 🎨 Design boards | Boards AB19★ invite acceptance, AB20 notice, AB21 erase |  |

### W2 · Week 2 (Oct 5 – 9)

Lanes — **product**: `m1-09`, `mi-05`, `m1-02` · **content**: `m3-01`, `m3-02` · **infra**: `mi-14`, `mi-03`, `mi-04`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 8 | [`m1-09`](prompts/prompt-m1-09.md) | 🧭 Curriculum spine (multi-course) | Curriculum converter, loader + guards, expand migration, content CI |  |
| 9 | [`m3-01`](prompts/prompt-m3-01.md) | 📦 Eval packs & authoring tooling | Authoring tooling: packlint, contract_hash, pre-push fingerprint hook |  |
| 10 | [`mi-05`](prompts/prompt-mi-05.md) | 📨 NATS auth & event plumbing | N0 (dark): NATS topology, dead letters, identity on NATS |  |
| 11 | [`m1-02`](prompts/prompt-m1-02.md) | 🧭 Curriculum spine (multi-course) | path_slug everywhere + v2 envelope consumers | v1.6.0 |
| 12 | [`mi-14`](prompts/prompt-mi-14.md) | 🧪 Sandbox & runner | Empty default-deny xlearn-runner namespace + admission policy |  |
| 13 | [`mi-03`](prompts/prompt-mi-03.md) | 🛡️ Cluster guardrails & fences | NetworkPolicy fences: databases, messaging, xlearn ingress |  |
| 14 | [`m3-02`](prompts/prompt-m3-02.md) | 📦 Eval packs & authoring tooling | Eval-pack pipeline: validation gates, data image, fixture pack |  |
| 15 | [`mi-04`](prompts/prompt-mi-04.md) | 🛡️ Cluster guardrails & fences | Admin consoles move to ops.sujaykumar.dev |  |

### W3 · Week 3 · spike week (Oct 12 – 16)

Lanes — **infra**: `mi-06` · **product**: `m1-03`, `m1-04`, `m1-10`, `m1-05`, `m1-06`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 19 | [`mi-06`](prompts/prompt-mi-06.md) | 📨 NATS auth & event plumbing | NATS auth live: nkey users + ACLs, legacy closed (N1–N3) |  |
| 20 | [`m1-03`](prompts/prompt-m1-03.md) | 🧭 Curriculum spine (multi-course) | Course resolution end to end; DSA stays pixel-identical |  |
| 21 | [`m1-04`](prompts/prompt-m1-04.md) | 🔐 Security floor | Roles in DB, revocable sessions, admin CLI, CSP, DEV_AUTH guard |  |
| 22 | [`m1-10`](prompts/prompt-m1-10.md) | 💬 Coach v2 | Coach keys: per-feature defaults, catalog, AEAD keyring, typed errors |  |
| 23 | [`m1-05`](prompts/prompt-m1-05.md) | 🔐 Security floor | Typed 429/413 limits + public-profile floor (count-only mocks) |  |
| 24 | [`m1-06`](prompts/prompt-m1-06.md) | 🔐 Security floor | withhold() on every route, safe Markdown, revision v2 |  |

### W4 · Week 4 (Oct 17 – 23)

Lanes — **infra**: `mi-08`, `mi-09` · **product**: `m1-07` · **design**: `ds-m3-01`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 25 | [`mi-08`](prompts/prompt-mi-08.md) | 🛡️ Cluster guardrails & fences | Limit hygiene (≈ −3 GiB), PSA labels, SA tokens off |  |
| 26 | [`mi-09`](prompts/prompt-mi-09.md) | 🧪 Sandbox & runner | Host-window prep: sandbox host block, kubelet reservation (Sat Oct 24) |  |
| 27 | [`m1-07`](prompts/prompt-m1-07.md) | 💬 Coach v2 | Coach assist capture, mode gate, BYO caps | v1.7.0 |
| 28 | [`ds-m3-01`](prompts/prompt-ds-m3-01.md) | 🎨 Design boards | Boards AB07★ workspace, AB08 results dock, AB11 badges |  |

### W5 · Late Oct · host window Sat Oct 24 (Oct 24 – Nov 1)

Lanes — **product**: `m1-08`, `m3-03`, `m3-04`, `m3-15`, `m2-01`, `m2-02`, `m2-03`, `m2-04` · **design**: `ds-m3-02`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 29 | [`m1-08`](prompts/prompt-m1-08.md) | 🧭 Curriculum spine (multi-course) | M1 contract: drop the v1 columns (snapshot first) | v1.8.0 |
| 30 | [`m3-03`](prompts/prompt-m3-03.md) | 🧪 Sandbox & runner | Runner core: supervisor, jail, per-case cgroups, job API |  |
| 31 | [`m3-04`](prompts/prompt-m3-04.md) | 🧪 Sandbox & runner | Runner profiles Go/C++/Python + harness codecs |  |
| 32 | [`m3-15`](prompts/prompt-m3-15.md) | 🧪 Sandbox & runner | Reproducible runner image + acceptance suite | runner-v1.0.0 |
| 33 | [`m2-01`](prompts/prompt-m2-01.md) | 🔁 Touch engine & projections | Touches become real attempts + review consumer |  |
| 34 | [`m2-02`](prompts/prompt-m2-02.md) | 🔁 Touch engine & projections | M2 consumers + projections v2 (producers idle) | v1.9.0 |
| 35 | [`ds-m3-02`](prompts/prompt-ds-m3-02.md) | 🎨 Design boards | Boards AB09 Problems, AB10 Arena, AB12 progress deltas |  |
| 36 | [`m2-03`](prompts/prompt-m2-03.md) | 🌐 Public profile v2 | public-read, /public/stats, visibility toggles, profile v2 |  |
| 37 | [`m2-04`](prompts/prompt-m2-04.md) | 🔁 Touch engine & projections | Touch UI ★ + cross-course Today/agenda in minutes |  |

### W6 · Early Nov (Nov 2 – 8)

Lanes — **product**: `m2-05`, `l-01`, `l-02`, `m3-05`, `m3-06` · **infra**: `mi-10`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 38 | [`m2-05`](prompts/prompt-m2-05.md) | 🔁 Touch engine & projections | Producers on, backfill, replay, D2 ladder rule | v1.10.0 |
| 39 | [`l-01`](prompts/prompt-l-01.md) | 🚪 Learner gate (built now, opened in v3) | Erase consumers in every service | v1.11.0 |
| 40 | [`l-02`](prompts/prompt-l-02.md) | 🚪 Learner gate (built now, opened in v3) | Erase producer + DELETE /api/me (testers only) | v1.12.0 |
| 41 | [`mi-10`](prompts/prompt-mi-10.md) | 🧪 Sandbox & runner | runner-v1.0.0 dark on production + acceptance suite |  |
| 42 | [`m3-05`](prompts/prompt-m3-05.md) | ⚖️ Judge service | judge service: skeleton, schema, pack loader, erase consumer |  |
| 43 | [`m3-06`](prompts/prompt-m3-06.md) | ⚖️ Judge service | judge core: queue, runner lane, graders, contexts |  |

### W7 · Mid Nov (Nov 9 – 20)

Lanes — **product**: `m3-14`, `m3-07`, `l-03`, `m3-08`, `m3-09`, `m3-10` · **design**: `ds-p-01`, `ds-m4-01` · **infra**: `mi-11`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 44 | [`m3-14`](prompts/prompt-m3-14.md) | ⚖️ Judge service | judge admission control, learner API, drafts, arena history |  |
| 45 | [`ds-p-01`](prompts/prompt-ds-p-01.md) | 🎨 Design boards | Boards AB14 quiz, AB15 race view; AB02/AB05 full fidelity |  |
| 46 | [`m3-07`](prompts/prompt-m3-07.md) | ⚖️ Judge service | judge on prod (dark): evalpack v1.0.0, ACL, HelmRelease | v1.13.0 |
| 47 | [`l-03`](prompts/prompt-l-03.md) | 🚪 Learner gate (built now, opened in v3) | Invite admission backend (inert while signup is closed) |  |
| 48 | [`m3-08`](prompts/prompt-m3-08.md) | ⚖️ Judge service | practice consumes judge evidence; the 45:00 method |  |
| 49 | [`m3-09`](prompts/prompt-m3-09.md) | ⚖️ Judge service | Gateway judge BFF, DTO deny-list, typed limits |  |
| 50 | [`m3-10`](prompts/prompt-m3-10.md) | ⚖️ Judge service | review + assessment on judge signals (pre-fill, judge-checked %) |  |
| 51 | [`ds-m4-01`](prompts/prompt-ds-m4-01.md) | 🎨 Design boards | Boards AB16★ AI suggestion/dispute, AB17, AB18 |  |
| 52 | [`mi-11`](prompts/prompt-mi-11.md) | 🛡️ Cluster guardrails & fences | Track B finish: egress, PG connection limits, Renovate, N4 |  |

### W8 · Late Nov (Nov 21 – 30)

Lanes — **product**: `m3-11`, `m3-12`, `m3-13`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 53 | [`m3-11`](prompts/prompt-m3-11.md) | 🖥️ Judge UI | Workspace-Code UI ★ + lazy CodeMirror |  |
| 54 | [`m3-12`](prompts/prompt-m3-12.md) | 🖥️ Judge UI | Results dock, Problems, Arena, degradation badges |  |
| 55 | [`m3-13`](prompts/prompt-m3-13.md) | 🖥️ Judge UI | Progress deltas + M3 exit tests; judge on for the owner | v1.14.0 |

### W9 · December (Dec 1 – 23)

Lanes — **infra**: `mi-12` · **product**: `p-01`, `p-02`, `p-03`, `m4-01`, `m4-02`, `m4-03`, `m4-04`, `m4-05`, `m4-06`, `m4-07`, `l-05`, `l-04`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 56 | [`mi-12`](prompts/prompt-mi-12.md) | ✨ Platform AI | M4 gates: judge 443 egress, LLM secret, provider runbook |  |
| 57 | [`p-01`](prompts/prompt-p-01.md) | 🧵 Pilot course: go-concurrency | go-race runner profile | runner-v1.1.0 |
| 58 | [`p-02`](prompts/prompt-p-02.md) | 🧵 Pilot course: go-concurrency | go-concurrency course as manifest (preview) + multi-course nav |  |
| 59 | [`p-03`](prompts/prompt-p-03.md) | 🧵 Pilot course: go-concurrency | Quiz + race verdict UI | v1.15.0 |
| 60 | [`m4-01`](prompts/prompt-m4-01.md) | ✨ Platform AI | Shared LLM layer + WIF auth + retention policy |  |
| 61 | [`m4-02`](prompts/prompt-m4-02.md) | ✨ Platform AI | judge AI engine: Scorer, ledger, llm lane, caps |  |
| 62 | [`m4-03`](prompts/prompt-m4-03.md) | ✨ Platform AI | Analyzer on concluded attempts + acceptance-set harness |  |
| 63 | [`m4-04`](prompts/prompt-m4-04.md) | ✨ Platform AI | Provisional grades, dispute, re-grade, honor claims |  |
| 64 | [`m4-05`](prompts/prompt-m4-05.md) | ✨ Platform AI | AI allowance + consents |  |
| 65 | [`m4-06`](prompts/prompt-m4-06.md) | ✨ Platform AI | AI UI ★: suggestion/dispute, pointer notes, allowance |  |
| 66 | [`m4-07`](prompts/prompt-m4-07.md) | ✨ Platform AI | Canary + caps sizing; platform AI on for the owner | v1.16.0 |
| 67 | [`l-05`](prompts/prompt-l-05.md) | 🚪 Learner gate (built now, opened in v3) | Invite front door: acceptance step ★, privacy notice, AI consents |  |
| 68 | [`l-04`](prompts/prompt-l-04.md) | 🚪 Learner gate (built now, opened in v3) | Web erase for non-owners + invite rehearsal on prod | v1.17.0 |

### W10 · GA (late Dec – Jan)

Lanes — **product**: `ga-01`, `ga-02`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 69 | [`ga-01`](prompts/prompt-ga-01.md) | 🏁 v2.0 GA | GA PR: .release-line = 2, default flips, rc rehearsal |  |
| 70 | [`ga-02`](prompts/prompt-ga-02.md) | 🏁 v2.0 GA | Widen ranges, snapshot, tag, verify | v2.0.0 |

### W11 · v2.1 (Q1 2027)

Lanes — **design**: `ds-m6a-01`, `ds-m6a-02`, `ds-m6b-01` · **product**: `m6a-01`, `m6a-02`, `m6a-03`, `m6a-04`, `m6a-05`, `m6a-06`, `m6b-01`, `m6b-02`, `m6b-03`, `m6b-04` · **infra**: `mi-13`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 72 | [`ds-m6a-01`](prompts/prompt-ds-m6a-01.md) | 🎨 Design boards | Accept ADR-0032 + boards AB13, AB24, AB25 |  |
| 73 | [`ds-m6a-02`](prompts/prompt-ds-m6a-02.md) | 🎨 Design boards | Boards AB26 grace/pause, AB27 debrief, AB28 a11y |  |
| 74 | [`m6a-01`](prompts/prompt-m6a-01.md) | 🎙️ Interviewer: text (M6a) | Interview core: state machine, failsafes, caps |  |
| 75 | [`m6a-02`](prompts/prompt-m6a-02.md) | 🎙️ Interviewer: text (M6a) | Text brain on the learner's key + replay tests |  |
| 76 | [`m6a-03`](prompts/prompt-m6a-03.md) | 🎙️ Interviewer: text (M6a) | Mock scoring: proposal → accept → ScoreMock once |  |
| 77 | [`m6a-04`](prompts/prompt-m6a-04.md) | 🎙️ Interviewer: text (M6a) | Coding rounds via judge mock + editor interview mode |  |
| 78 | [`m6a-05`](prompts/prompt-m6a-05.md) | 🎙️ Interviewer: text (M6a) | Interviewer UI part 1: setup, consent, text HUD |  |
| 79 | [`m6a-06`](prompts/prompt-m6a-06.md) | 🎙️ Interviewer: text (M6a) | Interviewer UI part 2 + M6a exit (dark) | v2.0.x |
| 80 | [`ds-m6b-01`](prompts/prompt-ds-m6b-01.md) | 🎨 Design boards | Boards AB29 voice pre-flight, AB30★ voice HUD |  |
| 81 | [`mi-13`](prompts/prompt-mi-13.md) | 🔊 Interviewer: voice (M6b) | Voice platform gates: Permissions-Policy, SDP route, egress | v2.0.x |
| 82 | [`m6b-01`](prompts/prompt-m6b-01.md) | 🔊 Interviewer: voice (M6b) | Voice shell + SDP broker + sideband (no media on the node) |  |
| 83 | [`m6b-02`](prompts/prompt-m6b-02.md) | 🔊 Interviewer: voice (M6b) | Voice robustness: lease, drain, cost cap, push-to-talk |  |
| 84 | [`m6b-03`](prompts/prompt-m6b-03.md) | 🔊 Interviewer: voice (M6b) | Voice UI + browser/EU gates (dark) | v2.0.x |
| 85 | [`m6b-04`](prompts/prompt-m6b-04.md) | 🔊 Interviewer: voice (M6b) | Fake-media e2e + interviewer flip | v2.1.0 |

### W12 · Later (H2 2027 · v2.2)

Lanes — **product**: `m5-01` · **outline**: `m6c-01`, `m6c-02`, `m6c-03`

| # | Prompt | Workstream | Brings | 🚩 |
|---|---|---|---|---|
| 86 | [`m5-01`](prompts/prompt-m5-01.md) | ⏭️ Later | DSA evaluator-only flip (once every item is packed) | ≥ v2.2.0 |
| 87 | [`m6c-01`](sprints/sprint-m6c-01.md) | ⏭️ Later | v2.2 outline: show-your-work photo, read-aloud, local download |  |
| 88 | [`m6c-02`](sprints/sprint-m6c-02.md) | ⏭️ Later | v2.2 outline: fairness check → AI voice Communication |  |
| 89 | [`m6c-03`](sprints/sprint-m6c-03.md) | ⏭️ Later | v2.2 outline: Safari, voice-lite, canvas board (SD course) |  |

---

_Generated by [`tools/gen_execution_order.py`](tools/gen_execution_order.py) from
[`tools/execution_order_data.py`](tools/execution_order_data.py). If a sprint is added, split or re-ordered,
update [build-plan.md](build-plan.md) first, then the data file, then re-run the generator._
