"use client";

import { createContext, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { transverseEventSource, type ConnectionStatus } from "@/lib/realtime/sse-client";
import { subscribeAccessToken } from "@/lib/auth/token-store";
import type { TransverseEvent, TransverseEventType } from "@/lib/api/types";

type EventListener<T = unknown> = (event: TransverseEvent<T>) => void;

interface SSEContextValue {
  status: ConnectionStatus;
  /** Subscribe to one event type. Returns an unsubscribe function. Prefer the `useTransverseEvent` hook in components. */
  subscribe: <T = unknown>(type: TransverseEventType, listener: EventListener<T>) => () => void;
  /** Subscribe to every event regardless of type. */
  subscribeAll: (listener: EventListener) => () => void;
}

const SSEContext = createContext<SSEContextValue | null>(null);

/**
 * One global subscription to GET /events/stream for the whole app (plan.md
 * §5 "SSE provider"). Mount once near the root (see `AppProviders`).
 *
 * Starts the connection once an access token exists, restarts it whenever
 * the token changes (login, silent refresh, logout — the Authorization
 * header can't be swapped mid-stream, only reconnected), and tears it down
 * on unmount.
 *
 * Also owns the default cache-invalidation wiring: `roadmap.updated` and
 * `node.unlocked` invalidate the `["roadmap"]` query, `job.completed`/
 * `job.failed` invalidate `["jobs", jobId]`. Feature code should still use
 * `useTransverseEvent` directly for anything beyond a plain invalidation
 * (e.g. driving the unlock animation off `node.unlocked`, or resolving a
 * hint promise off `hint.ready`).
 */
export function SSEProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<ConnectionStatus>(transverseEventSource.getStatus());

  useEffect(() => {
    transverseEventSource.start();
    return () => transverseEventSource.stop();
  }, []);

  useEffect(() => {
    return subscribeAccessToken(() => transverseEventSource.restart());
  }, []);

  useEffect(() => {
    return transverseEventSource.subscribeStatus(setStatus);
  }, []);

  useEffect(() => {
    return transverseEventSource.subscribe((event) => {
      switch (event.type) {
        case "roadmap.updated":
        case "node.unlocked":
          void queryClient.invalidateQueries({ queryKey: ["roadmap"] });
          break;
        case "job.completed":
        case "job.failed":
        case "hint.ready":
          void queryClient.invalidateQueries({ queryKey: ["jobs", event.job_id] });
          break;
      }
    });
  }, [queryClient]);

  const value = useMemo<SSEContextValue>(
    () => ({
      status,
      subscribe: (type, listener) =>
        transverseEventSource.subscribe((event) => {
          if (event.type === type) listener(event as TransverseEvent<never>);
        }),
      subscribeAll: (listener) => transverseEventSource.subscribe(listener),
    }),
    [status],
  );

  return <SSEContext.Provider value={value}>{children}</SSEContext.Provider>;
}

export function useTransverseEvents(): SSEContextValue {
  const ctx = useContext(SSEContext);
  if (!ctx) throw new Error("useTransverseEvents must be used within <SSEProvider>");
  return ctx;
}

/**
 * Sugar over `useTransverseEvents().subscribe` for the common case: react
 * to one event type for the lifetime of a component.
 *
 *   useTransverseEvent("node.unlocked", (e) => { ...play unlock animation... });
 */
export function useTransverseEvent<T = unknown>(
  type: TransverseEventType,
  listener: EventListener<T>,
): void {
  const { subscribe } = useTransverseEvents();
  const listenerRef = useRef(listener);
  listenerRef.current = listener;

  useEffect(() => {
    return subscribe<T>(type, (event) => listenerRef.current(event));
  }, [type, subscribe]);
}
