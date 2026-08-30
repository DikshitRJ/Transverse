"use client";

/**
 * Thin wrapper over the canonical sanitizer at `@/components/content/safe-html`.
 *
 * This originally used `DOMPurify` + `dangerouslySetInnerHTML`. DOMPurify is a
 * good library, but it was never a declared dependency here — it was present
 * only as a transitive dependency of `monaco-editor` (`npm ls dompurify`
 * confirms). Resting the XSS boundary for scraped LeetCode/Codeforces HTML on
 * a package nothing in `package.json` requires is not acceptable: a Monaco
 * upgrade or removal would break sanitization silently.
 *
 * Consolidated during Wave-2 merge onto the one implementation that needs no
 * dependency at all and never calls `dangerouslySetInnerHTML`. See plan.md
 * §9.5c item 1.
 *
 * Kept as a wrapper (rather than rewriting call sites) because it adds
 * `className` support and tolerates undefined html.
 */
import { SafeHtml } from "@/components/content/safe-html";

export interface SanitizedHtmlProps {
  html: string | undefined;
  className?: string;
}

export function SanitizedHtml({ html, className }: SanitizedHtmlProps) {
  if (!html) return null;
  return (
    <div className={className}>
      <SafeHtml html={html} />
    </div>
  );
}
