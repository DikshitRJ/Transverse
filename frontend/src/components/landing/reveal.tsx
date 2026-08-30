"use client";

import { motion, useReducedMotion, type Variants } from "motion/react";
import type { ReactNode } from "react";

const variants: Variants = {
  hidden: { opacity: 0, y: 16 },
  visible: { opacity: 1, y: 0 },
};

export interface RevealProps {
  children: ReactNode;
  className?: string;
  /** Stagger offset in seconds, for sequencing multiple `Reveal`s in one section. */
  delay?: number;
}

/**
 * Local scroll-reveal entrance, used only within `components/landing/` (per
 * BEACON's brief — the shared `components/motion/` vocabulary is BYTE's to
 * build, this doesn't duplicate it). Fades + lifts content into place once,
 * on first viewport entry, respecting plan.md's motion rules: entrance
 * ≤400ms, honours `prefers-reduced-motion`.
 */
export function Reveal({ children, className, delay = 0 }: RevealProps) {
  const shouldReduceMotion = useReducedMotion();

  if (shouldReduceMotion) {
    return <div className={className}>{children}</div>;
  }

  return (
    <motion.div
      className={className}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-80px" }}
      variants={variants}
      transition={{ duration: 0.35, delay, ease: "easeOut" }}
    >
      {children}
    </motion.div>
  );
}
