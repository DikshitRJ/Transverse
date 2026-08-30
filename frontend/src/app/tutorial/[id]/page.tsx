import { TutorialReader } from "@/components/tutorial/tutorial-reader";

/**
 * Route 11 (plan.md §2) — tutorial reader. Content comes from the roadmap
 * payload (no dedicated `GET /tutorial/{id}`), sourced externally — this
 * page shows summary/attribution/reading-time/difficulty plus an outbound
 * link, it does not mirror the tutorial body.
 */
export default async function TutorialPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <TutorialReader tutorialId={id} />;
}
