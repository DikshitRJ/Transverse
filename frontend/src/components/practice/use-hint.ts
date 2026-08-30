"use client";

/**
 * Async hint resolution (plan.md §3.2). `POST /practice/{id}/hint` returns
 * `202 { job_id }`; the actual hint text arrives on whichever of these two
 * paths wins:
 *
 *  1. the `hint.ready` SSE event (near-instant once the worker finishes —
 *     the mock fires it at the same moment the job flips to "done", ~2.5s)
 *  2. the `GET /jobs/{id}` poll fallback (`pollHintJob` in lib/api/endpoints.ts,
 *     2s interval / 45s ceiling) — the guaranteed-eventually-correct path
 *     if the SSE stream is down or reconnecting
 *
 * Implemented as a real `Promise.race` (not just "whichever setState call
 * lands second, wins") so the loser is actually torn down: the SSE side
 * unsubscribes itself the moment it wins or the request settles, and the
 * poll side receives an `AbortSignal` that gets aborted the moment the SSE
 * side wins, so a won-by-SSE race doesn't keep silently polling every 2s in
 * the background for up to 45s.
 *
 * A `429` (the real handler string is `"rate limit"`) is surfaced as
 * `status: "rate-limited"` — render this in-context on Byte, never as a red
 * error toast (plan.md §3.2, FOUNDATION.md §6).
 */
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/lib/api/client";
import { requestHint as requestHintApi, pollHintJob } from "@/lib/api/endpoints";
import type { HintReadyEventData } from "@/lib/api/types";
import { useTransverseEvents } from "@/components/providers/sse-provider";

export type HintStatus = "idle" | "pending" | "ready" | "rate-limited" | "error";

export interface UseHintResult {
  status: HintStatus;
  /** hint_level -> hint_text, accumulated across every level fetched so far this session. */
  hints: Record<number, string>;
  /** The highest hint level successfully fetched. 0 if none yet. */
  maxLevelReady: number;
  pendingLevel: number | null;
  error: string | null;
  requestHint: (hintLevel: number) => void;
}

export function useHint(sessionId: string | null, problemId?: string | null): UseHintResult {
  const { subscribe } = useTransverseEvents();
  const [status, setStatus] = useState<HintStatus>("idle");
  const [hints, setHints] = useState<Record<number, string>>({});
  const [pendingLevel, setPendingLevel] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);
  const abortRef = useRef<AbortController | null>(null);
  const unsubRef = useRef<(() => void) | null>(null);

  // Reset hints when problem changes
  useEffect(() => {
    setHints({});
    setStatus("idle");
    setPendingLevel(null);
    setError(null);
  }, [problemId, sessionId]);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      abortRef.current?.abort();
      unsubRef.current?.();
    };
  }, []);

  const requestHint = useCallback(
    (hintLevel: number) => {
      const targetId = sessionId || problemId;
      if (!targetId) return;
      // Tear down any still-in-flight previous race before starting a new one.
      abortRef.current?.abort();
      unsubRef.current?.();

      setStatus("pending");
      setPendingLevel(hintLevel);
      setError(null);

      void (async () => {
        try {
          const { job_id } = await requestHintApi(targetId, { hint_level: hintLevel });
          if (!mountedRef.current) return;

          const controller = new AbortController();
          abortRef.current = controller;

          const ssePromise = new Promise<HintReadyEventData>((resolve) => {
            const unsub = subscribe<HintReadyEventData>("hint.ready", (event) => {
              if (event.job_id !== job_id) return;
              unsub();
              resolve(event.data);
            });
            unsubRef.current = unsub;
          });

          const pollPromise = pollHintJob(job_id, {
            pollIntervalMs: 2000,
            timeoutMs: 45_000,
            signal: controller.signal,
          }).then((job) => job.output as HintReadyEventData);
          // A won-by-SSE abort rejects this branch — that rejection must
          // never surface as an unhandled promise rejection.
          pollPromise.catch(() => {});

          const data = await Promise.race([ssePromise, pollPromise]);

          unsubRef.current?.();
          controller.abort();

          if (!mountedRef.current) return;

          let parsedData: Record<string, unknown> | null = (data && typeof data === "object") ? (data as unknown as Record<string, unknown>) : null;
          if (typeof data === "string") {
            try {
              parsedData = JSON.parse(data) as Record<string, unknown>;
            } catch {
              parsedData = { hint_level: hintLevel, hint_text: data };
            }
          }

          const finalLevel = (parsedData && typeof parsedData.hint_level === "number")
            ? parsedData.hint_level
            : hintLevel;
          const finalText = (parsedData && typeof parsedData.hint_text === "string")
            ? parsedData.hint_text
            : (typeof data === "string" ? data : "Hint ready.");

          setHints((prev) => ({ ...prev, [finalLevel]: finalText }));
          setStatus("ready");
          setPendingLevel(null);
        } catch (err) {
          if (!mountedRef.current) return;
          if (err instanceof ApiError && err.status === 429) {
            setStatus("rate-limited");
            setError("Byte needs a moment — hints are rate-limited. Try again shortly.");
          } else {
            setStatus("error");
            setError(err instanceof Error ? err.message : "Couldn't fetch that hint.");
          }
          setPendingLevel(null);
        }
      })();
    },
    [sessionId, problemId, subscribe],
  );

  const maxLevelReady = Object.keys(hints).reduce((max, k) => Math.max(max, Number(k)), 0);

  return { status, hints, maxLevelReady, pendingLevel, error, requestHint };
}
