import { cn } from "@/lib/utils";
import styles from "./scanline-grid.module.css";

export interface ScanlineGridProps {
  className?: string;
}

/**
 * A very low-opacity animated grid — **hero + auth grounds only**
 * (plan.md §1.4; don't reach for this as generic background texture
 * elsewhere). Absolutely positioned and fills its nearest
 * `position: relative` ancestor; ignores pointer events; fades out toward
 * its own edges via a radial mask so it never reads as a hard-edged tile.
 * Render it before your content, or give your content `relative z-10`.
 *
 * Pure CSS `@keyframes` under the hood (no `motion/react`), so
 * `prefers-reduced-motion` is already handled globally — `globals.css`
 * collapses `animation-duration` app-wide. No extra branching needed
 * here, and no client-component boundary either.
 *
 * ```tsx
 * <section className="relative overflow-hidden bg-tv-bg">
 *   <ScanlineGrid />
 *   <div className="relative z-10">...hero content...</div>
 * </section>
 * ```
 */
export function ScanlineGrid({ className }: ScanlineGridProps) {
  return (
    <div
      aria-hidden
      className={cn("pointer-events-none absolute inset-0 overflow-hidden opacity-70", styles.grid, className)}
    />
  );
}
