"use client";

import dynamic from "next/dynamic";
import { useMemo, useState } from "react";
import { RotateCcw, Code2, Terminal, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CodeEditorSkeleton } from "@/components/editor/code-editor-skeleton";
import type { LanguageMeta } from "@/mocks/fixtures/languages";
import type { ProblemPayload } from "@/lib/api/types";
import { cn } from "@/lib/utils";

const CodeEditor = dynamic(
  () => import("@/components/editor/monaco-editor").then((mod) => mod.CodeEditor),
  { ssr: false, loading: () => <CodeEditorSkeleton /> },
);

export interface AnswerSurfaceProps {
  languages: LanguageMeta[];
  language: LanguageMeta;
  onLanguageChange: (language: LanguageMeta) => void;
  code: string;
  onCodeChange: (code: string) => void;
  disabled?: boolean;
  className?: string;
  problem?: ProblemPayload | null;
  onReset?: () => void;
}

/**
 * Full LeetCode-style code editor interface for quiz and practice sessions.
 * Features Monaco Editor, multi-language switcher, reset template controls,
 * and an interactive testcase console.
 */
export function AnswerSurface({
  languages,
  language,
  onLanguageChange,
  code,
  onCodeChange,
  disabled,
  className,
  problem,
  onReset,
}: AnswerSurfaceProps) {
  const sampleCases = useMemo(
    () => (problem?.test_cases ?? []).filter((tc) => !tc.is_hidden),
    [problem?.test_cases],
  );

  const [activeCaseTab, setActiveCaseTab] = useState<string>("case-0");
  const lineCount = useMemo(() => (code ? code.split("\n").length : 1), [code]);

  const handleReset = () => {
    if (onReset) {
      onReset();
      return;
    }
    // Default reset to problem template if available
    if (problem?.templates && problem.templates[language.key]) {
      onCodeChange(problem.templates[language.key]);
    }
  };

  return (
    <div className={cn("flex flex-col rounded-tv-card border border-tv-border bg-tv-surface-deep/90 shadow-md overflow-hidden", className)}>
      {/* Editor Header Toolbar (LeetCode style) */}
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-tv-border bg-tv-surface px-3 py-2">
        <div className="flex items-center gap-2.5">
          <div className="flex items-center gap-1.5 font-mono text-xs text-tv-cyan font-semibold">
            <Code2 className="size-4 text-tv-cyan" aria-hidden />
            <span>Code</span>
          </div>

          <div className="h-4 w-px bg-tv-border mx-1" />

          <Select
            value={language.key}
            onValueChange={(key) => {
              const next = languages.find((l) => l.key === key);
              if (next) onLanguageChange(next);
            }}
            disabled={disabled}
          >
            <SelectTrigger
              id="answer-language"
              size="sm"
              className="h-7 w-36 border-tv-border/80 bg-tv-surface-deep font-mono text-xs text-tv-text-hi focus:ring-tv-cyan/30"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="border-tv-border bg-tv-surface-deep font-mono text-xs">
              {languages.map((l) => (
                <SelectItem key={l.key} value={l.key} className="cursor-pointer">
                  {l.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-2">
          <span className="font-mono text-[11px] text-tv-text-muted hidden sm:inline-block">
            {lineCount} {lineCount === 1 ? "line" : "lines"}
          </span>

          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 gap-1 px-2 font-mono text-xs text-tv-text-muted hover:text-tv-text-hi"
            disabled={disabled}
            onClick={handleReset}
            title="Reset code to initial starter template"
          >
            <RotateCcw className="size-3.5" aria-hidden />
            <span className="hidden sm:inline">Reset</span>
          </Button>
        </div>
      </div>

      {/* Monaco Code Editor Workspace */}
      <div className="h-[340px] min-h-[260px] w-full border-b border-tv-border bg-[#0a0e14]">
        <CodeEditor
          value={code}
          monacoLanguage={language.monacoId}
          onChange={onCodeChange}
          readOnly={disabled}
        />
      </div>

      {/* LeetCode-style Testcase Console */}
      {sampleCases.length > 0 && (
        <div className="flex flex-col bg-tv-surface-deep/95 p-3">
          <div className="mb-2 flex items-center justify-between">
            <div className="flex items-center gap-1.5 font-mono text-xs font-semibold text-tv-text-hi">
              <Terminal className="size-3.5 text-tv-cyan" aria-hidden />
              <span>Test Cases</span>
            </div>
            <span className="font-mono text-[10px] text-tv-text-muted">
              {sampleCases.length} sample {sampleCases.length === 1 ? "case" : "cases"}
            </span>
          </div>

          <Tabs
            value={activeCaseTab}
            onValueChange={setActiveCaseTab}
            className="flex flex-col gap-2.5"
          >
            <TabsList className="h-7 w-fit bg-tv-surface p-0.5">
              {sampleCases.map((_, idx) => (
                <TabsTrigger
                  key={idx}
                  value={`case-${idx}`}
                  className="h-6 px-2.5 font-mono text-xs data-[state=active]:bg-tv-cyan/20 data-[state=active]:text-tv-cyan"
                >
                  <CheckCircle2 className="mr-1 size-3 opacity-60" />
                  Case {idx + 1}
                </TabsTrigger>
              ))}
            </TabsList>

            {sampleCases.map((tc, idx) => (
              <TabsContent key={idx} value={`case-${idx}`} className="mt-0 space-y-2">
                <div className="grid grid-cols-1 gap-2.5 font-mono text-xs sm:grid-cols-2">
                  <div className="space-y-1">
                    <div className="text-[11px] font-semibold text-tv-text-muted uppercase tracking-wider">
                      Input
                    </div>
                    <pre className="max-h-24 overflow-auto rounded border border-tv-border bg-tv-surface-2 p-2 text-tv-text-hi whitespace-pre-wrap">
                      {tc.input}
                    </pre>
                  </div>
                  <div className="space-y-1">
                    <div className="text-[11px] font-semibold text-tv-text-muted uppercase tracking-wider">
                      Expected Output
                    </div>
                    <pre className="max-h-24 overflow-auto rounded border border-tv-border bg-tv-surface-2 p-2 text-tv-cyan whitespace-pre-wrap">
                      {tc.output}
                    </pre>
                  </div>
                </div>
                {tc.explanation && (
                  <p className="font-sans text-xs text-tv-text-muted italic">
                    Note: {tc.explanation}
                  </p>
                )}
              </TabsContent>
            ))}
          </Tabs>
        </div>
      )}
    </div>
  );
}
