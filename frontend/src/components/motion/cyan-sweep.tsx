"use client";

import { AnimatePresence, motion } from "motion/react";
import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "./use-prefers-reduced-motion";
import { DURATION, EASE } from "./tokens";

export type SweepEdge = "top" | "bottom" | "left" | "right";

export interface CyanSweepProps {
  /** Controlled — typically the `active` value from `useHoverActive()`. */
  active: boolean;
  /** Which edge the line traverses. Default `"bottom"`. */
  edge?: SweepEdge;
  className?: string;
}

const VERTICAL_EDGES: readonly SweepEdge[] = ["left", "right"];

const EDGE_POSITION: Record<SweepEdge, string> = {
  top: "inset-x-0 top-0 h-px w-full",
  bottom: "inset-x-0 bottom-0 h-px w-full",
  left: "inset-y-0 left-0 h-full w-px",
  right: "inset-y-0 right-0 h-full w-px",
};

/**
 * The app's default "alive" gesture (plan.md §1.4): a 1px cyan line that
 * traverses a card edge on hover/focus. Fully controlled — drive `active`
 * from `useHoverActive()`, a manual boolean, or any other trigger you own.
 *
 * Drop it as a child of a `position: relative` container; it positions
 * itself absolutely along one edge and clips its own travel, so it never
 * causes the parent to reflow or scroll.
 *
 * ```tsx
 * const { active, bind } = useHoverActive();
 * <div {...bind} className="relative">
 *   {children}
 *   <CyanSweep active={active} edge="bottom" />
 * </div>
 * ```
 *
 * Reduced motion: the traveling comet is skipped entirely; only a static
 * dim cyan line fades in/out (opacity only, ~150ms) — an instant-enough
 * state change that still reads as "this is active."
 */
export function CyanSweep({ active, edge = "bottom", className }: CyanSweepProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const vertical = VERTICAL_EDGES.includes(edge);

  return (
    <span
      aria-hidden
      className={cn("pointer-events-none absolute overflow-hidden", EDGE_POSITION[edge], className)}
    >
      {/* persistent dim line while active — the reduced-motion fallback too */}
      <motion.span
        className="absolute inset-0 bg-tv-cyan"
        initial={false}
        animate={{ opacity: active ? 0.35 : 0 }}
        transition={{ duration: 0.15 }}
      />

      {/* traveling comet highlight — one pass per activation, skipped under reduced motion */}
      {!prefersReducedMotion && (
        <AnimatePresence>
          {active && (
            <motion.span
              key="comet"
              className={cn(
                "absolute bg-tv-cyan-pure shadow-[0_0_8px_2px_rgba(0,255,255,0.8)]",
                vertical ? "left-0 h-1/3 w-full" : "top-0 h-full w-1/3",
              )}
              initial={vertical ? { y: "-100%", opacity: 0 } : { x: "-100%", opacity: 0 }}
              animate={
                vertical
                  ? { y: "300%", opacity: [0, 1, 1, 0] }
                  : { x: "300%", opacity: [0, 1, 1, 0] }
              }
              exit={{ opacity: 0, transition: { duration: 0.15 } }}
              transition={{ duration: DURATION.sweep / 1000, ease: EASE.standard }}
            />
          )}
        </AnimatePresence>
      )}
    </span>
  );
}
