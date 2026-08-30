"use client";

/**
 * Visual QA / integration-reference surface for `components/motion` and
 * `components/byte`. Not mounted anywhere by this library (BYTE owns no
 * routes) — Wave 2 can temporarily drop `<MotionByteShowcase />` onto any
 * page (e.g. a scratch route) to see every primitive live, then remove the
 * import. Doubles as executable documentation: every section below is a
 * minimal, real usage of the export it demonstrates.
 */

import { useState } from "react";
import {
  CyanSweep,
  SweepFrame,
  GlowPulse,
  ScanlineGrid,
  TerminalType,
  UnlockTransition,
  VerdictFeedback,
  useHoverActive,
  type Verdict,
} from "@/components/motion";
import { Byte, type ByteState } from "./byte";
import { ByteSpeech } from "./byte-speech";
import { ByteDock } from "./byte-dock";
import { ByteMoment, type ByteMomentVariant } from "./byte-moment";

const BYTE_STATES: ByteState[] = ["idle", "thinking", "celebrating", "hinting", "error"];
const MOMENT_VARIANTS: ByteMomentVariant[] = [
  "empty-dashboard",
  "empty-roadmap",
  "judge0-failed",
  "hint-rate-limited",
];

export function MotionByteShowcase() {
  return (
    <div className="flex flex-col gap-16 bg-tv-bg-page p-10 font-body text-tv-text-hi">
      <SweepSection />
      <GlowSection />
      <UnlockSection />
      <TerminalTypeSection />
      <ScanlineSection />
      <VerdictSection />
      <ByteStatesSection />
      <ByteSpeechSection />
      <ByteMomentSection />
      <ByteDock state="hinting" message='Not sure? The quiz is a fun way to warm up!' />
    </div>
  );
}

function SectionHeading({ title, note }: { title: string; note: string }) {
  return (
    <div className="max-w-2xl">
      <h2 className="font-display text-h2 font-bold text-tv-text-hi">{title}</h2>
      <p className="mt-1 text-sm text-tv-text-body">{note}</p>
    </div>
  );
}

function SweepSection() {
  const { active, bind } = useHoverActive();
  return (
    <section className="flex flex-col gap-4">
      <SectionHeading
        title="Cyan sweep"
        note="Hover/focus either card. Left: SweepFrame (zero-config). Right: manual useHoverActive + CyanSweep for when you need the active boolean elsewhere too."
      />
      <div className="flex flex-wrap gap-6">
        <SweepFrame className="w-64 rounded-tv-card border border-tv-border bg-tv-surface p-6">
          <p className="font-mono text-sm text-tv-text-hi">SweepFrame</p>
        </SweepFrame>
        <div {...bind} className="relative w-64 rounded-tv-card border border-tv-border bg-tv-surface p-6">
          <p className="font-mono text-sm text-tv-text-hi">useHoverActive + CyanSweep</p>
          <CyanSweep active={active} edge="top" />
        </div>
      </div>
    </section>
  );
}

function GlowSection() {
  return (
    <section className="flex flex-col gap-4">
      <SectionHeading
        title="Glow pulse"
        note="2.4s breathing glow. Ration to one active element per screen — this is the only one on this page."
      />
      <GlowPulse className="w-64 border border-tv-border bg-tv-surface p-6">
        <p className="font-mono text-sm text-tv-text-hi">Active roadmap node</p>
      </GlowPulse>
    </section>
  );
}

function UnlockSection() {
  const [unlocked, setUnlocked] = useState(false);
  return (
    <section className="flex flex-col gap-4">
      <SectionHeading
        title="Unlock transition"
        note="The signature moment: ring completes → flare → lock dissolves → card lifts. Click to fire it — boolean-driven, so it composes with an SSE node.unlocked handler just as well as this button."
      />
      <div className="flex items-center gap-4">
        <UnlockTransition unlocked={unlocked} className="w-64">
          <div className="rounded-tv-card border border-tv-border bg-tv-surface p-6">
            <p className="font-mono text-sm text-tv-text-hi">Roadmap: Dynamic Programming</p>
          </div>
        </UnlockTransition>
        <button
          type="button"
          onClick={() => setUnlocked((v) => !v)}
          className="rounded-tv-btn border border-tv-border-cyan px-3 py-1.5 font-mono text-xs text-tv-cyan uppercase hover:bg-tv-cyan/10"
        >
          {unlocked ? "Reset" : "Unlock"}
        </button>
      </div>
    </section>
  );
}

function TerminalTypeSection() {
  const [text, setText] = useState('Not sure? The quiz is a fun way to warm up!');
  return (
    <section className="flex flex-col gap-4">
      <SectionHeading title="Terminal type-on" note="~28ms/char. Click to retype with a new string." />
      <div className="flex items-center gap-4">
        <TerminalType text={text} className="text-sm text-tv-text-body" />
        <button
          type="button"
          onClick={() =>
            setText((t) =>
              t.startsWith("Not sure")
                ? "Nice — that theta jump means the last three were genuinely harder."
                : 'Not sure? The quiz is a fun way to warm up!',
            )
          }
          className="shrink-0 rounded-tv-btn border border-tv-border-cyan px-3 py-1.5 font-mono text-xs text-tv-cyan uppercase hover:bg-tv-cyan/10"
        >
          Retype
        </button>
      </div>
    </section>
  );
}

function ScanlineSection() {
  return (
    <section className="flex flex-col gap-4">
      <SectionHeading title="Scanline grid" note="Very low-opacity animated grid — hero/auth grounds only." />
      <div className="relative h-40 overflow-hidden rounded-tv-card border border-tv-border bg-tv-bg">
        <ScanlineGrid />
        <div className="relative z-10 flex h-full items-center justify-center">
          <p className="text-gradient-display font-display text-h1 font-bold uppercase">Transverse</p>
        </div>
      </div>
    </section>
  );
}

function VerdictSection() {
  const [verdict, setVerdict] = useState<Verdict>(null);
  const [token, setToken] = useState(0);

  const fire = (next: Verdict) => {
    setVerdict(next);
    setToken((t) => t + 1);
  };

  return (
    <section className="flex flex-col gap-4">
      <SectionHeading
        title="Verdict feedback"
        note="Pass: cyan ripple. Fail: single 120ms rose shake, no bounce. playToken lets the same verdict replay back-to-back."
      />
      <div className="flex items-center gap-4">
        <VerdictFeedback verdict={verdict} playToken={token} className="w-64">
          <div className="rounded-tv-card border border-tv-border bg-tv-surface p-6">
            <p className="font-mono text-sm text-tv-text-hi">Test case 3/8</p>
          </div>
        </VerdictFeedback>
        <button
          type="button"
          onClick={() => fire("pass")}
          className="rounded-tv-btn border border-tv-cyan/40 px-3 py-1.5 font-mono text-xs text-tv-cyan uppercase hover:bg-tv-cyan/10"
        >
          Pass
        </button>
        <button
          type="button"
          onClick={() => fire("fail")}
          className="rounded-tv-btn border border-tv-rose/40 px-3 py-1.5 font-mono text-xs text-tv-rose uppercase hover:bg-tv-rose/10"
        >
          Fail
        </button>
      </div>
    </section>
  );
}

function ByteStatesSection() {
  const [state, setState] = useState<ByteState>("idle");
  return (
    <section className="flex flex-col gap-4">
      <SectionHeading title="Byte — states" note="idle · thinking · celebrating · hinting · error" />
      <div className="flex items-center gap-6">
        <Byte state={state} size="lg" variant="hero" />
        <div className="flex flex-wrap gap-2">
          {BYTE_STATES.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => setState(s)}
              className="rounded-tv-btn border border-tv-border px-3 py-1.5 font-mono text-xs text-tv-text-nav uppercase hover:bg-tv-surface-2 aria-pressed:border-tv-cyan aria-pressed:text-tv-cyan"
              aria-pressed={state === s}
            >
              {s}
            </button>
          ))}
        </div>
      </div>
    </section>
  );
}

function ByteSpeechSection() {
  return (
    <section className="flex flex-col gap-4">
      <SectionHeading title="Byte — speech pill" note="Matches Figma 15:10's mascot-mentor bubble exactly." />
      <div className="flex items-start gap-3">
        <Byte state="hinting" size="sm" />
        <ByteSpeech message='Not sure? The quiz is a fun way to warm up!' />
      </div>
    </section>
  );
}

function ByteMomentSection() {
  return (
    <section className="flex flex-col gap-4">
      <SectionHeading
        title="Byte — moments"
        note="Empty/error states built around Byte instead of a grey box. Four named moments plus two generic ones (not shown here)."
      />
      <div className="grid grid-cols-2 gap-4">
        {MOMENT_VARIANTS.map((variant) => (
          <ByteMoment key={variant} variant={variant} />
        ))}
      </div>
    </section>
  );
}
