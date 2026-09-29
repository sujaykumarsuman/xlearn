# xLearn — build, test and lint. Mirrors the sibling airlift/landscape targets.
# `make` builds; it never deploys (deployment is GitOps via ../infra + Flux).
BIN     := bin
GATEWAY := gateway
MODULE  := github.com/sujaykumarsuman/xlearn

# The version stamped into the binary (GET <base>/api/healthz, `gateway -version`):
# the nearest git tag, or pass VERSION=v1.2.3. Released images are stamped with the
# release tag itself — .github/workflows/deploy.yml runs only on a `v*` tag push
# (tag-only deploys since v1.0, ADR-0021); pushes to main build nothing. The other
# services stamp `main.version` instead (deploy/<svc>.Dockerfile, guarded by
# deploy/version_test.go).
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(MODULE).Version=$(VERSION)

.PHONY: all web build run test lint go-test go-lint web-test web-lint lint-migrations lint-bodies contentlint packlint install-hooks uninstall-hooks clean nats-acl-render nats-acl-test

all: build

## ---- web (Vite → web/dist) ----
web/node_modules: web/package.json web/package-lock.json
	npm --prefix web ci
	@touch web/node_modules

web: web/node_modules
	npm --prefix web run --silent build
	@touch web/dist/.gitkeep

web-lint: web/node_modules
	npm --prefix web run --silent typecheck
	npm --prefix web run --silent lint

web-test: web/node_modules
	npm --prefix web run --silent test

## ---- gateway (Go, embeds web/dist) ----
build: web
	mkdir -p $(BIN)
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN)/$(GATEWAY) ./cmd/gateway

# Run the built binary locally (serves the SPA at http://localhost:8080/xlearn).
run: build
	$(BIN)/$(GATEWAY)

go-lint:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt:"; echo "$$out"; exit 1; fi
	go vet ./...

go-test:
	go test -race ./...

## ---- migrations (ADR-0034 §3; m1-02) ----
# The contract-header lint over every post-v1.5.2 goose migration (hack/lint-migrations.sh:
# a DROP / SET NOT NULL / type change / RENAME needs `-- xlearn:contract floor=vX.Y.Z` or a
# reviewed `-- xlearn:relax <reason>` on its line), after its own self-tests. CI's `go` job
# runs the same two commands.
lint-migrations:
	sh hack/lint-migrations.sh --self-test
	sh hack/lint-migrations.sh

## ---- request bodies (ADR-0035 §4 L6; m1-05) ----
# The body gate (hack/lint-bodies.sh): no LimitReader(r.Body / ReadAll(r.Body under
# internal/gateway and internal/platform, and in the gateway no raw r.Body handed to
# http.NewRequest* and no json.NewDecoder(r.Body) — bodies go through httpx.ReadBody or a
# MaxBytesReader so an oversize one gets the typed 413. CI's `go` job runs the same two.
lint-bodies:
	sh hack/lint-bodies.sh --self-test
	sh hack/lint-bodies.sh

## ---- public content (curriculum/) ----
# The public content checks (cmd/contentlint; t1 §7.2): schema + strict decode, id and
# slug guards vs ids.lock.json and the previous release tag, the t4 §5.6 structure lints,
# the stamp gate and label-edit flag vs the PR base (CONTENTLINT_BASE), the Markdown
# profile, filename rules, the repo-wide pack-artefact pass, the embed allowlists of
# every //go:embed package. CI's `content` job also runs the
# seeded-row snapshot test (TestSeedMatchesV1Snapshot) on Postgres 18. After adding or
# removing content: go run ./cmd/contentlint -write-allowlist (and review the diff).
contentlint:
	go run ./cmd/contentlint

## ---- NATS auth (ADR-0035 §2; mi-05 N0) ----
# Print the NATS `authorization` block rendered from internal/platform/events/topology.go
# with real PUBLIC nkeys (NKEYS: a file of <identity>=U… lines, one per service + ops).
# LEGACY=allow|deny|none is the bridge stage (N1/N3/N4); FORMAT=yaml is the nats chart
# values fragment for ../infra, FORMAT=conf the server conf. The golden (placeholder
# keys) is internal/platform/events/testdata/nats-authorization.golden.{conf,yaml}.
NKEYS  ?=
LEGACY ?= allow
FORMAT ?= yaml
nats-acl-render:
	@test -n "$(NKEYS)" || { echo "usage: make nats-acl-render NKEYS=<pubkeys.env> LEGACY=allow|deny|none FORMAT=conf|yaml" >&2; exit 2; }
	@go run ./internal/platform/events/natsacl -nkeys "$(NKEYS)" -legacy "$(LEGACY)" -format "$(FORMAT)"

# The NATS-auth integration test: boots nats:2.14 (deploy/local/nats-acl.compose.yml)
# with the rendered block for throwaway nkeys and drives the real client code through
# every allowed/denied case and the three legacy stages. Needs docker.
nats-acl-test:
	go test -tags natsacl -count=1 -run 'TestNATSACL' -v ./internal/platform/events/

## ---- private eval pack authoring (m3-01; docs/v2/authoring.md) ----
# packlint sees both halves: the public item here and the private pack in the sibling
# xlearn-evalpack checkout (PACK=..., default $XLEARN_EVALPACK_DIR or ../xlearn-evalpack).
# Dev tools only: packlint and the hook never enter an image.
PACK ?= $(or $(XLEARN_EVALPACK_DIR),../xlearn-evalpack)

packlint:
	go run ./cmd/packlint check --public . --pack $(PACK)

# The pre-push fingerprint hook (hack/git-hooks/pre-push), per clone: it blocks a push
# whose commits carry pack data, and is a no-op on a machine without the sibling pack.
# Install it on every machine that has the xlearn-evalpack checkout.
install-hooks:
	git config core.hooksPath hack/git-hooks
	@echo "installed: core.hooksPath = $$(git config core.hooksPath)"

uninstall-hooks:
	git config --unset core.hooksPath || true

## ---- aggregate ----
lint: go-lint lint-migrations lint-bodies web-lint
test: go-test web-test

clean:
	rm -rf $(BIN) web/dist/*
	@touch web/dist/.gitkeep
