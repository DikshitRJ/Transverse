import Link from "next/link";
import { EyeIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { formatPercent, formatTopicLabel } from "@/components/charts/chart-theme";
import { DIFFICULTY_BADGE_VARIANT } from "../_lib/difficulty";
import type { ProblemPayload } from "@/lib/api/types";

export function ProblemCard({ problem, onPreview }: { problem: ProblemPayload; onPreview: () => void }) {
  return (
    <Card className="p-4 transition-colors hover:border-tv-border-cyan">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="mb-1.5 flex flex-wrap items-center gap-1.5">
            <Badge variant={DIFFICULTY_BADGE_VARIANT[problem.difficulty_label] ?? "outline"} className="capitalize">
              {problem.difficulty_label}
            </Badge>
            <Badge variant="outline" className="capitalize">
              {problem.source}
            </Badge>
            <Badge variant="secondary">{formatTopicLabel(problem.topic)}</Badge>
          </div>
          <h3 className="truncate font-heading text-base font-medium text-tv-text-hi">{problem.name}</h3>
          {problem.tags.length > 0 && (
            <p className="mt-1 truncate font-body text-xs text-tv-text-body">{problem.tags.join(" · ")}</p>
          )}
        </div>

        <div className="shrink-0 text-right">
          <div className="font-mono text-sm text-tv-cyan tabular-nums">{formatPercent(problem.solve_rate)}</div>
          <div className="font-mono text-[10px] text-tv-text-body uppercase">Solve rate</div>
        </div>
      </div>

      <div className="mt-3 flex justify-end gap-2 border-t border-tv-border pt-3">
        <Button variant="ghost" size="sm" onClick={onPreview}>
          <EyeIcon />
          Preview
        </Button>
        <Button render={<Link href={`/solve/${problem.id}`} />} variant="outline-cyan" size="sm">
          Solve
        </Button>
      </div>
    </Card>
  );
}
