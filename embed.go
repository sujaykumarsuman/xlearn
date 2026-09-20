// Package xlearn is the module root. It exists only to embed the built web SPA
// (web/dist) so that cmd/gateway can serve it from a single static binary.
// go:embed cannot reach a parent directory, which is why this file lives at the
// module root rather than under cmd/gateway or internal/gateway (mirrors the
// sibling airlift repo).
package xlearn

import "embed"

// Dist holds the Vite build output. It is empty (only .gitkeep) until
// `make web` (or the Docker web stage) has run; internal/gateway treats a
// missing index.html as "not built".
//
//go:embed all:web/dist
var Dist embed.FS

// Version is the build version reported by GET /xlearn/api/healthz. Override at
// build time with -ldflags "-X github.com/sujaykumarsuman/xlearn.Version=vX.Y.Z".
var Version = "dev"
