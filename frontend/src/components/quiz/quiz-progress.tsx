"use client";

import { Progress } from "@/components/ui/progress";
import { cn } from "@/lib/utils";

export interface QuizProgressProps {
  questionCount: number;
  cap: number;
  className?: string;
}

export function QuizProgress({ questionCount, cap, className }: QuizProgressProps) {
  const current = Math.min(questionCount + 1, cap);
  const pct = (Math.min(questionCount, cap) / cap) * 100;
  return (
    <div className={cn("flex items-center gap-3", className)}>
      <span className="shrink-0 font-mono text-xs text-tv-text-body uppercase tracking-wide">
        Question {current} / {cap}
      </span>
      <Progress value={pct} className="flex-1" />
    </div>
  );
}
