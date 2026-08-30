import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { setAccessToken } from "@/lib/auth/token-store";
import { roadmapState, buildRoadmap } from "@/mocks/fixtures/roadmap";
import { NodeDetailView } from "../node-detail-view";

describe("NodeDetailView", () => {
  beforeEach(() => {
    setAccessToken("test-access-token");
    roadmapState.current = buildRoadmap();
  });

  afterEach(() => {
    setAccessToken(null);
    cleanup();
  });

  function renderNode(nodeId: string) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    return render(
      <QueryClientProvider client={queryClient}>
        <NodeDetailView nodeId={nodeId} />
      </QueryClientProvider>,
    );
  }

  it("renders an in-progress node's tutorials, questions, and both actions", async () => {
    const nodeId = roadmapState.current.current_section!.subsections.find((s) => s.topic_id === "two-pointers")!
      .node_id;
    renderNode(nodeId);

    expect(await screen.findByRole("heading", { name: /two pointers/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /mark complete/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /test out/i })).toBeInTheDocument();
    expect(screen.getByText(/the two-pointer pattern/i)).toBeInTheDocument();
  });

  it("shows a locked, inert state for a locked node (no complete/test-out actions)", async () => {
    const nodeId = roadmapState.current.current_section!.subsections.find((s) => s.topic_id === "stack-queues")!
      .node_id;
    renderNode(nodeId);

    expect(await screen.findByText(/is locked/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /mark complete/i })).not.toBeInTheDocument();
  });

  it("shows a not-found state for an unknown node id", async () => {
    renderNode("does-not-exist");
    expect(await screen.findByText(/node not found/i)).toBeInTheDocument();
  });

  it("completing a node calls the API and reflects the mastered state", async () => {
    const nodeId = roadmapState.current.current_section!.subsections.find((s) => s.topic_id === "two-pointers")!
      .node_id;
    renderNode(nodeId);

    const completeBtn = await screen.findByRole("button", { name: /mark complete/i });
    fireEvent.click(completeBtn);

    await waitFor(() => {
      expect(screen.getByText(/mastered/i)).toBeInTheDocument();
    });
  });
});
