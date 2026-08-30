# Target Architecture

```
                         ┌───────────────────────────┐
                         │   Next.js (Frontend)      │
                         │  - Dynamic Roadmap (JIT)  │
                         │  - Single Active Section  │
                         │  - Split Coding / IDE UI  │
                         └───────────┬───────────────┘
                                     │ HTTPS (REST + SSE)
                                     ▼
┌────────────────────────────────────────────────────────────────────────┐
│                         Go Backend (this repo)                         │
│  cmd/server  →  Chi router  →  handlers  →  services  →  repositories  │
│                                                                        │
│  Subsystems:                                                           │
│   - internal/roadmap     (JIT Progressive Roadmap: 1 Active Section)   │
│   - internal/services    (Judge0 Multi-Test-Case Batch Execution)      │
│   - internal/scraper     (Unified LeetCode/Codeforces Problem Scraper) │
│   - internal/templates   (Starter Boilerplate: Python/C++/Java/Go/etc) │
│   - internal/llm         (Z.ai GLM 4.7 Flash client + prompt layer)    │
│   - internal/connectors  (GitHub / LeetCode / Codeforces profile sync) │
│   - internal/evidence    (Upload orchestration, extraction workers)    │
│   - internal/quiz        (Adaptive verification quiz engine)           │
│   - internal/realtime    (Redis pub/sub → SSE gateway)                 │
│   - internal/objectstore (MinIO client, presigned URLs, TTL sweep)     │
│   - internal/oauth       (GitHub/Google OAuth2 flows)                  │
└───────┬───────────────┬───────────────┬───────────────┬────────────────┘
        │               │               │               │
        ▼               ▼               ▼               ▼
 ┌─────────────┐  ┌───────────┐  ┌────────────┐  ┌───────────────────────┐
 │ Postgres 16 │  │  Redis 7  │  │   MinIO    │  │ Judge0 CE (Local      │
 │ + pgvector  │  │ cache/pub │  │ (ephemeral │  │ Docker Sandbox on     │
 │ HNSW index  │  │ sub/queue │  │  uploads)  │  │ port 2358, privileged)│
 └─────────────┘  └───────────┘  └────────────┘  └───────────────────────┘
                                                           ▲
                                                           │ HTTPS
                                                  ┌────────┴────────┐
                                                  │ Z.ai GLM-4.7    │
                                                  │ Flash (external)│
                                                  └─────────────────┘
```

## Key Architectural Principles

1. **Progressive Just-in-Time (JIT) Roadmap**:
   - Only **one section (phase)** of the roadmap is visible and active at any given moment.
   - When the frontend loads (`GET /api/v1/roadmap`), the backend delivers the active section with full tutorials, practice questions, difficulty ratings, user topic ability, and 8-language starter code templates.
   - Upcoming sections are locked and generated/unlocked only after all nodes in the active section reach mastery.

2. **Self-Hosted Sandboxed Execution via Local Judge0**:
   - Judge0 runs locally via Docker Compose on port `2358` without external RapidAPI dependencies or quotas.
   - `ExecuteMultipleTestCases` runs code against every test case individually and stops early on compilation errors to conserve CPU.

3. **Multi-Source Problem Scraping & Template Generation**:
   - `UnifiedScraper` extracts problem body HTML, input/output specifications, time/memory limits, and sample test case inputs/outputs directly from LeetCode GraphQL or Codeforces HTML.
   - `templates.GenerateTemplates` provides ready-to-run starter code with standard I/O driver loops for `Python`, `C++`, `Java`, `JavaScript`, `Go`, `Rust`, `C`, and `Kotlin`.
