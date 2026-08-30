/**
 * Re-export of the canonical sanitizer.
 *
 * This file originally held the implementation. During Wave-2 merge it was
 * promoted to `@/components/content/safe-html` as the single shared sanitizer
 * for all untrusted scraped HTML (tutorial bodies, problem statements), and
 * the three competing implementations written by other agents were retired
 * onto it. See plan.md §9.5c item 1.
 *
 * Kept as a re-export so existing imports and the test suite in
 * `./__tests__/safe-html.test.tsx` continue to resolve.
 */
export { SafeHtml, type SafeHtmlProps } from "@/components/content/safe-html";
