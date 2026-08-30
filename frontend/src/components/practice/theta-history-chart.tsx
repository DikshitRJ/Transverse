"use client";

import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  type TooltipContentProps,
} from "recharts";
import type { SessionResponse } from "@/lib/api/types";
import { formatTheta } from "./format";

export interface ThetaHistoryChartProps {
  thetaStart: number;
  responses: SessionResponse[];
  className?: string;
}

interface Point {
  index: number;
  theta: number;
  correct: boolean;
  skipped: boolean;
}

function CustomTooltip({ active, payload }: TooltipContentProps) {
  const [entry] = payload ?? [];
  if (!active || !entry) return null;
  const point = entry.payload as Point;
  const statusLabel = point.skipped ? "Skipped" : point.correct ? "Correct" : "Incorrect";
  return (
    <div className="rounded-tv-btn border border-tv-border-cyan bg-tv-surface px-2.5 py-1.5 font-mono text-xs shadow-lg">
      <div className="font-semibold text-tv-text-hi">θ {formatTheta(point.theta)}</div>
      <div className="text-tv-text-body">
        Question {point.index} · {statusLabel}
      </div>
    </div>
  );
}

/**
 * Single-series (theta), one hue by dataviz-skill rule ("sequential = one
 * hue"). No legend box per the skill's mark spec — a lone series needs none,
 * the chart's own heading names it. 2px line, hover crosshair + tooltip,
 * value led by the tooltip rather than labeling every point.
 */
export function ThetaHistoryChart({ thetaStart, responses, className }: ThetaHistoryChartProps) {
  if (responses.length === 0) {
    return (
      <div className={className}>
        <p className="font-mono text-xs text-tv-text-body">
          No attempts yet — theta history appears once you submit or skip a problem.
        </p>
      </div>
    );
  }

  const data: Point[] = [
    { index: 0, theta: thetaStart, correct: false, skipped: false },
    ...responses.map((r) => ({
      index: r.question_count,
      theta: r.theta_after,
      correct: r.is_correct,
      skipped: r.skipped,
    })),
  ];

  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height={220}>
        <LineChart data={data} margin={{ top: 8, right: 12, bottom: 0, left: -20 }}>
          <CartesianGrid vertical={false} stroke="var(--tv-border)" strokeOpacity={0.5} />
          <XAxis
            dataKey="index"
            tick={{ fill: "var(--tv-text-body)", fontSize: 11, fontFamily: "var(--tv-font-mono)" }}
            axisLine={{ stroke: "var(--tv-border)" }}
            tickLine={false}
            label={{
              value: "Question",
              position: "insideBottom",
              offset: -4,
              fill: "var(--tv-text-body)",
              fontSize: 11,
            }}
          />
          <YAxis
            domain={[0, 1]}
            tick={{ fill: "var(--tv-text-body)", fontSize: 11, fontFamily: "var(--tv-font-mono)" }}
            axisLine={false}
            tickLine={false}
            width={36}
          />
          <Tooltip content={(props) => <CustomTooltip {...props} />} cursor={{ stroke: "var(--tv-border-cyan)" }} />
          <Line
            type="monotone"
            dataKey="theta"
            stroke="var(--tv-cyan)"
            strokeWidth={2}
            dot={{ r: 4, fill: "var(--tv-cyan)", stroke: "var(--tv-bg-page)", strokeWidth: 2 }}
            activeDot={{ r: 5, fill: "var(--tv-cyan)", stroke: "var(--tv-bg-page)", strokeWidth: 2 }}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
