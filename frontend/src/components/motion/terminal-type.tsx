"use client";

import { cn } from "@/lib/utils";
import { useTypeOn, type UseTypeOnOptions } from "./use-type-on";

export interface TerminalTypeProps extends UseTypeOnOptions {
  text: string;
  className?: string;
  /** Render a blinking block cursor while typing. Default `true`. */
  showCursor?: boolean;
  as?: "span" | "p" | "div";
}

/**
 * Renders `text` typing in at plan.md §1.4's terminal speed. Wraps
 * `useTypeOn` — reach for the hook directly if you need the in-progress
 * string for something other than plain rendering (e.g. driving a second
 * element in sync).
 *
 * The accessible name is always the *complete* `text` (via `aria-label`)
 * — screen readers get the full string immediately rather than the
 * character-by-character reveal.
 *
 * ```tsx
 * <TerminalType text={`Transverse says: "Not sure? The quiz is a fun way to warm up!"`} className="text-tv-text-body" />
 * ```
 */
export function TerminalType({
  text,
  className,
  showCursor = true,
  as = "span",
  ...options
}: TerminalTypeProps) {
  const { displayedText, isTyping } = useTypeOn(text, options);
  const Tag = as;

  return (
    <Tag className={cn("font-mono", className)} aria-label={text}>
      <span aria-hidden="true">
        {displayedText}
        {showCursor && isTyping && (
          <span
            aria-hidden="true"
            className="ml-0.5 inline-block h-[1em] w-[0.5ch] translate-y-[0.15em] animate-pulse bg-tv-cyan align-baseline"
          />
        )}
      </span>
    </Tag>
  );
}
