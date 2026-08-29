# Target Architecture

```
                         ┌───────────────────────────┐
                         │   Next.js (NOT THIS REPO) │
                         │  Server Components / RSC /│
                         │  Route Handlers           │
                         └───────────┬───────────────┘
                                     │ HTTPS (REST + SSE)
                                     ▼
┌────────────────────────────────────────────────────────────────────┐
│                         Go Backend (this repo)                     │
│  cmd/server  →  Chi router  →  handlers  →  services  →  repos     │
│                                                                    │
│  Subsystems:                                                       │
│   - internal/llm        (Z.ai GLM 4.7 Flash client + prompt layer) │
│   - internal/connectors (GitHub / LeetCode / Codeforces scrapers)  │
│   - internal/evidence   (upload orchestration, extraction workers) │
│   - internal/roadmap    (roadmap generation + gating engine)       │
│   - internal/quiz       (adaptive verification quiz engine)        │
│   - internal/realtime   (Redis pub/sub → SSE gateway)              │
│   - internal/objectstore(MinIO client, presigned URLs, TTL sweep)  │
│   - internal/oauth      (GitHub/Google OAuth2 flows)               │
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
                                                 │ Z.ai GLM-4.7    │
                                                 │ Flash (external)│
                                                 └─────────────────┘
```
