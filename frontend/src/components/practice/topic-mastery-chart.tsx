"use client";

import {
  Bar,
  BarChart,
  Cell,
  LabelList,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  type RenderableText,
  type TooltipContentProps,
} from "recharts";
import { cn } from "@/lib/utils";
import { formatMasteryScore, topicLabel } from "./format";

export interface TopicMasteryDatum {
  topic: string;
  /** 0-100, matching the backend's `mastery_score` field directly (see `formatMasteryScore`) — never a 0-1 fraction. */
  masteryScore: number;
}

export interface TopicMasteryChartProps {
  data: TopicMasteryDatum[];
  className?: string;
}

type Tier = "strong" | "developing" | "weak";

function tierOf(score: number): Tier {
  if (score >= 66) return "strong";
  if (score >= 34) return "developing";
  return "weak";
}

const TIER_COLOR: Record<Tier, string> = {
  strong: "var(--tv-cyan)",
  developing: "var(--tv-warning)",
  weak: "var(--tv-rose)",
};

const TIER_TEXT: Record<Tier, string> = {
  strong: "text-tv-cyan",
  developing: "text-tv-warning",
  weak: "text-tv-rose",
};

function CustomTooltip({ active, payload }: TooltipContentProps) {
  const [entry] = payload ?? [];
  if (!active || !entry) return null;
  const point = entry.payload as { topic: string; masteryScore: number };
  const tier = tierOf(point.masteryScore);
  return (
    <div className="rounded-tv-btn border border-tv-border-cyan bg-tv-surface px-2.5 py-1.5 font-mono text-xs shadow-lg">
      <div className={cn("font-semibold", TIER_TEXT[tier])}>{formatMasteryScore(point.masteryScore)}</div>
      <div className="text-tv-text-body">{topicLabel(point.topic)}</div>
    </div>
  );
}

/**
 * Per-topic mastery, magnitude-per-category — one bar per topic, colored by
 * a reserved 3-step status tier (strong/developing/weak) rather than a
 * generated categorical hue, per dataviz-skill guidance ("status colors are
 * reserved, never reused for series identity"). Because color here carries
 * meaning that varies per bar (not just series identity), every bar also
 * carries a direct percentage label — identity is never color-alone.
 */
export function TopicMasteryChart({ data, className }: TopicMasteryChartProps) {
  if (data.length === 0) {
    return (
      <p className={cn("font-mono text-xs text-tv-text-body", className)}>
        No per-topic data yet.
      </p>
    );
  }

  const sorted = [...data].sort((a, b) => b.masteryScore - a.masteryScore);
  const height = Math.max(140, sorted.length * 32 + 24);

  return (
    <div className={className}>
      <div className="mb-2 flex flex-wrap items-center gap-3 font-mono text-[10px] uppercase tracking-wide text-tv-text-body">
        <span className="flex items-center gap-1">
          <span className="size-2 rounded-full" style={{ backgroundColor: TIER_COLOR.strong }} />
          Strong
        </span>
        <span className="flex items-center gap-1">
          <span className="size-2 rounded-full" style={{ backgroundColor: TIER_COLOR.developing }} />
          Developing
        </span>
        <span className="flex items-center gap-1">
          <span className="size-2 rounded-full" style={{ backgroundColor: TIER_COLOR.weak }} />
          Needs work
        </span>
      </div>
      <ResponsiveContainer width="100%" height={height}>
        <BarChart
          data={sorted}
          layout="vertical"
          margin={{ top: 4, right: 36, bottom: 4, left: 8 }}
          barCategoryGap={8}
        >
          <XAxis type="number" domain={[0, 100]} hide />
          <YAxis
            type="category"
            dataKey="topic"
            tickFormatter={topicLabel}
            tick={{ fill: "var(--tv-text-body)", fontSize: 11, fontFamily: "var(--tv-font-mono)" }}
            axisLine={false}
            tickLine={false}
            width={140}
          />
          <Tooltip
            content={(props) => <CustomTooltip {...props} />}
            cursor={{ fill: "var(--tv-surface-2)", opacity: 0.4 }}
          />
          <Bar dataKey="masteryScore" radius={[0, 4, 4, 0]} maxBarSize={20} isAnimationActive={false}>
            {sorted.map((d) => (
              <Cell key={d.topic} fill={TIER_COLOR[tierOf(d.masteryScore)]} />
            ))}
            <LabelList
              dataKey="masteryScore"
              position="right"
              formatter={(v: RenderableText) => (typeof v === "number" ? formatMasteryScore(v) : "")}
              fill="var(--tv-text-hi)"
              fontFamily="var(--tv-font-mono)"
              fontSize={11}
            />
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
