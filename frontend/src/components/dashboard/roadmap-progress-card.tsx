import Link from "next/link";
import { MapIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { EmptyPanel } from "./async-panels";
import type { RoadmapCurrentResponse, RoadmapSubsection } from "@/lib/api/types";

const NODE_BADGE: Record<RoadmapSubsection["status"], { label: string; variant: "success" | "secondary" | "locked" }> = {
  mastered: { label: "Mastered", variant: "success" },
  tested_out: { label: "Tested out", variant: "success" },
  in_progress: { label: "In progress", variant: "secondary" },
  unlocked: { label: "Unlocked", variant: "secondary" },
  locked: { label: "Locked", variant: "locked" },
};

export function pickActiveNode(subsections: RoadmapSubsection[]): RoadmapSubsection | undefined {
  return (
    subsections.find((s) => s.status === "in_progress") ??
    subsections.find((s) => s.status === "unlocked") ??
    subsections[0]
  );
}

export function RoadmapProgressCard({
  roadmap,
  className,
}: {
  roadmap: RoadmapCurrentResponse | null | undefined;
  className?: string;
}) {
  const section = roadmap?.current_section;

  if (!roadmap || !section) {
    return (
      <EmptyPanel
        icon={<MapIcon className="size-5" aria-hidden="true" />}
        title="No roadmap yet"
        description="Generate a personalized roadmap from your diagnostic results to see your path here."
        action={
          <Button render={<Link href="/roadmap" />} size="sm">
            Set up roadmap
          </Button>
        }
        className={className}
      />
    );
  }

  const activeNode = pickActiveNode(section.subsections);

  return (
    <div className={className}>
      <div className="mb-1 flex items-center justify-between gap-2">
        <div>
          <p className="font-mono text-xs tracking-wide text-tv-text-body uppercase">
            Section {section.sequence} of {roadmap.total_sections}
          </p>
          <h3 className="font-heading text-h4 text-tv-text-hi">{section.title}</h3>
        </div>
        <span className="font-mono text-sm text-tv-cyan tabular-nums">{section.progress_percentage}%</span>
      </div>

      <Progress value={section.progress_percentage} className="mb-4" />

      {activeNode && (
        <div className="glass-panel mb-4 flex items-center justify-between gap-3 rounded-tv-btn border border-tv-border-cyan px-4 py-3">
          <div className="min-w-0">
            <p className="font-mono text-[10px] text-tv-text-body uppercase">Up next</p>
            <p className="truncate font-body text-sm font-medium text-tv-text-hi">{activeNode.title}</p>
          </div>
          <Badge variant={NODE_BADGE[activeNode.status].variant} className="shrink-0">
            {NODE_BADGE[activeNode.status].label}
          </Badge>
        </div>
      )}

      <ol className="flex flex-col gap-1.5">
        {section.subsections.map((node) => (
          <li key={node.node_id} className="flex items-center justify-between gap-3 py-0.5">
            <span
              className={
                node.status === "locked"
                  ? "truncate font-body text-sm text-tv-locked"
                  : "truncate font-body text-sm text-tv-text-body"
              }
            >
              {node.sequence}. {node.title}
            </span>
            <Badge variant={NODE_BADGE[node.status].variant} className="shrink-0">
              {NODE_BADGE[node.status].label}
            </Badge>
          </li>
        ))}
      </ol>

      <Button render={<Link href="/roadmap" />} variant="outline-cyan" size="sm" className="mt-4 w-full">
        View full roadmap
      </Button>
    </div>
  );
}
