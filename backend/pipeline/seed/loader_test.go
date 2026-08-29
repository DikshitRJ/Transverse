package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProblems_Deduplication(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "transverse-loader-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfData := `[
		{"id": "cf-1-A", "source": "codeforces", "name": "Theatre Square", "url": "https://codeforces.com/1/A", "difficulty_rating": 1000, "tags": ["math"], "contest_id": "1"},
		{"id": "cf-1-B", "source": "codeforces", "name": "Spreadsheet", "url": "https://codeforces.com/1/B", "difficulty_rating": 1600, "tags": ["math"], "contest_id": "1"}
	]`

	allProblemsData := `[
		{"id": "cf-1-A", "source": "codeforces", "name": "Theatre Square Duplicate", "url": "https://codeforces.com/1/A", "difficulty_rating": 1000, "tags": ["math"], "contest_id": "1"},
		{"id": "cf-2-A", "source": "codeforces", "name": "Winner", "url": "https://codeforces.com/2/A", "difficulty_rating": 1500, "tags": ["hashing"], "contest_id": "2"}
	]`

	if err := os.WriteFile(filepath.Join(tempDir, "codeforces.json"), []byte(cfData), 0644); err != nil {
		t.Fatalf("failed writing cfData: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "all_problems.json"), []byte(allProblemsData), 0644); err != nil {
		t.Fatalf("failed writing allProblemsData: %v", err)
	}

	problems, err := LoadProblems(tempDir)
	if err != nil {
		t.Fatalf("LoadProblems failed: %v", err)
	}

	// Should have exactly 3 unique problems: cf-1-A, cf-1-B, cf-2-A
	if len(problems) != 3 {
		t.Fatalf("expected 3 deduplicated problems, got %d", len(problems))
	}

	// Verify priority: cf-1-A from codeforces.json must precede all_problems.json
	if problems[0].ID != "cf-1-A" || problems[0].Name != "Theatre Square" {
		t.Errorf("expected codeforces.json version of cf-1-A, got: %+v", problems[0])
	}
}
