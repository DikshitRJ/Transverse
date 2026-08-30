"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authLogout, getMe } from "@/lib/api/endpoints";
import { ApiError } from "@/lib/api/client";
import type { User } from "@/lib/api/types";
import {
  getAccessToken,
  setAccessToken,
  subscribeAccessToken,
  subscribeAuthExpired,
} from "@/lib/auth/token-store";

interface CompleteOAuthCallbackParams {
  accessToken: string;
  refreshToken: string;
}

interface AuthContextValue {
  /** `undefined` while the initial silent-refresh-on-load hasn't resolved yet. */
  user: User | null | undefined;
  isAuthenticated: boolean;
  /** True only during the very first silent-refresh attempt on app load. */
  isLoading: boolean;
  /**
   * Called by the `/auth/callback` page (THRESHOLD) once the OAuth redirect
   * chain lands with tokens. Stores the access token in memory, hands the
   * refresh token to the Next.js Route Handler to become an httpOnly
   * cookie, and fetches the user profile. Never call this with tokens read
   * from anywhere other than that one redirect.
   */
  completeOAuthCallback: (params: CompleteOAuthCallbackParams) => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, username?: string) => Promise<void>;
  logout: () => Promise<void>;
  /** Re-fetches GET /auth/me. Useful after evidence sync / onboarding completes. */
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null | undefined>(() => {
    if (typeof window !== "undefined") {
      try {
        const cached = window.localStorage.getItem("tv_user");
        if (cached) return JSON.parse(cached) as User;
      } catch {
        // Ignore JSON parse errors
      }
    }
    return undefined;
  });
  const [accessToken, setAccessTokenState] = useState<string | null>(getAccessToken());
  const queryClient = useQueryClient();

  const loadUser = useCallback(async () => {
    try {
      const me = await getMe();
      setUser(me);
      if (typeof window !== "undefined") {
        try {
          window.localStorage.setItem("tv_user", JSON.stringify(me));
        } catch {
          // Ignore storage quota errors
        }
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setUser(null);
        if (typeof window !== "undefined") {
          window.localStorage.removeItem("tv_user");
        }
      } else {
        // Network/backend error — leave `user` as-is rather than bouncing a
        // signed-in user to signed-out on a transient failure.
        setUser((current) => (current === undefined ? null : current));
      }
    }
  }, []);

  // Subscribe to the token store so any writer (client.ts's silent refresh,
  // this provider itself) keeps `accessToken`/`isAuthenticated` in sync.
  useEffect(() => {
    return subscribeAccessToken((token) => {
      setAccessTokenState(token);
      if (token) {
        void loadUser();
      }
    });
  }, [loadUser]);

  useEffect(() => {
    return subscribeAuthExpired(() => {
      setUser(null);
      if (typeof window !== "undefined") {
        window.localStorage.removeItem("tv_user");
      }
      queryClient.clear();
    });
  }, [queryClient]);

  // Attempt to restore/verify session on first load.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const currentToken = getAccessToken();
      if (currentToken) {
        await loadUser();
      }

      try {
        const res = await fetch("/api/auth/refresh", { method: "POST", credentials: "include" });
        if (cancelled) return;
        if (res.ok) {
          const body = (await res.json()) as { access_token: string };
          setAccessToken(body.access_token);
          await loadUser();
        } else if (!currentToken) {
          setUser(null);
          if (typeof window !== "undefined") {
            window.localStorage.removeItem("tv_user");
          }
        }
      } catch {
        if (!cancelled && !currentToken) {
          setUser(null);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const completeOAuthCallback = useCallback(
    async ({ accessToken: newAccessToken, refreshToken }: CompleteOAuthCallbackParams) => {
      setAccessToken(newAccessToken);
      await fetch("/api/auth/session", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      await loadUser();
    },
    [loadUser],
  );

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) {
        const errorJson = await res.json().catch(() => ({ error: "Invalid email or password" }));
        throw new Error(errorJson.error || "Invalid email or password");
      }
      const data = (await res.json()) as { access_token: string; user?: User };
      setAccessToken(data.access_token);
      if (data.user) {
        setUser(data.user);
      } else {
        await loadUser();
      }
    },
    [loadUser],
  );

  const register = useCallback(
    async (email: string, password: string, username?: string) => {
      const res = await fetch("/api/auth/register", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password, username }),
      });
      if (!res.ok) {
        const errorJson = await res.json().catch(() => ({ error: "Registration failed" }));
        throw new Error(errorJson.error || "Registration failed");
      }
      const data = (await res.json()) as { access_token: string; user?: User };
      setAccessToken(data.access_token);
      if (data.user) {
        setUser(data.user);
      } else {
        await loadUser();
      }
    },
    [loadUser],
  );

  const logout = useCallback(async () => {
    try {
      await authLogout();
    } catch {
      // Ignore network errors during logout
    }
    setAccessToken(null);
    setUser(null);
    if (typeof window !== "undefined") {
      try {
        window.localStorage.removeItem("tv_user");
        window.localStorage.removeItem("tv_access_token");
      } catch {
        // Ignore storage errors
      }
    }
    queryClient.clear();
  }, [queryClient]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated: Boolean(accessToken) && user !== null,
      isLoading: user === undefined,
      completeOAuthCallback,
      login,
      register,
      logout,
      refreshUser: loadUser,
    }),
    [user, accessToken, completeOAuthCallback, login, register, logout, loadUser],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within <AuthProvider>");
  return ctx;
}
