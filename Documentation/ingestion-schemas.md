# Ingestion Schemas

These are the expected NDJSON formats for offline scraping or direct submission to the backend API (`/api/v1/admin/...`).

### Tutorial Record Schema
```json
{
  "source": "neetcode | cp-algorithms | gfg | striver-a2z | other",
  "source_url": "https://...",
  "title": "string",
  "topic_tags": ["graphs", "bfs"],
  "type": "article | video | interactive | playlist",
  "difficulty": "beginner | intermediate | advanced",
  "estimated_minutes": 15,
  "summary": "short, original 2-3 sentence description — do not copy source text",
  "author": "string, optional",
  "thumbnail_url": "string, optional",
  "license_note": "e.g. 'linked externally, not mirrored'"
}
```

### Curated Roadmap Template Schema
```json
{
  "roadmap_name": "string",
  "target_role": "DSA Mastery | SDE Interview Prep | Competitive Programmer",
  "phases": [
    {
      "title": "string",
      "sequence": 1,
      "unlock_rule": { "type": "no_prerequisite" },
      "nodes": [
        {
          "topic_tag": "arrays",
          "sequence": 1,
          "tutorial_source_urls": ["https://..."],
          "practice_topic_tags": ["arrays"]
        }
      ]
    }
  ]
}
```
