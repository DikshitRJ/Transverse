"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { useTransverseEvent } from "@/components/providers/sse-provider";
import { ApiError } from "@/lib/api/client";
import {
  confirmEvidenceUpload,
  connectCodeforcesEvidence,
  connectGithubEvidence,
  connectLeetcodeEvidence,
  getEvidenceUploadUrl,
} from "@/lib/api/endpoints";
import type { ConnectorKind, SyncSource, UploadKind } from "@/components/onboarding/sync/types";

/**
 * Drives the per-source status list on `/onboarding/sync`
 * (POST /evidence/upload-url -> presigned PUT -> POST /evidence/{id}/confirm,
 * and POST /evidence/{github|leetcode|codeforces}).
 *
 * ## The gap this hook has to work around
 *
 * The brief for this screen says to resolve completion via the
 * `job.completed`/`job.failed` SSE events with a `GET /jobs/{id}` polling
 * fallback — the same pattern hints use (plan.md §3.2). That pattern needs
 * a `job_id` to key off. Evidence processing does not provide one, on
 * either side I checked:
 *
 *  - **Types**: `EvidenceUploadURLResponse` is `{evidence_id, upload_url}`,
 *    `EvidenceConfirmResponse` is `{status}`, and the 3 connector responses
 *    are `{evidence_id, status}` (`lib/api/types.ts`) — no `job_id` field
 *    anywhere on any evidence response.
 *  - **Mocks**: `mocks/fixtures/evidence.ts` manages source status
 *    (`pending -> processing -> done`, or `fetching -> processing -> done`
 *    for connectors) entirely in its own module-local `Map`, never touching
 *    `mocks/fixtures/jobs.ts`'s job store and never calling `emitMockEvent`
 *    — so a job is never created and no SSE event ever fires for evidence
 *    in mock mode.
 *  - **Real backend**: `evidence.Service` *does* enqueue a real job
 *    (`s.jobQueue.EnqueueHypothesisGeneration(ctx, source.UserID)`,
 *    `backend/internal/evidence/service.go` lines 123 & 222) — but discards
 *    the id it gets back (`_, _ = ...`) rather than returning it to the
 *    HTTP handler, so it never reaches the client either. There's also no
 *    `GET /evidence/{id}` status route mounted anywhere.
 *
 * So there is currently no way, live or mocked, for the client to learn
 * exactly when one specific evidence source finishes. Given that, this
 * hook does the most honest thing available:
 *
 *  1. Subscribes to `job.completed` globally (this doesn't need per-source
 *     wiring — once KEYSTONE's evidence flow starts publishing that real
 *     hypothesis-generation job's completion, this starts working exactly
 *     as intended) and, on any such event, resolves every source still
 *     in-flight to `"done"`. It's not attributable to one specific source,
 *     but it's a genuine signal, not a fabricated one.
 *  2. Does **not** wire `job.failed` the same way — flipping unrelated
 *     in-flight sources to `"failed"` off a signal that might belong to a
 *     totally different job would be a false negative, which is worse
 *     than just not reacting to it.
 *  3. Falls back to a bounded timeout per source (see `ASSUMED_DONE_MS`)
 *     so the mock demo (and a real backend before the SSE wiring lands)
 *     still reaches a real "done" state instead of spinning forever —
 *     clearly not a truth claim, just a UX floor.
 *
 * Genuine, request-level failures (a non-2xx from any of the 5 endpoints,
 * or the presigned PUT itself failing) always resolve to `"failed"`
 * immediately — that part of the signal is real.
 */
const ASSUMED_DONE_MS = 6000;

function errorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.message;
  if (err instanceof Error) return err.message;
  return "Something went wrong.";
}

let sourceCounter = 0;
function nextSourceId(): string {
  sourceCounter += 1;
  return `sync-source-${sourceCounter}`;
}

export function useEvidenceSync() {
  const [sources, setSources] = useState<SyncSource[]>([]);
  const timers = useRef(new Map<string, ReturnType<typeof setTimeout>>());

  const updateSource = useCallback((id: string, patch: Partial<SyncSource>) => {
    setSources((prev) => prev.map((s) => (s.id === id ? { ...s, ...patch } : s)));
  }, []);

  const scheduleAssumedDone = useCallback((id: string) => {
    const existing = timers.current.get(id);
    if (existing) clearTimeout(existing);
    const handle = setTimeout(() => {
      setSources((prev) =>
        prev.map((s) =>
          s.id === id && (s.status === "pending" || s.status === "fetching" || s.status === "processing")
            ? { ...s, status: "done" }
            : s,
        ),
      );
      timers.current.delete(id);
    }, ASSUMED_DONE_MS);
    timers.current.set(id, handle);
  }, []);

  // Best-effort global completion signal — see the doc comment above.
  useTransverseEvent("job.completed", () => {
    setSources((prev) =>
      prev.map((s) =>
        s.status === "pending" || s.status === "fetching" || s.status === "processing"
          ? { ...s, status: "done" }
          : s,
      ),
    );
    for (const handle of timers.current.values()) clearTimeout(handle);
    timers.current.clear();
  });

  const addConnector = useCallback(
    async (kind: ConnectorKind, value: string) => {
      const id = nextSourceId();
      const label = kind === "codeforces" ? `Codeforces: ${value}` : `${kind === "github" ? "GitHub" : "LeetCode"}: ${value}`;
      setSources((prev) => [...prev, { id, kind, label, status: "fetching" }]);

      try {
        const connect =
          kind === "github" ? connectGithubEvidence : kind === "leetcode" ? connectLeetcodeEvidence : connectCodeforcesEvidence;
        const res = await connect(value);
        updateSource(id, { evidenceId: res.evidence_id, status: "processing" });
        scheduleAssumedDone(id);
      } catch (err) {
        updateSource(id, { status: "failed", errorMessage: errorMessage(err) });
      }
    },
    [updateSource, scheduleAssumedDone],
  );

  const addUpload = useCallback(
    async (kind: UploadKind, file: File) => {
      const id = nextSourceId();
      setSources((prev) => [...prev, { id, kind, label: file.name, status: "pending" }]);

      try {
        const { evidence_id, upload_url } = await getEvidenceUploadUrl({
          kind,
          filename: file.name,
        });
        updateSource(id, { evidenceId: evidence_id });

        const putRes = await fetch(upload_url, {
          method: "PUT",
          body: file,
          headers: { "Content-Type": file.type || "application/octet-stream" },
        });
        if (!putRes.ok) {
          throw new Error(`Upload to storage failed (${putRes.status}).`);
        }

        await confirmEvidenceUpload(evidence_id);
        updateSource(id, { status: "processing" });
        scheduleAssumedDone(id);
      } catch (err) {
        updateSource(id, { status: "failed", errorMessage: errorMessage(err) });
      }
    },
    [updateSource, scheduleAssumedDone],
  );

  const removeSource = useCallback((id: string) => {
    const handle = timers.current.get(id);
    if (handle) clearTimeout(handle);
    timers.current.delete(id);
    setSources((prev) => prev.filter((s) => s.id !== id));
  }, []);

  const summary = useMemo(
    () => ({
      total: sources.length,
      done: sources.filter((s) => s.status === "done").length,
      failed: sources.filter((s) => s.status === "failed").length,
      inFlight: sources.filter((s) => s.status === "pending" || s.status === "fetching" || s.status === "processing").length,
    }),
    [sources],
  );

  return { sources, addConnector, addUpload, removeSource, summary };
}
