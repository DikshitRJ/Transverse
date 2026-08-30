"use client";

/**
 * A small local-only preference: "reduce motion in charts", set from
 * `/settings` and consumed by every chart in `components/charts/`. This is
 * in addition to (never a replacement for) the global `prefers-reduced-
 * motion: reduce` handling in `globals.css` — that's out of PRISM's owned
 * paths and already collapses CSS transitions/animations app-wide. This
 * hook only gates the recharts-level `isAnimationActive` prop, which CSS
 * media queries can't reach.
 *
 * There's no backend "preferences" endpoint (see plan.md's route table —
 * none exists), so this is deliberately `localStorage`-backed rather than
 * synced anywhere. A `CustomEvent` keeps every mounted chart on the same
 * page in sync the instant the Settings toggle changes, without needing a
 * shared context provider (which would live in `layout.tsx`, outside
 * PRISM's owned paths).
 */
import { useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "tv.pref.reduceChartMotion";
const EVENT_NAME = "tv:pref:reduceChartMotion";

function readInitial(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

export function useChartMotionPreference(): [boolean, (next: boolean) => void] {
  const [reduceMotion, setReduceMotion] = useState(false);

  useEffect(() => {
    setReduceMotion(readInitial());
    function onChange(e: Event) {
      const detail = (e as CustomEvent<boolean>).detail;
      if (typeof detail === "boolean") setReduceMotion(detail);
    }
    window.addEventListener(EVENT_NAME, onChange);
    return () => window.removeEventListener(EVENT_NAME, onChange);
  }, []);

  const setPreference = useCallback((next: boolean) => {
    setReduceMotion(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, next ? "1" : "0");
    } catch {
      // localStorage unavailable (private mode, storage quota) — preference just won't persist across reloads.
    }
    window.dispatchEvent(new CustomEvent(EVENT_NAME, { detail: next }));
  }, []);

  return [reduceMotion, setPreference];
}
