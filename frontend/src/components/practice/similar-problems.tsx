"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import { getSimilarProblems } from "@/lib/api/endpoints";
import { cn } from "@/lib/utils";
import { difficultyBadgeVariant } from "./format";

export interface SimilarProblemsProps {
  problemId: string;
  className?: string;
}

/** "More like this" — GET /practice/similar, informational, links forward to FORGE's /solve/[problemId]. */
export function SimilarProblems({ problemId, className }: SimilarProblemsProps) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["practice", "similar", problemId],
    queryFn: () => getSimilarProblems(problemId, { limit: 4 }),
  });

  if (isLoading) {
    return (
      <div className={cn("space-y-1.5", className)} aria-busy="true">
        <div className="h-3 w-24 animate-pulse rounded bg-tv-surface-2" />
        <div className="h-6 w-full animate-pulse rounded bg-tv-surface-2" />
        <div className="h-6 w-full animate-pulse rounded bg-tv-surface-2" />
      </div>
    );
  }

  if (isError || !data || data.similar_problems.length === 0) {
    return null;
  }

  return (
    <div className={className}>
      <h4 className="mb-1.5 font-mono text-xs text-tv-text-body uppercase tracking-wide">
        More like this
      </h4>
      <ul className="space-y-1">
        {data.similar_problems.map((p) => (
          <li key={p.id}>
            <Link
              href={`/solve/${p.id}`}
              className="flex items-center justify-between gap-2 rounded-tv-btn border border-tv-border px-2 py-1.5 text-xs text-tv-text-nav transition-colors hover:border-tv-border-cyan hover:text-tv-text-hi"
            >
              <span className="truncate">{p.name}</span>
              <Badge variant={difficultyBadgeVariant(p.difficulty_label)} className="shrink-0">
                {p.difficulty_label}
              </Badge>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
