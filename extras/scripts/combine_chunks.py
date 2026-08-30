import json
import os
import glob

def combine_for_source(source):
    chunks = glob.glob(f"data/problems/chunks/out_chunk_*_{source}.json")
    if not chunks:
        print(f"No chunks found for {source}")
        return
        
    all_data = []
    for chunk in chunks:
        try:
            with open(chunk, "r") as f:
                all_data.extend(json.load(f))
        except Exception as e:
            print(f"Error reading {chunk}: {e}")
            
    if all_data:
        out_path = f"data/problems/{source}.json"
        with open(out_path, "w") as f:
            json.dump(all_data, f, indent=2)
        print(f"Combined {len(all_data)} problems for {source} into {out_path}")

combine_for_source("codeforces")
combine_for_source("atcoder")
combine_for_source("cses")

print("Finished combining chunks.")
