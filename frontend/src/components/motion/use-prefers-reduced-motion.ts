"use client";

import { useReducedMotion } from "motion/react";

/**
 * Coalesced boolean version of `motion/react`'s `useReducedMotion()`
 * (which returns `boolean | null` — `null` before mount, to avoid a
 * hydration mismatch). Every primitive in this library uses this instead
 * of the raw hook so `if (prefersReducedMotion)` branches never need a
 * null check.
 *
 * Use this in any custom `motion/react` animation. CSS-only transitions/
 * animations are already covered globally (`globals.css` collapses
 * `animation-duration`/`transition-duration` to ~0 under
 * `prefers-reduced-motion: reduce`) — you don't need this hook for those.
 */
export function usePrefersReducedMotion(): boolean {
  return useReducedMotion() ?? false;
}
