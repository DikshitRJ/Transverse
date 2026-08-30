"use client";

import { useId, useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Button } from "@/components/ui/button";
import { useChartMotionPreference } from "@/components/dashboard/use-chart-motion-preference";
import { CHART_AXIS_TICK, CHART_COLOR, CHART_GRID_PROPS, CHART_TOOLTIP_PROPS, formatShortDate } from "./chart-theme";
import { ChartEmpty } from "./chart-empty";

export interface RatingTrendPoint {
  /** ISO timestamp — usually a session's `created_at`. */
  date: string;
  /** Ability estimate (θ) at the end of that session. */
  theta: number;
  questionCount?: number;
}

export interface RatingTrendChartProps {
  data: RatingTrendPoint[];
  height?: number;
  className?: string;
}

/**
 * Trend-over-time job -> single-series area/line, one hue (cyan), per
 * `dataviz`'s form table. There is no endpoint that snapshots Glicko rating
 * over time (only the current value on `GET /user/profile`), so this plots
 * θ — the adaptive ability estimate each practice session ends at, taken
 * from `GET /user/history` — which is the only real longitudinal skill
 * signal the backend exposes. Labeled honestly as "ability estimate," not
 * "rating history."
 */
export function RatingTrendChart({ data, height = 240, className }: RatingTrendChartProps) {
  const [showTable, setShowTable] = useState(false);
  const gradientId = useId();
  const [reduceMotion] = useChartMotionPreference();

  if (data.length === 0) {
    return (
      <ChartEmpty
        height={height}
        title="No sessions yet"
        description="Your ability estimate (θ) will chart here once you complete a practice session."
      />
    );
  }

  if (data.length === 1) {
    return (
      <ChartEmpty
        height={height}
        title={`θ = ${data[0]!.theta.toFixed(2)} after your first session`}
        description="Complete another session to see a trend line."
      />
    );
  }

  const last = data[data.length - 1]!;

  return (
    <div className={className}>
      <div className="mb-2 flex items-center justify-between">
        <span className="font-mono text-xs text-tv-text-body">
          Latest: <span className="text-tv-cyan">θ {last.theta.toFixed(2)}</span>
        </span>
        <Button variant="ghost" size="xs" onClick={() => setShowTable((v) => !v)}>
          {showTable ? "View chart" : "View table"}
        </Button>
      </div>

      {showTable ? (
        <div className="max-h-60 overflow-auto rounded-tv-btn border border-tv-border">
          <table className="w-full text-left text-xs">
            <thead className="sticky top-0 bg-tv-surface-2 font-mono text-tv-text-nav uppercase">
              <tr>
                <th className="px-3 py-2 font-medium">Session date</th>
                <th className="px-3 py-2 font-medium tabular-nums">θ</th>
                <th className="px-3 py-2 font-medium tabular-nums">Questions</th>
              </tr>
            </thead>
            <tbody className="font-body text-tv-text-hi">
              {data.map((d, i) => (
                <tr key={`${d.date}-${i}`} className="border-t border-tv-border">
                  <td className="px-3 py-2">{formatShortDate(d.date)}</td>
                  <td className="px-3 py-2 tabular-nums">{d.theta.toFixed(2)}</td>
                  <td className="px-3 py-2 tabular-nums">{d.questionCount ?? "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div style={{ height }} className="w-full">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 8, right: 12, bottom: 0, left: -16 }}>
              <defs>
                <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={CHART_COLOR.cyan} stopOpacity={0.18} />
                  <stop offset="100%" stopColor={CHART_COLOR.cyan} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid {...CHART_GRID_PROPS} />
              <XAxis
                dataKey="date"
                tickFormatter={formatShortDate}
                tick={CHART_AXIS_TICK}
                tickLine={false}
                axisLine={{ stroke: CHART_COLOR.grid }}
                minTickGap={32}
              />
              <YAxis
                tick={CHART_AXIS_TICK}
                tickLine={false}
                axisLine={false}
                width={36}
                domain={["dataMin - 0.05", "dataMax + 0.05"]}
                tickFormatter={(v: number) => v.toFixed(2)}
              />
              <Tooltip
                {...CHART_TOOLTIP_PROPS}
                labelFormatter={(label) => formatShortDate(String(label))}
                formatter={(value) => [Number(value).toFixed(3), "θ"]}
              />
              <Area
                type="monotone"
                dataKey="theta"
                stroke={CHART_COLOR.cyan}
                strokeWidth={2}
                fill={`url(#${gradientId})`}
                dot={{ r: 3, fill: CHART_COLOR.cyan, stroke: CHART_COLOR.surface, strokeWidth: 1.5 }}
                activeDot={{ r: 5, fill: CHART_COLOR.cyan, stroke: CHART_COLOR.surface, strokeWidth: 2 }}
                isAnimationActive={!reduceMotion}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
