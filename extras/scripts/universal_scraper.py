import sys
import json
import requests
from bs4 import BeautifulSoup
import time
import concurrent.futures
from tqdm import tqdm

def scrape_cses(url):
    try:
        r = requests.get(url, timeout=10, headers={"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"})
        if r.status_code != 200: return None
        soup = BeautifulSoup(r.content, "html.parser")
        
        content = soup.find("div", class_="content")
        if not content: return None
        
        # very basic time/memory limit parsing for CSES, usually in sidebar
        time_limit = "1.00 s"
        memory_limit = "512 MB"
        sidebar = soup.find("div", class_="nav sidebar")
        if sidebar:
            text = sidebar.get_text()
            if "Time limit:" in text:
                tl = text.split("Time limit:")[1].split("\n")[0].strip()
                time_limit = tl
            if "Memory limit:" in text:
                ml = text.split("Memory limit:")[1].split("\n")[0].strip()
                memory_limit = ml
                
        # input/output
        inputs = []
        outputs = []
        # CSES has <h1 id="example">Example</h1> followed by Input: <code>...</code> Output: <code>...</code>
        # Just grab all <code> inside content
        codes = content.find_all("code")
        for i in range(len(codes) - 1):
            # rudimentary heuristic
            pass
            
        # Or parse text
        text = content.get_text()
        parts = text.split("Input")
        if len(parts) > 1:
            io_part = parts[-1]
            if "Output" in io_part:
                inp = io_part.split("Output")[0].strip()
                out = io_part.split("Output")[1].strip()
                inputs.append(inp)
                outputs.append(out)
                
        return {
            "problem_statement": str(content),
            "input_testcases": inputs,
            "output_testcases": outputs,
            "time_limit": time_limit,
            "memory_limit": memory_limit,
            "stats": {}
        }
    except Exception as e:
        return None

def scrape_codeforces(url):
    try:
        r = requests.get(url, timeout=10, headers={"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"})
        if r.status_code != 200: return None
        soup = BeautifulSoup(r.content, "html.parser")
        
        ps = soup.find("div", class_="problem-statement")
        if not ps: return None
        
        time_limit = ps.find("div", class_="time-limit").get_text() if ps.find("div", class_="time-limit") else ""
        memory_limit = ps.find("div", class_="memory-limit").get_text() if ps.find("div", class_="memory-limit") else ""
        
        inputs = []
        outputs = []
        for inp in ps.find_all("div", class_="input"):
            pre = inp.find("pre")
            if pre: inputs.append(pre.get_text('\n').strip())
            
        for out in ps.find_all("div", class_="output"):
            pre = out.find("pre")
            if pre: outputs.append(pre.get_text('\n').strip())
            
        return {
            "problem_statement": str(ps),
            "input_testcases": inputs,
            "output_testcases": outputs,
            "time_limit": time_limit.replace("time limit per test", "").strip(),
            "memory_limit": memory_limit.replace("memory limit per test", "").strip(),
            "stats": {}
        }
    except Exception as e:
        return None

def scrape_atcoder(url):
    try:
        r = requests.get(url, timeout=10, headers={"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"})
        if r.status_code != 200: return None
        soup = BeautifulSoup(r.content, "html.parser")
        
        ts = soup.find("div", id="task-statement")
        if not ts: return None
        
        # usually limit is in preceding sibling or inside
        text = soup.get_text()
        time_limit = ""
        memory_limit = ""
        import re
        tl_match = re.search(r'Time Limit: (.*? sec)', text)
        ml_match = re.search(r'Memory Limit: (.*? MB)', text)
        if tl_match: time_limit = tl_match.group(1)
        if ml_match: memory_limit = ml_match.group(1)
        
        inputs = []
        outputs = []
        # Sample Input 1 ...
        for pre in ts.find_all("pre"):
            parent = pre.find_parent("section")
            if parent and parent.find("h3"):
                h3_text = parent.find("h3").get_text()
                if "Sample Input" in h3_text:
                    inputs.append(pre.get_text('\n').strip())
                elif "Sample Output" in h3_text:
                    outputs.append(pre.get_text('\n').strip())
                    
        return {
            "problem_statement": str(ts),
            "input_testcases": inputs,
            "output_testcases": outputs,
            "time_limit": time_limit,
            "memory_limit": memory_limit,
            "stats": {}
        }
    except Exception as e:
        return None

def process_chunk(input_file, output_file, source):
    with open(input_file, 'r') as f:
        problems = json.load(f)
        
    def fetch(p):
        url = p.get("url")
        if source == "codeforces":
            data = scrape_codeforces(url)
        elif source == "atcoder":
            data = scrape_atcoder(url)
        elif source == "cses":
            data = scrape_cses(url)
        else:
            data = None
            
        if data:
            p.update(data)
        return p

    print(f"Scraping {len(problems)} problems from {source}...")
    
    # 5 workers to be nice to servers
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
        for res in tqdm(executor.map(fetch, problems), total=len(problems)):
            results.append(res)
            
    with open(output_file, 'w') as f:
        json.dump(results, f, indent=2)

if __name__ == "__main__":
    if len(sys.argv) < 4:
        print("Usage: python universal_scraper.py <input_json> <output_json> <source>")
        sys.exit(1)
    process_chunk(sys.argv[1], sys.argv[2], sys.argv[3])
