package services

import (
	"fmt"
	"sort"

	"transverse/internal/graph"
)

// GraphService provides topic-scope resolution using the topic DAG.
type GraphService struct {
	graph graph.TopicGraph
}

// NewGraphService constructs a new GraphService with the provided curriculum topic DAG.
func NewGraphService(g graph.TopicGraph) *GraphService {
	return &GraphService{graph: g}
}

// ResolveScope resolves topic/subtopic names to canonical topic IDs.
// Accepts topic names, IDs, or tag aliases.
// Returns deduplicated, ordered list of canonical topic IDs.
func (gs *GraphService) ResolveScope(topics []string) ([]string, error) {
	if len(topics) == 0 {
		return []string{}, nil
	}

	seen := make(map[string]bool)
	resolved := make([]string, 0, len(topics))

	for _, t := range topics {
		canonicalID, ok := gs.graph.ResolveTag(t)
		if !ok {
			if gs.graph.IsValidTopic(t) {
				canonicalID = t
			} else {
				return nil, fmt.Errorf("services: topic or alias %q not found in topic graph", t)
			}
		}

		if !seen[canonicalID] {
			seen[canonicalID] = true
			resolved = append(resolved, canonicalID)
		}
	}

	// Sort resolved topic IDs by canonical curriculum order
	sort.SliceStable(resolved, func(i, j int) bool {
		nodeI, _ := gs.graph.GetTopic(resolved[i])
		nodeJ, _ := gs.graph.GetTopic(resolved[j])
		if nodeI != nil && nodeJ != nil {
			return nodeI.Order < nodeJ.Order
		}
		return resolved[i] < resolved[j]
	})

	return resolved, nil
}

// GetNextTopic returns the next recommended topic in the prerequisite DAG
// given a set of topics the user has mastered.
func (gs *GraphService) GetNextTopic(masteredTopics []string) *graph.TopicNode {
	masteredSet := make(map[string]bool, len(masteredTopics))
	for _, mt := range masteredTopics {
		if canon, ok := gs.graph.ResolveTag(mt); ok {
			masteredSet[canon] = true
		} else if gs.graph.IsValidTopic(mt) {
			masteredSet[mt] = true
		}
	}

	for _, node := range gs.graph.GetAllTopics() {
		if masteredSet[node.ID] {
			continue
		}

		prereqsMet := true
		for _, prereq := range node.Prerequisites {
			if !masteredSet[prereq] {
				prereqsMet = false
				break
			}
		}

		if prereqsMet {
			nodeCopy, _ := gs.graph.GetTopic(node.ID)
			return nodeCopy
		}
	}

	return nil
}

// ValidateTopics checks all topic strings exist in the graph.
func (gs *GraphService) ValidateTopics(topics []string) error {
	for _, t := range topics {
		if _, ok := gs.graph.ResolveTag(t); !ok {
			if !gs.graph.IsValidTopic(t) {
				return fmt.Errorf("services: invalid topic %q: not found in topic graph", t)
			}
		}
	}
	return nil
}
