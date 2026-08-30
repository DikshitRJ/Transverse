import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { setAccessToken } from "@/lib/auth/token-store";
import { roadmapState, buildRoadmap } from "@/mocks/fixtures/roadmap";
import { TutorialReader } from "../tutorial-reader";

describe("TutorialReader", () => {
  beforeEach(() => {
    setAccessToken("test-access-token");
    roadmapState.current = buildRoadmap();
  });

  afterEach(() => {
    setAccessToken(null);
    cleanup();
  });

  function renderTutorial(tutorialId: string) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    return render(
      <QueryClientProvider client={queryClient}>
        <TutorialReader tutorialId={tutorialId} />
      </QueryClientProvider>,
    );
  }

  it("renders the tutorial's metadata, summary, and outbound source link", async () => {
    // t-104 = "The Two-Pointer Pattern" on the two-pointers node, per mocks/fixtures/roadmap.ts.
    renderTutorial("t-104");

    expect(await screen.findByRole("heading", { name: /the two-pointer pattern/i })).toBeInTheDocument();
    expect(screen.getByText(/min read/i)).toBeInTheDocument();

    const outbound = screen.getByRole("link", { name: /read on/i });
    expect(outbound).toHaveAttribute("href", "https://example.com/tutorials/t-104");
    expect(outbound).toHaveAttribute("target", "_blank");
    expect(outbound).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("shows a not-found state for an unknown tutorial id", async () => {
    renderTutorial("does-not-exist");
    expect(await screen.findByText(/tutorial not found/i)).toBeInTheDocument();
  });

  it("marking complete calls node-complete (no dedicated tutorial endpoint exists)", async () => {
    renderTutorial("t-104");
    const markBtn = await screen.findByRole("button", { name: /mark complete/i });
    fireEvent.click(markBtn);

    await waitFor(() => {
      const node = roadmapState.current.current_section!.subsections.find((s) => s.topic_id === "two-pointers")!;
      expect(node.status).toBe("mastered");
    });
  });
});
