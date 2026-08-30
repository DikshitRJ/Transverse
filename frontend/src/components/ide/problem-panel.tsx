"use client";

import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { SafeHtml } from "@/components/content/safe-html";
import { HintPanel } from "@/components/practice/hint-panel";
import { useHint } from "@/components/practice/use-hint";
import type { ProblemPayload } from "@/lib/api/types";
import { cn } from "@/lib/utils";

function difficultyVariant(label: string): "success" | "warning" | "error" | "secondary" {
  const normalized = label.toLowerCase();
  if (normalized === "easy") return "success";
  if (normalized === "medium") return "warning";
  if (normalized === "hard" || normalized === "expert") return "error";
  return "secondary";
}

export interface ProblemPanelProps {
  problem: ProblemPayload;
  sessionId?: string;
}

export function ProblemPanel({ problem, sessionId }: ProblemPanelProps) {
  const hint = useHint(sessionId ?? null, problem.id);
  const sampleCases = (problem.test_cases ?? []).filter((tc) => !tc.is_hidden);

  return (
    <div className="flex h-full flex-col overflow-y-auto p-5">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Badge variant={difficultyVariant(problem.difficulty_label)} className="rounded-tv-pill capitalize">
          {problem.difficulty_label}
        </Badge>
        <Badge variant="outline" className="uppercase">
          {problem.source}
        </Badge>
        {problem.topic && (
          <span className="font-mono text-xs text-tv-text-body">
            {problem.topic}
            {problem.subtopic ? ` / ${problem.subtopic}` : ""}
          </span>
        )}
      </div>

      <h1 className="mb-3 font-display text-h2 text-tv-text-hi glow-text-cyan">{problem.name}</h1>

      {problem.tags.length > 0 && (
        <div className="mb-4 flex flex-wrap gap-1.5">
          {problem.tags.map((tag) => (
            <Badge key={tag} variant="secondary" className="font-mono text-[11px]">
              {tag}
            </Badge>
          ))}
        </div>
      )}

      <div className="mb-4 flex items-center gap-4 font-mono text-xs text-tv-text-body/70">
        <span>Solve rate {(problem.solve_rate * 100).toFixed(0)}%</span>
        <span>Avg {(problem.avg_time_ms / 1000).toFixed(1)}s</span>
      </div>

      <Separator className="mb-4 bg-tv-border" />

      <SafeHtml html={problem.statement ?? ""} />

      {sampleCases.length > 0 && (
        <div className="mt-6">
          <h2 className="mb-2 font-display text-h4 text-tv-text-hi">Sample Test Cases</h2>
          <div className="space-y-3">
            {sampleCases.map((tc, i) => (
              <div key={i} className="rounded-tv-btn border border-tv-border bg-tv-surface p-3">
                <div className="mb-1.5 font-mono text-xs font-semibold text-tv-text-body/70">
                  Example {i + 1}
                </div>
                <dl className="grid grid-cols-1 gap-2 font-mono text-xs sm:grid-cols-2">
                  <div>
                    <dt className="mb-1 text-tv-text-body/60">Input</dt>
                    <dd className="overflow-x-auto rounded-tv-chip bg-tv-surface-deep p-2 whitespace-pre text-tv-text-hi">
                      {tc.input}
                    </dd>
                  </div>
                  <div>
                    <dt className="mb-1 text-tv-text-body/60">Output</dt>
                    <dd className="overflow-x-auto rounded-tv-chip bg-tv-surface-deep p-2 whitespace-pre text-tv-text-hi">
                      {tc.output}
                    </dd>
                  </div>
                </dl>
                {tc.explanation && (
                  <p className={cn("mt-2 text-xs text-tv-text-body")}>{tc.explanation}</p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      <HintPanel hint={hint} className="mt-6" />
    </div>
  );
}
