export interface SvgIconProps {
  src: string;
  alt: string;
  className?: string;
}

/**
 * Renders a locally-downloaded Figma SVG icon (`public/figma/*.svg`) via a
 * plain `<img>` rather than `next/image`. `next/image` routes local sources
 * through Next's built-in optimizer, which refuses `image/svg+xml` unless
 * `images.dangerouslyAllowSVG` is set in `next.config.ts` — that file is
 * shared config outside this agent's owned paths, so this sidesteps it
 * instead of requesting the change. Raster assets (mascot PNGs) still use
 * `next/image` elsewhere on the landing page.
 */
export function SvgIcon({ src, alt, className }: SvgIconProps) {
  // eslint-disable-next-line @next/next/no-img-element -- SVG asset, see comment above
  return <img src={src} alt={alt} className={className} />;
}
