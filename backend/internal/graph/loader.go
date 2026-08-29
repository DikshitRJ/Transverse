// Package graph provides loader utilities and graph traversal implementations for DSA/CP topics.
package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// topicsContainer wraps the JSON topics array payload.
type topicsContainer struct {
	Topics []TopicNode `json:"topics"`
}

// topicGraphImpl is the concrete in-memory implementation of TopicGraph.
type topicGraphImpl struct {
	topics        map[string]*TopicNode
	orderedTopics []TopicNode
	tagToTopic    map[string]string
	children      map[string][]*TopicNode
}

// NewTopicGraph loads a topics.json file from the specified path and constructs an in-memory TopicGraph.
func NewTopicGraph(jsonPath string) (TopicGraph, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read topics file from %q: %w", jsonPath, err)
	}

	return NewTopicGraphFromJSON(data)
}

// NewTopicGraphFromJSON parses JSON bytes and initializes the in-memory TopicGraph DAG.
func NewTopicGraphFromJSON(data []byte) (TopicGraph, error) {
	var container topicsContainer
	if err := json.Unmarshal(data, &container); err != nil {
		// Fallback: try parsing as a direct array of TopicNode
		var directList []TopicNode
		if errDirect := json.Unmarshal(data, &directList); errDirect != nil {
			return nil, fmt.Errorf("failed to unmarshal topics json: %w (direct parse error: %v)", err, errDirect)
		}
		container.Topics = directList
	}

	g := &topicGraphImpl{
		topics:        make(map[string]*TopicNode, len(container.Topics)),
		orderedTopics: make([]TopicNode, 0, len(container.Topics)),
		tagToTopic:    make(map[string]string),
		children:      make(map[string][]*TopicNode),
	}

	// First pass: store nodes and sort order
	for i := range container.Topics {
		node := container.Topics[i]
		if node.Prerequisites == nil {
			node.Prerequisites = []string{}
		}
		if node.TagAliases == nil {
			node.TagAliases = []string{}
		}

		nodeCopy := node
		g.topics[node.ID] = &nodeCopy
		g.orderedTopics = append(g.orderedTopics, nodeCopy)

		// Self aliases: map ID and Name
		g.tagToTopic[strings.ToLower(strings.TrimSpace(node.ID))] = node.ID
		g.tagToTopic[strings.ToLower(strings.TrimSpace(node.Name))] = node.ID

		// Map each alias to the topic ID
		for _, alias := range node.TagAliases {
			normalized := normalizeTag(alias)
			if normalized != "" {
				g.tagToTopic[normalized] = node.ID
			}
		}
	}

	// Sort orderedTopics by Order ascending
	sort.SliceStable(g.orderedTopics, func(i, j int) bool {
		return g.orderedTopics[i].Order < g.orderedTopics[j].Order
	})

	// Second pass: build parent -> children relations
	for i := range g.orderedTopics {
		node := &g.orderedTopics[i]
		if node.Parent != "" {
			g.children[node.Parent] = append(g.children[node.Parent], node)
		}
	}

	return g, nil
}

// GetTopic returns a topic by ID with case-insensitive fallback.
func (g *topicGraphImpl) GetTopic(id string) (*TopicNode, bool) {
	if g == nil || g.topics == nil {
		return nil, false
	}

	if node, ok := g.topics[id]; ok {
		return node, true
	}

	// Try lowercase lookup
	normalized := strings.ToLower(strings.TrimSpace(id))
	if topicID, ok := g.tagToTopic[normalized]; ok {
		if node, ok := g.topics[topicID]; ok {
			return node, true
		}
	}

	return nil, false
}

// GetAllTopics returns all topic nodes ordered by their sequence order.
func (g *topicGraphImpl) GetAllTopics() []TopicNode {
	if g == nil {
		return []TopicNode{}
	}
	result := make([]TopicNode, len(g.orderedTopics))
	copy(result, g.orderedTopics)
	return result
}

// ResolveTag maps a raw tag (e.g. "dp", "dynamic programming") to its canonical topic ID.
func (g *topicGraphImpl) ResolveTag(tag string) (string, bool) {
	if g == nil || g.tagToTopic == nil {
		return "", false
	}

	norm := normalizeTag(tag)
	if norm == "" {
		return "", false
	}

	if topicID, ok := g.tagToTopic[norm]; ok {
		return topicID, true
	}

	// Try substituting hyphens and underscores with spaces
	alt1 := strings.ReplaceAll(norm, "-", " ")
	if topicID, ok := g.tagToTopic[alt1]; ok {
		return topicID, true
	}

	// Try substituting spaces with hyphens
	alt2 := strings.ReplaceAll(norm, " ", "-")
	if topicID, ok := g.tagToTopic[alt2]; ok {
		return topicID, true
	}

	return "", false
}

// GetPrerequisites returns all prerequisite topic IDs for a given topic (recursive DAG traversal).
func (g *topicGraphImpl) GetPrerequisites(topicID string) []string {
	if g == nil {
		return []string{}
	}

	visited := make(map[string]bool)
	var prereqs []string

	var dfs func(currID string)
	dfs = func(currID string) {
		node, ok := g.GetTopic(currID)
		if !ok || node == nil {
			return
		}

		for _, prereqID := range node.Prerequisites {
			if !visited[prereqID] {
				visited[prereqID] = true
				dfs(prereqID)
				prereqs = append(prereqs, prereqID)
			}
		}
	}

	dfs(topicID)
	return prereqs
}

// GetChildren returns direct children of a topic.
func (g *topicGraphImpl) GetChildren(topicID string) []TopicNode {
	if g == nil || g.children == nil {
		return []TopicNode{}
	}

	childrenPtrs, ok := g.children[topicID]
	if !ok || len(childrenPtrs) == 0 {
		return []TopicNode{}
	}

	children := make([]TopicNode, len(childrenPtrs))
	for i, cp := range childrenPtrs {
		children[i] = *cp
	}
	return children
}

// IsValidTopic checks if a topic ID exists in the knowledge graph.
func (g *topicGraphImpl) IsValidTopic(id string) bool {
	_, ok := g.GetTopic(id)
	return ok
}

// normalizeTag cleans and normalizes a tag string for consistent map lookups.
func normalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}
