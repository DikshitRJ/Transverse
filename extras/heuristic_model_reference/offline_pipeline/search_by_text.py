#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "numpy",
#     "psycopg[binary]>=3.3.4",
#     "sentence-transformers",
# ]
# ///
"""
Velocity Question Search by Text
───────────────────────────────
Uses pgvector HNSW index for fast cosine similarity search instead of
loading all questions into memory and computing brute-force numpy.
"""

import os
import sys
import json
import numpy as np
import psycopg
from sentence_transformers import SentenceTransformer


MODEL_NAME = 'BAAI/bge-small-en-v1.5'
_model = None

def get_model():
    global _model
    if _model is None:
        _model = SentenceTransformer(MODEL_NAME, trust_remote_code=True)
    return _model

DB_URL = os.environ.get("DATABASE_URL",
         "postgresql://velocity:velocity@localhost:5432/alphajee")


def embed_text(text):
    model = get_model()
    emb = model.encode([text], batch_size=1, show_progress_bar=False)[0]
    return np.array(emb)


def display_question(q, title="QUESTION DETAILS"):
    border = "=" * 80
    print(f"\n{border}")
    print(f" {title}")
    print(border)
    print(f"  ID            : {q.get('id', 'N/A')}")
    print(f"  Subject       : {q.get('_subject', 'N/A').upper()}")
    print(f"  Chapter       : {q.get('chapter', 'N/A')}")
    print(f"  Chapter Group : {q.get('chapterGroup', 'N/A')}")
    print(f"  Glicko Rating : {q.get('glicko_rating', 'N/A')} (rd={q.get('glicko_rd', 350.0)})")
    print("-" * 80)

    text = q.get("text", "").strip()
    print("  Text:")
    for line in text.splitlines():
        print(f"    {line}")
    print("-" * 80)

    options = q.get("options", [])
    if options:
        print("  Options:")
        for opt in options:
            if isinstance(opt, dict):
                opt_letter = opt.get("option", "")
                opt_text = opt.get("text", "").strip()
                print(f"    [{opt_letter}] {opt_text}")
    else:
        print("  Options       : None (Numerical/Subjective)")

    print("-" * 80)
    print(f"  Correct Answer: {q.get('correct', 'N/A')}")
    print(border + "\n")


def search_sql(query_emb, subject=None, top_n=10):
    """
    Search questions using pgvector's <=> cosine distance operator.
    Leverages the HNSW index for O(log n) search instead of brute force.
    """
    emb_list = query_emb.tolist()
    emb_json = json.dumps(emb_list)

    with psycopg.connect(DB_URL) as conn:
        with conn.cursor() as cur:
            if subject:
                cur.execute("""
                    SELECT id, type, question_text, options, correct,
                           subject, chapter, chapter_group, difficulty,
                           glicko_rating, glicko_rd, shift_date, source, exam_type,
                           embedding <=> %s::vector AS dist
                    FROM questions
                    WHERE subject = %s
                      AND embedding IS NOT NULL
                    ORDER BY embedding <=> %s::vector
                    LIMIT %s
                """, (emb_json, subject, emb_json, top_n))
            else:
                cur.execute("""
                    SELECT id, type, question_text, options, correct,
                           subject, chapter, chapter_group, difficulty,
                           glicko_rating, glicko_rd, shift_date, source, exam_type,
                           embedding <=> %s::vector AS dist
                    FROM questions
                    WHERE embedding IS NOT NULL
                    ORDER BY embedding <=> %s::vector
                    LIMIT %s
                """, (emb_json, emb_json, top_n))

            rows = cur.fetchall()

    results = []
    for row in rows:
        (
            qid, qtype, text, options_raw, correct,
            subject, chapter, chapter_group, difficulty,
            glicko_rating, glicko_rd, shift_date, source, exam_type,
            dist
        ) = row

        # Cosine similarity from distance: sim = 1 - dist (for <=> which is 1 - cosine)
        similarity = 1.0 - float(dist)

        options = json.loads(options_raw) if options_raw else []

        results.append({
            "id": qid,
            "_subject": subject,
            "chapter": chapter,
            "chapterGroup": chapter_group,
            "glicko_rating": glicko_rating,
            "glicko_rd": glicko_rd,
            "text": text,
            "options": options,
            "correct": correct,
            "similarity": similarity,
        })

    return results


VALID_SUBJECTS = ("physics", "chemistry", "maths")


def parse_args():
    subject = None
    args = sys.argv[1:]
    text_parts = []

    i = 0
    while i < len(args):
        if args[i] in ("-s", "--subject") and i + 1 < len(args):
            subject = args[i + 1].lower()
            i += 2
        else:
            text_parts.append(args[i])
            i += 1

    query_text = " ".join(text_parts) if text_parts else None
    return query_text, subject


def main():
    print("Velocity Question Search by Text")
    print("=" * 60)
    print("Using pgvector HNSW index\n")

    query_text, subject = parse_args()

    if not query_text:
        query_text = input("Enter your search text: ").strip()

    if not query_text:
        print("No text entered. Exiting.")
        return

    if not subject:
        sub_in = input("Filter by subject? (physics/chemistry/maths, or press Enter for all): ").strip().lower()
        if sub_in in VALID_SUBJECTS:
            subject = sub_in

    if subject and subject not in VALID_SUBJECTS:
        print(f"Invalid subject '{subject}'. Choose from {VALID_SUBJECTS}.")
        return

    print(f"\nGenerating embedding for: \"{query_text[:80]}{'...' if len(query_text) > 80 else ''}\"")
    query_emb = embed_text(query_text)
    print("Embedding generated.")

    print("Searching via pgvector HNSW index...")
    results = search_sql(query_emb, subject, top_n=10)

    if not results:
        print("No similar questions found.")
        return

    print("\nTop 10 Most Similar Questions:")
    print("-" * 80)
    for idx, r in enumerate(results, 1):
        snippet = r["text"].replace("\n", " ").strip()
        if len(snippet) > 80:
            snippet = snippet[:77] + "..."
        sim_pct = r["similarity"] * 100
        print(f"  [{idx}] Sim: {sim_pct:5.1f}% | ID: {r['id']:8s} | Sub: {r['_subject'].upper():9s} | Chap: {str(r['chapter']):15s} | {snippet}")
    print("-" * 80)

    while True:
        sub_choice = input("\nEnter option number (1-10) to view in full, or 'q' to quit: ").strip().lower()
        if sub_choice == 'q':
            print("Exiting.")
            return
        try:
            idx_val = int(sub_choice)
            if 1 <= idx_val <= len(results):
                selected_q = results[idx_val - 1]
                sim_score = selected_q["similarity"]
                display_question(selected_q, title=f"QUESTION DETAIL (Cosine Similarity: {sim_score*100:.1f}%)")
            else:
                print(f"Please enter a number between 1 and {len(results)}.")
        except ValueError:
            print("Invalid input. Enter a number or 'q'.")


if __name__ == "__main__":
    main()
