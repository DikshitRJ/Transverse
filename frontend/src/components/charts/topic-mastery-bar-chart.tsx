"use client";

import { Bar, BarChart, CartesianGrid, Cell, LabelList, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { useChartMotionPreference } from "@/components/dashboard/use-chart-motion-preference";
import { CHART_AXIS_TICK, CHART_COLOR, CHART_TOOLTIP_PROPS, formatPercent, formatTopicLabel } from "./chart-theme";
import { ChartEmpty } from "./chart-empty";

export interface TopicMasteryDatum {
  topic: string;
  /**
   * 0–1 fraction (matches `formatPercent`'s contract). The backend's
   * `TopicProgress.mastery_score` (`GET /practice/topics`) is 0–100 on the
   * wire (`CalculateMasteryScore`, `practice_analytics.go`) — callers must
   * divide by 100 when mapping API data into this shape; see `/dashboard`
   * and `/profile` for the normalization.
   */
  masteryScore: number;
  attemptCount: number;
}

export interface TopicMasteryBarChartProps {
  data: TopicMasteryDatum[];
  className?: string;
  /** Row height in px — controls total chart height (`data.length * rowHeight + chrome`). */
  rowHeight?: number;
}

/**
 * Magnitude-comparison job -> horizontal bars, one hue (cyan), sequential.
 * Topics are categories with no shared numeric identity, so — per the
 * anti-patterns list — this is deliberately ONE color for every bar, not a
 * value-ramp keyed to bar length (that would double-encode length as hue).
 * Values are direct-labeled at the bar end, so the chart needs no separate
 * table twin (every value is already readable without hovering).
 */
export function TopicMasteryBarChart({ data, className, rowHeight = 34 }: TopicMasteryBarChartProps) {
  const [reduceMotion] = useChartMotionPreference();
  if (data.length === 0) {
    return (
      <ChartEmpty
        title="No topic attempts yet"
        description="Mastery per topic will appear once you've practiced at least one problem."
      />
    );
  }

  const height = Math.max(140, data.length * rowHeight + 24);
  const longestLabel = Math.max(...data.map((d) => formatTopicLabel(d.topic).length));
  const yAxisWidth = Math.min(150, Math.max(70, longestLabel * 6.5));

  return (
    <div style={{ height }} className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={data}
          layout="vertical"
          margin={{ top: 4, right: 36, bottom: 4, left: 0 }}
          barCategoryGap={8}
        >
          <CartesianGrid stroke={CHART_COLOR.grid} horizontal={false} />
          <XAxis
            type="number"
            domain={[0, 1]}
            tickFormatter={formatPercent}
            tick={CHART_AXIS_TICK}
            tickLine={false}
            axisLine={{ stroke: CHART_COLOR.grid }}
          />
          <YAxis
            type="category"
            dataKey="topic"
            tickFormatter={formatTopicLabel}
            tick={CHART_AXIS_TICK}
            tickLine={false}
            axisLine={false}
            width={yAxisWidth}
          />
          <Tooltip
            {...CHART_TOOLTIP_PROPS}
            labelFormatter={(label) => formatTopicLabel(String(label))}
            formatter={(value, _name, item) => {
              const attemptCount = (item.payload as TopicMasteryDatum).attemptCount;
              return [`${formatPercent(Number(value))} · ${attemptCount} attempts`, "Mastery"];
            }}
            cursor={{ fill: "var(--tv-surface-2)" }}
          />
          <Bar dataKey="masteryScore" radius={[0, 4, 4, 0]} maxBarSize={20} isAnimationActive={!reduceMotion}>
            {data.map((entry) => (
              <Cell key={entry.topic} fill={CHART_COLOR.cyan} />
            ))}
            <LabelList
              dataKey="masteryScore"
              position="right"
              formatter={(value) => formatPercent(Number(value))}
              style={{ fill: "var(--tv-text-hi)", fontFamily: "var(--tv-font-mono)", fontSize: 11 }}
            />
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
