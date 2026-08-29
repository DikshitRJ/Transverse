# TRANSVERSE — Backend Completion Plan (Agent Execution Spec)

**Audience:** an autonomous coding agent extending the existing Go backend.
**Scope:** BACKEND ONLY. Do not build any frontend/Next.js code. Every feature must be exposed as a documented HTTP (and, where noted, Redis pub/sub) API so that a separate Next.js team can build the UI against this contract with zero backend changes.
**Golden rule:** heavy processing (LLM orchestration, scraping, embeddings, scoring, session state) lives in Go. Next.js will call these APIs server-side (Route Handlers / Server Actions) — never assume browser-side calls, so don't worry about CORS-for-browsers, but do version and document everything as if a stranger has to integrate blind.

---

## 0. Baseline — What Already Exists (do not re-architect, extend it)

Root: `backend/` (Clean Architecture, idiomatic Go).

| Layer | Path | Status |
|---|---|---|
| Entry point | `cmd/server/main.go` | Done — wires DB, cache, graph, repos, services, Chi router |
| Seed/ingestion pipeline | `pipeline/seed/*` | Done — loads CF/AtCoder/CSES/LeetCode JSON, normalizes difficulty, embeds, upserts |
| Embeddings | `internal/embedding/*` | Done — Go BPE tokenizer + ONNX runtime wrapper for `BAAI/bge-small-en-v1.5` (384-dim) |
| Scoring core | `internal/services/scoring.go`, `theta.go`, `glicko.go`, `practice_analytics.go`, `graph_service.go`, `helpers.go` | Done — 7-factor scorer, 1PL IRT, Glicko-2, LearningDNA |
| Judge0 | `internal/services/judge0.go` | Done — proxy submit + poll |
| Practice state machine | `internal/services/practice_session.go` | Done — Start/Submit/Skip/Close |
| Repositories | `internal/repository/*` | Done — problem (incl. HNSW `FindSimilar`), session, user, stats |
| Handlers | `internal/handlers/*` | Done — practice, user, health |
| Middleware | `internal/middleware/auth.go` (JWT validate, **login flow stubbed**), `ratelimit.go` (in-memory token bucket) | Partial |
| Cache | `internal/cache/memory.go` | In-memory TTL only — **no Redis yet** |
| Graph | `internal/graph/*` | Done — `topics.json`, 30 parents / 258 aliases |
| Infra | `docker-compose.yml` (Postgres+pgvector, API), `sql/001..005` | Partial — no Redis, no object storage, no Judge0 container |

**Pending per the Notion tracker (this plan closes all four):**
1. LLM integration (Z.ai GLM 4.7 Flash) — not wired up anywhere.
2. OAuth2 — JWT middleware exists, login flow stubbed.
3. Redis — needed for cache, pub/sub to frontend, job queues, session/rate-limit store.
4. Frontend — **explicitly out of scope for this plan.**

**New product requirements this plan adds on top of the tracker** (from the pitch deck / problem statement):
- Phase 1 "Deep Skill Verification": evidence upload (GitHub/LeetCode/Codeforces profiles, resume, codebase) → LLM hypothesis generation → Glicko-backed adaptive verification quiz.
- Phase 2 "Dynamic Roadmap": LLM+graph roadmap generation, Duolingo-style gated phases/subphases, tutorial content model, "test-out" bypass, closed-loop remediation (LLM hint/error-analysis feeding back into the heuristic engine).
- Zero-cloud privacy: raw uploaded files/media/identity markers must never be persisted; only derived skill signals are stored.
- Object storage container (MinIO) for ephemeral upload handling.
- Redis contracts so Next.js (server-side) can subscribe to async job/roadmap events.

---

## 1. Target Architecture

```
                         ┌───────────────────────────┐
                         │   Next.js (NOT THIS REPO)  │
                         │  Server Components / RSC / │
                         │  Route Handlers            │
                         └───────────┬───────────────┘
                                     │ HTTPS (REST + SSE)
                                     ▼
┌────────────────────────────────────────────────────────────────────┐
│                         Go Backend (this repo)                      │
│  cmd/server  →  Chi router  →  handlers  →  services  →  repos      │
│                                                                      │
│  New subsystems added by this plan:                                 │
│   - internal/llm        (Z.ai GLM 4.7 Flash client + prompt layer)  │
│   - internal/connectors (GitHub / LeetCode / Codeforces scrapers)   │
│   - internal/evidence   (upload orchestration, extraction workers)  │
│   - internal/roadmap    (roadmap generation + gating engine)        │
│   - internal/quiz       (adaptive verification quiz engine)         │
│   - internal/realtime   (Redis pub/sub → SSE gateway)               │
│   - internal/objectstore(MinIO client, presigned URLs, TTL sweep)   │
│   - internal/oauth      (GitHub/Google OAuth2 flows)                │
└───────┬───────────────┬───────────────┬───────────────┬────────────┘
        │               │               │               │
        ▼               ▼               ▼               ▼
 ┌─────────────┐  ┌───────────┐  ┌────────────┐  ┌────────────────┐
 │ Postgres 16 │  │  Redis 7  │  │   MinIO    │  │ Judge0 (self-  │
 │ + pgvector  │  │ cache/pub │  │ (ephemeral │  │ hosted, own    │
 │ HNSW index  │  │ sub/queue │  │  uploads)  │  │ compose stack) │
 └─────────────┘  └───────────┘  └────────────┘  └────────────────┘
                                                          ▲
                                                          │ HTTPS
                                                 ┌────────┴────────┐
                                                 │ Z.ai GLM-4.7     │
                                                 │ Flash (external) │
                                                 └──────────────────┘
```

**Communication contract with the (future) Next.js container:**
- Synchronous reads/writes → REST JSON over `/api/v1/*`.
- Long-running work (LLM roadmap generation, evidence processing, hint generation) → kick off a job, return `job_id` immediately (202 Accepted), client polls `GET /api/v1/jobs/{id}` **or** subscribes to `GET /api/v1/events/stream` (SSE, backed by Redis pub/sub — see §9.9). Next.js runs server-side, so both polling and SSE are viable; document both, let the frontend team choose.
- Redis itself is **not** exposed on a public port — only on the internal docker network — so if Next.js wants to subscribe directly (same compose network) that's an optional optimization, not the required integration path. The SSE endpoint is the supported contract.

---

## 2. Infrastructure Changes

### 2.1 `docker-compose.yml` additions

Add to the existing compose file (keep the existing `postgres` and `api` services untouched apart from new env vars/depends_on):

```yaml
  redis:
    image: redis:7-alpine
    command: ["redis-server", "--save", "60", "1", "--loglevel", "warning"]
    volumes:
      - redis_data:/data
    networks: [transverse_net]
    # no host port mapping — internal only

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio_data:/data
    ports:
      - "9001:9001"   # console only, for dev debugging; remove in prod
    networks: [transverse_net]

  judge0-server:
    image: judge0/judge0:1.13.1
    # follow the official judge0 docker-compose (server + workers + its own
    # postgres + redis). Vendor it as backend/infra/judge0/docker-compose.yml
    # and reference via `include:` (Compose v2) so it stays isolated from the
    # app's own Postgres/Redis. Do NOT reuse the app's Postgres for Judge0.
    ...

volumes:
  redis_data:
  minio_data:
```

Add an `entrypoint` init container or a `mc` (MinIO client) one-shot job that creates the required buckets and lifecycle rules on startup (see §9.2).

### 2.2 New environment variables (extend `internal/config/config.go`)

```
# Redis
REDIS_ADDR=redis:6379
REDIS_DB=0

# Object storage
MINIO_ENDPOINT=minio:9000
MINIO_ROOT_USER=...
MINIO_ROOT_PASSWORD=...
MINIO_USE_SSL=false
EVIDENCE_BUCKET=evidence-uploads
EVIDENCE_TTL_SECONDS=3600          # hard delete raw upload after 1h regardless of processing state

# LLM (Z.ai)
ZAI_API_KEY=...
ZAI_BASE_URL=https://api.z.ai/api/paas/v4
ZAI_MODEL=glm-4.7-flash
ZAI_TIMEOUT_SECONDS=30
ZAI_MAX_RETRIES=2

# OAuth2
OAUTH_GITHUB_CLIENT_ID=...
OAUTH_GITHUB_CLIENT_SECRET=...
OAUTH_GITHUB_REDIRECT_URL=...
OAUTH_GOOGLE_CLIENT_ID=...
OAUTH_GOOGLE_CLIENT_SECRET=...
OAUTH_GOOGLE_REDIRECT_URL=...
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=30

# Connectors
GITHUB_API_BASE=https://api.github.com
LEETCODE_GRAPHQL_URL=https://leetcode.com/graphql
CODEFORCES_API_BASE=https://codeforces.com/api
CONNECTOR_TIMEOUT_SECONDS=10
CONNECTOR_MAX_REPOS_SCANNED=15     # cap to bound cost/latency
```

`config.go`'s typed struct must be extended with matching fields; fail fast on missing required vars, same pattern as today.

---

## 3. Data Model Additions

Continue the existing `sql/00N_*.sql` migration sequence (next is `006`). Keep everything additive — no destructive changes to existing tables. Sketch (write full DDL with proper FKs, indexes, `created_at`/`updated_at`, and `pgvector` types matching the existing `problems` table conventions):

```
006_evidence.sql
  evidence_sources(
    id, user_id FK, kind ENUM(github,leetcode,codeforces,resume,codebase),
    external_ref TEXT,        -- username/handle/git URL; NULL for file uploads
    object_key TEXT,          -- MinIO key while file exists (nullable after purge)
    status ENUM(pending,fetching,processing,done,failed,purged),
    error_message TEXT,
    created_at, processed_at, purge_at
  )
  evidence_extracts(
    id, evidence_source_id FK,
    extracted_json JSONB,     -- ONLY derived signal: languages, topic tags,
                              -- repo/problem counts, claimed skills — NEVER
                              -- raw file text, emails, names, phone numbers
    confidence REAL,
    created_at
  )

007_hypotheses_and_quiz.sql
  skill_hypotheses(
    id, user_id FK, topic_id FK(topics), source_evidence_id FK NULL,
    rationale TEXT,           -- short LLM-generated justification, no PII
    confidence REAL,
    status ENUM(pending,confirmed,debunked,inconclusive),
    created_at, resolved_at
  )
  quiz_sessions(
    id, user_id FK, purpose ENUM(verification,placement), status ENUM(active,completed,abandoned),
    started_at, completed_at
  )
  quiz_items(
    id, quiz_session_id FK, problem_id FK(problems), hypothesis_id FK NULL,
    sequence INT, presented_at, answered_at, verdict, time_taken_ms
  )

008_roadmap.sql
  roadmap_templates(id, target_role TEXT, source ENUM(llm_generated,curated), version INT, created_at)
  roadmap_phases(id, roadmap_template_id FK, sequence INT, title TEXT, unlock_rule JSONB)
  roadmap_nodes(id, phase_id FK, topic_id FK(topics), sequence INT, tutorial_ids UUID[], practice_topic_ids UUID[])
  user_roadmaps(id, user_id FK UNIQUE, roadmap_template_id FK, status ENUM(active,completed,abandoned), current_phase_id FK NULL, created_at)
  user_roadmap_node_progress(id, user_roadmap_id FK, node_id FK, status ENUM(locked,unlocked,in_progress,mastered,tested_out), unlocked_at, mastered_at)

009_tutorials.sql
  tutorials(
    id, source TEXT, source_url TEXT UNIQUE, title TEXT, topic_id FK(topics) NULL,
    topic_tags TEXT[], type ENUM(article,video,interactive,playlist),
    difficulty ENUM(beginner,intermediate,advanced), estimated_minutes INT,
    summary TEXT,             -- short, original summary — never the scraped article body
    license_note TEXT, thumbnail_url TEXT, scraped_at, checksum TEXT
  )

010_llm_jobs_and_errors.sql
  llm_jobs(
    id, user_id FK, job_type ENUM(hypothesis,roadmap,hint,error_analysis,remediation),
    status ENUM(queued,running,done,failed), input_ref JSONB, output_json JSONB,
    error TEXT, created_at, started_at, completed_at
  )
  error_analyses(
    id, session_response_id FK(session responses table), verdict TEXT,
    llm_feedback TEXT, hint_level INT, created_at
  )

011_oauth.sql
  oauth_accounts(
    id, user_id FK, provider ENUM(github,google), provider_user_id TEXT,
    UNIQUE(provider, provider_user_id), created_at
  )
  refresh_tokens(id, user_id FK, token_hash TEXT UNIQUE, expires_at, revoked_at NULL, created_at)
```

Extend `stats_repo.go` / `problem_stats_repo.go` tables as needed rather than duplicating — the per-topic mastery + Glicko rating already tracked there is the canonical "skill profile"; hypotheses/roadmap just read/write into it.

---

## 4. Milestone Plan (execute strictly in this order; each milestone must compile, pass tests, and update `docs/API.md` before moving on)

### M1 — Infra & Config
- Add Redis, MinIO, Judge0-compose to `docker-compose.yml`; extend `config.go`; add `internal/cache/redis.go` implementing the same interface `memory.go` already satisfies (so `practice_cache.go` callers don't change); feature-flag which backend is active via `CACHE_DRIVER=redis|memory`.
- Add `internal/objectstore/minio.go`: `PresignPut`, `PresignGet`, `Delete`, `EnsureBucket`, lifecycle rule setup.
- **Acceptance:** `docker-compose up -d` brings up postgres, redis, minio, judge0, api; `GET /health` reports all dependency pings green (extend `health_handler.go`).

### M2 — OAuth2 + Session Completion
- Implement `internal/oauth/github.go`, `internal/oauth/google.go` (standard authorization-code flow using `golang.org/x/oauth2`).
- Wire `middleware/auth.go`'s stub login into real issuance: on OAuth callback, upsert `users` + `oauth_accounts`, issue short-lived JWT access token + opaque refresh token (hashed, stored in `refresh_tokens`, rotated on use).
- Add `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout` (revokes refresh token, and if using Redis, also pushes the JWT's jti onto a denylist set with TTL = remaining token life).
- **Acceptance:** full login → protected-route → refresh → logout cycle covered by integration tests.

### M3 — Evidence Intake Pipeline (connectors + uploads)
See §9.1 for full connector spec. Build:
- `internal/connectors/github.go`, `leetcode.go`, `codeforces.go` — each implements `Fetch(ctx, ref string) (RawSignal, error)`.
- `internal/evidence/service.go` — orchestrates: create `evidence_sources` row → dispatch to connector or object-store fetch → normalize into `evidence_extracts` → enqueue hypothesis generation job → purge any raw object from MinIO.
- Presigned upload flow for resume/codebase (see §9.2).
- **Acceptance:** for each of the 5 evidence kinds, an end-to-end test produces a populated `evidence_extracts` row and zero residual objects in MinIO after processing.

### M4 — LLM Integration Layer
See §9.3. Build:
- `internal/llm/client.go` — thin OpenAI-compatible chat-completions client against Z.ai (`ZAI_BASE_URL + /chat/completions`), with retry/backoff and response caching (Redis, keyed by SHA-256 of the rendered prompt + model).
- `internal/llm/prompts/` — versioned prompt templates (Go `text/template` files) for: hypothesis generation, roadmap generation, hint generation, error analysis, remediation. Every prompt MUST instruct the model to return strict JSON matching a documented schema; validate the response against that schema (e.g. with `encoding/json` + a JSON-schema validator) and retry once on parse failure before failing the job.
- `internal/jobs/` — a minimal Redis-backed job queue (a Redis Stream or List works; no need for a heavyweight framework) with a worker pool started from `cmd/server/main.go` (or a separate `cmd/worker/main.go` — prefer a separate binary so LLM/CPU-bound work can scale independently of the HTTP API; both share the same `internal/` packages).
- **Acceptance:** `POST` any LLM-backed endpoint returns `202` with a `job_id`; job transitions `queued→running→done` are visible via `GET /api/v1/jobs/{id}` and published to the user's Redis event channel.

### M5 — Hypothesis Generation
- `internal/hypothesis/service.go`: given a user's accumulated `evidence_extracts`, render the hypothesis prompt, call LLM, parse into `skill_hypotheses` rows (topic_id resolved via `graph_service.go`'s alias resolver — never trust the LLM's raw topic string).
- **Acceptance:** hypotheses always reference a valid `topics.json` node id; duplicate hypotheses for the same (user, topic) are merged, not duplicated.

### M6 — Adaptive Verification Quiz
See §9.4. Build `internal/quiz/service.go` on top of existing `scoring.go`/`theta.go`/`glicko.go`:
- `StartVerification(userID)`: for each `pending` hypothesis, pick one confirming/debunking problem via `PickBestProblem` (constrained to the hypothesis's topic, difficulty near the hypothesis's implied level) → create `quiz_session` + `quiz_items`.
- `Answer(sessionID, itemID, submission)`: routes through the *existing* Judge0 + Glicko/theta update path (`practice_session.go`) so a quiz answer updates the same skill profile practice does — do not fork the scoring logic.
- On quiz completion: mark each hypothesis `confirmed`/`debunked`/`inconclusive` based on result + updated theta, and enqueue roadmap generation (M7) automatically.
- **Acceptance:** a hypothesis confirmed by a correct-fast answer measurably raises the topic's Glicko rating in `stats_repo`; a failed answer measurably lowers it and flips the hypothesis to `debunked`.

### M7 — Roadmap Engine & Gated Progression
See §9.5. Build `internal/roadmap/service.go`:
- `Generate(userID, targetRole)`: gather skill profile (per-topic Glicko/mastery) + confirmed/debunked hypotheses + the topic DAG → render roadmap prompt → LLM returns phases/nodes referencing **existing topic IDs only** (validate, reject/repair any hallucinated topic) → persist as `roadmap_templates`/`roadmap_phases`/`roadmap_nodes` → create `user_roadmaps` + all node progress rows `locked` except phase 1 nodes with no prerequisites → `unlocked`.
- `Unlock(userRoadmapID)`: re-evaluates `unlock_rule` JSON (e.g. `{"type":"mastery_threshold","topic_id":...,"min_rating":1400}` or `{"type":"phase_complete","phase_id":...}`) whenever a practice/quiz submission changes a rating; publish `node.unlocked` Redis event.
- `TestOut(userRoadmapID, nodeID)`: lets a user attempt a harder placement problem to skip a node without doing every tutorial — implements the "test-out" churn-reduction mechanic from the pitch deck.
- `Regenerate(userRoadmapID)`: closed-loop remediation — called by M8 when repeated failures indicate a root gap; only restructures *upcoming, still-locked* phases, never rewrites completed history.
- **Acceptance:** locked nodes are provably unreachable via the API (attempting to fetch practice problems for a locked node returns `403 node_locked`); unlocking is idempotent and event-driven, not polled.

### M8 — Practice Loop: LLM Hints & Closed-Loop Remediation
- Extend `practice_session.go`'s `Submit` path: on Judge0 verdict `!= Accepted`, optionally enqueue an `error_analysis` LLM job (rate-limited per user per problem — reuse `ratelimit.go`, move its store to Redis so it's correct across multiple API replicas).
- Hints are staged (`hint_level` 1..3, each progressively more revealing); never return full source code — enforce this in the prompt AND with a static max-token/pattern check on the response before returning it.
- On N consecutive failures on the same topic (config threshold), call `roadmap.Regenerate` and drop the user's effective difficulty target for that topic (feed into `scoring.go`'s difficulty-fit factor) — this is the literal "closed-loop adaptive remediation" from the deck.
- **Acceptance:** a scripted run of 3 consecutive fails on one topic measurably lowers next-served difficulty and triggers exactly one `roadmap.updated` event.

### M9 — Tutorial & Roadmap Content Ingestion Tooling
See §9.6. This does **not** scrape live sites (out of scope / not yet done per the brief) — it builds the ingestion *mechanism* and *schema* so whoever scrapes NeetCode/CP-Algorithms/GfG/roadmap.sh later just POSTs conformant JSON:
- `POST /api/v1/admin/tutorials/ingest` (bulk upsert, schema-validated, dedup by `source_url`).
- `POST /api/v1/admin/roadmaps/ingest` (bulk-load a curated roadmap template in the schema from §9.6.2, marked `source=curated` so it can seed default roadmaps without waiting on the LLM).
- A CLI (`pipeline/ingest/main.go`, mirroring `pipeline/seed/`) that reads local JSON/NDJSON files matching the schema and calls the same service layer — so ingestion works both via API and offline batch.
- **Acceptance:** malformed records are rejected individually with a per-record error report, not an all-or-nothing failure.

### M10 — Realtime Gateway
See §9.9. Build `internal/realtime/`:
- Backend publishes to Redis channel `user:{userID}:events` for: `job.completed`, `job.failed`, `node.unlocked`, `roadmap.updated`, `hint.ready`.
- `GET /api/v1/events/stream` — Chi handler that authenticates the JWT, subscribes to that user's channel, streams Server-Sent Events, and closes cleanly on client disconnect.
- **Acceptance:** triggering any async job and watching `/events/stream` in a second terminal shows the corresponding event within one poll interval of the job finishing.

### M11 — Hardening, Observability, Docs
- Structured logging (already likely `log/slog` or similar — match existing style) for every new package.
- Unit tests for every new service (mirror the `_test.go` pattern used throughout `internal/services/`); integration tests for the full evidence→hypothesis→quiz→roadmap→practice loop.
- Generate the formal API docs (see §11) — this is a **required deliverable**, not optional polish.
- Update this repo's top-level status doc (the one this plan supersedes) to reflect final "done" state.

---

## 5. Subsystem Deep-Dives

### 9.1 Evidence Intake & Connectors

All connectors read **public** data only (no credential harvesting, no bypassing auth walls). Cap scope (`CONNECTOR_MAX_REPOS_SCANNED`, timeouts) to keep cost/latency bounded and to be a good API citizen.

| Connector | Source | What's fetched | Notes |
|---|---|---|---|
| GitHub | `GET {GITHUB_API_BASE}/users/{username}/repos`, then per-repo `GET /repos/{owner}/{repo}/languages` and the root `README` | Language histogram, repo names/topics, README first N chars (used only in-memory to detect keyword hits — never persisted verbatim) | Unauthenticated calls are rate-limited (60/hr); support an optional server-side PAT (`GITHUB_TOKEN` env) purely to raise Transverse's own outbound rate limit — this is our token, not the user's |
| LeetCode | `POST {LEETCODE_GRAPHQL_URL}` with a query for `matchedUser(username)` public stats (solved counts by difficulty, tag-level solve breakdown if exposed) | Solved-by-difficulty, solved-by-tag counts | Public profile data only; respect response for private profiles (fail gracefully, mark evidence `failed` with a user-facing reason) |
| Codeforces | `GET {CODEFORCES_API_BASE}/user.info?handles=...` and `GET {CODEFORCES_API_BASE}/user.status?handle=...` | Current rating, rank, and per-submission tag/verdict history (official public API) | This is Codeforces' own documented API — straightforward JSON, no scraping needed |
| Resume | User uploads PDF/DOCX via presigned MinIO URL | Text extracted server-side (reuse `docx`/`pdf` text-extraction libs), LLM asked to output a **skills-only** JSON hypothesis list — the extracted text itself is discarded immediately after the LLM call, never written to Postgres | See §9.2 for lifecycle |
| Codebase | User uploads a zip, or gives a public git URL | `git clone --depth 1` (git URL) or unzip (zip upload) into a tmpfs/ephemeral dir, walk the tree for language/import-statement signals only (no code semantics execution), summarize, then `rm -rf` the working dir | Never execute uploaded code. Never persist file contents. |

Each connector's `Fetch` returns a `RawSignal` struct (Go struct, in-memory only) that `evidence/service.go` normalizes into the `evidence_extracts.extracted_json` shape:

```json
{
  "languages": {"Go": 0.62, "Python": 0.38},
  "claimed_topics": ["graphs", "dynamic-programming"],
  "signals": [
    {"topic_tag": "graphs", "evidence": "used a Dijkstra implementation in repo X", "strength": "weak"}
  ]
}
```

`"strength": "weak"` matters — per the product spec, evidence is a **hypothesis, not proof** (e.g. importing a library ≠ implementing the algorithm). The LLM hypothesis-generation prompt (§9.3) is explicitly instructed to treat library usage / tutorial completion as weak signal and require the verification quiz to confirm anything above `beginner` difficulty.

### 9.2 Zero-Cloud Privacy & Object Storage Lifecycle

Non-negotiable constraints from the product brief: **no personal uploaded files, media, or identity markers persist on cloud backends.**

Implementation:
1. `POST /api/v1/evidence/upload-url` → backend creates an `evidence_sources` row (`status=pending`), generates a MinIO presigned `PUT` URL scoped to a single object key (`evidence/{userID}/{evidenceID}/{filename}`) with a short expiry (e.g. 5 min), returns `{evidence_id, upload_url, expires_at}`.
2. Caller (Next.js, server-side) uploads the file bytes directly to MinIO using that URL — the file never transits the Go API process.
3. Caller then calls `POST /api/v1/evidence/{id}/confirm` → backend enqueues a processing job.
4. The worker downloads the object **into memory/tmpfs only**, extracts signal, writes only `evidence_extracts.extracted_json`, then **immediately calls `objectstore.Delete`** on the object regardless of success/failure, and sets `status=done|failed`, `object_key=NULL`.
5. Defense in depth: the MinIO bucket also has a lifecycle rule (`EVIDENCE_TTL_SECONDS`) that force-expires any object that somehow wasn't explicitly deleted (worker crash, etc.) — belt and suspenders.
6. Nothing in `extracted_json` may contain: full names, emails, phone numbers, physical addresses, raw resume/code text. Enforce with a lightweight regex/PII scrub pass on the LLM's extraction output before persisting, and reject (retry once, then fail the evidence source) if PII patterns are detected post-scrub.

### 9.3 LLM Integration Layer (Z.ai GLM 4.7 Flash)

Z.ai exposes an **OpenAI-compatible** chat completions API:

```
POST https://api.z.ai/api/paas/v4/chat/completions
Authorization: Bearer <ZAI_API_KEY>
Content-Type: application/json

{
  "model": "glm-4.7-flash",
  "messages": [{"role": "system", "content": "..."}, {"role": "user", "content": "..."}],
  "max_tokens": 1200,
  "temperature": 0.2
}
```

Response shape mirrors OpenAI: `choices[0].message.content`. **Verify field names against Z.ai's current published API reference at implementation time** — the LLM API surface for less-established providers changes faster than this plan; treat the shape above as the expected default, not gospel, and write the client against an interface so the concrete provider is swappable.

Build `internal/llm/client.go`:
```go
type Client interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
```
- Always set `temperature` low (≤0.3) for hypothesis/roadmap/error-analysis calls — these feed a deterministic system and must be structurally consistent, not creative.
- Always request/parse **strict JSON** output (instruct via system prompt: "Respond with ONLY a JSON object matching this schema, no prose, no markdown fences"). Validate against a Go struct with `json.Unmarshal` + a schema-shape check; on failure, retry once with an added "your previous response was invalid JSON, fix it" turn, then fail the job cleanly.
- Cache: hash `(prompt_template_version, rendered_variables)` → Redis key `llm:cache:{sha256}` with a sane TTL (e.g. 24h for tutorial/roadmap-shape prompts, no cache for user-specific hint prompts). This directly addresses the pitch deck's "LLM latency & token cost" risk.
- Every LLM job type gets its own prompt template file under `internal/llm/prompts/` with an explicit **input contract** and **output JSON schema** documented in `docs/API.md` (§11) — e.g.:

  - `hypothesis.tmpl` — in: `extracted_json` for one or more evidence sources + list of valid topic IDs; out: `[{"topic_id": "...", "confidence": 0.0-1.0, "rationale": "<=200 chars"}]`
  - `roadmap.tmpl` — in: target role, per-topic mastery map, confirmed/debunked topic IDs, full topic DAG (id/parent/children only, not full graph payload); out: `{"phases":[{"title":...,"nodes":[{"topic_id":...,"unlock_rule":{...}}]}]}` — every `topic_id` MUST exist in the DAG; validate and drop/repair any that don't.
  - `hint.tmpl` — in: problem statement summary, user's submitted code, verdict/error, current `hint_level`; out: `{"hint": "<=2 sentences, no code>"}` for level 1, escalating detail for higher levels, **never** a full solution.
  - `error_analysis.tmpl` — in: verdict (`TLE`/`WA`/`RE`/`CE`), constraints, submitted code; out: `{"likely_cause": "...", "suggested_focus_topic": "<topic_id or null>"}`.
  - `remediation.tmpl` — in: recent failure streak per topic; out: `{"action":"lower_difficulty"|"insert_prerequisite","topic_id":...,"detail":"..."}`.

### 9.4 Adaptive Verification Quiz

Reuses, does not replace, the existing psychometric core:
- Given a `pending` hypothesis on topic T with implied difficulty D (derive D from evidence strength — "weak" signal → serve near the topic's median rating; "strong"/repeated signal → serve above-median), call the existing `PickBestProblem`/`ScoreCandidate` (`scoring.go`) constrained to topic T.
- Present via the same `practice_session.go` Start/Submit machinery (a quiz item *is* a practice session item tagged with `hypothesis_id`), so Judge0 execution, Glicko update, and theta update are all reused verbatim — this is the "deterministic, not generative" guarantee from the pitch.
- Resolution rule (simple, documented, tuneable):
  - Correct within expected time band → `confirmed`, rating bump applied as normal.
  - Incorrect, or correct but far outside expected time → `debunked`, rating drop applied as normal.
  - Ambiguous (e.g. partial test cases passed) → `inconclusive`; serve one more calibration problem before finalizing.

### 9.5 Roadmap Engine & Gated Progression

- The topic DAG (`internal/graph`) is the hard constraint; the LLM is only allowed to *sequence and phase* nodes drawn from it, never invent new topics. This is the "grounded, not generative" architecture claim from the deck — enforce it in code, not just in the prompt.
- `unlock_rule` JSON shapes (documented, extensible):
  ```json
  {"type": "no_prerequisite"}
  {"type": "mastery_threshold", "topic_id": "arrays", "min_rating": 1300}
  {"type": "phase_complete", "phase_id": "<uuid>"}
  {"type": "quiz_pass", "topic_id": "graphs"}
  ```
- "Test-out": `POST /api/v1/roadmap/{id}/nodes/{nodeId}/test-out` serves one hard, above-threshold problem; passing marks the node `tested_out` (counts as `mastered` for downstream unlock checks) without requiring the tutorial to be viewed. Directly implements the deck's churn-reduction mitigation.
- Regeneration only ever touches nodes with `status IN (locked, unlocked)` in phases at or after the user's current phase — completed/mastered history is immutable, so a roadmap never "rewrites the past" under the user.

### 9.6 Tutorial & Roadmap Scraping — Data Formats

Scraping the actual sites (NeetCode, CP-Algorithms, GeeksforGeeks, roadmap.sh, Striver's A2Z, etc.) is **explicitly not part of this backend milestone** — but the ingestion contract must exist now so that work can happen in parallel/later without touching this API. Whoever writes those scrapers must emit NDJSON (one JSON object per line) matching the schemas below and POST it to the ingest endpoints in M9.

**Copyright note for whoever builds the scraper:** store only metadata, tags, and a short *original* summary (2–3 sentences) — never mirror full article/video text. Tutorials are linked out to, not rehosted.

#### 9.6.1 Tutorial record schema
```json
{
  "source": "neetcode | cp-algorithms | gfg | striver-a2z | other",
  "source_url": "https://...",
  "title": "string",
  "topic_tags": ["graphs", "bfs"],
  "type": "article | video | interactive | playlist",
  "difficulty": "beginner | intermediate | advanced",
  "estimated_minutes": 15,
  "summary": "short, original 2-3 sentence description — do not copy source text",
  "author": "string, optional",
  "thumbnail_url": "string, optional",
  "license_note": "e.g. 'linked externally, not mirrored'"
}
```
`topic_tags` should use the same alias vocabulary as `data/topics.json` where possible; the ingest service resolves tags to canonical `topic_id`s via `graph_service.go`, exactly like problem seeding already does.

#### 9.6.2 Curated roadmap template schema
```json
{
  "roadmap_name": "string",
  "target_role": "DSA Mastery | SDE Interview Prep | Competitive Programmer",
  "phases": [
    {
      "title": "string",
      "sequence": 1,
      "unlock_rule": { "type": "no_prerequisite" },
      "nodes": [
        {
          "topic_tag": "arrays",
          "sequence": 1,
          "tutorial_source_urls": ["https://..."],
          "practice_topic_tags": ["arrays"]
        }
      ]
    }
  ]
}
```
This lets a human curate a roadmap.sh-style template offline and load it as a `source=curated` `roadmap_templates` row, giving every new user a sane default even before the LLM path is trusted/tuned.

### 9.7 Redis Contracts (cache, jobs, pub/sub) — Frontend Integration Surface

| Purpose | Key/Channel pattern | Notes |
|---|---|---|
| Generic cache (existing `practice_cache.go` keys, unchanged) | `seen:{userID}`, etc. | Same as today, just backed by Redis now |
| LLM response cache | `llm:cache:{sha256}` | TTL per prompt type |
| Rate limiting (moved off in-memory) | `ratelimit:{ip_or_userID}:{bucket}` | Enables correct limiting across multiple API replicas |
| Async job state | `job:{jobID}` (mirrors the `llm_jobs` row; Redis copy is a fast read-through cache, Postgres is source of truth) | |
| Per-user event stream | Pub/Sub channel `user:{userID}:events`, JSON payload `{"type":"job.completed","job_id":...,"data":{...}}` | Consumed by `internal/realtime` → SSE. Event `type` enum: `job.completed`, `job.failed`, `node.unlocked`, `roadmap.updated`, `hint.ready` |
| Auth denylist | `jwt:denylist:{jti}` with TTL = token remaining life | Set on logout/refresh-rotation |

Document this table verbatim in `docs/API.md` — it is the one place where the Next.js team might reasonably connect to Redis directly (same docker network) as a latency optimization instead of SSE; keep the channel/message contract stable either way.

### 9.8 Auth Summary
- OAuth2 (GitHub, Google) for login; email/password intentionally **not** required by the product but keep the door open — `users` table already generic per `user_repo.go`.
- Short-lived JWT access token (15 min default) + rotating opaque refresh token (30 day default, hashed at rest, single-use — reissue on every refresh, revoke the old one).
- `middleware/auth.go` gains a Redis-backed `jti` denylist check in addition to signature/expiry validation.

---

## 6. Full API Reference (implement all; publish formally per §11)

Base path: `/api/v1`. All authenticated routes require `Authorization: Bearer <access_token>`. Async endpoints return `202 {"job_id": "..."}`; poll `GET /jobs/{id}` or subscribe to SSE.

### Auth
| Method | Path | Purpose |
|---|---|---|
| GET | `/auth/oauth/{provider}/redirect` | Returns the OAuth authorization URL to redirect the user to (`provider` = `github`\|`google`) |
| GET | `/auth/oauth/{provider}/callback` | OAuth callback; issues access+refresh tokens |
| POST | `/auth/refresh` | Exchange refresh token for new access+refresh pair |
| POST | `/auth/logout` | Revoke refresh token + denylist current access token |
| GET | `/auth/me` | Current authenticated identity summary |

### Users (existing `user_handler.go`, extend)
| Method | Path | Purpose |
|---|---|---|
| GET | `/users/me` | Profile + global Glicko rating |
| PATCH | `/users/me` | Update display name/preferences (no PII beyond what OAuth already provided) |
| GET | `/users/me/skill-profile` | Per-topic mastery/rating map |
| GET | `/users/me/learning-dna` | Existing `practice_analytics.go` output |

### Evidence Intake
| Method | Path | Purpose |
|---|---|---|
| POST | `/evidence/upload-url` | Get a presigned MinIO PUT URL for resume/codebase upload |
| POST | `/evidence/{id}/confirm` | Confirm upload landed; enqueue processing |
| POST | `/evidence/github` | `{ "username": "..." }` → creates + processes a GitHub evidence source |
| POST | `/evidence/leetcode` | `{ "username": "..." }` |
| POST | `/evidence/codeforces` | `{ "handle": "..." }` |
| GET | `/evidence` | List current user's evidence sources + status |
| GET | `/evidence/{id}` | Single evidence source detail (status + extracted signal, never raw content) |
| DELETE | `/evidence/{id}` | User-initiated deletion (also purges any lingering object) |

### Hypotheses
| Method | Path | Purpose |
|---|---|---|
| POST | `/hypotheses/generate` | Trigger LLM hypothesis generation from all processed evidence → `202 job_id` |
| GET | `/hypotheses` | List hypotheses (filter by `status`) |

### Jobs & Realtime
| Method | Path | Purpose |
|---|---|---|
| GET | `/jobs/{id}` | Poll async job status/result |
| GET | `/events/stream` | SSE stream of this user's Redis events |

### Verification Quiz
| Method | Path | Purpose |
|---|---|---|
| POST | `/quiz/verification/start` | Build a quiz session from pending hypotheses |
| GET | `/quiz/{sessionId}/next` | Next unanswered item |
| POST | `/quiz/{sessionId}/answer` | Submit code for current item (routes through Judge0 + scoring) |
| POST | `/quiz/{sessionId}/complete` | Finalize session, resolve hypotheses |
| GET | `/quiz/{sessionId}/result` | Confirmed/debunked summary + updated skill profile delta |

### Roadmap
| Method | Path | Purpose |
|---|---|---|
| POST | `/roadmap/generate` | `{ "target_role": "..." }` → `202 job_id` |
| GET | `/roadmap/me` | Current active roadmap, phases, nodes, per-node lock status |
| GET | `/roadmap/{id}` | Specific roadmap detail |
| POST | `/roadmap/{id}/nodes/{nodeId}/test-out` | Attempt to bypass a node |
| POST | `/roadmap/{id}/regenerate` | Manually trigger remediation-style regeneration (also called internally by M8) |

### Tutorials & Topics
| Method | Path | Purpose |
|---|---|---|
| GET | `/topics` | List DAG nodes |
| GET | `/topics/{id}` | Single topic incl. resolved aliases |
| GET | `/tutorials?topic_id=` | List tutorials for a topic |
| GET | `/tutorials/{id}` | Tutorial detail |
| POST | `/admin/tutorials/ingest` | Bulk upsert (admin-scoped) — schema in §9.6.1 |
| POST | `/admin/roadmaps/ingest` | Bulk-load curated roadmap template — schema in §9.6.2 |

### Practice (existing, keep as-is; add hint/error-analysis)
| Method | Path | Purpose |
|---|---|---|
| POST | `/practice/start` | Existing |
| POST | `/practice/submit` | Existing |
| POST | `/practice/skip` | Existing |
| POST | `/practice/close` | Existing |
| GET | `/practice/session/{id}` | Existing (add if missing) |
| POST | `/practice/{sessionId}/hint` | Request next-level LLM hint (rate-limited) → `202 job_id` |
| GET | `/practice/{sessionId}/error-analysis` | Latest LLM error analysis for the most recent non-AC submission |

### Admin / Seed (existing pipeline, expose as endpoints for ops convenience)
| Method | Path | Purpose |
|---|---|---|
| POST | `/admin/seed/problems` | Trigger `pipeline/seed` ingestion run |
| POST | `/admin/seed/topics` | Reload `topics.json` |

### Health
| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Liveness |
| GET | `/health/ready` | Readiness — pings Postgres, Redis, MinIO, Judge0 |

---

## 7. API Documentation — Required Deliverable

Producing docs is **not optional polish**; treat it as a milestone deliverable (part of M11):

1. **`backend/api/openapi.yaml`** — a complete OpenAPI 3.1 spec covering every route in §6: request/response JSON schemas (reuse the schemas defined in §9.3/§9.6 verbatim), auth scheme (`bearerAuth`), error response shape (standardize one envelope, e.g. `{"error": {"code": "...", "message": "..."}}`, matching whatever `handlers/helpers.go`'s `writeError` already does — don't invent a second convention), and example payloads for at least the non-trivial endpoints (evidence, quiz, roadmap, jobs).
2. **`backend/docs/API.md`** — human-readable companion: architecture diagram (§1), the Redis contract table (§9.7), the two scraping JSON schemas (§9.6), and a "getting started" curl walkthrough of the full user journey: OAuth login → submit evidence → generate hypotheses → run verification quiz → generate roadmap → practice → get a hint.
3. Serve the OpenAPI spec at `GET /api/v1/openapi.yaml` (or via Swagger UI/Redoc mounted at `/docs`) so the Next.js team has a live, browsable contract without needing repo access.
4. Keep both files updated as a hard requirement of every milestone in §4 — "acceptance criteria" for each milestone implicitly includes "docs updated," even where not restated.

---

## 8. Testing Requirements
- Unit tests for every new `internal/*` package, following the existing `_test.go` co-location pattern.
- Integration test suite (can use `docker-compose -f docker-compose.test.yml`) exercising the full journey in §7.2's curl walkthrough end-to-end, including asserting that no evidence object survives in MinIO post-processing.
- Contract test that validates `openapi.yaml` against actual handler behavior (e.g. via a tool like `schemathesis` or a hand-rolled request/response schema check) so the docs can't silently drift from the implementation.

## 9. Definition of Done
- [ ] All milestones M1–M11 complete and merged in order.
- [ ] `docker-compose up -d` brings up postgres, redis, minio, judge0, api with zero manual steps beyond `.env` population.
- [ ] Every endpoint in §6 implemented, tested, and documented in `openapi.yaml` + `API.md`.
- [ ] Zero raw evidence file content ever appears in Postgres or survives in MinIO past processing (verified by test).
- [ ] LLM is used only where specified (hypothesis, roadmap sequencing, hints, error analysis, remediation triggers) — all question selection, difficulty, and similarity mapping remains fully on the deterministic Glicko/embedding engine, per the product's core "zero hallucination risk" claim.
- [ ] A Next.js engineer with no access to this repo's source, given only `openapi.yaml`/`API.md`, can implement the entire product frontend without asking a backend question.