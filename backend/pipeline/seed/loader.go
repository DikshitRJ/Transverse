package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// RawProblem represents an unnormalized problem record parsed from raw JSON dump files.
type RawProblem struct {
	ID               string      `json:"id"`
	Source           string      `json:"source"`
	Name             string      `json:"name"`
	URL              string      `json:"url"`
	DifficultyRating *int        `json:"difficulty_rating"` // nullable integer
	Tags             []string    `json:"tags"`
	ContestID        string      `json:"contest_id"`
	Notes            *string     `json:"notes"`
	Statement        string      `json:"problem_statement"`
	InputTestcases   []string    `json:"input_testcases"`
	OutputTestcases  []string    `json:"output_testcases"`
	TimeLimit        interface{} `json:"time_limit"`
	MemoryLimit      interface{} `json:"memory_limit"`

	// Leetcode specific
	QID               interface{}   `json:"QID"`
	Title             string        `json:"title"`
	TitleSlug         string        `json:"titleSlug"`
	Difficulty        string        `json:"difficulty"`
	RawTopics         []interface{} `json:"topics"`
	Topics            []string      `json:"-"`
	Body              string        `json:"body"`
	InputTestcasesLC  []string      `json:"input_test_cases"`
	OutputTestcasesLC []string      `json:"output_test_cases"`
}

func LoadProblems(dataDir string) ([]RawProblem, error) {
	if _, err := os.Stat(dataDir); err != nil {
		return nil, fmt.Errorf("specified data directory %q does not exist: %w", dataDir, err)
	}

	priorityFiles := []string{
		"codeforces.json",
		"cses.json",
		"leetcode_problems.json",
		"leetcode_index.json",
		"all_problems.json",
	}

	seenIDs := make(map[string]bool)
	processedFiles := make(map[string]bool)
	var dedupedProblems []RawProblem

	loadFile := func(filePath string) error {
		fileName := filepath.Base(filePath)
		if strings.EqualFold(fileName, "topics.json") {
			return nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read problem file %q: %w", filePath, err)
		}

		var items []RawProblem
		if err := json.Unmarshal(data, &items); err != nil {
			return fmt.Errorf("failed to unmarshal JSON from %q: %w", filePath, err)
		}

		loadedCount := 0
		skippedDupes := 0

		for _, p := range items {
			if len(p.RawTopics) > 0 {
				p.Topics = make([]string, 0, len(p.RawTopics))
				for _, rt := range p.RawTopics {
					switch v := rt.(type) {
					case string:
						if s := strings.TrimSpace(v); s != "" {
							p.Topics = append(p.Topics, s)
						}
					case map[string]interface{}:
						if name, ok := v["name"].(string); ok && strings.TrimSpace(name) != "" {
							p.Topics = append(p.Topics, strings.TrimSpace(name))
						} else if id, ok := v["id"].(string); ok && strings.TrimSpace(id) != "" {
							p.Topics = append(p.Topics, strings.TrimSpace(id))
						} else if slug, ok := v["slug"].(string); ok && strings.TrimSpace(slug) != "" {
							p.Topics = append(p.Topics, strings.TrimSpace(slug))
						}
					}
				}
			}

			if p.ID == "" && p.TitleSlug != "" {
				p.ID = "lc-" + p.TitleSlug
				p.Source = "leetcode"
				p.Name = p.Title
				p.URL = "https://leetcode.com/problems/" + p.TitleSlug
				p.Tags = p.Topics
				p.Statement = p.Body
				p.InputTestcases = p.InputTestcasesLC
				p.OutputTestcases = p.OutputTestcasesLC
			}

			p.ID = strings.TrimSpace(p.ID)
			if p.ID == "" || strings.EqualFold(p.Source, "atcoder") || strings.HasPrefix(strings.ToLower(p.ID), "atcoder-") {
				continue
			}

			if seenIDs[p.ID] {
				skippedDupes++
				continue
			}

			seenIDs[p.ID] = true
			if p.Tags == nil {
				p.Tags = []string{}
			}

			dedupedProblems = append(dedupedProblems, p)
			loadedCount++
		}

		log.Printf("[loader] loaded %d problems from %s (skipped %d duplicates)", loadedCount, fileName, skippedDupes)
		return nil
	}

	// Also check generated/ subdirectory
	dirsToCheck := []string{dataDir, filepath.Join(dataDir, "problems"), filepath.Join(dataDir, "generated")}

	for _, dir := range dirsToCheck {
		if _, err := os.Stat(dir); err == nil {
			for _, pFileName := range priorityFiles {
				candidatePath := filepath.Join(dir, pFileName)
				if _, err := os.Stat(candidatePath); err == nil && !processedFiles[candidatePath] {
					if err := loadFile(candidatePath); err != nil {
						return nil, err
					}
					processedFiles[candidatePath] = true
				}
			}
			entries, _ := os.ReadDir(dir)
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
					filePath := filepath.Join(dir, entry.Name())
					if !processedFiles[filePath] {
						if err := loadFile(filePath); err != nil {
							return nil, err
						}
						processedFiles[filePath] = true
					}
				}
			}
		}
	}

	log.Printf("[loader] total deduplicated problems loaded: %d", len(dedupedProblems))
	return dedupedProblems, nil
}
