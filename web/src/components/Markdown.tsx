import { Fragment, createContext, useContext, useEffect, useState, type ReactNode } from "react";
import ReactMarkdown, { type Components, type ExtraProps, type UrlTransform } from "react-markdown";
import remarkGfm from "remark-gfm";
import { Icon } from "./Icon";
import { codeLang, type CodeLang } from "./codeLang";

// The sanitizing Markdown renderer for curriculum reading (ADR-0037): concept bodies
// and problem sections. The reader-side profile (t1 §7.2) is CommonMark + GFM tables
// and fenced code, images only via `asset:` refs, https links only, no raw HTML.
//
// It builds React nodes — never dangerouslySetInnerHTML, never an HTML string:
//   - raw HTML is dropped (`skipHtml`), and only the elements in ALLOWED survive
//     (others are unwrapped to their children);
//   - every element is rendered by a component below that emits only the attributes
//     it names, so no stray attribute (on*, style, id, class) reaches the DOM;
//   - links keep only `https://` hrefs (target=_blank, rel=noopener noreferrer
//     nofollow); any other link renders as its text;
//   - images keep only `asset:<name>` refs that resolveAsset turns into a URL, and
//     render only as <img> (never inline SVG); anything else renders as its alt text;
//   - fenced code renders as plain text at once; a lazily imported highlighter
//     (./highlight) then swaps in `hljs-*` token spans styled from app.css (CSP-safe).
// The v1 "> [!warning]" / "> **Watch out**" blockquote stays the violet callout.

/** The only elements the renderer may emit (anything else is unwrapped). */
const ALLOWED = [
  "p", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "blockquote", "strong", "em", "del",
  "code", "pre", "a", "img", "table", "thead", "tbody", "tr", "th", "td", "hr", "br",
];

/** A safe asset name: path segments of [A-Za-z0-9._-] that start alphanumeric — so
 *  no scheme, no leading slash, no backslash, no percent-encoding, no `..`. */
const ASSET_NAME = /^[A-Za-z0-9][A-Za-z0-9._-]*(?:\/[A-Za-z0-9][A-Za-z0-9._-]*)*$/;

/** assetName returns the name of a safe `asset:<name>` ref, or null. */
function assetName(src: unknown): string | null {
  if (typeof src !== "string" || !src.startsWith("asset:")) return null;
  const name = src.slice("asset:".length);
  if (name.length > 200 || !ASSET_NAME.test(name) || name.split("/").includes("..")) return null;
  return name;
}

/** safeHref keeps an https link (scheme normalized to lower case), else null. */
function safeHref(url: unknown): string | null {
  if (typeof url !== "string" || !/^https:\/\//i.test(url)) return null;
  return "https://" + url.slice("https://".length);
}

/** urlTransform runs on every URL attribute before rendering: an <a> keeps only
 *  https, an <img> keeps only a safe `asset:` ref (resolved later by Img), and
 *  every other URL attribute is emptied. */
const urlTransform: UrlTransform = (url, key, node) => {
  if (key === "href" && node.tagName === "a") return safeHref(url) ?? "";
  if (key === "src" && node.tagName === "img") return assetName(url) !== null ? url : "";
  return "";
};

// --- the callout pass (a tiny rehype plugin) ---

/** The subset of hast the callout pass reads. */
type HNode = {
  type: string;
  value?: string;
  tagName?: string;
  children?: HNode[];
  data?: Record<string, unknown>;
};

type CalloutInfo = { variant: "warn" | "plain"; title: string };
const CALLOUT = "xlCallout";
// A leading "[!warning]"-style marker, as v1's ADMONITION matched it.
const MARKER = /^\s*\[!(\w+)\]\s*/;

function textOf(n: HNode | undefined): string {
  if (!n) return "";
  return n.type === "text" ? (n.value ?? "") : (n.children ?? []).map(textOf).join("");
}

/** markCallout strips a leading admonition ("[!warning]" or a bold title) from a
 *  blockquote's first paragraph and records the callout variant + title in `data`. */
function markCallout(bq: HNode): void {
  // Drop the whitespace text between block children (v1's callout had none).
  const kids = (bq.children ?? []).filter((c) => c.type !== "text" || (c.value ?? "").trim() !== "");
  bq.children = kids;
  const p = kids.find((c) => c.type === "element");
  let kind = "";
  let boldTitle = "";
  if (p?.tagName === "p" && p.children && p.children.length > 0) {
    const first = p.children[0];
    if (first?.type === "text") {
      const m = MARKER.exec(first.value ?? "");
      if (m) {
        kind = (m[1] ?? "").toLowerCase();
        first.value = (first.value ?? "").slice(m[0].length);
      }
    } else if (first?.type === "element" && first.tagName === "strong") {
      boldTitle = textOf(first).trim();
      p.children.shift();
      const next = p.children[0];
      if (next?.type === "text") next.value = (next.value ?? "").replace(/^[:.]?\s*/, "");
    }
    p.children = p.children.filter((c) => c.type !== "text" || (c.value ?? "") !== "");
    if (p.children.every((c) => c.type === "text" && (c.value ?? "").trim() === "")) {
      bq.children = kids.filter((c) => c !== p);
    }
  }
  const isWarn = kind === "warning" || kind === "caution" || kind === "warn" || /watch out/i.test(boldTitle);
  const info: CalloutInfo = { variant: isWarn ? "warn" : "plain", title: boldTitle || (isWarn ? "Watch out" : "") };
  bq.data = { ...bq.data, [CALLOUT]: info };
}

function walkCallouts(n: HNode): void {
  if (n.type === "element" && n.tagName === "blockquote") markCallout(n);
  for (const c of n.children ?? []) walkCallouts(c);
}

function rehypeCallouts() {
  return (tree: HNode) => walkCallouts(tree);
}

// --- element components (each emits only the attributes it names) ---

type Kids = { children?: ReactNode };
type HastProps = Kids & ExtraProps;

// Attribute-free elements: markdown-derived ids/classes (footnotes, task lists) are dropped.
const P = ({ children }: Kids) => <p>{children}</p>;
const Li = ({ children }: Kids) => <li>{children}</li>;
const Del = ({ children }: Kids) => <del>{children}</del>;
const Tr = ({ children }: Kids) => <tr>{children}</tr>;
const Thead = ({ children }: Kids) => <thead>{children}</thead>;
const Tbody = ({ children }: Kids) => <tbody>{children}</tbody>;
const Hr = () => <hr />;
const Br = () => <br />;
// v1 (and InlineMD, and the `.cn p b` styles) use <b>/<i>.
const Strong = ({ children }: Kids) => <b>{children}</b>;
const Em = ({ children }: Kids) => <i>{children}</i>;
const InlineCode = ({ children }: Kids) => <code>{children}</code>;
const Ul = ({ children }: Kids) => <ul className="cn-list">{children}</ul>;
const Ol = ({ children }: Kids) => <ol className="cn-list">{children}</ol>;
// Wide tables scroll inside their own box instead of widening the article.
const Table = ({ children }: Kids) => (
  <div className="cn-table">
    <table>{children}</table>
  </div>
);

/** Table cells keep only GFM column alignment, as a class. */
function cellClass(node: ExtraProps["node"]): string | undefined {
  const align = node?.properties?.align;
  return align === "center" || align === "right" ? `cn-t--${align}` : undefined;
}

function Th({ children, node }: HastProps) {
  return <th className={cellClass(node)}>{children}</th>;
}

function Td({ children, node }: HastProps) {
  return <td className={cellClass(node)}>{children}</td>;
}

/** Every heading level renders as the article's h2 (v1 parity; `.cn h2`). */
function Heading({ children }: Kids) {
  return <h2>{children}</h2>;
}

function Link({ href, children }: Kids & { href?: string }) {
  const url = safeHref(href);
  if (url === null) return <>{children}</>;
  return (
    <a href={url} target="_blank" rel="noopener noreferrer nofollow">
      {children}
    </a>
  );
}

type AssetResolver = (ref: string) => string | null;
const AssetContext = createContext<AssetResolver | undefined>(undefined);

/** A resolved asset URL must be same-origin (`/…`, not `//…`) or https. */
const RESOLVED_URL = /^(?:\/(?!\/)|https:\/\/)/;

function Img({ src, alt }: { src?: unknown; alt?: string }) {
  const resolve = useContext(AssetContext);
  const name = assetName(src);
  const url = name !== null && resolve ? resolve(name) : null;
  if (typeof url !== "string" || !RESOLVED_URL.test(url)) return <>{alt ?? ""}</>;
  return <img src={url} alt={alt ?? ""} />;
}

type Highlighted = { code: string; lang: CodeLang; nodes: ReactNode[] };

/** CodeBlock renders fenced code as plain text at once, then swaps in highlighted
 *  token spans once the lazy highlighter chunk has loaded (registered languages only). */
function CodeBlock({ code, info }: { code: string; info: string | undefined }) {
  const lang = codeLang(info);
  const [hl, setHl] = useState<Highlighted | null>(null);
  useEffect(() => {
    if (lang === null) return;
    let live = true;
    import("./highlight")
      .then((m) => {
        if (live) setHl({ code, lang, nodes: m.highlight(code, lang) });
      })
      .catch(() => {
        // Highlighting is cosmetic: on a failed chunk load the plain text stays.
      });
    return () => {
      live = false;
    };
  }, [code, lang]);
  const nodes = hl !== null && hl.code === code && hl.lang === lang ? hl.nodes : code;
  return (
    <pre className="xl-code">
      <code className={lang !== null ? `language-${lang}` : undefined}>{nodes}</code>
    </pre>
  );
}

/** Pre reads the fence (text + info-string language) off the hast node; the info
 *  string is data — it only ever reaches codeLang(). */
function Pre({ node }: ExtraProps) {
  const el = node as unknown as HNode | undefined;
  const codeEl = el?.children?.find((c) => c.type === "element" && c.tagName === "code");
  const cls = (codeEl as { properties?: { className?: unknown } } | undefined)?.properties?.className;
  const langClass = Array.isArray(cls) ? cls.find((c): c is string => typeof c === "string" && c.startsWith("language-")) : undefined;
  const code = textOf(codeEl ?? el).replace(/\n$/, "");
  return <CodeBlock code={code} info={langClass?.slice("language-".length)} />;
}

/** Callout renders a blockquote as the design system's callout card (v1 DOM). */
function Callout({ node, children }: HastProps) {
  const data = (node as unknown as HNode | undefined)?.data?.[CALLOUT] as CalloutInfo | undefined;
  const info = data ?? { variant: "plain", title: "" };
  const warn = info.variant === "warn";
  return (
    <div className={warn ? "ds-card ds-card--violet cn-callout" : "ds-card cn-callout"}>
      {info.title && (
        <div className="cn-callout__h">
          {warn && <Icon name="alert" className="xl-ico--sm" />}
          <span>{info.title}</span>
        </div>
      )}
      {children}
    </div>
  );
}

const COMPONENTS: Components = {
  p: P,
  h1: Heading,
  h2: Heading,
  h3: Heading,
  h4: Heading,
  h5: Heading,
  h6: Heading,
  ul: Ul,
  ol: Ol,
  li: Li,
  blockquote: Callout,
  strong: Strong,
  em: Em,
  del: Del,
  code: InlineCode,
  pre: Pre,
  a: Link,
  img: Img,
  table: Table,
  thead: Thead,
  tbody: Tbody,
  tr: Tr,
  th: Th,
  td: Td,
  hr: Hr,
  br: Br,
};

const REMARK = [remarkGfm];
const REHYPE = [rehypeCallouts];

/** Markdown renders sanitized curriculum reading (the ADR-0037 profile). Wrap it in
 *  a `.cn` container for the article typography (see app.css). `resolveAsset` maps
 *  an image's `asset:<name>` ref (it receives `<name>`) to a same-origin or https
 *  URL, or null; without it, or when it returns null, the image renders as its alt text. */
export function Markdown({ source, resolveAsset }: { source: string; resolveAsset?: AssetResolver }) {
  return (
    <AssetContext.Provider value={resolveAsset}>
      <ReactMarkdown
        remarkPlugins={REMARK}
        rehypePlugins={REHYPE}
        allowedElements={ALLOWED}
        unwrapDisallowed
        skipHtml
        urlTransform={urlTransform}
        components={COMPONENTS}
      >
        {source ?? ""}
      </ReactMarkdown>
    </AssetContext.Provider>
  );
}

// --- InlineMD: one string's inline `code` / **bold** / *italic* (hand-written, v1) ---
// It only ever creates <code>/<b>/<i> elements and text nodes, so it needs no sanitizing.

function renderInline(text: string, keyBase: string): ReactNode[] {
  const out: ReactNode[] = [];
  const codeRe = /`([^`]+)`/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let k = 0;
  while ((m = codeRe.exec(text)) !== null) {
    if (m.index > last) out.push(...renderEmphasis(text.slice(last, m.index), `${keyBase}-t${k++}`));
    out.push(<code key={`${keyBase}-c${k++}`}>{m[1]}</code>);
    last = m.index + m[0].length;
  }
  if (last < text.length) out.push(...renderEmphasis(text.slice(last), `${keyBase}-t${k}`));
  return out;
}

function renderEmphasis(text: string, keyBase: string): ReactNode[] {
  const out: ReactNode[] = [];
  const re = /\*\*([^*]+)\*\*|\*([^*]+)\*|_([^_]+)_/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let k = 0;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) out.push(<Fragment key={`${keyBase}-p${k++}`}>{text.slice(last, m.index)}</Fragment>);
    const bold = m[1];
    const italic = m[2] ?? m[3];
    if (bold !== undefined) {
      out.push(<b key={`${keyBase}-b${k++}`}>{bold}</b>);
    } else {
      out.push(<i key={`${keyBase}-i${k++}`}>{italic}</i>);
    }
    last = m.index + m[0].length;
  }
  if (last < text.length) out.push(<Fragment key={`${keyBase}-p${k}`}>{text.slice(last)}</Fragment>);
  return out;
}

/** InlineMD renders one string's inline formatting (`code`, **bold**, *italic*) —
 *  for single-paragraph copy like a callout, where block wrappers aren't wanted. */
export function InlineMD({ text }: { text: string }) {
  return <>{renderInline(text ?? "", "inline")}</>;
}
