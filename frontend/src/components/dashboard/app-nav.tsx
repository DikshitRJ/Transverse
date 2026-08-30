"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import { LogOutIcon, SettingsIcon, UserIcon } from "lucide-react";
import { useAuth } from "@/components/providers/auth-provider";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { TopNavLink } from "@/components/shell/top-nav";

/** The signed-in nav link set — TopNav defaults to the public Analyze/Quiz pair, this replaces it on every authenticated PRISM route. */
export const AUTHED_NAV_LINKS: TopNavLink[] = [
  { label: "Dashboard", href: "/dashboard" },
  { label: "Roadmap", href: "/roadmap" },
  { label: "Practice", href: "/practice" },
  { label: "Problems", href: "/problems" },
];

/**
 * Right-side `<TopNav actions>` slot for every authenticated PRISM route —
 * profile/settings icon buttons + logout. Built once here so dashboard,
 * profile, problems, and settings all present the same signed-in chrome
 * (TopNav itself is FOUNDRY's shared shell component; this only supplies
 * its `actions` prop, per FOUNDATION.md's documented extension point).
 */
export function AppNavActions() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const [loggingOut, setLoggingOut] = useState(false);

  async function handleLogout() {
    setLoggingOut(true);
    try {
      await logout();
      router.push("/");
    } catch {
      setLoggingOut(false);
    }
  }

  return (
    <div className="flex items-center gap-1.5">
      {user?.username && (
        <span className="hidden font-mono text-xs text-tv-text-nav lg:inline">{user.username}</span>
      )}
      <Button
        render={<Link href="/profile" aria-current={pathname === "/profile" ? "page" : undefined} />}
        variant="ghost"
        size="icon-sm"
        aria-label="Profile"
        title="Profile"
        className={cn(pathname === "/profile" && "text-tv-cyan")}
      >
        <UserIcon />
      </Button>
      <Button
        render={<Link href="/settings" aria-current={pathname === "/settings" ? "page" : undefined} />}
        variant="ghost"
        size="icon-sm"
        aria-label="Settings"
        title="Settings"
        className={cn(pathname === "/settings" && "text-tv-cyan")}
      >
        <SettingsIcon />
      </Button>
      <Button variant="outline" size="sm" onClick={handleLogout} disabled={loggingOut}>
        <LogOutIcon />
        {loggingOut ? "Logging out…" : "Log out"}
      </Button>
    </div>
  );
}
