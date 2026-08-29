package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTopicGraph(t *testing.T) {
	// Create temporary topics JSON
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "topics.json")

	content := `{
		"topics": [
			{
				"id": "arrays-hashing",
				"name": "Arrays & Hashing",
				"parent": "foundations",
				"prerequisites": [],
				"tag_aliases": ["array", "hash-table"],
				"difficulty_range": [800, 1600],
				"order": 1
			},
			{
				"id": "two-pointers",
				"name": "Two Pointers",
				"parent": "search-sort",
				"prerequisites": ["arrays-hashing"],
				"tag_aliases": ["2-pointers"],
				"difficulty_range": [900, 1700],
				"order": 2
			}
		]
	}`

	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	tg, err := Load(jsonPath)
	if err != nil {
		t.Fatalf("failed to load graph: %v", err)
	}

	if len(tg.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(tg.Nodes))
	}
	if len(tg.Ordered) != 2 {
		t.Errorf("expected 2 ordered nodes, got %d", len(tg.Ordered))
	}

	// Test aliases
	if tg.AliasMap["array"] != "arrays-hashing" {
		t.Errorf("expected alias 'array' -> 'arrays-hashing', got %q", tg.AliasMap["array"])
	}
	if tg.AliasMap["arrays & hashing"] != "arrays-hashing" {
		t.Errorf("expected alias 'arrays & hashing' -> 'arrays-hashing', got %q", tg.AliasMap["arrays & hashing"])
	}
	if tg.AliasMap["2-pointers"] != "two-pointers" {
		t.Errorf("expected alias '2-pointers' -> 'two-pointers', got %q", tg.AliasMap["2-pointers"])
	}
}
