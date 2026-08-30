"use client";

import type { ReactNode } from "react";
import { AlertTriangleIcon, InboxIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ApiError } from "@/lib/api/client";

/** Turns a caught query error into a short, honest message — never a raw stack trace. */
export function describeError(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 401) return "Your session expired — sign in again to continue.";
    if (error.status === 0) return "Couldn't reach the server. Check your connection and try again.";
    return error.message || "The server returned an error.";
  }
  if (error instanceof Error) return error.message;
  return "Something went wrong loading this.";
}

export function ErrorPanel({
  error,
  onRetry,
  className,
}: {
  error?: unknown;
  onRetry?: () => void;
  className?: string;
}) {
  return (
    <div
      role="alert"
      className={cn(
        "glass-panel flex flex-col items-center gap-3 rounded-tv-card border border-tv-rose/30 px-6 py-10 text-center",
        className,
      )}
    >
      <AlertTriangleIcon className="size-6 text-tv-rose" aria-hidden="true" />
      <p className="max-w-sm font-body text-sm text-tv-text-body">{describeError(error)}</p>
      {onRetry && (
        <Button variant="outline-cyan" size="sm" onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  );
}

export function EmptyPanel({
  icon,
  title,
  description,
  action,
  className,
}: {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "glass-panel flex flex-col items-center gap-3 rounded-tv-card border border-tv-border px-6 py-12 text-center",
        className,
      )}
    >
      <div className="flex size-12 items-center justify-center rounded-tv-pill bg-tv-surface-2 text-tv-text-body">
        {icon ?? <InboxIcon className="size-5" aria-hidden="true" />}
      </div>
      <h3 className="font-heading text-h4 text-tv-text-hi">{title}</h3>
      {description && <p className="max-w-sm font-body text-sm text-tv-text-body">{description}</p>}
      {action}
    </div>
  );
}
