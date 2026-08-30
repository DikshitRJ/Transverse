import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SSEProvider } from "@/components/providers/sse-provider";
import { setAccessToken } from "@/lib/auth/token-store";
import { roadmapState, buildRoadmap } from "@/mocks/fixtures/roadmap";
import { emitMockEvent } from "@/mocks/fixtures/event-bus";
import { RoadmapView } from "../roadmap-view";

/**
 * Full-stack render against the real MSW handlers (`src/mocks/handlers.ts`,
 * started globally by `vitest.setup.ts`) — not mocked-out query hooks.
 * Also exercises the real SSE stack (`SSEProvider` -> `transverseEventSource`
 * -> `fetch`+`ReadableStream` against MSW's `GET /events/stream` handler) to
 * prove the unlock animation actually fires off a `node.unlocked` event
 * pushed through the mock event bus, per ATLAS's "done when" criterion.
 */
describe("RoadmapView", () => {
  beforeEach(() => {
    setAccessToken("test-access-token");
    // Fixture state is a mutable module singleton the mock handlers write
    // to — reset it so each test starts from the known 3-section roadmap.
    roadmapState.current = buildRoadmap();
  });

  afterEach(() => {
    setAccessToken(null);
    cleanup();
  });

  function renderRoadmap() {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    return render(
      <QueryClientProvider client={queryClient}>
        <SSEProvider>
          <RoadmapView />
        </SSEProvider>
      </QueryClientProvider>,
    );
  }

  it("renders the active section's nodes and the locked upcoming-section previews", async () => {
    renderRoadmap();

    expect(await screen.findByText(/Foundations & Linear Structures/i)).toBeInTheDocument();

    // ACTIVE-section node, unlocked/in-progress — visible and a real link.
    const twoPointers = await screen.findByTestId(
      `node-card-${roadmapState.current.current_section!.subsections[2].node_id}`,
    );
    expect(twoPointers).toHaveAttribute("data-status", "in_progress");

    // Locked node *within* the active section — inert (no navigable link).
    const stackQueues = roadmapState.current.current_section!.subsections.find(
      (s) => s.topic_id === "stack-queues",
    )!;
    const lockedCard = screen.getByTestId(`node-card-${stackQueues.node_id}`);
    expect(lockedCard).toHaveAttribute("data-status", "locked");
    expect(lockedCard.closest("a")).toBeNull();

    // Genuinely-locked upcoming sections — title/sequence only, per the backend contract.
    expect(screen.getByText(/Trees, Graphs & Traversal/i)).toBeInTheDocument();
    expect(screen.getByText(/Dynamic Programming & Greedy/i)).toBeInTheDocument();
  });

  it("plays the unlock animation on a real node.unlocked SSE event", async () => {
    renderRoadmap();
    await screen.findByText(/Foundations & Linear Structures/i);

    const section = roadmapState.current.current_section!;
    const target = section.subsections.find((s) => s.topic_id === "stack-queues")!;
    expect(target.status).toBe("locked");

    // Simulate the backend unlocking this node "elsewhere" (e.g. another
    // tab completing the prerequisite): mutate the shared mock state the
    // same way the real `POST /roadmap/nodes/{id}/complete` handler does,
    // then push the event through the actual mock SSE bus — this reaches
    // the component only via the real fetch+ReadableStream SSE client.
    target.status = "unlocked";
    emitMockEvent({
      type: "node.unlocked",
      job_id: target.node_id,
      data: { node_id: target.node_id, topic_id: target.topic_id, title: target.title },
    });

    // SSEProvider auto-invalidates ["roadmap"] on node.unlocked, the query
    // refetches, diffRoadmap sees locked -> unlocked, and NodeCard renders
    // with justUnlocked (the lock-dissolve beat) for one render pass.
    await waitFor(
      () => {
        const card = screen.getByTestId(`node-card-${target.node_id}`);
        expect(card).toHaveAttribute("data-just-unlocked", "true");
      },
      { timeout: 5000 },
    );

    // And it clears back to a settled, non-transitional state shortly after
    // (the "≤400ms entrance" — we just assert it doesn't get stuck "on").
    await waitFor(
      () => {
        const card = screen.getByTestId(`node-card-${target.node_id}`);
        expect(card).not.toHaveAttribute("data-just-unlocked");
        expect(card).toHaveAttribute("data-status", "unlocked");
      },
      { timeout: 5000 },
    );
  });
});
