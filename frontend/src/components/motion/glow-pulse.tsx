"use client";

import type { ReactNode } from "react";
import { motion } from "motion/react";
import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "./use-prefers-reduced-motion";
import { DURATION, GLOW_COLOR, type GlowColor } from "./tokens";

export interface GlowPulseProps {
  children: ReactNode;
  /** Default `"cyan"`. Reach for `"rose"` only for an urgent state (e.g. a rate-limit warning). */
  color?: GlowColor;
  className?: string;
}

/**
 * 2.4s breathing glow (plan.md §1.4 — the exact cycle length is fixed, not
 * a default you should override). **Ration this to the single active
 * element on a screen at a time** — the current roadmap node, Byte's chip
 * when he has something to say. Wrapping more than one element per
 * viewport defeats the point, and breaks the "never more than two things
 * animating at once" rule on its own.
 *
 * Implemented as an absolutely-positioned glow layer behind `children`
 * whose *opacity* breathes (compositor-friendly) rather than animating
 * `box-shadow` directly every frame.
 *
 * ```tsx
 * <GlowPulse color="cyan" className="rounded-tv-card">
 *   <RoadmapNode active />
 * </GlowPulse>
 * ```
 */
export function GlowPulse({ children, color = "cyan", className }: GlowPulseProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const rgb = GLOW_COLOR[color];
  const peakGlow = `0 0 22px 4px rgba(${rgb},0.35)`;

  return (
    <div className={cn("relative rounded-tv-card", className)}>
      {prefersReducedMotion ? (
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 rounded-[inherit]"
          style={{ boxShadow: `0 0 15px 2px rgba(${rgb},0.22)` }}
        />
      ) : (
        <motion.div
          aria-hidden
          className="pointer-events-none absolute inset-0 rounded-[inherit]"
          style={{ boxShadow: peakGlow }}
          animate={{ opacity: [0.35, 1, 0.35] }}
          transition={{
            duration: DURATION.glowPulseCycle / 1000,
            repeat: Infinity,
            ease: "easeInOut",
          }}
        />
      )}
      {children}
    </div>
  );
}
