#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "numpy",
#     "psycopg[binary]>=3.3.4",
# ]
# ///

import os
import json
import random
import numpy as np
import psycopg


def parse_embedding(emb):
    if emb is None:
        return None
    if isinstance(emb, (list, tuple)):
        return emb
    if isinstance(emb, str):
        return json.loads(emb)
    try:
        return list(emb)
    except TypeError:
        return None

DB_URL = os.environ.get("DATABASE_URL",
         "postgresql://velocity:velocity@localhost:5432/alphajee")


def load_all_processed_questions():
    with psycopg.connect(DB_URL) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT * FROM questions WHERE embedding IS NOT NULL")
            columns = [desc.name for desc in cur.description]
            questions = []
            for row in cur:
                q = dict(zip(columns, row))
                q["text"] = q.pop("question_text", "")
                q["chapterGroup"] = q.pop("chapter_group", "N/A")
                q["isBonus"] = q.pop("is_bonus", False)
                q["_subject"] = q.pop("subject", "N/A")
                q["_source_file"] = "DB"
                emb = q.pop("embedding", None)
                q["embedding"] = parse_embedding(emb)
                if isinstance(q.get("options"), str):
                    q["options"] = json.loads(q["options"])
                if isinstance(q.get("images"), str):
                    q["images"] = json.loads(q["images"])
                questions.append(q)

    print(f"Successfully loaded {len(questions)} questions with embeddings from PostgreSQL.")
    return questions


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
    print(f"  Source File   : {q.get('_source_file', 'N/A')}")
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


def find_similar_questions(target_q, all_questions, top_n=5):
    target_chapter = target_q.get("chapter")
    target_id = target_q.get("id")

    candidates = [
        q for q in all_questions
        if q.get("chapter") == target_chapter and q.get("id") != target_id
    ]

    if not candidates:
        return []

    target_emb = np.array(target_q["embedding"])
    embs = np.array([q["embedding"] for q in candidates])

    target_norm = np.linalg.norm(target_emb)
    embs_norms = np.linalg.norm(embs, axis=1)

    if target_norm == 0:
        return []
    embs_norms[embs_norms == 0] = 1.0

    similarities = np.dot(embs, target_emb) / (target_norm * embs_norms)

    sorted_indices = np.argsort(similarities)[::-1]

    results = []
    for idx in sorted_indices:
        q = candidates[idx]
        sim = similarities[idx]
        results.append((q, sim))
        if len(results) >= top_n:
            break

    return results


def pick_random_question(questions):
    return random.choice(questions)


def prompt_yes_no_quit(prompt_text):
    while True:
        choice = input(prompt_text).strip().lower()
        if choice in ('q', 'n', 'y'):
            return choice
        print("Invalid choice. Please enter 'y', 'n', or 'q'.")


def prompt_option_number(similar_results):
    while True:
        sub_choice = input("Enter option number (1-5) to view in full, or 'n' to return: ").strip().lower()
        if sub_choice == 'n':
            return None
        try:
            idx_val = int(sub_choice)
            if 1 <= idx_val <= len(similar_results):
                return idx_val - 1
            print(f"Please enter a number between 1 and {len(similar_results)}.")
        except ValueError:
            print("Invalid input. Please enter 1-5 or 'n'.")


def display_similar_results_header():
    print("\nTop 5 Most Similar Questions:")
    print("-" * 80)


def display_similar_result_row(idx, q, sim):
    snippet = q.get("text", "").replace("\n", " ").strip()
    if len(snippet) > 80:
        snippet = snippet[:77] + "..."
    print(f"  [{idx}] Sim: {sim*100:5.1f}% | ID: {q.get('id'):8s} | Sub: {q.get('_subject').upper():9s} | {snippet}")


def display_similar_results_footer():
    print("-" * 80)


def handle_similar_questions(target_q, questions):
    print("\nCalculating cosine similarities across the corpus...")
    similar_results = find_similar_questions(target_q, questions, top_n=5)
    if not similar_results:
        print("No similar questions found.")
        return

    display_similar_results_header()
    for idx, (q, sim) in enumerate(similar_results, 1):
        display_similar_result_row(idx, q, sim)
    display_similar_results_footer()

    selected = prompt_option_number(similar_results)
    if selected is None:
        return

    selected_sim_q, sim_score = similar_results[selected]
    display_question(selected_sim_q, title=f"SIMILAR QUESTION DETAIL (Cosine Similarity: {sim_score*100:.1f}%)")
    input("Press Enter to return to results...")


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Test embedding similarity search")
    parser.add_argument("--iterations", type=int, default=0, help="Max iterations (0 = infinite, default)")
    args = parser.parse_args()

    print("Velocity Question & Embedding Search Test Utility")
    print("=" * 60)

    questions = load_all_processed_questions()
    if not questions:
        return

    iter_count = 0
    while True:
        iter_count += 1
        if args.iterations > 0 and iter_count > args.iterations:
            print(f"Reached {args.iterations} iteration(s). Exiting.")
            return

        target_q = pick_random_question(questions)
        display_question(target_q)

        choice = prompt_yes_no_quit("Would you like to view similar questions? (y/n/q to quit): ")
        if choice == 'q':
            print("Exiting. Keep coding!")
            return
        if choice == 'n':
            continue

        handle_similar_questions(target_q, questions)


if __name__ == "__main__":
    main()
