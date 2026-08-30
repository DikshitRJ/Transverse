"use client";

import type { ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Lock } from "lucide-react";
import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "./use-prefers-reduced-motion";
import { DURATION, EASE } from "./tokens";
import { useUnlockSequence, type UnlockStage } from "./use-unlock-sequence";

export interface UnlockTransitionProps {
  /** Flip this from `false` to `true` to play the sequence once. */
  unlocked: boolean;
  children: ReactNode;
  /** Replaces the default lucide `Lock` glyph shown while locked/unlocking. */
  lockIcon?: ReactNode;
  /** Fires once, exactly when the sequence settles into `"lifted"`. */
  onSequenceComplete?: () => void;
  className?: string;
}

const LIFT_VARIANTS = {
  rest: { y: 0, scale: 1 },
  lifted: { y: -6, scale: 1.015 },
};

const BADGE_VISIBLE_STAGES: readonly UnlockStage[] = ["locked", "ring", "flare"];

/**
 * The signature unlock moment (plan.md §1.4): a ring completes around a
 * lock glyph → a glow flare → the lock dissolves → the whole card lifts.
 * This is the canonical implementation — ATLAS's roadmap-local version is
 * meant to be replaced by this at merge. Boolean-driven, so it composes
 * with SSE (`node.unlocked`), a mock timer, a test harness — anything that
 * can produce a `boolean`.
 *
 * `children` always renders (this never hides content while locked — pair
 * it with your own dimmed/`pointer-events-none` styling on `children` for
 * the pre-unlock look, e.g. the `locked` Badge variant). This component
 * only owns the lock glyph + ring + lift chrome layered on top.
 *
 * Need custom per-stage visuals instead of this chrome? Use
 * `useUnlockSequence()` directly — same stage machine, no rendering
 * opinions.
 *
 * ```tsx
 * <UnlockTransition unlocked={node.status === "unlocked"} onSequenceComplete={() => toast.success("Section unlocked")}>
 *   <RoadmapNodeCard node={node} />
 * </UnlockTransition>
 * ```
 */
export function UnlockTransition({
  unlocked,
  children,
  lockIcon,
  onSequenceComplete,
  className,
}: UnlockTransitionProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const { stage } = useUnlockSequence(unlocked, { onComplete: onSequenceComplete });
  const showLockBadge = BADGE_VISIBLE_STAGES.includes(stage);

  return (
    <motion.div
      className={cn("relative", className)}
      variants={LIFT_VARIANTS}
      animate={stage === "lifted" ? "lifted" : "rest"}
      transition={{
        duration: prefersReducedMotion ? 0 : DURATION.unlockLiftMs / 1000,
        ease: EASE.standard,
      }}
    >
      {children}

      <AnimatePresence>
        {showLockBadge && (
          <motion.div
            key="unlock-badge"
            className="absolute top-3 right-3 flex size-8 items-center justify-center"
            exit={{
              opacity: 0,
              scale: 0.6,
              transition: { duration: prefersReducedMotion ? 0 : DURATION.unlockDissolveMs / 1000 },
            }}
          >
            <UnlockRing stage={stage} prefersReducedMotion={prefersReducedMotion} />
            <span className="relative z-10 flex size-5 items-center justify-center text-tv-locked">
              {lockIcon ?? <Lock className="size-3.5" strokeWidth={2.5} />}
            </span>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}

function UnlockRing({
  stage,
  prefersReducedMotion,
}: {
  stage: UnlockStage;
  prefersReducedMotion: boolean;
}) {
  const radius = 14;
  const circumference = 2 * Math.PI * radius;
  const ringDrawn = stage === "ring" || stage === "flare" || stage === "dissolve";

  return (
    <svg aria-hidden viewBox="0 0 32 32" className="absolute inset-0 size-8 -rotate-90">
      <circle cx="16" cy="16" r={radius} fill="none" stroke="var(--tv-border)" strokeWidth="2" />
      <motion.circle
        cx="16"
        cy="16"
        r={radius}
        fill="none"
        stroke="var(--tv-cyan-pure)"
        strokeWidth="2"
        strokeLinecap="round"
        strokeDasharray={circumference}
        initial={false}
        animate={{ strokeDashoffset: ringDrawn ? 0 : circumference }}
        transition={{
          duration: prefersReducedMotion ? 0 : DURATION.unlockRingMs / 1000,
          ease: EASE.standard,
        }}
      />
      {stage === "flare" && !prefersReducedMotion && (
        <motion.circle
          cx="16"
          cy="16"
          r={radius}
          fill="none"
          stroke="var(--tv-cyan-pure)"
          strokeWidth="2"
          initial={{ scale: 1, opacity: 0.9 }}
          animate={{ scale: 1.6, opacity: 0 }}
          transition={{ duration: DURATION.unlockFlareMs / 1000, ease: EASE.standard }}
          style={{ transformOrigin: "16px 16px" }}
        />
      )}
    </svg>
  );
}
