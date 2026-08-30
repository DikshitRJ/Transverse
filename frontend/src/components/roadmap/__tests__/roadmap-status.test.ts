import { describe, expect, it } from "vitest";
import { diffRoadmap, NO_TRANSITION, normalizeMastery } from "../roadmap-status";
import type { RoadmapCurrentResponse, RoadmapSubsection } from "@/lib/api/types";

function sub(overrides: Partial<RoadmapSubsection>): RoadmapSubsection {
  return {
    node_id: "n1",
    topic_id: "t1",
    title: "Node",
    sequence: 1,
    status: "unlocked",
    user_rating: 1000,
    target_rating: 1200,
    mastery_score: 0,
    tutorials: [],
    questions: [],
    ...overrides,
  };
}

function roadmap(phaseId: string, subsections: RoadmapSubsection[]): RoadmapCurrentResponse {
  return {
    roadmap_id: "r1",
    user_id: "u1",
    user_rating: 1000,
    target_role: "role",
    status: "active",
    total_sections: 3,
    current_section: {
      phase_id: phaseId,
      sequence: 1,
      title: "Section",
      status: "ACTIVE",
      progress_percentage: 0,
      subsections,
    },
    upcoming_sections: [],
  };
}

describe("normalizeMastery", () => {
  it("scales a 0-1 fraction (the mock fixtures' actual shape) to 0-100", () => {
    expect(normalizeMastery(0.62)).toBe(62);
    expect(normalizeMastery(1)).toBe(100);
    expect(normalizeMastery(0)).toBe(0);
  });

  it("passes through an already-0-100 value (per the Go doc comment)", () => {
    expect(normalizeMastery(85)).toBe(85);
  });

  it("clamps out-of-range and non-finite input", () => {
    expect(normalizeMastery(150)).toBe(100);
    expect(normalizeMastery(-5)).toBe(0);
    expect(normalizeMastery(Number.NaN)).toBe(0);
  });
});

describe("diffRoadmap", () => {
  it("returns NO_TRANSITION when there's no previous snapshot", () => {
    const next = roadmap("p1", [sub({ node_id: "a", status: "locked" })]);
    expect(diffRoadmap(null, next)).toBe(NO_TRANSITION);
  });

  it("detects a node transitioning into mastered", () => {
    const prev = roadmap("p1", [sub({ node_id: "a", status: "in_progress" })]);
    const next = roadmap("p1", [sub({ node_id: "a", status: "mastered", mastery_score: 1 })]);
    const t = diffRoadmap(prev, next);
    expect(t.masteredNodeId).toBe("a");
    expect(t.unlockedNodeId).toBeNull();
    expect(t.sectionChanged).toBe(false);
  });

  it("treats tested_out as settled too", () => {
    const prev = roadmap("p1", [sub({ node_id: "a", status: "unlocked" })]);
    const next = roadmap("p1", [sub({ node_id: "a", status: "tested_out" })]);
    expect(diffRoadmap(prev, next).masteredNodeId).toBe("a");
  });

  it("detects a node leaving locked — the unlock-dissolve trigger", () => {
    const prev = roadmap("p1", [
      sub({ node_id: "a", status: "in_progress" }),
      sub({ node_id: "b", status: "locked" }),
    ]);
    const next = roadmap("p1", [
      sub({ node_id: "a", status: "mastered", mastery_score: 1 }),
      sub({ node_id: "b", status: "unlocked" }),
    ]);
    const t = diffRoadmap(prev, next);
    expect(t.masteredNodeId).toBe("a");
    expect(t.unlockedNodeId).toBe("b");
  });

  it("detects a section change (phase_id differs)", () => {
    const prev = roadmap("p1", [sub({ node_id: "a", status: "mastered" })]);
    const next = roadmap("p2", [sub({ node_id: "c", status: "in_progress" })]);
    expect(diffRoadmap(prev, next).sectionChanged).toBe(true);
  });

  it("returns NO_TRANSITION for two identical snapshots (no-op refetch)", () => {
    const snap = roadmap("p1", [sub({ node_id: "a", status: "in_progress" })]);
    expect(diffRoadmap(snap, snap)).toBe(NO_TRANSITION);
  });
});
