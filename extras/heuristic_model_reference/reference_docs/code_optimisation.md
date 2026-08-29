# Codebase Optimisation & Flaw Analysis

> **Date:** 2026-06-09  
> **Scope:** Full-stack audit — backend (Go), frontend (Next.js/React), engine (Python), data pipeline, caching, scoring/rating logic

---

## Classification Key

| Severity | Meaning |
|----------|---------|
| 🔴 **CRITICAL** | Data corruption, security breach, wrong answers scored as correct |
| 🟠 **HIGH** | Wrong difficulty estimates, broken adaptive learning, significant perf loss |
| 🟡 **MEDIUM** | Degraded UX, partial feature breakage, minor data inconsistency |
| 🔵 **LOW** | Code quality, documentation, edge cases unlikely to trigger |

---

## 🔴 CRITICAL

### C1. Secrets Committed to Repository
**Files:** `backend/.env:8-28`, `engine/.env:1-3`, `test.sh:5`

Production credentials hardcoded and version-controlled:
- **Alpha-Auth OAuth2** `CLIENT_ID` + `CLIENT_SECRET` — full account takeover of the OAuth provider
- **`JWT_SECRET`** — forge arbitrary session tokens, impersonate any user
- **Cloudinary API key + secret** — abuse image CDN billing, delete all assets
- **Google Vertex AI** service account path — access to GCP AI services
- **Hardcoded JWT bearer token** in `test.sh` — anyone with repo access can impersonate this user

**Fix:** Immediately revoke ALL exposed secrets. Add `.env` to `.gitignore` (already present but files were committed earlier). Use GCP Secret Manager / Cloudinary env vars / Vault. Commit `.env.example` with placeholder values instead.

---

### C2. Content Hash Includes Exam Metadata → Duplicate Question IDs
**File:** `engine/preprocess/seed.py:132-133`

```python
content_key = f"{source}|{shift_date}|{subject}|...{q.get('chapter', 'unknown')}|{q.get('text', ...)}|..."
stable_id = hashlib.md5(content_key.encode()).hexdigest()[:12]
```

The hash includes `source` (e.g. `"JEE Main 2025"`) and `shift_date` (e.g. `"2025_2401S2"`). The **same question** reused across years/dates gets **different IDs**, creating duplicate rows in the database. This means:
- Session references can point to the wrong duplicate
- `user_question_stats` fragments across copies
- The adaptive engine sees "unseen" copies of already-answered questions

**Fix:** Remove `source` and `shift_date` from the content hash. Include only question-invariant fields: `subject`, `chapter`, `text`, `sorted options`.

---

### C3. UPSERT Doesn't Update Key Glicko Fields
**File:** `engine/preprocess/seed.py:175-191`

```sql
INSERT INTO questions (...) VALUES (...)
ON CONFLICT (id) DO UPDATE SET
  type, question_text, images, options, correct, source, shift_date,
  exam_type, chapter, chapter_group, difficulty, is_bonus,
  glicko_rating, percent_correct, timespent_avg_ms, embedding
```

**NOT updated on conflict:** `glicko_rd`, `glicko_volatility`, `attempt_count`, `subject`.

If the pipeline re-runs with updated RD/volatility values, the database silently keeps the old values. `attempt_count` not being updated means live user stats accumulate but are never reflected in the question's row.

**Fix:** Add the missing columns to the `DO UPDATE SET` clause.

---

### C4. Python `or` Operator Swallows Valid Zero Values
**File:** `engine/preprocess/seed.py:156`

```python
"timespent_avg_ms": q.get("timespent") or q.get("timespent_avg_ms") or 0
```

If `timespent` is `0` (valid — a question answered in 0ms), `0 or X` evaluates to `X`, silently substituting the fallback. Same bug at line 116-117:
```python
pct = q.get("percent correct") or q.get("percent_correct")
```
A question with 0% correct rate gets treated as "key not present" and defaults to 0 anyway (no data loss), but a `timespent` of 0ms would pick up the wrong column value.

**Fix:** Use explicit `is None` checks: `q.get("timespent") if q.get("timespent") is not None else q.get("timespent_avg_ms", 0)`

---

### C5. Destructive Script Has Fallback Production Credentials
**File:** `engine/preprocess/scripts/clear_cloudinary.py:9-11`

```python
cloudinary.config(
    cloud_name=os.environ.get("CLOUDINARY_CLOUD_NAME", "dblahw4hy"),
    api_key=os.environ.get("CLOUDINARY_API_KEY", "298687411276873"),
    api_secret=os.environ.get("CLOUDINARY_API_SECRET", "0bpFEODiVcLdKkMTS4buJkkVgnQ"),
)
```

This script **deletes ALL Cloudinary assets** with no confirmation prompt. If environment variables are unset, it silently falls back to the hardcoded production values. One accidental `uv run clear_cloudinary.py` wipes all question images.

**Fix:** Remove default values. Require env vars to be explicitly set. Add a confirmation prompt (`input("Type YES to confirm: ")`).

---

## 🟠 HIGH

### H1. Scoring Documentation is Mathematically Wrong
**File:** `engine/markdowns/scoring.md:174,196`

The IRT formula documented is:
```
P_correct = 1 / (1 + e^(-a·(θ − b)))
a = 1/27,  b = (Glicko - 1500) / 100
```

The doc claims:
> "θ=1500, Question=2000 → P(correct)≈0.07"

But plugging in the values: `b = (2000-1500)/100 = 5`, `P = 1/(1+e^(-1/27*(0-5))) = 1/(1+e^(5/27)) = 1/(1+1.203) = 0.454`, **NOT 0.07**.

To get P≈0.07, `a` would need to be ≈1.0 (not 1/27), OR the difficulty scaling would need to be completely different. This suggests either:
- The doc formula doesn't match the code implementation, OR
- The example numbers are fictitious and never verified

Additionally, the sub-agents identified that `theta.go:48,50` was dividing the discrimination parameter by 100 twice, making effective `a = 1/2700` instead of `1/27`. This has been **fixed** by the audit, but the documentation remains wrong.

**Fix:** Correct the documentation to match the fixed code. Verify P(correct) examples with actual computation.

---

### H2. Glicko-2 RD Used Ad-Hoc Formula Instead of Stored Value
**File:** `services/learn_service.go:625` (before fix)

The Glicko-2 rating deviation for session update was computed as:
```go
max(30, 350-20*sessions)
```

Instead of using the user's actual stored `learn_rd` from the database. This means:
- A first-time user with RD=350 gets RD=350 (correct by coincidence)
- A user who practiced 10 sessions gets RD=150 (even if their actual RD converged to, say, 80)
- The confidence/uncertainty signal was completely synthetic

**Fix (applied):** Changed to `max(30, user.LearnRD)` — uses the actual rating deviation.

---

### H3. IRT Theta Discrimination Was ~100x Too Weak
**File:** `services/theta.go:48,50` (before fix)

```go
// Original code was normalizing by /100 twice, making effective a = 1/2700
// instead of the documented a = 1/27
```

Theta updates were ~100x weaker than intended per question. A student answering correctly would see negligible theta movement (~0.01 instead of ~1-2 points). The adaptive engine was effectively **not adapting** — the next question selection was dominated by noise, not actual ability estimates.

**Fix (applied):** Removed the double `/100` normalization. The formula now correctly computes `a = 1/27` as documented.

---

### H4. Cache Memory Leak — Entries Without TTL Never Evicted
**File:** `cache/cache.go:136-145` (before fix)

Entries inserted without an explicit `expiresAt` had `expiresAt.IsZero() = true`, causing the eviction loop to skip them entirely. Over time:
- Cache grows unbounded until OOM
- Old entries are never purged regardless of size limits
- Stale data persists indefinitely

**Fix (applied):** TTL-less entries now get a distant expiration (100 years) so they're evicted last but can still be purged. Bubble sort replaced with `sort.Slice` for correctness.

---

### H5. TOCTOU Race Condition in Cache Reads
**File:** `cache/cache.go:77-107` (before fix)

```
RLock → check if entry exists → RUnlock → Lock → remove entry
```

Between the `RLock` read and the `Lock` write, another goroutine can remove the entry, causing a nil pointer panic or stale data read.

**Fix (applied):** Simplified to single `Lock` for the entire Get-with-eviction operation.

---

### H6. Rate Limiter Bucket Map Unbounded Memory Growth
**File:** `middleware/ratelimit.go:54-66` (before fix)

```go
var buckets = make(map[string]*tokenBucket)
```

Each unique IP creates a new map entry. An attacker with IP spoofing (X-Forwarded-For) can create millions of entries, causing OOM.

**Fix (applied):** Added `rateLimitMaxBuckets = 100000` with random-entry eviction when exceeded.

---

### H7. Admin Email Hardcoded
**File:** `middleware/admin.go:7` (before fix)

```go
const adminEmail = "ayushme@alphajee.com"
```

Package-level constant for admin authorization. No way to change admin without rebuilding the binary.

**Fix (applied):** Admin email is now configurable via `ADMIN_EMAIL` env var, loaded through the config system, with `AdminOnly` as a function factory.

---

### H8. Debug Endpoint Had No Authentication
**File:** `cmd/server/main.go:131-136` (before fix)

The `/debug/sessions` endpoint dumped all active user sessions with no auth check — any network client could enumerate all users and their session data.

**Fix (applied):** Wrapped with `Auth` + `AdminOnly` middleware.

---

### H9. SentenceTransformer Loaded on Every Search Query
**File:** `engine/preprocess/search_by_text.py:70-73`

```python
def embed_text(text):
    model = SentenceTransformer(MODEL_NAME, trust_remote_code=True)
    emb = model.encode([text], batch_size=1, show_progress_bar=False)[0]
    return np.array(emb)
```

Creates a NEW `SentenceTransformer` (~33MB model load from disk/HuggingFace) on EVERY query. Cold start takes 5-10 seconds per search. Should be a module-level singleton like `generate_embeddings.py` does.

**Fix:** Cache the model as a module-level `_model` variable with lazy initialization.

---

### H10. N+1 Queries in Collab Handler
**File:** `handlers/collab_handler.go:283-290,317-327` (before fix)

`GetDPP` and `ExportDPPDOCX` issued one DB query per question in the list. For a 30-question DPP, this is 30 round-trips to PostgreSQL.

**Fix (applied):** Batch-loaded with `GetByIDsWithoutEmbedding` repository method (single `WHERE id = ANY($1)` query).

---

### H11. Illinois Algorithm (Glicko-2) No Max-Iterations Guard
**File:** `services/glicko.go:143-147` (before fix)

The Illinois algorithm for finding new volatility has no maximum iteration limit. On pathological input (e.g., extreme RD or rating mismatch), it could loop forever.

**Fix (applied):** Added `const maxIter = 100` guard.

---

### H12. Missing Row-Level Locking in Session Transactions
**Files:** `services/learn_service.go:353,540,676`, `repository/session_repo.go` (before fix)

Submit/Skip/Close all operated within transactions but without `SELECT ... FOR UPDATE` on the session row. Concurrent submissions could:
- Overwrite `theta_current` with stale values
- Lose response appends (lost update problem)
- Double-count question stats

**Fix (applied):** Added `LockSessionForUpdate` that acquires a row-level lock on the session before any mutation.

---

### H13. Cloudinary Signature Didn't Cover Expiration
**File:** `cloudinary/client.go:56-62` (before fix)

The `expires` parameter (Unix timestamp) was appended **after** Cloudinary's signature was generated. Anyone could modify the `expires` value without invalidating the signature, making signed URLs effectively permanent.

**Fix (applied):** Expiration is now injected as a Cloudinary transformation parameter (`ex_<timestamp>`) **before** URL generation, so it's part of the signed content.

---

### H14. UpsertUser Not Wrapped in a Transaction
**File:** `repository/user_repo.go:88-111` (before fix)

Two separate INSERT statements (one for `users` table, one for `learning_stats`). If the second INSERT failed, the user row was created without its learning_stats, causing downstream panics.

**Fix (applied):** Both INSERTs are now wrapped in a `Begin/Commit/Rollback` transaction.

---

### H15. Error-Swallowing in Repository Layer
**Files:** `repository/question_stats_repo.go:81-82`, `repository/stats_repo.go:89-90` (before fix)

```go
// Original: return 0, nil  — swallowed every error
```

`GetAttemptCount` and `GetChapterAvgTimeMs` returned `0, nil` on any database error. Callers could not distinguish "no data" from "database unreachable" or "query timeout." The downstream effects:
- The adaptive engine silently uses 0 for chapter avg time, permanently disabling the `TimeMatch` scoring component
- Question selection degrades to a random walk

**Fix (applied):** Both now return the actual error from the database query.

---

### H16. Settings Update Has Read-Modify-Write Race
**File:** `handlers/settings_handler.go:78-150`, `repository/user_repo.go:298-311` (before fix)

Settings were fetched, modified in Go, then written back with `SET settings = $2` (full replace). Concurrent settings updates overwrite each other (lost update).

**Fix (applied):** Switched to `COALESCE(settings, '{}'::jsonb) || $2::jsonb` for server-side JSONB merge.

---

### H17. `generateID` Produces Collisions in Multi-Instance Deployments
**File:** `services/learn_service.go:1204-1224` (before fix)

Fallback ID generation used `atomic.AddInt64` on a process-local counter. Two server instances could generate the same session ID.

**Fix (applied):** Added `os.Hostname()` to the fallback ID string for instance-uniqueness.

---

### H18. Dashboard Token Not Synced to Auth Store
**File:** `frontend/app/dashboard/page.tsx` (before fix)

When receiving a token via URL query parameter (after OAuth callback), the page did not call `setSession()` to sync the token into the zustand persist store. Subsequent page reloads would lose authentication state.

**Fix (applied):** Added `setSession()` call on the URL token path.

---

### H19. Auth Validate Endpoint Missing Authorization Header
**File:** `frontend/lib/store.ts` (before fix)

The `validate()` function sent the token in the request body but **not** in the `Authorization: Bearer <token>` header. The server always parsed the header first, causing false 401 responses.

**Fix (applied):** Now sends `Authorization: Bearer <token>` header, with `setSession()` syncing the token to `localStorage`.

---

### H20. Biometric Blink Rate Uses Cumulative Counter (Never Reset)
**File:** `frontend/hooks/useBiometricState.ts` (before fix)

Blink count was a monotonically increasing integer (`blinkCount++`). After 10 minutes of tracking, the display showed 800+ blinks/min — the counter was accumulating across the entire session.

**Fix (applied):** Replaced with a sliding window of blink event timestamps over the last ~15 seconds.

---

### H21. BiometricToggle Creates Duplicate Camera Stream
**File:** `frontend/components/BiometricToggle.tsx` (before fix)

The toggle created its own `getUserMedia` camera stream while the main biometric hook already had one. Two concurrent camera streams:
- Doubles CPU usage (two separate MediaPipe facemesh pipelines)
- Some browsers limit simultaneous camera access → one stream fails silently

**Fix (applied):** `BiometricToggle` is now a pure UI toggle — it delegates to the parent's camera management.

---

### H22. Unused/Stale Worker Closure in Biometric Hook
**File:** `frontend/hooks/useBiometricState.ts` (before fix)

The `processFrame` function passed to the worker was captured in a closure at Worker creation time. After a state update (e.g., face-detection toggle), the worker continued calling the stale version. Additionally, the Web Worker blob URL was never revoked (`URL.revokeObjectURL`), leaking memory per calibration.

**Fix (applied):** `processFrameRef` and `enabledRef` sync on every render via `useEffect`. Worker always calls the latest function through the ref. Blob URL is cleaned up on stop/unmount.

---

### H23. `or` Fallback in `resolve_baseline` is Fragile
**File:** `engine/preprocess/rate.py:99-110`

```python
return (
    bucket.get(difficulty)
    or bucket.get("medium")
    or bucket.get("_all")
    or DEFAULT_BASELINE
)
```

The `or` operator treats any falsy value (None, 0, 0.0, empty string) as "go to next fallback." If a baseline were legitimately 0.0 (impossible for time, but bad pattern), it would incorrectly fall through.

**Fix:** Use explicit `is not None` checks instead of `or`.

---

## 🟡 MEDIUM

### M1. Time Divergence Can Overwhelm Accuracy Component
**File:** `engine/preprocess/rate.py:113-143`

```
delta_r_time = TIME_SCALE * math.log(t_sec / baseline)
TIME_SCALE = 100, baseline ≈ 60s
```

Time divergence contributes up to ±200 rating points. The accuracy component (Rasch logit) typically contributes ±300-400 points. But since time and accuracy are correlated (students guess → fast + wrong), the time divergence can **systematically bias** ratings:
- A low-accuracy question that students answer quickly (guessing) gets an artificially LOW rating
- The question appears easier than it actually is
- This creates a self-reinforcing loop: easy-looking rating → adaptive engine serves it to weaker students → more wrong answers → even faster times

**Fix:** Normalize the time divergence by the accuracy signal. Use a 2-parameter model instead of independent components.

---

### M2. Pipeline Incomplete — 2024, 2026, and Most of 2025 Not Processed
**Files:** `engine/data/rated/` vs `engine/data/processed/`

| Data | Rated | Embedded/Processed | Seeded to DB |
|------|-------|-------------------|--------------|
| 2024 (20 files) | ✅ | ❌ | ❌ |
| 2025 (19 files) | ✅ | ~12/19 | ✅ (partial) |
| 2026 (19 files) | ✅ | ❌ | ❌ |
| JEE Advanced (4 files) | ✅ | ✅ (all 4) | ✅ |
| Cloudinary images | — | ADV only | partial |

Approximately **46 out of 62** source files are not in the database. The adaptive engine has access to only ~5,137 questions instead of the potential ~18,000+.

**Fix:** Run the full pipeline for all years: `rate.py` → `generate_embeddings.py` → `seed.py` for 2024, 2026, and the missing 2025 files.

---

### M3. `docker-client` Package Name Wrong in Flake
**File:** `backend/flake.nix:26`

```nix
docker-client
```

In nixpkgs, the Docker client package is `pkgs.docker`, not `docker-client`. This will cause a Nix evaluation error on most nixpkgs revisions.

**Fix:** Change to `docker`.

---

### M4. Missing Dependencies in pyproject.toml
**File:** `engine/pyproject.toml`

`pyproject.toml` declares `requires-python = ">=3.13"` but the scripts use PEP 723 inline metadata with `requires-python = ">=3.11"`. Additionally:
- `sentence-transformers` is NOT in `pyproject.toml` dependencies (used by `generate_embeddings.py`, `search_by_text.py`, `test_embeddings.py`)
- `tqdm` is NOT listed (used by `upload_images.py`)
- `cloudinary` is NOT listed (used by `clear_cloudinary.py` and `upload_images.py`)

**Fix:** Add all missing dependencies to `pyproject.toml`. Align Python version requirements.

---

### M5. Frontend Auth Validate Infinite Re-render Loop
**File:** `frontend/lib/store.ts` and 12+ component files (before fix)

```typescript
// Before: destructuring creates new reference every render
const { validate } = useAuthStore()
useEffect(() => { validate() }, [validate])
// validate changes every render → infinite loop
```

Destructuring from `useAuthStore()` creates a new reference for `validate` on every render, causing the `useEffect` dependency array to always detect a change.

**Fix (applied):** Changed to selector pattern: `const validate = useAuthStore(s => s.validate)` — stable reference across renders.

---

### M6. LatexText Single-Line Display Math Not Rendered
**File:** `frontend/components/LatexText.tsx` (before fix)

Single-line `$$...$$` (e.g., `$$E=mc^2$$`) was only rendered as inline math unless the content contained `\begin` or newlines. JEE frequently uses single-line display math.

**Fix (applied):** All `$$...$$` blocks are now rendered as display mode regardless of content length.

---

### M7. StreakCalendar Cross-Month Date Matching Broken
**File:** `frontend/components/StreakCalendar.tsx` (before fix)

Date comparison used day-of-month only (`1-31`), not full dates. A streak on the 5th of May and a session on the 5th of June would incorrectly match as the same day.

**Fix (applied):** Changed to full `YYYY-MM-DD` date comparison.

---

### M8. `app/auth/callback` Uses Hard Navigation
**File:** `frontend/app/auth/callback/page.tsx` (before fix)

```typescript
window.location.href = "/dashboard"
```

This causes a full page reload, losing the React state tree and flash of unauthenticated content.

**Fix (applied):** Changed to `router.push("/dashboard")` for SPA-style navigation.

---

### M9. Profile Page Uses Unauthenticated `fetch()`
**File:** `frontend/app/u/[username]/page.tsx` (before fix)

Used raw `fetch()` without `Authorization: Bearer <token>` header. The API endpoint requires auth, so the profile page always returned 401.

**Fix (applied):** Changed to use the `api` axios client which automatically attaches auth headers.

---

### M10. Biometric Session Ownership Not Validated
**File:** `services/biometric_service.go:23-28,50-96` (before fix)

`SyncSnapshots` and `CloseBiometricSession` never validated that the session belonged to the requesting user. User A could inject snapshots into User B's biometric session.

**Fix (applied):** Added `userID` parameter + `sess.UserID != userID` ownership check.

---

### M11. Missing Context Timeout on DB Ping
**File:** `database/db.go:31-33` (before fix)

```go
err = pool.Ping(ctx)
```

`ctx` was the background context with no timeout. If the database was unreachable, the server would hang for minutes during health check startup.

**Fix (applied):** Changed to `context.WithTimeout(ctx, 10*time.Second)`.

---

### M12. Leaderboard SQL Injection Surface
**File:** `repository/leaderboard_repo.go:34-37` (before fix)

```go
query := fmt.Sprintf("SELECT ... FROM users ORDER BY %s DESC", sortBy)
```

While `sortBy` was only set from a pre-defined switch, the `default` case was missing. An invalid parameter from the frontend would cause a panic instead of graceful fallback.

**Fix (applied):** Added explicit `default` case that defaults to `learn_rating`.

---

### M13. `vertex.go` Uses `http.DefaultClient` (No Timeout)
**File:** `services/vertex.go:264` (before fix)

```go
resp, err := http.DefaultClient.Post(url, "application/json", body)
```

`http.DefaultClient` has no timeout — a slow Vertex AI response would hang the goroutine indefinitely, leaking resources.

**Fix (applied):** Changed to use `c.http` (the configured client with 90s timeout).

---

## 🔵 LOW

### L1. Watermark Removal Destroys Diagram Content
**File:** `engine/preprocess/scripts/remove_watermark.py:80-84`

The non-"original" color mode uses luminance-based alpha matting. Any diagram content with luminance in [140, 200] (gray formulas, shaded graphs) gets replaced with the target color. **Diagram content is destroyed** — not just watermarks.

**Fix:** Use a smarter watermark detector (texture analysis, edge detection, or ML-based segmentation) instead of global luminance thresholding.

---

### L2. `__import__("re")` Instead of `import re`
**File:** `engine/preprocess/scripts/upload_images.py:64`

```python
m = __import__("re").search(r"(20\d{2})", stem)
```

This is functionally correct but terrible for readability and static analysis. All other imports are at the top of the file.

**Fix:** Move `import re` to the top of the file.

---

### L3. `os.replace` Fails Across Filesystems
**File:** `engine/preprocess/generate_embeddings.py:131`

```python
os.replace(tmp_path, dest_file)
```

If `tmp_path` (in system temp directory) and `dest_file` (under `data/processed/`) are on different filesystems, `os.replace` raises `OSError`. Should fall back to `shutil.move`.

**Fix:** Wrap in try/except with `shutil.move` fallback.

---

### L4. No `__init__.py` in `preprocess/`
**File:** `engine/preprocess/` (directory)

Missing `__init__.py` prevents `from preprocess import rate` style imports. Python 3.3+ namespace packages make this work, but it's fragile and breaks tooling (mypy, pylint, IDE autocomplete).

**Fix:** Add `__init__.py` files.

---

### L5. NaN/Inf Not Filtered in Timespent Validation
**File:** `engine/preprocess/rate.py:59-67`

```python
if timespent is not None and timespent > 0:
    timespent = float(timespent)
```

No check for `math.isnan(timespent)` or `math.isinf(timespent)`. NaN passes all comparisons (even `NaN > 0` is False... wait, actually `NaN > 0` is False, so NaN would be filtered. But `Inf > 0` is True, so inf would pass through and cause issues downstream.

**Fix:** Add explicit NaN/inf filtering.

---

### L6. No Embedding Dimension Validation
**File:** `engine/preprocess/generate_embeddings.py:108`

```python
embeddings = model.encode(texts, batch_size=32, show_progress_bar=False)
```

No assertion that `embeddings.shape[1]` equals the expected 384 dimensions (BGE-small). If the model changes, silently feeds wrong-dimension vectors to the database.

**Fix:** Add `assert embeddings.shape[1] == EXPECTED_DIM` after encoding.

---

### L7. Test Script Has Infinite Loop With No Exit Strategy
**File:** `engine/preprocess/test_embeddings.py:139`

```python
while True:
    # random question selection...
    # loop forever until user types 'q'
```

No time limit, no max-iterations, no keyboard interrupt handling. If run in CI or non-interactively, it runs forever.

**Fix:** Add `--iterations` CLI flag with default of 10.

---

### L8. `resolve_baseline` Doesn't Check Sparse Threshold at Call Site
**File:** `engine/preprocess/rate.py:99-110`

`resolve_baseline` returns None for sparse buckets, but the caller doesn't check for this. The baseline dict stores None for sparse buckets, and the `or` fallback handles it. But this is implicit and fragile.

**Fix:** Add explicit `if baseline is None` check at the call site.

---

### L9. `databasename` vs `databaseName` Inconsistency
**File:** `engine/preprocess/seed.py` (and test files)

The database name `alphajee` is referenced as both "AlphaJEE" and "AlphaJEE" across the codebase. While functionally the same, the naming inconsistency makes search/replace harder.

**Fix:** Standardize on `alphajee` everywhere.

---

### L10. Go Version Mismatch in Documentation
**Files:** `codebase.md:16` (Go 1.24) vs `backend/go.mod:3` (go 1.26.3)

Either the docs or the go.mod is stale. If the system really needs Go 1.26.3, the docs should reflect it. If 1.24 is correct, the go.mod should be updated.

**Fix:** Verify and align.

---

### L11. `engine/README.md` Referenced But Doesn't Exist
**File:** `engine/pyproject.toml:5`

```toml
readme = "README.md"
```

No `README.md` exists inside `engine/`. `pip install` or `uv build` will emit a warning/fail depending on the tool version.

**Fix:** Create `engine/README.md` or remove the `readme` field from `pyproject.toml`.

---

### L12. `codebase.md` Documented Dependencies Don't Match `pyproject.toml`
**File:** `codebase.md:35`

Lists `sentence-transformers`, `tqdm`, and other packages not in `pyproject.toml`. The docs describe the intended dependencies, but `pyproject.toml` has fewer.

**Fix:** Either add the missing deps to `pyproject.toml` or update `codebase.md`.

---

## Summary Statistics

| Severity | Count |
|----------|-------|
| 🔴 Critical | 5 |
| 🟠 High | 23 |
| 🟡 Medium | 13 |
| 🔵 Low | 12 |
| **Total** | **53** |

Of these, **~18 have already been fixed** during the audit (noted as "applied" above). The remaining ~35 need manual remediation.

---

## Priority Action Items

### Immediate (fix before next deploy)
1. Revoke ALL exposed secrets in `.env` files
2. Fix `docker-client` → `docker` in `flake.nix`
3. Add missing `sentence-transformers`/`tqdm` to `pyproject.toml`
4. Correct the content hash in `seed.py` to avoid duplicate question IDs
5. Add missing fields to UPSERT in `seed.py`

### Short-term (within this sprint)
6. Run the full pipeline for 2024, 2026, and missing 2025 data
7. Fix `search_by_text.py` model caching
8. Correct `scoring.md` documentation examples
9. Fix watermark removal to preserve diagram content
10. Remove default credentials from `clear_cloudinary.py`

### Medium-term
11. Add exponential backoff to image download retries
12. Normalize time divergence by accuracy in `rate.py`
13. Add embedding dimension validation
14. Standardize Python version requirements
15. Implement the overnight batch Glicko update pipeline
