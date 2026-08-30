"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft, BookOpen, Lock, ShieldCheck } from "lucide-react";
import { ApiError, completeRoadmapNode, getRoadmap, testOutRoadmapNode } from "@/lib/api";
import type { ProblemPayload, RoadmapCurrentResponse, Tutorial } from "@/lib/api/types";
import { PageContainer } from "@/components/shell/page-container";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { MasteryRing } from "./mastery-ring";
import { isNodeLocked, STATUS_LABEL, statusBadgeVariant } from "./roadmap-status";

export interface NodeDetailViewProps {
  nodeId: string;
}

export function NodeDetailView({ nodeId }: NodeDetailViewProps) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError, error, refetch } = useQuery<RoadmapCurrentResponse, ApiError>({
    queryKey: ["roadmap"],
    queryFn: () => getRoadmap(),
  });

  const invalidateRoadmap = () => queryClient.invalidateQueries({ queryKey: ["roadmap"] });

  const completeMutation = useMutation({
    mutationFn: () => completeRoadmapNode(nodeId),
    onSuccess: async () => {
      toast.success("Node completed — checking what's unlocked next.");
      await invalidateRoadmap();
    },
    onError: () => toast.error("Couldn't complete this node. Try again."),
  });

  const testOutMutation = useMutation({
    mutationFn: () => testOutRoadmapNode(nodeId),
    onSuccess: async () => {
      toast.success("Tested out — moving you forward.");
      await invalidateRoadmap();
    },
    onError: () => toast.error("Couldn't test out of this node. Try again."),
  });

  if (isLoading) {
    return (
      <Shell>
        <Skeleton className="h-10 w-40" />
        <Skeleton className="h-40 w-full rounded-tv-card" />
        <Skeleton className="h-64 w-full rounded-tv-card" />
      </Shell>
    );
  }

  if (isError) {
    return (
      <Shell>
        <ErrorState
          message={error instanceof ApiError ? error.message : "Something went wrong loading this node."}
          onRetry={() => void refetch()}
        />
      </Shell>
    );
  }

  const subsection = data?.current_section?.subsections.find((s) => s.node_id === nodeId);

  if (!subsection) {
    return (
      <Shell>
        <NotFoundState />
      </Shell>
    );
  }

  if (isNodeLocked(subsection.status)) {
    return (
      <Shell>
        <LockedState title={subsection.title} />
      </Shell>
    );
  }

  const busy = completeMutation.isPending || testOutMutation.isPending;
  const canAct = subsection.status !== "mastered" && subsection.status !== "tested_out";

  return (
    <Shell>
      <Link href="/roadmap" className="flex w-fit items-center gap-1.5 font-mono text-xs text-tv-text-body hover:text-tv-cyan">
        <ArrowLeft className="size-3.5" aria-hidden />
        Back to roadmap
      </Link>

      <header className="flex flex-col gap-5 rounded-tv-card border border-tv-border bg-tv-surface p-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="flex items-center gap-4">
            <MasteryRing score={subsection.mastery_score} status={subsection.status} size={72} />
            <div className="flex flex-col gap-1.5">
              <Badge variant={statusBadgeVariant(subsection.status)}>{STATUS_LABEL[subsection.status]}</Badge>
              <h1 className="font-display text-h1 font-bold tracking-tight text-tv-text-hi uppercase">
                {subsection.title}
              </h1>
              <span className="font-mono text-sm text-tv-text-body">
                Rating {Math.round(subsection.user_rating)} → target {Math.round(subsection.target_rating)}
              </span>
            </div>
          </div>
        </div>

        {canAct && (
          <div className="flex flex-wrap gap-3 border-t border-tv-border pt-5">
            <Button onClick={() => completeMutation.mutate()} disabled={busy}>
              {completeMutation.isPending ? "Completing…" : "Mark Complete"}
            </Button>
            <Button variant="outline-cyan" onClick={() => testOutMutation.mutate()} disabled={busy}>
              <ShieldCheck className="size-4" aria-hidden />
              {testOutMutation.isPending ? "Testing out…" : "Test Out (I already know this)"}
            </Button>
          </div>
        )}
      </header>

      <section className="flex flex-col gap-3">
        <h2 className="font-mono text-xs tracking-wide text-tv-text-body uppercase">
          Tutorials ({subsection.tutorials.length})
        </h2>
        {subsection.tutorials.length === 0 ? (
          <EmptyRow label="No tutorials attached to this node yet." />
        ) : (
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            {subsection.tutorials.map((tutorial) => (
              <TutorialRow key={tutorial.id} tutorial={tutorial} />
            ))}
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="font-mono text-xs tracking-wide text-tv-text-body uppercase">
          Practice questions ({subsection.questions.length})
        </h2>
        {subsection.questions.length === 0 ? (
          <EmptyRow label="No practice questions attached to this node yet." />
        ) : (
          <div className="flex flex-col gap-2">
            {subsection.questions.map((problem) => (
              <ProblemRow key={problem.id} problem={problem} />
            ))}
          </div>
        )}
      </section>
    </Shell>
  );
}

function TutorialRow({ tutorial }: { tutorial: Tutorial }) {
  return (
    <Link
      href={`/tutorial/${tutorial.id}`}
      className="flex items-start gap-3 rounded-tv-card border border-tv-border bg-tv-surface p-4 transition-colors hover:border-tv-border-cyan"
    >
      <BookOpen className="mt-0.5 size-4 shrink-0 text-tv-cyan" aria-hidden />
      <div className="flex flex-1 flex-col gap-1">
        <span className="font-body text-sm font-semibold text-tv-text-hi">{tutorial.title}</span>
        <span className="font-mono text-xs text-tv-text-body">
          {tutorial.type} · {tutorial.difficulty} · {tutorial.estimated_minutes} min
        </span>
      </div>
      <Badge variant={tutorial.status === "COMPLETED" ? "success" : "outline"}>{tutorial.status}</Badge>
    </Link>
  );
}

function ProblemRow({ problem }: { problem: ProblemPayload }) {
  return (
    <Link
      href={`/solve/${problem.id}`}
      className="flex items-center justify-between gap-3 rounded-tv-card border border-tv-border bg-tv-surface px-4 py-3 transition-colors hover:border-tv-border-cyan"
    >
      <div className="flex flex-col gap-0.5">
        <span className="font-body text-sm font-semibold text-tv-text-hi">{problem.name}</span>
        <span className="font-mono text-xs text-tv-text-body">
          {problem.source} · {problem.subtopic}
        </span>
      </div>
      <Badge variant="outline" className="capitalize">
        {problem.difficulty_label}
      </Badge>
    </Link>
  );
}

function EmptyRow({ label }: { label: string }) {
  return (
    <div className="rounded-tv-card border border-dashed border-tv-border px-4 py-6 text-center font-mono text-xs text-tv-text-body">
      {label}
    </div>
  );
}

function LockedState({ title }: { title: string }) {
  return (
    <div className="flex flex-col items-center gap-4 rounded-tv-card border border-tv-border bg-tv-surface px-8 py-16 text-center">
      <Lock className="size-10 text-tv-locked" aria-hidden />
      <h1 className="font-display text-h2 font-bold text-tv-locked uppercase">{title} is locked</h1>
      <p className="max-w-md text-sm text-tv-text-body">
        Master the nodes ahead of this one to unlock it.
      </p>
      <Button variant="outline" render={<Link href="/roadmap" />}>
        Back to roadmap
      </Button>
    </div>
  );
}

function NotFoundState() {
  return (
    <div className="flex flex-col items-center gap-4 rounded-tv-card border border-tv-border bg-tv-surface px-8 py-16 text-center">
      <h1 className="font-display text-h2 font-bold text-tv-text-hi uppercase">Node not found</h1>
      <p className="max-w-md text-sm text-tv-text-body">
        This node isn&apos;t part of your current active section.
      </p>
      <Button variant="outline" render={<Link href="/roadmap" />}>
        Back to roadmap
      </Button>
    </div>
  );
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center gap-4 rounded-tv-card border border-tv-rose/30 bg-tv-surface px-8 py-16 text-center">
      <h1 className="font-display text-h2 font-bold text-tv-text-hi uppercase">Couldn&apos;t load this node</h1>
      <p className="max-w-md text-sm text-tv-text-body">{message}</p>
      <Button variant="outline" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg-page">
      <TopNav />
      <PageContainer className="flex flex-1 flex-col gap-8">{children}</PageContainer>
      <Footer />
    </div>
  );
}
