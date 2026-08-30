import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface PageContainerProps {
  children: ReactNode;
  className?: string;
  /** Set false to let content bleed full-width (e.g. the IDE split view). Default true. */
  constrained?: boolean;
}

/** The 1280px content-width wrapper every Figma frame is authored against. */
export function PageContainer({ children, className, constrained = true }: PageContainerProps) {
  return (
    <div
      className={cn(
        "w-full px-6 py-12 md:px-12",
        constrained && "mx-auto max-w-[1280px]",
        className,
      )}
    >
      {children}
    </div>
  );
}
