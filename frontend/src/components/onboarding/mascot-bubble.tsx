import { cn } from "@/lib/utils";
import { ByteAvatar } from "@/components/onboarding/byte-avatar";

/**
 * The "Mascot Mentor" pill (Figma `15:10` node `15:64`) — Byte's dialogue
 * chip. Reused wherever Byte needs to say something in-context across the
 * onboarding funnel (the chooser, and evidence-sync's empty/error states),
 * so the `quote` is a prop rather than hard-coded.
 */
export function MascotBubble({ quote, className }: { quote: string; className?: string }) {
  return (
    <div
      className={cn(
        "flex items-center gap-4 rounded-tv-pill border border-tv-border-muted bg-tv-surface-deep py-[9px] pr-[25px] pl-[9px]",
        className,
      )}
    >
      <ByteAvatar className="size-12 border border-tv-border-cyan bg-tv-bg-page" />
      <p className="font-mono text-sm leading-5">
        <span className="font-bold text-tv-cyan">Transverse says:</span>{" "}
        <span className="text-tv-text-body">{quote}</span>
      </p>
    </div>
  );
}
