import { Skeleton } from "@/components/ui/skeleton";

/**
 * Deliberately its own file, separate from `monaco-editor.tsx`. That file
 * imports the real `monaco-editor` package at module scope (needed for
 * `loader.config({ monaco })`), which touches `window` as a side effect of
 * import — fine under `next/dynamic(..., { ssr: false })`, but only if
 * *nothing* statically imports `monaco-editor.tsx` outside that dynamic()
 * call. This file exists so `editor-panel.tsx` can render a loading
 * fallback (both as `dynamic()`'s `loading` option and passed straight
 * through) without ever triggering a static import of the Monaco module,
 * which previously broke SSR with `ReferenceError: window is not defined`
 * (monaco-editor's `browser.js` references `window` at import time).
 */
export function CodeEditorSkeleton() {
  return (
    <div className="flex h-full w-full flex-col gap-2.5 bg-tv-bg p-4" aria-hidden>
      <Skeleton className="h-3.5 w-2/3 bg-tv-surface-2" />
      <Skeleton className="h-3.5 w-1/2 bg-tv-surface-2" />
      <Skeleton className="h-3.5 w-5/6 bg-tv-surface-2" />
      <Skeleton className="h-3.5 w-1/3 bg-tv-surface-2" />
      <Skeleton className="h-3.5 w-3/5 bg-tv-surface-2" />
    </div>
  );
}
