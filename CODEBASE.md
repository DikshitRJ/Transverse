# Codebase & Architecture Guide

This document details the structural layout, design patterns, and core systems of the Transverse Go backend.

## 1. Directory Structure

```text
/
├── backend/
│   ├── cmd/server/          # Application entry point, dependency injection, and Chi router setup
│   ├── internal/            # Domain logic (isolated via Clean Architecture)
│   │   ├── cache/           # Cache interfaces (Redis & Memory)
│   │   ├── config/          # Environment variable loading and validation
│   │   ├── connectors/      # External scrapers (GitHub, LeetCode, Codeforces)
│   │   ├── database/        # Postgres connection pooling (pgxpool)
│   │   ├── embedding/       # ONNX runtime and BPE tokenizer for BGE-small embeddings
│   │   ├── evidence/        # MinIO upload orchestration and signal extraction
│   │   ├── graph/           # In-memory topic DAG loading
│   │   ├── handlers/        # HTTP controllers (translating REST to Service calls)
│   │   ├── jobs/            # Redis-backed async job queue (workers)
│   │   ├── llm/             # Z.ai API client and structured prompt templates
│   │   ├── middleware/      # Auth (JWT Denylist) and Rate Limiting (Redis Lua)
│   │   ├── models/          # DTOs and DB Structs
│   │   ├── oauth/           # GitHub/Google OAuth2 state management
│   │   ├── realtime/        # SSE streaming from Redis pub/sub
│   │   ├── repository/      # Data access layer (Postgres queries)
│   │   ├── roadmap/         # LLM roadmap generation and progression gating
│   │   └── services/        # Core business logic (Practice, Scoring, Glicko-2)
│   ├── pipeline/            # Offline CLI pipelines
│   │   ├── ingest/          # Tutorial and Curated Roadmap NDJSON ingester
│   │   └── seed/            # Problem embedding generator and DB seeder
│   └── sql/                 # Ordered Postgres migrations (001 - 011)
├── data/                    # The canonical topics.json DAG and generated seed data
├── Documentation/           # OpenAPI specs, Walkthroughs, and specific design schemas
└── docker-compose.yml       # Dev/Prod infrastructure stack
```

## 2. Core Design Patterns

### Clean Architecture & Dependency Injection
The codebase enforces strict boundaries:
- **Handlers** do not write SQL; they parse JSON and call Services.
- **Services** contain business logic (e.g., Glicko-2 scoring, hypothesis evaluation) but do not tie themselves to a specific DB.
- **Repositories** handle all `pgxpool` queries.
- Interfaces are defined centrally and injected in `cmd/server/main.go`.

### Zero-Cloud Privacy Model
In `internal/evidence/`, uploaded resumes and codebase zips are strictly handled in memory. An inline `defer` block guarantees that the raw object in MinIO is purged the moment the LLM extracts the skill signals. As a fallback, MinIO buckets have aggressive TTL policies.

### Decoupled AI & Determinism
Transverse separates generation from evaluation:
- **LLMs** (`internal/llm/`) are used to read messy inputs (codebases) and generate hypotheses or structure roadmaps. All outputs are strictly JSON-enforced.
- **Scoring** (`internal/services/scoring.go`) is 100% deterministic (Glicko-2). The LLM never decides if an answer is correct or what a user's rating is; it only feeds data to the deterministic psychometric engine.

### Asynchronous Processing
Heavy tasks (embeddings, API scraping, LLM API calls) run in the background.
1. The client hits an endpoint (e.g., `/api/v1/evidence/github`) and receives a `202 Accepted` with a `job_id`.
2. The `jobs/` worker pool executes the task and pushes updates to Redis.
3. Next.js listens to the SSE stream (`/api/v1/events/stream`) and reflects the completion in real-time.
