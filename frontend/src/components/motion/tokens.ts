/**
 * Shared motion vocabulary — durations, easings and keyframes reused by
 * every primitive in this library (and by `components/byte`).
 *
 * Import from here instead of hand-rolling a new timing value. The point of
 * having one `components/motion` package is that every screen built against
 * it reads as one designed product, not eight agents' individual taste —
 * see plan.md §1.4.
 */

/** All durations in milliseconds. */
export const DURATION = {
  /** Cyan sweep traversal across a card edge (hover/focus). */
  sweep: 900,
  /** Glow pulse full breath cycle — plan.md §1.4 fixes this at 2.4s. */
  glowPulseCycle: 2400,
  /** Hard ceiling for any entrance animation — plan.md §1.4. */
  entranceMax: 400,
  /** Terminal type-on speed, ms per character — plan.md §1.4. */
  typeOnCharMs: 28,
  /** Verdict pass: cyan ripple expansion. */
  verdictPassMs: 550,
  /** Verdict fail: single rose shake, no bounce — plan.md §1.4 fixes this at 120ms. */
  verdictFailMs: 120,
  /** Unlock sequence stage durations — ring completes, then flare, then dissolve, then lift. */
  unlockRingMs: 500,
  unlockFlareMs: 260,
  unlockDissolveMs: 240,
  unlockLiftMs: 320,
  /** Page transition — plan.md §1.4 caps this at ≤400ms. */
  pageTransitionMs: 300,
  pageTransitionExitMs: 200,
  /** Byte's one-shot "celebrating" pop (components/byte). */
  byteCelebrateMs: 480,
  /** Byte's gentle idle bob, full cycle. */
  byteIdleBobMs: 3200,
} as const;

export const EASE = {
  /** Fast-out, gentle settle — the app's default "arrival" curve. */
  standard: [0.22, 1, 0.36, 1] as [number, number, number, number],
  /** Sharper, for quick state flips (verdicts, dissolves). */
  sharp: [0.4, 0, 0.2, 1] as [number, number, number, number],
  linear: "linear" as const,
} as const;

/**
 * Single rose shake, no bounce/spring. Shared by `VerdictFeedback`
 * (fail state) and `<Byte state="error">` so a "no" reads identically
 * everywhere it appears.
 */
export const SHAKE_KEYFRAMES_X = [0, -6, 6, -4, 4, 0];

/** Cyan / rose glow color pair used by GlowPulse, ripples and shakes. */
export const GLOW_COLOR = {
  cyan: "0,242,255",
  rose: "255,107,107",
} as const;

export type GlowColor = keyof typeof GLOW_COLOR;
