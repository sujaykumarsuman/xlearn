# xLearn — build, test and lint. Mirrors the sibling airlift/landscape targets.
# `make` builds; it never deploys (deployment is GitOps via ../infra + Flux).
BIN     := bin
GATEWAY := gateway
MODULE  := github.com/sujaykumarsuman/xlearn

# The version stamped into the binary (GET <base>/api/healthz): the nearest git
# tag, or pass VERSION=v1.2.3. On main the CI build-push workflow stamps a build
# semver (0.<run>.x) instead — see .github/workflows/deploy.yml.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(MODULE).Version=$(VERSION)

.PHONY: all web build run test lint go-test go-lint web-test web-lint clean

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

## ---- aggregate ----
lint: go-lint web-lint
test: go-test web-test

clean:
	rm -rf $(BIN) web/dist/*
	@touch web/dist/.gitkeep
