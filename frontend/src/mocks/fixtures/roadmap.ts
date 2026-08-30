/**
 * A full 3-section roadmap mirroring `models.RoadmapCurrentResponse`
 * (backend/internal/models/roadmap.go): one ACTIVE section with fully
 * populated subsections (tutorials + questions), two LOCKED upcoming
 * section previews. Topic ids/names are pulled straight from the real
 * curriculum graph (`data/topics.json`) so this lines up with what the
 * live backend would actually return.
 */

import type {
  RoadmapCurrentResponse,
  RoadmapSubsection,
  Tutorial,
  UpcomingSectionPreview,
} from "@/lib/api/types";
import { PROBLEMS } from "./problems";

function questionsFor(topicId: string, count = 3) {
  const matches = PROBLEMS.filter((p) => p.topic === topicId);
  return matches.slice(0, count).map((p) => ({ ...p, statement: undefined, test_cases: undefined }));
}

function tutorial(
  id: string,
  topicId: string,
  title: string,
  opts: Partial<Tutorial> = {},
): Tutorial {
  return {
    id,
    source: opts.source ?? "neetcode",
    source_url: opts.source_url ?? `https://example.com/tutorials/${id}`,
    title,
    topic_id: topicId,
    topic_tags: opts.topic_tags ?? [topicId],
    type: opts.type ?? "article",
    difficulty: opts.difficulty ?? "beginner",
    estimated_minutes: opts.estimated_minutes ?? 12,
    summary: opts.summary ?? `A focused walkthrough of ${title.toLowerCase()}, with worked examples.`,
    license_note: opts.license_note,
    thumbnail_url: opts.thumbnail_url,
    status: opts.status ?? "UNREAD",
  };
}

function subsection(
  nodeId: string,
  topicId: string,
  title: string,
  sequence: number,
  status: RoadmapSubsection["status"],
  userRating: number,
  targetRating: number,
  masteryScore: number,
  tutorials: Tutorial[],
): RoadmapSubsection {
  return {
    node_id: nodeId,
    topic_id: topicId,
    title,
    sequence,
    status,
    user_rating: userRating,
    target_rating: targetRating,
    mastery_score: masteryScore,
    tutorials,
    questions: questionsFor(topicId),
  };
}

const ACTIVE_SUBSECTIONS: RoadmapSubsection[] = [
  subsection(
    "00000000-0000-4000-8000-000000000101",
    "foundations",
    "Foundations & Implementation",
    1,
    "mastered",
    1450,
    1200,
    1,
    [
      tutorial("t-101", "foundations", "Reading Constraints Like a Pro", {
        type: "article",
        estimated_minutes: 8,
        status: "COMPLETED",
      }),
      tutorial("t-102", "foundations", "Simulation Problems Walkthrough", {
        type: "video",
        estimated_minutes: 15,
        status: "COMPLETED",
      }),
    ],
  ),
  subsection(
    "00000000-0000-4000-8000-000000000102",
    "arrays-hashing",
    "Arrays & Hashing",
    2,
    "mastered",
    1480,
    1300,
    95,
    [
      tutorial("t-103", "arrays-hashing", "Hash Maps in 10 Minutes", {
        type: "video",
        estimated_minutes: 10,
        status: "COMPLETED",
      }),
    ],
  ),
  subsection(
    "00000000-0000-4000-8000-000000000103",
    "two-pointers",
    "Two Pointers",
    3,
    "in_progress",
    1310,
    1400,
    62,
    [
      tutorial("t-104", "two-pointers", "The Two-Pointer Pattern", {
        type: "article",
        estimated_minutes: 12,
        status: "COMPLETED",
      }),
      tutorial("t-105", "two-pointers", "Opposite-Direction Pointers, Visually", {
        type: "interactive",
        estimated_minutes: 20,
        status: "UNREAD",
      }),
    ],
  ),
  subsection(
    "00000000-0000-4000-8000-000000000104",
    "sliding-window",
    "Sliding Window",
    4,
    "unlocked",
    1180,
    1450,
    15,
    [
      tutorial("t-106", "sliding-window", "Fixed vs Variable Windows", {
        type: "article",
        estimated_minutes: 14,
        status: "UNREAD",
      }),
    ],
  ),
  subsection(
    "00000000-0000-4000-8000-000000000105",
    "stack-queues",
    "Stacks & Queues",
    5,
    "locked",
    0,
    1500,
    0,
    [
      tutorial("t-107", "stack-queues", "Monotonic Stacks From Scratch", {
        type: "article",
        estimated_minutes: 18,
        status: "UNREAD",
      }),
    ],
  ),
  subsection(
    "00000000-0000-4000-8000-000000000106",
    "binary-search",
    "Binary Search",
    6,
    "locked",
    0,
    1550,
    0,
    [
      tutorial("t-108", "binary-search", "Binary Search on the Answer", {
        type: "article",
        estimated_minutes: 16,
        status: "UNREAD",
      }),
    ],
  ),
];

const UPCOMING: UpcomingSectionPreview[] = [
  { sequence: 2, title: "Trees, Graphs & Traversal", status: "LOCKED" },
  { sequence: 3, title: "Dynamic Programming & Greedy", status: "LOCKED" },
];

export function buildRoadmap(): RoadmapCurrentResponse {
  return {
    roadmap_id: "00000000-0000-4000-8000-0000000000f1",
    user_id: "user-mock-001",
    user_rating: 1340,
    target_role: "Software Engineer - DSA & Problem Solving",
    status: "active",
    total_sections: 3,
    current_section: {
      phase_id: "00000000-0000-4000-8000-0000000000a1",
      sequence: 1,
      title: "Foundations & Linear Structures",
      status: "ACTIVE",
      progress_percentage: 42,
      subsections: ACTIVE_SUBSECTIONS,
    },
    upcoming_sections: UPCOMING,
  };
}

/** Mutable in-memory copy the handlers read/write against (node unlock progression, etc). */
export const roadmapState: { current: RoadmapCurrentResponse } = {
  current: buildRoadmap(),
};

export function findSubsection(nodeId: string): RoadmapSubsection | undefined {
  return roadmapState.current.current_section?.subsections.find((s) => s.node_id === nodeId);
}
