import { NodeDetailView } from "@/components/roadmap/node-detail-view";

/**
 * Route 10 (plan.md §2) — subsection detail: tutorials, practice questions,
 * `POST /roadmap/nodes/{id}/complete` and `.../test-out`. Next.js 15 params
 * are async — resolved here (Server Component) and handed to the client view.
 */
export default async function NodeDetailPage({ params }: { params: Promise<{ nodeId: string }> }) {
  const { nodeId } = await params;
  return <NodeDetailView nodeId={nodeId} />;
}
