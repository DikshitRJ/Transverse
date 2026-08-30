/**
 * Verdict feedback animations (plan.md §1.4 "Verdict": pass = cyan ripple
 * out from the test row; fail = single 120ms rose shake, no bounce).
 *
 * Kept local and self-contained rather than reaching into a shared motion
 * library — BYTE owns `components/motion/**`, but that path doesn't exist
 * in this worktree (each Wave-1 agent works in an isolated git worktree;
 * BYTE's output lands at merge time, not before). Per the FORGE brief
 * ("keep animations local to your components... don't wait for it, don't
 * duplicate it") these two keyframes are intentionally tiny and easy to
 * delete in favor of BYTE's equivalents once merged.
 *
 * Pure CSS `@keyframes`, triggered by mounting a fresh row (see the `key`
 * usage in `results-panel.tsx`/`submit-panel.tsx` — a new key means a new
 * DOM node, and a CSS `animation` on a freshly-mounted node plays
 * automatically) rather than JS-driven — `globals.css` already collapses
 * every `animation-duration` to ~0.01ms under `prefers-reduced-motion:
 * reduce`, so no extra guard is needed here.
 */
export function VerdictFxStyles() {
  return (
    <style>{`
      @keyframes tv-verdict-ripple {
        0% { box-shadow: 0 0 0 0 rgba(0, 242, 255, 0.5); }
        100% { box-shadow: 0 0 0 14px rgba(0, 242, 255, 0); }
      }
      @keyframes tv-verdict-shake {
        0% { transform: translateX(0); }
        20% { transform: translateX(-4px); }
        45% { transform: translateX(4px); }
        70% { transform: translateX(-2px); }
        100% { transform: translateX(0); }
      }
      .tv-anim-pass { animation: tv-verdict-ripple 550ms ease-out; }
      .tv-anim-fail { animation: tv-verdict-shake 120ms linear; }
    `}</style>
  );
}

export function verdictFxClass(passed: boolean): string {
  return passed ? "tv-anim-pass" : "tv-anim-fail";
}
