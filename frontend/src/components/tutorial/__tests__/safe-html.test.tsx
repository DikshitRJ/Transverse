import { describe, expect, it } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { SafeHtml } from "../safe-html";

describe("SafeHtml", () => {
  it("renders allowlisted structural tags with their text content", async () => {
    render(
      <SafeHtml html="<h2>Two Sum</h2><p>Given an array <strong>nums</strong>.</p><ul><li>O(n)</li></ul>" />,
    );
    expect(await screen.findByRole("heading", { level: 2, name: "Two Sum" })).toBeInTheDocument();
    expect(screen.getByText("O(n)")).toBeInTheDocument();
    const strong = await screen.findByText("nums");
    expect(strong.tagName).toBe("STRONG");
  });

  it("never executes or renders <script> content, and drops it entirely rather than unwrapping", async () => {
    render(<SafeHtml html="<p>before</p><script>window.__pwned = true;</script><p>after</p>" />);
    await screen.findByText("before");
    expect(screen.getByText("after")).toBeInTheDocument();
    expect(screen.queryByText(/__pwned/)).not.toBeInTheDocument();
    expect((window as unknown as { __pwned?: boolean }).__pwned).toBeUndefined();
  });

  it("strips event-handler attributes (onerror etc.) — never copies unlisted attrs", async () => {
    const { container } = render(
      <SafeHtml html={'<img src="https://example.com/a.png" onerror="window.__pwned2 = true" alt="x">'} />,
    );
    await waitFor(() => expect(container.querySelector("img")).toBeInTheDocument());
    const img = container.querySelector("img");
    expect(img).not.toHaveAttribute("onerror");
    expect(img).toHaveAttribute("src", "https://example.com/a.png");
  });

  it("rejects javascript: URLs on links, keeps http(s) ones and forces rel=noopener", async () => {
    render(
      <SafeHtml
        html={
          '<a href="javascript:alert(1)" id="bad">bad</a><a href="https://example.com" id="good">good</a>'
        }
      />,
    );
    const bad = await screen.findByText("bad");
    const good = screen.getByText("good");
    expect(bad).not.toHaveAttribute("href");
    expect(good).toHaveAttribute("href", "https://example.com");
    expect(good).toHaveAttribute("rel", "noopener noreferrer");
    expect(good).toHaveAttribute("target", "_blank");
  });

  it("unwraps disallowed but harmless wrapper tags, keeping their sanitized children", async () => {
    render(<SafeHtml html="<section><figure><p>caption text</p></figure></section>" />);
    expect(await screen.findByText("caption text")).toBeInTheDocument();
  });
});
