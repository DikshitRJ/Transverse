import type { ReactNode } from "react";
import { ArrowDownIcon, ArrowUpIcon, MinusIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { Sparkline } from "./sparkline";

export interface StatTileDelta {
  /** Pre-formatted, signed where meaningful — e.g. "+42", "-0.03". */
  text: string;
  direction: "up" | "down" | "flat";
  /** "up is good" by default; flip for metrics where a decrease is the win (e.g. carelessness). */
  upIsGood?: boolean;
  description?: string;
}

export interface StatTileProps {
  label: string;
  value: string;
  hint?: string;
  delta?: StatTileDelta;
  /** Oldest -> newest. Renders a small supplementary trend line under the value. */
  sparkline?: number[];
  icon?: ReactNode;
  className?: string;
}

const DELTA_ICON = { up: ArrowUpIcon, down: ArrowDownIcon, flat: MinusIcon };

function deltaColorClass(delta: StatTileDelta): string {
  if (delta.direction === "flat") return "text-tv-text-body";
  const isGood = delta.upIsGood ?? true;
  const positive = delta.direction === "up" ? isGood : !isGood;
  return positive ? "text-tv-cyan" : "text-tv-rose";
}

/**
 * Stat-tile contract per `dataviz`: label (sentence case, no trailing
 * colon) · value (proportional figures, not tabular-nums — this is a
 * standalone display number, not a table column) · optional signed delta
 * (color = direction × whether up is good, never color alone — an
 * icon + text always ships with it) · optional 12-point sparkline.
 */
export function StatTile({ label, value, hint, delta, sparkline, icon, className }: StatTileProps) {
  const DeltaIcon = delta ? DELTA_ICON[delta.direction] : null;
  return (
    <div
      className={cn(
        "glass-panel flex flex-col gap-2 rounded-tv-card border border-tv-border px-5 py-4",
        className,
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="font-mono text-xs tracking-wide text-tv-text-body uppercase">{label}</span>
        {icon && <span className="text-tv-cyan">{icon}</span>}
      </div>
      <div className="flex items-end justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <span className="font-display text-h1 leading-none font-bold text-tv-text-hi">{value}</span>
          {hint && <span className="font-body text-xs text-tv-text-body">{hint}</span>}
        </div>
        {sparkline && sparkline.length >= 2 && (
          <Sparkline values={sparkline} label={`${label} trend`} />
        )}
      </div>
      {delta && DeltaIcon && (
        <div className={cn("flex items-center gap-1 font-mono text-xs", deltaColorClass(delta))}>
          <DeltaIcon className="size-3" aria-hidden="true" />
          <span>{delta.text}</span>
          {delta.description && <span className="text-tv-text-body">{delta.description}</span>}
        </div>
      )}
    </div>
  );
}
