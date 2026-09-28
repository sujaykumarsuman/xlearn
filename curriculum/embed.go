// Package curriculum embeds the versioned public course content so the curriculum
// service can seed it on startup (ADR-0005, PRD Q2: seed files, not an in-app CMS;
// ADR-0027 §2 for the layout, described in README.md). It is deliberately data-only —
// no logic, no dependency on the store — so the curriculum binary embeds just the
// content and never drags in the gateway's web/dist. go:embed cannot reach a parent
// directory, so this file lives beside the content at the repo root rather than under
// internal/curriculum (mirrors embed.go embedding web/dist for the gateway).
package curriculum

import "embed"

// FS holds the public content: paths.json (the catalog), ids.lock.json (every item id
// ever published), the JSON Schemas under _schema/, and every course under courses/
// (manifest, phases/weeks/concepts, concept sidecars, items with their section
// sidecars). internal/curriculum.Seed loads it with the glob loader at startup.
//
// The directive must be `all:courses`: a plain `courses` silently drops the `_code/`
// directories. `all:` also embeds dotfiles, which the loader ignores and CI rejects:
// cmd/contentlint diffs the embedded file list against embed.allowlist.
//
//go:embed paths.json ids.lock.json all:courses _schema/*.json
var FS embed.FS
