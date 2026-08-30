import { RoadmapView } from "@/components/roadmap/roadmap-view";

/**
 * Route 9 (plan.md §2) — `GET /roadmap`: the active section fully populated,
 * plus locked previews of what's ahead. See `RoadmapView` for the actual
 * fetch/loading/empty/error states and the unlock choreography.
 */
export default function RoadmapPage() {
  return <RoadmapView />;
}
