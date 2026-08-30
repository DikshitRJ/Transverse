import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function SectionHeading({
  title,
  description,
  action,
  className,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-wrap items-end justify-between gap-3", className)}>
      <div>
        <h2 className="font-heading text-h3 text-tv-text-hi">{title}</h2>
        {description && <p className="mt-0.5 font-body text-sm text-tv-text-body">{description}</p>}
      </div>
      {action}
    </div>
  );
}
