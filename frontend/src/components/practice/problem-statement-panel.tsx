"use client";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { ProblemPayload } from "@/lib/api/types";
import { cn } from "@/lib/utils";
import { difficultyBadgeVariant, formatDuration, topicLabel } from "./format";
import { SanitizedHtml } from "./sanitized-html";

export interface ProblemStatementPanelProps {
  problem: ProblemPayload | null;
  className?: string;
}

export function ProblemStatementPanel({ problem, className }: ProblemStatementPanelProps) {
  if (!problem) {
    return (
      <Card className={cn("glow-card-cyan", className)}>
        <CardHeader>
          <CardTitle className="text-tv-text-body">No problem loaded</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-tv-text-body">
            Waiting for a problem to be assigned by the adaptive engine.
          </p>
        </CardContent>
      </Card>
    );
  }

  const visibleCases = (problem.test_cases ?? []).filter((tc) => !tc.is_hidden);
  const hiddenCount = (problem.test_cases ?? []).length - visibleCases.length;

  return (
    <Card className={cn("glow-card-cyan", className)}>
      <CardHeader className="gap-3">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <CardTitle className="font-display text-h4 leading-tight text-tv-text-hi uppercase">
            {problem.name}
          </CardTitle>
          <Badge variant={difficultyBadgeVariant(problem.difficulty_label)} className="shrink-0">
            {problem.difficulty_label}
          </Badge>
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          <Badge variant="outline">{topicLabel(problem.topic)}</Badge>
          {problem.subtopic && <Badge variant="ghost">{problem.subtopic}</Badge>}
          <Badge variant="secondary" className="uppercase">
            {problem.source}
          </Badge>
          <span className="ml-auto font-mono text-xs text-tv-text-body">
            ~{formatDuration(problem.avg_time_ms)} avg
          </span>
        </div>
        {problem.tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {problem.tags.map((tag) => (
              <span
                key={tag}
                className="rounded-tv-chip bg-tv-surface-2 px-1.5 py-0.5 font-mono text-[10px] text-tv-text-body uppercase"
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[320px] pr-3">
          <SanitizedHtml html={problem.statement} />

          {visibleCases.length > 0 && (
            <div className="mt-4 space-y-2">
              <h4 className="font-display text-sm font-bold text-tv-text-hi uppercase tracking-wide">
                Sample cases
              </h4>
              {visibleCases.map((tc, i) => (
                <div
                  key={i}
                  className="grid grid-cols-1 gap-2 rounded-tv-btn border border-tv-border bg-tv-surface-deep p-2 font-mono text-xs sm:grid-cols-2"
                >
                  <div>
                    <div className="mb-1 text-tv-text-body uppercase">Input</div>
                    <pre className="overflow-x-auto whitespace-pre-wrap text-tv-text-hi">{tc.input}</pre>
                  </div>
                  <div>
                    <div className="mb-1 text-tv-text-body uppercase">Output</div>
                    <pre className="overflow-x-auto whitespace-pre-wrap text-tv-text-hi">{tc.output}</pre>
                  </div>
                  {tc.explanation && (
                    <p className="col-span-full text-tv-text-body">{tc.explanation}</p>
                  )}
                </div>
              ))}
              {hiddenCount > 0 && (
                <p className="font-mono text-xs text-tv-text-body">
                  + {hiddenCount} hidden case{hiddenCount === 1 ? "" : "s"}
                </p>
              )}
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
