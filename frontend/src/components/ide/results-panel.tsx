"use client";

/**
 * Results for the "Run" path — `POST /execute/batch` against the
 * problem's sample (non-hidden) test cases. Independent of Submit; this
 * response never feeds `/practice/submit` (see `lib/api/endpoints.ts`'s
 * doc comment on `executeBatch`).
 */
import { CheckCircle2, XCircle, TriangleAlert } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { BatchExecuteResponse, TestCaseResult } from "@/lib/api/types";
import { VerdictBadge, isCompileError } from "./verdict-badge";
import { VerdictFxStyles, verdictFxClass } from "./verdict-fx";
import { JudgeStatus } from "./judge-status";

export type RunStatus = "idle" | "running" | "done" | "error";

export interface ResultsPanelProps {
  status: RunStatus;
  result: BatchExecuteResponse | null;
  error: string | null;
  elapsedMs: number;
  sampleCount: number;
  /** Bumped on every new run so result rows remount and the pass/fail
   * animation plays fresh each time. */
  runVersion: number;
}

function TestCaseRow({ testCase }: { testCase: TestCaseResult }) {
  return (
    <div
      className={cn(
        "rounded-tv-btn border p-3",
        testCase.passed ? "border-tv-border bg-tv-surface" : "border-tv-rose/30 bg-tv-surface",
        verdictFxClass(testCase.passed),
      )}
    >
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          {testCase.passed ? (
            <CheckCircle2 className="size-4 text-tv-cyan" aria-hidden />
          ) : (
            <XCircle className="size-4 text-tv-rose" aria-hidden />
          )}
          <span className="font-mono text-xs font-semibold text-tv-text-hi">
            Case {testCase.index + 1}
          </span>
          <VerdictBadge statusId={testCase.status_id} statusDesc={testCase.status_desc} />
        </div>
        <span className="font-mono text-[11px] text-tv-text-body/70">
          {testCase.time_ms}ms | {(testCase.memory_kb / 1024).toFixed(1)}MB
        </span>
      </div>

      <dl className="grid grid-cols-1 gap-2 font-mono text-xs sm:grid-cols-3">
        <div>
          <dt className="mb-1 text-tv-text-body/60">Input</dt>
          <dd className="overflow-x-auto rounded-tv-chip bg-tv-surface-deep p-2 whitespace-pre text-tv-text-body">
            {testCase.input || "—"}
          </dd>
        </div>
        <div>
          <dt className="mb-1 text-tv-text-body/60">Expected</dt>
          <dd className="overflow-x-auto rounded-tv-chip bg-tv-surface-deep p-2 whitespace-pre text-tv-text-hi">
            {testCase.expected_output || "—"}
          </dd>
        </div>
        <div>
          <dt className="mb-1 text-tv-text-body/60">Actual</dt>
          <dd
            className={cn(
              "overflow-x-auto rounded-tv-chip p-2 whitespace-pre",
              testCase.passed ? "bg-tv-surface-deep text-tv-text-hi" : "bg-tv-rose/10 text-tv-rose",
            )}
          >
            {testCase.actual_stdout || "—"}
          </dd>
        </div>
      </dl>

      {testCase.stderr && (
        <div className="mt-2">
          <div className="mb-1 font-mono text-xs text-tv-rose/80">stderr</div>
          <pre className="overflow-x-auto rounded-tv-chip bg-tv-rose/10 p-2 font-mono text-xs text-tv-rose">
            {testCase.stderr}
          </pre>
        </div>
      )}
    </div>
  );
}

export function ResultsPanel({
  status,
  result,
  error,
  elapsedMs,
  sampleCount,
  runVersion,
}: ResultsPanelProps) {
  if (status === "idle") {
    return (
      <div className="flex h-full items-center justify-center rounded-tv-card border border-dashed border-tv-border p-6 text-center">
        <p className="max-w-xs text-sm text-tv-locked">
          Run your code against the sample test cases to see results here.
        </p>
      </div>
    );
  }

  if (status === "running") {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 p-6">
        <JudgeStatus phase="processing" elapsedMs={elapsedMs} />
        <p className="text-xs text-tv-text-body">
          Running against {sampleCount} sample test case{sampleCount === 1 ? "" : "s"}
        </p>
      </div>
    );
  }

  if (status === "error") {
    return (
      <div className="flex h-full items-center justify-center rounded-tv-card border border-tv-rose/30 bg-tv-rose/5 p-6 text-center glow-card-rose">
        <div>
          <TriangleAlert className="mx-auto mb-2 size-5 text-tv-rose" aria-hidden />
          <p className="text-sm text-tv-rose">{error ?? "Run failed."}</p>
        </div>
      </div>
    );
  }

  if (!result) return null;

  if (isCompileError(result.overall_status_id)) {
    const compileOutput = result.test_cases.find((tc) => tc.compile_output)?.compile_output;
    return (
      <div className="rounded-tv-card border border-tv-rose/40 bg-tv-rose/5 p-4 glow-card-rose">
        <div className="mb-2 flex items-center gap-2">
          <XCircle className="size-5 text-tv-rose" aria-hidden />
          <span className="font-mono text-sm font-semibold text-tv-rose">Compilation Error</span>
        </div>
        <pre className="overflow-x-auto rounded-tv-btn bg-tv-surface-deep p-3 font-mono text-xs text-tv-rose">
          {compileOutput || "The compiler reported an error but returned no output."}
        </pre>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col gap-3">
      <VerdictFxStyles />
      <div className="flex items-center justify-between rounded-tv-card border border-tv-border bg-tv-surface p-3">
        <div className="flex items-center gap-2">
          {result.all_passed ? (
            <CheckCircle2 className="size-5 text-tv-cyan" aria-hidden />
          ) : (
            <XCircle className="size-5 text-tv-rose" aria-hidden />
          )}
          <span className="font-mono text-sm font-semibold text-tv-text-hi">
            {result.passed_count} / {result.total_count} passed
          </span>
          <Badge variant={result.all_passed ? "success" : "error"}>{result.overall_status}</Badge>
        </div>
        <span className="font-mono text-xs text-tv-text-body/70">
          max {result.max_time_ms}ms | {(result.max_memory_kb / 1024).toFixed(1)}MB
        </span>
      </div>

      <div className="flex-1 space-y-2 overflow-y-auto pr-1">
        {result.test_cases.map((tc) => (
          <TestCaseRow key={`${runVersion}-${tc.index}`} testCase={tc} />
        ))}
      </div>
    </div>
  );
}
