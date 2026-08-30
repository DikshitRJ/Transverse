"use client";

import { useCallback, useState } from "react";

export interface HoverActiveHandlers {
  onMouseEnter: () => void;
  onMouseLeave: () => void;
  onFocus: () => void;
  onBlur: () => void;
}

export interface UseHoverActiveResult {
  /** True while the bound element is hovered OR focused. */
  active: boolean;
  /** Spread onto the element you want to watch: `<div {...bind}>`. */
  bind: HoverActiveHandlers;
}

/**
 * Tracks "hover or focus" as one boolean, for driving primitives like
 * `CyanSweep` off a real element instead of relying on Tailwind's
 * `group-hover`/`group-focus-within` (which those primitives don't assume
 * you're using). Hover and focus are tracked independently and OR'd, so
 * tabbing to an element keeps the same "alive" affordance a mouse hover
 * gets.
 *
 * ```tsx
 * const { active, bind } = useHoverActive();
 * <div {...bind} className="relative">
 *   <CyanSweep active={active} />
 *   ...card content...
 * </div>
 * ```
 */
export function useHoverActive(): UseHoverActiveResult {
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);

  const onMouseEnter = useCallback(() => setHovered(true), []);
  const onMouseLeave = useCallback(() => setHovered(false), []);
  const onFocus = useCallback(() => setFocused(true), []);
  const onBlur = useCallback(() => setFocused(false), []);

  return {
    active: hovered || focused,
    bind: { onMouseEnter, onMouseLeave, onFocus, onBlur },
  };
}
