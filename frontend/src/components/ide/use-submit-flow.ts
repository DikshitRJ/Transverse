"use client";

/**
 * Drives the Submit handshake (plan.md §3.1 / the FORGE brief's "single-run
 * token handshake"):
 *
 *   POST /execute        -> { judge0_token }
 *   GET  /execute/{token}   poll until is_done, 600ms backoff, 30s ceiling
 *   POST /practice/submit   -> SubmitResponse
 *
 * `lib/api/endpoints.ts` already exports this exact handshake as one
 * opaque call (`submitSolution`, built on `pollVerdict`) — the FOUNDATION
 * brief says to use it rather than rebuilding it. This hook intentionally
 * composes the same three underlying calls (`executeCode`, `getVerdict`,
 * `submitPracticeAnswer`) with the same 600ms/30s contract instead of
 * calling `submitSolution` as a black box, for one reason: the FORGE brief
 * requires the UI to "show queue vs. processing distinctly" using Judge0's
 * real status, and `pollVerdict`/`submitSolution` intentionally only
 * resolve once, with no way to observe intermediate poll results. Every
 * `GET /execute/{token}` response along the way is otherwise handled
 * identically to `pollVerdict` — same interval, same timeout, same
 * "keep polling until is_done" loop — this only adds the state update
 * `pollVerdict` has no hook for.
 */
import { useCallback, useRef, useState } from "react";
import { executeCode, getVerdict, submitPracticeAnswer } from "@/lib/api/endpoints";
import type { SubmitResponse, VerdictPollResponse } from "@/lib/api/types";

export type SubmitPhase = "idle" | "queued" | "processing" | "submitting" | "done" | "error";

export interface SubmitFlowState {
  phase: SubmitPhase;
  elapsedMs: number;
  result: SubmitResponse | null;
  verdict: VerdictPollResponse | null;
  error: string | null;
}

export interface RunSubmitParams {
  sessionId: string;
  problemId: string;
  languageId: number;
  sourceCode: string;
  customStdin?: string;
  timeTakenMs: number;
}

const POLL_INTERVAL_MS = 600;
const POLL_TIMEOUT_MS = 30_000;
const TICK_MS = 100;

const IDLE_STATE: SubmitFlowState = {
  phase: "idle",
  elapsedMs: 0,
  result: null,
  verdict: null,
  error: null,
};

export function useSubmitFlow() {
  const [state, setState] = useState<SubmitFlowState>(IDLE_STATE);
  const startedAtRef = useRef<number>(0);
  const tickHandleRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const stopTicking = useCallback(() => {
    if (tickHandleRef.current !== null) {
      clearInterval(tickHandleRef.current);
      tickHandleRef.current = null;
    }
  }, []);

  const startTicking = useCallback(() => {
    stopTicking();
    tickHandleRef.current = setInterval(() => {
      setState((s) => ({ ...s, elapsedMs: Date.now() - startedAtRef.current }));
    }, TICK_MS);
  }, [stopTicking]);

  const reset = useCallback(() => {
    stopTicking();
    abortRef.current?.abort();
    setState(IDLE_STATE);
  }, [stopTicking]);

  const run = useCallback(
    async (params: RunSubmitParams) => {
      stopTicking();
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;
      startedAtRef.current = Date.now();
      setState({ phase: "queued", elapsedMs: 0, result: null, verdict: null, error: null });
      startTicking();

      try {
        const { judge0_token } = await executeCode({
          problem_id: params.problemId,
          language_id: params.languageId,
          source_code: params.sourceCode,
          custom_stdin: params.customStdin,
        });

        const deadline = Date.now() + POLL_TIMEOUT_MS;
        let verdict: VerdictPollResponse | null = null;
        for (;;) {
          controller.signal.throwIfAborted();
          verdict = await getVerdict(judge0_token);
          const settledVerdict = verdict;
          setState((s) => ({
            ...s,
            phase: settledVerdict.is_done ? s.phase : settledVerdict.status_id <= 1 ? "queued" : "processing",
            verdict: settledVerdict,
          }));
          if (verdict.is_done) break;
          if (Date.now() >= deadline) throw new Error("Judge0 didn't respond in time — please try again.");
          await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
        }

        setState((s) => ({ ...s, phase: "submitting" }));

        const result = await submitPracticeAnswer({
          session_id: params.sessionId,
          problem_id: params.problemId,
          judge0_token,
          time_taken_ms: params.timeTakenMs,
        });

        stopTicking();
        setState((s) => ({ ...s, phase: "done", result, elapsedMs: Date.now() - startedAtRef.current }));
      } catch (err) {
        stopTicking();
        if (controller.signal.aborted) return;
        const message = err instanceof Error ? err.message : "Submit failed.";
        setState((s) => ({ ...s, phase: "error", error: message }));
      }
    },
    [startTicking, stopTicking],
  );

  return { state, run, reset };
}
