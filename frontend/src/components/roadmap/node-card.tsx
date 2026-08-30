"use client";

import Link from "next/link";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { Lock, CheckCircle2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { MasteryRing } from "./mastery-ring";
import { UnlockFlare } from "./unlock-flare";
import { STATUS_LABEL, isNodeAlive, isNodeLocked, isNodeSettled, statusBadgeVariant } from "./roadmap-status";
import type { RoadmapSubsection } from "@/lib/api/types";

export interface NodeCardProps {
  subsection: RoadmapSubsection;
  /** This node just reached mastered/tested-out this render — plays ring-complete + glow-flare. */
  justMastered?: boolean;
  /** This node just left "locked" this render — plays the lock-dissolve beat. */
  justUnlocked?: boolean;
}

/**
 * A single roadmap node. Three registers, per the ATLAS brief: the active
 * node should feel alive (glow-pulse), mastered ones settled (static cyan,
 * checkmark), locked ones inert (dimmed, unclickable, lock icon) — and the
 * unlock moment is a local, choreographed transition between the last two,
 * not a jump cut.
 */
export function NodeCard({ subsection, justMastered = false, justUnlocked = false }: NodeCardProps) {
  const reduceMotion = useReducedMotion();
  const locked = isNodeLocked(subsection.status) && !justUnlocked;
  const alive = isNodeAlive(subsection.status);
  const settled = isNodeSettled(subsection.status);

  const card = (
    <motion.div
      data-testid={`node-card-${subsection.node_id}`}
      data-status={subsection.status}
      data-just-unlocked={justUnlocked ? "true" : undefined}
      data-just-mastered={justMastered ? "true" : undefined}
      className={cn(
        "relative overflow-hidden rounded-tv-card border bg-tv-surface transition-colors",
        locked ? "border-tv-border opacity-50" : "border-tv-border hover:border-tv-border-cyan",
        alive && "border-tv-border-cyan glow-card-cyan",
        settled && "border-tv-border-cyan/60",
      )}
      initial={justUnlocked && !reduceMotion ? { opacity: 0.55, y: 4 } : false}
      animate={justUnlocked && !reduceMotion ? { opacity: 1, y: 0 } : undefined}
      transition={{ duration: 0.3, delay: 0.15, ease: "easeOut" }}
    >
      {alive && !reduceMotion && (
        <motion.div
          aria-hidden
          className="pointer-events-none absolute inset-0 rounded-tv-card"
          animate={{
            boxShadow: [
              "0 0 12px 0 rgba(0,242,255,0.12)",
              "0 0 26px 3px rgba(0,242,255,0.32)",
              "0 0 12px 0 rgba(0,242,255,0.12)",
            ],
          }}
          transition={{ duration: 2.4, repeat: Infinity, ease: "easeInOut" }}
        />
      )}

      {justMastered && <UnlockFlare />}

      <div className="flex items-start justify-between gap-4 p-4">
        <div className="flex items-center gap-3">
          <MasteryRing score={subsection.mastery_score} status={subsection.status} />
          <div className="flex flex-col gap-1">
            <span className="font-mono text-xs text-tv-text-body/70">
              {String(subsection.sequence).padStart(2, "0")}
            </span>
            <h3
              className={cn(
                "font-display text-base leading-tight font-bold tracking-tight uppercase",
                locked ? "text-tv-locked" : "text-tv-text-hi",
              )}
            >
              {subsection.title}
            </h3>
          </div>
        </div>

        <AnimatePresence>
          {locked ? (
            <motion.div key="lock" exit={reduceMotion ? undefined : { opacity: 0, scale: 0.5 }} transition={{ duration: 0.25 }}>
              <Lock className="size-5 text-tv-locked" aria-hidden />
            </motion.div>
          ) : settled ? (
            <CheckCircle2 className="size-5 shrink-0 text-tv-cyan" aria-hidden />
          ) : null}
        </AnimatePresence>
      </div>

      <div className="flex items-center justify-between gap-3 px-4 pb-4">
        <Badge variant={statusBadgeVariant(subsection.status)}>{STATUS_LABEL[subsection.status]}</Badge>
        <span className="font-mono text-xs text-tv-text-body">
          {locked
            ? `Target ${Math.round(subsection.target_rating)}`
            : `${Math.round(subsection.user_rating)} → ${Math.round(subsection.target_rating)}`}
        </span>
      </div>

      <div className="flex items-center gap-4 border-t border-tv-border px-4 py-3 font-mono text-xs text-tv-text-body">
        <span>{subsection.tutorials.length} tutorials</span>
        <span>{subsection.questions.length} problems</span>
      </div>
    </motion.div>
  );

  if (locked) {
    return (
      <div aria-disabled className="cursor-not-allowed select-none">
        {card}
      </div>
    );
  }

  return (
    <Link
      href={`/roadmap/node/${subsection.node_id}`}
      className="block rounded-tv-card focus-visible:ring-2 focus-visible:ring-tv-cyan focus-visible:outline-none"
    >
      {card}
    </Link>
  );
}
