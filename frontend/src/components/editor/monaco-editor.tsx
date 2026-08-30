"use client";

/**
 * Thin wrapper around `@monaco-editor/react`'s `<Editor>`. This module
 * touches browser globals at import time (`monaco-editor` itself, and
 * `@monaco-editor/react`'s loader) — it must only ever be reached through
 * `next/dynamic(..., { ssr: false })` from the consumer (see
 * `src/components/ide/editor-panel.tsx`), never imported statically from a
 * module that might render on the server.
 *
 * `monaco-editor` is bundled locally (via `loader.config({ monaco })`)
 * rather than fetched from Monaco's CDN default — this repo already lists
 * `monaco-editor` as a direct dependency specifically so the editor works
 * offline / against the mock backend with no third-party network
 * dependency. See the FORGE report for the one open risk this trades in
 * (no `MonacoWebpackPlugin` wired into `next.config.ts`, which is outside
 * FORGE's owned paths — language web workers fall back to running
 * synchronously on the main thread, which Monaco itself does gracefully,
 * at a minor cost to intellisense responsiveness on very large files).
 */
import Editor, { loader, type BeforeMount, type OnChange } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
import { TRANSVERSE_MONACO_THEME, TRANSVERSE_THEME_NAME } from "./monaco-theme";
import { CodeEditorSkeleton } from "./code-editor-skeleton";

loader.config({ monaco });

export interface CodeEditorProps {
  value: string;
  /** Monaco's own language id (`LanguageMeta.monacoId`), not the Judge0/backend slug. */
  monacoLanguage: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  onMountReady?: () => void;
}

export function CodeEditor({
  value,
  monacoLanguage,
  onChange,
  readOnly = false,
  onMountReady,
}: CodeEditorProps) {
  const handleBeforeMount: BeforeMount = (m) => {
    m.editor.defineTheme(TRANSVERSE_THEME_NAME, TRANSVERSE_MONACO_THEME);
  };

  const handleChange: OnChange = (nextValue) => {
    onChange(nextValue ?? "");
  };

  return (
    <Editor
      height="100%"
      language={monacoLanguage}
      theme={TRANSVERSE_THEME_NAME}
      value={value}
      onChange={handleChange}
      beforeMount={handleBeforeMount}
      onMount={onMountReady}
      loading={<CodeEditorSkeleton />}
      options={{
        fontFamily: "var(--font-jetbrains-mono), 'JetBrains Mono', ui-monospace, monospace",
        fontSize: 13,
        lineHeight: 20,
        minimap: { enabled: false },
        readOnly,
        scrollBeyondLastLine: false,
        automaticLayout: true,
        tabSize: 4,
        insertSpaces: true,
        padding: { top: 14, bottom: 14 },
        renderLineHighlight: "all",
        smoothScrolling: true,
        cursorBlinking: "smooth",
        fontLigatures: true,
        scrollbar: { verticalScrollbarSize: 10, horizontalScrollbarSize: 10 },
      }}
    />
  );
}
