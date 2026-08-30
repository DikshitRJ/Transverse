/**
 * Pure helpers shared by the roadmap surfaces (`/roadmap`,
 * `/roadmap/node/[nodeId]`). No React here on purpose — keeps the unlock
 * choreography's trigger condition (`diffRoadmap`) trivially unit-testable
 * without mounting anything.
 */
import type { NodeStatus, RoadmapCurrentResponse, RoadmapSubsection } from "@/lib/api/types";

/**
 * `models.RoadmapSubsection.mastery_score` — the Go doc comment implies a
 * 0-100 scale, but the MSW fixtures (`mocks/fixtures/roadmap.ts`) actually
 * populate it 0-1 (e.g. `0.62`, `0.95`, `1`). Normalize defensively so the
 * ring/progress UI is correct against either shape without caring which
 * one a given backend/mock revision sends.
 */
export function normalizeMastery(score: number): number {
  if (!Number.isFinite(score)) return 0;
  const pct = score <= 1 ? score * 100 : score;
  return Math.max(0, Math.min(100, Math.round(pct)));
}

export function isNodeLocked(status: NodeStatus): boolean {
  return status === "locked";
}

/** "Settled" — done, no longer needs the viewer's attention. */
export function isNodeSettled(status: NodeStatus): boolean {
  return status === "mastered" || status === "tested_out";
}

/** The single node that should feel "alive" (glow-pulse, plan.md §1.4 — rationed). */
export function isNodeAlive(status: NodeStatus): boolean {
  return status === "in_progress";
}

export const STATUS_LABEL: Record<NodeStatus, string> = {
  locked: "Locked",
  unlocked: "Unlocked",
  in_progress: "In Progress",
  mastered: "Mastered",
  tested_out: "Tested Out",
};

export type BadgeVariant = "default" | "secondary" | "success" | "warning" | "error" | "locked" | "outline";

export function statusBadgeVariant(status: NodeStatus): BadgeVariant {
  switch (status) {
    case "mastered":
    case "tested_out":
      return "success";
    case "in_progress":
      return "warning";
    case "unlocked":
      return "outline";
    case "locked":
    default:
      return "locked";
  }
}

export interface RoadmapTransition {
  /** A subsection that just became mastered/tested-out — plays the ring-complete + glow-flare beat. */
  masteredNodeId: string | null;
  /** A subsection that just left "locked" — plays the lock-dissolve beat. */
  unlockedNodeId: string | null;
  /** The active section itself changed (a new phase arrived) — plays the section-arrival beat. */
  sectionChanged: boolean;
}

export const NO_TRANSITION: RoadmapTransition = {
  masteredNodeId: null,
  unlockedNodeId: null,
  sectionChanged: false,
};

/**
 * Diffs two `GET /roadmap` snapshots to find what just happened between
 * them — this is what decides whether the unlock animation fires, and on
 * which node. Works whether the new snapshot arrived via direct mutation
 * refetch (complete/test-out) or via SSE-triggered cache invalidation
 * (`node.unlocked` / `roadmap.updated`) — same code path either way, so
 * the signature moment fires consistently regardless of trigger source.
 *
 * Only compares `current_section.subsections` — the only ones the backend
 * ever fully populates (locked upcoming sections are metadata-only
 * previews with nothing to diff).
 */
export function diffRoadmap(
  prev: RoadmapCurrentResponse | null,
  next: RoadmapCurrentResponse | null,
): RoadmapTransition {
  if (!prev?.current_section || !next?.current_section) return NO_TRANSITION;

  const sectionChanged = prev.current_section.phase_id !== next.current_section.phase_id;
  const prevById = new Map<string, RoadmapSubsection>(
    prev.current_section.subsections.map((s) => [s.node_id, s]),
  );

  let masteredNodeId: string | null = null;
  let unlockedNodeId: string | null = null;

  for (const sub of next.current_section.subsections) {
    const before = prevById.get(sub.node_id);
    if (!before) continue;
    if (!masteredNodeId && !isNodeSettled(before.status) && isNodeSettled(sub.status)) {
      masteredNodeId = sub.node_id;
    }
    if (!unlockedNodeId && isNodeLocked(before.status) && !isNodeLocked(sub.status)) {
      unlockedNodeId = sub.node_id;
    }
  }

  if (!masteredNodeId && !unlockedNodeId && !sectionChanged) return NO_TRANSITION;
  return { masteredNodeId, unlockedNodeId, sectionChanged };
}
