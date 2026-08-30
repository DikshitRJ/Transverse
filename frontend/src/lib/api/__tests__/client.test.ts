import { describe, expect, it } from "vitest";
import { getHealth } from "@/lib/api/endpoints";
import { apiFetch, ApiError } from "@/lib/api/client";

describe("apiFetch / mock layer smoke test", () => {
  it("GET /health resolves through the MSW mock handlers", async () => {
    const health = await getHealth();
    expect(health.status).toBe("ok");
  });

  it("throws a typed ApiError with the real {error} envelope on a protected route without a token", async () => {
    await expect(apiFetch("/user/profile")).rejects.toBeInstanceOf(ApiError);
    try {
      await apiFetch("/user/profile");
      throw new Error("expected apiFetch to reject");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(401);
    }
  });
});
