# `components/motion` + `components/byte`

Built by BYTE (Wave 1, creative-latitude agent). Two self-contained
libraries, no routes, zero coupling to any screen:

- **`src/components/motion/`** — plan.md §1.4's shared motion vocabulary,
  implemented once. This is what makes the app read as one designed
  product instead of eight screens with eight different ideas of what
  "hover" means.
- **`src/components/byte/`** (this directory) — Byte the Beaver, the AI
  tutor mascot, in his product-facing states, plus the empty/error
  "moments" built around him.

Both barrel-export from `index.ts`:

```ts
import { CyanSweep, GlowPulse, UnlockTransition, /* ... */ } from "@/components/motion";
import { Byte, ByteSpeech, ByteDock, ByteMoment, /* ... */ } from "@/components/byte";
```

Everything here is a client component (or a hook) unless noted. Every
animated primitive honours `prefers-reduced-motion` — see each entry for
exactly what "reduced" means for that primitive (never just "faster").

See `src/components/byte/demo.tsx`'s `<MotionByteShowcase />` for a live,
composed reference of everything below — temporarily mount it on any page
to see it all working, then remove the import. It is not mounted anywhere
by this library.

---

## `components/motion`

### Tokens — `tokens.ts`

The shared timing/easing vocabulary. Import these instead of hand-rolling
a new duration or easing curve anywhere you write custom `motion/react`
code.

```ts
export const DURATION: {
  sweep: 900;                // CyanSweep comet traversal, ms
  glowPulseCycle: 2400;      // GlowPulse full breath — fixed by plan.md §1.4
  entranceMax: 400;          // hard ceiling for any entrance — plan.md §1.4
  typeOnCharMs: 28;          // TerminalType speed — fixed by plan.md §1.4
  verdictPassMs: 550;
  verdictFailMs: 120;        // fixed by plan.md §1.4 ("single 120ms rose shake")
  unlockRingMs: 500;
  unlockFlareMs: 260;
  unlockDissolveMs: 240;
  unlockLiftMs: 320;
  pageTransitionMs: 300;     // plan.md §1.4 caps page transitions at ≤400ms
  pageTransitionExitMs: 200;
  byteCelebrateMs: 480;
  byteIdleBobMs: 3200;
};

export const EASE: {
  standard: [0.22, 1, 0.36, 1]; // fast-out, gentle settle — the app's default "arrival" curve
  sharp: [0.4, 0, 0.2, 1];      // quick state flips (verdicts, dissolves)
  linear: "linear";
};

export const SHAKE_KEYFRAMES_X: number[]; // [0, -6, 6, -4, 4, 0] — shared by VerdictFeedback (fail) and <Byte state="error">
export const GLOW_COLOR: { cyan: "0,242,255"; rose: "255,107,107" };
export type GlowColor = "cyan" | "rose";
```

**Use this when** you're writing a one-off `motion/react` animation
somewhere in the app and want it to feel like it belongs to the same
system as everything else.

---

### `usePrefersReducedMotion()` — `use-prefers-reduced-motion.ts`

```ts
function usePrefersReducedMotion(): boolean
```

Coalesced wrapper around `motion/react`'s `useReducedMotion()` (which
returns `boolean | null`). Every primitive below uses this internally.

**Use this when** you're writing custom `motion/react` code outside this
library and need the reduced-motion branch. You do **not** need this for
plain CSS `transition`/`animation` — `globals.css` already collapses those
globally.

---

### `useHoverActive()` — `use-hover-active.ts`

```ts
function useHoverActive(): {
  active: boolean; // true while hovered OR focused
  bind: { onMouseEnter, onMouseLeave, onFocus, onBlur }; // spread onto the element you're watching
}
```

**Use this when** you want an "is this element engaged" boolean to drive
`CyanSweep`, a border color, anything — without relying on Tailwind's
`group-hover`. `onFocus`/`onBlur` bubble in React, so focus landing on an
interactive descendant (a link/button inside a card) counts too.

---

### `<CyanSweep />` — `cyan-sweep.tsx`

```tsx
interface CyanSweepProps {
  active: boolean;                                 // controlled
  edge?: "top" | "bottom" | "left" | "right";       // default "bottom"
  className?: string;
}
function CyanSweep(props: CyanSweepProps): JSX.Element
```

The app's default "alive" gesture (plan.md §1.4): a 1px cyan line that
traverses one edge. Fully controlled — drive `active` from
`useHoverActive()`, a manual boolean, an `onFocus` handler, anything.
Positions itself absolutely; place it as a child of a `position: relative`
container. Clips its own travel internally, so it never causes reflow or
scroll.

**Reduced motion:** the traveling comet is skipped; only a static dim line
fades in/out (opacity only, ~150ms).

**Use this when** you need the sweep as one signal among several driven
off the same `active` boolean (e.g. also toggling a `GlowPulse` or a badge
color). For the common "just make this card sweep on hover" case, use
`SweepFrame` instead.

---

### `<SweepFrame />` — `sweep-frame.tsx`

```tsx
interface SweepFrameProps {
  children: ReactNode;
  edge?: "top" | "bottom" | "left" | "right"; // default "bottom"
  className?: string;
}
function SweepFrame(props: SweepFrameProps): JSX.Element
```

Zero-config wrapper: manages its own hover/focus state and wires up
`CyanSweep` for you.

```tsx
<SweepFrame edge="bottom" className="rounded-tv-card">
  <Card>...</Card>
</SweepFrame>
```

**Use this when** you just want "make this card feel alive on hover," no
state management of your own.

---

### `<GlowPulse />` — `glow-pulse.tsx`

```tsx
interface GlowPulseProps {
  children: ReactNode;
  color?: "cyan" | "rose"; // default "cyan"
  className?: string;
}
function GlowPulse(props: GlowPulseProps): JSX.Element
```

2.4s breathing glow behind `children` (opacity-only animation on a
separate glow layer — `box-shadow` itself stays static, so this is
compositor-friendly).

**⚠ Ration this to the single active element on a screen at a time** —
the current roadmap node, Byte's chip when he's hinting. More than one
`GlowPulse` visible at once also breaks the "never more than two things
animating at once" rule on its own.

**Reduced motion:** renders a static glow at mid-intensity — no breathing.

**Use this when** marking the one thing on screen that most wants
attention right now.

---

### `useUnlockSequence()` + `<UnlockTransition />` — `use-unlock-sequence.ts`, `unlock-transition.tsx`

```ts
type UnlockStage = "locked" | "ring" | "flare" | "dissolve" | "lifted";

function useUnlockSequence(
  unlocked: boolean,
  options?: { onComplete?: () => void },
): { stage: UnlockStage; isSequencing: boolean }
```

```tsx
interface UnlockTransitionProps {
  unlocked: boolean;
  children: ReactNode;
  lockIcon?: ReactNode;           // replaces the default lucide Lock glyph
  onSequenceComplete?: () => void;
  className?: string;
}
function UnlockTransition(props: UnlockTransitionProps): JSX.Element
```

**The signature moment** (plan.md §1.4): ring completes around a lock
glyph → glow flare → lock dissolves → the whole card lifts. Plays exactly
once, on the `false → true` transition of `unlocked` — mounting
already-`unlocked` renders straight into the end state, no replay (a page
load with 6 already-unlocked nodes must not fire 6 animations). This is
the **canonical implementation** — ATLAS's roadmap-local version is meant
to be replaced by this at merge.

```tsx
<UnlockTransition
  unlocked={node.status === "unlocked"}
  onSequenceComplete={() => toast.success("Section unlocked")}
>
  <RoadmapNodeCard node={node} />
</UnlockTransition>
```

`children` always renders — `UnlockTransition` never hides content while
locked; pair it with your own dimmed/`pointer-events-none` styling (e.g.
the `locked` Badge variant) for the pre-unlock look. It only owns the lock
glyph + ring + lift chrome layered on top.

**Reduced motion:** `stage` jumps directly `"locked" → "lifted"`;
`onSequenceComplete` still fires.

**Use `<UnlockTransition>` when** you want the full turnkey visual. **Use
`useUnlockSequence()` directly when** you need custom per-stage visuals —
same state machine, no rendering opinions attached.

---

### `useTypeOn()` + `<TerminalType />` — `use-type-on.ts`, `terminal-type.tsx`

```ts
interface UseTypeOnOptions {
  speedMs?: number;   // default 28 (DURATION.typeOnCharMs)
  enabled?: boolean;  // default true; false renders `text` immediately
}
function useTypeOn(text: string, options?: UseTypeOnOptions): {
  displayedText: string;
  isTyping: boolean;
  skip: () => void;   // reveal the rest of `text` immediately
}
```

```tsx
interface TerminalTypeProps extends UseTypeOnOptions {
  text: string;
  className?: string;
  showCursor?: boolean;      // default true
  as?: "span" | "p" | "div"; // default "span"
}
function TerminalType(props: TerminalTypeProps): JSX.Element
```

Types `text` in at plan.md §1.4's ~28ms/char (Byte's dialogue, verdict
copy). Resets and retypes whenever `text` changes. The rendered element's
accessible name (`aria-label`) is always the complete string — screen
readers get it immediately, not character-by-character.

**Reduced motion (or `enabled: false`):** the full string appears
instantly — never a faster typing speed.

**Use `<TerminalType>` when** you just want styled, typing-in text. **Use
`useTypeOn()` directly when** you need the in-progress string for
something other than plain rendering (e.g. driving a second element in
sync, or building your own cursor treatment).

---

### `<ScanlineGrid />` — `scanline-grid.tsx`

```tsx
interface ScanlineGridProps { className?: string; }
function ScanlineGrid(props: ScanlineGridProps): JSX.Element
```

A very low-opacity animated grid, **hero + auth grounds only** (plan.md
§1.4 — not a generic background texture). Absolutely positioned, fills
its nearest `position: relative` ancestor, ignores pointer events, fades
toward its own edges via a radial mask.

```tsx
<section className="relative overflow-hidden bg-tv-bg">
  <ScanlineGrid />
  <div className="relative z-10">...hero content...</div>
</section>
```

Implemented with a plain CSS `@keyframes` (not `motion/react`) —
`prefers-reduced-motion` is already handled globally by `globals.css`, no
JS branch needed. **Not** a client component boundary on its own.

**Use this when** you're building the landing hero or an auth screen
ground, and only then.

---

### `<VerdictFeedback />` — `verdict-feedback.tsx`

```ts
type Verdict = "pass" | "fail" | null;

interface VerdictFeedbackProps {
  verdict: Verdict;
  playToken?: string | number; // bump to replay the same verdict again; defaults to `verdict`
  children: ReactNode;
  className?: string;
}
function VerdictFeedback(props: VerdictFeedbackProps): JSX.Element
```

Wraps a test-case row / answer row. On `verdict` (keyed by `playToken`)
changing to non-null: **pass** → a cyan ring ripples outward once;
**fail** → a single 120ms rose shake, no bounce, no spring.

```tsx
<VerdictFeedback verdict={result?.verdict ?? null} playToken={result?.submissionId}>
  <TestCaseRow ... />
</VerdictFeedback>
```

**Reduced motion:** no ripple, no shake — a static colored outline
(cyan/rose) applies immediately and holds for as long as `verdict` does.
That's the "instant state change," not a faster animation.

**Use this when** rendering IDE test-case results or practice-mode answer
feedback.

---

### `<PageTransition />` — `page-transition.tsx`

```tsx
interface PageTransitionProps {
  transitionKey: string; // e.g. usePathname()
  children: ReactNode;
  className?: string;
}
function PageTransition(props: PageTransitionProps): JSX.Element
```

Fades + lifts incoming content ~8px over 300ms; outgoing content fades out
faster (200ms) so there's never a moment where both compete for attention.
Total ≤400ms per plan.md §1.4.

Next's `layout.tsx` doesn't remount on navigation — mount this inside a
route's `template.tsx` (which does):

```tsx
// app/(app)/template.tsx
"use client";
export default function Template({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  return <PageTransition transitionKey={pathname}>{children}</PageTransition>;
}
```

**Reduced motion:** all durations collapse to 0 — an instant swap.

**Use this when** wiring route-level transitions into any route group's
`template.tsx`.

---

## `components/byte`

### `<Byte />` — `byte.tsx`

```tsx
type ByteState = "idle" | "thinking" | "celebrating" | "hinting" | "error";
type ByteVariant = "chip" | "nav" | "hero";
type ByteSize = "sm" | "md" | "lg"; // 32px / 56px / 140px bounding box

interface ByteProps {
  state?: ByteState;      // default "idle"
  variant?: ByteVariant;  // default "chip" — which mascot asset
  size?: ByteSize;        // default "md"
  playToken?: string | number; // bump to replay celebrate/error flourish on a same-state repeat
  className?: string;
}
function Byte(props: ByteProps): JSX.Element
```

Byte the Beaver, in his five product states. Always the real Figma asset
(`public/figma/byte-mascot-{chip,nav,hero}.png`) via `next/image`, never
redrawn as inline SVG. Fully self-contained — no providers, no app-shell
dependency.

| State | Visual |
|---|---|
| `idle` | slow gentle vertical bob — the resting state |
| `thinking` | mascot holds still; a small three-dot pulse badge appears |
| `celebrating` | one-shot scale pop + brief cyan ring flash, plus a soft persistent cyan ring while the state holds. No bounce, no confetti. |
| `hinting` | wrapped in `GlowPulse` — this is the one place a screen should have that breathing glow active |
| `error` | a single 120ms rose shake (identical keyframes to `VerdictFeedback`'s fail state) plus a persistent soft rose ring while the state holds |

Transitioning **into** a state (from a different one) always plays its
one-shot flourish; mounting already in that state does not (no surprise
animation on first paint). Use `playToken` only for a genuine back-to-back
repeat (e.g. two wrong answers in a row while `state` stays `"error"`).

**Reduced motion:** the one-shot pop/shake and idle bob are skipped; the
persistent rings (celebrating/error) and `GlowPulse`'s static fallback
still render, so the state stays visually legible.

**Use this when** you need Byte's avatar anywhere a state needs conveying
— a hint affordance, a submit-result flourish, an empty state (see
`ByteMoment` below for the packaged version of that last one).

---

### `<ByteSpeech />` — `byte-speech.tsx`

```tsx
interface ByteSpeechProps {
  label?: string;    // default "Transverse says:" — matches Figma 15:10
  message: string;
  typeOn?: boolean;  // default true
  className?: string;
}
function ByteSpeech(props: ByteSpeechProps): JSX.Element
```

Byte's dialogue pill, matching Figma `15:10` exactly: cyan JetBrains Mono
label, body in `--tv-text-body`, rounded pill on the `--tv-surface-deep`
well. Body text types in via `TerminalType`. Renders only the pill — pair
it with `<Byte>` yourself:

```tsx
<div className="flex items-start gap-3">
  <Byte state="hinting" size="sm" />
  <ByteSpeech message='Not sure? The quiz is a fun way to warm up!' />
</div>
```

**Use this when** you need the bubble in a layout you control (inline in a
card, in a modal). Use `ByteDock` for the persistent floating case.

---

### `<ByteDock />` — `byte-dock.tsx`

```tsx
type ByteDockPosition = "bottom-right" | "bottom-left";

interface ByteDockProps {
  state?: ByteState;    // default "idle"
  message?: string;     // presence shows the bubble
  label?: string;
  position?: ByteDockPosition; // default "bottom-right"
  open?: boolean;        // controlled bubble visibility
  onOpenChange?: (open: boolean) => void;
  className?: string;
}
function ByteDock(props: ByteDockProps): JSX.Element
```

The persistent companion placement. `position: fixed`, `z-50`, owns its
own box — mount once near an app-shell layout:

```tsx
<ByteDock state="hinting" message='Stuck? Try breaking the loop invariant down first.' />
```

Uncontrolled by default: the bubble opens whenever `message` changes to a
new value; the viewer can dismiss it via the bubble's close button or by
clicking Byte. Pass `open`/`onOpenChange` to drive visibility yourself
(e.g. tie it to an SSE `hint.ready` event, or let the avatar re-open a
dismissed bubble).

**Use this when** wiring Byte into the authenticated app shell as a
persistent presence, distinct from `ByteSpeech` used inline in a specific
card/panel.

---

### `<ByteMoment />` — `byte-moment.tsx`

```tsx
type ByteMomentVariant =
  | "empty-dashboard"
  | "empty-roadmap"
  | "judge0-failed"
  | "hint-rate-limited"
  | "generic-empty"
  | "generic-error";

interface ByteMomentProps {
  variant: ByteMomentVariant;
  title?: string;        // overrides the variant's default copy
  description?: string;  // overrides the variant's default copy
  action?: ReactNode;     // a retry button, a CTA, whatever fits
  className?: string;
}
function ByteMoment(props: ByteMomentProps): JSX.Element
```

A real, human empty/error moment built around Byte instead of a grey box —
plan.md's explicit ask for exactly the first four situations below. Two
generic variants cover anything else needing the same treatment.

| Variant | Byte state | Default title |
|---|---|---|
| `empty-dashboard` | idle | "Nothing to show yet" |
| `empty-roadmap` | thinking | "No roadmap drawn yet" |
| `judge0-failed` | error | "That run didn't come back clean" |
| `hint-rate-limited` | thinking | "Slow down a second" |
| `generic-empty` | idle | "Nothing here yet" |
| `generic-error` | error | "Something went sideways" |

```tsx
<ByteMoment
  variant="judge0-failed"
  action={<Button onClick={retry}>Try again</Button>}
/>
```

**Use this when** a screen has nothing to show or something failed and the
placeholder is user-facing (not a dev-only error boundary) — the four
named spots from plan.md are `/dashboard` (empty state before onboarding
completes), `/roadmap` (before `POST /roadmap/generate` has run), the IDE
after a Judge0 run comes back malformed/errored, and a `429` from
`POST /practice/{id}/hint`.

---

### `<MotionByteShowcase />` — `demo.tsx`

```tsx
function MotionByteShowcase(): JSX.Element
```

Live reference for everything above, interactive where it matters (hover
the sweep cards, click to fire the unlock sequence, retype the terminal
text, fire pass/fail verdicts, cycle Byte's states). Not mounted by this
library — temporarily drop it on a scratch page to see it working, then
remove the import.

---

## Wiring this in — Wave 2

1. **App shell** — mount `<ByteDock />` once in whatever layout wraps the
   authenticated app (dashboard/roadmap/solve/practice/profile). Feed it
   `state`/`message` from wherever your hint/notification state already
   lives; it doesn't need a provider of its own.
2. **Route transitions** — add a `template.tsx` per route group wrapping
   `children` in `<PageTransition transitionKey={pathname}>`.
3. **Roadmap unlock** — replace ATLAS's local unlock animation with
   `<UnlockTransition unlocked={...}>` wrapping the existing node card
   markup. Drive `unlocked` off whatever local boolean you already flip on
   the `node.unlocked` SSE event — no other wiring required.
4. **Cards everywhere** — wrap dashboard/roadmap/problem-browser cards in
   `<SweepFrame>` for the default "alive" hover affordance. Don't also add
   `GlowPulse` to more than one of them at a time.
5. **IDE / practice verdicts** — wrap each test-case row and the
   submit-result row in `<VerdictFeedback>`.
6. **Onboarding quiz + hint UI** — `<Byte state="hinting" />` +
   `<ByteSpeech>` for in-context copy (e.g. the `429` rate-limit message
   plan.md §3.2 asks to surface on Byte, not as a red toast — pair with
   `<ByteMoment variant="hint-rate-limited">` if it needs a fuller block
   rather than an inline aside).
7. **Empty/error states** — swap any "nothing here" grey box or raw error
   message for the matching `<ByteMoment variant="...">`.
8. **Landing / auth grounds** — `<ScanlineGrid />` behind hero content
   only, per plan.md §1.4.

### Discipline checklist before shipping a screen against this library

- At most one `GlowPulse` (including the one inside `<Byte state="hinting">`) visible at a time.
- Never more than two things animating in a viewport at once.
- Every entrance ≤400ms (`DURATION.entranceMax` is the reference value).
- No parallax, no confetti, no auto-playing looping background video.
- If you write new `motion/react` code instead of using a primitive above,
  call `usePrefersReducedMotion()` yourself — it's not automatic outside
  this library.
