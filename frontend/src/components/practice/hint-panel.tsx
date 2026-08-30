"use client";

import Image from "next/image";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { UseHintResult } from "./use-hint";

export interface HintPanelProps {
  hint: UseHintResult;
  className?: string;
}

const LEVELS = [1, 2, 3];

/** Byte-voiced, escalating hint affordance. Rate limits and errors render in-context here, never as a toast. */
export function HintPanel({ hint, className }: HintPanelProps) {
  const { status, hints, maxLevelReady, pendingLevel, error, requestHint } = hint;
  const revealedLevels = LEVELS.filter((l) => hints[l] !== undefined);
  const nextLevel = maxLevelReady + 1 <= 3 ? maxLevelReady + 1 : null;

  return (
    <div className={cn("glass-panel rounded-tv-card p-4", className)}>
      <div className="mb-3 flex items-center gap-2">
        <Image
          src="/figma/byte-mascot-chip.png"
          alt=""
          width={28}
          height={28}
          className="size-7 shrink-0 rounded-full object-contain"
        />
        <h3 className="font-display text-sm font-bold text-tv-text-hi uppercase tracking-wide">
          Ask Byte for a hint
        </h3>
      </div>

      {revealedLevels.length > 0 && (
        <ol className="mb-3 space-y-2">
          {revealedLevels.map((level) => (
            <li
              key={level}
              className="rounded-tv-btn border border-tv-border-cyan bg-tv-surface-deep p-2.5 font-mono text-xs text-tv-text-hi"
            >
              <span className="mb-1 block text-tv-cyan uppercase">Hint {level}</span>
              {hints[level]}
            </li>
          ))}
        </ol>
      )}

      {status === "pending" && (
        <p
          className="mb-3 flex items-center gap-2 font-mono text-xs text-tv-text-body"
          role="status"
          aria-live="polite"
        >
          <span
            aria-hidden
            className="size-3 animate-spin rounded-full border-2 border-tv-cyan/25 border-t-tv-cyan motion-reduce:animate-none"
          />
          Byte is thinking about hint {pendingLevel}…
        </p>
      )}

      {status === "rate-limited" && (
        <p className="mb-3 rounded-tv-btn border border-tv-warning/30 bg-tv-warning/10 px-2.5 py-1.5 font-mono text-xs text-tv-warning">
          {error}
        </p>
      )}

      {status === "error" && (
        <p className="mb-3 rounded-tv-btn border border-tv-rose/30 bg-tv-rose/10 px-2.5 py-1.5 font-mono text-xs text-tv-rose">
          {error}
        </p>
      )}

      {nextLevel !== null ? (
        <Button
          variant="outline-cyan"
          size="sm"
          disabled={status === "pending"}
          onClick={() => requestHint(nextLevel)}
        >
          {revealedLevels.length === 0 ? "Get a hint" : `Get hint ${nextLevel}`}
        </Button>
      ) : (
        <p className="font-mono text-xs text-tv-text-body">All hints revealed for this problem.</p>
      )}
    </div>
  );
}
