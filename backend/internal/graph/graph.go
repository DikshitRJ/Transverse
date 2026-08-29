// Package graph defines the DSA/CP topic knowledge graph, node representations, and DAG operations.
package graph

// TopicNode represents a node in the DSA/CP knowledge graph.
type TopicNode struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Parent          string   `json:"parent,omitempty"`
	Prerequisites   []string `json:"prerequisites"`
	TagAliases      []string `json:"tag_aliases"`
	DifficultyRange [2]int   `json:"difficulty_range"`
	Order           int      `json:"order"`
}

// TopicGraph is the in-memory topic DAG interface.
type TopicGraph interface {
	// GetTopic returns a topic by ID.
	GetTopic(id string) (*TopicNode, bool)
	// GetAllTopics returns all topic nodes.
	GetAllTopics() []TopicNode
	// ResolveTag maps a raw tag (e.g., "dp", "dynamic programming") to a topic ID.
	ResolveTag(tag string) (string, bool)
	// GetPrerequisites returns all prerequisite topic IDs for a given topic (recursive).
	GetPrerequisites(topicID string) []string
	// GetChildren returns direct children of a topic.
	GetChildren(topicID string) []TopicNode
	// IsValidTopic checks if a topic ID exists.
	IsValidTopic(id string) bool
}
