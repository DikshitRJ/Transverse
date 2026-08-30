/**
 * Fetch-based SSE client for GET /events/stream.
 *
 * NOT `EventSource` — the browser's native `EventSource` cannot set custom
 * request headers, and `/events/stream` sits behind `middleware.Auth`,
 * which only accepts `Authorization: Bearer <token>` (no cookie fallback —
 * verified against `backend/internal/middleware/auth.go`). So this reads
 * the stream manually via `fetch` + `ReadableStream`, parsing raw
 * `data: <json>\n\n` frames by hand.
 *
 * Frame shape note: the backend never sets a named SSE `event:` field
 * except the literal initial handshake (`event: connected\ndata: {}\n\n`,
 * see `realtime/handler.go`) and a `: keepalive` comment every 15s. Every
 * real event is a bare `data: {"type": "...", "job_id": "...", "data": {}}`
 * line — `TransverseEvent.type` is what you switch on, not the SSE
 * `event:` field.
 */

import { getAccessToken } from "@/lib/auth/token-store";
import type { TransverseEvent } from "@/lib/api/types";

export type ConnectionStatus = "idle" | "connecting" | "open" | "closed";

type EventListener = (event: TransverseEvent) => void;
type StatusListener = (status: ConnectionStatus) => void;

const STREAM_PATH = "/api/v1/events/stream";
const MAX_BACKOFF_MS = 30_000;
const BASE_BACKOFF_MS = 1000;

export class TransverseEventSource {
  private controller: AbortController | null = null;
  private eventListeners = new Set<EventListener>();
  private statusListeners = new Set<StatusListener>();
  private reconnectAttempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private started = false;
  private status: ConnectionStatus = "idle";

  start(): void {
    if (this.started) return;
    this.started = true;
    this.reconnectAttempt = 0;
    void this.connect();
  }

  stop(): void {
    this.started = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.controller?.abort();
    this.controller = null;
    this.setStatus("idle");
  }

  /** Restart with a fresh Authorization header — call after the access token changes. */
  restart(): void {
    if (!this.started) return;
    this.controller?.abort();
    this.reconnectAttempt = 0;
    void this.connect();
  }

  subscribe(listener: EventListener): () => void {
    this.eventListeners.add(listener);
    return () => {
      this.eventListeners.delete(listener);
    };
  }

  subscribeStatus(listener: StatusListener): () => void {
    this.statusListeners.add(listener);
    listener(this.status);
    return () => {
      this.statusListeners.delete(listener);
    };
  }

  getStatus(): ConnectionStatus {
    return this.status;
  }

  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    for (const listener of this.statusListeners) listener(status);
  }

  private async connect(): Promise<void> {
    const token = getAccessToken();
    if (!token) {
      // Not signed in yet — try again shortly rather than spinning immediately.
      this.scheduleReconnect();
      return;
    }

    this.controller = new AbortController();
    this.setStatus("connecting");

    try {
      const res = await fetch(STREAM_PATH, {
        headers: {
          Authorization: `Bearer ${token}`,
          Accept: "text/event-stream",
        },
        signal: this.controller.signal,
        cache: "no-store",
      });

      if (!res.ok || !res.body) {
        throw new Error(`SSE connect failed: ${res.status}`);
      }

      this.reconnectAttempt = 0;
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        let frameEnd = buffer.indexOf("\n\n");
        while (frameEnd !== -1) {
          const rawFrame = buffer.slice(0, frameEnd);
          buffer = buffer.slice(frameEnd + 2);
          this.handleFrame(rawFrame);
          frameEnd = buffer.indexOf("\n\n");
        }
      }
    } catch (err) {
      if (this.controller?.signal.aborted) return; // deliberate stop/restart, not a failure
      // swallow — reconnect handles recovery, no console noise for expected network blips
      void err;
    } finally {
      this.setStatus("closed");
      if (this.started) this.scheduleReconnect();
    }
  }

  private handleFrame(rawFrame: string): void {
    let eventName: string | undefined;
    const dataLines: string[] = [];

    for (const line of rawFrame.split("\n")) {
      if (line.startsWith(":")) continue; // heartbeat/comment
      if (line.startsWith("event:")) eventName = line.slice("event:".length).trim();
      else if (line.startsWith("data:")) dataLines.push(line.slice("data:".length).trim());
    }

    if (eventName === "connected") {
      this.setStatus("open");
      return;
    }
    if (dataLines.length === 0) return;

    try {
      const parsed = JSON.parse(dataLines.join("\n")) as TransverseEvent;
      if (this.status !== "open") this.setStatus("open");
      for (const listener of this.eventListeners) listener(parsed);
    } catch {
      // malformed frame — ignore rather than crash the stream loop
    }
  }

  private scheduleReconnect(): void {
    if (!this.started) return;
    const delay = Math.min(BASE_BACKOFF_MS * 2 ** this.reconnectAttempt, MAX_BACKOFF_MS);
    this.reconnectAttempt += 1;
    this.reconnectTimer = setTimeout(() => {
      if (this.started) void this.connect();
    }, delay);
  }
}

/** One connection for the whole app — see `SSEProvider`. */
export const transverseEventSource = new TransverseEventSource();
