"use client";

import { CheckCircle2, XCircle, TriangleAlert, TrendingUp, TrendingDown } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { VerdictBadge, isCompileError } from "./verdict-badge";
import { VerdictFxStyles, verdictFxClass } from "./verdict-fx";
import { JudgeStatus } from "./judge-status";
import type { SubmitFlowState } from "./use-submit-flow";

export interface SubmitPanelProps {
  state: SubmitFlowState;
  hasSession: boolean;
}

function ThetaDelta({ before, after }: { before: number; after: number }) {
  const delta = after - before;
  const Icon = delta >= 0 ? TrendingUp : TrendingDown;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 font-mono text-xs",
        delta >= 0 ? "text-tv-cyan" : "text-tv-rose",
      )}
    >
      <Icon className="size-3.5" aria-hidden />
      theta {before.toFixed(2)} to {after.toFixed(2)} ({delta >= 0 ? "+" : ""}
      {delta.toFixed(2)})
    </span>
  );
}

export function SubmitPanel({ state, hasSession }: SubmitPanelProps) {
  if (state.phase === "idle") {
    return (
      <div className="flex h-full items-center justify-center rounded-tv-card border border-dashed border-tv-border p-6 text-center">
        <p className="max-w-xs text-sm text-tv-locked">
          {hasSession
            ? "Submit your solution to score it and advance your practice session."
            : "Start a practice session to submit for scoring. Use Run to check your code against the sample cases in the meantime."}
        </p>
      </div>
    );
  }

  if (state.phase === "queued" || state.phase === "processing" || state.phase === "submitting") {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 p-6">
        <JudgeStatus phase={state.phase} elapsedMs={state.elapsedMs} />
        {state.verdict && state.phase !== "submitting" && (
          <p className="font-mono text-xs text-tv-text-body/70">{state.verdict.status_desc}</p>
        )}
      </div>
    );
  }

  if (state.phase === "error") {
    return (
      <div className="flex h-full items-center justify-center rounded-tv-card border border-tv-rose/30 bg-tv-rose/5 p-6 text-center glow-card-rose">
        <div>
          <TriangleAlert className="mx-auto mb-2 size-5 text-tv-rose" aria-hidden />
          <p className="text-sm text-tv-rose">{state.error ?? "Submit failed."}</p>
        </div>
      </div>
    );
  }

  const result = state.result;
  if (!result) return null;

  const compileError = isCompileError(result.verdict.status_id);

  return (
    <div className={cn("rounded-tv-card border p-4", result.is_correct ? "border-tv-border" : "border-tv-rose/30")}>
      <VerdictFxStyles />
      <div className={cn("mb-3 flex items-center gap-2", verdictFxClass(result.is_correct))}>
        {result.is_correct ? (
          <CheckCircle2 className="size-5 text-tv-cyan" aria-hidden />
        ) : (
          <XCircle className="size-5 text-tv-rose" aria-hidden />
        )}
        <span className="font-mono text-sm font-semibold text-tv-text-hi">
          {result.is_correct ? "Accepted" : "Not accepted"}
        </span>
        <VerdictBadge statusId={result.verdict.status_id} statusDesc={result.verdict.status_desc} />
      </div>

      {compileError ? (
        <pre className="overflow-x-auto rounded-tv-btn bg-tv-surface-deep p-3 font-mono text-xs text-tv-rose">
          {result.verdict.compile_output || "The compiler reported an error but returned no output."}
        </pre>
      ) : (
        <div className="grid grid-cols-2 gap-3 font-mono text-xs text-tv-text-body">
          <span>{result.verdict.time_ms}ms</span>
          <span>{(result.verdict.memory_kb / 1024).toFixed(1)}MB</span>
        </div>
      )}

      {result.verdict.stderr && (
        <pre className="mt-2 overflow-x-auto rounded-tv-btn bg-tv-rose/10 p-2 font-mono text-xs text-tv-rose">
          {result.verdict.stderr}
        </pre>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-3 border-t border-tv-border pt-3">
        <ThetaDelta before={result.theta_before} after={result.theta_after} />
        <Badge variant="outline" className="font-mono text-[11px]">
          {result.session_status}
        </Badge>
        <span className="font-mono text-[11px] text-tv-text-body/70">
          question {result.question_count}
        </span>
        {typeof result.carelessness_penalty === "number" && result.carelessness_penalty > 0 && (
          <span className="font-mono text-[11px] text-tv-warning">
            carelessness -{result.carelessness_penalty.toFixed(2)}
          </span>
        )}
      </div>

      {result.next_problem && (
        <p className="mt-3 font-mono text-xs text-tv-text-body">
          Next up: <span className="text-tv-cyan">{result.next_problem.name}</span>
        </p>
      )}
    </div>
  );
}
