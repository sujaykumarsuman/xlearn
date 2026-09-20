import { Fragment, type ReactNode } from "react";
import { Icon } from "./Icon";

// A small, dependency-free, sanitized Markdown renderer for curriculum reading
// (concept body_md). It builds React nodes directly — never dangerouslySetInnerHTML —
// so page text can never inject markup regardless of the source. It supports the
// subset the seed content uses: headings, paragraphs, unordered lists, inline
// **bold** / *italic* / `code`, and a "> [!warning]" (or "> **Watch out**")
// blockquote rendered as the design system's violet "Watch out" callout. Anything
// it doesn't recognize falls through as plain text.

/** A block-level chunk of the source, split on blank lines. */
type Block =
  | { kind: "heading"; text: string }
  | { kind: "list"; items: string[] }
  | { kind: "callout"; variant: "warn" | "plain"; title: string; lines: string[] }
  | { kind: "para"; text: string };

const BULLET = /^\s*[-*]\s+/;
const HEADING = /^\s*#{1,6}\s+/;
const QUOTE = /^\s*>\s?/;
// A leading admonition marker: "[!warning]"/"[!caution]" or a bold "**Watch out**".
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

    // Heading — a single line.
    if (HEADING.test(line)) {
      blocks.push({ kind: "heading", text: line.replace(HEADING, "").trim() });
      i++;
      continue;
    }

    // Blockquote → callout (consume consecutive "> " lines).
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

    // Unordered list (consume consecutive bullet lines).
    if (BULLET.test(line)) {
      const items: string[] = [];
      while (i < lines.length && BULLET.test(at(i))) {
        items.push(at(i).replace(BULLET, "").trim());
        i++;
      }
      blocks.push({ kind: "list", items });
      continue;
    }

    // Paragraph (consume consecutive non-blank, non-structural lines).
    const para: string[] = [];
    for (let l = at(i); i < lines.length && l.trim() !== "" && !BULLET.test(l) && !HEADING.test(l) && !QUOTE.test(l); l = at(i)) {
      para.push(l.trim());
      i++;
    }
    blocks.push({ kind: "para", text: para.join(" ") });
  }

  return blocks;
}

// --- inline parsing: `code`, then **bold**, then *italic* / _italic_ ---

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

/** Markdown renders sanitized curriculum reading. Wrap it in a `.cn` container for
 *  the article typography (see app.css). */
export function Markdown({ source }: { source: string }) {
  const blocks = splitBlocks(source ?? "");
  return (
    <>
      {blocks.map((b, i) => {
        const key = `b${i}`;
        switch (b.kind) {
          case "heading":
            return <h2 key={key}>{renderInline(b.text, key)}</h2>;
          case "list":
            return (
              <ul key={key} className="cn-list">
                {b.items.map((it, j) => (
                  <li key={`${key}-${j}`}>{renderInline(it, `${key}-${j}`)}</li>
                ))}
              </ul>
            );
          case "callout":
            return (
              <div
                key={key}
                className={b.variant === "warn" ? "ds-card ds-card--violet cn-callout" : "ds-card cn-callout"}
              >
                {b.title && (
                  <div className="cn-callout__h">
                    {b.variant === "warn" && <Icon name="alert" className="xl-ico--sm" />}
                    <span>{b.title}</span>
                  </div>
                )}
                {b.lines.map((l, j) => (
                  <p key={`${key}-${j}`}>{renderInline(l, `${key}-${j}`)}</p>
                ))}
              </div>
            );
          default:
            return <p key={key}>{renderInline(b.text, key)}</p>;
        }
      })}
    </>
  );
}
