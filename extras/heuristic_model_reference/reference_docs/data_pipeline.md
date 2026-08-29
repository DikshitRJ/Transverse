# AlphaJEE — Data Pipeline & Rating System

**Version 2.0 | Adaptive Question Bank for JEE Mains**

---

## Table of Contents
1. [Repository Layout](#1-repository-layout)
2. [Unified Data Schema](#2-unified-data-schema)
3. [Rating Architecture](#3-rating-architecture)
4. [Cold-Start vs IRT-Fitted Ratings](#4-cold-start-vs-irt-fitted-ratings)
5. [Pipeline Scripts](#5-pipeline-scripts)
6. [Running the Full Pipeline](#6-running-the-full-pipeline)
7. [Adding New Exam Data](#7-adding-new-exam-data)

---

## 1. Repository Layout

```
rl_agent/
├── data/
│   ├── 2024/                   # per-year shift JSONs + questions.csv
│   │   ├── 2024_a_4_s1_final.json
│   │   ├── ...
│   │   └── questions.csv       # auto-generated subset
│   ├── 2025/                   # same structure
│   ├── all_questions.csv       # ★ MASTER question bank (unified)
│   ├── all_options.csv         # ★ MASTER normalised options table
│   ├── response.csv            # student responses (JEE Session 2)
│   └── answer_key.csv          # correct answers (for IRT calibration)
│
├── scripts/
│   └── B_json_to_csv.py        # ★ JSON → CSV pipeline (append-safe)
│
├── notebooks/
│   └── 2PL_IRT.ipynb           # ★ 2PL IRT calibration → updates all_questions.csv
│
├── build_questions_json.py     # ★ CSV → JSON for prototype/app
│
└── docs/
    ├── main.md                 ← this file
    ├── questions_readme.md     # how to add new shifts
    └── ratings_docs/
        └── midnight-question-ratings.md  # Midnight Crunch format
```

---

## 2. Unified Data Schema

### `all_questions.csv`

| Column | Type | Description |
|--------|------|-------------|
| `qid` | str | Unique NTA question ID |
| `year` | int | Exam year (2024, 2025…) |
| `month` | str | jan / feb / apr |
| `day` | int | Exam day |
| `session` | str | s1 / s2 |
| `shift_id` | str | `{year}_{m}_{day}_{session}` e.g. `2025_j_22_s1` |
| `subject` | str | physics / chemistry / maths |
| `type` | str | MCQ / NUMERICAL |
| `correct_option` | str | A/B/C/D for MCQ; integer string for NUMERICAL |
| `p_value_raw` | float | Fraction correct from PYQ source |
| `p_value_adj` | float | `p_raw − 0.08` (skip-correction) |
| `b_value` | float | IRT difficulty in logit units, range [−4, +4] |
| `a_value` | float | IRT discrimination, range [0.3, 3.5]; **1.0 for cold-start** |
| `question_elo` | int | `1500 + b × 150`, clipped [800, 2800] |
| `rd` | float | Glicko-2 rating deviation (350 cold-start, 200 after IRT) |
| `volatility` | float | Glicko-2 volatility (default 0.06) |
| `irt_fitted` | bool | `True` after 2PL notebook has run on this question |
| `median_time_sec` | float | Median solve time in seconds |
| `img_url` | str | CDN URL of question image (blank if text-only) |
| `options_json` | str | JSON array: `[{option, text, images}, …]` |
| `text` | str | Question text (LaTeX-safe) |
| `chapter` | str / null | Physics/Chem/Maths chapter tag |
| `concept` | str / null | Sub-concept tag |

### `all_options.csv`

| Column | Type | Description |
|--------|------|-------------|
| `qid` | str | Foreign key → `all_questions.csv` |
| `option_label` | str | A/B/C/D for MCQ; `ANSWER` for NUMERICAL |
| `text` | str | Option text (LaTeX) |
| `images_json` | str | JSON array of CDN image URLs for this option |
| `is_correct` | bool | `True` for the correct option(s) |

---

## 3. Rating Architecture

### The unified formula (both cold-start and IRT must use this)

```
b_value (logit units)  →  question_elo = 1500 + b × 150
```

| b_value | question_elo | Meaning |
|---------|-------------|---------|
| −4.0 | 900 | Trivially easy — 98% of students get it right |
| −2.0 | 1200 | Easy — bottom quartile gets it right |
| 0.0 | 1500 | Average — 50% of average students succeed |
| +2.0 | 1800 | Hard — only top quartile cracks it |
| +4.0 | 2100 | Extremely hard — 2% solve it |

### Student Elo scale

```
student_elo = 1500 + normalised_theta × 150
```

- **Same 1500 baseline → student_elo ≈ question_elo means ~50% expected P(correct)**
- 68% of students fall in [1350, 1650]
- JEE Adv qualifier territory ≈ 1800+

### Glicko-2 components

| Field | Cold-Start | After IRT | Meaning |
|-------|------------|-----------|---------|
| `question_elo` | from `p_value_adj` logit | from 2PL regression | Estimated difficulty |
| `rd` | 350 | 200 | Uncertainty (high = new/unknown) |
| `volatility` | 0.06 | 0.06 | Erratic behaviour risk |

**Why Glicko-2 for questions?** A question seen by 10,000 students should barely move after one new attempt. A brand-new question with only 5 attempts should move dramatically. RD handles this automatically.

---

## 4. Cold-Start vs IRT-Fitted Ratings

### Cold-start (any new question, before real response data)

```python
p_adj    = p_value_raw - 0.08           # skip-bias correction
b_value  = -log(p_adj / (1 - p_adj))   # logit transform
a_value  = 1.0                           # neutral discrimination
question_elo = 1500 + b * 150           # Elo seed
rd       = 350.0                         # maximum uncertainty
irt_fitted = False
```

### IRT-fitted (after running `notebooks/2PL_IRT.ipynb`)

```python
# From logistic regression over real student responses:
a_value  = clf.coef_[0][0]              # discrimination
b_value  = -clf.intercept_[0] / a      # difficulty
question_elo = 1500 + b * 150           # SAME FORMULA
rd       = 200.0                         # reduced uncertainty
irt_fitted = True
```

> ⚠️ **These use the EXACT same `question_elo` formula.** Cold-start and IRT-fitted questions live on the same Elo scale. There is no discontinuity when a question becomes IRT-fitted.

---

## 5. Pipeline Scripts

### `scripts/B_json_to_csv.py` — JSON to master CSV

```bash
# Normal run (adds only NEW shifts):
python scripts/B_json_to_csv.py

# Dry-run (no files written, just reports):
python scripts/B_json_to_csv.py --dry-run

# Force-reparse a specific year (e.g. you updated a JSON):
python scripts/B_json_to_csv.py --force-year 2026
```

**Append-safe behaviour:**
- Detects which `shift_id`s are already in `all_questions.csv`
- Only processes new shift files
- Never overwrites `irt_fitted=True` parameters (IRT notebook owns those)

### `notebooks/2PL_IRT.ipynb` — IRT Calibration

Run after you have real response data (`data/response.csv`).  
Updates `a_value`, `b_value`, `question_elo`, `rd`, `irt_fitted` in-place.

### `build_questions_json.py` — CSV to App JSON

```bash
python build_questions_json.py                         # all questions
python build_questions_json.py --year 2025             # 2025 only
python build_questions_json.py --subjects physics maths  # filter subjects
python build_questions_json.py --web-dir ../web        # custom output dir
```

---

## 6. Running the Full Pipeline

```bash
cd rl_agent/
source env/bin/activate

# Step 1: Add new shifts (or run fresh if all_questions.csv doesn't exist)
python scripts/B_json_to_csv.py

# Step 2: If you have response.csv, run IRT calibration in Jupyter:
jupyter notebook notebooks/2PL_IRT.ipynb

# Step 3: Build app JSONs for the prototype
python build_questions_json.py --web-dir ../web
```

---

## 7. Adding New Exam Data

See **[questions_readme.md](./questions_readme.md)** for the complete step-by-step guide.

**Quick summary:**
1. Get the shift JSON: `2026_a_5_s1_final.json`
2. Drop it in `data/2026/`
3. Run `python scripts/B_json_to_csv.py`
4. That's it — the new shift is appended; nothing existing is touched.
