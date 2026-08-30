import os
import json
import re
from bs4 import BeautifulSoup

def parse_html_for_io(html):
    soup = BeautifulSoup(html, 'html.parser')
    examples = []
    
    # Standard format: in <pre> tags
    for pre in soup.find_all('pre'):
        text = pre.get_text()
        input_match = re.search(r'Input:\s*(.*?)\n\s*Output:\s*(.*?)(?:\n\s*Explanation|\n|$)', text, re.DOTALL | re.IGNORECASE)
        if input_match:
            inp = input_match.group(1).strip()
            out = input_match.group(2).strip()
            examples.append({"input": inp, "output": out})
            
    # Fallback if no <pre> block matched
    if not examples:
        text = soup.get_text()
        # Find all occurrences of Input: and Output:
        for match in re.finditer(r'Input:\s*(.*?)\s*Output:\s*(.*?)(?:\s*Explanation:|\s*Example|\s*Constraints:|$)', text, re.DOTALL | re.IGNORECASE):
            inp = match.group(1).strip()
            out = match.group(2).strip()
            if len(inp) < 1000 and len(out) < 1000:
                examples.append({"input": inp, "output": out})
                
    return examples

def process_file(f):
    if not f.endswith(".json"): return None
    try:
        with open(f"data/{f}") as fp:
            d = json.load(fp)
    except:
        return None
        
    if not isinstance(d, dict):
        return None
        
    html = d.get("body", "")
    
    input_test_cases = []
    output_test_cases = []
    if html:
        exs = parse_html_for_io(html)
        for ex in exs:
            input_test_cases.append(ex["input"])
            output_test_cases.append(ex["output"])
            
    d["input_test_cases"] = input_test_cases
    d["output_test_cases"] = output_test_cases
    
    d["time_limit_sec"] = 2.0
    d["memory_limit_mb"] = 256.0
    
    return d

def main():
    files = os.listdir("data")
    all_problems = []
    
    print(f"Processing {len(files)} files...")
    for f in files:
        res = process_file(f)
        if res:
            all_problems.append(res)
            
    os.makedirs("generated", exist_ok=True)
    out_path = "generated/leetcode_problems.json"
    print(f"Writing to {out_path}...")
    with open(out_path, "w") as fp:
        json.dump(all_problems, fp, separators=(',', ':'))  # more compact JSON
    print("Done!")

if __name__ == "__main__":
    main()
