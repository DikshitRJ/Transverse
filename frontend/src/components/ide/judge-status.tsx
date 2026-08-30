"use client";

/**
 * Live "waiting on Judge0" indicator for the Submit handshake (execute ->
 * poll -> practice/submit). The FORGE brief is explicit that this is core
 * UX, not an afterthought: "show queue vs. processing distinctly, and
 * never let the UI look frozen." Distinguishes the two states using the
 * real `status_id`/`status_desc` from each `GET /execute/{token}` poll
 * (1 = In Queue, 2 = Processing per `VerdictPollResponse`'s doc comment in
 * `lib/api/types.ts`) rather than guessing from elapsed time.
 */
import { Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

export type JudgePhase = "queued" | "processing" | "submitting";

const PHASE_LABEL: Record<JudgePhase, string> = {
  queued: "In queue",
  processing: "Running on Judge0",
  submitting: "Recording result",
};

export interface JudgeStatusProps {
  phase: JudgePhase;
  elapsedMs: number;
  className?: string;
}

function formatElapsed(ms: number): string {
  const seconds = ms / 1000;
  return `${seconds.toFixed(1)}s`;
}

export function JudgeStatus({ phase, elapsedMs, className }: JudgeStatusProps) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-tv-btn border border-tv-border-cyan bg-tv-surface-deep px-3 py-2 font-mono text-xs text-tv-text-body",
        className,
      )}
      role="status"
      aria-live="polite"
    >
      <Loader2 className="size-3.5 animate-spin text-tv-cyan" aria-hidden />
      <span className="text-tv-text-hi">{PHASE_LABEL[phase]}</span>
      <span className="text-tv-text-body/70">{formatElapsed(elapsedMs)}</span>
      {phase === "queued" && (
        <span className="ml-auto text-tv-text-body/60">Judge0 can take a few seconds under load</span>
      )}
    </div>
  );
}
