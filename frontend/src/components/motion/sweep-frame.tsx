"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { CyanSweep, type SweepEdge } from "./cyan-sweep";
import { useHoverActive } from "./use-hover-active";

export interface SweepFrameProps {
  children: ReactNode;
  /** Which edge the sweep traverses. Default `"bottom"`. */
  edge?: SweepEdge;
  className?: string;
}

/**
 * The zero-config drop-in for the cyan sweep: wraps `children` in a
 * `position: relative` frame that tracks its own hover/focus and wires up
 * `CyanSweep` for you. Use this when you just want "make this card feel
 * alive on hover" with no state management of your own; reach for
 * `useHoverActive()` + `<CyanSweep />` directly when you need the active
 * boolean for something else too (e.g. also driving a `GlowPulse`).
 *
 * ```tsx
 * <SweepFrame edge="bottom" className="rounded-tv-card">
 *   <Card>...</Card>
 * </SweepFrame>
 * ```
 */
export function SweepFrame({ children, edge = "bottom", className }: SweepFrameProps) {
  const { active, bind } = useHoverActive();

  return (
    // `onFocus`/`onBlur` bubble in React's synthetic event system, so this
    // reacts to focus landing on any interactive descendant without the
    // wrapper itself needing to be in the tab order.
    <div {...bind} className={cn("relative", className)}>
      {children}
      <CyanSweep active={active} edge={edge} />
    </div>
  );
}
