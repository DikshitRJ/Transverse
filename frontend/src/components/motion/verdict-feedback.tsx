"use client";

import { useEffect, type ReactNode } from "react";
import { useAnimate } from "motion/react";
import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "./use-prefers-reduced-motion";
import { DURATION, EASE, SHAKE_KEYFRAMES_X } from "./tokens";

export type Verdict = "pass" | "fail" | null;

export interface VerdictFeedbackProps {
  /** `null` renders `children` inert — no feedback playing. */
  verdict: Verdict;
  /**
   * Bump this (a submission id, a timestamp) to replay the same verdict
   * again — e.g. two consecutive "fail" submissions on the same row.
   * Defaults to `verdict` itself, which is enough when a row only ever
   * receives one verdict.
   */
  playToken?: string | number;
  children: ReactNode;
  className?: string;
}

/**
 * Wraps a test-case row / answer row and plays plan.md §1.4's verdict
 * gesture whenever `verdict` (keyed by `playToken`) changes to a non-null
 * value: **pass** → a cyan ring ripples outward once; **fail** → a single
 * 120ms rose shake, no bounce, no spring.
 *
 * Reduced motion: no ripple, no shake — instead a static colored outline
 * (cyan/rose) applies immediately and holds for as long as `verdict` does.
 * That's the "instant state change" plan.md §1.4 asks for, not a faster
 * version of the same animation.
 *
 * ```tsx
 * <VerdictFeedback verdict={result?.verdict ?? null} playToken={result?.submissionId}>
 *   <TestCaseRow ... />
 * </VerdictFeedback>
 * ```
 */
export function VerdictFeedback({ verdict, playToken, children, className }: VerdictFeedbackProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const [scope, animate] = useAnimate<HTMLDivElement>();
  const token = playToken ?? verdict;

  useEffect(() => {
    if (!verdict || prefersReducedMotion) return;

    if (verdict === "fail") {
      void animate(
        scope.current,
        { x: SHAKE_KEYFRAMES_X },
        { duration: DURATION.verdictFailMs / 1000, ease: EASE.sharp },
      );
    } else {
      void animate(
        "[data-verdict-ripple]",
        { opacity: [0.9, 0], scale: [0.92, 1.06] },
        { duration: DURATION.verdictPassMs / 1000, ease: EASE.standard },
      );
    }
    // `token` (not `verdict`) is the intentional replay trigger — see prop docs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, prefersReducedMotion]);

  return (
    <div ref={scope} className={cn("relative rounded-[inherit]", className)}>
      {children}

      <span
        aria-hidden
        data-verdict-ripple
        className="pointer-events-none absolute inset-0 rounded-[inherit] border-2 border-tv-cyan opacity-0"
      />

      {prefersReducedMotion && verdict && (
        <span
          aria-hidden
          className={cn(
            "pointer-events-none absolute inset-0 rounded-[inherit] border-2",
            verdict === "pass" ? "border-tv-cyan" : "border-tv-rose",
          )}
        />
      )}
    </div>
  );
}
