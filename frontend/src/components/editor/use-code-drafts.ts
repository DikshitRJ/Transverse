"use client";

/**
 * Per-problem, per-language draft persistence (FORGE brief: "Persist draft
 * code per problem in localStorage, wrapped in try/catch" + "Language
 * switch preserves per-language buffers, warns before clobbering edited
 * code"). Each language gets its own independent localStorage slot keyed
 * by problem + language, so switching languages never loses anything —
 * the one real "clobber" risk is the explicit "reset to starter template"
 * action, which callers should confirm with the user before invoking
 * `resetToTemplate` (see `components/ide/editor-panel.tsx`'s reset dialog).
 */
import { useCallback, useEffect, useState } from "react";
import { LANGUAGES, generateTemplates } from "@/mocks/fixtures/languages";
import type { ProblemPayload } from "@/lib/api/types";

const STORAGE_PREFIX = "tv:solve-draft:";
export const DEFAULT_LANGUAGE_KEY = LANGUAGES[0]?.key ?? "py";

function draftKey(problemId: string, languageKey: string): string {
  return `${STORAGE_PREFIX}${problemId}:${languageKey}`;
}

function readDraft(problemId: string, languageKey: string): string | null {
  try {
    return window.localStorage.getItem(draftKey(problemId, languageKey));
  } catch {
    return null;
  }
}

function writeDraft(problemId: string, languageKey: string, code: string): void {
  try {
    window.localStorage.setItem(draftKey(problemId, languageKey), code);
  } catch {
    // localStorage can throw (private browsing, storage disabled, quota
    // exceeded) — draft persistence is a nicety, never something that
    // should break editing.
  }
}

function starterFor(problem: ProblemPayload, languageKey: string): string {
  return (
    problem.templates?.[languageKey] ?? generateTemplates(problem.name, problem.slug)[languageKey] ?? ""
  );
}

export interface UseCodeDraftsResult {
  languageKey: string;
  code: string;
  setCode: (code: string) => void;
  /** Switches the active language. Per-language buffers are independent —
   * this never discards anything. */
  setLanguageKey: (key: string) => void;
  /** True when the current buffer differs from that language's starter
   * template — i.e. there's something a "reset" would actually discard. */
  isDirty: boolean;
  /** Discards the current language's saved draft and reloads its starter
   * template. Callers should confirm with the user first — this is the
   * one destructive action in the draft lifecycle. */
  resetToTemplate: () => void;
}

export function useCodeDrafts(
  problem: ProblemPayload | undefined,
  problemId: string,
): UseCodeDraftsResult {
  const [languageKey, setLanguageKeyState] = useState<string>(DEFAULT_LANGUAGE_KEY);
  const [code, setCodeState] = useState<string>("");

  // (Re)load the active language's buffer whenever the problem becomes
  // available or the language changes — saved draft if one exists,
  // otherwise that language's starter template.
  useEffect(() => {
    if (!problem) return;
    const saved = readDraft(problemId, languageKey);
    setCodeState(saved ?? starterFor(problem, languageKey));
  }, [problem, problemId, languageKey]);

  const setCode = useCallback(
    (next: string) => {
      setCodeState(next);
      writeDraft(problemId, languageKey, next);
    },
    [problemId, languageKey],
  );

  const setLanguageKey = useCallback((key: string) => {
    setLanguageKeyState(key);
  }, []);

  const resetToTemplate = useCallback(() => {
    if (!problem) return;
    const template = starterFor(problem, languageKey);
    setCodeState(template);
    writeDraft(problemId, languageKey, template);
  }, [problem, problemId, languageKey]);

  const isDirty = problem ? code !== starterFor(problem, languageKey) : false;

  return { languageKey, code, setCode, setLanguageKey, isDirty, resetToTemplate };
}
