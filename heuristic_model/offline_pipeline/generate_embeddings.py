# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "sentence-transformers",
#     "tqdm",
# ]
# ///

import os
# Set thread limits per worker before importing sentence-transformers/torch
os.environ["OMP_NUM_THREADS"] = "1"
os.environ["MKL_NUM_THREADS"] = "1"
os.environ["OPENBLAS_NUM_THREADS"] = "1"
os.environ["VECLIB_MAXIMUM_THREADS"] = "1"
os.environ["NUMEXPR_NUM_THREADS"] = "1"

import json
import glob
import time
import shutil
from pathlib import Path
from concurrent.futures import ProcessPoolExecutor, as_completed
from tqdm import tqdm # type: ignore

# Paths
BASE_DIR = Path(__file__).resolve().parent.parent
RATED_DIR = BASE_DIR / "data" / "rated"
PROCESSED_DIR = BASE_DIR / "data" / "processed"
EXPECTED_DIM = 384  # BAAI/bge-small-en-v1.5 embedding dimension

# Global model instance for worker processes (lazy-loaded once per worker)
_model = None

def get_model():
    global _model
    if _model is None:
        from sentence_transformers import SentenceTransformer # type: ignore
        # Lazy load BGE model
        _model = SentenceTransformer('BAAI/bge-small-en-v1.5', trust_remote_code=True)
    return _model

def process_file_worker(args):
    """Worker function that processes a single JSON file."""
    src_file, dest_file = args
    start_time = time.time()
    
    stats = {
        "file_name": src_file.name,
        "total_questions": 0,
        "missing_text_warnings": [],
        "success": False,
        "error": None
    }
    
    try:
        # Load rated file
        with open(src_file, "r", encoding="utf-8") as f:
            data = json.load(f)
    except Exception as e:
        stats["error"] = f"Failed to load file: {e}"
        return stats

    # Get local model instance (will load model on first call in this process)
    model = get_model()

    # We will gather all questions to batch embed them
    questions_to_embed = []
    
    for subject, lst in data.items():
        if subject not in ("physics", "chemistry", "maths"):
            continue
        if not isinstance(lst, list):
            continue
        
        for item in lst:
            if not isinstance(item, dict):
                continue
            
            stats["total_questions"] += 1
            q_id = item.get("id", "unknown")
            text = item.get("text")
            vis_desc = item.get("visual_description")
            
            # Formulate text for embedding
            text_str = ""
            if text and isinstance(text, str):
                text_str = text.strip()
            
            if vis_desc and isinstance(vis_desc, str):
                text_str += " " + vis_desc.strip()
                
            if not text_str:
                warning_msg = f"Question ID '{q_id}' has missing or empty text/visual_description."
                stats["missing_text_warnings"].append(warning_msg)
                # Fallback to empty string so index alignment is preserved
                text_str = ""
                
            questions_to_embed.append({
                "item_ref": item,
                "text_to_embed": text_str
            })

    if not questions_to_embed:
        stats["success"] = True
        return stats

    # Batch encode the texts
    texts = [q["text_to_embed"] for q in questions_to_embed]
    try:
        embeddings = model.encode(texts, batch_size=32, show_progress_bar=False)
        if embeddings.shape[1] != EXPECTED_DIM:
            stats["error"] = f"Model returned dim {embeddings.shape[1]}, expected {EXPECTED_DIM}"
            return stats
    except Exception as e:
        stats["error"] = f"Model embedding failed: {e}"
        return stats

    # Write embeddings back to objects
    for q, emb in zip(questions_to_embed, embeddings):
        # Convert numpy array/list to a clean list of floats
        q["item_ref"]["embedding"] = emb.tolist()

    # Atomic write back to processed directory
    dest_file.parent.mkdir(parents=True, exist_ok=True)
    tmp_path = dest_file.with_suffix(".json.tmp")
    
    try:
        with open(tmp_path, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
            
        # Verify tmp file is valid JSON
        with open(tmp_path, "r", encoding="utf-8") as f:
            json.load(f)
            
        # Swap (with cross-filesystem fallback)
        try:
            os.replace(tmp_path, dest_file)
        except OSError:
            shutil.move(str(tmp_path), str(dest_file))
        stats["success"] = True
    except Exception as e:
        stats["error"] = f"Atomic write failed: {e}"
        if os.path.exists(tmp_path):
            try:
                os.remove(tmp_path)
            except Exception:
                pass

    stats["duration"] = time.time() - start_time
    return stats

def main():
    print("Velocity Embedding Generation Script started.")
    print(f"Source (Rated) Directory: {RATED_DIR}")
    print(f"Output (Processed) Directory: {PROCESSED_DIR}")

    if not RATED_DIR.exists():
        print(f"Error: Rated directory {RATED_DIR} does not exist. Run rate.py first.")
        return

    # Scan for files
    pattern = str(RATED_DIR / "**" / "*_final.json")
    rated_files = [Path(f) for f in sorted(glob.glob(pattern, recursive=True))]

    if not rated_files:
        print(f"No JSON files found in {RATED_DIR}")
        return

    print(f"Found {len(rated_files)} JSON files to embed.")

    # Build work queue (source_path, dest_path)
    work_queue = []
    for f in rated_files:
        try:
            rel = f.relative_to(RATED_DIR)
        except ValueError:
            rel = f.name
        dest = PROCESSED_DIR / rel
        work_queue.append((f, dest))

    # Multiprocessing configuration
    # Use 8 workers suitable for CPU
    max_workers = 8
    print(f"Starting multiprocessing pool with {max_workers} workers.")

    total_questions = 0
    total_missing_text = 0
    total_files_success = 0
    
    start_time = time.time()
    
    with ProcessPoolExecutor(max_workers=max_workers) as executor:
        # Submit tasks
        futures = {executor.submit(process_file_worker, task): task for task in work_queue}
        
        # Monitor progress using tqdm
        for future in tqdm(as_completed(futures), total=len(futures), desc="Embedding Files"):
            stats = future.result()
            file_name = stats["file_name"]
            
            if stats["success"]:
                total_files_success += 1
                total_questions += stats["total_questions"]
                missing_cnt = len(stats["missing_text_warnings"])
                total_missing_text += missing_cnt
                
                # Report stats for this file
                dur = stats.get("duration", 0.0)
                print(f"  [OK] {file_name:30s} | {stats['total_questions']:3d} Qs | {missing_cnt:2d} missing text | {dur:.2f}s")
                
                # Show individual warnings if any
                for warning in stats["missing_text_warnings"]:
                    print(f"    [WARNING] {file_name}: {warning}")
            else:
                print(f"  [ERROR] {file_name:30s} | Failed with error: {stats['error']}")

    total_duration = time.time() - start_time
    print("\n" + "="*60)
    print("EMBEDDING TASK COMPLETED")
    print(f"Total Time Taken     : {total_duration:.2f} seconds")
    print(f"Files Processed      : {total_files_success} / {len(rated_files)}")
    print(f"Total Questions      : {total_questions}")
    print(f"Missing Text Warnings: {total_missing_text}")
    print("="*60)

if __name__ == "__main__":
    main()
