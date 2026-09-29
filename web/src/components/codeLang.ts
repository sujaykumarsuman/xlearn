// Fenced-code language mapping for the Markdown renderer (ADR-0037).
//
// A fence's info string is author data, never markup: only its first word is read,
// case-insensitively, and it maps to one of the six registered highlighter
// languages or to null (plain text — `text`, unknown names and anything carrying
// markup). The result is also the only language name that may reach a class name.

/** The languages the lazy highlighter registers (t1 §7.2 profile). */
export type CodeLang = "bash" | "cpp" | "go" | "json" | "python" | "sql";

// A Map, not an object literal, so names like `__proto__` or `constructor` can't
// resolve to inherited properties.
const ALIASES: ReadonlyMap<string, CodeLang> = new Map<string, CodeLang>([
  ["bash", "bash"],
  ["sh", "bash"],
  ["shell", "bash"],
  ["zsh", "bash"],
  ["cpp", "cpp"],
  ["c++", "cpp"],
  ["cc", "cpp"],
  ["cxx", "cpp"],
  ["go", "go"],
  ["golang", "go"],
  ["json", "json"],
  ["python", "python"],
  ["python3", "python"],
  ["py", "python"],
  ["sql", "sql"],
  ["postgres", "sql"],
  ["postgresql", "sql"],
  ["psql", "sql"],
]);

/** codeLang maps a fence info string (or its first word) to a registered
 *  language, or null for plain text. */
export function codeLang(info: string | undefined | null): CodeLang | null {
  if (typeof info !== "string") return null;
  const first = info.trim().split(/\s+/, 1)[0] ?? "";
  if (first === "" || first.length > 32) return null;
  return ALIASES.get(first.toLowerCase()) ?? null;
}
