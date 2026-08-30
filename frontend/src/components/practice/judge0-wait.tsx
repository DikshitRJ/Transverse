"use client";

/**
 * The deliberate "your code is running" state (plan.md §3.1 — "Judge0 can be
 * slow — design the waiting state deliberately, it's a core part of this
 * UX"). `submitSolution()` (lib/api/endpoints.ts) is a single opaque
 * `execute -> poll -> submit` promise per FOUNDATION.md's instruction to use
 * that helper rather than re-implement the handshake, so this component
 * can't show real per-poll status transitions — instead it drives an honest
 * elapsed-time counter plus a cosmetic phase label that tracks the same
 * timing budget the poller actually uses (600ms backoff, ~30s ceiling,
 * `pollVerdict` in endpoints.ts), so the copy never claims more than the
 * elapsed clock supports.
 */
import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";

export interface Judge0WaitProps {
  startedAt: number;
  className?: string;
}

function phaseLabel(elapsedMs: number): string {
  if (elapsedMs < 1500) return "Sending to Judge0…";
  if (elapsedMs < 6000) return "Compiling and running your code…";
  if (elapsedMs < 15000) return "Still running — larger inputs take a moment…";
  return "This is taking a while. Judge0 allows up to 30s before we give up.";
}

export function Judge0Wait({ startedAt, className }: Judge0WaitProps) {
  const [now, setNow] = useState(startedAt);

  useEffect(() => {
    const interval = setInterval(() => setNow(Date.now()), 250);
    return () => clearInterval(interval);
  }, []);

  const elapsedMs = Math.max(0, now - startedAt);
  const elapsedSeconds = (elapsedMs / 1000).toFixed(1);

  return (
    <div
      className={cn(
        "glass-panel flex flex-col items-center gap-3 rounded-tv-card border-tv-border-cyan px-6 py-8 text-center",
        className,
      )}
      role="status"
      aria-live="polite"
    >
      <span
        aria-hidden
        className="size-8 animate-spin rounded-full border-2 border-tv-cyan/25 border-t-tv-cyan motion-reduce:animate-none"
      />
      <p className="font-mono text-sm text-tv-text-hi">{phaseLabel(elapsedMs)}</p>
      <p className="font-mono text-xs tabular-nums text-tv-text-body">{elapsedSeconds}s elapsed · up to 30s</p>
    </div>
  );
}
