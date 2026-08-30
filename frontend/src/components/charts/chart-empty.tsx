import type { ReactNode } from "react";
import { BarChart3Icon } from "lucide-react";
import { cn } from "@/lib/utils";

export interface ChartEmptyProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  height?: number;
  className?: string;
}

/** Compact "nothing to plot yet" placeholder sized to sit inside a chart card, not a full page panel. */
export function ChartEmpty({ title, description, icon, height = 220, className }: ChartEmptyProps) {
  return (
    <div
      style={{ minHeight: height }}
      className={cn(
        "flex flex-col items-center justify-center gap-2 rounded-tv-btn border border-dashed border-tv-border bg-tv-surface-deep/40 px-6 text-center",
        className,
      )}
    >
      <span className="text-tv-locked">{icon ?? <BarChart3Icon className="size-5" aria-hidden="true" />}</span>
      <p className="font-body text-sm font-medium text-tv-text-body">{title}</p>
      {description && <p className="max-w-xs font-body text-xs text-tv-text-body/80">{description}</p>}
    </div>
  );
}
