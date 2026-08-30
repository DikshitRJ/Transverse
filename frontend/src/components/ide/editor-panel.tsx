"use client";

import dynamic from "next/dynamic";
import { useEffect, useMemo, useRef, useState } from "react";
import { RotateCcw, Play, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogClose,
} from "@/components/ui/dialog";
import { CodeEditorSkeleton } from "@/components/editor/code-editor-skeleton";
import { LanguageSwitcher } from "@/components/editor/language-switcher";
import { findLanguageMeta } from "@/components/editor/language-meta";
import { useCodeDrafts } from "@/components/editor/use-code-drafts";
import { executeBatch } from "@/lib/api/endpoints";
import type { BatchExecuteResponse, ProblemPayload } from "@/lib/api/types";
import { ResultsPanel, type RunStatus } from "./results-panel";
import { SubmitPanel } from "./submit-panel";
import { useSubmitFlow } from "./use-submit-flow";

const CodeEditor = dynamic(
  () => import("@/components/editor/monaco-editor").then((mod) => mod.CodeEditor),
  { ssr: false, loading: () => <CodeEditorSkeleton /> },
);

function useElapsedWhile(active: boolean): number {
  const [elapsed, setElapsed] = useState(0);
  const startRef = useRef(0);

  useEffect(() => {
    if (!active) return;
    startRef.current = Date.now();
    setElapsed(0);
    const id = setInterval(() => setElapsed(Date.now() - startRef.current), 100);
    return () => clearInterval(id);
  }, [active]);

  return elapsed;
}

export interface EditorPanelProps {
  problem: ProblemPayload;
  sessionId?: string;
}

export function EditorPanel({ problem, sessionId }: EditorPanelProps) {
  const { languageKey, code, setCode, setLanguageKey, isDirty, resetToTemplate } = useCodeDrafts(
    problem,
    problem.id,
  );
  const language = findLanguageMeta(languageKey);

  const sampleCases = useMemo(
    () => (problem.test_cases ?? []).filter((tc) => !tc.is_hidden),
    [problem.test_cases],
  );

  const [runStatus, setRunStatus] = useState<RunStatus>("idle");
  const [runResult, setRunResult] = useState<BatchExecuteResponse | null>(null);
  const [runError, setRunError] = useState<string | null>(null);
  const [runVersion, setRunVersion] = useState(0);
  const runElapsedMs = useElapsedWhile(runStatus === "running");

  const [resultTab, setResultTab] = useState<string>("run");
  const [resetDialogOpen, setResetDialogOpen] = useState(false);
  const [customStdin, setCustomStdin] = useState("");

  const submitFlow = useSubmitFlow();
  const mountedAtRef = useRef<number>(Date.now());
  useEffect(() => {
    mountedAtRef.current = Date.now();
  }, [problem.id]);

  async function handleRun() {
    if (sampleCases.length === 0) {
      setRunStatus("error");
      setRunError("This problem has no sample test cases to run against.");
      return;
    }
    setRunStatus("running");
    setRunError(null);
    try {
      const result = await executeBatch({
        problem_id: problem.id,
        language_id: language.judge0Id,
        source_code: code,
        test_cases: sampleCases,
      });
      setRunResult(result);
      setRunStatus("done");
      setRunVersion((v) => v + 1);
    } catch (err) {
      setRunError(err instanceof Error ? err.message : "Run failed.");
      setRunStatus("error");
    }
  }

  async function handleSubmit() {
    if (!sessionId) return;
    setResultTab("submit");
    await submitFlow.run({
      sessionId,
      problemId: problem.id,
      languageId: language.judge0Id,
      sourceCode: code,
      customStdin: customStdin.trim() ? customStdin : undefined,
      timeTakenMs: Date.now() - mountedAtRef.current,
    });
  }

  const isSubmitBusy =
    submitFlow.state.phase === "queued" ||
    submitFlow.state.phase === "processing" ||
    submitFlow.state.phase === "submitting";

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-tv-border bg-tv-surface px-3 py-2">
        <LanguageSwitcher value={languageKey} onChange={setLanguageKey} disabled={runStatus === "running" || isSubmitBusy} />

        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="gap-1.5"
          disabled={!isDirty}
          onClick={() => setResetDialogOpen(true)}
        >
          <RotateCcw className="size-3.5" aria-hidden />
          Reset
        </Button>

        <div className="ml-auto flex items-center gap-2">
          <Button
            type="button"
            variant="outline-cyan"
            size="sm"
            className="gap-1.5"
            disabled={runStatus === "running"}
            onClick={handleRun}
          >
            <Play className="size-3.5" aria-hidden />
            Run
          </Button>
          <Button
            type="button"
            size="sm"
            className="gap-1.5"
            disabled={!sessionId || isSubmitBusy}
            title={sessionId ? undefined : "Start a practice session to submit"}
            onClick={handleSubmit}
          >
            <Send className="size-3.5" aria-hidden />
            Submit
          </Button>
        </div>
      </div>

      <Dialog open={resetDialogOpen} onOpenChange={setResetDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reset to starter template?</DialogTitle>
            <DialogDescription>
              This discards your edited {language.label} code for this problem and reloads the starter
              template. This can&apos;t be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
            <Button
              type="button"
              variant="destructive"
              onClick={() => {
                resetToTemplate();
                setResetDialogOpen(false);
              }}
            >
              Reset code
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <div className="min-h-[280px] flex-1 border-b border-tv-border">
        <CodeEditor value={code} monacoLanguage={language.monacoId} onChange={setCode} />
      </div>

      <div className="flex h-[38%] min-h-[220px] shrink-0 flex-col overflow-hidden">
        <Tabs
          value={resultTab}
          onValueChange={(next: string) => setResultTab(next)}
          className="flex h-full min-h-0 flex-col"
        >
          <TabsList className="mx-3 mt-2 w-fit shrink-0">
            <TabsTrigger value="run">Run</TabsTrigger>
            <TabsTrigger value="submit">Submit</TabsTrigger>
          </TabsList>

          <TabsContent value="run" className="min-h-0 flex-1 overflow-y-auto p-3">
            <ResultsPanel
              status={runStatus}
              result={runResult}
              error={runError}
              elapsedMs={runElapsedMs}
              sampleCount={sampleCases.length}
              runVersion={runVersion}
            />
          </TabsContent>

          <TabsContent value="submit" className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-3">
            <SubmitPanel state={submitFlow.state} hasSession={Boolean(sessionId)} />
            {submitFlow.state.phase === "idle" && (
              <div>
                <label className="mb-1 block font-mono text-[11px] text-tv-text-body/70" htmlFor="custom-stdin">
                  Custom stdin (optional)
                </label>
                <Textarea
                  id="custom-stdin"
                  value={customStdin}
                  onChange={(e) => setCustomStdin(e.target.value)}
                  placeholder="Leave blank to run against the problem's own input"
                  className="min-h-16 border-tv-border bg-tv-surface-deep font-mono text-xs text-tv-text-hi"
                />
              </div>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
