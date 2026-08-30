"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { ProblemPayload } from "@/lib/api/types";

export interface SolveHeaderProps {
  problem: ProblemPayload | null;
}

export function SolveHeader({ problem }: SolveHeaderProps) {
  return (
    <header className="glass-header flex h-12 shrink-0 items-center gap-3 border-b border-tv-border-nav px-4">
      <Link
        href="/problems"
        className="flex shrink-0 items-center gap-1.5 font-mono text-xs text-tv-text-nav transition-colors hover:text-tv-cyan"
      >
        <ArrowLeft className="size-3.5" aria-hidden />
        Problems
      </Link>
      <span className="text-tv-border">/</span>
      {problem ? (
        <>
          <span className="truncate font-mono text-xs font-medium text-tv-text-hi">{problem.name}</span>
          <Badge variant="outline" className="ml-1 shrink-0 capitalize">
            {problem.difficulty_label}
          </Badge>
        </>
      ) : (
        <span className="font-mono text-xs text-tv-locked">Loading problem...</span>
      )}
    </header>
  );
}
