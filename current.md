# Transverse Adaptive Heuristic Engine - Current Status

This document provides a comprehensive overview of the current state of the Transverse codebase as of the latest milestone. It details the system architecture, file-by-file explanations, and maps the completed work against the original Notion workspace requirements.

---

## 📋 Notion Requirements: What's Done vs. What Isn't

Based on the Notion workspace (`OVERVIEW`, `USP`, `Tech Stack`, `PROBLEM STATEMENT`, `What we have?`), here is the status of the requirements:

### ✅ Completed
- **100% Golang Backend**: The entire Python reference model (`heuristic_model/`) has been successfully rewritten and adapted into a scalable Go binary. No Python remains in the runtime.
- **Vector Database (pgvector)**: Migrated to PostgreSQL 16 with the `pgvector` extension. The HNSW indexing schema is fully implemented (`sql/005_indexes.sql`).
- **Glicko & IRT (Theta) Engine**: Both the 1PL Rasch Model (IRT) and Glicko-2 psychometric algorithms have been faithfully ported and adapted for DSA/CP.
- **7-Factor Heuristic Scorer**: The 6-factor JEE scorer was extended to a 7-factor scorer for DSA/CP, factoring in: Difficulty Fit, Concept Similarity, Topic Progression, Novelty Factor, Immediate Reinforce, Platform Diversity, and Carelessness Penalty.
- **Code Compilation Engine (Judge0)**: The backend fully integrates with Judge0. It handles proxy submissions, polls verdicts, and natively understands `status_id = 3 (Accepted)` vs `status_id = 7 (Compilation Error)`.
- **Similar Concept Mapping**: Powered by the new Go ONNX embedding pipeline (`BAAI/bge-small-en-v1.5`), semantic concept similarity searches (`FindSimilar`) are executed natively during "After Wrong" flows.
- **Single Dockerized Stack**: The entire application (Postgres + pgvector + Go Backend API) is unified into a single `docker-compose.yml`.

### ⏳ Pending / Out of Scope (For Now)
- **Frontend / UI**: No frontend application exists yet.
- **LLM Integration (Z.ai GLM 4.7 Flash)**: The roadmap generation features using LLMs have not yet been wired up in the handlers.
- **Authentication (OAuth2)**: The JWT middleware is present (`middleware/auth.go`), but the actual OAuth login flow is stubbed for development.
- **Redis Pub/Sub**: The system currently uses an in-memory TTL cache (`internal/cache/memory.go`). A Redis adapter needs to be added for true distributed caching/pub-sub if scaling beyond a single instance.

---

## 🗂️ Codebase Architecture (File-by-File)

The backend root is located at `/mnt/Data/smart_india_hackathon/Screening_transverse/backend/`. The architecture strictly follows idiomatic Go structures (Clean Architecture principles).

### `/backend/cmd/`
- **`server/main.go`**: The primary entry point for the REST API. It wires together the database pool, cache, topic graph, repositories, services, HTTP middleware, Chi router, and starts the server with graceful shutdown.

### `/backend/pipeline/seed/` (The Embedding & Ingestion Pipeline)
This replaces the old Python data pipeline. It ingests the ~21,000 raw JSON problems, generates 384-dimensional vectors via CGO ONNX, and upserts them to Postgres.
- **`main.go`**: The CLI tool to run migrations and trigger the data ingestion.
- **`loader.go` & `loader_test.go`**: Loads Codeforces, AtCoder, CSES, and LeetCode JSON files from disk and deduplicates them using a strict priority order.
- **`normalizer.go` & `normalizer_test.go`**: Calibrates various platform difficulties into a unified 800-3500 Glicko rating scale and maps tags to the unified topic graph.
- **`embedder.go`**: Orchestrates a concurrent channel-based worker pool to feed the tokenizer and ONNX Runtime for batch embeddings.
- **`seeder.go`**: Upserts the processed, embedded problems into Postgres using `pgx.Batch` transactions.

### `/backend/internal/embedding/`
- **`provider.go`**: Defines the `Provider` interface for vector generation.
- **`tokenizer.go` & `tokenizer_test.go`**: A pure Go BPE WordPiece tokenizer compatible with Hugging Face BERT models, handling truncation and padding.
- **`onnx.go` & `onnx_test.go`**: Wraps the `yalue/onnxruntime_go` library to execute the `bge-small-en-v1.5` model via CGO. Extracts the `[CLS]` token and applies L2 normalization.

### `/backend/internal/services/` (The Mathematical Core)
- **`practice_session.go`**: The master state machine. Handles `Start`, `Submit`, `Skip`, and `Close`. Wires Judge0 evaluations, IRT updates, HNSW vector lookups, and transactional data persistence.
- **`practice_analytics.go` & `practice_analytics_test.go`**: Calculates user `LearningDNA` (Carelessness Index, Peak Performance Hour, Avg Velocity) via EMA smoothing.
- **`scoring.go` & `scoring_test.go`**: The 7-factor `ScoreCandidate` logic and the `PickBestProblem` routing.
- **`theta.go` & `theta_test.go`**: The 1PL Item Response Theory implementation. Adjusts a student's latent ability ($\theta$) based on time spent vs expected time.
- **`glicko.go` & `glicko_test.go`**: The complete Glicko-2 Illinois psychometric algorithm. Computes new ratings and volatility at the end of a session.
- **`judge0.go`**: The HTTP client wrapper connecting to the Judge0 code execution engine.
- **`graph_service.go` & `graph_service_test.go`**: Resolves arbitrary string tags/aliases to canonical IDs in the Topic DAG.
- **`helpers.go` & `helpers_test.go`**: Contains core math functions (`clamp`, `cosineSimilarity`, `dotProduct`, `l2Norm`).
- **`practice_cache.go`**: Defines dynamic cache keys (`seen:{userID}`, etc.).

### `/backend/internal/repository/`
- **`problem_repo.go`**: Handles `GetByScope` and `FindSimilar` (the HNSW nearest-neighbor vector query `ORDER BY embedding <=> $1`).
- **`session_repo.go`**: Saves ongoing practice sessions using atomic JSONB array appends (`responses = responses || $1::jsonb`).
- **`user_repo.go`**: Manages the `users` table, including their global Glicko ratings and `LearningDNA`.
- **`stats_repo.go` & `problem_stats_repo.go`**: Tracks topic-level mastery scores and problem-level attempt counters.

### `/backend/internal/handlers/`
- **`practice_handler.go`**: Decodes JSON, extracts JWT claims, and routes HTTP `POST /api/v1/practice/submit`, `/start`, `/skip`, `/close` to `practice_session.go`.
- **`user_handler.go`**: Serves the user profile and historical session metrics.
- **`health_handler.go`**: Simple `GET /health` to expose Postgres pool connection metrics.
- **`helpers.go`**: `writeJSON` and `writeError` HTTP envelope standardizers.

### `/backend/internal/models/`
- **`db_models.go`**: Maps exactly to the PostgreSQL schema (`User`, `Problem`, `PracticeSession`, `SessionResponse`).
- **`dto.go`**: Contains strictly defined input/output JSON schemas for the REST API (e.g., `SubmitRequest`, `StartResult`).

### `/backend/internal/middleware/`
- **`auth.go`**: Validates `Bearer` JWT tokens and injects user context.
- **`ratelimit.go`**: Thread-safe IP token bucket rate limiter to prevent abuse on the scoring engine endpoints.

### `/backend/internal/graph/`
- **`graph.go` & `graph_test.go`**: Defines the interface for the DAG.
- **`loader.go`**: Parses `topics.json` and builds a fast, in-memory reverse-lookup map for tag aliases.

### `/backend/internal/config/`, `/cache/`, `/database/`
- **`config.go`**: Loads env vars into a strictly typed struct.
- **`memory.go` & `cache.go`**: An application-wide in-memory TTL caching layer to relieve database pressure.
- **`postgres.go`**: Bootstraps the `pgxpool` with optimized connection limits.

### Infrastructure & Roots
- **`backend/Dockerfile`**: A multi-stage build downloading the ONNX C++ shared libraries and compiling the Go binary statically.
- **`backend/docker-compose.yml`**: Provisions `pgvector/pgvector:pg16` and the API, automatically running SQL migrations via volume mounts.
- **`backend/sql/`**: Contains 5 sequenced SQL files establishing the tables, JSONB columns, and `HNSW` vector indexes.
- **`backend/data/topics.json`**: The canonical knowledge graph DAG, housing 30 parent topics and 258 mapped tag aliases (e.g., `dynamic programming` $\rightarrow$ `dp`).
- **`backend/go.mod`**: Dependencies containing `pgx/v5`, `chi/v5`, `pgvector-go`, and `onnxruntime_go`.

---

## 🎯 Current Status

The Go backend code compiles **with zero errors**. The duplicate file conflict between `practice_session.go` and `practice_service.go` has been completely resolved.

**Ready for deployment!** You can start the stack right now using:
```bash
cd backend
docker-compose up -d
go run ./pipeline/seed
```
