"use client";

/**
 * Renders untrusted, scraped HTML (tutorial/problem bodies — plan.md's
 * scraper output, LeetCode/Codeforces/tutorial-site markup) without ever
 * touching `dangerouslySetInnerHTML`.
 *
 * Why not the `react-markdown` + `rehype-sanitize` pipeline FOUNDATION.md
 * points at: that pipeline sanitizes HAST produced by parsing *Markdown*.
 * Bridging raw HTML strings (what this content actually is — see
 * `mocks/fixtures/problems.ts`) into that tree requires `rehype-raw`,
 * which is not in `package.json`/`node_modules` as of this writing (only
 * `rehype-sanitize` + `remark-gfm` are installed) — without it,
 * `react-markdown` silently drops embedded HTML tags rather than
 * rendering them. Rather than reach into `package.json` (off-limits —
 * report gaps instead of patching shared config), this parses the HTML
 * with the browser's own parser (`DOMParser`, which handles malformed
 * markup exactly like a browser would — safer than a hand-rolled regex
 * tokenizer) and rebuilds only an allowlisted subset as real React
 * elements. Anything not on the allowlist is dropped; a fixed set of
 * genuinely dangerous containers (`script`, `style`, `iframe`, ...) is
 * dropped *with* its contents rather than unwrapped.
 *
 * FORGE is building an equivalent at `components/content/safe-html.tsx`
 * with the same `{ html }` signature (to be reconciled to one at merge)
 * — if they hit the same `rehype-raw` gap, this file's approach is a
 * reasonable drop-in.
 *
 * `DOMParser` is a browser API — this component only parses after mount
 * (`mounted` gate below) so it never runs during SSR.
 */

import { createElement, useEffect, useMemo, useState, type ReactNode } from "react";
import { Skeleton } from "@/components/ui/skeleton";

/** Tags rendered as real elements. Everything else is unwrapped (children kept) unless listed in DROP_WITH_CONTENTS. */
const ALLOWED_TAGS = new Set([
  "p",
  "br",
  "hr",
  "strong",
  "b",
  "em",
  "i",
  "u",
  "s",
  "del",
  "mark",
  "small",
  "code",
  "pre",
  "kbd",
  "samp",
  "ul",
  "ol",
  "li",
  "blockquote",
  "a",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "table",
  "thead",
  "tbody",
  "tfoot",
  "tr",
  "th",
  "td",
  "span",
  "sup",
  "sub",
  "img",
]);

/** Dropped entirely, contents included — never unwrapped into text. */
const DROP_WITH_CONTENTS = new Set([
  "script",
  "style",
  "iframe",
  "object",
  "embed",
  "noscript",
  "template",
  "form",
  "input",
  "button",
  "select",
  "option",
  "textarea",
  "link",
  "meta",
  "head",
  "title",
  "svg",
]);

const ALLOWED_ATTRS: Record<string, Set<string>> = {
  a: new Set(["href", "title"]),
  img: new Set(["src", "alt", "title", "width", "height"]),
  td: new Set(["colspan", "rowspan"]),
  th: new Set(["colspan", "rowspan"]),
};

/** Allowlist-only URL check for href/src — rejects `javascript:`, `data:`, `vbscript:`, etc. */
const SAFE_URL = /^(https?:|mailto:|\/|#)/i;

const TAG_CLASS: Partial<Record<string, string>> = {
  h1: "font-display text-h2 font-bold uppercase tracking-tight text-tv-text-hi mt-2",
  h2: "font-display text-h3 font-bold uppercase tracking-tight text-tv-text-hi mt-2",
  h3: "font-display text-h4 font-bold uppercase tracking-tight text-tv-text-hi mt-1",
  h4: "font-display text-body font-bold text-tv-text-hi",
  h5: "font-display text-sm font-bold text-tv-text-hi",
  h6: "font-display text-sm font-bold text-tv-text-hi",
  p: "text-body leading-relaxed text-tv-text-body",
  a: "text-tv-cyan underline underline-offset-2 hover:text-tv-cyan-pure",
  ul: "list-disc flex flex-col gap-1 pl-5 text-tv-text-body",
  ol: "list-decimal flex flex-col gap-1 pl-5 text-tv-text-body",
  li: "text-body text-tv-text-body",
  code: "rounded bg-tv-surface-2 px-1 py-0.5 font-mono text-xs text-tv-text-hi",
  pre: "overflow-x-auto rounded-tv-btn border border-tv-border bg-tv-surface-2 p-3 font-mono text-xs text-tv-text-hi",
  blockquote: "border-l-2 border-tv-border-cyan pl-3 text-tv-text-body/80 italic",
  table: "w-full border-collapse text-xs",
  th: "border border-tv-border px-2 py-1 text-left font-mono text-tv-text-hi",
  td: "border border-tv-border px-2 py-1 text-tv-text-body",
  img: "my-2 max-w-full rounded-tv-btn",
  hr: "border-tv-border",
};

function sanitizeUrl(value: string): string | null {
  const trimmed = value.trim();
  return SAFE_URL.test(trimmed) ? trimmed : null;
}

let keySeed = 0;

function nodeToReact(node: ChildNode): ReactNode {
  if (node.nodeType === Node.TEXT_NODE) {
    return node.textContent;
  }
  if (node.nodeType !== Node.ELEMENT_NODE) return null;

  const el = node as Element;
  const tag = el.tagName.toLowerCase();

  if (DROP_WITH_CONTENTS.has(tag)) return null;

  const children = Array.from(el.childNodes).map((child) => nodeToReact(child));

  if (!ALLOWED_TAGS.has(tag)) {
    // Not on the allowlist — drop the wrapper, keep sanitized children (e.g. <section>, <figure>, <font>).
    return children;
  }

  const props: Record<string, unknown> = { key: `n${keySeed++}` };
  const allowedAttrs = ALLOWED_ATTRS[tag];
  if (allowedAttrs) {
    for (const attr of Array.from(el.attributes)) {
      if (!allowedAttrs.has(attr.name)) continue;
      if (attr.name === "href" || attr.name === "src") {
        const safe = sanitizeUrl(attr.value);
        if (safe === null) continue;
        props[attr.name] = safe;
      } else {
        props[attr.name] = attr.value;
      }
    }
  }
  if (tag === "a") {
    props.target = "_blank";
    props.rel = "noopener noreferrer";
  }
  if (TAG_CLASS[tag]) props.className = TAG_CLASS[tag];

  if (tag === "img") return createElement(tag, props);
  if (tag === "br" || tag === "hr") return createElement(tag, props);

  return createElement(tag, props, ...children);
}

export interface SafeHtmlProps {
  html: string;
}

/** Sanitizes and renders an untrusted HTML string as React elements. Never uses `dangerouslySetInnerHTML`. */
export function SafeHtml({ html }: SafeHtmlProps) {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  const content = useMemo(() => {
    if (!mounted || typeof window === "undefined") return null;
    try {
      const doc = new DOMParser().parseFromString(html, "text/html");
      keySeed = 0;
      return Array.from(doc.body.childNodes).map((child) => nodeToReact(child));
    } catch {
      return null;
    }
  }, [mounted, html]);

  if (!mounted) {
    return (
      <div className="flex flex-col gap-2" aria-hidden>
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-4 w-2/3" />
      </div>
    );
  }

  return <div className="flex flex-col gap-3">{content}</div>;
}
