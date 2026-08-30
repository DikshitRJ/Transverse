"use client";

import { useReducedMotion } from "motion/react";
import { cn } from "@/lib/utils";
import { normalizeMastery } from "./roadmap-status";
import type { NodeStatus } from "@/lib/api/types";

export interface MasteryRingProps {
  score: number;
  status: NodeStatus;
  size?: number;
  className?: string;
}

/** Circular 0–100 mastery indicator. Cyan when reachable, dimmed `--tv-locked` when not. */
export function MasteryRing({ score, status, size = 56, className }: MasteryRingProps) {
  const pct = normalizeMastery(score);
  const reduceMotion = useReducedMotion();
  const stroke = 4;
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const offset = c - (pct / 100) * c;
  const isLocked = status === "locked";

  return (
    <div
      className={cn("relative inline-flex shrink-0 items-center justify-center", className)}
      style={{ width: size, height: size }}
      role="img"
      aria-label={isLocked ? "Locked" : `${pct}% mastery`}
    >
      <svg width={size} height={size} className="-rotate-90" aria-hidden>
        <circle cx={size / 2} cy={size / 2} r={r} stroke="var(--tv-border)" strokeWidth={stroke} fill="none" />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke={isLocked ? "var(--tv-locked)" : "var(--tv-cyan)"}
          strokeWidth={stroke}
          fill="none"
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={offset}
          style={{
            transition: reduceMotion ? "none" : "stroke-dashoffset 380ms ease-out, stroke 200ms ease-out",
          }}
        />
      </svg>
      <span
        className={cn(
          "absolute font-mono text-xs font-semibold",
          isLocked ? "text-tv-locked" : "text-tv-text-hi",
        )}
      >
        {isLocked ? "—" : `${pct}%`}
      </span>
    </div>
  );
}
