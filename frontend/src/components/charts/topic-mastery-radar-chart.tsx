"use client";

import { useState } from "react";
import {
  PolarAngleAxis,
  PolarGrid,
  PolarRadiusAxis,
  Radar,
  RadarChart,
  ResponsiveContainer,
  Tooltip,
} from "recharts";
import { Button } from "@/components/ui/button";
import { useChartMotionPreference } from "@/components/dashboard/use-chart-motion-preference";
import { CHART_COLOR, CHART_TOOLTIP_PROPS, formatPercent, formatTopicLabel } from "./chart-theme";
import { ChartEmpty } from "./chart-empty";
import type { TopicMasteryDatum } from "./topic-mastery-bar-chart";

export interface TopicMasteryRadarChartProps {
  /** Already capped to a legible axis count by the caller (soft cap ~8). */
  data: TopicMasteryDatum[];
  height?: number;
  className?: string;
}

/**
 * A radar/spider chart is normally the wrong default (area is hard to
 * compare, per `dataviz`'s choosing-a-form guidance) — but a single
 * entity's shape across many same-scale (0–1 mastery) dimensions is one of
 * its few legitimate uses (the "skill profile" pattern), and it's what
 * plan.md's stack table names for this exact surface. Kept single-series
 * (cyan only, ~15% fill wash) and given a table twin so nothing is
 * color/shape-only.
 */
export function TopicMasteryRadarChart({ data, height = 320, className }: TopicMasteryRadarChartProps) {
  const [showTable, setShowTable] = useState(false);
  const [reduceMotion] = useChartMotionPreference();

  if (data.length === 0) {
    return (
      <ChartEmpty
        height={height}
        title="No mastery data yet"
        description="Your skill profile fills in as you complete practice problems across topics."
      />
    );
  }

  return (
    <div className={className}>
      <div className="mb-2 flex justify-end">
        <Button variant="ghost" size="xs" onClick={() => setShowTable((v) => !v)}>
          {showTable ? "View chart" : "View table"}
        </Button>
      </div>

      {showTable ? (
        <div className="max-h-80 overflow-auto rounded-tv-btn border border-tv-border">
          <table className="w-full text-left text-xs">
            <thead className="sticky top-0 bg-tv-surface-2 font-mono text-tv-text-nav uppercase">
              <tr>
                <th className="px-3 py-2 font-medium">Topic</th>
                <th className="px-3 py-2 font-medium tabular-nums">Mastery</th>
                <th className="px-3 py-2 font-medium tabular-nums">Attempts</th>
              </tr>
            </thead>
            <tbody className="font-body text-tv-text-hi">
              {data.map((d) => (
                <tr key={d.topic} className="border-t border-tv-border">
                  <td className="px-3 py-2">{formatTopicLabel(d.topic)}</td>
                  <td className="px-3 py-2 tabular-nums">{formatPercent(d.masteryScore)}</td>
                  <td className="px-3 py-2 tabular-nums">{d.attemptCount}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div style={{ height }}>
          <ResponsiveContainer width="100%" height="100%">
            <RadarChart data={data} outerRadius="68%">
              <PolarGrid stroke={CHART_COLOR.grid} />
              <PolarAngleAxis
                dataKey="topic"
                tickFormatter={formatTopicLabel}
                tick={{ fill: CHART_COLOR.axis, fontFamily: "var(--tv-font-mono)", fontSize: 10 }}
              />
              <PolarRadiusAxis
                domain={[0, 1]}
                tickCount={4}
                tickFormatter={formatPercent}
                tick={{ fill: CHART_COLOR.axis, fontFamily: "var(--tv-font-mono)", fontSize: 9 }}
                axisLine={false}
              />
              <Tooltip
                {...CHART_TOOLTIP_PROPS}
                labelFormatter={(label) => formatTopicLabel(String(label))}
                formatter={(value) => [formatPercent(Number(value)), "Mastery"]}
              />
              <Radar
                dataKey="masteryScore"
                stroke={CHART_COLOR.cyan}
                fill={CHART_COLOR.cyan}
                fillOpacity={0.15}
                strokeWidth={2}
                dot={{ r: 3, fill: CHART_COLOR.cyan, stroke: CHART_COLOR.surface, strokeWidth: 1.5 }}
                isAnimationActive={!reduceMotion}
              />
            </RadarChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
