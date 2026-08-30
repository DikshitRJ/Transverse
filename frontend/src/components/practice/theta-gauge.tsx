"use client";

import { Progress } from "@/components/ui/progress";
import { cn } from "@/lib/utils";
import { formatTheta } from "./format";

export interface ThetaGaugeProps {
  theta: number;
  label?: string;
  className?: string;
  size?: "default" | "sm";
}

/**
 * Compact confidence/ability (theta) indicator. The mock engine clamps
 * theta to [0, 1] (`sessions.ts`), which is what this gauge's fill assumes;
 * the numeric readout is shown unclamped regardless so a live backend's
 * wider IRT range still displays an honest number even if the bar itself
 * saturates at the ends.
 */
export function ThetaGauge({ theta, label = "θ Confidence", size = "default" }: ThetaGaugeProps) {
  const pct = Math.max(0, Math.min(1, theta)) * 100;
  return (
    <div className="flex w-full items-center gap-2">
      <span
        className={cn(
          "shrink-0 font-mono uppercase tracking-wide text-tv-text-body",
          size === "sm" ? "text-[10px]" : "text-xs",
        )}
      >
        {label}
      </span>
      <Progress
        value={pct}
        className={cn("flex-1", size === "sm" && "[&_[data-slot=progress-track]]:h-1")}
      />
      <span
        className={cn(
          "shrink-0 font-mono tabular-nums text-tv-cyan glow-text-cyan",
          size === "sm" ? "text-[10px]" : "text-xs",
        )}
      >
        {formatTheta(theta)}
      </span>
    </div>
  );
}

export interface ThetaDeltaProps {
  before: number;
  after: number;
  className?: string;
}

/** theta_before -> theta_after readout with a directional delta chip. */
export function ThetaDelta({ before, after, className }: ThetaDeltaProps) {
  const delta = after - before;
  const isUp = delta > 0.0005;
  const isDown = delta < -0.0005;
  return (
    <div className={cn("flex items-center gap-2 font-mono text-xs", className)}>
      <span className="text-tv-text-body">{formatTheta(before)}</span>
      <span aria-hidden className="text-tv-text-body">
        →
      </span>
      <span className={cn(isUp && "text-tv-cyan glow-text-cyan", isDown && "text-tv-rose")}>
        {formatTheta(after)}
      </span>
      <span
        className={cn(
          "tabular-nums",
          isUp && "text-tv-cyan",
          isDown && "text-tv-rose",
          !isUp && !isDown && "text-tv-text-body",
        )}
      >
        {isUp ? "▲" : isDown ? "▼" : "–"} {Math.abs(delta).toFixed(2)}
      </span>
    </div>
  );
}
