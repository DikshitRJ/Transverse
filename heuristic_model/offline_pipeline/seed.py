#!/usr/bin/env python3
"""
preprocess/seed.py
──────────────────
Seeds processed question JSON files into PostgreSQL.

Expects:
  - DATABASE_URL in environment (set automatically by nix develop)
  - schemas/questions_init.sql  relative to project root
  - engine/data/processed/**/*_final.json  with embedded questions
"""

import hashlib
import json
import os
import re
import sys
from pathlib import Path
from dotenv import load_dotenv

import psycopg

ROOT        = Path(__file__).resolve().parent.parent.parent
load_dotenv(ROOT / "engine" / ".env")
SCHEMA_FILE = ROOT / "backend" / "sql-schemas" / "questions_init.sql"
DATA_DIR    = ROOT / "engine" / "data" / "processed"


# ── Filename → (source, shift_date) ──────────────────────────────────────────
#
# shift_date format (new convention):
#   Main:   {YYYY}_{DD}{MM}S{shift}       e.g. "2026_2401S1"
#   Adv:    {YYYY}_P{paper}                e.g. "2024_P1"
#   Legacy: falls back to stem as-is

_MONTH_LETTER_NUM = {
    "j":"01","f":"02","m":"03","a":"04",
    "s":"09","o":"10","n":"11","d":"12",
}

_MONTH_NAME_NUM = {
    "jan":"01","feb":"02","mar":"03","apr":"04",
    "may":"05","jun":"06","jul":"07","aug":"08",
    "sep":"09","oct":"10","nov":"11","dec":"12",
}

def parse_filename(stem: str) -> tuple[str, str | None, str]:
    """Returns (source, shift_date, exam_type)."""
    s = stem.lower().replace("-", "_")
    year_match = re.search(r"(20\d{2})", s)
    year = year_match.group(1) if year_match else None

    # ── JEE Advanced ──────────────────────────────────────────────────────
    if "adv" in s:
        source = f"JEE Advanced {year}" if year else "JEE Advanced"
        paper_match = re.search(r"(?:paper|p)_?([12])", s)
        paper = paper_match.group(1) if paper_match else None
        shift_date = f"{year}_P{paper}" if year and paper else None
        return source, shift_date, "JEE_ADV"

    source = f"JEE Main {year}" if year else "JEE Main"

    # ── New convention: {year}_{month_letter}_{day}_s{shift} ─────────────
    #    e.g. 2026_j_24_s1  →  2026_2401S1
    m = re.search(
        r"(\d{4})_([" + "".join(_MONTH_LETTER_NUM) + r"])_(\d{1,2})_s([12])", s
    )
    if m:
        yr    = m.group(1)
        mnum  = _MONTH_LETTER_NUM[m.group(2)]
        day   = f"{int(m.group(3)):02d}"
        shift = m.group(4)
        return source, f"{yr}_{day}{mnum}S{shift}", "JEE_MAIN"

    # ── Full month name format: {day}_{month}_{shift} ────────────────────
    #    e.g. 24_jan_shift_1  →  2026_2401S1
    m2 = re.search(
        r"(\d{1,2})_?(" + "|".join(_MONTH_NAME_NUM) + r")_?(?:shift|s)_?([12])", s
    )
    if m2:
        day   = f"{int(m2.group(1)):02d}"
        mnum  = _MONTH_NAME_NUM[m2.group(2)]
        shift = m2.group(3)
        return source, (f"{year}_{day}{mnum}S{shift}" if year else f"{day}{mnum}S{shift}"), "JEE_MAIN"

    # ── Single-letter month format (backward compat) ─────────────────────
    #    e.g. 24_j_s_1  →  2026_2401S1
    m3 = re.search(
        r"(\d{1,2})_?([" + "".join(_MONTH_LETTER_NUM) + r"])_?(?:shift|s)_?([12])", s
    )
    if m3:
        day   = f"{int(m3.group(1)):02d}"
        mnum  = _MONTH_LETTER_NUM[m3.group(2)]
        shift = m3.group(3)
        return source, (f"{year}_{day}{mnum}S{shift}" if year else f"{day}{mnum}S{shift}"), "JEE_MAIN"

    # ── Shift-only fallback ──────────────────────────────────────────────
    shift_only = re.search(r"(?:shift|s)_?([12])", s)
    if shift_only:
        return source, (f"{year}_S{shift_only.group(1)}" if year else f"S{shift_only.group(1)}"), "JEE_MAIN"

    if year and ("main" in s or s.startswith(year)):
        return source, None, "JEE_MAIN"

    pretty = stem.replace("_", " ").replace("-", " ").title()
    return pretty, None, "JEE_MAIN"


# ── Row builder ───────────────────────────────────────────────────────────────

def build_row(q: dict, subject: str, source: str, shift_date: str | None, exam_type: str) -> dict:
    raw_correct = q.get("correct", "")
    if isinstance(raw_correct, list):
        correct = ",".join(str(c).strip() for c in raw_correct)
    else:
        correct = str(raw_correct).strip()

    pct = q.get("percent correct")
    if pct is None:
        pct = q.get("percent_correct")
    pct = int(pct) if pct is not None else 0

    difficulty_raw = q.get("difficulty")
    difficulty = (difficulty_raw or "medium").strip().lower()
    if difficulty not in ("easy", "medium", "hard"):
        difficulty = "medium"

    # Normalise option keys: source data uses "option", Go model expects "key"
    raw_opts = q.get("options", [])
    normalized_opts = []
    for opt in raw_opts:
        if isinstance(opt, dict):
            opt_copy = dict(opt)
            if "option" in opt_copy and "key" not in opt_copy:
                opt_copy["key"] = opt_copy.pop("option")
            normalized_opts.append(opt_copy)
        else:
            normalized_opts.append(opt)

    content_key = f"{subject}|{q.get('chapter', 'unknown')}|{q.get('text', q.get('question_text', ''))}|{json.dumps(normalized_opts, sort_keys=True)}"
    stable_id = hashlib.md5(content_key.encode()).hexdigest()[:12]

    # Avoid double-looking up timespent
    _timespent = q.get("timespent")
    if _timespent is None:
        _timespent = q.get("timespent_avg_ms") or 0

    return {
        "id":                stable_id,
        "type":              q.get("type", "MCQ").upper(),
        "question_text":     q.get("text") or q.get("question_text", ""),
        "images":            json.dumps(q.get("images", [])),
        "options":           json.dumps(normalized_opts),
        "correct":           correct,
        "subject":           subject,
        "source":            source,
        "shift_date":        shift_date,
        "exam_type":         exam_type,
        "chapter":           q.get("chapter", "unknown"),
        "chapter_group":     q.get("chapterGroup") or q.get("chapter_group", "unknown"),
        "difficulty":        difficulty,
        "is_bonus":          bool(q.get("isBonus") or q.get("is_bonus", False)),
        "glicko_rating":     q.get("glicko_rating", 1500.0),
        "glicko_rd":         q.get("glicko_rd", 350.0),
        "glicko_volatility": q.get("glicko_volatility", 0.06),
        "attempt_count":     0,
        "percent_correct":   pct,
        "timespent_avg_ms":  _timespent,
        "embedding":         q.get("embedding"),
    }


# ── Database helpers ──────────────────────────────────────────────────────────

UPSERT = """
INSERT INTO questions (
    id, type, question_text, images, options, correct,
    subject, source, shift_date, exam_type, chapter, chapter_group, difficulty, is_bonus,
    glicko_rating, glicko_rd, glicko_volatility,
    attempt_count, percent_correct, timespent_avg_ms, embedding
) VALUES (
    %(id)s, %(type)s, %(question_text)s, %(images)s::jsonb, %(options)s::jsonb, %(correct)s,
    %(subject)s, %(source)s, %(shift_date)s, %(exam_type)s, %(chapter)s, %(chapter_group)s, %(difficulty)s, %(is_bonus)s,
    %(glicko_rating)s, %(glicko_rd)s, %(glicko_volatility)s,
    %(attempt_count)s, %(percent_correct)s, %(timespent_avg_ms)s,
    %(embedding)s::vector
)
ON CONFLICT (id) DO UPDATE SET
    type             = EXCLUDED.type,
    question_text    = EXCLUDED.question_text,
    images           = EXCLUDED.images,
    options          = EXCLUDED.options,
    correct          = EXCLUDED.correct,
    subject          = EXCLUDED.subject,
    source           = EXCLUDED.source,
    shift_date       = EXCLUDED.shift_date,
    exam_type        = EXCLUDED.exam_type,
    chapter          = EXCLUDED.chapter,
    chapter_group    = EXCLUDED.chapter_group,
    difficulty       = EXCLUDED.difficulty,
    is_bonus         = EXCLUDED.is_bonus,
    glicko_rating    = EXCLUDED.glicko_rating,
    glicko_rd        = EXCLUDED.glicko_rd,
    glicko_volatility = EXCLUDED.glicko_volatility,
    attempt_count    = EXCLUDED.attempt_count,
    percent_correct  = EXCLUDED.percent_correct,
    timespent_avg_ms = EXCLUDED.timespent_avg_ms,
    embedding        = EXCLUDED.embedding;
"""

UPSERT_NO_EMBEDDING = """
INSERT INTO questions (
    id, type, question_text, images, options, correct,
    subject, source, shift_date, exam_type, chapter, chapter_group, difficulty, is_bonus,
    glicko_rating, glicko_rd, glicko_volatility,
    attempt_count, percent_correct, timespent_avg_ms
) VALUES (
    %(id)s, %(type)s, %(question_text)s, %(images)s::jsonb, %(options)s::jsonb, %(correct)s,
    %(subject)s, %(source)s, %(shift_date)s, %(exam_type)s, %(chapter)s, %(chapter_group)s, %(difficulty)s, %(is_bonus)s,
    %(glicko_rating)s, %(glicko_rd)s, %(glicko_volatility)s,
    %(attempt_count)s, %(percent_correct)s, %(timespent_avg_ms)s
)
ON CONFLICT (id) DO UPDATE SET
    type             = EXCLUDED.type,
    question_text    = EXCLUDED.question_text,
    images           = EXCLUDED.images,
    options          = EXCLUDED.options,
    correct          = EXCLUDED.correct,
    subject          = EXCLUDED.subject,
    source           = EXCLUDED.source,
    shift_date       = EXCLUDED.shift_date,
    exam_type        = EXCLUDED.exam_type,
    chapter          = EXCLUDED.chapter,
    chapter_group    = EXCLUDED.chapter_group,
    difficulty       = EXCLUDED.difficulty,
    is_bonus         = EXCLUDED.is_bonus,
    glicko_rating    = EXCLUDED.glicko_rating,
    glicko_rd        = EXCLUDED.glicko_rd,
    glicko_volatility = EXCLUDED.glicko_volatility,
    attempt_count    = EXCLUDED.attempt_count,
    percent_correct  = EXCLUDED.percent_correct,
    timespent_avg_ms = EXCLUDED.timespent_avg_ms;
"""


def run_schema(conn: psycopg.Connection) -> None:
    sql = SCHEMA_FILE.read_text()
    with conn.cursor() as cur:
        cur.execute(sql)
    conn.commit()
    print(f"  ✓ schema applied from {SCHEMA_FILE}")


def load_questions(data: dict, source: str, shift_date: str | None, exam_type: str) -> list[dict]:
    rows = []
    for subject_key in ("physics", "chemistry", "maths"):
        items = data.get(subject_key, [])
        if not isinstance(items, list):
            continue
        for q in items:
            if not isinstance(q, dict):
                continue
            if not q.get("text") and not q.get("question_text"):
                continue
            rows.append(build_row(q, subject_key, source, shift_date, exam_type))
    return rows


def main() -> None:
    db_url = os.environ.get("DATABASE_URL",
             "postgresql://velocity:velocity@localhost:5432/alphajee")

    files = sorted(DATA_DIR.rglob("*_final.json"))
    if not files:
        print(f"✗  No *_final.json files found under {DATA_DIR}")
        print("   Make sure generate_embeddings.py has run first.")
        sys.exit(1)

    print(f"\n⚡ AlphaJEE seed — {len(files)} file(s) found\n")

    total_seeded = 0
    total_skipped = 0

    with psycopg.connect(db_url) as conn:
        for fpath in files:
            source, shift_date, exam_type = parse_filename(fpath.stem)
            print(f"  → {fpath.name}  ({source} / {shift_date or '—'} / {exam_type})")

            try:
                questions = json.loads(fpath.read_text())
            except json.JSONDecodeError as e:
                print(f"     ✗ JSON parse error: {e} — skipping")
                continue

            if not isinstance(questions, dict):
                print(f"     ✗ expected a JSON object — skipping")
                continue

            rows = load_questions(questions, source, shift_date, exam_type)
            if not rows:
                print(f"     ~ no valid questions found — skipping")
                continue

            with conn.cursor() as cur:
                with_emb    = [r for r in rows if r["embedding"] is not None]
                without_emb = [r for r in rows if r["embedding"] is None]
                if with_emb:
                    cur.executemany(UPSERT, with_emb)
                if without_emb:
                    cur.executemany(UPSERT_NO_EMBEDDING, without_emb)
            conn.commit()

            total_seeded += len(rows)
            print(f"     ✓ {len(rows)} rows upserted")

    print(f"\n✅ Done — {total_seeded} questions seeded, {total_skipped} skipped\n")


if __name__ == "__main__":
    main()
