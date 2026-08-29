package main

import (
	"strings"
	"testing"
	"transverse/internal/graph"
)

func createTestGraph(t *testing.T) graph.TopicGraph {
	jsonStr := `{
		"topics": [
			{"id": "foundations", "name": "Foundations", "parent": "", "prerequisites": [], "tag_aliases": ["implementation", "math"], "difficulty_range": [800, 1600], "order": 1},
			{"id": "arrays-hashing", "name": "Arrays & Hashing", "parent": "foundations", "prerequisites": [], "tag_aliases": ["array", "hash table", "hash map", "arrays & hashing"], "difficulty_range": [800, 1600], "order": 2},
			{"id": "dynamic-programming", "name": "Dynamic Programming", "parent": "foundations", "prerequisites": ["arrays-hashing"], "tag_aliases": ["dp", "dynamic programming"], "difficulty_range": [1100, 2800], "order": 3},
			{"id": "two-pointers", "name": "Two Pointers", "parent": "foundations", "prerequisites": ["arrays-hashing"], "tag_aliases": ["two pointers", "2-pointers"], "difficulty_range": [900, 1800], "order": 4}
		]
	}`
	g, err := graph.NewTopicGraphFromJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("failed creating test graph: %v", err)
	}
	return g
}

func TestNormalize_Codeforces(t *testing.T) {
	g := createTestGraph(t)
	rating := 1600
	raw := RawProblem{
		ID:               "cf-1-B",
		Source:           "codeforces",
		Name:             "Spreadsheet",
		URL:              "https://codeforces.com/problemset/problem/1/B",
		DifficultyRating: &rating,
		Tags:             []string{"implementation", "math"},
		ContestID:        "1",
	}

	norm := Normalize(raw, g)

	if norm.Source != "codeforces" {
		t.Errorf("expected source 'codeforces', got %s", norm.Source)
	}
	if norm.GlickoRating != 1600.0 {
		t.Errorf("expected glicko rating 1600, got %f", norm.GlickoRating)
	}
	if norm.DifficultyLabel != "medium" {
		t.Errorf("expected difficulty label 'medium', got %s", norm.DifficultyLabel)
	}
	if norm.Topic != "foundations" {
		t.Errorf("expected primary topic 'foundations', got %s", norm.Topic)
	}
	if !strings.HasPrefix(norm.EmbedText, "[PROBLEM] Spreadsheet") {
		t.Errorf("invalid embed text: %s", norm.EmbedText)
	}
}

func TestNormalize_AtCoder(t *testing.T) {
	g := createTestGraph(t)
	rating := 2000 // 800 + 2000 * 0.5 = 1800
	raw := RawProblem{
		ID:               "atcoder-abc100_d",
		Source:           "atcoder",
		Name:             "D. Patisserie ABC",
		URL:              "https://atcoder.jp/contests/abc100/tasks/abc100_d",
		DifficultyRating: &rating,
		Tags:             []string{"dp"},
		ContestID:        "abc100",
	}

	norm := Normalize(raw, g)

	expectedRating := 1800.0
	if norm.GlickoRating != expectedRating {
		t.Errorf("expected atcoder calibrated rating %f, got %f", expectedRating, norm.GlickoRating)
	}
	if norm.DifficultyLabel != "hard" {
		t.Errorf("expected difficulty label 'hard', got %s", norm.DifficultyLabel)
	}
	if norm.Topic != "dynamic-programming" {
		t.Errorf("expected topic 'dynamic-programming', got %s", norm.Topic)
	}
}

func TestNormalize_LeetCode(t *testing.T) {
	g := createTestGraph(t)
	raw := RawProblem{
		ID:        "leetcode-3sum",
		Source:    "leetcode-index",
		Name:      "3Sum",
		URL:       "https://leetcode.com/problems/3sum/",
		Tags:      []string{"Two Pointers", "Sorting", "Medium"},
		ContestID: "3sum",
	}

	norm := Normalize(raw, g)

	if norm.Source != "leetcode" {
		t.Errorf("expected source 'leetcode', got %s", norm.Source)
	}
	if norm.GlickoRating != 1400.0 {
		t.Errorf("expected glicko rating 1400 for medium, got %f", norm.GlickoRating)
	}
	if norm.DifficultyLabel != "medium" {
		t.Errorf("expected difficulty label 'medium', got %s", norm.DifficultyLabel)
	}
	if norm.Topic != "two-pointers" {
		t.Errorf("expected topic 'two-pointers', got %s", norm.Topic)
	}
}
