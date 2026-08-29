# Transverse problem corpus extractor

Build the JSON corpus with Python 3.10+ and no third-party packages:

```bash
python3 scripts/extract_problems.py
```

The command writes `codeforces.json`, `cses.json`, `atcoder.json`,
`leetcode_index.json`, and `all_problems.json` to `data/generated/`, and logs
the count for every source. Use `--output-dir <directory>` to write elsewhere.

The extractor uses a single official Codeforces batch API request, requests
only CSES's public list page after checking its `robots.txt`, and uses the
public AtCoder Problems APIs. It never requests LeetCode pages: the
`data/leetcode_index_seed.json` file is local, hand-curated link metadata and
can be extended without adding statements, constraints, or examples.

If a public source fails, the command still writes a valid empty file for that
source and builds the merged file from successful sources. For an offline check
of the local index and output schema, run:

```bash
python3 scripts/extract_problems.py --skip-network
```
