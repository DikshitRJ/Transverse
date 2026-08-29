# Preprocessing Pipeline Architecture

The preprocessing pipeline transforms raw JEE question data (JSON) into a database-ready corpus enriched with psychometric ratings, semantic embeddings, watermark-free images, and Cloudinary-backed asset URLs.

---

## 1. Pipeline Overview

Four sequential phases execute in a directed acyclic graph (DAG). Each phase reads from a dedicated directory and writes to the next stage's input directory.

```mermaid
flowchart LR
    subgraph Input["Source Layer"]
        A["data/24_25_26_&_adv/"]
    end

    subgraph Phase1["Rating Engine"]
        B["rate.py"]
    end

    subgraph Phase2["Embedding Engine"]
        C["generate_embeddings.py"]
    end

    subgraph Phase3["Image Pipeline"]
        D["upload_images.py"]
        DW["remove_watermark.py"]
    end

    subgraph Phase4["Database Load"]
        E["seed.py"]
    end

    subgraph Output["Destination Layer"]
        F[("PostgreSQL<br/>(pgvector)")]
    end

    A -->|"raw JSON"| B
    B -->|"rated JSON<br/>data/rated/"| C
    C -->|"embedded JSON<br/>data/processed/"| D
    D -->|"cl:public_id<br/>in JSON"| E
    E --> F

    D -.->|"invokes"| DW

    style A fill:#2d2d5e,stroke:#6a6a9a,color:#fff
    style B fill:#1e3a5f,stroke:#3a6a9a,color:#fff
    style C fill:#1e4a3f,stroke:#3a7a5f,color:#fff
    style D fill:#5a2a3f,stroke:#9a4a6a,color:#fff
    style E fill:#3a2a5a,stroke:#6a4a9a,color:#fff
    style F fill:#2a2a2a,stroke:#6a6a6a,color:#fff
```

### 1.1 Execution Order

| Command | Phase | Entry Point |
|---|---|---|
| `just embed` | 1 — Rating | `preprocess/rate.py` |
| `just embed` | 2 — Embedding | `preprocess/generate_embeddings.py` |
| `just upload-images` | 3 — Image Upload | `preprocess/scripts/upload_images.py` |
| `just seed` | 4 — Database Seed | `preprocess/seed.py` |

The compound recipe `just embed-all` runs phases 1 and 2 sequentially.

---

## 2. Rating Engine (`rate.py`)

Computes item-difficulty metrics by synthesising a Glicko rating from correctness telemetry and response-time divergence.

### 2.1 Data Flow

```mermaid
flowchart TD
    subgraph Input["Source"]
        F1["*_final.json<br/>(raw corpus)"]
    end

    subgraph Pass1["Pass 1: Baseline Telemetry"]
        P1["Extract timespent<br/>per subject"]
        P2["Compute median<br/>T_baseline"]
    end

    subgraph Pass2["Pass 2: Per-Question Rating"]
        P3["Read percent_correct"]
        P4["Clamp & compute<br/>Rasch logit (b)"]
        P5["Read timespent"]
        P6["Compute ΔR_time<br/>(log-ratio divergence)"]
        P7["Synthesise R_final<br/>= 1500 + b·173.717 + ΔR_time"]
    end

    subgraph Output["Output"]
        O1["glicko_rating<br/>glicko_rd: 350<br/>glicko_volatility: 0.06"]
    end

    F1 --> P1
    P1 --> P2
    F1 --> P3
    F1 --> P5
    P2 -.->|"T_baseline"| P6
    P3 --> P4
    P4 --> P7
    P6 --> P7
    P7 --> O1
```

### 2.2 Glicko Rating Synthesis

**Rasch Logit Difficulty**

$$p = \text{clamp}\left(\frac{P}{100}, 0.001, 0.999\right) \quad \quad b = \ln\left(\frac{1-p}{p}\right)$$

**Time Divergence Modifier**

$$T_{\text{baseline}} = \text{median}(\{t_{\text{sec}} \mid \text{question} \in \text{Subject}\})$$

$$\Delta R_{\text{time}} = 150.0 \times \ln\left(\frac{t_c}{T_{\text{baseline}}}\right), \quad \text{clamped to } \pm 200$$

**Final Rating**

$$R_{\text{final}} = \text{round}\big(1500.0 + (b \times 173.717) + \Delta R_{\text{time\_clamped}}\big)$$

| Parameter | Value | Source |
|---|---|---|
| `glicko_rating` | `R_final` | Computed per question |
| `glicko_rd` | `350.0` | Fixed (initial uncertainty) |
| `glicko_volatility` | `0.06` | Fixed |

### 2.3 Atomic Disk Commit

All writes follow a three-stage transactional protocol:

1. **Stage**: Serialise to `<file>.json.tmp`
2. **Verify**: Re-parse the temp file to guarantee structural validity
3. **Swap**: `os.replace(tmp, target)` — kernel-level atomic rename

If the process is interrupted, only the temp file is lost; the original remains intact.

---

## 3. Embedding Engine (`generate_embeddings.py`)

Generates 384-dimensional semantic vector embeddings using `BAAI/bge-small-en-v1.5`.

### 3.1 Architecture

```mermaid
flowchart LR
    subgraph WorkerPool["ProcessPoolExecutor (8 workers)"]
        W1[Worker 1]
        W2[Worker 2]
        W3["..."]
        W8[Worker 8]
    end

    subgraph Runtime["Per-Worker (lazy-loaded)"]
        M["SentenceTransformer<br/>BAAI/bge-small-en-v1.5"]
    end

    subgraph Encoding["Batch Encoding"]
        T["text + visual_description"]
        V["384-dim vector"]
    end

    subgraph Output["Output"]
        O["embedding array<br/>injected into JSON"]
    end

    WorkerPool --> Runtime
    Runtime --> T
    T -->|"batch_size=32"| V
    V --> O
```

### 3.2 Thread Confinement

Environment variables restrict BLAS threading to prevent resource contention across workers:

```
OMP_NUM_THREADS=1   MKL_NUM_THREADS=1
OPENBLAS_NUM_THREADS=1   VECLIB_MAXIMUM_THREADS=1
NUMEXPR_NUM_THREADS=1
```

### 3.3 Text Compilation

Each question's embedding input is concatenated from two fields:

$$\text{input} = \text{strip}(\text{question.text}) + \text{" "} + \text{strip}(\text{question.visual\_description})$$

If both are empty, a fallback empty string is used to preserve array alignment.

### 3.4 Batch Inference

- **Model**: `BAAI/bge-small-en-v1.5` (384 dimensions)
- **Batch size**: 32
- **Hardware**: CPU (GPU fallback not required for this model size)
- **Workers**: 8 (configurable via `max_workers`)

---

## 4. Image Pipeline (`upload_images.py`)

Downloads CDN-hosted images, removes ExamGOAL watermarks, uploads to Cloudinary as authenticated assets, and substitutes CDN URLs with Cloudinary public IDs.

### 4.1 Pipeline Per Image

```mermaid
flowchart TD
    subgraph Assets["Per Image (12 threads)"]
        A["CDN URL from JSON"]
        B["Download via HTTP"]
        C["Remove watermark<br/>remove_watermark.py"]
        D["Upload to Cloudinary<br/>type=authenticated"]
        E["Replace URL in JSON<br/>cl:public_id"]
    end

    subgraph Cloudinary["Cloudinary"]
        F["Authenticated asset<br/>signed URL required"]
    end

    A --> B
    B --> C
    C --> D
    D --> E
    E -.->|"serve via"| F

    style C fill:#5a2a3f,stroke:#9a4a6a,color:#fff
    style F fill:#2a4a3a,stroke:#4a8a6a,color:#fff
```

### 4.2 Watermark Removal (`remove_watermark.py`)

Uses luminance-based alpha matting:

1. Flatten RGBA/transparent images onto a white background
2. Crop browser chrome (dark headers/footers) via row-mean threshold
3. Compute grayscale luminance: $Y = 0.299R + 0.587G + 0.114B$
4. Generate alpha mask between black point (140) and white point (200)
5. Blend: $\text{out} = \alpha \cdot \text{white} + (1 - \alpha) \cdot \text{target\_color}$
6. Apply unsharp mask for clarity (radius=2, percent=150)

### 4.3 Cloudinary Public ID Convention

```
{year}/{shift_id}/{subject}/Q_{qid}_{index}
{year}/{shift_id}/{subject}/Q_{qid}_opt_{option}_{index}
```

After upload, the original CDN URL is replaced with:

```
cl:{public_id}
```

The `cl:` prefix signals downstream that this requires on-the-fly URL signing.

### 4.4 Idempotency

The pipeline skips URLs that already contain `cloudinary.com` or start with `cl:`, making re-runs safe. To force re-upload, revert the JSON files and re-run.

### 4.5 Dry Run Mode

```bash
uv run preprocess/scripts/upload_images.py --dry-run
```

Skips Cloudinary upload; downloads, watermarks, and validates the pipeline locally.

---

## 5. Database Seeding (`seed.py`)

Scans `data/processed/`, parses metadata from filenames (exam source, shift date), and upserts into PostgreSQL via pgvector.

### 5.1 Schema Target

```mermaid
erDiagram
    questions {
        text id PK
        varchar type
        text question_text
        jsonb images
        jsonb options
        varchar correct
        varchar subject
        varchar source
        varchar shift_date
        varchar chapter
        varchar chapter_group
        varchar difficulty
        boolean is_bonus
        real glicko_rating
        real glicko_rd
        real glicko_volatility
        int attempt_count
        int percent_correct
        int timespent_avg_ms
        vector embedding "384-dim"
    }
```

### 5.2 Filename Parsing

Extracts `(source, shift_date)` tuples from file stems:

| Pattern | Source | shift_date |
|---|---|---|
| `2026_j_24_s1_final.json` | JEE Main 2026 | `2026_2401S1` |
| `2025_a_4_s2_final.json` | JEE Main 2025 | `2025_0404S2` |
| `2024_adv_p1_final.json` | JEE Advanced 2024 | `2024_P1` |

### 5.3 Upsert Strategy

Uses `ON CONFLICT (id) DO UPDATE` with two query variants:
- **With embedding**: includes the `vector` column
- **Without embedding**: omits the vector column (for partial pipelines)

### 5.4 Vector Index

After seeding, an HNSW index on `embedding` accelerates cosine-similarity searches:

```sql
CREATE INDEX ON questions USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
```

---

## 6. Directory State Machine

```mermaid
stateDiagram-v2
    [*] --> Raw: data/24_25_26_&_adv/
    Raw --> Rated: rate.py
    state Raw {
        [*] --> hasImages
        hasImages --> rated
    }
    Rated --> Processed: generate_embeddings.py
    Rated --> UploadedImages: upload_images.py
    Processed --> Seeded: seed.py
    UploadedImages --> Seeded
    Seeded --> [*]

    note right of Rated
        data/rated/
        adds glicko_rating,
        glicko_rd, glicko_volatility
    end note

    note right of Processed
        data/processed/
        adds embedding[]
    end note

    note right of UploadedImages
        Same data/processed/
        but CDN URLs replaced
        with cl:public_id
    end note
```

---

## 7. Configuration Constants

| Name | Value | Location |
|---|---|---|
| `ELO_CENTER` | 1500.0 | `rate.py` |
| `ELO_SCALE` | 173.717 | `rate.py` |
| `TIME_SCALE` | 150.0 | `rate.py` |
| `TIME_CLAMP` | 200.0 | `rate.py` |
| `DEFAULT_BASELINE` | 60.0s | `rate.py` |
| `EMBED_DIM` | 384 | `generate_embeddings.py` |
| `BATCH_SIZE` | 32 | `generate_embeddings.py` |
| `WORKERS` | 8 (embed), 12 (upload) | respective scripts |
| `RETRY_LIMIT` | 3 | `upload_images.py` |
