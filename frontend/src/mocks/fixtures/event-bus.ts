/**
 * In-memory pub/sub the mock SSE stream handler (handlers.ts, GET
 * /events/stream) reads from. Anything in the mock layer that should
 * "push" a realtime event — a hint job finishing, a roadmap node
 * unlocking — calls `emitMockEvent` and every open mock SSE connection
 * receives it as a `data: <json>\n\n` frame, exactly like the real
 * `jobs.redisQueue.PublishEvent` -> `realtime.Handler.StreamEvents` path.
 */
import type { TransverseEvent } from "@/lib/api/types";

type Listener = (event: TransverseEvent) => void;

const listeners = new Set<Listener>();

export function emitMockEvent(event: TransverseEvent): void {
  for (const listener of listeners) listener(event);
}

export function subscribeMockEvents(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
