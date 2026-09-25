#!/usr/bin/env python3
"""Generate docs/v2/execution-order.md (mermaid diagrams + run list) from execution_order_data.py.

Usage (repo root):  python3 docs/v2/tools/gen_execution_order.py
Edit execution_order_data.py (workstreams, weeks, one-liners, tags) after build-plan.md changes, then re-run.
"""
import json, sys, os, datetime as dt
sys.path.insert(0, os.path.dirname(__file__))
from execution_order_data import FAMILIES, CLUBS, WAVES, SPRINTS, TAGS, DESIGN_FEEDS, CLUB_EDGES

OUT = sys.argv[1] if len(sys.argv) > 1 else os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "execution-order.md")

# Short node labels (≤ ~34 chars) for diagrams
SHORT = {
 "mi-01":"DB/NATS prune guards + chart 0.3.0","ds-m1-01":"M1 boards AB01–03","mi-02":"host-verify checks",
 "m1-01":"manifest + item schema","mi-07":"eval-pack repo + GHCR","ds-m2-01":"M2 boards AB04★ AB05 AB06 AB22",
 "ds-l-01":"gate boards AB19★ AB20 AB21","m1-09":"converter, loader, expand","m3-01":"packlint + fingerprint hook",
 "mi-05":"N0: topology, dead letters","m1-02":"path_slug + v2 envelope","mi-14":"runner namespace, default-deny",
 "mi-03":"NetworkPolicy fences","m3-02":"eval-pack pipeline","mi-04":"consoles → ops.",
 "spk-01":"spike: sandbox P0–P2","spk-02":"spike: amd64 + image volume","spk-03":"spike: WIF credential",
 "mi-06":"NATS auth N1–N3","m1-03":"course resolution","m1-04":"roles, sessions, admin CLI",
 "m1-10":"coach keys + providers","m1-05":"limits + public floor","m1-06":"withhold() + Markdown",
 "mi-08":"limit hygiene, PSA","mi-09":"host-window prep (Oct 24)","m1-07":"coach assist + caps",
 "ds-m3-01":"M3 boards AB07★ AB08 AB11","m1-08":"M1 contract","m3-03":"runner core: jail, cgroups",
 "m3-04":"Go/C++/Python profiles","m3-15":"runner release","m2-01":"touch attempts",
 "m2-02":"consumers + projections v2","ds-m3-02":"M3 boards AB09 AB10 AB12","m2-03":"public-read + visibility",
 "m2-04":"Touch UI★ + Today","m2-05":"producers on + D2","l-01":"erase consumers","l-02":"erase producer + UI",
 "mi-10":"runner dark on prod","m3-05":"judge skeleton + loader","m3-06":"queue, graders, contexts",
 "m3-14":"admission + learner API","ds-p-01":"pilot boards AB14 AB15","m3-07":"judge on prod (dark)",
 "l-03":"invite backend (inert)","m3-08":"practice ← judge; 45:00","m3-09":"gateway judge BFF",
 "m3-10":"review/assessment on judge","ds-m4-01":"AI boards AB16★ AB17 AB18","mi-11":"egress, PG limits, N4",
 "m3-11":"Workspace-Code UI★","m3-12":"results dock, Problems, Arena","m3-13":"progress deltas + exit",
 "mi-12":"M4 gates: egress, secret","p-01":"go-race profile","p-02":"go-concurrency course",
 "p-03":"quiz + race verdict UI","m4-01":"LLM layer + WIF","m4-02":"Scorer, ledger, caps",
 "m4-03":"analyzer","m4-04":"provisional grades, dispute","m4-05":"allowance + consents",
 "m4-06":"AI UI★","m4-07":"canary + caps sizing","l-05":"acceptance★ + notice",
 "l-04":"web erase + rehearsal","ga-01":"GA PR + rc rehearsal","ga-02":"cut v2.0.0",
 "spk-04":"spike S6: voice shell","ds-m6a-01":"ADR-0032 + AB13 AB24 AB25","ds-m6a-02":"boards AB26 AB27 AB28",
 "m6a-01":"interview core + failsafes","m6a-02":"text brain","m6a-03":"scoring: accept → ScoreMock",
 "m6a-04":"coding rounds","m6a-05":"interviewer UI 1","m6a-06":"interviewer UI 2 + exit",
 "ds-m6b-01":"voice boards AB29 AB30★","mi-13":"voice platform gates","m6b-01":"voice shell + SDP broker",
 "m6b-02":"voice robustness + cost cap","m6b-03":"voice UI + gates","m6b-04":"voice e2e + flip",
 "m5-01":"DSA evaluator-only","m6c-01":"v2.2 outline: extras","m6c-02":"v2.2 outline: fairness",
 "m6c-03":"v2.2 outline: Safari, canvas",
}

S = {s[1]: dict(order=s[0], id=s[1], club=s[2], wave=s[3], brings=s[4], cuts=s[5], lane=s[6]) for s in SPRINTS}
CL = {c[0]: dict(id=c[0], fam=c[1], emoji=c[2], name=c[3], brings=c[4], also=c[5]) for c in CLUBS}
FAM = {f[0]: dict(id=f[0], name=f[1], when=f[2]) for f in FAMILIES}
WV = {w[0]: dict(id=w[0], name=w[1], dates=w[2]) for w in WAVES}
order = sorted(S.values(), key=lambda s: s["order"])
assert all(i in SHORT for i in S), [i for i in S if i not in SHORT]

def nid(i): return i.replace("-", "")
def members(c): return [s for s in order if s["club"] == c]
def link(i): return f"[`{i}`](prompts/prompt-{i}.md)" if S[i]["lane"] != "outline" else f"[`{i}`](sprints/sprint-{i}.md)"
def kindclass(s):
    if s["lane"] == "spike": return "spike"
    if s["lane"] == "design": return "design"
    if s["lane"] == "outline": return "outline"
    if s["lane"] == "infra": return "infra"
    return "build"

CLASSDEFS = """    classDef build fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef infra fill:#1f2937,stroke:#f59e0b,color:#e5e7eb
    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb
    classDef spike fill:#1f2937,stroke:#f472b6,color:#e5e7eb,stroke-dasharray:4 3
    classDef outline fill:#111827,stroke:#6b7280,color:#9ca3af,stroke-dasharray:2 2
    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef shared fill:#111827,stroke:#9ca3af,color:#9ca3af,stroke-dasharray:3 3"""

W = []
w = W.append

# ------------------------------------------------------------------ header
w("# xLearn v2 — prompt execution order")
w("")
w("> **One page to see what runs when, and what each prompt brings.** The 89 sprint prompts are clubbed into")
w("> **19 workstreams** in **7 families**, so work that is split across several prompts reads as one thing.")
w("> Order numbers match [build-plan.md](build-plan.md) (the static plan); live state is in [status.md](status.md);")
w("> each prompt is self-contained in [`prompts/`](prompts/). Dates are **indicative** (BP4 calendar, 2026-09-25).")
w("")
w("**How to read it**")
w("")
w("- **Run prompts in numbered order.** Inside a week, prompts in different **lanes** (infra · product · design ·")
w("  content/spike) can run as parallel sessions (up to about three), as long as their entry gates pass. Check")
w("  peers' open PRs first; parallel prompts in the same service collide on migrations.")
w("- **Every prompt ends shipped** (D40): land-and-sync, owner approval pre-granted. A **tag** prompt deploys; a")
w("  merge-only prompt ships dark in the next tag. Owner-only steps are listed under each prompt's")
w("  `## Before you launch (owner)`.")
w("- **Legend** (diagrams): blue = build · amber = infra · violet = design boards · pink dashed = spike (throwaway)")
w("  · green pill = a release tag · grey dashed = shared with another workstream / v2.2 outline.")
w("")

# ------------------------------------------------------------------ 1. big picture
w("## 1. The big picture")
w("")
w("Seven families of work, in the order they land. Arrows are the hand-offs that matter; design boards feed every")
w("family that has screens, always one milestone ahead.")
w("")
FAMNODE = {"found": "F", "engine": "E", "judgefam": "J", "aifam": "A", "gatefam": "G", "ivfam": "I", "laterfam": "L"}
FAMICON = {"found": "🧱", "engine": "⚙️", "judgefam": "⚖️", "aifam": "✨", "gatefam": "🚪", "ivfam": "🎙️", "laterfam": "⏭️"}
w("```mermaid")
w("flowchart TB")
for f in FAMILIES:
    fid = f[0]
    if fid == "cross": continue
    tot = sum(len(members(c[0])) for c in CLUBS if c[1] == fid)
    ends = [s["cuts"] for s in order if CL[s["club"]]["fam"] == fid and s["cuts"] and not s["cuts"].startswith("≥")]
    last = f"<br/>→ {ends[-1]}" if ends else ""
    w(f'    {FAMNODE[fid]}["<b>{FAMICON[fid]} {f[1]}</b><br/>{tot} prompts · {f[2]}{last}"]')
w(f'    D["<b>{CL["design"]["emoji"]} Design boards</b><br/>{len(members("design"))} prompts · one milestone ahead"]:::design')
w("    F --> E --> J --> A --> G --> I --> L")
w("    F --> J")
w("    J --> G")
w("    D -.-> E")
w("    D -.-> J")
w("    D -.-> G")
w("    D -.-> I")
w("    classDef design fill:#1f2937,stroke:#c084fc,color:#e5e7eb")
w("```")
w("")
w("Families overlap in time (the judge's runner and eval packs start in October, next to the foundations): see the")
w("timeline in §2. The workstreams inside each family:")
w("")
w("| Family | Workstream | Prompts (in order) | What it brings |")
w("|---|---|---|---|")
for f in FAMILIES:
    for c in CLUBS:
        if c[1] != f[0]: continue
        ids = " ".join(link(s["id"]) for s in members(c[0]))
        also = f" · shares {', '.join('`'+x+'`' for x in c[5])}" if c[5] else ""
        w(f"| {f[1]} | {c[2]} **{c[3]}** | {ids} | {c[4]}{also} |")
w("")

# ------------------------------------------------------------------ 2. timeline (gantt)
WDATES = {
 "W1": ("2026-09-25", "2026-10-02"), "W2": ("2026-10-05", "2026-10-09"), "W3": ("2026-10-12", "2026-10-16"),
 "W4": ("2026-10-17", "2026-10-23"), "W5": ("2026-10-24", "2026-11-01"), "W6": ("2026-11-02", "2026-11-08"),
 "W7": ("2026-11-09", "2026-11-20"), "W8": ("2026-11-21", "2026-11-30"), "W9": ("2026-12-01", "2026-12-23"),
 "W10": ("2026-12-28", "2027-01-15"),
}
WIDX = [x[0] for x in WAVES]
w("## 2. When: the v2.0 timeline")
w("")
w("One row per workstream, **Sep 25 → mid-Jan (v2.0)**. A bar is a run of consecutive weeks with that workstream's")
w("prompts in flight; the number on it is how many prompts. Diamonds are owner calendar dates and key tags. v2.1")
w("(Q1 2027) is the interviewer: text, then voice, after GA.")
w("")
w("```mermaid")
w("---")
w("displayMode: compact")
w("config:")
w("  gantt:")
w("    leftPadding: 230")
w("    fontSize: 12")
w("    sectionFontSize: 12")
w("---")
w("gantt")
w("    title xLearn v2.0 — workstreams over time (indicative)")
w("    dateFormat YYYY-MM-DD")
w("    axisFormat %d %b")
w("    tickInterval 2week")
w("    section Owner calendar")
w("    H0 :milestone, ev0, 2026-09-25, 0d")
w("    spikes :crit, ev1, 2026-10-12, 5d")
w("    window :milestone, ev2, 2026-10-24, 0d")
k = 0
for c in CLUBS:
    if c[0] in ("later", "ivtext", "ivvoice"): continue
    ms = [s for s in members(c[0]) if s["wave"] in WDATES]
    wv = sorted({WIDX.index(s["wave"]) for s in ms})
    if not wv: continue
    runs, cur = [], [wv[0]]
    for x in wv[1:]:
        if x == cur[-1] + 1: cur.append(x)
        else: runs.append(cur); cur = [x]
    runs.append(cur)
    name = c[3].replace("&", "and").replace(":", " -")
    w(f"    section {name}")
    for r in runs:
        a2 = WDATES[WIDX[r[0]]][0]; b2 = WDATES[WIDX[r[-1]]][1]
        n = len([s for s in ms if WIDX.index(s["wave"]) in r])
        k += 1
        w(f"    {n} :t{k}, {a2}, {(dt.date.fromisoformat(b2) + dt.timedelta(days=1)).isoformat()}")
w("    section Key tags")
for tag in ["v1.6.0", "v1.10.0", "v1.14.0", "v1.17.0", "v2.0.0"]:
    sid = [t for t in TAGS if t[0] == tag][0][1]
    d = WDATES[S[sid]["wave"]][1]
    k += 1
    w(f"    {tag} :milestone, t{k}, {d}, 0d")
w("```")
w("")

# ------------------------------------------------------------------ 3. release train
TRAIN = [
 ("M1 · October", [("v1.6.0", "M1a expand", "m1-02"), ("v1.7.0", "M1b: routing, security, coach", "m1-07"), ("v1.8.0", "M1c contract", "m1-08")]),
 ("M2 + erase · late Oct – early Nov", [("v1.9.0", "M2 consumers", "m2-02"), ("v1.10.0", "Touch UI, D2, public-read", "m2-05"), ("v1.11.0", "erase consumers", "l-01"), ("v1.12.0", "erase producer", "l-02")]),
 ("Runner + judge · Oct – Nov", [("runner-v1.0.0", "runner: Go/C++/Python", "m3-15"), ("v1.13.0", "judge dark + evalpack 1.0", "m3-07"), ("v1.14.0", "judge on for the owner", "m3-13")]),
 ("Pilot, AI, gate · December", [("runner-v1.1.0", "go-race profile", "p-01"), ("v1.15.0", "pilot course (preview)", "p-03"), ("v1.16.0", "platform AI on", "m4-07"), ("v1.17.0", "invite front door + erase", "l-04")]),
 ("GA · late Dec – Jan", [("v2.0.0", "owner-facing GA", "ga-02")]),
 ("v2.1 · Q1 2027", [("v2.0.x", "interviewer patches, dark", "m6a-06 · mi-13 · m6b-03"), ("v2.1.0", "interviewer GA", "m6b-04")]),
]
w("## 3. The release train")
w("")
w("Only a tag deploys (1.x minors until GA; ADR-0034). Every prompt between two tags merges dark and ships with the")
w("next one; infra-only prompts deploy through their own GitOps PRs. Each pill shows the tag, what it carries and")
w("the prompt that cuts it.")
w("")
w("```mermaid")
w("flowchart TB")
for pi, (pname, tags) in enumerate(TRAIN):
    w(f'    subgraph ph{pi}["{pname}"]')
    w("        direction LR")
    ids = []
    for ti, (tag, what, sid) in enumerate(tags):
        n = f"r{pi}_{ti}"; ids.append(n)
        w(f'        {n}(["<b>{tag}</b><br/>{what}<br/><i>{sid}</i>"]):::tag')
    for x, y in zip(ids, ids[1:]):
        w(f"        {x} --> {y}")
    w("    end")
for pi in range(len(TRAIN) - 1):
    w(f"    ph{pi} --> ph{pi+1}")
w("    classDef tag fill:#064e3b,stroke:#34d399,color:#ecfdf5")
w("```")
w("")

# ------------------------------------------------------------------ 4. families
w("## 4. What each workstream brings, prompt by prompt")
w("")
w("One diagram per family: each column is a workstream, its prompts top to bottom in run order (`#` = order")
w("number). A green pill is the tag the prompt above it cuts. A grey dashed box at the bottom of a column belongs")
w("to another workstream but carries part of this one's work.")
w("")
for f in FAMILIES:
    fid = f[0]
    clubs = [c for c in CLUBS if c[1] == fid]
    if not clubs: continue
    w(f"### {f[1]} · {f[2]}")
    w("")
    if fid == "cross":
        w("Boards are drafted one milestone ahead and frozen by their own merge (D40). They are the only prompts that")
        w("produce no product code; each row names the prompts that build those screens.")
        w("")
        w("| # | Design prompt | Boards | Week | Built by |")
        w("|---|---|---|---|---|")
        for s in members("design"):
            feeds = ", ".join(link(t) for t in DESIGN_FEEDS.get(s["id"], []))
            w(f"| {s['order']} | {link(s['id'])} | {s['brings']} | {s['wave']} | {feeds} |")
        w("")
        continue
    w("```mermaid")
    w("flowchart TB")
    for c in reversed(clubs):
        ms = members(c[0])
        w(f'    subgraph {c[0]}_g["{c[2]} {c[3]}"]')
        w("        direction TB")
        chain = []
        for s in ms:
            w(f'        {nid(s["id"])}["<b>{s["order"]} · {s["id"]}</b><br/>{SHORT[s["id"]]}"]:::{kindclass(s)}')
            chain.append(nid(s["id"]))
            if s["cuts"]:
                tn = nid(s["id"]) + "_tag"
                w(f'        {tn}(["{s["cuts"]}"]):::tag')
                chain.append(tn)
        for x, y in zip(chain, chain[1:]):
            w(f"        {x} --> {y}")
        for x in c[5]:
            sid = nid(x) + "_sh_" + c[0]
            w(f'        {sid}["{S[x]["order"]} · {x}<br/>{SHORT[x]}"]:::shared')
            w(f"        {chain[-1]} ~~~ {sid}")
        w("    end")
    w(CLASSDEFS)
    w("```")
    w("")
    for c in clubs:
        w(f"**{c[2]} {c[3]}** — {c[4]}")
        w("")
        w("| # | Prompt | Brings | Ships |")
        w("|---|---|---|---|")
        for s in members(c[0]):
            if s["cuts"]: ships = f"**tag {s['cuts']}**"
            elif s["lane"] == "outline": ships = "outline only"
            elif s["lane"] == "spike": ships = "results docs only"
            elif s["lane"] == "infra": ships = "infra PRs"
            else: ships = "merge (dark until next tag)"
            w(f"| {s['order']} | {link(s['id'])} | {s['brings']} | {ships} |")
        if c[5]:
            w(f"| — | also {', '.join(link(x) for x in c[5])} | {'these prompts carry' if len(c[5]) > 1 else 'this prompt carries'} part of this workstream | — |")
        w("")

# ------------------------------------------------------------------ 5. run list by week
w("## 5. The run list, week by week")
w("")
w("The numbered order to launch prompts. Lanes in the same week can run side by side; the 🚩 column is the tag a")
w("prompt cuts.")
w("")
for wv in WAVES:
    ms = [s for s in order if s["wave"] == wv[0]]
    w(f"### {wv[0]} · {wv[1]} ({wv[2]})")
    w("")
    lanes = {}
    for s in ms: lanes.setdefault(s["lane"], []).append(s["id"])
    lanetxt = " · ".join(f"**{ln}**: {', '.join('`'+i+'`' for i in ids)}" for ln, ids in lanes.items())
    w(f"Lanes — {lanetxt}")
    w("")
    w("| # | Prompt | Workstream | Brings | 🚩 |")
    w("|---|---|---|---|---|")
    for s in ms:
        c = CL[s["club"]]
        w(f"| {s['order']} | {link(s['id'])} | {c['emoji']} {c['name']} | {s['brings']} | {s['cuts']} |")
    w("")

w("---")
w("")
w("_Generated by [`tools/gen_execution_order.py`](tools/gen_execution_order.py) from")
w("[`tools/execution_order_data.py`](tools/execution_order_data.py). If a sprint is added, split or re-ordered,")
w("update [build-plan.md](build-plan.md) first, then the data file, then re-run the generator._")

open(OUT, "w").write("\n".join(W) + "\n")
print("wrote", OUT, len(W), "lines")
