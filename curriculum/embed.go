// Package curriculum embeds the versioned curriculum seed files (the DSA path and
// coming-soon stubs) so the curriculum service can seed them idempotently on startup
// (ADR-0005, PRD Q2: seed files, not an in-app CMS). It is deliberately data-only —
// no logic, no dependency on the store — so the curriculum binary embeds just the
// seed and never drags in the gateway's web/dist. go:embed cannot reach a parent
// directory, so this file lives beside the seed data at the repo root rather than
// under internal/curriculum (mirrors embed.go embedding web/dist for the gateway).
package curriculum

import "embed"

// FS holds the JSON seed files: paths.json (all paths) + dsa/*.json (the DSA
// content). internal/curriculum.Seed reads and upserts them at startup.
//
// It also carries, additively (v2 M1a, sprint m1-01), the course manifests
// courses/*/course.json and the JSON Schemas _schema/*.json (the manifest schema and
// the frozen item schema). internal/course.Load reads the manifests; nothing reads
// them at runtime yet, and the v1 loader keeps reading dsa/*.json until m1-09 moves
// the content under courses/<slug>/ and replaces this directive with all:courses.
//
//go:embed paths.json dsa/*.json courses/*/course.json _schema/*.json
var FS embed.FS
