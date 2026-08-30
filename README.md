# Transverse

Transverse is a skill verification and adaptive learning platform. This repository contains the Go backend, which provides an asynchronous, LLM-orchestrated, and highly private learning pipeline. The Next.js frontend is built against this backend's formal API contract.

## Features

- **Deep Skill Verification**: Uploads public profiles (GitHub, LeetCode, Codeforces) or private documents (resumes, codebase zips). Evidence is extracted into JSON hypotheses and the raw files are immediately deleted (zero-cloud privacy).
- **Adaptive Quiz Engine**: Hypotheses are tested via a deterministic Glicko-2 scoring engine. Correct answers confirm hypotheses and raise topic ratings; failures debunk them.
- **Dynamic Roadmaps**: An LLM orchestrates a DAG-based learning progression. Nodes are gated by mastery thresholds. Users can "test out" of nodes they already know.
- **Closed-Loop Remediation**: If a user fails 3 times on the same practice topic, the engine automatically drops the difficulty threshold, triggers an LLM error analysis, and regenerates upcoming roadmap phases.

## Repository Structure

- `backend/cmd/server/`: The main entry point and Chi router wiring.
- `backend/internal/`: The core domain logic (Clean Architecture):
  - `evidence/`, `connectors/`: Ingests and scrubs user signals.
  - `llm/`, `jobs/`, `realtime/`: Async LLM orchestration and SSE streaming.
  - `hypothesis/`, `quiz/`: The adaptive assessment loop.
  - `roadmap/`: Progression and gating engine.
  - `oauth/`, `middleware/`: Identity and access management.
- `backend/sql/`: Postgres schema migrations.
- `backend/pipeline/seed/`: Offline CLI tool for seeding problems and tutorials.
- `frontend/`: Next.js 15 (App Router) frontend application.
- `Documentation/`: Contains the formal `openapi.yaml`, architecture diagrams, and end-to-end integration walkthroughs.
- `data/`: Contains `topics.json`, the canonical topic DAG, generated seed data, and `tutorial.json`.

## Status

**Completed:**
- Full infrastructure configuration (Postgres/pgvector, Redis, MinIO, Judge0).
- All 11 milestones of the backend architecture (Evidence, Connectors, OAuth, LLM, Quiz, Roadmap, Practice Loop, Remediation, Content Ingestion).
- Formal API contracts (OpenAPI 3.1) and testing suites.
- Next.js Frontend implementation (UI and Server Actions).
- Content ingestion for problems and tutorials.

**Left to do:**
- Ongoing improvements and bug fixes.

## Getting Started

1. Set up your `.env` files (use `backend/.env.example` and `frontend/.env.example` as templates).
2. Start the entire infrastructure and applications (Backend + Frontend + DB + Redis + Judge0 + MinIO):
   ```bash
   docker-compose up --build -d
   ```
3. The frontend is accessible at `http://localhost:3000` and the backend at `http://localhost:8080`.
4. Refer to `Documentation/end-to-end-walkthrough.md` to simulate a user journey.
