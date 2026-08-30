"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/components/providers/auth-provider";
import { AUTHED_NAV_LINKS, AppNavActions } from "@/components/dashboard/app-nav";

export interface TopNavLink {
  label: string;
  href: string;
}

const DEFAULT_LINKS: TopNavLink[] = [
  { label: "Analyze", href: "/#analyze" },
  { label: "Quiz", href: "/onboarding/quiz" },
];

export interface TopNavProps {
  links?: TopNavLink[];
  /** Right-side slot. Defaults to avatar/profile/logout when authenticated or "Get Started" when signed out. */
  actions?: ReactNode;
  className?: string;
}

/**
 * App shell top nav — matches Figma `61:110`: glass panel, `rgba(0,255,255,0.5)`
 * border + glow, Space Grotesk logo with cyan text-glow.
 * Automatically adapts between signed-in and guest chrome based on authentication state.
 */
export function TopNav({ links, actions, className }: TopNavProps) {
  const { isAuthenticated, user, isLoading } = useAuth();
  const pathname = usePathname();

  const signedIn = isAuthenticated || Boolean(user);
  const effectiveLinks = links ?? (signedIn ? AUTHED_NAV_LINKS : DEFAULT_LINKS);

  return (
    <header
      className={cn(
        "glass-panel sticky top-0 z-50 flex w-full items-center justify-between border-tv-border-nav px-6 py-4 shadow-[0_0_15px_0_rgba(0,255,255,0.1)]",
        className,
      )}
    >
      <Link href={signedIn ? "/dashboard" : "/"} className="flex shrink-0 items-center gap-3">
        <Image
          src="/figma/byte-mascot-nav.png"
          alt="Byte the Beaver"
          width={40}
          height={36}
          className="h-9 w-10 object-contain"
          priority
        />
        <span className="glow-text-cyan font-display text-[28px] leading-8 font-bold tracking-[-1.2px] text-tv-text-hi uppercase">
          Transverse
        </span>
      </Link>

      <nav className="hidden items-center gap-8 md:flex">
        {effectiveLinks.map((link) => {
          const isActive = pathname === link.href || (link.href !== "/" && pathname.startsWith(link.href));
          return (
            <Link
              key={link.href}
              href={link.href}
              className={cn(
                "font-body px-2 py-1 text-sm font-semibold uppercase transition-colors hover:text-tv-text-hi",
                isActive ? "text-tv-cyan glow-text-cyan border-b-2 border-tv-cyan" : "text-tv-text-nav",
              )}
            >
              {link.label}
            </Link>
          );
        })}
      </nav>

      <div className="flex shrink-0 items-center gap-4">
        {actions ?? (
          signedIn ? (
            <AppNavActions />
          ) : !isLoading ? (
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                render={<Link href="/signin" />}
                className="font-mono text-xs text-tv-text-body hover:text-tv-text-hi"
              >
                Sign In
              </Button>
              <Button size="sm" render={<Link href="/onboarding" />}>
                Get Started
              </Button>
            </div>
          ) : null
        )}
      </div>
    </header>
  );
}
