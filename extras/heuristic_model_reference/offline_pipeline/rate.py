#!/usr/bin/env python3
import os
import json
import glob
import math
import statistics
from pathlib import Path
import warnings

BASE_DIR = Path(__file__).resolve().parent.parent
DATA_DIR = BASE_DIR / "data" / "24_25_26_&_adv"
OUT_DIR  = BASE_DIR / "data" / "rated"

ELO_CENTER       = 1500.0
ELO_SCALE        = 173.717
TIME_SCALE       = 150.0
TIME_CLAMP       = 200.0
DEFAULT_BASELINE = 60.0   # fallback seconds
SPARSE_THRESHOLD = 10     # min samples before a bucket's median is trusted

SUBJECTS    = ("physics", "chemistry", "maths")
DIFFICULTIES = ("easy", "medium", "hard")
ACCURACY_TIME_INTERACTION = 0.3


def get_json_files(data_dir):
    pattern = str(data_dir / "**" / "*_final.json")
    return [Path(f) for f in sorted(glob.glob(pattern, recursive=True))]


def extract_timespent_baselines(files):
    """
    Pass 1: bucket timespent (seconds) by subject × difficulty.
    Returns nested dict: baselines[subject][difficulty] = median_seconds.

    Fallback chain per lookup:
      1. subject + matched difficulty   (if >= SPARSE_THRESHOLD samples)
      2. subject + "medium"             (if >= SPARSE_THRESHOLD samples)
      3. subject-wide median            (all difficulties pooled)
      4. DEFAULT_BASELINE
    """
    # Accumulate raw lists
    buckets = {s: {d: [] for d in DIFFICULTIES} for s in SUBJECTS}

    print("Pass 1: extracting timespent telemetry...")
    for f in files:
        try:
            with open(f, "r", encoding="utf-8") as fh:
                data = json.load(fh)
        except Exception as e:
            print(f"  Warning: skipping {f.name} — {e}")
            continue

        for subject in SUBJECTS:
            lst = data.get(subject)
            if not isinstance(lst, list):
                continue
            for item in lst:
                if not isinstance(item, dict):
                    continue
                ts = item.get("timespent")
                if ts is None:
                    continue
                try:
                    t_sec = float(ts) / 1000.0
                except (ValueError, TypeError):
                    continue
                if t_sec <= 0 or not math.isfinite(t_sec):
                    continue

                diff = item.get("difficulty", "medium")
                if not isinstance(diff, str) or diff.lower() not in DIFFICULTIES:
                    diff = "medium"
                else:
                    diff = diff.lower()

                buckets[subject][diff].append(t_sec)

    # Compute medians; build resolved baseline dict
    baselines = {}
    for subject in SUBJECTS:
        baselines[subject] = {}
        all_times = []
        for diff in DIFFICULTIES:
            times = buckets[subject][diff]
            all_times.extend(times)
            if len(times) >= SPARSE_THRESHOLD:
                med = statistics.median(times)
                baselines[subject][diff] = med
                print(f"  {subject:10s} [{diff:6s}]  median={med:.1f}s  n={len(times)}")
            else:
                baselines[subject][diff] = None   # resolve at call time
                print(f"  {subject:10s} [{diff:6s}]  sparse (n={len(times)}) — will fall back")

        # Subject-wide fallback
        baselines[subject]["_all"] = statistics.median(all_times) if all_times else DEFAULT_BASELINE

    return baselines


def resolve_baseline(baselines, subject, difficulty):
    bucket = baselines.get(subject, {})
    val = bucket.get(difficulty)
    if val is not None:
        return val
    val = bucket.get("medium")
    if val is not None:
        return val
    val = bucket.get("_all")
    if val is not None:
        return val
    return DEFAULT_BASELINE


def compute_rating(percent_correct, timespent, baseline):
    """
    Rasch logit difficulty projected onto Elo scale, plus a
    difficulty-normalised time divergence modifier.
    """
    # --- Accuracy component ---
    if percent_correct is None:
        p_val = 0.50
    else:
        try:
            p_val = float(percent_correct) / 100.0
        except (ValueError, TypeError):
            p_val = 0.50

    p = max(0.001, min(0.999, p_val))
    b = math.log((1.0 - p) / p)           # Rasch logit
    r_accuracy = ELO_CENTER + (b * ELO_SCALE)

    # --- Time divergence component ---
    delta_r_time = 0.0
    if timespent is not None:
        try:
            t_sec = float(timespent) / 1000.0
            if t_sec > 0 and baseline > 0:
                raw_time = TIME_SCALE * math.log(t_sec / baseline)
                accuracy_weight = 1.0 - ACCURACY_TIME_INTERACTION * (1.0 - p)
                delta_r_time = raw_time * accuracy_weight
        except (ValueError, TypeError):
            pass

    delta_r_time = max(-TIME_CLAMP, min(TIME_CLAMP, delta_r_time))

    return int(round(r_accuracy + delta_r_time))


def process_and_enrich_files(files, baselines, source_dir, dest_dir):
    """Pass 2: enrich each question with rating fields; write atomically."""
    print("Pass 2: rating questions and writing output...")
    processed = 0

    for f in files:
        try:
            with open(f, "r", encoding="utf-8") as fh:
                data = json.load(fh)
        except Exception as e:
            print(f"  Error loading {f.name}: {e} — skipping")
            continue

        for subject in SUBJECTS:
            lst = data.get(subject)
            if not isinstance(lst, list):
                continue
            for item in lst:
                if not isinstance(item, dict):
                    continue

                diff = item.get("difficulty", "medium")
                if not isinstance(diff, str) or diff.lower() not in DIFFICULTIES:
                    diff = "medium"
                else:
                    diff = diff.lower()

                baseline = resolve_baseline(baselines, subject, diff)

                item["glicko_rating"]     = compute_rating(
                    item.get("percent correct"),
                    item.get("timespent"),
                    baseline,
                )
                item["glicko_rd"]         = 350.0
                item["glicko_volatility"] = 0.06

        # Resolve destination path
        try:
            rel = f.relative_to(source_dir)
        except ValueError:
            rel = Path(f.name)
        target = dest_dir / rel
        target.parent.mkdir(parents=True, exist_ok=True)

        tmp = target.with_suffix(".json.tmp")
        try:
            with open(tmp, "w", encoding="utf-8") as fh:
                json.dump(data, fh, indent=2, ensure_ascii=False)
            with open(tmp, "r", encoding="utf-8") as fh:
                json.load(fh)                      # verify
            os.replace(tmp, target)
            processed += 1
        except Exception as e:
            print(f"  Error writing {target.name}: {e}")
            if tmp.exists():
                try:
                    tmp.unlink()
                except Exception:
                    pass

    print(f"Done — {processed} files written to {dest_dir}")


def main():
    print(f"Source : {DATA_DIR}")
    print(f"Output : {OUT_DIR}")

    if not DATA_DIR.exists():
        print(f"Error: {DATA_DIR} does not exist.")
        return

    files = get_json_files(DATA_DIR)
    if not files:
        print("No *_final.json files found.")
        return
    print(f"Found {len(files)} files.\n")

    baselines = extract_timespent_baselines(files)
    print()
    process_and_enrich_files(files, baselines, DATA_DIR, OUT_DIR)


if __name__ == "__main__":
    main()