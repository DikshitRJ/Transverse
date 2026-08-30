"use client";

import type { ReactNode } from "react";
import { MockProvider } from "@/components/providers/mock-provider";
import { QueryProvider } from "@/components/providers/query-provider";
import { AuthProvider } from "@/components/providers/auth-provider";
import { SSEProvider } from "@/components/providers/sse-provider";

/**
 * The full provider stack, mounted once in `src/app/layout.tsx`. Order
 * matters: MSW must be ready before anything fetches, TanStack Query must
 * exist before Auth/SSE (both call `useQueryClient`), Auth must exist
 * before SSE (SSE restarts its connection off auth-token changes).
 */
export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <MockProvider>
      <QueryProvider>
        <AuthProvider>
          <SSEProvider>{children}</SSEProvider>
        </AuthProvider>
      </QueryProvider>
    </MockProvider>
  );
}
