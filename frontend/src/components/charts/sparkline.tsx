"use client";

import { useId } from "react";
import { CHART_COLOR } from "./chart-theme";

export interface SparklineProps {
  /** Ordered oldest -> newest. Rendered as-is, no resampling. */
  values: number[];
  width?: number;
  height?: number;
  className?: string;
  /** Accessible label — the sparkline itself is `aria-hidden`; describe the trend in surrounding text too. */
  label?: string;
}

/**
 * A small inline trend line for a stat tile — hand-rolled SVG rather than a
 * full Recharts `ResponsiveContainer` (overkill at this footprint, and
 * ResponsiveContainer is unreliable at very small fixed heights). Single
 * hue (cyan), 2px line, ~10% area wash, small end-dot — same mark language
 * as the full-size charts, just scaled down for a supplementary motif.
 */
export function Sparkline({ values, width = 96, height = 28, className, label }: SparklineProps) {
  const gradientId = useId();
  if (values.length < 2) return null;

  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const stepX = width / (values.length - 1);
  const pad = 3;
  const innerH = height - pad * 2;

  const points = values.map((v, i) => {
    const x = i * stepX;
    const y = pad + innerH - ((v - min) / range) * innerH;
    return { x, y };
  });

  const linePath = points.map((p, i) => `${i === 0 ? "M" : "L"}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(" ");
  const areaPath = `${linePath} L${points[points.length - 1]!.x.toFixed(1)},${height} L${points[0]!.x.toFixed(1)},${height} Z`;
  const last = points[points.length - 1]!;

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      width={width}
      height={height}
      className={className}
      role="img"
      aria-hidden={label ? undefined : true}
      aria-label={label}
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={CHART_COLOR.cyan} stopOpacity={0.22} />
          <stop offset="100%" stopColor={CHART_COLOR.cyan} stopOpacity={0} />
        </linearGradient>
      </defs>
      <path d={areaPath} fill={`url(#${gradientId})`} stroke="none" />
      <path d={linePath} fill="none" stroke={CHART_COLOR.cyan} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />
      <circle cx={last.x} cy={last.y} r={2.5} fill={CHART_COLOR.cyan} stroke={CHART_COLOR.surface} strokeWidth={1.5} />
    </svg>
  );
}
