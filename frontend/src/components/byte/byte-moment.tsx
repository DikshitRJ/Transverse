import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { Byte, type ByteState, type ByteVariant, type ByteSize } from "./byte";

export type ByteMomentVariant =
  | "empty-dashboard"
  | "empty-roadmap"
  | "judge0-failed"
  | "hint-rate-limited"
  | "generic-empty"
  | "generic-error";

export interface ByteMomentProps {
  variant: ByteMomentVariant;
  /** Overrides the variant's default copy. */
  title?: string;
  description?: string;
  /** A retry button, a CTA into onboarding, whatever fits — rendered below the copy. */
  action?: ReactNode;
  className?: string;
}

interface MomentConfig {
  byteState: ByteState;
  byteVariant: ByteVariant;
  byteSize: ByteSize;
  title: string;
  description: string;
}

/**
 * Default Byte state + copy per named moment. Wave-2 screens get a real,
 * human placeholder for free; override `title`/`description`/`action` for
 * anything more specific to the data at hand.
 */
const MOMENT_CONFIG: Record<ByteMomentVariant, MomentConfig> = {
  "empty-dashboard": {
    byteState: "idle",
    byteVariant: "hero",
    byteSize: "lg",
    title: "Nothing to show yet",
    description: "Finish onboarding and I'll fill this in with your roadmap and what to work on next.",
  },
  "empty-roadmap": {
    byteState: "thinking",
    byteVariant: "hero",
    byteSize: "lg",
    title: "No roadmap drawn yet",
    description: "Take the diagnostic quiz and I'll figure out exactly where to start you.",
  },
  "judge0-failed": {
    byteState: "error",
    byteVariant: "chip",
    byteSize: "md",
    title: "That run didn't come back clean",
    description: "The judge choked, not your code. Try running it again in a moment.",
  },
  "hint-rate-limited": {
    byteState: "thinking",
    byteVariant: "chip",
    byteSize: "md",
    title: "Slow down a second",
    description: "You're asking faster than I can think. Try again in a few seconds.",
  },
  "generic-empty": {
    byteState: "idle",
    byteVariant: "hero",
    byteSize: "lg",
    title: "Nothing here yet",
    description: "There's nothing to show right now.",
  },
  "generic-error": {
    byteState: "error",
    byteVariant: "chip",
    byteSize: "md",
    title: "Something went sideways",
    description: "That didn't work. Give it another try.",
  },
};

/**
 * A real, human empty/error moment built around Byte instead of a grey
 * box — plan.md's explicit ask for exactly these four situations: a blank
 * dashboard, a roadmap not yet generated, a failed Judge0 run, a
 * rate-limited hint. Two more generic variants cover anything else that
 * needs the same treatment without a bespoke moment.
 *
 * Byte's state, art size and default copy are chosen per `variant` — pass
 * `title`/`description`/`action` to override any of it, the visual
 * treatment stays.
 *
 * ```tsx
 * <ByteMoment
 *   variant="judge0-failed"
 *   action={<Button onClick={retry}>Try again</Button>}
 * />
 * ```
 */
export function ByteMoment({ variant, title, description, action, className }: ByteMomentProps) {
  const config = MOMENT_CONFIG[variant];

  return (
    <div
      className={cn(
        "flex flex-col items-center gap-4 rounded-tv-card border border-tv-border bg-tv-surface px-6 py-10 text-center",
        className,
      )}
    >
      <Byte state={config.byteState} variant={config.byteVariant} size={config.byteSize} />
      <div className="max-w-sm space-y-1.5">
        <h3 className="font-display text-h3 font-bold text-tv-text-hi">{title ?? config.title}</h3>
        <p className="text-sm text-tv-text-body">{description ?? config.description}</p>
      </div>
      {action}
    </div>
  );
}
