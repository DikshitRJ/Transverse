package services

import (
	"testing"
	"transverse/internal/graph"
)

func buildMockTopicGraph(t *testing.T) graph.TopicGraph {
	jsonStr := `{
		"topics": [
			{
				"id": "arrays-hashing",
				"name": "Arrays & Hashing",
				"parent": "",
				"prerequisites": [],
				"tag_aliases": ["array", "hash-table", "hashing", "arrays & hashing"],
				"difficulty_range": [800, 1600],
				"order": 1
			},
			{
				"id": "two-pointers",
				"name": "Two Pointers",
				"parent": "",
				"prerequisites": ["arrays-hashing"],
				"tag_aliases": ["two-pointers", "two pointers", "2-pointers"],
				"difficulty_range": [900, 1800],
				"order": 2
			},
			{
				"id": "sliding-window",
				"name": "Sliding Window",
				"parent": "two-pointers",
				"prerequisites": ["two-pointers"],
				"tag_aliases": ["sliding-window", "sliding window"],
				"difficulty_range": [1000, 1900],
				"order": 3
			}
		]
	}`

	g, err := graph.NewTopicGraphFromJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("failed to create mock topic graph: %v", err)
	}
	return g
}

func TestGraphService_ResolveScope(t *testing.T) {
	tg := buildMockTopicGraph(t)
	gs := NewGraphService(tg)

	input := []string{"array", "Two Pointers", "arrays-hashing"}
	resolved, err := gs.ResolveScope(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should deduplicate and order canonically by Order (1: arrays-hashing, 2: two-pointers)
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved topics, got %d: %+v", len(resolved), resolved)
	}
	if resolved[0] != "arrays-hashing" || resolved[1] != "two-pointers" {
		t.Errorf("expected [arrays-hashing, two-pointers], got %+v", resolved)
	}

	// Test invalid topic error
	_, errInvalid := gs.ResolveScope([]string{"non-existent-topic"})
	if errInvalid == nil {
		t.Error("expected error for invalid topic, got nil")
	}
}

func TestGraphService_GetNextTopic(t *testing.T) {
	tg := buildMockTopicGraph(t)
	gs := NewGraphService(tg)

	// User has mastered nothing -> should recommend first topic ("arrays-hashing")
	next1 := gs.GetNextTopic([]string{})
	if next1 == nil || next1.ID != "arrays-hashing" {
		t.Fatalf("expected next topic to be arrays-hashing, got %+v", next1)
	}

	// User has mastered "arrays-hashing" -> should recommend "two-pointers"
	next2 := gs.GetNextTopic([]string{"arrays-hashing"})
	if next2 == nil || next2.ID != "two-pointers" {
		t.Fatalf("expected next topic to be two-pointers, got %+v", next2)
	}

	// User has mastered all 3 -> should return nil
	nextNone := gs.GetNextTopic([]string{"arrays-hashing", "two-pointers", "sliding-window"})
	if nextNone != nil {
		t.Errorf("expected nil when all mastered, got %+v", nextNone)
	}
}

func TestGraphService_ValidateTopics(t *testing.T) {
	tg := buildMockTopicGraph(t)
	gs := NewGraphService(tg)

	errValid := gs.ValidateTopics([]string{"array", "Two Pointers"})
	if errValid != nil {
		t.Errorf("expected valid topics to pass, got: %v", errValid)
	}

	errInvalid := gs.ValidateTopics([]string{"quantum-algorithms"})
	if errInvalid == nil {
		t.Error("expected invalid topic to return error, got nil")
	}
}
