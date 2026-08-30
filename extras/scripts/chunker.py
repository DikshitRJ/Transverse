import json
import math
import os

os.makedirs("data/problems/chunks", exist_ok=True)
chunk_index = 0

def chunk_file(file_path, source, num_chunks):
    global chunk_index
    with open(file_path, "r") as f:
        data = json.load(f)
    
    chunk_size = math.ceil(len(data) / num_chunks)
    if chunk_size == 0: return
    
    for i in range(num_chunks):
        chunk_data = data[i*chunk_size : (i+1)*chunk_size]
        if not chunk_data: break
        out_path = f"data/problems/chunks/chunk_{chunk_index}_{source}.json"
        with open(out_path, "w") as f:
            json.dump(chunk_data, f, indent=2)
        chunk_index += 1

# Let's target about 10 subagents overall.
# Codeforces: 11370 problems -> 5 chunks
# AtCoder: 9435 problems -> 4 chunks
# CSES: 400 problems -> 1 chunk
# Total = 10 chunks
chunk_file("data/problems/codeforces.json", "codeforces", 5)
chunk_file("data/problems/atcoder.json", "atcoder", 4)
chunk_file("data/problems/cses.json", "cses", 1)

print(f"Created {chunk_index} chunks.")
