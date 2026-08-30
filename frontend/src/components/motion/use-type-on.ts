"use client";

import { useEffect, useRef, useState } from "react";
import { usePrefersReducedMotion } from "./use-prefers-reduced-motion";
import { DURATION } from "./tokens";

export interface UseTypeOnOptions {
  /** ms per character. Default plan.md §1.4's ~28ms/char — don't override to "faster", that's what reduced motion is for. */
  speedMs?: number;
  /** Set `false` to render `text` immediately without animating (e.g. a "skip" affordance you drive yourself). Default `true`. */
  enabled?: boolean;
}

export interface UseTypeOnResult {
  /** The text to actually render — grows one character at a time. */
  displayedText: string;
  isTyping: boolean;
  /** Immediately reveals the full `text`, ending the animation early. */
  skip: () => void;
}

/**
 * Types `text` in at plan.md §1.4's ~28ms/char (Byte's dialogue, verdict
 * copy). Resets and retypes whenever `text` changes. Reduced motion (or
 * `enabled: false`) collapses straight to the full string — never a
 * faster typing speed, per plan.md's rule.
 *
 * `<TerminalType>` (same directory) wraps this for the common case where
 * you just want to render styled, typing-in text with no extra state.
 */
export function useTypeOn(text: string, options: UseTypeOnOptions = {}): UseTypeOnResult {
  const { speedMs = DURATION.typeOnCharMs, enabled = true } = options;
  const prefersReducedMotion = usePrefersReducedMotion();
  const instant = prefersReducedMotion || !enabled;
  const [displayedLength, setDisplayedLength] = useState(() => (instant ? text.length : 0));
  const skippedRef = useRef(false);

  useEffect(() => {
    skippedRef.current = false;

    if (instant) {
      setDisplayedLength(text.length);
      return;
    }

    setDisplayedLength(0);
    if (text.length === 0) return;

    const id = setInterval(() => {
      setDisplayedLength((prev) => {
        if (skippedRef.current || prev >= text.length) {
          clearInterval(id);
          return text.length;
        }
        const next = prev + 1;
        if (next >= text.length) clearInterval(id);
        return next;
      });
    }, speedMs);

    return () => clearInterval(id);
  }, [text, instant, speedMs]);

  return {
    displayedText: text.slice(0, displayedLength),
    isTyping: displayedLength < text.length,
    skip: () => {
      skippedRef.current = true;
      setDisplayedLength(text.length);
    },
  };
}
