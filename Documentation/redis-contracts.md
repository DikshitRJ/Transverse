# Redis Contracts

The backend utilizes Redis for caching, queues, rate limiting, and pub/sub. The frontend may connect directly for pub/sub if running on the same internal network, although Server-Sent Events (SSE) via `/api/v1/events/stream` is the recommended integration path.

| Purpose | Key/Channel pattern | Notes |
|---|---|---|
| Generic cache | `seen:{userID}`, etc. | For general entity caching |
| LLM response cache | `llm:cache:{sha256}` | TTL per prompt type |
| Rate limiting | `ratelimit:{ip_or_userID}:{bucket}` | Enables correct limiting across multiple API replicas |
| Async job state | `job:{jobID}` | Fast read-through cache (mirrors `llm_jobs` row in Postgres) |
| Per-user event stream | Pub/Sub channel `user:{userID}:events` | JSON payload: `{"type":"job.completed","job_id":...,"data":{...}}`. Consumed by SSE. Event `type` enum: `job.completed`, `job.failed`, `node.unlocked`, `roadmap.updated`, `hint.ready` |
| Auth denylist | `jwt:denylist:{jti}` | Set on logout/refresh-rotation; TTL = token remaining life |
