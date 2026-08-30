import json
import math
import os

with open("data/problems/codeforces.json", "r") as f:
    data = json.load(f)

num_chunks = 10
chunk_size = math.ceil(len(data) / num_chunks)
for i in range(num_chunks):
    chunk_data = data[i*chunk_size : (i+1)*chunk_size]
    out_path = f"data/problems/chunks/chunk_{i}_codeforces.json"
    with open(out_path, "w") as f:
        json.dump(chunk_data, f, indent=2)
print("10 Codeforces chunks created.")
