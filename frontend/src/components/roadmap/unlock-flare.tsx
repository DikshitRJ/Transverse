"use client";

import { motion, useReducedMotion } from "motion/react";

/**
 * The "glow flare" beat of the unlock moment (plan.md §1.4): a brief cyan
 * radial burst over a node that just settled. Purely decorative/`aria-hidden`
 * — mount only for the ~400ms window, unmount after. Skips entirely under
 * `prefers-reduced-motion` (collapses the moment to an instant state change).
 */
export function UnlockFlare() {
  const reduceMotion = useReducedMotion();
  if (reduceMotion) return null;

  return (
    <motion.div
      aria-hidden
      className="pointer-events-none absolute inset-0 z-20 rounded-tv-card"
      style={{
        background: "radial-gradient(circle, rgba(0,242,255,0.55) 0%, rgba(0,242,255,0) 70%)",
      }}
      initial={{ opacity: 0, scale: 0.6 }}
      animate={{ opacity: [0, 1, 0], scale: [0.6, 1.15, 1.3] }}
      transition={{ duration: 0.4, ease: "easeOut", times: [0, 0.35, 1] }}
    />
  );
}
