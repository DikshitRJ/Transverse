# Transverse codebase guide

This guide maps the current implementation of Transverse and the primary responsibility of each area. The system is a Go API backed by PostgreSQL, Redis, Judge0, and MinIO, with a Next.js frontend.

## System overview

```text
Next.js frontend
       |
       v
Go HTTP API (Chi) -----> PostgreSQL + pgvector
       |                         |
       +----> Redis <-------------+
       |
       +----> Judge0 (code execution)
       |
       +----> MinIO (temporary evidence uploads)
       |
       +----> Z.ai-compatible LLM API (optional)
```

`docker-compose.yml` runs these local services, initializes data through `init-data`, and then starts the backend and frontend. Keep the optional Cloudflare tunnel disabled for normal local development.

## Repository layout

```text
backend/
  cmd/server/            Application entry point and dependency wiring
  internal/
    cache/               Redis, memory, and no-op cache implementations
    config/              Environment configuration
    connectors/          GitHub, LeetCode, and Codeforces evidence sources
    database/            PostgreSQL pool setup
    embedding/           ONNX embedding support and tokenizer
    evidence/            Evidence intake and cleanup orchestration
    graph/               Topic-DAG loading and traversal
    handlers/            Chi HTTP handlers
    jobs/                Redis-backed asynchronous job queue and workers
    llm/                 Z.ai-compatible client and prompt rendering
    middleware/          CORS, auth, and rate limiting
    models/              Database models and transport DTOs
    oauth/               OAuth provider helpers
    objectstore/         MinIO client
    realtime/            Server-Sent Events over Redis pub/sub
    repository/          PostgreSQL queries and persistence
    roadmap/             Roadmap generation and progression logic
    scraper/             Problem metadata scraping
    services/            Practice, scoring, graph, and Judge0 services
  pipeline/
    ingest/              Tutorial and roadmap ingestion command
    seed/                Problem normalization, embedding, and seed command
  sql/                    Ordered PostgreSQL migrations
frontend/                 Next.js App Router application
data/                     Topic DAG, tutorials, and problem seed data
Documentation/            OpenAPI contract and design/workflow documents
extras/                   Reference code and non-production utilities
```

## Request and data flow

1. A frontend request reaches a Chi handler in `backend/internal/handlers`.
2. Middleware applies request IDs, recovery, CORS, authentication, and rate limiting as appropriate.
3. The handler calls a domain service or feature package.
4. Services use repositories for PostgreSQL, cache interfaces for Redis, and adapters for external systems.
5. Long-running work is submitted to `internal/jobs`; the frontend receives updates through `internal/realtime` SSE endpoints.

The router and dependency graph are assembled in `backend/cmd/server/main.go`. Use it as the authoritative starting point for endpoint registration and implementation wiring.

## Frontend modes

`frontend/.env.local` controls the API mode:

- `NEXT_PUBLIC_API_MODE=mock` starts the UI with MSW fixtures and does not require the backend.
- `NEXT_PUBLIC_API_MODE=live` sends requests through the Next.js rewrite to `BACKEND_URL`. Docker Compose builds the frontend in this mode.

The public frontend API layer lives under `frontend/src/lib/api`; mock fixtures and handlers live under `frontend/src/mocks`.

## Persistence and initialization

- PostgreSQL migrations in `backend/sql` create the schema when the database container first starts.
- `init-data` runs the seed and tutorial ingestion pipelines after PostgreSQL becomes healthy.
- Docker named volumes `pgdata` and `miniodata` retain local state across `docker compose down`. Remove them only when intentionally resetting data.
- `data/topics.json` is the canonical topic DAG used by graph and roadmap features. Keep topic identifiers stable unless migrations and clients are updated together.

## Contracts and validation

- `Documentation/openapi.yaml` is the HTTP API contract.
- `Documentation/end-to-end-walkthrough.md` documents a representative user flow.
- Go tests live beside the code and under `backend/integration_tests`.
- Frontend scripts in `frontend/package.json` provide linting, type checks, builds, and tests.
