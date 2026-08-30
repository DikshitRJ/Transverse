import json
import os
import concurrent.futures
import time
import requests
import pandas as pd
from leetscrape import GetQuestion
from tqdm import tqdm

DATA_DIR = "data"

print("Fetching question list from api/problems/all...")
HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
}
req = requests.get("https://leetcode.com/api/problems/all/", headers=HEADERS).json()
questions_df = pd.json_normalize(req["stat_status_pairs"])
slugs = questions_df["stat.question__title_slug"].tolist()
slugs = [s for s in slugs if pd.notnull(s)]

# Cache the questions info so it's not fetched 4000 times
print("Caching question info for GetQuestion...")
question_data = questions_df.rename(
    columns={
        "stat.frontend_question_id": "QID",
        "stat.question__title_slug": "titleSlug",
    }
)[["QID", "titleSlug"]]
_cached_questions_info = question_data.sort_values("QID").set_index("titleSlug")

# Monkey patch GetQuestion to use cached info
original_init = GetQuestion.__init__
def new_init(self, titleSlug: str):
    self.titleSlug = titleSlug
    self.questions_info = _cached_questions_info
GetQuestion.__init__ = new_init

def scrape_question(slug):
    file_path = os.path.join(DATA_DIR, f"{slug}.json")
    if os.path.exists(file_path):
        return
    try:
        q = GetQuestion(titleSlug=slug).scrape()
        data = {
            "QID": q.QID,
            "title": q.title,
            "titleSlug": q.titleSlug,
            "difficulty": q.difficulty,
            "hints": q.Hints,
            "topics": q.topics,
            "isPaidOnly": q.isPaidOnly,
            "body": q.Body,
            "code": q.Code,
        }
        with open(file_path, "w") as f:
            json.dump(data, f)
    except Exception as e:
        # ignore or print
        pass

def main():
    print(f"Found {len(slugs)} questions. Starting to scrape...")
    # 5 workers to reduce probability of 429 errors
    with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
        list(tqdm(executor.map(scrape_question, slugs), total=len(slugs)))

if __name__ == "__main__":
    main()
