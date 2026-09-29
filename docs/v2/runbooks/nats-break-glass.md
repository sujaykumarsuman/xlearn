# Runbook — NATS break-glass (the offline `ops` identity)

**What:** the **only sanctioned manual path into NATS** ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first),
[rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag), "No hand-applied changes"). You open a
`kubectl port-forward` tunnel to the `nats` Service over `ssh sujaykumar-vps`, then run the `nats` CLI on the owner's machine
with the **offline `ops` seed**. Written in [mi-06](../sprints/sprint-mi-06.md) (MI-7), after N3 made every service
authenticate with its own nkey and left anonymous clients (`legacy`) with no permissions.

> **Log every use** in [`status.md` → NATS break-glass log](../status.md#nats-break-glass-log): the date, who, why, the
> commands, and the outcome. Reads count too. Never paste a seed, a message payload or learner data into the log.

## When (never routine)

- **Re-seal** ([ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md)): purge a stream, then republish
  it from the owning service's outbox. The outbox is the source of truth; the stream is a buffer.
- **Stream surgery:** a poison message, a consumer reset, or a stream whose limits need a look.
- **A NATS emergency** that `host-verify --cluster` or a service log points at and that no GitOps change can fix.

Anything that can be a PR is a PR. Stream and consumer **config** belongs to `internal/platform/events/topology.go`
(the owning service applies it on start), and the ACL belongs to `infra/infrastructure/messaging/release.yaml`. Neither is
ever changed by hand.

## Who and what `ops` may do

| Identity | Public key | Publish | Subscribe | Seed |
|---|---|---|---|---|
| `ops` | `UAEO5FIZFCY52EU6SYMNBO2XQNF53JDPQIRPRS33UJTLRRZLQIULABIR` | `$JS.API.>` (purge included) | `_INBOX.>` (the CLI's default inbox) | **offline only**, kept with the two age-key copies (MI-1). Never in git, the cluster, CI or a transcript |

`ops` can read, update, purge or delete any stream or consumer, which no service can. It cannot publish on
`xlearn.*` event subjects: republishing is done by the owning service's relay from its outbox, never by hand.

## How

On the **owner's machine** only (never the VPS, never CI, never a pod):

```sh
# 0. Tools (once): the nats CLI
go install github.com/nats-io/natscli/nats@latest

# 1. The seed: copy it from offline storage to a mode-600 temp file OUTSIDE any git checkout
umask 077; SEED=$(mktemp -t nats-ops); cp /path/to/offline/ops.nk "$SEED"   # or decrypt it into $SEED

# 2. The tunnel (terminal 1): port-forward on the node, bound to its loopback, forwarded to yours
ssh -L 14222:127.0.0.1:14222 sujaykumar-vps 'k3s kubectl -n messaging port-forward svc/nats 14222:4222'

# 3. Read first (terminal 2): no --inbox-prefix, because ops subscribes on the default _INBOX.>
nats --server nats://127.0.0.1:14222 --nkey "$SEED" stream ls
nats --server nats://127.0.0.1:14222 --nkey "$SEED" stream info XLEARN_PRACTICE
nats --server nats://127.0.0.1:14222 --nkey "$SEED" consumer info XLEARN_PRACTICE review

# 4. Mutate only against a written plan (in the log entry first): e.g. the re-seal
#    nats … stream purge XLEARN_<SVC>   → then the owning service republishes from its outbox

# 5. Close: Ctrl-C the tunnel, then remove the seed copy
rm -P "$SEED" 2>/dev/null || rm -f "$SEED"; unset SEED
```

- **Close the tunnel** as soon as you're done. Check that nothing is left:
  `ssh sujaykumar-vps 'ps -eo pid,args | grep "[k]ubectl -n messaging port-forward" || echo none'`.
- **Leave no seed copy** in the working directory, shell history, a scratch file or a clipboard manager.
- **Pick a free local port.** 14222 is only a convention; `-L` must match the port-forward's local port.
- After any mutation, run a plain `ssh sujaykumar-vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` and check
  each outbox's unsent count and each durable's pending count.

### Why the tunnel passes MI-5

`messaging/nats-ingress` (MI-5) admits only the `xlearn-*` service pods on 4222. `kubectl port-forward` goes
**through the kubelet into the pod's network namespace**, and connections arrive on the pod's loopback
(`127.0.0.1`), so the NetworkPolicy doesn't apply. This was proved on 2026-09-29 by mi-06's N3 probe: its connections showed up in
the `nats-0` log as `127.0.0.1:<port>`.

## Rotation and revocation (every step is a reload)

For a **service** key (`practice`, `review`, `assessment`, `identity`; later `judge`, `coach`):

1. **Add the new public key.** Generate the new seed on the owner's machine (`nk -gen user`, never printed),
   derive the public key (`nk -inkey … -pubout`), and add a second user with the **same permissions** to
   `infra/infrastructure/messaging/release.yaml`. Re-render with `make nats-acl-render` and never hand-edit a subject.
   Merge it; the reloader SIGHUPs `nats-server` (a reload, not a restart).
2. **Roll the seed.** Re-encrypt `infra/apps/secrets/xlearn-nats-<svc>.enc.yaml` with the new seed (`sops --encrypt
   --in-place`) and bump `podAnnotations` `xlearn.dev/nats-seed-rev` in `apps/xlearn-<svc>.yaml` in the same PR.
   The Deployment rolls, and `/connz?auth=true` then shows the service on the new key.
3. **Remove the old key** from the ACL. That alone revokes it; no restart is needed.

For **`ops`**, do the same without step 2: add the new key, swap the offline copies, then remove the old key.
**Revoking** a compromised key is step 3 on its own, done first.

## Never

- a Job, pod or CronJob carrying the ops seed, or the ops seed anywhere in the cluster, CI or git;
- `kubectl apply`, `edit`, `patch` or `delete` on anything in `messaging` (GitOps only);
- SOPS decryption in `messaging` (public keys are plaintext; seeds live in `apps/secrets`);
- `nats pub` on an `xlearn.*` subject by hand (republishing goes through the outbox relay);
- a `legacy` or anonymous "just for now" connection (since N3, `legacy` can do nothing; it is removed at N4, [mi-11](../sprints/sprint-mi-11.md)).

## First use (`ev-nats-ops-first-use`, owner)

After mi-06 the **owner** runs one read through the steps above: `nats … --nkey "$SEED" stream ls`. That proves the
offline ops seed and its ACL before anyone needs them in an emergency. mi-06's N3 probe already proved the tunnel with no seed.
Log the run in the break-glass log and tick `ev-nats-ops-first-use` in `status.md`.

Expected output: the streams `XLEARN_PRACTICE`, `XLEARN_REVIEW`, `XLEARN_ASSESSMENT` and `XLEARN_IDENTITY`.
An `Authorization Violation` means the seed doesn't match the `ops` public key above. A `Permissions Violation
for Subscription to "_INBOX…"` means an `--inbox-prefix` or a context overrode the default inbox.
