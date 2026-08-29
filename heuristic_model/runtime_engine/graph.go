package graph

import (
	"encoding/json"
	"fmt"
	"os"
)

type SyllabusGraph map[string]map[string]*ChapterNode

type ChapterNode struct {
	ID            string   `json:"id"`      // set from map key (not in JSON)
	Title         string   `json:"chapter"` // maps JSON "chapter" → display name
	ChapterGroup  string   `json:"group"`   // maps JSON "group" → e.g. "mechanics"
	Subject       string   `json:"subject"` // set from outer map key (not in JSON)
	Order         int      `json:"-"`       // unused, excluded from JSON
	Prerequisites []string `json:"prerequisites,omitempty"`
}

func Load(path string) (SyllabusGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("graph: read file: %w", err)
	}

	var raw map[string]map[string]*ChapterNode
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("graph: unmarshal: %w", err)
	}

	for subject, chapters := range raw {
		for id, node := range chapters {
			if node.Prerequisites == nil {
				node.Prerequisites = []string{}
			}
			node.ID = id
			node.Subject = subject
		}
	}

	return SyllabusGraph(raw), nil
}
