/**
 * Shared Recharts theming for every chart PRISM ships. Built per the
 * `dataviz` skill: color comes last, after form; the frozen palette here is
 * effectively "sequential, one hue" (cyan) for magnitude/trend, with rose
 * reserved for regression/error and amber for warning-only accents — never
 * a multi-hue categorical ramp, so no palette-validator run is needed (see
 * PRISM's report for why).
 *
 * Recharts renders inline SVG/style attributes, which is the one place
 * `theme.css`'s own docs permit a `var(--tv-*)` string instead of a
 * Tailwind utility class — never a raw hex here either.
 */

export const CHART_COLOR = {
  cyan: "var(--tv-cyan)",
  cyanPure: "var(--tv-cyan-pure)",
  rose: "var(--tv-rose)",
  warning: "var(--tv-warning)",
  grid: "var(--tv-border)",
  axis: "var(--tv-text-body)",
  surface: "var(--tv-surface)",
  surfaceDeep: "var(--tv-surface-deep)",
  textHi: "var(--tv-text-hi)",
  textNav: "var(--tv-text-nav)",
} as const;

/** Hairline, recessive gridlines/axes — never dashed, one step off-surface. */
export const CHART_GRID_PROPS = {
  stroke: CHART_COLOR.grid,
  strokeDasharray: "0",
  vertical: false,
} as const;

export const CHART_AXIS_TICK = {
  fill: CHART_COLOR.axis,
  fontFamily: "var(--tv-font-mono)",
  fontSize: 11,
} as const;

/** Themed Tooltip content/cursor styling — pass as props on Recharts' <Tooltip>. */
export const CHART_TOOLTIP_PROPS = {
  contentStyle: {
    backgroundColor: CHART_COLOR.surface,
    border: "1px solid var(--tv-border-cyan)",
    borderRadius: 8,
    boxShadow: "0 0 15px 0 rgba(0,242,255,0.12)",
    fontFamily: "var(--tv-font-mono)",
    fontSize: 12,
    padding: "8px 12px",
  },
  labelStyle: {
    color: CHART_COLOR.textNav,
    marginBottom: 4,
    fontSize: 11,
    textTransform: "uppercase" as const,
    letterSpacing: "0.04em",
  },
  itemStyle: {
    color: CHART_COLOR.textHi,
    padding: 0,
  },
  cursor: { stroke: "var(--tv-border-cyan)", strokeWidth: 1 },
} as const;

/** Formats a 0–1 mastery/accuracy score as a whole-number percentage string. */
export function formatPercent(value: number): string {
  return `${Math.round(value * 100)}%`;
}

/** Compact axis-tick date formatter — "Jun 3" style, stable across locales enough for a demo. */
export function formatShortDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/** `arrays-hashing` -> `Arrays Hashing` — topic ids are kebab-case curriculum slugs everywhere in the API. */
export function formatTopicLabel(topicId: string): string {
  return topicId
    .split("-")
    .filter(Boolean)
    .map((part) => part[0]!.toUpperCase() + part.slice(1))
    .join(" ");
}
