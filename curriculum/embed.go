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
//go:embed paths.json dsa/*.json
var FS embed.FS
