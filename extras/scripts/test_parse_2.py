import json
import os
import re
from bs4 import BeautifulSoup

def parse_html_for_io(html):
    soup = BeautifulSoup(html, 'html.parser')
    examples = []
    
    for pre in soup.find_all('pre'):
        text = pre.get_text()
        input_match = re.search(r'Input:\s*(.*?)\n\s*Output:\s*(.*?)(?:\n\s*Explanation|\n|$)', text, re.DOTALL | re.IGNORECASE)
        if input_match:
            inp = input_match.group(1).strip()
            out = input_match.group(2).strip()
            examples.append({"input": inp, "output": out})
    return examples

count = 0
for f in os.listdir("data"):
    if not f.endswith(".json"): continue
    with open(f"data/{f}") as fp:
        d = json.load(fp)
    html = d.get("body", "")
    if html:
        exs = parse_html_for_io(html)
        if not exs:
            count += 1
print(f"Empty examples for {count} problems.")
