"use client";

import { cn } from "@/lib/utils";
import { TerminalType } from "@/components/motion/terminal-type";

export interface ByteSpeechProps {
  /** Default `"Transverse says:"` — matches Figma `15:10`. */
  label?: string;
  message: string;
  /** Types the message in at terminal speed. Default `true`; set `false` for copy that should just be present (e.g. re-rendering the same message after a re-mount). */
  typeOn?: boolean;
  className?: string;
}

/**
 * Byte's dialogue pill — matches the Figma `15:10` mascot-mentor bubble
 * exactly: cyan JetBrains Mono label, body in `--tv-text-body`, rounded
 * pill on the `--tv-surface-deep` well. Body text types in via
 * `TerminalType`.
 *
 * Pairs with `<Byte />` but doesn't render him — compose them yourself
 * (`ByteDock` does this for the persistent-companion case):
 *
 * ```tsx
 * <div className="flex items-start gap-3">
 *   <Byte state="hinting" size="sm" />
 *   <ByteSpeech message='Not sure? The quiz is a fun way to warm up!' />
 * </div>
 * ```
 */
export function ByteSpeech({ label = "Transverse says:", message, typeOn = true, className }: ByteSpeechProps) {
  return (
    <div
      className={cn(
        "glass-panel max-w-md rounded-tv-pill border-tv-border bg-tv-surface-deep px-4 py-3",
        className,
      )}
    >
      <p className="font-mono text-xs font-semibold text-tv-cyan">{label}</p>
      {typeOn ? (
        <TerminalType
          as="p"
          text={message}
          className="mt-1 text-sm leading-snug font-normal text-tv-text-body"
        />
      ) : (
        <p className="mt-1 font-mono text-sm leading-snug text-tv-text-body">{message}</p>
      )}
    </div>
  );
}
