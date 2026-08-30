"use client";

import { Bar, BarChart, CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { useChartMotionPreference } from "@/components/dashboard/use-chart-motion-preference";
import { CHART_AXIS_TICK, CHART_COLOR, CHART_GRID_PROPS, CHART_TOOLTIP_PROPS, formatPercent, formatShortDate } from "./chart-theme";
import { ChartEmpty } from "./chart-empty";

export interface AccuracyVolumePoint {
  date: string;
  /** 0–1 */
  accuracy: number;
  /** questions attempted in that session */
  volume: number;
}

export interface AccuracyVolumeChartProps {
  data: AccuracyVolumePoint[];
  height?: number;
  className?: string;
}

/**
 * Accuracy (a ratio, 0–1) and volume (a count) live on incompatible
 * scales — per `dataviz`'s #1 anti-pattern this is never one dual-axis
 * plot. Two single-axis small multiples instead, sharing the same x
 * domain so they read together without inventing a false correlation.
 */
export function AccuracyVolumeChart({ data, height = 180, className }: AccuracyVolumeChartProps) {
  const [reduceMotion] = useChartMotionPreference();
  if (data.length === 0) {
    return (
      <ChartEmpty
        height={height}
        title="No sessions yet"
        description="Accuracy and volume trends appear once you've completed practice sessions."
      />
    );
  }

  return (
    <div className={`grid gap-4 sm:grid-cols-2 ${className ?? ""}`}>
      <div>
        <p className="mb-1 font-mono text-xs tracking-wide text-tv-text-body uppercase">Accuracy per session</p>
        <div style={{ height }}>
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -20 }}>
              <CartesianGrid {...CHART_GRID_PROPS} />
              <XAxis
                dataKey="date"
                tickFormatter={formatShortDate}
                tick={CHART_AXIS_TICK}
                tickLine={false}
                axisLine={{ stroke: CHART_COLOR.grid }}
                minTickGap={24}
              />
              <YAxis
                domain={[0, 1]}
                tickFormatter={formatPercent}
                tick={CHART_AXIS_TICK}
                tickLine={false}
                axisLine={false}
                width={34}
              />
              <Tooltip
                {...CHART_TOOLTIP_PROPS}
                labelFormatter={(label) => formatShortDate(String(label))}
                formatter={(value) => [formatPercent(Number(value)), "Accuracy"]}
              />
              <Line
                type="monotone"
                dataKey="accuracy"
                stroke={CHART_COLOR.cyan}
                strokeWidth={2}
                dot={{ r: 3, fill: CHART_COLOR.cyan, stroke: CHART_COLOR.surface, strokeWidth: 1.5 }}
                activeDot={{ r: 5 }}
                isAnimationActive={!reduceMotion}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div>
        <p className="mb-1 font-mono text-xs tracking-wide text-tv-text-body uppercase">Questions per session</p>
        <div style={{ height }}>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -20 }}>
              <CartesianGrid {...CHART_GRID_PROPS} />
              <XAxis
                dataKey="date"
                tickFormatter={formatShortDate}
                tick={CHART_AXIS_TICK}
                tickLine={false}
                axisLine={{ stroke: CHART_COLOR.grid }}
                minTickGap={24}
              />
              <YAxis allowDecimals={false} tick={CHART_AXIS_TICK} tickLine={false} axisLine={false} width={28} />
              <Tooltip
                {...CHART_TOOLTIP_PROPS}
                labelFormatter={(label) => formatShortDate(String(label))}
                formatter={(value) => [String(value), "Questions"]}
                cursor={{ fill: "var(--tv-surface-2)" }}
              />
              <Bar dataKey="volume" fill={CHART_COLOR.cyan} radius={[4, 4, 0, 0]} maxBarSize={22} isAnimationActive={!reduceMotion} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}
