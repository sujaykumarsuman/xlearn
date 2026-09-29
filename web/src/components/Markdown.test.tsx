import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { render, screen, waitFor } from "@testing-library/react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { InlineMD, Markdown } from "./Markdown";

// The sanitizing renderer's contract (ADR-0037): an XSS corpus, the positive profile
// (tables, highlighted fenced code, asset: images, https links, the callout) and block
// parity with the v1 renderer over every concept body and problem section in curriculum/.

const FORBIDDEN = "script,iframe,object,embed,svg,style,form,input,link,meta,base,frame,frameset,math";
const RESOLVED = "/xlearn/api/curriculum/media/test/";

/** renderSafe renders `source` with a spy resolver and asserts the DOM invariants. */
function renderSafe(source: string) {
  const returned: string[] = [];
  const resolveAsset = vi.fn((ref: string) => {
    const url = RESOLVED + ref;
    returned.push(url);
    return url;
  });
  const view = render(
    <div className="cn">
      <Markdown source={source} resolveAsset={resolveAsset} />
    </div>,
  );
  assertSafe(view.container, returned);
  return { ...view, resolveAsset };
}

/** The renderer's own callout icon: the design-system sprite reference, not content. */
function isCalloutIcon(el: Element): boolean {
  if (el.tagName.toLowerCase() === "use") return el.getAttribute("href") === "#i-alert" && isCalloutIcon(el.parentElement!);
  return (
    el.tagName.toLowerCase() === "svg" &&
    el.parentElement?.classList.contains("cn-callout__h") === true &&
    el.innerHTML === '<use href="#i-alert"></use>'
  );
}

function assertSafe(root: HTMLElement, resolved: string[]) {
  const forbidden = Array.from(root.querySelectorAll(FORBIDDEN)).filter((el) => !isCalloutIcon(el));
  expect(forbidden.map((el) => el.outerHTML)).toEqual([]);
  for (const el of root.querySelectorAll("*")) {
    for (const attr of Array.from(el.attributes)) {
      expect(attr.name.toLowerCase().startsWith("on"), `${el.tagName} @${attr.name}`).toBe(false);
      expect(attr.name.toLowerCase(), `${el.tagName} @${attr.name}`).not.toMatch(/^(style|srcdoc|formaction|xlink:href)$/);
    }
  }
  for (const a of root.querySelectorAll("a")) {
    expect(a.getAttribute("href") ?? "").toMatch(/^https:\/\//);
    expect(a.getAttribute("target")).toBe("_blank");
    expect(a.getAttribute("rel")).toBe("noopener noreferrer nofollow");
  }
  for (const img of root.querySelectorAll("img")) {
    expect(resolved).toContain(img.getAttribute("src"));
  }
}

/** Each case: [name, markdown]. The DOM invariants are asserted by renderSafe. */
const XSS: Array<[string, string]> = [
  ["script element", "<script>alert(1)</script>"],
  ["inline script", "hello <script>alert(1)</script> world"],
  ["img onerror", "<img src=x onerror=alert(1)>"],
  ["inline img onerror", "text <img src=x onerror=alert(1)> more"],
  ["svg onload", "<svg onload=alert(1)>"],
  ["iframe", "<iframe src=https://evil.example></iframe>"],
  ["object", "<object data=x></object>"],
  ["embed", "<embed src=x>"],
  ["style element", "<style>body{display:none}</style>"],
  ["form + input", "<form action=https://evil.example><input name=p></form>"],
  ["html comment", "<!-- c -->"],
  ["inline comment", "a <!-- c --> b"],
  ["javascript link", "[x](javascript:alert(1))"],
  ["mixed-case javascript link", "[x](JaVaScRiPt:alert(1))"],
  ["whitespace javascript link", "[x](  javascript:alert(1))"],
  ["hex-entity javascript link", "[x](java&#x73;cript:alert(1))"],
  ["dec-entity javascript link", "[x](&#106;avascript:alert(1))"],
  ["tab-split javascript link", "[x](<java\tscript:alert(1)>)"],
  ["data link", "[x](data:text/html;base64,PHNjcmlwdD4=)"],
  ["vbscript link", "[x](vbscript:msgbox)"],
  ["http link", "[x](http://insecure.example)"],
  ["relative link", "[x](/xlearn/admin)"],
  ["protocol-relative link", "[x](//evil.example)"],
  ["mailto link", "[x](mailto:a@b.example)"],
  ["javascript autolink", "<javascript:alert(1)>"],
  ["https autolink", "<https://ok.example>"],
  ["reference-style javascript link", "[x][r]\n\n[r]: javascript:alert(1)"],
  ["reference-style https link", "[x][r]\n\n[r]: https://ok.example \"t\""],
  ["javascript image", "![a](javascript:alert(1))"],
  ["asset traversal image", "![a](asset:../../etc/passwd)"],
  ["percent-encoded traversal image", "![a](asset:%2e%2e/x)"],
  ["absolute asset image", "![a](asset:/etc/passwd)"],
  ["backslash asset image", "![a](asset:..\\x)"],
  ["data svg image", "![a](data:image/svg+xml;base64,PHN2Zz4=)"],
  ["https image", "![a](https://evil.example/x.png)"],
  ["relative image", "![a](x.png)"],
  ["reference-style javascript image", "![a][i]\n\n[i]: javascript:alert(1)"],
  ["link wrapping a bad image", "[![a](javascript:x)](javascript:y)"],
  ["fence info with markup", "```\"><script>alert(1)</script>\ncode\n```"],
  ["fence info with attribute", "```go onload=x\ncode\n```"],
  ["tilde fence info with markup", "~~~<img src=x onerror=alert(1)>\ncode\n~~~"],
  ["table cell html", "| a | b |\n|---|---|\n| <b onclick=x>y</b> | <script>alert(1)</script> |"],
  ["table cell javascript link", "| a |\n|---|\n| [x](javascript:alert(1)) |"],
  ["gfm task list input", "- [ ] todo\n- [x] done"],
  ["footnote", "a[^1]\n\n[^1]: note"],
  ["html block after text", "para\n\n<div onclick=alert(1)>x</div>"],
];

describe("Markdown XSS corpus", () => {
  it.each(XSS)("%s renders without executable markup", (_name, source) => {
    const { container } = renderSafe(source);
    // Dangerous strings may survive only as inert text, never inside an attribute.
    for (const el of container.querySelectorAll("*")) {
      for (const attr of Array.from(el.attributes)) {
        expect(attr.value, `${el.tagName} @${attr.name}`).not.toMatch(/javascript:|vbscript:|data:|<|>|"/i);
      }
    }
  });

  it("renders unsafe links as their text", () => {
    for (const src of ["[x](javascript:alert(1))", "[x](JaVaScRiPt:alert(1))", "[x](java&#x73;cript:alert(1))", "[x](http://insecure.example)"]) {
      const { container, unmount } = renderSafe(src);
      expect(container.querySelector("a")).toBeNull();
      expect(container.textContent).toBe("x");
      unmount();
    }
  });

  it("renders unsafe or unresolvable images as their alt text", () => {
    for (const src of ["![a](javascript:alert(1))", "![a](asset:../../etc/passwd)", "![a](asset:%2e%2e/x)", "![a](https://evil.example/x.png)"]) {
      const { container, resolveAsset, unmount } = renderSafe(src);
      expect(container.querySelector("img")).toBeNull();
      expect(container.textContent).toBe("a");
      expect(resolveAsset).not.toHaveBeenCalled();
      unmount();
    }
  });

  it("keeps a fence info string out of the DOM", () => {
    const { container } = renderSafe("```\"><script>alert(1)</script>\ncode\n```");
    const code = container.querySelector("pre code");
    expect(code?.textContent).toBe("code");
    expect(code?.getAttribute("class")).toBeNull();
    expect(container.textContent).not.toContain("script");
  });

  it("drops raw HTML inside table cells but keeps the text", () => {
    const { container } = renderSafe("| a | b |\n|---|---|\n| <b onclick=x>y</b> | <script>alert(1)</script> |");
    const cells = Array.from(container.querySelectorAll("td")).map((td) => td.textContent);
    // The tags are skipped; the text between them stays as inert text.
    expect(cells).toEqual(["y", "alert(1)"]);
    expect(container.querySelector("b")).toBeNull();
  });

  const BOMBS: Array<[string, string]> = [
    ["emphasis bomb", "*a **".repeat(3000)],
    ["underscore bomb", "_a __".repeat(3000)],
    ["bracket bomb", "[".repeat(10000)],
    ["link-open bomb", "[a](".repeat(3000)],
    ["blockquote nesting bomb", "> ".repeat(500) + "x"],
    ["list marker nesting bomb", "- ".repeat(300) + "x"],
    ["indented list nesting bomb", Array.from({ length: 300 }, (_, i) => "  ".repeat(i) + "- x").join("\n")],
    ["backtick bomb", "`a``".repeat(3000)],
  ];

  it.each(BOMBS)("%s renders in bounded time", (_name, source) => {
    // The bound is on the renderer (parse, sanitize, React elements → markup). jsdom's
    // own node insertion is quadratic on a 900-deep tree (~1.5 s here), which a browser's
    // isn't, so the DOM render below is checked for safety but not timed.
    const t0 = performance.now();
    renderToStaticMarkup(<Markdown source={source} resolveAsset={() => null} />);
    expect(performance.now() - t0).toBeLessThan(2000);
    const { container } = render(<Markdown source={source} resolveAsset={() => null} />);
    assertSafe(container, []);
  }, 15_000);
});

describe("Markdown profile", () => {
  it("renders a GFM table with column alignment", () => {
    const { container } = renderSafe("| n | cost |\n|:--|--:|\n| 1 | O(n) |\n| 2 | O(1) |");
    expect(container.querySelectorAll("table th")).toHaveLength(2);
    expect(Array.from(container.querySelectorAll("td")).map((td) => td.textContent)).toEqual(["1", "O(n)", "2", "O(1)"]);
    expect(container.querySelector("td:nth-child(2)")).toHaveClass("cn-t--right");
  });

  it("renders fenced Go as plain code, then highlights it once the lazy chunk loads", async () => {
    const { container } = renderSafe("```go\nfunc main() {\n\treturn \"hi\"\n}\n```");
    const code = container.querySelector("pre.xl-code > code");
    expect(code).toHaveClass("language-go");
    expect(code?.textContent).toBe('func main() {\n\treturn "hi"\n}');
    await waitFor(() => expect(code?.querySelector('span[class^="hljs-"]')).not.toBeNull());
    expect(code?.textContent).toBe('func main() {\n\treturn "hi"\n}');
    assertSafe(container, []);
  });

  it("leaves text and unknown fences unhighlighted", async () => {
    const { container } = renderSafe("```text\nplain\n```\n\n```brainfuck\n+++\n```");
    const codes = container.querySelectorAll("pre code");
    expect(codes).toHaveLength(2);
    await new Promise((r) => setTimeout(r, 50));
    for (const c of codes) {
      expect(c.getAttribute("class")).toBeNull();
      expect(c.querySelector("span")).toBeNull();
    }
  });

  it("renders an asset: image through resolveAsset", () => {
    const { container, resolveAsset } = renderSafe("![diagram](asset:diagram.svg)");
    expect(resolveAsset).toHaveBeenCalledWith("diagram.svg");
    const img = container.querySelector("img");
    expect(img).toHaveAttribute("src", RESOLVED + "diagram.svg");
    expect(img).toHaveAttribute("alt", "diagram");
  });

  it("falls back to alt text without a resolver or with an unusable URL", () => {
    const bare = render(<Markdown source="![diagram](asset:diagram.svg)" />);
    expect(bare.container.querySelector("img")).toBeNull();
    expect(bare.container.textContent).toBe("diagram");
    bare.unmount();
    for (const url of [null, "javascript:alert(1)", "//evil.example/x.png", "data:image/png;base64,AA=="]) {
      const view = render(<Markdown source="![diagram](asset:diagram.svg)" resolveAsset={() => url} />);
      expect(view.container.querySelector("img")).toBeNull();
      expect(view.container.textContent).toBe("diagram");
      view.unmount();
    }
  });

  it("renders an https link as a safe external link", () => {
    renderSafe("See [the Go spec](https://go.dev/ref/spec).");
    const a = screen.getByRole("link", { name: "the Go spec" });
    expect(a).toHaveAttribute("href", "https://go.dev/ref/spec");
  });

  it("renders the [!warning] blockquote as the violet Watch out callout", () => {
    const { container } = renderSafe("> [!warning] The width is `right - left + 1`, not `right - left`.");
    const callout = container.querySelector(".cn-callout");
    expect(callout).toHaveClass("ds-card", "ds-card--violet");
    expect(callout?.querySelector(".cn-callout__h")?.textContent).toBe("Watch out");
    expect(callout?.querySelector("p")?.textContent).toBe("The width is right - left + 1, not right - left.");
    expect(container.textContent).not.toContain("[!warning]");
  });

  it("renders a bold Watch out lead and a plain blockquote like v1", () => {
    const { container } = renderSafe("> **Watch out**: off by one.\n\n> Just a quote.");
    const [warn, plain] = Array.from(container.querySelectorAll(".cn-callout"));
    expect(warn).toHaveClass("ds-card--violet");
    expect(warn?.querySelector("p")?.textContent).toBe("off by one.");
    expect(plain).not.toHaveClass("ds-card--violet");
    expect(plain?.querySelector(".cn-callout__h")).toBeNull();
    expect(plain?.textContent).toBe("Just a quote.");
  });

  it("renders every heading level as the article h2", () => {
    const { container } = renderSafe("# One\n\n### Three\n\nSetext\n---");
    expect(Array.from(container.querySelectorAll("h2")).map((h) => h.textContent)).toEqual(["One", "Three", "Setext"]);
  });

  it("InlineMD renders inline code, bold and italic without block wrappers", () => {
    const { container } = render(<InlineMD text="Use `map[T]int`, **bold** and *italic*." />);
    expect(container.querySelector("code")?.textContent).toBe("map[T]int");
    expect(container.querySelector("b")?.textContent).toBe("bold");
    expect(container.querySelector("i")?.textContent).toBe("italic");
    expect(container.querySelector("p")).toBeNull();
  });
});

// --- v1 parity ---
// v1's block splitter, frozen verbatim from the pre-ADR-0037 Markdown.tsx (m1-06) as
// the reference: parity means the same top-level block structure.

type Block =
  | { kind: "heading"; text: string }
  | { kind: "list"; items: string[] }
  | { kind: "callout"; variant: "warn" | "plain"; title: string; lines: string[] }
  | { kind: "para"; text: string };

const BULLET = /^\s*[-*]\s+/;
const HEADING = /^\s*#{1,6}\s+/;
const QUOTE = /^\s*>\s?/;
const ADMONITION = /^\s*(?:\[!(\w+)\]\s*|\*\*([^*]+)\*\*[:.]?\s*)/;

function splitBlocks(src: string): Block[] {
  const lines = src.replace(/\r\n/g, "\n").split("\n");
  const blocks: Block[] = [];
  let i = 0;
  const at = (idx: number): string => lines[idx] ?? "";

  while (i < lines.length) {
    const line = at(i);
    if (line.trim() === "") {
      i++;
      continue;
    }
    if (HEADING.test(line)) {
      blocks.push({ kind: "heading", text: line.replace(HEADING, "").trim() });
      i++;
      continue;
    }
    if (QUOTE.test(line)) {
      const quoted: string[] = [];
      while (i < lines.length && QUOTE.test(at(i))) {
        quoted.push(at(i).replace(QUOTE, ""));
        i++;
      }
      const first = quoted[0] ?? "";
      const m = first.match(ADMONITION);
      const kind = (m?.[1] ?? "").toLowerCase();
      const boldTitle = m?.[2] ?? "";
      const isWarn = kind === "warning" || kind === "caution" || kind === "warn" || /watch out/i.test(boldTitle);
      const title = boldTitle.trim() || (isWarn ? "Watch out" : "");
      const rest = [first.replace(ADMONITION, ""), ...quoted.slice(1)].filter((l, idx) => idx === 0 || l.trim() !== "");
      blocks.push({ kind: "callout", variant: isWarn ? "warn" : "plain", title, lines: rest });
      continue;
    }
    if (BULLET.test(line)) {
      const items: string[] = [];
      while (i < lines.length && BULLET.test(at(i))) {
        items.push(at(i).replace(BULLET, "").trim());
        i++;
      }
      blocks.push({ kind: "list", items });
      continue;
    }
    const para: string[] = [];
    for (let l = at(i); i < lines.length && l.trim() !== "" && !BULLET.test(l) && !HEADING.test(l) && !QUOTE.test(l); l = at(i)) {
      para.push(l.trim());
      i++;
    }
    blocks.push({ kind: "para", text: para.join(" ") });
  }
  return blocks;
}

function v1Summary(src: string): string[] {
  return splitBlocks(src).map((b) =>
    b.kind === "list" ? `list:${b.items.length}` : b.kind === "callout" ? `callout:${b.variant}` : b.kind,
  );
}

/** v2Summary reads the same summary off the rendered top-level elements. */
function v2Summary(root: Element): string[] {
  return Array.from(root.children).map((el) => {
    const tag = el.tagName.toLowerCase();
    if (tag === "h2") return "heading";
    if (tag === "p") return "para";
    if (tag === "ul" && el.classList.contains("cn-list")) return `list:${el.children.length}`;
    if (el.classList.contains("cn-callout")) return `callout:${el.classList.contains("ds-card--violet") ? "warn" : "plain"}`;
    if (el.classList.contains("cn-table")) return "table";
    if (tag === "ol") return `ol:${el.children.length}`;
    return tag;
  });
}

// vitest runs in web/, so the curriculum is one level up.
const COURSES = resolve(process.cwd(), "../curriculum/courses");

/** Every concept body (incl. *.when.md) and every problem section file. */
function markdownFiles(dir: string): string[] {
  const out: string[] = [];
  for (const name of readdirSync(dir).sort()) {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) out.push(...markdownFiles(path));
    else if (name.endsWith(".md")) out.push(path);
  }
  return out;
}

const PARITY_FILES = markdownFiles(COURSES)
  .map((p) => relative(COURSES, p))
  .filter((rel) => /^[^/]+\/concepts\/[^/]+\.md$/.test(rel) || /^[^/]+\/items\/[^/]+\/sections\/.+\.md$/.test(rel));

/** Files whose v2 block structure legitimately differs from v1 because v1 lacked the
 *  feature (ordered lists, fenced code, tables, lazy continuation lines). Each entry
 *  must still differ, so a stale entry fails. */
const V1_DIFFERS: Record<string, string> = {
  "dsa/items/16/sections/solution/02-steps.md":
    "a `1)`-style ordered list: v1 had no ordered lists and ran the five steps together as one paragraph",
};

describe("Markdown v1 parity (curriculum/)", () => {
  it("finds the concept bodies and problem sections", () => {
    expect(PARITY_FILES.some((f) => f.includes("/concepts/"))).toBe(true);
    expect(PARITY_FILES.some((f) => f.includes("/sections/"))).toBe(true);
  });

  it.each(PARITY_FILES)("%s keeps v1's block structure", (rel) => {
    const source = readFileSync(join(COURSES, rel), "utf8");
    const { container } = renderSafe(source);
    const v2 = v2Summary(container.firstElementChild!);
    if (rel in V1_DIFFERS) expect(v2).not.toEqual(v1Summary(source));
    else expect(v2).toEqual(v1Summary(source));
  });

  it("matches the recorded v2 block summaries", () => {
    const summaries = Object.fromEntries(
      PARITY_FILES.map((rel) => {
        const { container, unmount } = render(<Markdown source={readFileSync(join(COURSES, rel), "utf8")} />);
        const s = v2Summary(container).join(" ");
        unmount();
        return [rel, s];
      }),
    );
    expect(summaries).toMatchSnapshot();
  });
});

