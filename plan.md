# Transverse — Frontend Implementation Plan

> **Status:** Decisions settled (§9). Awaiting go-ahead to spawn the fleet. No code written yet.
> **Target:** `./frontend` — Next.js 15 (App Router) + TypeScript, on git branch `frontend`.
> **Sources of truth:** Figma `Project_Final` (file `5FBPyLzXWSFSv9ahnzTXFs`) · `Documentation/openapi.yaml` · the Go handlers in `backend/internal/**` · `Documentation/Pitchdeck.pdf` · Notion "Transverse Project Documentation".
> **Constraints honoured:** no git commits · subagents isolated via `git worktree` · everything containerised and joined to the existing compose stack.

---

## 0. What I found first (read this before the rest)

Three findings materially shape the plan. Each has a decision attached at the end of this document.

**0.1 — The Figma file contains exactly two screens.**
`Page 1` holds `15:10` (Onboarding chooser, 1280×1089) and `61:46` (Landing page, 1280×3419). That is the entire design file. The dashboard, roadmap, IDE, quiz, practice, profile and sign-in screens do not exist in Figma and must be designed from scratch against the extracted token set. Roughly **2 pixel-traced screens vs. ~15 net-new screens**. This is the single biggest scope fact in the plan.

**0.2 — `Documentation/openapi.yaml` has drifted from the actual Go DTOs.** The spec is a good map of *which* endpoints exist, but it is wrong about *shapes*. Verified mismatches:

| openapi.yaml says | `backend/internal/models/dto.go` actually says |
|---|---|
| `TestCaseResult.test_case_index` | `index` |
| `ProblemPayload.glicko_rating`, `.status` (`SOLVED`/`UNSOLVED`) | neither field exists |
| `ProblemPayload` lacks tags/subtopic | has `tags[]`, `subtopic`, `avg_time_ms`, `contest_id` |
| `SubmitResponse` = verdict + theta + next | also has `session_status`, `question_count`, `carelessness_penalty` |
| walkthrough: submit takes `source_code` + `language_id` | **submit takes `judge0_token`** — you must `POST /execute` first, then submit the returned token |
| spec omits them entirely | `GET /practice/similar`, `GET /practice/topics`, `GET /jobs/{id}`, `GET /user/profile`, `GET /user/history`, `POST /admin/*` all exist and are routed |

→ **TypeScript types will be derived from the Go structs, not from `openapi.yaml`.** A single `frontend/src/lib/api/types.ts` is hand-transcribed from `models/dto.go`, `models/roadmap.go` and `models/evidence.go`, with a checked-in note pointing at the Go file each block mirrors.

**0.3 — Four backend gaps sit directly under designed UI.**

1. **Evidence upload / repo-connect is not routed.** `handlers/evidence_handler.go` is fully written (presigned upload, confirm, GitHub/LeetCode/Codeforces connectors) but is never mounted in `cmd/server/main.go`. The Figma onboarding screen's entire left card — *"SYNC PAST EXPERIENCES → Connect Repo"* — has no reachable endpoint. Two sub-problems: it reads `r.Context().Value("user_id")` while the auth middleware writes key `UserIDKey("userID")`, so it would panic on the type assert as written; and `backend/docker-compose.yml` has **no MinIO service** (MinIO only appears in `docker-compose.test.yml`), so presigned uploads have nowhere to go.
2. **No CORS middleware anywhere in the chi router.** A browser on `:3000` calling `:8080` fails preflight. *Mitigated frontend-side* — see §5.
3. **The OAuth callback returns raw JSON.** `authH.Callback` ends in `issueTokens`, which writes `{access_token, refresh_token, expires_in}` straight to the response body. The browser lands on the backend URL and sees JSON; there is no redirect back to the frontend, so a real sign-in cannot complete in a browser.
4. **There is no diagnostic-quiz endpoint.** The onboarding right card — *"TAKE A QUICK QUIZ"* — has no dedicated route. The closest existing engine is the adaptive `/practice/*` session.

None of these block *building* the UI (the mock layer in §5 covers all four). Per **Decision A**, gaps 1–3 plus MinIO are fixed by a scoped backend change-set — see §9.1. Gap 4 is resolved by reusing `/practice/*` — see §9.2.

---

## 1. Design system — extracted, exact, frozen

Pulled from Figma nodes `15:10`, `61:110`, `61:72`. The file defines **no Figma variables**, so every value below is a raw hex read out of the design and is now the canonical token set. **These values do not get "improved".**

### 1.1 Colour tokens → `frontend/src/app/theme.css`

```css
:root {
  /* ground */
  --tv-bg:            #0A0A12;  /* landing / deep app ground */
  --tv-bg-page:       #111318;  /* onboarding + inner page ground */
  --tv-surface-deep:  #0C0E12;  /* mascot chip, inset wells */
  --tv-surface:       #161B22;  /* cards */
  --tv-surface-2:     #282A2E;  /* avatar / icon wells */
  --tv-glass:         rgba(255,255,255,0.03);   /* + backdrop-blur(5px) */
  --tv-header:        rgba(17,19,24,0.8);       /* + backdrop-blur(6px) */

  /* line */
  --tv-border:        #30363D;
  --tv-border-muted:  #3A494B;
  --tv-border-cyan:   rgba(0,242,255,0.3);
  --tv-border-nav:    rgba(0,255,255,0.5);

  /* accent */
  --tv-cyan:          #00F2FF;  /* primary action */
  --tv-cyan-pure:     #00FFFF;  /* glow + gradient stop + logo */
  --tv-blue:          #0055FF;  /* gradient end */
  --tv-cyan-ink:      #006A71;  /* text ON a cyan fill */
  --tv-btn-ink:       #0A0A12;  /* text ON the cyan→blue gradient */
  --tv-rose:          #FF6B6B;  /* "problems" / error / regression */

  /* type */
  --tv-text-hi:       #E2E2E8;
  --tv-text-hi-alt:   #E5E7EB;
  --tv-text-body:     #B9CACB;
  --tv-text-nav:      #D1D5DB;

  /* radius */
  --tv-r-chip: 4px;  --tv-r-btn: 8px;  --tv-r-card: 12px;  --tv-r-pill: 9999px;
}
```

**Signature effects** (used consistently, never invented per-page):
- Cyan text glow — `text-shadow: 0 0 10px rgba(0,255,255,0.5)`
- Cyan card glow — `box-shadow: 0 0 15px rgba(0,255,255,0.1)`
- Rose card glow — `box-shadow: 0 0 15px rgba(255,107,107,0.1)`; rose chip — `0 0 15px rgba(255,107,107,0.3)`
- Primary button — `linear-gradient(90deg, #00FFFF, #0055FF)`, ink `#0A0A12`, radius 4px, `drop-shadow(0 0 7.5px rgba(0,255,255,0.5))`
- Display gradient text — `linear-gradient(90deg, #D9E9EA 0%, #00FFFF 96.635%)`, `background-clip: text`
- Glass panel — `background: rgba(255,255,255,0.03)` + `backdrop-blur(5px)` + 1px border

**Semantic extension** (net-new screens need states Figma never drew — derived from the frozen palette, no new hues):
`success` = `--tv-cyan` · `error`/`wrong` = `--tv-rose` · `warning` = `#FFD166` (only permitted addition, for TLE/partial-pass verdicts) · `locked` = `--tv-text-body` at 40% on `--tv-surface`.

### 1.2 Typography

| Role | Family | Weights | Notes |
|---|---|---|---|
| Display / headings / logo | **Space Grotesk** | 700, 900 | uppercase; tracking `-4.8px` @ 69px, `-1.2px` @ 24–32px |
| UI / code / labels / mascot voice | **JetBrains Mono** | 400, 600, 700 | all buttons, badges, terminal, editor |
| Body copy | **Inter** | 400, 600 | card paragraphs, nav links |

All three ship via `next/font/google` with `display: swap` and CSS-variable binding. Scale (from Figma): `69/68`, `32/32`, `30/36`, `24/32`, `20/24`, `18/28`, `16/24`, `14/20`, `12/16`.

### 1.3 Assets

Figma asset URLs **expire in ~7 days**, so every icon and image is downloaded and committed to `frontend/public/figma/` during the foundation phase — nothing references a `figma.com/api/mcp/asset/...` URL at runtime. Assets: Byte the Beaver mascot (nav + hero + chip variants), GitHub mark, quiz glyph, the four journey-node vector icons, three USP icons, three solution icons, upload-cloud and monitor decorations, footer mark.

**Icons are never hand-drawn as inline SVG paths** — the exported vectors are the only correct source.

### 1.4 Motion language (creative liberty, disciplined)

`get_design_context` reports the onboarding node contains **animated nodes**; the real keyframes get pulled via `get_motion_context` before any of it is authored. On top of that, one shared vocabulary — defined once in `components/motion/`, consumed everywhere, so the app reads as one product rather than eight agents' taste:

- **Cyan sweep** — a 1px cyan line traversing a card edge on hover/focus. The app's default "alive" gesture.
- **Glow pulse** — 2.4s breathing on the active roadmap node and Byte's chip only. Rationed deliberately.
- **Unlock** — the signature moment: node ring completes → glow flare → lock icon dissolves → card lifts. Fired by the `node.unlocked` SSE event.
- **Terminal type-on** — Byte's dialogue and verdict text type in at ~28ms/char, `prefers-reduced-motion` collapses it to instant.
- **Scanline grid** — a very low-opacity animated grid on hero + auth pages only.
- **Verdict** — pass: cyan ripple out from the test row; fail: single 120ms rose shake, no bounce.

Hard rules against slop: no parallax on scroll, no confetti, no more than two things animating in a viewport at once, every entrance ≤ 400ms, every animation honours `prefers-reduced-motion`, no auto-playing looping background video.

---

## 2. Route map — every screen, every endpoint

`✓ Figma` = pixel-trace an existing frame. `✚ New` = design from the frozen token set.

### Public

| # | Route | Source | Backend |
|---|---|---|---|
| 1 | `/` Landing | ✓ Figma `61:46` | none (+ `GET /health` for the status dot) |
| 2 | `/signin` | ✚ New | `GET /auth/oauth/{github\|google}/redirect` |
| 3 | `/auth/callback` | ✚ New | token capture → `POST /auth/refresh` (see §5.3) |

### Onboarding — the Phase-1 "deep skill verification" funnel

| # | Route | Source | Backend |
|---|---|---|---|
| 4 | `/onboarding` chooser | ✓ Figma `15:10` | none |
| 5 | `/onboarding/sync` evidence + repo connect | ✚ New | `POST /evidence/upload-url`, `POST /evidence/{id}/confirm`, `POST /evidence/{github\|leetcode\|codeforces}` — *routed by KEYSTONE, §9.1* |
| 6 | `/onboarding/quiz` diagnostic | ✚ New | capped adaptive session: `/practice/start` · `/execute` · `/practice/submit` · `/practice/skip` · `/practice/close` → breakdown feeds `/roadmap/generate` |
| 7 | `/onboarding/results` hypotheses → roadmap | ✚ New | `POST /roadmap/generate` with `confirmed_hypotheses` / `debunked_hypotheses` |

### Core application

| # | Route | Source | Backend |
|---|---|---|---|
| 8 | `/dashboard` | ✚ New | `GET /user/profile`, `GET /practice/topics`, `GET /user/history`, `GET /roadmap` |
| 9 | `/roadmap` active section + locked previews | ✚ New | `GET /roadmap` |
| 10 | `/roadmap/node/[nodeId]` subsection detail | ✚ New | `POST /roadmap/nodes/{id}/complete`, `POST /roadmap/nodes/{id}/test-out` |
| 11 | `/tutorial/[id]` reader | ✚ New | from roadmap payload; complete → node complete |
| 12 | `/problems` browse + scrape-by-URL | ✚ New | `GET /problems/search`, `POST /problems/scrape` |
| 13 | `/solve/[problemId]` **the IDE** | ✚ New | `POST /execute`, `GET /execute/{token}`, `POST /execute/batch` |
| 14 | `/practice` adaptive loop | ✚ New | `/practice/*`, `POST /practice/{id}/hint` → `GET /jobs/{id}` |
| 15 | `/practice/session/[id]` summary + Learning DNA | ✚ New | `POST /practice/close`, `GET /practice/session/{id}`, `GET /practice/{id}/error-analysis` |
| 16 | `/profile` | ✚ New | `GET /user/profile`, `GET /user/history` |
| 17 | `/settings` | ✚ New | `POST /auth/logout`, connector management |

### Cross-cutting
- **SSE** `GET /events/stream` — one global subscription in a provider; routes `job.completed`, `job.failed`, `node.unlocked`, `roadmap.updated`, `hint.ready` to toasts, cache invalidation and the unlock animation.
- **Shell** — top nav (Figma `61:110`), footer (`61:128`), command palette (`⌘K`), Byte dock.

---

## 3. Two flows worth spelling out

**3.1 Submit is a two-step handshake.** The walkthrough doc is wrong here; the handler is authoritative (`handlers/practice_handler.go`, `submitPayload`):

```
POST /execute        { language_id, source_code, custom_stdin }  → { judge0_token }
GET  /execute/{token}   poll until is_done (status_id > 2), 600ms backoff, 30s ceiling
POST /practice/submit   { session_id, problem_id, judge0_token, time_taken_ms }
                     → { is_correct, verdict, theta_before, theta_after, next_problem, ... }
```
`POST /execute/batch` is the separate "Run all test cases" path and does **not** feed `/practice/submit`. The UI surfaces both: **Run** (batch, against sample cases) and **Submit** (single → token → session).

**3.2 Hints are asynchronous.** `POST /practice/{id}/hint` returns `202 { job_id }`. The UI shows a pending state, then resolves on whichever arrives first: the `hint.ready` SSE event, or a `GET /jobs/{id}` poll as fallback. `429` surfaces as an in-context rate-limit message on Byte, not a red error toast.

---

## 4. Stack

| Concern | Choice | Why |
|---|---|---|
| Framework | **Next.js 15**, App Router, TS strict | asked for; RSC for roadmap/profile reads, client islands for IDE |
| Styling | **Tailwind CSS v4** (CSS-first `@theme`) | tokens live in CSS, Figma output is already Tailwind — least translation loss |
| Components | **shadcn/ui** | `shadcn@^4.19.0` is already in the repo's root `package.json` — following the decision already made |
| Server state | **TanStack Query v5** | polling, retries, SSE-driven invalidation |
| Editor | **Monaco** (`@monaco-editor/react`) | 8 languages required; VS Code parity beats CodeMirror here |
| Animation | **Motion** (`motion/react`) | layout animations for the unlock moment |
| Charts | **Recharts** | mastery radar + rating history (built under the `dataviz` skill) |
| Mocking | **MSW v2** | see §5 |
| Markdown | `react-markdown` + `rehype-sanitize` | problem statements arrive as **untrusted HTML from scraped LeetCode/Codeforces** — sanitising is mandatory, not optional |
| Tests | Vitest + Testing Library; Playwright smoke | |

---

## 5. Backend integration — how it actually connects

**5.1 CORS is sidestepped, not fought.** All browser traffic goes to same-origin `/api/v1/*` and Next.js rewrites it to the backend server-side. No preflight, no backend change needed:

```ts
// next.config.ts
async rewrites() {
  return [{ source: '/api/v1/:path*', destination: `${process.env.BACKEND_URL}/api/v1/:path*` }];
}
```
`BACKEND_URL` = `http://backend:8080` in compose, `http://localhost:8080` locally. **SSE passes through this rewrite correctly** provided the route is not wrapped in a buffering handler — verified as an explicit acceptance test, since a buffered SSE stream is a silent failure mode.

**5.2 Typed client.** `lib/api/client.ts` — one `fetch` wrapper: attaches the bearer token, retries idempotent GETs, normalises the `{error:{code,message}}` envelope, and throws a typed `ApiError`. Every endpoint gets a thin named function; no raw `fetch` anywhere else in the app.

**5.3 Auth — real OAuth only (Decision C).** No dev-bypass affordance ships in the UI. Access token in memory + refresh token in an `httpOnly` cookie set by a Next Route Handler (never `localStorage` — the app renders untrusted scraped HTML). `/signin` links to `GET /auth/oauth/{provider}/redirect`; KEYSTONE makes the callback redirect to `/auth/callback?...`, which hands the payload to the Route Handler and forwards to `/onboarding` or `/dashboard`. Silent refresh on 401, single-flight, logout revokes via `POST /auth/logout`.

> **You need to register two OAuth apps before sign-in can be exercised end-to-end** — GitHub and Google — with callback URLs `http://localhost:8080/api/v1/auth/oauth/github/callback` and `.../google/callback`, and their client ID/secret in `backend/.env`. Until those exist, agents build the sign-in UI against mocks; every *other* screen is unaffected, because `BYPASS_AUTH=true` in `backend/docker-compose.yml` means protected endpoints answer without a token during development.

**5.4 Mock layer — the thing that unblocks everything.** MSW handlers for all 22 endpoints, with fixtures generated from the real Go DTOs (including a full 3-section roadmap, ~40 problems with 8-language templates, Judge0 verdict sequences, and a scripted SSE event stream). Toggled by `NEXT_PUBLIC_API_MODE=mock|live`.

This is what makes ~15 net-new screens buildable in parallel by eight agents while Docker is unavailable on this machine and while `/evidence/*` stays unrouted. Every screen is built and demoable against mocks, then flipped to `live` per-endpoint as the backend is confirmed. It is also the contract test: the fixtures *are* the transcribed Go DTOs, so a shape drift shows up as a type error rather than a runtime blank screen.

---

## 6. Docker

New `frontend/Dockerfile` — multi-stage, `output: 'standalone'`, non-root, ~180MB final:

```
deps  (node:22-alpine, npm ci)
  ↓
build (npm run build, standalone output)
  ↓
run   (node:22-alpine, USER node, EXPOSE 3000, HEALTHCHECK → /api/health)
```

Added to `backend/docker-compose.yml` alongside the existing `db` / `redis` / `judge0-server` / `judge0-workers` / `backend` services:

```yaml
  frontend:
    build: { context: ../frontend, dockerfile: Dockerfile }
    restart: unless-stopped
    depends_on: { backend: { condition: service_started } }
    environment:
      BACKEND_URL: http://backend:8080
      NEXT_PUBLIC_API_MODE: live
      NODE_ENV: production
    ports: ["3000:3000"]
```

Also delivered: `frontend/.dockerignore`, `frontend/.env.example`, a `compose.override.yml` for hot-reload dev, and a **`minio` service** plus `MINIO_*` env on `backend` (Decision A.4) — the evidence-upload flow requires it and it exists today only in `docker-compose.test.yml`.

⚠️ **Docker is not installed on this machine** (`docker` is absent from PATH and Docker Desktop is not at its default location). Per **Decision D** the compose integration is written now and reviewed line-by-line, but `docker compose build` is **not run here** — live validation happens once you've installed Docker Desktop. Everything else proceeds on the mock layer, so this blocks nothing.

---

## 7. The subagent fleet

Eleven agents. **fable** ×1 (imagination, used sparingly), **sonnet** ×8 (the bulk), **haiku** ×2 (fast mechanical work).

### Wave 0 — foundation (blocks everything; the two run in parallel with each other)

| Agent | Model | Owns |
|---|---|---|
| **FOUNDRY** | sonnet | Next.js scaffold, `theme.css` from §1.1, fonts, shadcn init, downloaded Figma assets, base primitives (Button/Card/Badge/Input/Dialog/Toast/Skeleton/Tabs), app shell + nav + footer, `lib/api/*` typed client, `types.ts` transcribed from the Go DTOs, MSW mock server + fixtures, SSE provider, auth context, `next.config.ts` rewrites. |
| **KEYSTONE** | sonnet | The **only** agent permitted to touch Go. The scoped backend change-set from Decision A — §9.1 and nothing beyond it. Runs `go build ./...` and the existing tests to prove nothing regressed. |

No Wave-1 agent starts until FOUNDRY lands — every one of them imports its tokens, client and primitives. KEYSTONE touches only `backend/`, so it never collides with the frontend fleet.

### Wave 1 — features (7 agents in parallel, one worktree each, disjoint file ownership)

| Agent | Model | Owns | Routes |
|---|---|---|---|
| **BEACON** | sonnet | Landing page, pixel-exact to Figma `61:46` — nav, hero, problems-vs-solutions, USP cards, journey grid, footer. Pulls `get_motion_context`. | 1 |
| **THRESHOLD** | sonnet | Sign-in, callback, onboarding chooser (pixel-exact `15:10`), evidence/repo sync, route guards. | 2–5 |
| **PULSE** | sonnet | Diagnostic quiz, adaptive practice loop, async hints, error analysis, session summary + Learning DNA. | 6, 7, 14, 15 |
| **ATLAS** | sonnet | Roadmap: active section, locked previews, node detail, tutorial reader, complete/test-out, SSE-driven live unlock. | 9, 10, 11 |
| **FORGE** | sonnet | The IDE — split statement/editor, Monaco, 8-language switcher, Run vs Submit, Judge0 polling, test-case results panel, sanitised statement rendering. | 13 |
| **PRISM** | sonnet | Dashboard, profile, settings, problem browser + scrape-by-URL, mastery radar + rating history (via `dataviz`). | 8, 12, 16, 17 |
| **BYTE** | **fable** | *Imagination, rationed.* The Byte the Beaver character system (idle/thinking/celebrate/hint/error states), the unlock moment, page transitions, the scanline-grid signature, empty & error states. Ships a self-contained `components/motion/` + `components/byte/` library the others consume. | — |

### Wave 2 — integration (sequential)

| Agent | Model | Owns |
|---|---|---|
| **HARNESS** | haiku | Dockerfile, `.dockerignore`, compose service + wiring, env files, healthcheck route, README. Mechanical and fast. |
| **LENS** | haiku | QA sweep — `tsc --noEmit`, ESLint, `next build`, route reachability, a11y (contrast is a real risk on `#B9CACB` over `#161B22`), responsive checks at 1280/1024/768/390, reduced-motion audit. Runs between waves too. |

### Worktree protocol — isolation without commits

You asked for worktrees *and* no commits, which need reconciling: a fresh worktree checks out from a commit, so it cannot see FOUNDRY's uncommitted foundation. The resolution:

```
1. FOUNDRY builds ./frontend in the main tree.            (no commit)
2. Orchestrator creates 7 worktrees off `frontend`:
     git worktree add ../tv-wt-<agent> -b wt/<agent> frontend
3. Copy the foundation into each worktree (excluding node_modules),
   then junction node_modules to the main tree:
     cmd //c mklink /J <wt>\frontend\node_modules <main>\frontend\node_modules
   — keeps 7 worktrees cheap instead of ~3.5GB of duplicated deps.
4. Agents work in their worktree. File ownership is disjoint by design.
5. Orchestrator copies each agent's owned paths back into the main tree.
   Disjoint sets ⇒ file-level merge, no git merge, no commits.
6. git worktree remove ../tv-wt-<agent> --force  (branches left for inspection)
```

**No agent runs `git commit`, `git add`, `git push`, or `git merge`.** That is in every agent brief. Everything lands as working-tree changes in `C:\Users\aksha\Transverse\frontend` for you to review and commit yourself.

### Shared brief given to every agent

Design tokens are frozen — use `theme.css` variables, never a raw hex · reuse FOUNDRY's primitives, never re-roll a Button · icons come from the committed Figma exports, never hand-drawn SVG paths · all data through `lib/api`, never bare `fetch` · TS strict, no `any`, no non-null `!` · loading/empty/error state for every async surface · `prefers-reduced-motion` respected · stay inside your owned paths · **no git commits** · when Figma has the frame, match it to the pixel; when it doesn't, extend the system, don't invent a new one.

---

## 8. Sequence & acceptance

| Phase | Work | Done when |
|---|---|---|
| 0 | FOUNDRY ‖ KEYSTONE | `next build` passes; mock server answers all 22 endpoints; theme tokens render; `go build ./...` clean and evidence routes reachable |
| 1 | 7 agents in parallel | each route renders against mocks with loading/empty/error states |
| 2 | Merge back + HARNESS + LENS | typecheck + lint + build clean; compose file reviewed |
| 3 | Live flip | `NEXT_PUBLIC_API_MODE=live` against a running stack — **deferred until you install Docker (Decision D)** |

**Acceptance for the whole build**
1. Landing and onboarding are pixel-faithful to Figma at 1280px — spot-checked against `get_screenshot` diffs.
2. Every net-new screen uses only §1.1 tokens.
3. Every endpoint in `openapi.yaml` + the six unlisted routed endpoints is either wired or explicitly logged as blocked.
4. The full journey runs end-to-end on mocks: land → sign in → onboard → quiz → roadmap → tutorial → solve → submit → unlock → dashboard.
5. `docker compose up` builds and serves the frontend at `:3000` proxying `:8080` — *pending Docker availability*.
6. Zero TS errors, zero ESLint errors, no console errors on any route.
7. Reduced-motion honoured everywhere; AA contrast on all text.

---

## 9. Decisions — settled

### 9.1 — Decision A: scoped backend change-set ✅ approved

**KEYSTONE owns these four edits and nothing else in the backend.** Any other Go change is out of scope and gets reported back rather than made.

| # | File | Change |
|---|---|---|
| 1 | `backend/cmd/server/main.go` | Construct `evidence.Service` + `handlers.NewEvidenceHandler` and mount inside the authenticated group: `POST /evidence/upload-url`, `POST /evidence/{id}/confirm`, `POST /evidence/github`, `/leetcode`, `/codeforces` |
| 2 | `backend/internal/handlers/evidence_handler.go` | Replace `r.Context().Value("user_id").(string)` with `middleware.GetUserID(r.Context())` — the current form panics, since the middleware writes key `UserIDKey("userID")` |
| 3 | `backend/cmd/server/main.go` | Add CORS middleware (allowed origin from `FRONTEND_ORIGIN`, credentials on, `Authorization` + `Content-Type` headers) — belt-and-braces alongside the Next proxy |
| 4 | `backend/internal/handlers/auth_handler.go` | `issueTokens` gains a browser path: on the OAuth callback, `302` to `${FRONTEND_ORIGIN}/auth/callback` instead of writing JSON. The JSON body is retained for `POST /auth/refresh`, which is called programmatically. |
| 5 | `backend/docker-compose.yml` | Add the `minio` service (mirroring `docker-compose.test.yml`) + `MINIO_*` env on `backend`; required by the presigned-upload path in change 1 |

Acceptance: `go build ./...` clean, existing tests still pass, the five evidence routes return non-404, and no handler outside this table is modified.

### 9.2 — Decision B: quiz reuses `/practice/*` ✅

`/onboarding/quiz` starts an adaptive session capped at ~8–10 questions across seed topics, closes it, and feeds `per_topic_breakdown` into `POST /roadmap/generate` as `confirmed_hypotheses` / `debunked_hypotheses`. No new backend endpoint. This is exactly the "hypothesis → verification" loop the pitch deck describes, running on the engine that already exists.

### 9.3 — Decision C: real OAuth only ✅

No dev-bypass affordance in the UI. See §5.3 — **action on you:** register the GitHub and Google OAuth apps and put the credentials in `backend/.env`, or sign-in stays mock-only while every other screen proceeds normally on `BYPASS_AUTH`.

### 9.4 — Decision D: Docker written now, verified later ✅

HARNESS writes the Dockerfile and compose wiring; LENS reviews it line-by-line; `docker compose build` is **not** run in this session. Ping me once Docker Desktop is installed and I'll do the live-flip pass (§8 phase 3).

### 9.5 — Open items carried forward

- OAuth app credentials (§9.3) — blocks only the sign-in screen.
- Docker install (§9.4) — blocks only phase 3.
- `Documentation/openapi.yaml` is drifted (§0.2). Not in scope for this build; worth regenerating from the Go source afterwards so the Notion "Detailed API Documentation" page stops describing shapes that no longer exist.
