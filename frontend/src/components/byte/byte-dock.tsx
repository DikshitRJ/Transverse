"use client";

import { useEffect, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "@/components/motion/use-prefers-reduced-motion";
import { Byte, type ByteState } from "./byte";
import { ByteSpeech } from "./byte-speech";

export type ByteDockPosition = "bottom-right" | "bottom-left";

export interface ByteDockProps {
  state?: ByteState;
  /** When present, shows the speech bubble. Changing it retypes the new message. */
  message?: string;
  label?: string;
  /** Default `"bottom-right"`. */
  position?: ByteDockPosition;
  /** Controlled bubble visibility. Omit to let `ByteDock` manage it itself. */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  className?: string;
}

const POSITION_CLASS: Record<ByteDockPosition, string> = {
  "bottom-right": "right-6 bottom-6 items-end",
  "bottom-left": "left-6 bottom-6 items-start",
};

/**
 * The persistent companion placement — a fixed-position dock for the app
 * shell. Mount it once near your root layout (or any authenticated
 * layout); it owns its own `position: fixed` box and `z-50`, so it doesn't
 * need a slot from you.
 *
 * Uncontrolled by default: set `message` and the bubble appears (opens
 * automatically whenever `message` changes to a new value); the viewer can
 * dismiss it via the bubble's close button or by clicking Byte himself.
 * Pass `open`/`onOpenChange` if you want to drive visibility yourself
 * (e.g. tie it to an SSE `hint.ready` event, or re-open it on click after
 * dismissal).
 *
 * ```tsx
 * // app/(app)/layout.tsx (or wherever your authenticated shell lives)
 * <ByteDock state="hinting" message='Stuck? Try breaking the loop invariant down first.' />
 * ```
 */
export function ByteDock({
  state = "idle",
  message,
  label,
  position = "bottom-right",
  open: openProp,
  onOpenChange,
  className,
}: ByteDockProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const [openState, setOpenState] = useState(Boolean(message));
  const open = openProp ?? openState;

  useEffect(() => {
    if (openProp === undefined) setOpenState(Boolean(message));
    // Only re-open on a genuinely new `message`; dismissal is a separate action.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [message]);

  const setOpen = (next: boolean) => {
    onOpenChange?.(next);
    if (openProp === undefined) setOpenState(next);
  };

  return (
    <div className={cn("pointer-events-none fixed z-50 flex flex-col gap-2", POSITION_CLASS[position], className)}>
      <AnimatePresence>
        {open && message && (
          <motion.div
            key="byte-dock-bubble"
            className="pointer-events-auto relative"
            initial={prefersReducedMotion ? false : { opacity: 0, y: 8, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={prefersReducedMotion ? undefined : { opacity: 0, y: 8, scale: 0.98 }}
            transition={{ duration: prefersReducedMotion ? 0 : 0.2 }}
          >
            <ByteSpeech message={message} label={label} />
            <button
              type="button"
              onClick={() => setOpen(false)}
              aria-label="Dismiss"
              className="absolute -top-2 -right-2 flex size-5 items-center justify-center rounded-tv-pill border border-tv-border bg-tv-surface text-tv-text-body hover:text-tv-text-hi"
            >
              <X className="size-3" />
            </button>
          </motion.div>
        )}
      </AnimatePresence>

      <button
        type="button"
        onClick={() => setOpen(!open)}
        aria-label={open && message ? "Hide Byte" : "Show Byte"}
        className="pointer-events-auto rounded-tv-pill focus-visible:outline focus-visible:outline-2 focus-visible:outline-tv-cyan"
      >
        <Byte state={state} variant="chip" size="md" />
      </button>
    </div>
  );
}
