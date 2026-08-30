"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { findProblemById } from "./find-problem";
import { SolveHeader } from "./solve-header";
import { ProblemPanel } from "./problem-panel";
import { EditorPanel } from "./editor-panel";
import { useMediaQuery } from "./use-media-query";

export interface SolveWorkspaceProps {
  problemId: string;
  /** Practice session this solve is scored against, if any (see the
   * FORGE report: `/solve/[problemId]` can be reached standalone — from
   * the problem browser or a roadmap tutorial link, with no active
   * session — in which case Run still works but Submit is disabled with
   * an explanation rather than silently doing nothing). */
  sessionId?: string;
}

function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <div className="max-w-sm rounded-tv-card border border-tv-rose/30 bg-tv-rose/5 p-6 text-center glow-card-rose">
        <AlertTriangle className="mx-auto mb-2 size-6 text-tv-rose" aria-hidden />
        <p className="mb-3 text-sm text-tv-rose">{message}</p>
        {onRetry && (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            Try again
          </Button>
        )}
      </div>
    </div>
  );
}

function WorkspaceSkeleton() {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="w-full max-w-md space-y-3 p-6" aria-hidden>
        <Skeleton className="h-6 w-2/3" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-4 w-1/2" />
      </div>
    </div>
  );
}

export function SolveWorkspace({ problemId, sessionId }: SolveWorkspaceProps) {
  const {
    data: problem,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["problem", problemId],
    queryFn: () => findProblemById(problemId),
  });

  const isDesktop = useMediaQuery("(min-width: 1024px)");
  const [mobileTab, setMobileTab] = useState("problem");

  return (
    <div className="flex h-screen min-h-0 flex-col bg-tv-bg">
      <SolveHeader problem={problem ?? null} />

      <div className="min-h-0 flex-1">
        {isLoading && <WorkspaceSkeleton />}

        {!isLoading && isError && (
          <ErrorState
            message={error instanceof Error ? error.message : "Couldn't load this problem."}
            onRetry={() => void refetch()}
          />
        )}

        {!isLoading && !isError && problem === null && (
          <ErrorState message={`No problem found for id "${problemId}".`} />
        )}

        {!isLoading && !isError && problem
          ? (isDesktop ? (
              <div className="grid h-full min-h-0 grid-cols-2 divide-x divide-tv-border">
                <ProblemPanel problem={problem} sessionId={sessionId} />
                <EditorPanel problem={problem} sessionId={sessionId} />
              </div>
            ) : (
              <Tabs
                value={mobileTab}
                onValueChange={(next: string) => setMobileTab(next)}
                className="flex h-full min-h-0 flex-col"
              >
                <TabsList className="mx-3 mt-2 w-fit shrink-0">
                  <TabsTrigger value="problem">Problem</TabsTrigger>
                  <TabsTrigger value="editor">Editor</TabsTrigger>
                </TabsList>
                <TabsContent value="problem" className="min-h-0 flex-1 overflow-y-auto">
                  <ProblemPanel problem={problem} sessionId={sessionId} />
                </TabsContent>
                <TabsContent value="editor" className="min-h-0 flex-1 overflow-hidden">
                  <EditorPanel problem={problem} sessionId={sessionId} />
                </TabsContent>
              </Tabs>
            ))
          : null}
      </div>
    </div>
  );
}
