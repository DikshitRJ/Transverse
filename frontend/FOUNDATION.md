# FOUNDATION.md — the Wave-1 contract

Built by FOUNDRY. This is the exported surface every other agent codes
against: every component, hook, and API function you may import, with
signatures. If something you need isn't listed here, it doesn't exist yet —
add it in your own owned paths rather than reaching into `lib/api`,
`components/providers`, or `components/ui` to patch them (those are shared;
collisions between Wave-1 agents happen there first).

Verified working as of this writing: `npm run typecheck` (tsc --noEmit,
strict, zero `any`/`!`), `npm run lint` (zero errors/warnings), `npm test`
(vitest, 2/2 passing against the real MSW handlers), `npm run build`
(clean production build, standalone output, `/` prerenders statically).

---

## 0. Running it

```bash
npm install
cp .env.example .env.local   # NEXT_PUBLIC_API_MODE=mock by default
npm run dev
```

Mock mode needs no backend, no Docker, no OAuth apps. See `README.md` for
the mock/live switch and the SSE-buffering verification command.

---

## 1. Design tokens — `src/app/theme.css`

Every `--tv-*` variable from plan.md §1.1 is emitted verbatim (`:root`),
then wired into Tailwind v4 as first-class utilities via `@theme inline` —
**use `bg-tv-surface`, `text-tv-cyan`, `border-tv-border-cyan`, etc.
directly in `className`, never a raw hex, never `var(--tv-*)` inline
unless you're inside a `<style>`/`box-shadow` value that Tailwind has no
utility for.**

Radius tokens: `rounded-tv-chip` (4px), `rounded-tv-btn` (8px),
`rounded-tv-card` (12px), `rounded-tv-pill` (9999px).

Font utilities: `font-display` (Space Grotesk), `font-mono` (JetBrains
Mono), `font-body` (Inter, also the default via `html`/`body`).
`font-heading` is an alias for `font-display` (shadcn's Card/etc. use that
name internally — same font, use either).

Type scale utilities (Figma's exact scale, each pairs size + line-height
automatically): `text-display-1` (69/68), `text-display-2` (32/32),
`text-h1` (30/36), `text-h2` (24/32), `text-h3` (20/24), `text-h4` (18/28),
`text-body` (16/24), `text-sm` (14/20), `text-xs` (12/16).

**Signature-effect utility classes** (plan.md §1.1 — defined once, reuse
everywhere, never hand-roll a glow/gradient/glass treatment):

| Class | Effect |
|---|---|
| `glow-text-cyan` | `text-shadow: 0 0 10px rgba(0,255,255,.5)` |
| `glow-card-cyan` | `box-shadow: 0 0 15px rgba(0,255,255,.1)` |
| `glow-card-rose` | `box-shadow: 0 0 15px rgba(255,107,107,.1)` |
| `glow-chip-rose` | `box-shadow: 0 0 15px rgba(255,107,107,.3)` |
| `btn-gradient-primary` | cyan→blue gradient, `--tv-btn-ink` text, 4px radius, cyan drop-shadow — this is what `<Button>`'s default variant uses |
| `text-gradient-display` | the hero gradient-text treatment (`background-clip:text`) |
| `glass-panel` | `rgba(255,255,255,.03)` + `backdrop-blur(5px)` + 1px `--tv-border` |
| `glass-header` | `rgba(17,19,24,.8)` + `backdrop-blur(6px)` — `TopNav`/`Footer` use this |

Semantic extension (plan.md's only permitted new hue): `--tv-warning`
(`#FFD166`, for TLE/partial-pass), plus `--tv-success` (= `--tv-cyan`),
`--tv-error` (= `--tv-rose`), `--tv-locked` (`--tv-text-body` at 40%
opacity) — all wired as `bg-tv-warning`/`text-tv-success`/etc.

**shadcn semantic tokens are remapped onto these** (`globals.css`) —
`bg-card`, `border-border`, `text-muted-foreground`, `bg-primary`, `--ring`,
etc. all already resolve to the correct Transverse colors. Every shadcn
primitive (Dialog, Select, Sheet, Input focus rings...) is already
correctly themed with **zero per-component overrides needed**. The app is
dark-only — `:root` itself IS the dark theme, there is no `.dark` class
toggle and no `<ThemeProvider>`. Don't add one.

`prefers-reduced-motion: reduce` is handled globally in `globals.css`
(collapses all animation/transition durations to ~0) — you don't need to
re-implement this per-component, but any custom `motion/react` animation
you write should still check `useReducedMotion()` for anything that isn't
a pure CSS transition, per plan.md's motion rules.

## 2. Fonts

`src/app/layout.tsx` loads all three via `next/font/google`:
`--font-space-grotesk` (700 only), `--font-jetbrains-mono` (400/600/700),
`--font-inter` (400/600). **Deviation from plan.md:** Space Grotesk does
not have a 900/Black weight on Google Fonts (max is 700) — requesting it
throws a build error. Use `font-bold` (700), not `font-black`, on
`font-display` text; a stray `font-black` will fall back to the browser's
synthetic-bold heuristic, which is inconsistent across browsers. This
matches what Figma's `font-['Space_Grotesk:Bold']` actually references
even where the Tailwind class in Figma's reference code says `font-black`.

## 3. Figma assets — `public/figma/`

File key `5FBPyLzXWSFSv9ahnzTXFs`, nodes `15:10` (onboarding) and `61:46`
(landing). All bytes downloaded and committed — nothing references a
`figma.com/api/mcp/asset/...` URL. Reference as `/figma/<name>.{png,svg}`.

| File | Source | Use |
|---|---|---|
| `byte-mascot-nav.png` | both nodes, nav instance | Byte in `TopNav`/`Footer` (identical bytes in both Figma nodes — deduped to one file) |
| `byte-mascot-hero.png` | 61:46 hero "device" mockup | Byte, large, landing hero |
| `byte-mascot-chip.png` | 15:10 "Mascot Mentor" bubble | Byte's small avatar chip (Byte's dialogue bubble) |
| `github-mark.png` | 15:10 | GitHub logo, "Sync Past Experiences" card |
| `quiz-glyph.png` | 15:10 | glowing question-mark glyph, "Take a Quick Quiz" card |
| `icon-upload-cloud.svg` | 15:10 | upload-cloud decoration behind the sync card |
| `icon-quiz-monitor.svg` | 15:10 | monitor+question-mark decoration behind the quiz card |
| `icon-sync-repo.svg` | 15:10 | small icon inside "Connect Repo" button |
| `icon-lightning-quiz.svg` | 15:10 | small icon inside "Start Quiz" button |
| `icon-nav-menu.svg` | 61:46 nav | icon button next to "GET STARTED" in `TopNav` |
| `icon-solution-test-out.svg`, `icon-solution-roadmap.svg`, `icon-solution-diagnosis.svg` | 61:46 | the 3 "SOLUTION WE OFFER" icons |
| `icon-usp-gamified-unlock.svg`, `icon-usp-learn-solve.svg`, `icon-usp-closed-loop.svg` | 61:46 | the 3 USP-card icons |
| `icon-journey-evaluate.svg`, `-learn.svg`, `-master.svg`, `-practice.svg` | 61:46 | the 4 journey-grid node icons |

**Motion note:** `get_design_context` on node `15:10` flagged animated
nodes and instructed a `get_motion_context` follow-up — I did not pull
that (out of scope for asset extraction; it's about implementing the
actual keyframes, which is BYTE/THRESHOLD's job when they build those
screens for real). Call `get_motion_context` on `15:10` yourselves before
animating anything there.

## 4. UI kit — `src/components/ui/*`

shadcn/ui, style **`base-nova`** (their current default — built on
**`@base-ui/react`**, not Radix). This matters for one thing that will
trip you up if you've seen classic shadcn before:

> **There is no `asChild` prop.** Base UI's polymorphism is a `render`
> prop that takes a ReactElement: `<Button render={<Link href="/x" />}>Get
> Started</Button>`, not
> `<Button asChild><Link href="/x">...</Link></Button>`. `Badge` uses the
> same pattern internally. Every primitive below follows this.

Restyled to `theme.css` tokens (everything else inherits the semantic
remap automatically, see §1):

- **`Button`** (`@/components/ui/button`) — variants: `default` (the
  primary cyan→blue gradient CTA, `btn-gradient-primary`, 4px radius),
  `outline-cyan` (cyan 2px border, transparent fill — matches Figma's
  "Start Quiz" button), `outline`, `secondary`, `ghost`, `destructive`,
  `link`. Sizes: `default`/`xs`/`sm`/`lg`/`icon`/`icon-xs`/`icon-sm`/`icon-lg`.
  Base classes force `font-mono uppercase tracking-wide` — override
  `className` if a specific button needs sentence case.
- **`Card`, `CardHeader`, `CardTitle`, `CardDescription`, `CardAction`,
  `CardContent`, `CardFooter`** (`@/components/ui/card`) — `bg-tv-surface`
  equivalent + `border-tv-border` + `rounded-tv-card`, matches every
  Figma card exactly.
- **`Badge`, `badgeVariants`** (`@/components/ui/badge`) — variants:
  `default`, `secondary`, `destructive`, `outline`, `ghost`, `link`, plus
  the 4 **verdict/status semantics you'll actually want**: `success`
  (cyan), `warning` (amber), `error` (rose + `glow-chip-rose`), `locked`
  (dimmed, for locked roadmap nodes). Default radius is `rounded-tv-chip`
  (4px, matching Figma's labeled chips) — pass `className="rounded-tv-pill"`
  for a pill-shaped status badge instead.
- **`Input`, `Textarea`, `Select` (+`SelectGroup/Value/Trigger/Content/
  Label/Item/Separator/ScrollUpButton/ScrollDownButton`), `Label`**
  — unmodified beyond the global token remap; already correct.
- **`Dialog` (+`Trigger/Portal/Close/Overlay/Content/Header/Footer/Title/
  Description`), `Sheet`** (same sub-component set) — unmodified, correct.
- **`Tabs`, `TabsList`, `TabsTrigger`, `TabsContent`, `tabsListVariants`**
  — unmodified, correct.
- **`Toaster`** (`@/components/ui/sonner`) — already mounted once in
  `layout.tsx` with `theme="dark"` and Transverse-colored toast classes.
  **Don't mount another one.** To fire a toast from anywhere:
  `import { toast } from "sonner"; toast("message")` /
  `toast.success(...)` / `toast.error(...)` — that import is from the
  `sonner` package directly, not from our wrapper.
- **`Skeleton`, `Tooltip` (+`TooltipProvider`/`Trigger`/`Content`/`Arrow`),
  `Progress` (+`Track`/`Indicator`/`Label`/`Value`), `Separator`,
  `ScrollArea`/`ScrollBar`** — unmodified, correct. `TooltipProvider` is
  already mounted once in `layout.tsx` — don't nest another one.

## 5. Shell — `src/components/shell/*`

- **`TopNav`** (`@/components/shell/top-nav`) — `{ links?: {label,href}[],
  actions?: ReactNode, className? }`. Matches Figma `61:110`: glass panel,
  `rgba(0,255,255,.5)` border + glow, Byte + glowing wordmark, nav links,
  gradient CTA. Defaults to public (Analyze/Quiz + "Get Started" → `/onboarding`)
  — pass `actions` (e.g. an avatar/menu) for authenticated chrome instead of
  forking the component.
- **`Footer`** (`@/components/shell/footer`) — `{ className? }`. Matches
  Figma `61:128`: glass panel, mascot + wordmark, Documentation/Privacy/
  Terms/GitHub links, copyright.
- **`PageContainer`** (`@/components/shell/page-container`) —
  `{ children, className?, constrained?: boolean }`. The 1280px
  content-width wrapper every Figma frame is authored against
  (`constrained` defaults `true`; set `false` for full-bleed content like
  the IDE split view).

`layout.tsx` does **not** render `TopNav`/`Footer` globally — routes that
want the public shell compose it themselves (see `app/page.tsx`), since
authenticated app routes will want different nav content. Route guards and
the authenticated shell chrome are not built — that's route-owner territory
(THRESHOLD for guards).

## 6. API layer — `src/lib/api/`

**No raw `fetch` anywhere else in the app. Ever.** Import from
`@/lib/api` (barrel) or the specific module.

### `types.ts`

Hand-transcribed from the Go source (`backend/internal/models/{dto,roadmap,
evidence}.go` + the handler-local structs not in `models/`), NOT from
`Documentation/openapi.yaml`. Every block has a comment naming its exact Go
source. Read the file — it's the map of every DTO in the system — but the
highlights that will bite you if you assume `openapi.yaml` instead:

- `TestCaseResult.index`, not `test_case_index`.
- `ProblemPayload` has `tags[]`, `subtopic`, `avg_time_ms`, `contest_id`;
  has **no** `glicko_rating` or `status`.
- `SubmitResponse` also carries `session_status`, `question_count`,
  `carelessness_penalty`.
- `GET /problems/search`'s query param is **`q`**, not `query`
  (`ProblemSearchParams.q`), even though the JSON DTO field is `query`.
- Two *different* user shapes: `User` (raw, from `GET /auth/me`, `dna` is
  unparsed) vs `UserProfileResponse` (decoded, from `GET /user/profile`,
  `dna: LearningDNA`). Don't conflate them.
- **The real error envelope is `{"error": "message string"}`** — flat, not
  the `{error:{code,message}}` shape plan.md originally assumed. Verified
  against `handlers/helpers.go:writeError`. `client.ts` already normalizes
  this for you into `ApiError` — you should never touch the envelope
  directly.
- SSE frames are `{type, job_id, data}`, always sent as a bare `data:`
  line (no named `event:` field except the literal `connected` handshake)
  — see §8.
- The 5 evidence endpoints are typed and mocked, but **not mounted on
  `main` as of this writing** — KEYSTONE mounts them per plan.md §9.1. If
  you build against them and they 404 against a live backend, that's why;
  they work fine in mock mode already.

### `client.ts`

- `apiFetch<T>(path, options?): Promise<T>` — the core function. `options`
  extends `RequestInit` with `json` (object, auto-stringified),
  `query` (object → querystring), `skipAuthRefresh`, `skipPrefix` (only
  `getHealth()` needs this — see below).
- `apiGet<T>(path, options?)`, `apiPost<T>(path, json?, options?)`,
  `apiDelete<T>(path, options?)` — sugar over `apiFetch`.
- `class ApiError extends Error { status: number; body: unknown }` — every
  non-2xx response throws this. `err.status === 401` after a failed silent
  refresh means the session is genuinely dead.
- Automatically: prefixes `/api/v1`, attaches `Authorization: Bearer
  <token>` from the token store, retries idempotent GETs on network
  failure (not on HTTP error status) with 300/900ms backoff, and on a 401
  does one single-flight silent refresh (`POST /api/auth/refresh`) then
  retries the original request once.

### `endpoints.ts`

One named function per backend route — see the file for the full list (34
functions covering all 34 routes). The two you must use instead of the raw
per-route calls:

```ts
// The two-step execute -> poll -> submit handshake (plan.md §3.1).
// NEVER call executeCode + submitPracticeAnswer yourself — use this.
submitSolution({ sessionId, problemId?, languageId, sourceCode, customStdin?, timeTakenMs, pollOptions? }): Promise<SubmitResponse>

// POST /execute/batch ("Run all test cases") is separate and unrelated —
// executeBatch()/pollVerdict() are exposed individually for that path.
```

```ts
// Hint resolution (plan.md §3.2) — race the SSE event against the poll fallback:
const { job_id } = await requestHint(sessionId, { hint_level });
const job = await Promise.race([
  new Promise<Job>((resolve) => {
    // one-shot: use useTransverseEvent("hint.ready", ...) in a component instead of this raw pattern
  }),
  pollHintJob(job_id),
]);
```
In practice, do the SSE half via the `useTransverseEvent` hook (§8) in your
component and call `pollHintJob` as the fallback in the same effect —
`pollHintJob` alone is already correct and complete if you don't want to
bother with the SSE race at all, just slower to resolve (2s poll interval,
45s timeout) than winning the SSE race would be. `429` from
`requestHint` means the backend's rate limit — the real handler string is
`"rate limit"`; surface it in-context on Byte, not as a red error toast
(plan.md §3.2 — not currently special-cased in `apiFetch`, so catch it at
the call site: `if (err instanceof ApiError && err.status === 429)`).

`oauthRedirectPath(provider)` returns a path string for `<a href>` /
`window.location.href` — it is **not** a fetch call, don't call it through
`apiFetch`.

`getHealth()` is the one endpoint outside `/api/v1` (`GET /health` is
mounted on the backend's root router, not under `/api/v1` — verified
against `cmd/server/main.go`). `next.config.ts` has a matching second
rewrite rule for `/health`.

### `index.ts`

Barrel: `import { ... } from "@/lib/api"` gets everything from `types.ts` +
`client.ts` + `endpoints.ts` in one import.

## 7. Auth — `src/components/providers/auth-provider.tsx`, `src/lib/auth/`

```ts
useAuth(): {
  user: User | null | undefined;   // undefined = initial silent-refresh still pending
  isAuthenticated: boolean;
  isLoading: boolean;               // true only during that initial check
  completeOAuthCallback: (p: { accessToken: string; refreshToken: string }) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>; // re-fetch GET /auth/me
}
```

**Architecture** (plan.md §5.3 — access token in memory, refresh token in
an httpOnly cookie): the access token lives only in
`src/lib/auth/token-store.ts` (a plain module singleton — deliberately not
React state, not `localStorage`; the app renders untrusted scraped HTML,
so anything an XSS could read is a real risk). The refresh token never
reaches client JS — it's set/read/rotated by 3 Next.js Route Handlers:

- `POST /api/auth/session` — `{refresh_token}` → sets the httpOnly cookie.
  Call this once, right after `/auth/callback` captures tokens from the URL.
- `POST /api/auth/refresh` — reads the cookie server-side, calls the
  backend's `POST /auth/refresh` directly (server-to-server, not through
  the `/api/v1` rewrite), rotates the cookie to the new refresh token,
  returns `{access_token, expires_in}` only. This is what `client.ts`'s
  silent-refresh and `AuthProvider`'s initial-load check both call.
- `POST /api/auth/logout` — revokes server-side, always clears the cookie
  locally even on backend failure.

### ⚠️ What THRESHOLD needs to know — the OAuth callback contract is an assumption, not a confirmed spec

Backend change #4 in plan.md §9.1 (KEYSTONE, not yet landed as of this
writing) says the OAuth callback "gains a browser path: 302 to
`${FRONTEND_ORIGIN}/auth/callback` instead of writing JSON" — but doesn't
specify how the tokens themselves cross that redirect. **I designed the
frontend half assuming tokens arrive as query params** —
`/auth/callback?access_token=...&refresh_token=...&expires_in=...` — the
simplest thing that works for a redirect-based flow, and what the mock
`GET /api/v1/auth/oauth/:provider/redirect` handler actually does (302s
straight to `/auth/callback` with working mock tokens — try it, sign-in is
fully clickable in mock mode with zero real OAuth app needed).

**Build `/auth/callback` against that contract**:
```ts
// THRESHOLD's /auth/callback page
const params = new URLSearchParams(window.location.search);
await completeOAuthCallback({
  accessToken: params.get("access_token")!,
  refreshToken: params.get("refresh_token")!,
});
router.replace("/onboarding"); // or wherever; also strips tokens from history
```
**But confirm this against KEYSTONE's actual implementation before
shipping it against a live backend** — if KEYSTONE instead sets a
temporary backend-origin cookie and redirects with just a one-time code, or
any other shape, `completeOAuthCallback`'s signature
(`{accessToken, refreshToken}`) is still right, just the *source* of those
two strings on the callback page changes. This is the one piece of the
auth story that's a documented assumption rather than a verified fact —
everything else in this section is confirmed against the real Go source.

## 8. Realtime — `src/components/providers/sse-provider.tsx`, `src/lib/realtime/sse-client.ts`

```ts
useTransverseEvents(): {
  status: "idle" | "connecting" | "open" | "closed";
  subscribe: <T>(type: TransverseEventType, listener: (e: TransverseEvent<T>) => void) => () => void;
  subscribeAll: (listener: (e: TransverseEvent) => void) => () => void;
}

// sugar — auto subscribe/unsubscribe for the component's lifetime:
useTransverseEvent<T>(type: TransverseEventType, listener: (e: TransverseEvent<T>) => void): void
```

`TransverseEventType` = `"job.completed" | "job.failed" | "node.unlocked" |
"roadmap.updated" | "hint.ready"`.

**Not `EventSource`.** The stream is read via `fetch` +
`ReadableStream` (`lib/realtime/sse-client.ts`) because native
`EventSource` cannot set the `Authorization` header
`middleware.Auth` requires (verified — no cookie fallback exists). Auto
single global connection (started/stopped by `SSEProvider`, already
mounted in `AppProviders`), auto-reconnects with exponential backoff
(1s → 30s cap), auto-restarts on access-token change (login/refresh/logout).

**Reality check on what's actually live:** as of this writing the real
backend (`jobs/worker.go`) only ever publishes `job.completed` and
`job.failed`. `node.unlocked`, `roadmap.updated`, and `hint.ready` are part
of the intended design (plan.md §2) but nothing server-side emits them
yet. **The mock stream does fire all 5** (`hint.ready` when a mock hint job
finishes; `node.unlocked`+`roadmap.updated` when you call
`completeRoadmapNode`) so Wave-1 UI can be built and demoed against the
full intended behavior today — just know that `node.unlocked`/
`roadmap.updated`/`hint.ready` won't fire against a live backend until
someone (KEYSTONE, or a later pass) adds the `PublishEvent` calls
server-side. Build the UI against the mock now; it's not extra throwaway
work, it's exactly what real events will look like once wired.

`SSEProvider` already wires baseline cache invalidation
(`roadmap.updated`/`node.unlocked` → invalidates `["roadmap"]`;
`job.*`/`hint.ready` → invalidates `["jobs", job_id]`). Use
`useTransverseEvent` directly for anything beyond that (driving the unlock
animation, resolving a hint promise, etc).

## 9. TanStack Query — `src/components/providers/query-provider.tsx`

Already mounted in `AppProviders`. Defaults: `staleTime: 30s`,
`refetchOnWindowFocus: false`, no retry on 4xx `ApiError`s (2 retries on
everything else), no mutation retry. Use TanStack Query key convention
`["roadmap"]`, `["jobs", id]`, etc. to line up with the SSE invalidation in
§8 — if you invent a new key that a realtime event should invalidate, add
the case in `sse-provider.tsx`, don't duplicate the wiring elsewhere.

## 10. Mock layer — `src/mocks/`

`NEXT_PUBLIC_API_MODE=mock` runs MSW **both** client-side
(`MockProvider` → `mocks/browser.ts`, for TanStack Query hooks in client
components) **and** server-side (`src/instrumentation.ts` → `mocks/
server.ts`, for RSC data fetches and the `/api/auth/*` Route Handlers'
calls to `BACKEND_URL`) — both load the exact same `mocks/handlers.ts`
array, so behavior is identical no matter where a request originates.

`handlers.ts` answers all 34 routes, auth-enforced exactly like the real
`middleware.Auth` (missing/malformed `Authorization` header → real 401
envelope). Get a token: visit `/signin` (once THRESHOLD builds it) and
click a provider — the mock redirect short-circuits straight to
`/auth/callback` with working tokens, no real OAuth needed.

Fixtures (`src/mocks/fixtures/`):

- **`problems.ts`** — 40 `ProblemPayload`s (10 hand-authored "hero"
  problems with full HTML statements + real test cases — Two Sum, Valid
  Parentheses, Binary Search, Merge Two Sorted Lists, Longest Substring
  w/o Repeating Chars, Course Schedule, Climbing Stairs, Kth Largest
  Element, A+B Problem, Watermelon — plus 30 generated fillers spread
  across the real curriculum topics). `templates` map keys are the
  **real backend's 8 languages**: `py, cpp, java, js, go, rust, c, kt`
  (mirrors `templates.GenerateTemplates` exactly — note this is NOT
  `typescript`/`csharp`, a real gotcha if you assumed otherwise).
  Statements are plain HTML strings — treat as **untrusted** even though
  it's fixture data; build the sanitization pipeline (`react-markdown` +
  `rehype-raw` + `rehype-sanitize`) as if it were live scraped HTML,
  because in live mode it will be.
- **`languages.ts`** — `LANGUAGES: LanguageMeta[]` (`key, label, judge0Id,
  monacoId`) for the 8 languages above, plus `languageByJudge0Id()`. FORGE:
  build the language switcher off this array, not a hand-rolled list.
- **`roadmap.ts`** — a full 3-section `RoadmapCurrentResponse`: one ACTIVE
  section with 6 subsections across real topic ids (`foundations` through
  `binary-search`, mastered → in_progress → locked, each with real
  tutorials + questions pulled from `problems.ts` by topic), 2 LOCKED
  upcoming-section previews. `roadmapState` is mutable — `completeRoadmapNode`
  actually unlocks the next node and fires SSE events, so the roadmap
  screen genuinely progresses across a demo session.
- **`user.ts`** — `userProfile` (`UserProfileResponse`, decoded
  `LearningDNA`) and `rawUser` (`User`, what `GET /auth/me` returns).
- **`verdicts.ts`** — Judge0 verdict state machine keyed by
  execution token: every 4th `POST /execute` cycles through Accepted →
  Wrong Answer → Time Limit Exceeded → Compilation Error (2 polls of
  queue/processing before settling, regardless of your poll interval).
  `buildBatchResult(n)` for `/execute/batch`.
- **`sessions.ts`** — in-memory adaptive-practice state machine (theta
  nudges on correct/incorrect, next-problem rotation respecting session
  scope, real per-topic close-out breakdown).
- **`jobs.ts`** — hint job lifecycle (queued → running@800ms → done@2.5s
  with real hint text), fires `hint.ready` on the mock event bus when done.
- **`evidence.ts`** — upload/connector lifecycle (pending → processing →
  done on a timer).
- **`event-bus.ts`** — the pub/sub the mock SSE stream handler reads;
  call `emitMockEvent({type, job_id, data})` from a new fixture if you add
  a scenario that should push a realtime event.

**Adding a new mock scenario**: add/extend a fixture module, wire it into
the matching handler in `handlers.ts`. Keep the shape typed against
`lib/api/types.ts` — a mismatch there is a compile error, which is the
whole point (plan.md §5.4: "the fixtures ARE the transcribed Go DTOs, so a
shape drift shows up as a type error").

## 11. `next.config.ts`

`output: "standalone"`; rewrites `/api/v1/:path*` → `${BACKEND_URL}/api/v1/:path*`
and `/health` → `${BACKEND_URL}/health` (the latter needed because `/health`
lives outside `/api/v1` on the backend). `outputFileTracingRoot` is pinned
explicitly to this app directory — the repo root's own `package.json`/
lockfile (unrelated root-level tooling) would otherwise get picked up by
Next's workspace-root inference and pollute the standalone trace.

See README.md for the `curl -N` SSE-buffering verification command — do
this once a live backend exists, before assuming the realtime layer works
end-to-end.

## 12. Testing

Vitest + Testing Library + jsdom, configured (`vitest.config.ts`,
`vitest.setup.ts`) with the **same MSW server** as mock mode
(`src/mocks/server.ts`) auto-started/reset/closed around every suite — any
test that calls `@/lib/api` functions already has a working, realistic
backend without further setup. See `src/lib/api/__tests__/client.test.ts`
for the pattern (asserts a real `ApiError` shape, not just "it throws").

`npm test` (single run) / `npm run test:watch`.

## 13. What's explicitly NOT built here (Wave-1/KEYSTONE territory)

- No pages beyond the placeholder `/` (BEACON replaces it) — no
  `/signin`, `/onboarding/*`, `/dashboard`, `/roadmap`, `/solve/*`,
  `/practice/*`, `/profile`, `/settings`. Route guards are not built (no
  middleware.ts redirecting unauthenticated users) — THRESHOLD owns that.
- No Monaco integration, no Recharts usage, no `components/motion/` or
  `components/byte/` library (BYTE owns the shared motion vocabulary —
  cyan sweep, glow pulse, unlock sequence, terminal type-on, scanline
  grid — plan.md §1.4; don't invent your own animation primitives, wait
  for/consume BYTE's).
- `react-markdown`/`rehype-sanitize` are installed but no rendering
  pipeline is wired up yet — whoever renders the first problem statement
  or tutorial body owns building that (sanitize scraped/mock HTML before
  render, always).
- Evidence routes are typed + mocked but not confirmed live (§6, §7 sidebar).
- No Dockerfile/compose wiring (HARNESS, Wave-2).
- `getUserHistory()` mock currently returns `[]` — the shape
  (`PracticeSession[]`) is real and correct, just not populated with
  sample rows. Extend `mocks/handlers.ts`'s `*/api/v1/user/history` handler
  if you need non-empty history to build PRISM's profile screen against.
