/**
 * Difficulty severity ramp — reuses the Badge's verdict/status variants
 * (`success`/`warning`/`error`) rather than inventing a new color meaning,
 * per FOUNDATION.md's semantic extension. Always paired with the visible
 * difficulty text on the badge itself, never color alone.
 */
export const DIFFICULTY_BADGE_VARIANT: Record<string, "success" | "warning" | "error"> = {
  easy: "success",
  medium: "warning",
  hard: "error",
  expert: "error",
};
