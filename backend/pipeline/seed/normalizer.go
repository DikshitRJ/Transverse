// Package main provides problem data normalization for the Transverse knowledge graph and database.
package main

import (
	"fmt"
	"regexp"
	"encoding/json"
	"strings"
	"transverse/internal/graph"
)

// NormalizedProblem represents a sanitized, validated problem record ready for database insertion.
type NormalizedProblem struct {
	ID               string   `json:"id"`
	Source           string   `json:"source"`
	Name             string   `json:"name"`
	URL              string   `json:"url"`
	Slug             string   `json:"slug"`
	ContestID        string   `json:"contest_id"`
	Tags             []string `json:"tags"`
	Topic            string   `json:"topic"`            // Canonical primary topic from graph
	Subtopic         string   `json:"subtopic"`         // Secondary topic or child specialization
	DifficultyLabel  string   `json:"difficulty_label"` // "easy" | "medium" | "hard" | "expert"
	GlickoRating     float64  `json:"glicko_rating"`
	GlickoRD         float64  `json:"glicko_rd"`
	GlickoVolatility float64  `json:"glicko_volatility"`
	EmbedText        string   `json:"embed_text"`
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// Normalize converts a RawProblem into a NormalizedProblem using knowledge graph topic resolution
// and source-specific Glicko psychometric calibration rules.
func Normalize(raw RawProblem, g graph.TopicGraph) NormalizedProblem {
	source := normalizeSource(raw.Source)
	slug := generateSlug(raw.Name, raw.URL, raw.ID)

	cleanTags := make([]string, 0, len(raw.Tags))
	for _, t := range raw.Tags {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			cleanTags = append(cleanTags, trimmed)
		}
	}

	// 1. Resolve primary Topic and secondary Subtopic using the TopicGraph
	primaryTopic, secondaryTopic := resolveTopics(cleanTags, raw.Name, g)

	// 2. Compute Glicko psychometrics and difficulty label
	rating, label := computePsychometrics(source, raw, cleanTags)

	// 3. Construct unified concept embedding string
	tagsStr := strings.Join(cleanTags, " ")
	embedText := fmt.Sprintf("[PROBLEM] %s [TOPIC] %s [SUBTOPIC] %s [TAGS] %s [SOURCE] %s",
		strings.TrimSpace(raw.Name), primaryTopic, secondaryTopic, tagsStr, source)

	ret := NormalizedProblem{
		ID:               raw.ID,
		Source:           source,
		Name:             strings.TrimSpace(raw.Name),
		URL:              strings.TrimSpace(raw.URL),
		Slug:             slug,
		ContestID:        strings.TrimSpace(raw.ContestID),
		Tags:             cleanTags,
		Topic:            primaryTopic,
		Subtopic:         secondaryTopic,
		DifficultyLabel:  label,
		GlickoRating:     rating,
		GlickoRD:         350.0,
				GlickoVolatility: 0.06,
		EmbedText:        embedText,
		Statement:        raw.Statement,
	}
	
	// Convert test cases to JSON
	type tc struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	var tcs []tc
	for i := range raw.InputTestcases {
	    if i < len(raw.OutputTestcases) {
	        tcs = append(tcs, tc{Input: raw.InputTestcases[i], Output: raw.OutputTestcases[i]})
	    }
	}
	if b, err := json.Marshal(tcs); err == nil {
	    ret.TestCases = b
	} else {
	    ret.TestCases = []byte("[]")
	}
	
	return ret
}

// normalizeSource maps source identifiers to canonical platform names.
func normalizeSource(src string) string {
	s := strings.ToLower(strings.TrimSpace(src))
	switch {
	case strings.Contains(s, "codeforces") || strings.HasPrefix(s, "cf"):
		return "codeforces"
	case strings.Contains(s, "atcoder") || strings.HasPrefix(s, "at"):
		return "atcoder"
	case strings.Contains(s, "leetcode"):
		return "leetcode"
	case strings.Contains(s, "cses"):
		return "cses"
	default:
		if s == "" {
			return "general"
		}
		return s
	}
}

// generateSlug extracts or creates a URL-friendly slug from name, URL, or ID.
func generateSlug(name, rawURL, id string) string {
	if rawURL != "" {
		parts := strings.Split(strings.Trim(rawURL, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != "" && !strings.Contains(last, "?") {
				return strings.ToLower(last)
			}
		}
	}

	target := name
	if strings.TrimSpace(target) == "" {
		target = id
	}

	slug := nonAlphanumericRegex.ReplaceAllString(strings.ToLower(target), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "problem-" + id
	}
	return slug
}

// resolveTopics maps raw tags and name semantics to canonical topics in the knowledge graph.
func resolveTopics(tags []string, name string, g graph.TopicGraph) (string, string) {
	var resolved []string
	seen := make(map[string]bool)

	if g != nil {
		for _, tag := range tags {
			if topicID, ok := g.ResolveTag(tag); ok {
				if !seen[topicID] {
					seen[topicID] = true
					resolved = append(resolved, topicID)
				}
			}
		}
	}

	if len(resolved) == 0 {
		// Fallback heuristic: check name for topic keywords
		lowerName := strings.ToLower(name)
		nameKeywords := map[string]string{
			"tree":       "trees",
			"bst":        "trees",
			"graph":      "graphs",
			"path":       "shortest-paths",
			"binary":     "binary-search",
			"search":     "binary-search",
			"dp":         "dynamic-programming",
			"array":      "arrays-hashing",
			"hash":       "arrays-hashing",
			"string":     "string-algorithms",
			"prime":      "math-number-theory",
			"math":       "math-number-theory",
			"game":       "game-theory",
			"query":      "range-queries",
			"segment":    "range-queries",
			"stack":      "stack-queues",
			"queue":      "stack-queues",
			"matrix":     "matrices-fft",
			"flow":       "network-flows",
			"bit":        "bit-manipulation",
			"sort":       "sorting-searching",
			"two sum":    "arrays-hashing",
			"anagram":    "arrays-hashing",
			"palindrome": "two-pointers",
			"subsets":    "backtracking",
		}

		for kw, topicID := range nameKeywords {
			if strings.Contains(lowerName, kw) {
				if !seen[topicID] {
					seen[topicID] = true
					resolved = append(resolved, topicID)
					break
				}
			}
		}
	}

	primary := "foundations"
	secondary := "general"

	if len(resolved) >= 1 {
		primary = resolved[0]
		if len(resolved) >= 2 {
			secondary = resolved[1]
		} else if g != nil {
			// Pick direct child if available, else primary
			children := g.GetChildren(primary)
			if len(children) > 0 {
				secondary = children[0].ID
			} else {
				secondary = primary
			}
		}
	}

	return primary, secondary
}

// computePsychometrics derives the initial Glicko-2 rating and difficulty classification label.
func computePsychometrics(source string, raw RawProblem, tags []string) (float64, string) {
	var rating float64
	var explicitLabel string

	switch source {
	case "codeforces":
		if raw.DifficultyRating != nil && *raw.DifficultyRating > 0 {
			rating = float64(*raw.DifficultyRating)
		} else {
			rating = 1200.0
		}
		if rating < 800 {
			rating = 800
		} else if rating > 3500 {
			rating = 3500
		}

	case "atcoder":
		if raw.DifficultyRating != nil && *raw.DifficultyRating > 0 {
			r := float64(*raw.DifficultyRating)
			// Normalize from AtCoder 0-4000 scale to standard 800-2800 Glicko scale
			rating = 800.0 + (r / 4000.0) * (2800.0 - 800.0)
		} else {
			rating = 1200.0
		}
		if rating < 800 {
			rating = 800
		} else if rating > 2800 {
			rating = 2800
		}

	case "leetcode":
		// Inspect tags for explicit difficulty tier
		for _, t := range tags {
			lower := strings.ToLower(t)
			if lower == "easy" {
				explicitLabel = "easy"
				rating = 900.0
				break
			} else if lower == "medium" {
				explicitLabel = "medium"
				rating = 1400.0
				break
			} else if lower == "hard" {
				explicitLabel = "hard"
				rating = 1800.0
				break
			}
		}

		if explicitLabel == "" {
			// Check known common LeetCode problem difficulty
			lowerName := strings.ToLower(raw.Name)
			switch {
			case containsAny(lowerName, "two sum", "valid anagram", "contains duplicate", "invert binary tree", "maximum depth", "climbing stairs", "reverse linked list", "binary search", "valid palindrome"):
				explicitLabel = "easy"
				rating = 900.0
			case containsAny(lowerName, "trapping rain water", "median of two sorted", "merge k sorted", "largest rectangle in histogram", "n-queens", "minimum window substring", "word ladder"):
				explicitLabel = "hard"
				rating = 1800.0
			default:
				explicitLabel = "medium"
				rating = 1400.0
			}
		}

	case "cses":
		if raw.DifficultyRating != nil && *raw.DifficultyRating > 0 {
			rating = float64(*raw.DifficultyRating)
		} else {
			rating = 1400.0
		}

	default:
		if raw.DifficultyRating != nil && *raw.DifficultyRating > 0 {
			rating = float64(*raw.DifficultyRating)
		} else {
			rating = 1400.0
		}
	}

	label := explicitLabel
	if label == "" {
		switch {
		case rating < 1200:
			label = "easy"
		case rating < 1700:
			label = "medium"
		case rating < 2200:
			label = "hard"
		default:
			label = "expert"
		}
	}

	return rating, label
}

// containsAny checks if the haystack contains any of the needle substrings.
func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}
