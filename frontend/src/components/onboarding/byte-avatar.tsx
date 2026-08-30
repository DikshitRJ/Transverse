import { cn } from "@/lib/utils";

/**
 * `public/figma/byte-mascot-chip.png` is not a clean circular avatar — it's
 * a crop of a wider "Byte + TRANSVERSE wordmark" composition, with the
 * mascot's face sitting left-of-center. Figma's own node (`15:66`) crops it
 * with `width: 140.66%, left: -20.33%, top: 0, height: 100%` inside an
 * `overflow-hidden` circle to isolate just the face — replicated here
 * exactly (a plain `<img>`, not `next/image`, since `next/image`'s
 * `fill`/`object-position` model can't express this literal percentage
 * offset transform) so every consumer (the onboarding mascot bubble, the
 * `/signin` card) gets the same crop instead of each re-deriving it.
 */
export function ByteAvatar({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-tv-pill",
        className,
      )}
    >
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src="/figma/byte-mascot-chip.png"
        alt="Byte the Beaver"
        className="absolute top-0 left-[-20.33%] h-full w-[140.66%] max-w-none object-cover"
      />
    </span>
  );
}
