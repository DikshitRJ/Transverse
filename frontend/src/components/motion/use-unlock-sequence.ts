"use client";

import { useEffect, useRef, useState } from "react";
import { usePrefersReducedMotion } from "./use-prefers-reduced-motion";
import { DURATION } from "./tokens";

export type UnlockStage = "locked" | "ring" | "flare" | "dissolve" | "lifted";

export interface UseUnlockSequenceOptions {
  /** Fires once, exactly when the sequence settles into `"lifted"`. */
  onComplete?: () => void;
}

export interface UseUnlockSequenceResult {
  stage: UnlockStage;
  /** True only while the ring/flare/dissolve leg is actively playing. */
  isSequencing: boolean;
}

/**
 * Drives the signature "unlock" moment (plan.md §1.4): ring completes →
 * glow flare → lock dissolves → card lifts. This is a pure state machine —
 * render whatever you like off `stage`. `<UnlockTransition>` (same
 * directory) is the turnkey wrapper if you don't need custom visuals per
 * stage.
 *
 * The sequence plays exactly once, on the `false → true` transition of
 * `unlocked` (e.g. the moment a `node.unlocked` SSE event flips your local
 * state). Mounting already-`unlocked` renders straight into the `"lifted"`
 * end state with no replay — a page load with 6 already-unlocked nodes
 * must not fire 6 unlock animations at once.
 *
 * Reduced motion collapses the whole sequence: `stage` jumps directly from
 * `"locked"` to `"lifted"`, `onComplete` still fires. Consumers don't need
 * their own reduced-motion branch for the *sequencing* — only for any
 * extra flourish they layer on top.
 */
export function useUnlockSequence(
  unlocked: boolean,
  { onComplete }: UseUnlockSequenceOptions = {},
): UseUnlockSequenceResult {
  const prefersReducedMotion = usePrefersReducedMotion();
  const [stage, setStage] = useState<UnlockStage>(unlocked ? "lifted" : "locked");
  const wasUnlocked = useRef(unlocked);

  useEffect(() => {
    const justUnlocked = unlocked && !wasUnlocked.current;
    wasUnlocked.current = unlocked;

    if (!unlocked) {
      setStage("locked");
      return;
    }
    if (!justUnlocked) {
      setStage("lifted");
      return;
    }
    if (prefersReducedMotion) {
      setStage("lifted");
      onComplete?.();
      return;
    }

    const timers: ReturnType<typeof setTimeout>[] = [];
    const schedule = (delay: number, next: UnlockStage, fireComplete?: boolean) => {
      timers.push(
        setTimeout(() => {
          setStage(next);
          if (fireComplete) onComplete?.();
        }, delay),
      );
    };

    setStage("ring");
    let elapsed = DURATION.unlockRingMs;
    schedule(elapsed, "flare");
    elapsed += DURATION.unlockFlareMs;
    schedule(elapsed, "dissolve");
    elapsed += DURATION.unlockDissolveMs;
    schedule(elapsed, "lifted", true);

    return () => {
      timers.forEach(clearTimeout);
    };
    // `onComplete` is intentionally excluded — it's a fire-once callback,
    // not a value this sequence should restart for.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unlocked, prefersReducedMotion]);

  return { stage, isSequencing: stage !== "locked" && stage !== "lifted" };
}
