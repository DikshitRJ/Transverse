/**
 * Node MSW server — intercepts server-side requests (RSC data fetches,
 * the `src/app/api/auth/*` Route Handlers calling `BACKEND_URL`, and
 * Vitest). Started by `instrumentation.ts` in mock mode, and directly by
 * test setup files.
 */
import { setupServer } from "msw/node";
import { handlers } from "./handlers";

export const server = setupServer(...handlers);
