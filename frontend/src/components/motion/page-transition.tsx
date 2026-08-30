"use client";

import type { ReactNode } from "react";
import { AnimatePresence, motion, type Variants } from "motion/react";
import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "./use-prefers-reduced-motion";
import { DURATION, EASE } from "./tokens";

export interface PageTransitionProps {
  /** Unique per screen — e.g. the pathname from `usePathname()`. Changing it triggers the transition. */
  transitionKey: string;
  children: ReactNode;
  className?: string;
}

/**
 * Subtle route-level transition (plan.md §1.4 — every entrance ≤400ms).
 * Fades + lifts incoming content ~8px over 300ms; the outgoing content
 * fades out faster (200ms) so there's never a moment where both are
 * visibly competing for attention.
 *
 * Next's `layout.tsx` doesn't remount on navigation, so mount this inside
 * a route's `template.tsx` (which does) with `usePathname()` as the key —
 * or anywhere else you already control remount timing.
 *
 * ```tsx
 * // app/(app)/template.tsx
 * "use client";
 * export default function Template({ children }: { children: React.ReactNode }) {
 *   const pathname = usePathname();
 *   return <PageTransition transitionKey={pathname}>{children}</PageTransition>;
 * }
 * ```
 */
export function PageTransition({ transitionKey, children, className }: PageTransitionProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const d = (ms: number) => (prefersReducedMotion ? 0 : ms / 1000);

  const variants: Variants = {
    initial: { opacity: 0, y: 8 },
    animate: {
      opacity: 1,
      y: 0,
      transition: { duration: d(DURATION.pageTransitionMs), ease: EASE.standard },
    },
    exit: {
      opacity: 0,
      y: -4,
      transition: { duration: d(DURATION.pageTransitionExitMs), ease: EASE.standard },
    },
  };

  return (
    <AnimatePresence mode="wait" initial={false}>
      <motion.div
        key={transitionKey}
        className={cn(className)}
        variants={variants}
        initial="initial"
        animate="animate"
        exit="exit"
      >
        {children}
      </motion.div>
    </AnimatePresence>
  );
}
