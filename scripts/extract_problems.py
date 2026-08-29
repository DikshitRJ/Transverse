#!/usr/bin/env python3
"""Build Transverse's ToS-conscious competitive-programming problem corpus.

Network sources used by this script:
  * Codeforces' official public API (one batch request)
  * CSES's public problem index (names, sections, and links only)
  * Kenkoooo's public AtCoder Problems API (metadata and difficulty)

It deliberately does not request LeetCode (or any other proprietary problem
page).  The LeetCode output is built only from a checked-in, manually curated
index containing titles, slugs, links, and Transverse topic tags.
"""

from __future__ import annotations

import argparse
import json
import logging
import os
import re
import sys
import tempfile
import time
from html.parser import HTMLParser
from pathlib import Path
from typing import Any, Iterable
from urllib.error import HTTPError, URLError
from urllib.parse import urljoin
from urllib.request import Request, urlopen
from urllib.robotparser import RobotFileParser


CODEFORCES_URL = "https://codeforces.com/api/problemset.problems"
CSES_LIST_URL = "https://cses.fi/problemset/list/"
ATCODER_PROBLEMS_URL = "https://kenkoooo.com/atcoder/resources/problems.json"
ATCODER_MODELS_URL = "https://kenkoooo.com/atcoder/resources/problem-models.json"
USER_AGENT = "TransverseProblemIndexer/1.0 (metadata-only; contact: team@transverse.local)"
TASK_PATH = re.compile(r"^/problemset/task/(\d+)/?$")


class FetchError(RuntimeError):
    """Raised when a public source cannot be fetched or decoded safely."""


def fetch_bytes(url: str, timeout: int) -> bytes:
    request = Request(url, headers={"User-Agent": USER_AGENT, "Accept": "application/json, text/html;q=0.9, */*;q=0.1"})
    try:
        with urlopen(request, timeout=timeout) as response:
            if response.status != 200:
                raise FetchError(f"{url} returned HTTP {response.status}")
            return response.read()
    except HTTPError as exc:
        raise FetchError(f"{url} returned HTTP {exc.code}") from exc
    except URLError as exc:
        raise FetchError(f"Could not reach {url}: {exc.reason}") from exc
    except TimeoutError as exc:
        raise FetchError(f"Timed out while fetching {url}") from exc


def fetch_json(url: str, timeout: int) -> Any:
    try:
        return json.loads(fetch_bytes(url, timeout).decode("utf-8"))
    except json.JSONDecodeError as exc:
        raise FetchError(f"{url} did not return valid JSON") from exc


class CsesProblemListParser(HTMLParser):
    """Extract task anchors and their nearest CSES h2 section heading."""

    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.current_section = "Uncategorized"
        self._in_h2 = False
        self._heading_parts: list[str] = []
        self._active_href: str | None = None
        self._anchor_parts: list[str] = []
        self.problems: list[tuple[str, str, str]] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag == "h2":
            self._in_h2 = True
            self._heading_parts = []
        elif tag == "a":
            href = dict(attrs).get("href")
            if href and TASK_PATH.match(href):
                self._active_href = href
                self._anchor_parts = []

    def handle_data(self, data: str) -> None:
        if self._in_h2:
            self._heading_parts.append(data)
        if self._active_href is not None:
            self._anchor_parts.append(data)

    def handle_endtag(self, tag: str) -> None:
        if tag == "h2":
            heading = " ".join("".join(self._heading_parts).split())
            if heading:
                self.current_section = heading
            self._in_h2 = False
        elif tag == "a" and self._active_href is not None:
            name = " ".join("".join(self._anchor_parts).split())
            match = TASK_PATH.match(self._active_href)
            if name and match:
                self.problems.append((match.group(1), name, self.current_section))
            self._active_href = None
            self._anchor_parts = []


def as_problem(
    *,
    identifier: str,
    source: str,
    name: str,
    url: str,
    difficulty_rating: int | float | None,
    tags: Iterable[str],
    contest_id: str | None,
) -> dict[str, Any]:
    """Return exactly the shared problem schema used by all output files."""
    return {
        "id": identifier,
        "source": source,
        "name": name,
        "url": url,
        "difficulty_rating": difficulty_rating,
        "tags": list(tags),
        "contest_id": contest_id,
        "notes": None,
    }


def codeforces_problems(timeout: int) -> list[dict[str, Any]]:
    """Fetch Codeforces once. This endpoint returns the complete problem set."""
    payload = fetch_json(CODEFORCES_URL, timeout)
    if payload.get("status") != "OK":
        raise FetchError(f"Codeforces API returned {payload.get('status', 'an unknown status')}")

    items: list[dict[str, Any]] = []
    for problem in payload.get("result", {}).get("problems", []):
        contest_id = problem.get("contestId")
        index = problem.get("index")
        name = problem.get("name")
        if contest_id is None or not index or not name:
            logging.warning("Skipping malformed Codeforces problem: %r", problem)
            continue
        contest = str(contest_id)
        problem_index = str(index)
        items.append(
            as_problem(
                identifier=f"cf-{contest}-{problem_index}",
                source="codeforces",
                name=name,
                url=f"https://codeforces.com/problemset/problem/{contest}/{problem_index}",
                difficulty_rating=problem.get("rating"),
                tags=problem.get("tags", []),
                contest_id=contest,
            )
        )
    return unique_by_id(items)


def cses_allows_index_fetch(timeout: int) -> bool:
    """Honor CSES crawl guidance before requesting its HTML index."""
    robots_url = urljoin(CSES_LIST_URL, "/robots.txt")
    parser = RobotFileParser()
    try:
        parser.parse(fetch_bytes(robots_url, timeout).decode("utf-8", errors="replace").splitlines())
    except FetchError as exc:
        # A missing robots.txt is not a prohibition. Under the conventional
        # robots policy it is equivalent to no crawl restrictions; any other
        # retrieval failure remains fail-closed so we do not guess.
        if "HTTP 404" in str(exc):
            logging.info("CSES has no robots.txt; proceeding with its public problem index.")
            return True
        logging.warning("Could not verify CSES robots.txt; skipping CSES: %s", exc)
        return False
    allowed = parser.can_fetch(USER_AGENT, CSES_LIST_URL)
    if not allowed:
        logging.warning("CSES robots.txt does not permit this index fetch; skipping CSES.")
    return allowed


def cses_problems(timeout: int) -> list[dict[str, Any]]:
    if not cses_allows_index_fetch(timeout):
        return []
    parser = CsesProblemListParser()
    parser.feed(fetch_bytes(CSES_LIST_URL, timeout).decode("utf-8", errors="replace"))
    parser.close()
    return unique_by_id(
        as_problem(
            identifier=f"cses-{task_id}",
            source="cses",
            name=name,
            url=urljoin(CSES_LIST_URL, f"/problemset/task/{task_id}"),
            difficulty_rating=None,
            tags=[section],
            contest_id=task_id,
        )
        for task_id, name, section in parser.problems
    )


def atcoder_problems(timeout: int) -> list[dict[str, Any]]:
    # The two public resources are fetched separately because models are keyed
    # by problem id. Neither endpoint carries problem-statement text.
    problems = fetch_json(ATCODER_PROBLEMS_URL, timeout)
    models = fetch_json(ATCODER_MODELS_URL, timeout)
    if not isinstance(problems, list) or not isinstance(models, dict):
        raise FetchError("AtCoder Problems API returned an unexpected data shape")

    items: list[dict[str, Any]] = []
    for problem in problems:
        problem_id = problem.get("id")
        contest_id = problem.get("contest_id")
        name = problem.get("title") or problem.get("name")
        if not problem_id or not contest_id or not name:
            logging.warning("Skipping malformed AtCoder problem: %r", problem)
            continue
        model = models.get(problem_id, {})
        difficulty = model.get("difficulty") if isinstance(model, dict) else None
        items.append(
            as_problem(
                identifier=f"atcoder-{problem_id}",
                source="atcoder",
                name=name,
                url=f"https://atcoder.jp/contests/{contest_id}/tasks/{problem_id}",
                difficulty_rating=difficulty if isinstance(difficulty, (int, float)) else None,
                tags=[],  # This API intentionally has no authoritative topic taxonomy.
                contest_id=str(contest_id),
            )
        )
    return unique_by_id(items)


def leetcode_index(index_path: Path) -> list[dict[str, Any]]:
    """Load local, manually curated links; deliberately make no network request."""
    try:
        entries = json.loads(index_path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise FetchError(f"LeetCode index file is missing: {index_path}") from exc
    except json.JSONDecodeError as exc:
        raise FetchError(f"LeetCode index file is invalid JSON: {index_path}") from exc
    if not isinstance(entries, list):
        raise FetchError("LeetCode index must be a JSON array")

    items: list[dict[str, Any]] = []
    for entry in entries:
        slug, name, tags = entry.get("slug"), entry.get("name"), entry.get("tags")
        if not isinstance(slug, str) or not isinstance(name, str) or not isinstance(tags, list):
            logging.warning("Skipping malformed local LeetCode index entry: %r", entry)
            continue
        items.append(
            as_problem(
                identifier=f"leetcode-{slug}",
                source="leetcode-index",
                name=name,
                url=f"https://leetcode.com/problems/{slug}/",
                difficulty_rating=None,
                tags=tags,
                contest_id=slug,
            )
        )
    return unique_by_id(items)


def unique_by_id(items: Iterable[dict[str, Any]]) -> list[dict[str, Any]]:
    unique: dict[str, dict[str, Any]] = {}
    for item in items:
        if item["id"] in unique:
            logging.warning("Duplicate id ignored: %s", item["id"])
            continue
        unique[item["id"]] = item
    return sorted(unique.values(), key=lambda item: item["id"])


def write_json(path: Path, payload: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", dir=path.parent, delete=False) as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=2)
        handle.write("\n")
        temporary_path = Path(handle.name)
    os.replace(temporary_path, path)


def parse_args() -> argparse.Namespace:
    repository_root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description="Extract public problem metadata for Transverse.")
    parser.add_argument("--output-dir", type=Path, default=repository_root / "data" / "generated")
    parser.add_argument("--leetcode-index", type=Path, default=repository_root / "data" / "leetcode_index_seed.json")
    parser.add_argument("--timeout", type=int, default=30, help="HTTP timeout in seconds (default: 30)")
    parser.add_argument("--skip-cses", action="store_true", help="Do not fetch the CSES public index")
    parser.add_argument("--skip-network", action="store_true", help="Only generate leetcode_index.json from local metadata")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")
    source_items: dict[str, list[dict[str, Any]]] = {"leetcode_index": leetcode_index(args.leetcode_index)}

    if not args.skip_network:
        # Codeforces requests are deliberately limited to one batch request.
        try:
            source_items["codeforces"] = codeforces_problems(args.timeout)
        except FetchError as exc:
            logging.error("Codeforces extraction failed: %s", exc)
            source_items["codeforces"] = []

        try:
            source_items["atcoder"] = atcoder_problems(args.timeout)
        except FetchError as exc:
            logging.error("AtCoder extraction failed: %s", exc)
            source_items["atcoder"] = []

        if not args.skip_cses:
            try:
                source_items["cses"] = cses_problems(args.timeout)
            except FetchError as exc:
                logging.error("CSES extraction failed: %s", exc)
                source_items["cses"] = []
        else:
            source_items["cses"] = []
    else:
        source_items.update({"codeforces": [], "atcoder": [], "cses": []})

    output_names = {"codeforces": "codeforces.json", "cses": "cses.json", "atcoder": "atcoder.json", "leetcode_index": "leetcode_index.json"}
    for source, filename in output_names.items():
        write_json(args.output_dir / filename, source_items[source])
        logging.info("%s: %d problems -> %s", source, len(source_items[source]), args.output_dir / filename)

    merged = unique_by_id(item for items in source_items.values() for item in items)
    write_json(args.output_dir / "all_problems.json", merged)
    logging.info("all sources: %d problems -> %s", len(merged), args.output_dir / "all_problems.json")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except FetchError as exc:
        logging.error("Extraction aborted: %s", exc)
        raise SystemExit(1)
