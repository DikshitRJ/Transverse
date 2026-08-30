/**
 * In-memory async job store for POST /practice/{id}/hint -> GET /jobs/{id},
 * mirroring `jobs.Job` (backend/internal/jobs/queue.go). A hint job is
 * "queued" immediately, flips to "running" after ~800ms, and "done" (with
 * `output = { hint_level, hint_text }`) after ~2.5s — long enough that
 * Wave-1 can build a real pending state, short enough to not be annoying
 * in dev. On completion it also emits a `hint.ready` SSE event via the
 * mock event bus, so `useTransverseEvent("hint.ready", ...)` and the
 * GET /jobs/{id} fallback poll both resolve, exactly as the real
 * plan.md §3.2 race is supposed to work.
 */
import type { HintReadyEventData, Job } from "@/lib/api/types";
import { emitMockEvent } from "./event-bus";

const jobStore = new Map<string, Job>();
let jobCounter = 0;

const HINTS_BY_LEVEL: Record<number, string> = {
  1: "Think about what data structure gives you O(1) lookups — you're re-scanning the array on every element right now.",
  2: "A hash map from value -> index lets you check \"have I seen target - nums[i] before?\" in one pass.",
  3: "Concretely: iterate once, and before inserting nums[i], check if (target - nums[i]) is already a key in your map.",
};

export function createHintJob(userId: string, hintLevel: number): Job {
  jobCounter += 1;
  const id = `job-hint-${jobCounter}-${Date.now()}`;
  const now = new Date().toISOString();
  const job: Job = {
    id,
    user_id: userId,
    job_type: "hint",
    status: "queued",
    created_at: now,
  };
  jobStore.set(id, job);

  setTimeout(() => {
    const running = jobStore.get(id);
    if (!running) return;
    running.status = "running";
    running.started_at = new Date().toISOString();
  }, 800);

  setTimeout(() => {
    const done = jobStore.get(id);
    if (!done) return;
    const hintText =
      HINTS_BY_LEVEL[Math.min(hintLevel, 3)] ?? HINTS_BY_LEVEL[1]!;
    const output: HintReadyEventData = { hint_level: hintLevel, hint_text: hintText };
    done.status = "done";
    done.completed_at = new Date().toISOString();
    done.output = output;

    emitMockEvent({ type: "hint.ready", job_id: id, data: output });
  }, 2500);

  return job;
}

export function getJobById(id: string): Job | undefined {
  return jobStore.get(id);
}
