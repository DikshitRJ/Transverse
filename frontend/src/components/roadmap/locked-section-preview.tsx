import { Lock } from "lucide-react";
import type { UpcomingSectionPreview } from "@/lib/api/types";

export interface LockedSectionPreviewProps {
  section: UpcomingSectionPreview;
}

/**
 * A genuinely-locked upcoming section — title + sequence only, per the
 * backend contract (`RoadmapCurrentResponse.upcoming_sections` never
 * carries subsections). Visibly out of reach: inert, no link, dimmed
 * `--tv-locked` text, no hover state.
 */
export function LockedSectionPreview({ section }: LockedSectionPreviewProps) {
  return (
    <div
      aria-disabled
      className="flex cursor-not-allowed items-center gap-4 rounded-tv-card border border-tv-border bg-tv-surface/30 px-5 py-4 select-none"
    >
      <div className="flex size-10 shrink-0 items-center justify-center rounded-tv-btn border border-tv-border bg-tv-surface-deep">
        <Lock className="size-4 text-tv-locked" aria-hidden />
      </div>
      <div className="flex flex-col gap-0.5">
        <span className="font-mono text-xs text-tv-locked">Section {section.sequence}</span>
        <span className="font-display text-sm font-bold tracking-tight text-tv-locked uppercase">
          {section.title}
        </span>
      </div>
    </div>
  );
}
