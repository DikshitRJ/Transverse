# Agent Codebase Guide

**Attention Future Agents:**
When modifying or extending the Transverse backend, you must adhere strictly to the following rules established by the initial squad:

1. **Clean Architecture Strictness**:
   - Do NOT write SQL queries in HTTP Handlers or Services.
   - Handlers translate HTTP/JSON. Services contain pure business logic. Repositories (`internal/repository/`) execute the `pgxpool` queries.
   - All external dependencies (Redis, MinIO, Z.ai, Postgres) are abstracted via interfaces in `internal/` and injected inside `cmd/server/main.go`.

2. **Generative vs. Deterministic Boundaries**:
   - The LLM (`internal/llm/`) is ONLY used for natural language parsing (hypotheses, hints, roadmap structures). 
   - You must NEVER allow the LLM to decide if an answer is correct or calculate a user's rating. That logic belongs exclusively to the deterministic Glicko-2 engine (`internal/services/scoring.go`).
   - All LLM prompts must enforce strict JSON responses.

3. **Privacy by Default**:
   - Do NOT persist raw evidence (resumes, codebase zips, raw scraped profiles) to the database.
   - Processing logic inside `internal/evidence/` uses `defer` blocks to instantly delete raw objects from MinIO once signal JSON is extracted. Do not bypass this mechanism.

4. **Async-First**:
   - Long-running tasks (embeddings, scraping, LLM API calls) must return `202 Accepted` with a `job_id`. 
   - Work is processed by `internal/jobs/` and published via SSE `internal/realtime/` over Redis.

---

# The Transverse Agent Squad

The backend of Transverse was entirely constructed in parallel by a squad of 9 autonomous AI agents orchestrated by a parent agent. This section details the historical role, ownership, and contributions of each agent.


## 1. Forge (Foundation & Infrastructure)
- **Role**: Bootstrapped the core infrastructure.
- **Ownership**: `docker-compose.yml`, `internal/config/`, `internal/cache/`, `internal/objectstore/`, SQL migrations `006`-`011`, and the isolated Judge0 stack.
- **Key Contribution**: Established the database schema and published Go interface stubs on day one, unblocking all other agents to work simultaneously against mocked interfaces.

## 2. Warden (Identity & Access)
- **Role**: Secured the application.
- **Ownership**: `internal/oauth/`, `internal/middleware/auth.go`.
- **Key Contribution**: Implemented GitHub and Google OAuth2 flows, issued short-lived JWTs, managed rotating refresh tokens, and enforced a Redis-backed JWT denylist for secure logouts.

## 3. Prospector (Evidence & Connectors)
- **Role**: Built the intake pipeline.
- **Ownership**: `internal/connectors/`, `internal/evidence/`.
- **Key Contribution**: Developed rate-limited public scrapers for GitHub, LeetCode, and Codeforces. Orchestrated the MinIO presigned upload flow, ensuring zero-cloud privacy via `defer` blocks that instantly delete raw resume/codebase objects post-extraction.

## 4. Oracle (LLM & Async Platform)
- **Role**: The AI communication bridge.
- **Ownership**: `internal/llm/`, `internal/jobs/`, `internal/realtime/`.
- **Key Contribution**: Built the OpenAI-compatible Z.ai GLM 4.7 Flash client with Redis caching. Designed the background job queue (worker pool) and the Server-Sent Events (SSE) realtime gateway over Redis pub/sub.

## 5. Examiner (Assessment)
- **Role**: Evaluated the user's skills.
- **Ownership**: `internal/hypothesis/`, `internal/quiz/`.
- **Key Contribution**: Transformed evidence into pending skill hypotheses via LLMs, and constructed the adaptive verification quiz loop leveraging the existing deterministic Glicko-2 scoring engine.

## 6. Cartographer (Roadmap & Progression)
- **Role**: Charted the learning path.
- **Ownership**: `internal/roadmap/`.
- **Key Contribution**: Built the gating engine and dynamic roadmap generator using the `topics.json` DAG. Managed phase unlocks based on mastery thresholds and implemented the Duolingo-style "test-out" bypass mechanic.

## 7. Medic (Practice Loop & Remediation)
- **Role**: Supported the user in practice.
- **Ownership**: Extensions to `internal/services/practice_session.go`, `internal/middleware/ratelimit.go`.
- **Key Contribution**: Moved rate limiting to a distributed Redis Lua script. Injected LLM hints and error analysis into practice sessions. Implemented closed-loop remediation: 3 consecutive failures trigger an automatic difficulty drop and upcoming roadmap regeneration.

## 8. Archivist (Content Ingestion)
- **Role**: Populated the platform.
- **Ownership**: `internal/tutorials/`, `pipeline/ingest/`.
- **Key Contribution**: Defined the schemas for tutorials and curated roadmap templates. Built robust NDJSON parsing endpoints that safely reject individual malformed records without failing batch inserts.

## 9. Notary (QA & Documentation)
- **Role**: The contract enforcer.
- **Ownership**: `backend/api/openapi.yaml`, `backend/docs/API.md`, integration test suites.
- **Key Contribution**: Drafted the OpenAPI 3.1 specification concurrently with development. Built `contract_test.go` and `integration_test.go` to ensure the final system matched the documented curl walkthrough end-to-end.
