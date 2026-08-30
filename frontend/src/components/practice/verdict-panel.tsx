"use client";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { SubmitResponse } from "@/lib/api/types";
import { cn } from "@/lib/utils";
import { formatDuration, formatMemory } from "./format";
import { ThetaDelta } from "./theta-gauge";

export interface VerdictPanelProps {
  result: SubmitResponse;
  className?: string;
}

/** Judge0 status_id -> badge semantics. 3 = Accepted; 4 = Wrong Answer; 5 = TLE; 6 = Compile Error. */
function verdictVariant(result: SubmitResponse): "success" | "warning" | "error" {
  if (result.is_correct) return "success";
  if (result.verdict.status_id === 5) return "warning";
  return "error";
}

export function VerdictPanel({ result, className }: VerdictPanelProps) {
  const variant = verdictVariant(result);
  const errorText = result.verdict.compile_output || result.verdict.stderr || result.verdict.message;

  return (
    <Card
      className={cn(
        variant === "success" && "glow-card-cyan",
        variant === "error" && "glow-card-rose",
        className,
      )}
    >
      <CardHeader className="flex flex-row items-center justify-between gap-2">
        <CardTitle className="font-display text-h4 uppercase text-tv-text-hi">
          {result.is_correct ? "Accepted" : "Not quite"}
        </CardTitle>
        <Badge variant={variant} className="rounded-tv-pill">
          {result.verdict.status_desc}
        </Badge>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex flex-wrap gap-x-4 gap-y-1 font-mono text-xs text-tv-text-body">
          <span>Runtime: {formatDuration(result.verdict.time_ms)}</span>
          <span>Memory: {formatMemory(result.verdict.memory_kb)}</span>
          <span>Question #{result.question_count}</span>
        </div>

        {errorText && (
          <pre className="max-h-40 overflow-auto rounded-tv-btn border border-tv-rose/30 bg-tv-surface-deep p-2 font-mono text-xs whitespace-pre-wrap text-tv-rose">
            {errorText}
          </pre>
        )}

        {!!result.carelessness_penalty && result.carelessness_penalty > 0 && (
          <p className="rounded-tv-btn border border-tv-warning/30 bg-tv-warning/10 px-2 py-1.5 font-mono text-xs text-tv-warning">
            Carelessness penalty applied — that answer came back fast after a miss. Slow down
            before your next submit.
          </p>
        )}

        <div className="rounded-tv-btn border border-tv-border bg-tv-surface-deep px-3 py-2">
          <ThetaDelta before={result.theta_before} after={result.theta_after} />
        </div>
      </CardContent>
    </Card>
  );
}
