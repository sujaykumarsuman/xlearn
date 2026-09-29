# ADR-0037 — Sanitizing Markdown renderer (react-markdown + remark-gfm, lazy lowlight)

- **Status:** Accepted (2026-09-29, sprint [m1-06](../v2/sprints/sprint-m1-06.md) task 3; accepted by the owner's launch of the prompt, D40)
- **Date:** 2026-09-29
- **Deciders:** @sujaykumarsuman
- **Related:** [0033](0033-invite-only-admission-and-owner-admin.md) §11 (why an XSS on this origin matters), [0027](0027-content-evalpack-and-user-data-model.md) (content model and public media)
- **Links:** [t1 §7.2](../v2/research/t1-content-data-model.md#72-ci-validation) (the Markdown profile), [t1 §10](../v2/research/t1-content-data-model.md#10-security-and-threat-notes) ("Public media and Markdown" row), [sprint m1-06](../v2/sprints/sprint-m1-06.md) task 3.

## Context

- v1's `web/src/components/Markdown.tsx` was a dependency-free subset: headings, bullet lists, inline code/bold/italic
  and the `> [!warning]` "Watch out" callout. It built React nodes, so it was safe, but v2 content needs **GFM tables,
  fenced code with highlighting, images and links** (t1 §7.2).
- The profile is: CommonMark + GFM tables and fenced code; images only as `asset:` refs; https links; no raw HTML; SVG
  only through `<img>`. m1-09's content CI enforces it on the author side. The reader has to enforce it too, because
  the renderer is the last line of defence.
- **This origin is high-value.** An XSS on `projects.sujaykumar.dev` can drive kubescope to `exec` into identity and run
  the owner admin CLI ([ADR-0033 §11](0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b)).
  MI-5b's signed-in probe is still pending, so the renderer can't lean on console isolation.
- The gateway's CSP (m1-04, `internal/gateway/security.go`) is `script-src 'self'`, `style-src 'self'` plus Google Fonts,
  and `img-src 'self' data:`. There are no inline `<style>` elements or scripts.

## Decision

Render curriculum Markdown with **`react-markdown` 10.1.0 + `remark-gfm` 4.0.1**, locked down. Highlight fenced code
with **`lowlight` 3.3.0** (`highlight.js` 11.11.2) in a **lazily loaded chunk**. All four are pinned to exact versions.

1. **No raw HTML.** Use `skipHtml` (HTML blocks, inline tags and comments are dropped, never passed through) and no
   `rehype-raw`. Never use `dangerouslySetInnerHTML` or build HTML strings.
2. **Element allowlist.** `allowedElements` covers p, h1–h6, ul, ol, li, blockquote, strong, em, del, code, pre, a, img,
   table, thead, tbody, tr, th, td, hr and br, with `unwrapDisallowed`. Every allowed element is rendered by our own
   component, which emits only the attributes it names. So no `on*`, `style`, `id` or markdown-derived class reaches
   the DOM. Table cells keep only GFM alignment, as a class.
3. **URLs.** A custom `urlTransform` keeps only `https://` hrefs and only safe `asset:` image refs, and empties every
   other URL attribute.
   - **Links:** `https://` only, rendered with `target="_blank" rel="noopener noreferrer nofollow"`. Any other scheme
     (`javascript:`, `data:`, `vbscript:`, `http:`, `mailto:`, relative, entity-, case- or whitespace-obfuscated)
     renders as its text.
   - **Images:** only `asset:<name>`, where the name is `[A-Za-z0-9][A-Za-z0-9._-]*` segments separated by `/` (no
     scheme, no leading slash, no backslash, no `%`, no `..`). The name goes through a **`resolveAsset(name) → url | null`
     prop**, and the result must be same-origin (`/…`) or `https://`. Images render only as `<img>`, never as inline
     SVG. Anything else renders as its alt text. The curriculum media route and its resolver land with the first
     content that needs them.
4. **Highlighting.** Fenced code renders as plain `<pre class="xl-code"><code>` at once. `import("./highlight")` then
   swaps in token spans. Only six languages are registered: go, cpp, python, sql, json and bash (plus aliases); `text`
   and unknown languages stay plain. The fence info string is data: its first word is only mapped by `codeLang()`. The
   lowlight hast is converted to React `<span className="hljs-…">` nodes by a small walker. Token colours come from
   app.css classes on theme tokens, so no inline styles are needed and the page stays CSP-safe.
5. **v1 parity.** The `> [!warning]` / `> **Watch out**` blockquote is still the violet callout, with the same DOM as
   v1. Every heading level renders as the article `h2`. Bold and italic render as `<b>`/`<i>`, as before. `InlineMD`
   stays hand-written (inline code, bold and italic only).
6. **Author side.** Enforcing the profile when content is written is m1-09's content CI (t1 §7.2). This renderer is the
   reader side and doesn't depend on it.

## Consequences

- **Bundle** (vite build, isolated against `origin/main` `8358aeb`): the main entry chunk grows from 581.76 kB to
  739.64 kB (**+157.9 kB raw, +48.0 kB gzip**: 164.27 → 212.31 kB). The lazy highlighter chunk is **41.96 kB (16.14 kB
  gzip)** and loads only when a page has a registered-language fence. CSS grows by +1.7 kB (+0.4 kB gzip). Route-level
  code splitting (Concept and Problem) would move the parser off the entry chunk. That is a separate, app-wide change.
- **Supply chain.** Four new direct dependencies (the unified/micromark tree underneath) are pinned exactly in
  `web/package.json` and the lockfile. Upgrades are deliberate and re-run the corpus.
- **Tests** (`web/src/components/Markdown.test.tsx`):
  - an **XSS corpus**: `<script>`, `<img onerror>`, `<svg onload>`, iframe, object and embed, comments, obfuscated
    `javascript:`, `data:` and `vbscript:` links, autolinks, reference-style links, `javascript:` and traversal image
    refs, markup in fence info strings, HTML in table cells;
  - **nesting and emphasis bombs**, which must render in under 2 s;
  - the positive profile;
  - **v1 block-structure parity** over every concept body and problem section in `curriculum/`, with an explicit
    allowlist for the files where v2 renders a feature v1 lacked, and a snapshot of the v2 summaries.
- CommonMark semantics differ from v1's line splitter in edge cases, such as lazy continuation lines and ordered
  lists. The parity test's allowlist makes each such difference explicit (today: one `1)`-style ordered list).

## Alternatives considered

- **Keep the hand-written parser and extend it.** No dependencies, but tables, fences, link and image parsing and the
  CommonMark edge cases would all be ours to get right, which is where sanitizer bugs come from. It was acceptable only
  if it passed the same corpus; the audited parser was cheaper.
- **markdown-it or marked plus DOMPurify.** These produce HTML strings that need `dangerouslySetInnerHTML`, so safety
  would rest on a string sanitizer's config staying correct. Rejected: the React-node pipeline never has markup to
  sanitize.
- **Shiki for highlighting.** Better grammars, but much heavier (WASM/oniguruma plus themes), and it emits inline
  `style` colours, which clash with the class-based, CSP-friendly token styling. Rejected in favour of lowlight's
  six-language subset.
