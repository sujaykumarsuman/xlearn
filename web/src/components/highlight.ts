// The lazily loaded syntax highlighter for the Markdown renderer's fenced code
// (ADR-0037). Markdown.tsx reaches this module only through a dynamic import(), so
// lowlight and its grammars land in their own chunk, off the main bundle.
//
// Only the six profile languages are registered (t1 §7.2). lowlight returns a hast
// tree; it is converted here to React nodes by a tiny walker that emits only <span>
// elements with `hljs-*` class names and text — never an HTML string, never
// dangerouslySetInnerHTML — so the token colours come from app.css (CSP-safe).
import { createElement, type ReactNode } from "react";
import { createLowlight } from "lowlight";
import bash from "highlight.js/lib/languages/bash";
import cpp from "highlight.js/lib/languages/cpp";
import go from "highlight.js/lib/languages/go";
import json from "highlight.js/lib/languages/json";
import python from "highlight.js/lib/languages/python";
import sql from "highlight.js/lib/languages/sql";
import type { CodeLang } from "./codeLang";

const lowlight = createLowlight({ bash, cpp, go, json, python, sql });

/** The subset of hast that lowlight emits. */
type HastNode = {
  type: string;
  value?: string;
  tagName?: string;
  properties?: { className?: unknown };
  children?: HastNode[];
};

/** Keep only highlight.js token classes (`hljs-keyword`, `hljs-string`, …). */
function tokenClass(className: unknown): string | undefined {
  if (!Array.isArray(className)) return undefined;
  const kept = className.filter((c): c is string => typeof c === "string" && /^hljs-[a-z_-]+$/.test(c));
  return kept.length > 0 ? kept.join(" ") : undefined;
}

function toReact(nodes: HastNode[] | undefined, keyBase: string): ReactNode[] {
  const out: ReactNode[] = [];
  (nodes ?? []).forEach((n, i) => {
    const key = `${keyBase}.${i}`;
    if (n.type === "text") {
      out.push(n.value ?? "");
    } else if (n.type === "element") {
      // lowlight only emits <span>; anything else is flattened to its text.
      const kids = toReact(n.children, key);
      out.push(n.tagName === "span" ? createElement("span", { key, className: tokenClass(n.properties?.className) }, ...kids) : kids);
    }
  });
  return out;
}

/** highlight returns `code` as React nodes with `hljs-*` token spans. */
export function highlight(code: string, lang: CodeLang): ReactNode[] {
  const tree = lowlight.highlight(lang, code) as unknown as HastNode;
  return toReact(tree.children, "h");
}
