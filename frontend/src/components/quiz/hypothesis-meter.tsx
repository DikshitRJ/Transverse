"use client";

import { Badge } from "@/components/ui/badge";
import { ThetaGauge } from "@/components/practice/theta-gauge";
import { topicLabel } from "@/components/practice/format";
import { cn } from "@/lib/utils";
import type { TopicTally } from "./use-quiz-session";

export interface HypothesisMeterProps {
  theta: number;
  topics: string[];
  tally: Record<string, TopicTally>;
  className?: string;
}

/**
 * The "hypothesis -> verification" loop, made visible while the quiz runs
 * (plan.md's pitch language, PULSE's brief): every seed topic starts as an
 * untested hypothesis chip; as questions land, each one flips to
 * confirmed (majority correct) or debunked (majority incorrect) in real
 * time, alongside a live theta reading. This is the product's core USP
 * surfaced mid-quiz, not just on the results screen.
 */
export function HypothesisMeter({ theta, topics, tally, className }: HypothesisMeterProps) {
  return (
    <div className={cn("glass-panel rounded-tv-card p-4", className)}>
      <ThetaGauge theta={theta} label="Live reading" />
      <div className="mt-4 flex flex-wrap gap-1.5">
        {topics.map((topic) => {
          const entry = tally[topic];
          if (!entry || entry.attempts === 0) {
            return (
              <Badge key={topic} variant="locked" className="rounded-tv-pill">
                {topicLabel(topic)}
              </Badge>
            );
          }
          const confirmed = entry.correct / entry.attempts >= 0.5;
          return (
            <Badge
              key={topic}
              variant={confirmed ? "success" : "error"}
              className="rounded-tv-pill"
            >
              {confirmed ? "✓" : "✕"} {topicLabel(topic)}
            </Badge>
          );
        })}
      </div>
    </div>
  );
}
