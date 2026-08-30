# Transverse — frontend

Next.js 15 (App Router) + TypeScript strict + Tailwind CSS v4 + shadcn/ui +
TanStack Query v5 + MSW v2. See `FOUNDATION.md` for the full contract this
app is built against (types, API client, providers, UI kit, mock layer).

## Getting started

```bash
npm install
cp .env.example .env.local   # defaults to mock mode, no backend needed
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). By default
`NEXT_PUBLIC_API_MODE=mock` — MSW intercepts every API call (client-side via
the browser worker, server-side via `src/instrumentation.ts`), so the whole
app runs and is fully clickable with zero backend, zero Docker, zero OAuth
app registration. Flip to `NEXT_PUBLIC_API_MODE=live` (and point
`BACKEND_URL` at a running backend) once you want real data.

## Scripts

| Command | Does |
|---|---|
| `npm run dev` | Dev server |
| `npm run build` | Production build (`output: "standalone"`) |
| `npm run typecheck` | `tsc --noEmit`, strict |
| `npm run lint` | ESLint (flat config, `next/core-web-vitals` + `next/typescript`) |
| `npm test` | Vitest, once (runs against the same MSW handlers as mock mode) |
| `npm run test:watch` | Vitest, watch mode |

## Mock mode vs live mode

`NEXT_PUBLIC_API_MODE` switches between two fully-typed-identical code
paths — nothing in application code branches on this flag; it only decides
whether MSW is running:

- **mock** (default) — `src/mocks/handlers.ts` answers all 34 routes with
  realistic fixtures (`src/mocks/fixtures/`). Started client-side by
  `MockProvider` and server-side by `src/instrumentation.ts` (so RSC data
  fetches and the `src/app/api/auth/*` Route Handlers are covered too, not
  just client components).
- **live** — requests go to `/api/v1/*`, which `next.config.ts` rewrites to
  `BACKEND_URL` server-side (same-origin from the browser's perspective, so
  no CORS preflight). `GET /health` gets its own rewrite rule — it's
  mounted outside `/api/v1` on the backend.

## Verifying SSE isn't buffered

`GET /events/stream` is consumed via a hand-rolled `fetch` + `ReadableStream`
reader (see `src/lib/realtime/sse-client.ts`) — not the browser's native
`EventSource`, because `EventSource` can't set the `Authorization` header
the backend's `middleware.Auth` requires. A **buffered** stream is a
*silent* failure mode: the client just never receives events until the
connection eventually times out, with nothing that looks like an error.

Once running against a live backend, verify with:

```bash
curl -N http://localhost:3000/api/v1/events/stream \
  -H "Authorization: Bearer <a real access token>"
```

`-N` disables curl's own output buffering. Expect the initial
`event: connected` frame immediately, then a `: keepalive` comment line
every 15 seconds (the backend's heartbeat ticker) even with no real events
firing. If everything only shows up in one burst when the connection
finally closes, something upstream is buffering the response — check that
whatever reverse proxy sits in front of Next in production isn't gzipping
or buffering this route (an nginx-style `X-Accel-Buffering: no` on the
response, or the equivalent for whatever's actually in front of you, is the
usual fix). `next.config.ts`'s own rewrite is a pass-through and does not
buffer.

## Docker and the full stack

To run the complete backend + frontend + dependencies (Postgres, Redis, Judge0, MinIO):

```bash
cd backend
docker compose -f docker-compose.yml up
```

This builds and starts the entire stack:
- **frontend**: `http://localhost:3000` (built with `NEXT_PUBLIC_API_MODE=live`)
- **backend**: `http://localhost:8080`
- **Postgres**: port 5432
- **Redis**: port 6379
- **Judge0**: port 2358
- **MinIO**: ports 9000/9001

The frontend container is built with `NEXT_PUBLIC_API_MODE=live`, which overrides `.env.local` at build time. The `.dockerignore` excludes `.env.local`, ensuring the mock-mode default never leaks into the image. The app connects to the backend at `http://backend:8080` (Docker's internal DNS).

**Key constraint**: `NEXT_PUBLIC_API_MODE` is baked into the Next.js build — setting it at runtime has no effect. If you manually build the Docker image without passing this ARG, the image ships with mock mode and silently never talks to the backend.

The frontend's `next.config.ts` uses `output: "standalone"`, which creates a self-contained build. The Docker image:
1. Installs dependencies and builds with `npm run build` + `NEXT_PUBLIC_API_MODE=live`
2. Copies only `.next/standalone`, `.next/static`, and `public`
3. Runs as non-root on Node 22-Alpine
4. Exposes port 3000 with a healthcheck

## Directory map

See `FOUNDATION.md` for the full API surface. Quick orientation:

```
src/app/                 routes (App Router) + the 3 auth Route Handlers
src/app/theme.css         frozen design tokens (@theme, Tailwind v4 CSS-first)
src/components/ui/        shadcn/ui primitives, restyled to theme.css tokens
src/components/shell/     TopNav, Footer, PageContainer
src/components/providers/ AppProviders stack: MSW, TanStack Query, Auth, SSE
src/lib/api/               types.ts, client.ts, endpoints.ts — the ONLY way to call the backend
src/lib/auth/               token-store.ts (in-memory access token), cookie.ts
src/lib/realtime/           sse-client.ts (the fetch/ReadableStream SSE reader)
src/mocks/                  MSW handlers + fixtures (mock mode)
public/figma/                downloaded Figma assets — never reference figma.com URLs at runtime
```
