"use client";

/**
 * Thin wrapper over the canonical sanitizer at `@/components/content/safe-html`.
 *
 * This originally carried its own `DOMParser` allowlist implementation in
 * `_lib/sanitize-html.ts`, written because `rehype-raw` is not installed and
 * `react-markdown` silently drops raw HTML without it. Three other agents hit
 * the same wall and solved it three more ways; all four were consolidated onto
 * one implementation during Wave-2 merge. See plan.md §9.5c item 1.
 *
 * Kept as a wrapper (rather than deleting it and rewriting call sites) because
 * it adds `className` support and tolerates null/undefined html, neither of
 * which the canonical `SafeHtml` handles.
 */
import { SafeHtml } from "@/components/content/safe-html";

export function SanitizedHtml({
  html,
  className,
}: {
  html: string | undefined | null;
  className?: string;
}) {
  if (!html) return null;
  return (
    <div className={className}>
      <SafeHtml html={html} />
    </div>
  );
}
