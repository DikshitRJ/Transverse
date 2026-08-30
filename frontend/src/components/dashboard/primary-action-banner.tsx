import Link from "next/link";
import { ArrowRightIcon, MapIcon, ZapIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { pickActiveNode } from "./roadmap-progress-card";
import type { RoadmapCurrentResponse } from "@/lib/api/types";

/**
 * The dashboard's one clear "what do I do next" action — a returning user
 * should not have to hunt for it. Primary = continue the roadmap if one
 * exists and has an actionable node; otherwise the primary shifts to
 * starting a practice session (or generating a roadmap for a brand-new
 * user), with the other path always offered as the secondary.
 */
export function PrimaryActionBanner({
  roadmap,
  className,
}: {
  roadmap: RoadmapCurrentResponse | null | undefined;
  className?: string;
}) {
  const section = roadmap?.current_section;
  const activeNode = section ? pickActiveNode(section.subsections) : undefined;
  const hasActionableRoadmap = Boolean(activeNode && activeNode.status !== "locked");

  return (
    <div
      className={`glass-panel glow-card-cyan flex flex-col items-start gap-4 rounded-tv-card border border-tv-border-cyan px-6 py-5 sm:flex-row sm:items-center sm:justify-between ${className ?? ""}`}
    >
      <div>
        <p className="font-mono text-xs tracking-wide text-tv-cyan uppercase">Next up</p>
        {hasActionableRoadmap && activeNode ? (
          <h3 className="font-heading text-h4 text-tv-text-hi">Continue: {activeNode.title}</h3>
        ) : roadmap && section ? (
          <h3 className="font-heading text-h4 text-tv-text-hi">Keep your streak going</h3>
        ) : (
          <h3 className="font-heading text-h4 text-tv-text-hi">Get your personalized roadmap</h3>
        )}
      </div>

      <div className="flex shrink-0 flex-wrap gap-2">
        {hasActionableRoadmap && activeNode ? (
          <>
            <Button render={<Link href={`/roadmap/node/${activeNode.node_id}`} />}>
              Continue roadmap
              <ArrowRightIcon />
            </Button>
            <Button render={<Link href="/practice" />} variant="outline-cyan">
              <ZapIcon />
              Start practice
            </Button>
          </>
        ) : roadmap && section ? (
          <>
            <Button render={<Link href="/practice" />}>
              <ZapIcon />
              Start practice
            </Button>
            <Button render={<Link href="/roadmap" />} variant="outline-cyan">
              <MapIcon />
              View roadmap
            </Button>
          </>
        ) : (
          <>
            <Button render={<Link href="/roadmap" />}>
              <MapIcon />
              Set up roadmap
            </Button>
            <Button render={<Link href="/practice" />} variant="outline-cyan">
              <ZapIcon />
              Start practice
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
