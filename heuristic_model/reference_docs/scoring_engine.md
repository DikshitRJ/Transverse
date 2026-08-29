# AlphaJEE — Scoring Engine Architecture

## Overview

The scoring engine covers the full lifecycle of a learn session: picking the first question, updating the student's ability estimate after each answer, picking the next question, and updating the global rating when the session closes. It also governs answer matching for JEE's diverse question types.

Four independent systems work together:

| System | File | Scope |
|---|---|---|
| **IRT Theta Ladder** | `theta.go` | Per-question ability update (1PL Rasch) |
| **Question Selection** | `scoring.go` | Rule-based next-question picker with JEE type weighting |
| **Glicko-2 Session Update** | `glicko.go` | Session-level global rating update |
| **Answer Matching** | `learn_service.go` | JEE-style numerical range and multi-correct matching |

---

## 1. Adaptive Session Flow Architecture

This section describes the complete lifecycle of a single ADAPTIVE learn session, from creation to close.

### 1.1 High-Level Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                         START SESSION                               │
│                                                                     │
│  1. Resolve scope (chapters/groups/subjects → chapter slugs)        │
│  2. Check for existing active session → resume if found             │
│  3. Load chapter theta (default 1300)                               │
│  4. Load all questions in scope                                     │
│  5. Filter unseen questions (ADAPTIVE filters too, unlike before)   │
│  6. Cold start: PickBestQuestion(nil, ...) → first question         │
│  7. Create learn_sessions row (status=ACTIVE)                       │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         SUBMIT ANSWER                               │
│                                                                     │
│  For each answer in the session:                                    │
│                                                                     │
│  1. Validate session ownership + ACTIVE status                      │
│  2. Load question, get correct options                              │
│  3. Match answer (set equality, numerical range, OR-separator)      │
│  4. IRT theta update: θ_new = θ + K·(actual - P_correct)           │
│  5. Compute streaks from response history                           │
│  6. Build ScState (theta, subject, bias, streaks, phase)            │
│  7. Filter session-level seen set from candidates                   │
│  8. PickBestQuestion(candidates, state, currentQ, isCorrect)        │
│  9. Persist: question_stats + session response (in transaction)     │
│  10. Invalidate seenIDs cache                                       │
│                                                                     │
│  Returns: {isCorrect, correctOptions, thetaBefore, thetaAfter,      │
│            nextQuestion}                                             │
└──────────────────────────┬──────────────────────────────────────────┘
                           │ (repeat for each question)
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         CLOSE SESSION                               │
│                                                                     │
│  1. Compute session accuracy + average time                         │
│  2. Batch-load question chapters (avoid N+1)                        │
│  3. Per-chapter accuracy stats                                      │
│  4. Batch-load opponent Glicko ratings                              │
│  5. Glicko-2 update (session as single game)                        │
│  6. Mastery score = (θ - 1300) / (2800 - 1300) × 100              │
│  7. Update per-chapter stats in learning_stats.chapters             │
│  8. Recompute LearningDNA rolling aggregates                        │
│  9. Update user global Glicko (learn_rating, learn_rd, learn_vol)   │
│  10. Set session status = COMPLETED                                 │
│                                                                     │
│  All writes happen in a single transaction.                         │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Session State Machine

```
                    ┌─────────┐
         Start()    │         │
        ──────────► │ ACTIVE  │ ◄── Submit() updates theta + appends response
                    │         │
                    └────┬────┘
                         │
              ┌──────────┼──────────┐
              │          │          │
         Close()    Abandon()   Start() on
              │          │      existing session
              ▼          ▼          │
        ┌──────────┐ ┌──────────┐  │
        │COMPLETED │ │ABANDONED │  │ (resume returns existing ACTIVE)
        └──────────┘ └──────────┘  │
                                   │
                    ┌──────────────┘
                    ▼
              Returns existing session + current question
```

### 1.3 Data Flow Per Submission

```
Client sends: {question_id, selected_options, time_taken_ms}
                         │
                         ▼
              ┌─────────────────────┐
              │ Load question from  │──► cache hit? use cached
              │ DB (with embedding) │──► cache miss? query + cache 24h
              └─────────┬───────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Answer matching     │
              │ (see Section 8)     │
              └─────────┬───────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ IRT theta update    │
              │ θ_new = θ + K(a-P) │
              └─────────┬───────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Streak computation  │──► scan responses backwards
              └─────────┬───────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Build ScState       │──► load DNA (cached 60s)
              └─────────┬───────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Filter candidates   │──► exclude session-seen questions
              └─────────┬───────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ PickBestQuestion    │──► cold start / after correct / after wrong
              └─────────┬───────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ BEGIN TRANSACTION   │
              │ • upsert question_  │
              │   stats             │
              │ • append response   │
              │   + update theta    │
              │ COMMIT              │
              └─────────┬───────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Invalidate cache    │──► seen:{userID}
              └─────────────────────┘
```

---

## 2. IRT Theta Ladder (`theta.go`)

After every answer, the student's ability estimate (`θ`, theta) is updated using a 1PL (Rasch) IRT approximation.

### Formula

```
θ_irt = (θ − 1500) / 100
b_irt = (Glicko − 1500) / 100
P_correct = 1 / (1 + e^(-a·(θ_irt − b_irt)))
θ_new = θ + K · timeFactor · (actual − P_correct)
```

| Symbol | Value | Meaning |
|---|---|---|
| `a` | 1/27 | Discrimination — one JEE scale point ≈ 27 rating points |
| `b_irt` | (Glicko − 1500) / 100 | Question difficulty on shared IRT scale |
| `K` | 30 | Learning rate — max theta change per question |
| `θ` | 800–3500 | Student ability on JEE score scale |
| `timeFactor` | 0.3–2.0 | Ratio of expected/actual solve time scales the theta update |
| `actual` | 0 or 1 | 1 for correct, 0 for wrong |

Theta is maintained on the **JEE score scale** (~1300–2800), NOT the standard IRT N(0,1) scale. This keeps it human-readable and directly mappable to mastery scores and question Glicko ratings.

### Scale Mapping

The formula maps both θ and question difficulty to a shared IRT scale by centering on 1500 (midpoint of JEE range) and dividing by 100:

```
θ_irt = (θ - 1500) / 100     →  student on IRT scale
b     = (Glicko - 1500) / 100 →  question on IRT scale
```

This means:
- θ = 1500 (average student) → θ_irt = 0 (neutral)
- Question Glicko = 1500 (average) → b = 0 (neutral)
- θ = 2000, Question = 1500 → P(correct) = 1/(1+e^(-1/27·(5-0))) ≈ 0.55 (moderate student, moderate question)
- θ = 1500, Question = 2000 → P(correct) = 1/(1+e^(-1/27·(0-5))) ≈ 0.45 (slightly less than even)

The discrimination a=1/27 means each 27-point Glicko gap shifts P(correct) by ~1 percentage point at the steepest region of the logistic curve. This keeps theta movement gradual and stable across a session.

### Time-Aware Scaling

The update magnitude is scaled by how quickly the student answered relative to the question's expected solve time:

| Ratio (expected/actual) | timeFactor | Interpretation |
|---|---|---|
| 1.0 (answered at expected pace) | 1.0 | Neutral — typical signal strength |
| 2.0 (answered 2× faster) | 2.0 | Strong signal — clearly knew/didn't know |
| 0.5 (answered 2× slower) | 0.5 | Weak signal — struggled/guessed |
| Floor | 0.3 | Minimum signal even for very slow answers |
| Ceiling | 2.0 | Maximum signal for extremely fast answers |

This rewards fast correct answers with larger theta gains and penalises fast wrong answers with larger drops.

### Clamping

Theta is clamped to `[800, 3500]` — an empty guesser can never go below 800, and a perfect performer caps at 3500.

### Implementation

```go
func ComputeThetaUpdate(thetaBefore, questionGlicko float64, isCorrect bool, timeTakenMs, expectedTimeMs int64) float64
```

Called from `Submit()` in `learn_service.go` after every answer. Internally converts both theta and question Glicko to IRT scale before computing P(correct). The new theta is stored as `session.ThetaCurrent`.

---

## 3. Effective Theta (`ComputeEffectiveTheta`)

Before picking a question, the raw theta is adjusted by several modifiers to produce a **difficulty target** (`thetaEff`). The picker then finds a question whose Glicko rating is closest to this target.

### Modifiers (applied in order)

```
thetaEff = ThetaCurrent
         + SubjectBias × 200       // per-subject strength/weakness
         + Momentum                 // streak boost or penalty
         + Circadian                // peak hour bonus / off-hour penalty
         + SessionPhase             // warm-up / cool-down adjustment
```

**Subject Bias** — from `LearningDNA.SubjectBias`. A student who is +12% above average in physics gets `0.12 × 200 = +24` points when picking a physics question. Clamped to `[-0.5, 0.5]` (max ±100 theta points).

**Momentum** — computed from consecutive correct/wrong streaks:
```
rawMomentum = ConsecutiveCorrect × 15 − ConsecutiveWrong × 20
momentumF = clamp(rawMomentum, -60, 60)
```
A student on a 3-correct streak gets +45; a 2-wrong streak gets −40.

**Circadian** — if the current hour matches the student's `PeakPerformanceHour`, theta gets +25; otherwise −15.

**Session Phase** — based on `QuestionCount / AvgSessionLength`:
| Phase | Condition | Adjustment |
|---|---|---|
| Warm-up | pct < 20% | −30 |
| Normal | 20%–70% | 0 |
| Cool-down | ≥ 70% | −20 |

### Implementation

```go
func ComputeEffectiveTheta(s *ScState) (thetaEff float64, momentum int)
```

Called once per `PickBestQuestion` invocation.

---

## 4. Question Selection (`PickBestQuestion`)

The selection uses a **6-factor weighted scorer** with context-dependent weight sets. Three distinct strategies exist, chosen by context, and all use the same unified scoring function with different weights.

### JEE Question Type Weights

Different JEE question types have fundamentally different difficulty profiles. The picker applies a Glicko-point shift to the difficulty fit:

| Type | Bonus | Rationale |
|---|---|---|
| `MCQ` | 0 (baseline) | 4 options, -1/+4 marking → standard negative marking pressure |
| `MULTI_CORRECT` / `MSQ` | −50 | Partial marking, must identify ALL correct options → harder |
| `NUMERICAL` / `INTEGER` | +30 | No negative marking but requires computation, not elimination |

The bonus is applied as a shift to the difficulty fit score:
```
df = clamp(df + typeBonus / 1500, 0, 1)
```

### The 6-Factor Scorer

Every candidate question receives six scores, each normalised to [0, 1]:

| # | Factor | Weight | Description |
|---|---|---|---|
| 1 | **DifficultyFit** | 15–70% | `1 − \|thetaEff − Glicko\| / 1500` — how close the question's Glicko rating matches the student's effective theta |
| 2 | **VectorSimilarity** | 0–25% | Cosine similarity of embeddings — encourages topic clustering (same concept as previous question) |
| 3 | **TimeMatch** | 10% | `1 − \|avgTime − qTime\| / max(avgTime, qTime)` — how well expected solve time matches the user's pace. Defaults to 0.5 when unknown. |
| 4 | **NoveltyFactor** | 5–20% | `1 − min(attempts/5, 0.5)` — penalises questions the user has already seen many times |
| 5 | **ImmediateReinforce** | 0–35% | Same as VectorSimilarity — boosts similar questions after a wrong answer to reinforce the concept |
| 6 | **CarelessnessPenalty** | 0–10% | `carelessnessIndex × (1 − df)` — subtracted from total for careless users to penalise easy questions |

**Total formula:**
```
total = w_df × df + w_vs × vs + w_tm × tm + w_nf × nf + w_ir × ir − w_cp × cp
total = clamp(total, 0, 1)
```

Note: CarelessnessPenalty is **subtracted**, not added.

### Context-Dependent Weight Sets

#### 4a. Cold Start — first question of a session

When there is no previous question (`currentQuestion == nil`), VS and IR can't be computed, so weight shifts heavily to difficulty fit and novelty:

| Factor | Weight |
|---|---|
| DifficultyFit | 70% |
| TimeMatch | 10% |
| NoveltyFactor | 20% |

No pre-filter — all candidates in scope are scored.

**Seen filtering:** ADAPTIVE mode filters out questions seen in previous sessions for cold start too. If all questions have been seen, falls back to the full candidate pool.

#### 4b. After a Correct Answer

**Pre-filter:** Only consider questions with a strictly higher Glicko rating than the current question. If none exist, fall back to all candidates.

| Factor | Weight |
|---|---|
| DifficultyFit | 50% |
| VectorSimilarity | 15% |
| TimeMatch | 10% |
| NoveltyFactor | 10% |
| ImmediateReinforce | 5% |
| CarelessnessPenalty | 10% |

This pushes the student toward slightly harder questions while maintaining some topical flow.

#### 4c. After a Wrong Answer (or Skip)

**Pre-filter:** Only candidates in the same chapter as the current question. If none exist, fall back to all candidates.

| Factor | Weight |
|---|---|
| DifficultyFit | 15% |
| VectorSimilarity | 25% |
| TimeMatch | 10% |
| NoveltyFactor | 5% |
| ImmediateReinforce | 35% |
| CarelessnessPenalty | 10% |

This emphasises **similar questions** (same concept, via VS weighting) with a strong **reinforcement** bias (IR weight) to solidify understanding.

### Carelessness Tuning

When `carelessnessIndex > 0`, weights are dynamically adjusted:
```
w_difficulty += carelessnessIndex × 0.15
w_penalty    -= carelessnessIndex × 0.15
```
All six weights are then re-normalised to sum to 1.0.

### After a Skip

Skips are treated identically to wrong answers (`wasCorrect=false`) but streaks are reset to 0 before calling `PickBestQuestion`.

### Never Repeat Within a Session

Before `PickBestQuestion` is called, the caller (`Submit` in `learn_service.go`) builds a session-level seen-set from the session's response history plus the current question ID. Any question already shown in the session is excluded from the candidate pool.

Across sessions, ADAPTIVE mode allows re-showing questions (unlike REGULAR mode which permanently excludes seen questions).

### Implementation

```go
func PickBestQuestion(
    candidates []models.Question,
    state *ScState,
    currentQuestion *models.Question,  // nil for cold start
    wasCorrect bool,
) *PickResult

func ScoreCandidate(
    q models.Question,
    current *models.Question,
    thetaEff float64,
    state *ScState,
    weights WeightSet,
) ScoreComponents
```

Returns a `PickResult` containing the chosen question, its component scores (for logging), and the theta effective + momentum values used.

---

## 5. ScScores (Debug Logging)

Every pick logs all six component scores into the session response for debugging:

| Field | Range | Purpose |
|---|---|---|
| `DifficultyFit` | [0, 1] | `1 − \|thetaEff − Glicko\| / 1500 + typeBonus/1500` — how close the question's rating matches the student |
| `VectorSimilarity` | [0, 1] | Cosine similarity between current and candidate question embeddings |
| `TimeMatch` | [0, 1] | `1 − \|avgTime − qTime\| / max(avgTime, qTime)` — defaults to 0.5 when data unavailable |
| `NoveltyFactor` | [0.5, 1] | `1 − min(attempts/5, 0.5)` — 1.0 for new questions, 0.5 at 5+ attempts |
| `ImmediateReinforce` | [0, 1] | Same as VectorSimilarity — used with higher weight after wrong answers |
| `CarelessnessPenalty` | [0, 1] | `carelessnessIndex × (1 − df)` — subtracted from total, not added |
| `ThetaEffective` | 800–3500 | The final difficulty target after all modifiers (bias, momentum, circadian, phase) |
| `Momentum` | [-60, 60] | Raw momentum: correct×15 − wrong×20 |
| `Total` | [0, 1] | `Σ(w_i × f_i) − w_cp × cp` — the weighted total that determined the winner |

These are stored in the `SessionResponse` JSONB and exposed to the frontend for the debug panel.

---

## 6. Streak Computation

`computeStreaks` scans the session's response history backwards to count consecutive correct and wrong answers:

```go
func computeStreaks(responses []SessionResponse, latestCorrect bool) (consecutiveCorrect, consecutiveWrong int)
```

The latest answer is counted as base=1, then the scan adds matching consecutive answers from the history. Returns only one of (correct, wrong) as non-zero.

Used to populate `ScState.ConsecutiveCorrect` / `ConsecutiveWrong` for the momentum modifier.

---

## 7. Glicko-2 Session-Level Update (`glicko.go`)

When a session closes (`Close`), the entire session is treated as a single "game" against the average question difficulty. This updates the user's global `learn_rating` / `learn_rd` / `learn_vol` on the `users` table.

### Why separate from the IRT ladder?

- **IRT theta** is the live in-session tracker — it moves after every answer and drives question selection.
- **Glicko-2** is the persistent global rating — it only updates on session close and provides a calibrated uncertainty (RD) for matchmaking and mastery scoring.

### Glicko-2 Steps

1. Convert player rating/RD/volatility from Glicko scale to Glicko-2 scale (divide by 173.7178).
2. Treat the session as one opponent with:
   - Rating = average Glicko rating of all questions served
   - RD = 50/173.7178 (residual uncertainty for the session)
3. Compute the Glicko-2 `g(φ)` and `E(μ, μⱼ, φⱼ)` functions.
4. Compute variance `v` and rating change `Δ`.
5. Find the new volatility `σ'` using the Illinois algorithm (iterative solver).
6. Compute new pre-rating RD `φ* = sqrt(φ² + σ'²)`.
7. Compute new rating `μ'` and new RD `φ'`.
8. Convert back to Glicko scale.

### Constants

| Parameter | Value | Meaning |
|---|---|---|
| `τ` (tau) | 0.5 | Volatility change rate (moderately conservative) |
| Default RD | 350 | Starting uncertainty for a fresh user |
| Default Vol | 0.06 | Starting volatility |
| Min RD | 30 | Floor on rating deviation |

### Implementation

```go
func UpdateGlickoFromSession(in GlickoSessionInput) GlickoSessionOutput
```

Called from `Close()` in `learn_service.go`. The input includes the pre-session Glicko state (from `users` table, NOT session theta), average opponent rating, and session accuracy.

**Important:** The `PlayerRating` must be `user.LearnRating` (Glicko scale), NOT `session.ThetaStart` (IRT scale). These are different metrics.

---

## 8. Mastery Score

A simple linear mapping from theta to a 0–100 score:

```
mastery = round((theta − 1300) / (2800 − 1300) × 100, 1 decimal)
```

| Theta | Mastery |
|---|---|
| ≤ 1300 | 0.0 |
| 1500 | 13.3 |
| 2000 | 46.7 |
| 2500 | 80.0 |
| ≥ 2800 | 100.0 |

### Implementation

```go
func ComputeMasteryScore(theta float64) float64
```

Called in `Close()` and stored per-chapter in `learning_stats.chapters`.

---

## 9. Session Lifecycle — Detailed

### `Start`

1. **Resolve scope:** chapters/chapter_groups/subjects → deduplicated chapter slugs via `s.questionRepo.ResolveScopeByDB()` (delegates to DB, resolves groups/subjects by querying `questions` table).
2. **Resume check:** If single-chapter scope, check for existing ACTIVE session. If found, load its current question and return it (session resume).
3. **Load theta:** Fetch chapter stats from `learning_stats.chapters`. Use chapter's stored theta, or default 1300 for new chapters.
4. **Load candidates:** Fetch all questions in scope with embeddings from DB.
5. **Load seen IDs:** Fetch all previously attempted question IDs (cached 60s, invalidated on submit).
6. **First question selection:**
   - **REGULAR mode:** Filter unseen → random pick from unseen.
   - **ADAPTIVE mode:** Filter unseen → `PickBestQuestion(unseen, state, nil, false)` (cold start).
7. **Create session:** Insert `learn_sessions` row with status `ACTIVE`, theta_start, scope JSONB.

### `Submit`

1. **Validate:** Session exists, belongs to user, status is ACTIVE.
2. **Load question:** Fetch question by ID (cached 24h). Get correct options.
3. **Match answer:** Compare selected vs correct (see Section 10).
4. **IRT update:** `thetaAfter = ComputeThetaUpdate(thetaBefore, question.GlickoRating, isCorrect, timeTakenMs, question.ExpectedTimeMs)`.
5. **Streaks:** `computeStreaks(responses, isCorrect)`.
6. **Load DNA:** Fetch `LearningDNA` from user (cached 60s).
7. **Build ScState:** theta, subject, subject bias, peak hour, streaks, question count, avg session length.
8. **Filter candidates:** Build session-level seen set from response history + current question. Exclude from scope questions.
9. **Pick next:**
   - **REGULAR mode:** `pickRandomUnseen(chapter, seenIDs)`.
   - **ADAPTIVE mode:** `PickBestQuestion(available, scState, currentQ, isCorrect)`.
10. **Persist (transaction):**
    - Upsert `user_question_stats` (attempt count, correct count, time).
    - Append response to `learn_sessions.responses` JSONB + update theta_current + question_count + current_question_id.
11. **Invalidate cache:** `seen:{userID}`.
12. **Return:** isCorrect, correctOptions, thetaBefore, thetaAfter, nextQuestion (with signed Cloudinary images).

### `Close`

1. **Validate:** Session exists, belongs to user, status is ACTIVE.
2. **Handle empty session:** If no responses, abandon session.
3. **Compute stats:** accuracy = correct/total, avgTime = totalTime/total.
4. **Load user DNA:** For Glicko input and DNA recomputation.
5. **Batch-load chapters:** Single query for all question chapters (avoids N+1).
6. **Per-chapter accuracy:** Aggregate correct/total/time per chapter.
7. **Batch-load ratings:** Single query for all question Glicko ratings (avoids N+1).
8. **Glicko-2 update:** `UpdateGlickoFromSession()` with user.LearnRating (NOT theta).
9. **Mastery:** `ComputeMasteryScore(thetaFinal)`.
10. **Update chapter stats (transaction):**
    - For each chapter: update theta, mastery, glicko, attempts, time.
    - Only overwrite theta/mastery if single-chapter session OR first time practicing that chapter.
    - Accumulate total attempts, correct attempts, session count.
11. **Recompute DNA:** EMA of accuracy, time, session length, subject bias.
12. **Update user Glicko:** learn_rating, learn_rd, learn_vol.
13. **Close session:** Set status=COMPLETED, theta_current=final.
14. **Commit transaction.**

---

## 10. Answer Matching

JEE uses multiple question types with different answer formats. The `optionsMatch` function handles all of them:

### MCQ (Single Correct)

Simple set equality: student selects one option, compared against the single correct option.

```
Selected: ["A"]    Correct: ["A"]     → Match ✓
Selected: ["B"]    Correct: ["A"]     → No match ✗
```

### Multi-Correct (MSQ)

Set equality with multiple options: student must select ALL correct options (order-independent).

```
Selected: ["A", "C"]    Correct: ["A", "C"]     → Match ✓
Selected: ["A"]         Correct: ["A", "C"]     → No match ✗ (missing C)
Selected: ["A", "B", "C"] Correct: ["A", "C"]   → No match ✗ (extra B)
```

### Numerical / Integer

Exact match after whitespace trimming.

```
Selected: ["105.5"]     Correct: ["105.5"]      → Match ✓
Selected: [" 105.5 "]   Correct: ["105.5"]      → Match ✓ (trimmed)
```

### Numerical Range (JEE Advanced)

Range format: `"105.4TO105.6"` means any value in `[105.4, 105.6]`.

```
Selected: ["105.5"]     Correct: ["105.4TO105.6"]  → Match ✓ (105.5 in range)
Selected: ["105.3"]     Correct: ["105.4TO105.6"]  → No match ✗ (below range)
```

### OR-Separated Ranges

Multiple acceptable ranges separated by `"OR"`:

```
Selected: ["-29.9"]     Correct: ["-29.95TO-29.8OR29.8TO29.95"]  → Match ✓
Selected: ["29.9"]      Correct: ["-29.95TO-29.8OR29.8TO29.95"]  → Match ✓
Selected: ["0.0"]       Correct: ["-29.95TO-29.8OR29.8TO29.95"]  → No match ✗
```

### Implementation

```go
func optionsMatch(selected, correct []string) bool      // handles all types
func numericalMatch(selected, rangeStr string) bool      // checks OR-separated ranges
```

---

## 11. REGULAR vs ADAPTIVE Mode

| Aspect | REGULAR | ADAPTIVE |
|---|---|---|
| **First question** | Random unseen | Cold start (difficulty fit + type weight) |
| **Next question** | Random unseen | Rule-based (correct → harder, wrong → similar) |
| **Seen filtering (cold start)** | DB-level `filterUnseen` (permanent) | DB-level `filterUnseen` (permanent) |
| **Seen filtering (mid-session)** | DB-level `filterUnseen` | Session-level seen set (per-session) |
| **Cross-session repeats** | Never (once seen, excluded forever) | Allowed (questions can reappear in new sessions) |
| **Scoring metadata** | Not computed | All ScScores logged per response |

In REGULAR mode, once a question has been attempted in any session, it is permanently excluded from future REGULAR sessions via the `filterUnseen` helper.

In ADAPTIVE mode, cold start now also filters previously seen questions (falling back to all if every question has been seen). Mid-session, only the current session's response history is used as the exclusion set — new sessions start fresh.

---

## 12. ScState

The state object passed to `PickBestQuestion` and `ComputeEffectiveTheta`:

| Field | Source | Used By |
|---|---|---|
| `ThetaCurrent` | Session's `theta_current` after latest update | EffectiveTheta |
| `Subject` | Current question's subject | Difficulty fit context |
| `SubjectBias` | `LearningDNA.SubjectBias[subject]` | EffectiveTheta |
| `PeakPerformanceHour` | `LearningDNA.PeakPerformanceHour` | EffectiveTheta |
| `CurrentHour` | `time.Now().Hour()` | EffectiveTheta |
| `ConsecutiveCorrect` | Streak computation | EffectiveTheta (momentum) |
| `ConsecutiveWrong` | Streak computation | EffectiveTheta (momentum) |
| `QuestionCount` | `len(responses) + 1` | EffectiveTheta (session phase) |
| `AvgSessionLength` | `LearningDNA.AvgSessionLength` | EffectiveTheta (session phase) |

---

## 13. LearningDNA

Rolling aggregates stored as JSONB on the `users` row, recomputed on session close:

| Field | Purpose |
|---|---|
| `AvgAccuracy` | Correct / total (all-time, EMA-weighted) |
| `AvgTimeTakenMs` | Mean ms per question (EMA-weighted) |
| `AvgSolveVelocity` | Questions per hour across sessions |
| `AvgFatigueTolerance` | Questions before accuracy drops > 15% |
| `CarelessnessIndex` | Wrong rate on questions rated ≤ theta − 200 |
| `PeakPerformanceHour` | Hour with best accuracy (0–23) |
| `AvgSessionLength` | Mean questions per completed session |
| `TotalSessions` | Number of completed sessions |
| `TotalQuestionsSolved` | Total questions ever answered |
| `SubjectBias` | Per-subject accuracy delta vs overall average (EMA α=0.15) |

### SubjectBias Computation

On session close, per-subject accuracy is computed (minimum 3 questions per subject to avoid thrashing). The bias is updated via exponential moving average:

```
sessionSubjAcc = subjectCorrect / subjectTotal
delta = sessionSubjAcc - dna.AvgAccuracy
newBias = oldBias × (1 - 0.15) + delta × 0.15
bias = clamp(bias, -0.5, 0.5)
```

A bias of +0.12 in physics means the student is 12% above their overall average in physics. This translates to +24 theta effective points when picking physics questions.

---

## 14. Caching Strategy

The scoring engine uses a multi-layer cache to minimize DB hits during hot paths:

| Cache Key | TTL | Invalidated By | Purpose |
|---|---|---|---|
| `q:{id}` | 24h | Never (questions immutable) | Full question with embedding |
| `q_chapter:{id}` | 24h | Never | Question chapter slug |
| `q_subject:{id}` | 24h | Never | Question subject |
| `seen:{userID}` | 5 min | Every `Submit()` | All attempted question IDs |
| `dna:{userID}` | 60s | Every `Submit()` | LearningDNA struct |

Batch queries (`loadQuestionChaptersBatch`, `loadQuestionSubjectsBatch`) check cache first, then fetch uncached IDs in a single `WHERE id = ANY($1)` query.

---

## 15. Transaction Safety

All writes happen in database transactions:

**Submit transaction:**
```
BEGIN
  → upsert user_question_stats (attempt++, correct++, time++)
  → append response to learn_sessions.responses JSONB
  → update theta_current, question_count, current_question_id
COMMIT
```

**Close transaction:**
```
BEGIN
  → update users.learn_rating/rd/vol (Glicko-2 output)
  → upsert learning_stats.chapters (per-chapter mastery)
  → update users.learning_dna (recomputed DNA)
  → update learn_sessions status=COMPLETED
COMMIT
```

If any step fails, the entire transaction rolls back. The `defer tx.Rollback(ctx)` ensures cleanup on error paths.
