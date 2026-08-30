"use client";

import { useEffect, useRef, useState } from "react";
import Image from "next/image";
import { motion } from "motion/react";
import { cn } from "@/lib/utils";
import { usePrefersReducedMotion } from "@/components/motion/use-prefers-reduced-motion";
import { GlowPulse } from "@/components/motion/glow-pulse";
import { DURATION, EASE, SHAKE_KEYFRAMES_X } from "@/components/motion/tokens";

export type ByteState = "idle" | "thinking" | "celebrating" | "hinting" | "error";
export type ByteVariant = "chip" | "nav" | "hero";
export type ByteSize = "sm" | "md" | "lg";

export interface ByteProps {
  /** Default `"idle"`. */
  state?: ByteState;
  /** Which mascot asset to render. Default `"chip"` — the small dialogue-avatar crop, the right default for a companion placement. */
  variant?: ByteVariant;
  /** Bounding-box size preset. Default `"md"`. */
  size?: ByteSize;
  /**
   * Bump to replay the `"celebrating"` pop / `"error"` shake again while
   * `state` stays the same value across two occurrences (e.g. two wrong
   * answers in a row). Transitioning *into* a state from a different one
   * always plays; this is only needed for back-to-back repeats.
   */
  playToken?: string | number;
  className?: string;
}

const ASSET: Record<ByteVariant, string> = {
  chip: "/figma/byte-mascot-chip.png",
  nav: "/figma/byte-mascot-nav.png",
  hero: "/figma/byte-mascot-hero.png",
};

const BOX_SIZE: Record<ByteSize, number> = { sm: 32, md: 56, lg: 140 };

const STATE_LABEL: Record<ByteState, string> = {
  idle: "Byte the Beaver",
  thinking: "Byte the Beaver is thinking",
  celebrating: "Byte the Beaver is celebrating",
  hinting: "Byte the Beaver has a hint",
  error: "Byte the Beaver flags a problem",
};

/**
 * Byte the Beaver — the AI tutor mascot, in his five product states. Always
 * renders the real Figma asset (never redrawn as inline SVG). Fully
 * self-contained: no providers, no app-shell dependency — drop it anywhere.
 *
 * State → visual:
 * - `idle` — a slow, gentle vertical bob. The resting state.
 * - `thinking` — mascot holds still; a small three-dot pulse badge appears.
 * - `celebrating` — a one-shot scale pop + a brief cyan ring flash, plus a
 *   soft persistent cyan ring while the state holds. No bounce, no confetti.
 * - `hinting` — wrapped in `GlowPulse` (the one place in a screen that
 *   should be breathing at a time — see `GlowPulse`'s own rationing note).
 * - `error` — a single 120ms rose shake (identical keyframes to
 *   `VerdictFeedback`'s fail state) plus a persistent soft rose ring while
 *   the state holds.
 *
 * Every animated flourish here respects reduced motion; the persistent
 * rings/glow (which are static, not animated) still render either way, so
 * the state is always visually legible even with motion off.
 *
 * ```tsx
 * <Byte state="hinting" size="md" />
 * ```
 */
export function Byte({ state = "idle", variant = "chip", size = "md", playToken, className }: ByteProps) {
  const prefersReducedMotion = usePrefersReducedMotion();
  const box = BOX_SIZE[size];

  const [celebrateKey, setCelebrateKey] = useState(0);
  const [errorKey, setErrorKey] = useState(0);
  const prevCombined = useRef(`${state}:${playToken ?? ""}`);

  useEffect(() => {
    const combined = `${state}:${playToken ?? ""}`;
    if (combined !== prevCombined.current) {
      if (state === "celebrating") setCelebrateKey((k) => k + 1);
      if (state === "error") setErrorKey((k) => k + 1);
      prevCombined.current = combined;
    }
  }, [state, playToken]);

  const figureAnimate =
    state === "celebrating"
      ? prefersReducedMotion
        ? { scale: 1 }
        : { scale: [1, 1.08, 1] }
      : state === "error"
        ? prefersReducedMotion
          ? { x: 0 }
          : { x: SHAKE_KEYFRAMES_X }
        : state === "idle" && !prefersReducedMotion
          ? { y: [0, -3, 0] }
          : { y: 0, x: 0 };

  const figureTransition =
    state === "celebrating"
      ? { duration: DURATION.byteCelebrateMs / 1000, ease: EASE.standard }
      : state === "error"
        ? { duration: DURATION.verdictFailMs / 1000, ease: EASE.sharp }
        : state === "idle"
          ? { duration: DURATION.byteIdleBobMs / 1000, repeat: Infinity, ease: "easeInOut" as const }
          : undefined;

  // Remount key: forces the pop/shake to replay from `initial` even when
  // `state` itself hasn't changed (see `playToken` doc).
  const oneShotKey =
    state === "celebrating" ? `celebrate-${celebrateKey}` : state === "error" ? `error-${errorKey}` : state;

  const figure = (
    <motion.div
      key={oneShotKey}
      className="relative h-full w-full"
      animate={figureAnimate}
      transition={figureTransition}
    >
      <Image
        src={ASSET[variant]}
        alt=""
        width={box}
        height={box}
        className="h-full w-full object-contain"
        priority={size === "lg"}
      />
    </motion.div>
  );

  return (
    <div
      role="img"
      aria-label={STATE_LABEL[state]}
      className={cn("relative inline-flex shrink-0 items-center justify-center", className)}
      style={{ width: box, height: box }}
    >
      {(state === "celebrating" || state === "error") && (
        <span
          aria-hidden
          className={cn(
            "pointer-events-none absolute inset-0 rounded-tv-pill ring-2",
            state === "celebrating" ? "ring-tv-cyan/50" : "ring-tv-rose/60",
          )}
        />
      )}

      {state === "hinting" ? (
        <GlowPulse className="h-full w-full rounded-tv-pill">{figure}</GlowPulse>
      ) : (
        figure
      )}

      {state === "thinking" && <ThinkingDots />}
    </div>
  );
}

function ThinkingDots() {
  const prefersReducedMotion = usePrefersReducedMotion();

  return (
    <span
      aria-hidden
      className="absolute -right-1 -bottom-1 flex items-center gap-0.5 rounded-tv-pill border border-tv-border bg-tv-surface-deep px-1.5 py-1"
    >
      {[0, 1, 2].map((i) => (
        <motion.span
          key={i}
          className="size-1 rounded-full bg-tv-cyan"
          animate={prefersReducedMotion ? undefined : { opacity: [0.25, 1, 0.25] }}
          style={prefersReducedMotion ? { opacity: 0.7 } : undefined}
          transition={
            prefersReducedMotion
              ? undefined
              : { duration: 1.1, repeat: Infinity, ease: "easeInOut", delay: i * 0.15 }
          }
        />
      ))}
    </span>
  );
}
