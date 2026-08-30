import json
import re
from bs4 import BeautifulSoup

def parse_html_for_io(html):
    soup = BeautifulSoup(html, 'html.parser')
    examples = []
    
    # Usually in <pre> tags
    for pre in soup.find_all('pre'):
        text = pre.get_text()
        input_match = re.search(r'Input:\s*(.*?)\n\s*Output:\s*(.*?)(?:\n|$)', text, re.DOTALL | re.IGNORECASE)
        if input_match:
            inp = input_match.group(1).strip()
            out = input_match.group(2).strip()
            examples.append({"input": inp, "output": out})
        else:
            # Maybe strong tags instead of plain text inside pre
            # E.g. <strong>Input:</strong> ... <strong>Output:</strong> ...
            pass
    return examples

with open("data/two-sum.json") as f:
    d = json.load(f)
    print("Test cases:")
    print(parse_html_for_io(d["body"]))

