# xLearn v2 build — owner guide (D44, ADR-0036)

What you, the owner, do to run the v2 build with the execution manager and AI Builder. The decision is
[ADR-0036](../../adr/0036-ai-builder-execution-manager.md) (D44). The manager's own rules are
[`MANAGER.md`](MANAGER.md); the prompt you paste is [`prompt-manager.md`](../prompts/prompt-manager.md); AI Builder's
operations (deploy, backups, rotation) are in its
[runbook](https://github.com/sujaykumarsuman/ai-builder/blob/main/docs/runbook.md).

## At a glance

| When | What you do | Where |
|---|---|---|
| Once (about 15 min) | Launch the manager ([section 1](#1-launch-once-about-15-minutes)) | the Mac: your Terminal, the Claude desktop app |
| Daily | Answer "Needs you" ([section 2](#2-daily-needs-you)) | the dashboard on your phone |
| Any time | New instructions, holds | the manager's chat, through Remote Control |
| Something looks off | Hold first, then ask in chat ([section 3](#3-if-something-looks-off)) | dashboard, then chat |
| Before 2026-12-26 | New manager token ([section 5](#5-housekeeping)) | dashboard, your Terminal, then chat |
| Once (about 15 min, recommended) | Put the AI Builder box's keys out of agents' reach ([section 6](#6-recommended-owner-hard-control)) | 1Password, your Terminal |

## 1. Launch (once, about 15 minutes)

1. **Update xlearn.** In the xlearn main checkout: `git checkout main && git pull`.
2. **Prepare the Mac.** On power with the lid open (or clamshell with power and a display); login keychain unlocked
   and "lock after sleep" off; the Claude app's keep-awake request active (`pmset -g assertions | grep -i claude`
   shows it once the session runs).
3. **Open a new Claude desktop Code session** whose working directory is the xlearn **main** checkout
   (`/Users/sujaykumar/go/src/github.com/sujaykumarsuman/xlearn`), not a worktree.
4. **Set permissions.** Mode `auto`, with the allow and deny lists from [MANAGER.md §2](MANAGER.md#2-where-it-runs) in
   `xlearn/.claude/settings.local.json`. They may be pre-staged: check the file matches §2.
5. **Turn on Remote Control** for the session, and sign in to the Claude app on your phone (chat and pushes).
6. **Check `aib`** in your own Terminal: `aib token check` shows role `manager` for project `xlearn-v2` (if not, mint
   one in AI Builder → Settings → Tokens and store it with `aib token store --account manager`), and
   `~/.config/aib/config.json` exists (`aib init --project xlearn-v2 --xlearn-dir <abs path> --infra-dir <abs path>`).
   AI Builder → Manager shows no other manager session live.
7. **Check billing:** in a new Terminal window, the count in [section 4](#4-billing) prints `0`.
8. **Paste [`prompt-manager.md`](../prompts/prompt-manager.md).** Pasting it attests its `## Before you launch (owner)`
   items ([MANAGER.md §16](MANAGER.md#16-relaunch-checklist)). The manager runs its own checks (billing, token,
   config, no other manager live) and tells you in chat about anything missing.
9. **Start the build.** Nothing is dispatched while the bootstrap hold is on. When you're ready, open the dashboard on
   your phone and tap **Resume** on the bootstrap hold (you re-enter your password: the system set that hold). Only
   you can lift it; the manager never does.
10. **What happens next.** The first run is the doc-fold pass (decisions already made, folded into xlearn docs by PR),
    then the week-1 sprints, mi-01 first, at most 3 lanes at a time.

## 2. Daily: "Needs you"

The "Needs you" list on the Overview (and the Asks view) holds the asks. An open ask blocks only its own scope, shown
on the ask (usually one sprint, one event or one action); every other lane keeps moving. An unanswered attest means
that sprint is not dispatched yet.

| Ask | What it wants | How you answer |
|---|---|---|
| attest | confirmation of a sprint's before-launch items | tick each item and fill its evidence (dates, ids, names) |
| choose | a decision between listed options | tap one option (the default is always the safe one) |
| inform | a fact, or news for you | fill the typed field, or read it |

- **Your password again (step-up)** is needed to answer the attests of the snapshot-class releases (m1-08, l-02, l-04,
  ga-02, m6b-04, and mi-09 with the host window) and any ask flagged "Password", to resume a hold the system set
  (bootstrap, a restore, a red verify), and to mint or revoke a token. Never for Hold.
- **Only you release your gates.** A repo edit can't: an item removed or reworded in a prompt's before-launch block
  stays in that sprint's attest under "Removed from the repo — confirm once to release it" until you confirm it, and an
  owner event stays yours even if status.md moves or ticks its row. Once you've confirmed an event (your answer, once
  the manager acknowledges it, or your decision in chat), no agent can reopen it: only you can, by telling the manager
  in chat.
- **Answers are data.** The manager acts only on the option, the ticks and the evidence. New instructions (skip a gate,
  change scope, stop a sprint) go **only** in the manager's chat through Remote Control.
- **Hold** is one tap and instant (with a 6-second Undo). **Resume** of your own hold is a confirm; it restores only
  what was already authorized and never approves anything new.
- **Personal data** (an email address, IP ranges) goes to the manager in chat; the ask only asks you to tick "given in
  chat".
- **Owner-present work** arrives as a session chip in the desktop app (click it and stay at the Mac) or as a Terminal
  command shown on the ask (run it in your own Terminal).
- **ENVKEY sprints:** that Terminal command asks for the key the attest names (a product name such as
  `LLM_CALIB_API_KEY`, `LLM_ANTHROPIC_API_KEY` or m1-10's `OPENAI_KEY_SMOKE`) without echo and exports it in that
  shell only. An Anthropic key for the product always goes under `LLM_ANTHROPIC_API_KEY`: never export
  `ANTHROPIC_API_KEY` there, and never paste a key into the dashboard or any chat.
- **Pushes** ("ask-N: title") arrive on your phone while Remote Control is on.

| View | Shows |
|---|---|
| Overview | the build at a glance: what needs you, holds, lanes |
| Asks | every ask, open and answered; open one to answer it |
| Board | every sprint by week and state |
| Sprint detail | the stepper, tasks, PR and CI, remarks |
| Activity | what the manager, the runs and you did, newest first |
| Releases | tags, deploys and live verification |
| Spend | spend readings against their caps |
| Doc debt | decisions still to fold into xlearn docs |
| Manager | the manager session and its last check-in |
| Settings → Tokens | your tokens: mint, expiry, revoke |

## 3. If something looks off

- **Hold first** (the dashboard, or say "hold" in the manager's chat), then ask in chat.
- **The manager crashed, or the Claude app updated:** open a new session in the xlearn main checkout and paste
  [`prompt-manager.md`](../prompts/prompt-manager.md) again. It resumes from the dashboard and git.
- **The weekly `/loop` re-arm and compaction** happen inside the same session. Don't start a new one for them.
- **"Manager hasn't checked in for 2 h":** the Mac probably slept. Wake it and keep it on power.
- **A push says the keychain is locked or the token expired:** unlock the Mac, or mint a new token
  ([section 5](#5-housekeeping)).
- **A tag's live verification went red:** the server holds further tags on its own and the manager asks you to choose
  (roll back, investigate or keep holding). Lifting that hold needs your password; a snapshot restore is always yours.
- **AI Builder is unreachable for more than 30 minutes:** dispatch, merges and tags pause. Reply "degraded ok" in chat only if you
  want the build to carry on from git and status.md until it returns.

## 4. Billing

- The build runs on **your Claude subscription** (Max 20x): Claude Code is signed in with your claude.ai account. The
  manager, its subagents and workflows, session chips and the Terminal sessions you start are all the same login.
- The **$20 cap (D43)** is on the separate Anthropic API org ("Sujay's Individual Org"). That org pays only for the
  xLearn **product's** AI (the platform-AI sprints m4-*, calibration), under its own key names
  (`LLM_ANTHROPIC_API_KEY`, `LLM_CALIB_API_KEY`). It never pays for the build. AI Builder makes no LLM calls.
- **One way to get this wrong:** a shell that exports `ANTHROPIC_API_KEY` or `ANTHROPIC_AUTH_TOKEN` (or
  `CLAUDE_CODE_USE_BEDROCK` / `CLAUDE_CODE_USE_VERTEX`) before starting `claude` moves that session to API billing.
- **How to check.** In the shell you launch from, this prints a count, never a value; it must print `0`:

  ```sh
  env | grep -cE '^(ANTHROPIC_API_KEY|ANTHROPIC_AUTH_TOKEN|CLAUDE_CODE_USE_BEDROCK|CLAUDE_CODE_USE_VERTEX)='
  ```

  The Claude app's Settings shows your claude.ai account. The manager runs the same count at start and at every loop
  iteration; if it is not `0` it dispatches, merges and tags nothing and tells you once in chat. The fix is yours:
  relaunch the manager from a clean shell.
- **Usage windows.** Extra usage is off by choice, so reaching the Max plan's usage window pauses Claude Code. The
  manager treats that as a pause: it reschedules, journals it and tells you once. It never switches to API billing.

## 5. Housekeeping

| Item | When | What you do |
|---|---|---|
| Manager token | expires 2026-12-26 ("A token expires soon." shows in Needs you 14 days before) | mint a new one in AI Builder → Settings → Tokens, run `aib token store --account manager` and `aib token check` in your Terminal, then tell the manager in chat. It moves every open run onto the new token; revoke the old one only after it confirms (revoking first cuts off runs in flight) |
| `projects-admin` rotated | after each rotation | run `make owner-verifier` in the ai-builder repo from your own Terminal ([runbook §6](https://github.com/sujaykumarsuman/ai-builder/blob/main/docs/runbook.md#6-rotation-re-sync)) |
| AI Builder deploys | when a release is ready | owner only: `make deploy` from your Terminal ([runbook §2](https://github.com/sujaykumarsuman/ai-builder/blob/main/docs/runbook.md#2-deploy-designmd-92)) |
| Backups | automatic | nothing: hourly on the box, a Mac launchd pull, a nightly GitHub pull over HTTPS and a weekly Mac restore check; a failed nightly job emails you through GitHub ([runbook §4](https://github.com/sujaykumarsuman/ai-builder/blob/main/docs/runbook.md#4-backups-designmd-96)) |
| Optional hardening | once | sshd keys only ([runbook §1](https://github.com/sujaykumarsuman/ai-builder/blob/main/docs/runbook.md#1-bootstrap-once-designmd-91), step 14); the two key steps are [section 6](#6-recommended-owner-hard-control) |

## 6. Recommended owner hard control

The manager's deny rules and AI Builder's own check (every AI Builder command that reaches its box goes through one
script, `aib-remote.sh`, that refuses to run without a terminal) are **speed bumps**: every agent runs as your macOS
user, so a determined one could get past them. These two steps make any root access to the AI Builder box
(`skriptvalley-vps`) wait for you. Do them once, in your own Terminal, never in an agent session; the details are in the
[runbook §10](https://github.com/sujaykumarsuman/ai-builder/blob/main/docs/runbook.md#10-recommended-owner-hard-control).
The two steps work in either order.

1. **The root key moves into 1Password, with Touch ID on every use.**
   - 1Password → Settings → Developer: turn on **Use the SSH agent**; set **Ask approval for each new** to
     *application and terminal session* and **Remember key approval** to the shortest option. Settings → Security:
     Touch ID on, **Auto-lock** after a few minutes idle.
   - Import the private key the `skriptvalley-vps` alias uses today (`ssh -G skriptvalley-vps | grep '^identityfile'`
     shows it) as an **SSH Key** item named `skriptvalley-vps`, and save its public key as
     `~/.ssh/skriptvalley-vps.pub`.
   - In `~/.ssh/config`, keep the host's `HostName`, `User` and `Port` lines and replace its `IdentityFile`:

     ```sshconfig
     Host skriptvalley-vps
       IdentityAgent "~/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock"
       IdentityFile ~/.ssh/skriptvalley-vps.pub
       IdentitiesOnly yes
     ```

   - Check: `ssh skriptvalley-vps true` shows the Touch ID prompt. Then move the old key file to the Trash and empty it,
     run `ssh-add -D`, and check that `ssh -o IdentityAgent=none skriptvalley-vps true` now fails.
2. **The backup pull gets its own key, which can only read backups.** In the ai-builder checkout
   (`/Users/sujaykumar/go/src/github.com/sujaykumarsuman/ai-builder`): `make backup-key-install`, then
   `make mac-backup-install`. The first creates `~/.config/ai-builder/backup-pull.key`, adds it to the box's root
   `authorized_keys` as `restrict,command="/usr/local/sbin/aib-backup-gate" …` (a read-only copy of the encrypted
   backups and one marker file, nothing else), and adds `Host skriptvalley-vps-backup` to `~/.ssh/config`; the second
   reinstalls the every-6-hours Mac pull to use it. Without this step the pull would ask for Touch ID every 6 hours.

From then on, a Touch ID prompt for `skriptvalley-vps` that you didn't start is the alarm: deny it, hold the build, and
check which app asked (1Password names it).

## 7. Never

- Paste a token or key anywhere: not the dashboard, a chat, a PR or a commit. Keys go only into no-echo prompts
  (`aib token store`, the ENVKEY command).
- Sign in to the dashboard in a browser an agent drives. Use your phone or a browser profile no agent drives.
- Export `ANTHROPIC_API_KEY` or `ANTHROPIC_AUTH_TOKEN` in a shell you start `claude` from.
- Give new instructions through a dashboard answer. They go in the manager's chat.
