# DKVMN Implementation Plan — Dynamic Key-Value Memory Network for Knowledge Tracing

> **Status:** Future Architecture — Not Yet Implemented
> **Prerequisite:** Current IRT + Glicko scoring engine is operational and producing training data.

---

## 1. What is DKVMN?

DKVMN (Dynamic Key-Value Memory Network) is a deep knowledge tracing model that uses neural memory networks to track a student's evolving knowledge state. Unlike classical IRT (Item Response Theory) which assumes a static ability parameter, DKVMN models knowledge as a **dynamic memory** that grows and decays with each student interaction.

### 1.1 Core Architecture

```
Input (question + answer) → Embedding → Attention → Memory Read → Memory Write → Prediction
```

DKVMN maintains two memory matrices:

| Memory | Shape | Purpose |
|--------|-------|---------|
| **Key Memory (Mk)** | `[N_concepts × d_model]` | Stores knowledge concept representations (learned). Each row is a latent "knowledge component". |
| **Value Memory (Mv)** | `[N_concepts × d_model]` | Stores the student's current mastery of each concept. Updated on every interaction. |

**Key insight:** The key memory is **static** (learned during training, shared across all students). The value memory is **dynamic** (updated per-student per-interaction). This is analogous to how the brain stores concept definitions (static) vs. recall strength (dynamic).

### 1.2 How It Works

1. **Question Embedding:** Each question is embedded into a vector via its question ID embedding (similar to word2vec).
2. **Read Step (Attention):**
   - Compute correlation between the question embedding and each key in Mk.
   - This produces an attention weight vector over concepts.
   - Read from Mv using these weights → produces a "knowledge state summary".
3. **Write Step (Update):**
   - Based on whether the answer was correct, update Mv.
   - Correct answers increase mastery; wrong answers decrease it.
   - The write is gated: the update magnitude depends on the correlation strength.
4. **Prediction:** The read state + question embedding → sigmoid → P(correct).

### 1.3 Key Advantage Over IRT/Glicko

| Aspect | IRT (Current) | DKVMN (Future) |
|--------|---------------|-----------------|
| Knowledge model | Single scalar θ | Vector of per-concept mastery |
| Question difficulty | Single scalar b | Embedded in key space |
| Interaction modeling | Independent per-question | Sequential, captures learning curves |
| Forgetting | Not modeled | Built-in via write decay |
| Cross-concept transfer | Manual via prerequisites | Learned from data |

---

## 2. Why AlphaJEE Needs DKVMN

### 2.1 Limitations of Current System

The current IRT theta + Glicko system is effective but has structural limitations:

1. **Scalar ability is lossy:** A student with θ=1800 in Physics could be strong in mechanics but weak in optics. The single number collapses this.
2. **No learning trajectory:** IRT treats each question independently. It can't model that answering Q1 wrong → practicing → answering Q2 right represents learning.
3. **Prerequisite graph is static:** The prerequisite edges in `syllabus.json` are hand-authored. DKVMN can discover latent prerequisites from data.
4. **No forgetting model:** Current system doesn't model that knowledge decays over time without practice.

### 2.2 What DKVMN Adds

1. **Per-concept mastery vector:** Instead of one θ per chapter, a learned vector of 64–128 latent concept mastery values.
2. **Sequential modeling:** The model sees the entire session history, not just the last answer.
3. **Data-driven prerequisites:** The key memory implicitly captures which concepts are related.
4. **Predictive power:** Can predict which concepts a student will struggle with before they even attempt questions.

---

## 3. Data Requirements

### 3.1 What We Already Have

The current AlphaJEE infrastructure already collects everything needed:

| Data | Table/Source | Status |
|------|-------------|--------|
| Question embeddings | `questions.embedding` (384-dim) | ✅ Available |
| Per-question metadata | `questions.*` | ✅ Available |
| Student attempt history | `user_question_stats` | ✅ Available |
| Session response sequences | `learn_sessions.responses` JSONB | ✅ Available |
| Question-chapter mapping | `questions.chapter` | ✅ Available |
| Glicko ratings per question | `questions.glicko_rating` | ✅ Available |

### 3.2 What We Need to Collect

| Data | Description | Priority |
|------|-------------|----------|
| **Answer sequence per session** | Ordered list of (question_id, is_correct, time_taken_ms) per session | Already in `learn_sessions.responses` |
| **Question-concept mapping** | Maps each question to one or more latent concepts | Derived from chapter + embedding clustering |
| **Inter-session time gaps** | Time between sessions for forgetting model | Derivable from `learn_sessions.created_at` |

### 3.3 Training Data Format

```python
# Each training example is a session interaction sequence
{
    "session_id": "sess_abc123",
    "user_id": "user_456",
    "interactions": [
        {
            "question_id": "q_001",
            "question_type": "MCQ",
            "chapter": "electrostatics",
            "subject": "physics",
            "is_correct": True,
            "time_taken_ms": 45000,
            "glicko_rating": 1850.0,
            "timestamp": "2026-05-29T10:30:00Z"
        },
        {
            "question_id": "q_042",
            "question_type": "NUMERICAL",
            "chapter": "electrostatics",
            "subject": "physics",
            "is_correct": False,
            "time_taken_ms": 120000,
            "glicko_rating": 2100.0,
            "timestamp": "2026-05-29T10:32:15Z"
        }
        // ... more interactions
    ]
}
```

---

## 4. Integration Architecture

### 4.1 Hybrid Approach (Recommended)

The most practical path is a **hybrid system** where DKVMN augments (not replaces) the current IRT engine:

```
Current Flow:
  User Answer → IRT Theta Update → Glicko Update → Question Picker

Hybrid Flow:
  User Answer → IRT Theta Update → DKVMN Update → Fused Prediction → Question Picker
```

### 4.2 Integration Points in Current Code

The current architecture has clear extension points for DKVMN:

| Current Component | DKVMN Integration | File |
|-------------------|-------------------|------|
| `ComputeThetaUpdate()` | Add DKVMN prediction as alternative ability estimate | `theta.go` |
| `ComputeEffectiveTheta()` | Blend IRT θ with DKVMN concept vectors | `scoring.go` |
| `PickBestQuestion()` | Use DKVMN predicted success probability for selection | `scoring.go` |
| `SessionResponse` | Store DKVMN read/write state for replay | `db_models.go` |
| `ChapterStats` | Add per-concept mastery alongside θ | `db_models.go` |
| `LearningDNA` | Add DKVMN concept mastery vector | `db_models.go` |

### 4.3 Proposed Schema Additions

```sql
-- Per-user DKVMN state (stored as JSONB for flexibility)
ALTER TABLE users ADD COLUMN dkvmn_state JSONB;

-- DKVMN concept definitions (key memory, learned during training)
CREATE TABLE dkvmn_concepts (
    id          SERIAL PRIMARY KEY,
    name        TEXT,           -- human-readable (auto-generated or from chapter)
    subject     TEXT,
    chapter     TEXT,
    embedding   VECTOR(128),    -- concept key embedding
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Per-user per-concept mastery (value memory snapshot)
CREATE TABLE dkvmn_user_concepts (
    user_id     TEXT REFERENCES users(id),
    concept_id  INT REFERENCES dkvmn_concepts(id),
    mastery     FLOAT,          -- current mastery level [0, 1]
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, concept_id)
);
```

### 4.4 DKVMN State Structure (JSONB on users.dkvmn_state)

```json
{
    "version": 1,
    "num_concepts": 64,
    "value_memory": [0.23, 0.87, 0.45, ...],
    "last_updated_session": "sess_abc123",
    "total_interactions": 1247,
    "concept_names": {
        "0": "coulombs-law",
        "1": "gauss-law",
        "2": "electric-potential",
        "3": "capacitance",
        "...": "..."
    }
}
```

---

## 5. Implementation Roadmap

### Phase 1: Data Collection & Export (Week 1–2)

**Goal:** Build training data pipeline from existing session data.

1. Create a data export job that serializes `learn_sessions.responses` into training format.
2. Cluster questions by embedding to define initial concept vocabulary (K-means on 384-dim embeddings → 64–128 clusters).
3. Map existing chapters to latent concepts via cluster membership.
4. Export to Parquet/CSV for training.

**Files to create:**
```
preprocess/export_dkvmn_training.py    # Export session data for training
preprocess/cluster_concepts.py         # K-means clustering on question embeddings
```

### Phase 2: Model Training (Week 3–4)

**Goal:** Train DKVMN on exported data.

1. Implement DKVMN in PyTorch (see Section 6 for architecture).
2. Train on exported AlphaJEE data.
3. Evaluate: AUC, RMSE, NLL on held-out sessions.
4. Export trained model weights.

**Files to create:**
```
ml/dkvmn/model.py              # DKVMN architecture
ml/dkvmn/train.py              # Training loop
ml/dkvmn/evaluate.py           # Evaluation metrics
ml/dkvmn/export.py             # Export to ONNX for Go inference
```

### Phase 3: Go Inference Integration (Week 5–6)

**Goal:** Run DKVMN predictions in the Go server.

1. Export trained model to ONNX format.
2. Use `onnxruntime` Go bindings for inference.
3. Integrate into `ComputeEffectiveTheta()` as a blending signal.
4. Store DKVMN state in `users.dkvmn_state`.

**Files to modify:**
```
server/internal/services/dkvmn.go       # New: DKVMN inference service
server/internal/services/scoring.go     # Modify: blend DKVMN with IRT
server/internal/services/learn_service.go # Modify: update DKVMN state on submit
```

### Phase 4: Adaptive Question Picker Upgrade (Week 7–8)

**Goal:** Replace rule-based picker with DKVMN-informed picker.

1. Use DKVMN predicted P(correct) for each candidate question.
2. Select questions where predicted P(correct) is in the "zone of proximal development" (0.3–0.7).
3. Blend with current DifficultyFit score for stability.

**Files to modify:**
```
server/internal/services/scoring.go    # Modify: DKVMN-informed selection
```

---

## 6. DKVMN Model Architecture (PyTorch Reference)

```python
import torch
import torch.nn as nn

class DKVMN(nn.Module):
    """
    Dynamic Key-Value Memory Network for Knowledge Tracing.
    
    Based on: "Dynamic Key-Value Memory Networks for Knowledge Tracing"
    (Zhang et al., WWW 2017)
    """
    
    def __init__(self, num_questions, num_concepts, embed_dim=64, hidden_dim=128):
        super().__init__()
        self.num_concepts = num_concepts
        self.embed_dim = embed_dim
        
        # Question embedding (maps question_id → dense vector)
        self.question_embed = nn.Embedding(num_questions, embed_dim)
        
        # Key memory (static, learned during training)
        self.key_memory = nn.Parameter(torch.randn(num_concepts, embed_dim) * 0.1)
        
        # Value memory (dynamic, initialized per-student)
        # Stored externally, passed as input during training
        self.value_dim = embed_dim
        self.value_memory_init = nn.Parameter(torch.randn(num_concepts, embed_dim) * 0.1)
        
        # Read/write controllers
        self.read_controller = nn.Sequential(
            nn.Linear(embed_dim * 2, hidden_dim),
            nn.ReLU(),
            nn.Linear(hidden_dim, num_concepts),
            nn.Softmax(dim=-1)
        )
        
        self.write_controller = nn.Sequential(
            nn.Linear(embed_dim * 2, hidden_dim),
            nn.ReLU(),
            nn.Linear(hidden_dim, num_concepts),
            nn.Sigmoid()
        )
        
        # Prediction head
        self.predict = nn.Sequential(
            nn.Linear(embed_dim * 2, hidden_dim),
            nn.ReLU(),
            nn.Linear(hidden_dim, 1),
            nn.Sigmoid()
        )
    
    def attention(self, question_embed, memory_keys):
        """Compute correlation between question and each concept key."""
        # question_embed: [batch, embed_dim]
        # memory_keys: [num_concepts, embed_dim]
        # Returns: [batch, num_concepts] attention weights
        sim = torch.mm(question_embed, memory_keys.T)  # [batch, num_concepts]
        sim = sim / (self.embed_dim ** 0.5)  # scaled dot-product
        return torch.softmax(sim, dim=-1)
    
    def read(self, question_embed, value_memory):
        """Read from value memory using question-key attention."""
        attn = self.attention(question_embed, self.key_memory)  # [batch, num_concepts]
        read_state = torch.mm(attn, value_memory)  # [batch, value_dim]
        return read_state, attn
    
    def write(self, question_embed, value_memory, is_correct):
        """Write to value memory based on answer correctness."""
        # Compute write gate
        combined = torch.cat([question_embed, value_memory.mean(dim=0, keepdim=True).expand_as(question_embed)], dim=-1)
        write_gate = self.write_controller(combined)  # [batch, num_concepts]
        
        # Correlation strength determines write magnitude
        attn = self.attention(question_embed, self.key_memory)  # [batch, num_concepts]
        
        # Update value memory
        # Correct → increase mastery; Wrong → decrease mastery
        update = write_gate * attn  # [batch, num_concepts]
        sign = torch.where(is_correct.unsqueeze(1).float() > 0, 1.0, -1.0)
        value_memory = value_memory + sign * update.unsqueeze(2) * self.key_memory.unsqueeze(0)
        value_memory = torch.clamp(value_memory, 0, 1)  # keep mastery in [0, 1]
        
        return value_memory
    
    def forward(self, question_ids, value_memory, is_correct=None):
        """
        Args:
            question_ids: [batch] question IDs
            value_memory: [batch, num_concepts, value_dim] current student state
            is_correct: [batch] whether answer was correct (None during inference)
        
        Returns:
            pred: [batch, 1] predicted P(correct)
            value_memory: [batch, num_concepts, value_dim] updated state
        """
        q_embed = self.question_embed(question_ids)  # [batch, embed_dim]
        
        # Read
        read_state, attn = self.read(q_embed, value_memory)  # [batch, value_dim]
        
        # Predict
        pred_input = torch.cat([q_embed, read_state], dim=-1)
        pred = self.predict(pred_input)  # [batch, 1]
        
        # Write (only during training)
        if is_correct is not None:
            value_memory = self.write(q_embed, value_memory, is_correct)
        
        return pred, value_memory
```

### 6.1 Training Loop

```python
def train_dkvmn(model, train_loader, val_loader, epochs=50, lr=1e-3):
    optimizer = torch.optim.Adam(model.parameters(), lr=lr)
    criterion = nn.BCELoss()
    
    for epoch in range(epochs):
        model.train()
        total_loss = 0
        
        for batch in train_loader:
            question_ids = batch['question_ids']      # [batch, seq_len]
            is_correct = batch['is_correct']          # [batch, seq_len]
            batch_size, seq_len = question_ids.shape
            
            # Initialize value memory (all zeros for new student)
            value_memory = torch.zeros(batch_size, model.num_concepts, model.value_dim)
            
            session_loss = 0
            for t in range(seq_len):
                q = question_ids[:, t]
                c = is_correct[:, t].float()
                
                pred, value_memory = model(q, value_memory, c)
                loss = criterion(pred.squeeze(), c)
                session_loss += loss
            
            session_loss /= seq_len
            optimizer.zero_grad()
            session_loss.backward()
            torch.nn.utils.clip_grad_norm_(model.parameters(), 1.0)
            optimizer.step()
            total_loss += session_loss.item()
        
        # Validation
        val_auc = evaluate(model, val_loader)
        print(f"Epoch {epoch}: loss={total_loss/len(train_loader):.4f}, val_auc={val_auc:.4f}")
```

---

## 7. Blending IRT and DKVMN

### 7.1 Prediction Fusion

```go
// In scoring.go — blended difficulty targeting
func ComputeEffectiveThetaBlended(
    irtTheta float64,
    dkvmnP float64,     // DKVMN predicted P(correct) for candidate
    questionGlicko float64,
    irtWeight float64,   // e.g., 0.6
    dkvmnWeight float64, // e.g., 0.4
) float64 {
    // IRT-based difficulty target
    irtTarget := irtTheta
    
    // DKVMN-based difficulty target: find difficulty where DKVMN predicts P=0.5
    // (the "zone of proximal development" midpoint)
    dkvmnTarget := questionGlicko
    
    // Blend
    return irtWeight*irtTarget + dkvmnWeight*dkvmnTarget
}
```

### 7.2 Question Selection with DKVMN

```go
// Pick question where DKVMN predicts P(correct) in [0.3, 0.7]
func pickWithDKVMN(candidates []Question, dkvmn PredictFunc) *Question {
    var best *Question
    var bestScore float64 = -1
    
    for _, q := range candidates {
        pCorrect := dkvmn(q.ID) // DKVMN prediction
        
        // Zone of Proximal Development: prefer P(correct) ≈ 0.5
        zpd := 1.0 - 2.0*math.Abs(pCorrect-0.5) // peaks at 0.5, 0 at extremes
        
        if zpd > bestScore {
            bestScore = zpd
            best = &q
        }
    }
    return best
}
```

---

## 8. Performance Considerations

### 8.1 Inference Latency

| Operation | Time | Notes |
|-----------|------|-------|
| Question embedding lookup | ~0.01ms | O(1) from in-memory table |
| DKVMN attention (64 concepts) | ~0.1ms | Single matrix multiply |
| DKVMN read + predict | ~0.05ms | Two small linear layers |
| **Total per-question** | **~0.16ms** | Well within 100ms SLO |

### 8.2 Memory per User

| Component | Size | Notes |
|-----------|------|-------|
| Value memory (64 concepts × 64 dim × float32) | 16 KB | Stored as JSONB |
| Key memory (shared) | 16 KB | In-memory, shared across all users |
| Total per-user state | **16 KB** | vs. current DNA ~1 KB |

### 8.3 Batch Inference

For the question picker (evaluating 20–50 candidates):
- Batch all candidates through DKVMN in a single forward pass.
- ~0.5ms total for 50 candidates on CPU.
- Can be further optimized with ONNX Runtime.

---

## 9. Migration Strategy

### Phase 1: Shadow Mode (No User Impact)
- Run DKVMN predictions alongside current system.
- Log predictions vs actual outcomes.
- Compare AUC with current IRT-based prediction.
- **Duration:** 2–4 weeks of data collection.

### Phase 2: A/B Testing
- Route 10% of sessions through DKVMN-informed picker.
- Measure: learning gain per session, session completion rate, user satisfaction.
- **Duration:** 2 weeks.

### Phase 3: Gradual Rollout
- Increase DKVMN weight from 0.1 → 0.3 → 0.5 → 0.7.
- Keep IRT as fallback for cold-start users (no history yet).
- **Duration:** 4 weeks.

### Phase 4: Full Integration
- DKVMN becomes primary ability estimator.
- IRT θ maintained as backup and for backward compatibility.
- Glicko-2 still used for session-level calibration.

---

## 10. Current System → DKVMN Readiness

The AlphaJEE scoring engine is already well-positioned for DKVMN integration:

| Current Feature | DKVMN Readiness | Gap |
|-----------------|-----------------|-----|
| Question embeddings (384-dim) | ✅ Can be used for concept clustering | Reduce to 64–128 dim |
| Session response sequences | ✅ Contains full interaction history | Already in JSONB |
| Per-question Glicko rating | ✅ Provides difficulty prior | No gap |
| LearningDNA concept vectors | ✅ Similar structure to DKVMN value memory | Expand dimensionality |
| ComputeEffectiveTheta blending | ✅ Clean interface for adding signals | Add DKVMN input |
| Batch query patterns | ✅ Already optimized for multi-question loads | No gap |
| Per-user state storage | ✅ JSONB on users table | Add dkvmn_state column |

The transition from rule-based → ML-enhanced scoring can be incremental, using the existing `ScState` and `ScScores` structures as the integration surface.
