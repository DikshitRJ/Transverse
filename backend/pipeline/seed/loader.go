// Package main provides the offline database seeding and embedding pipeline for Transverse problem repositories.
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
	ID               string   `json:"id"`
	Source           string   `json:"source"`
	Name             string   `json:"name"`
	URL              string   `json:"url"`
	DifficultyRating *int     `json:"difficulty_rating"` // nullable integer
	Tags             []string `json:"tags"`
	ContestID        string   `json:"contest_id"`
	Notes            *string  `json:"notes"`
}

// LoadProblems loads all problem JSON files from dataDir and deduplicates by ID.
// Priority order for deduplication: codeforces > atcoder > cses > leetcode > all_problems > other json files.
func LoadProblems(dataDir string) ([]RawProblem, error) {
	if _, err := os.Stat(dataDir); err != nil {
		return nil, fmt.Errorf("specified data directory %q does not exist: %w", dataDir, err)
	}

	// Check if dataDir points to parent data folder containing a 'generated' subfolder
	targetDir := dataDir
	generatedSubdir := filepath.Join(dataDir, "generated")
	if fi, err := os.Stat(generatedSubdir); err == nil && fi.IsDir() {
		targetDir = generatedSubdir
	}

	priorityFiles := []string{
		"codeforces.json",
		"atcoder.json",
		"cses.json",
		"leetcode_index.json",
		"all_problems.json",
	}

	seenIDs := make(map[string]bool)
	processedFiles := make(map[string]bool)
	var dedupedProblems []RawProblem

	// Helper to load and append problems from a specific JSON file path
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
			p.ID = strings.TrimSpace(p.ID)
			if p.ID == "" {
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

	// 1. Process priority files first in strict precedence order
	for _, pFileName := range priorityFiles {
		candidatePath := filepath.Join(targetDir, pFileName)
		if _, err := os.Stat(candidatePath); err == nil {
			if err := loadFile(candidatePath); err != nil {
				return nil, err
			}
			processedFiles[candidatePath] = true
		}
	}

	// 2. Discover any additional JSON files in targetDir not already processed
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read entries from directory %q: %w", targetDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}

		filePath := filepath.Join(targetDir, entry.Name())
		if !processedFiles[filePath] {
			if err := loadFile(filePath); err != nil {
				return nil, err
			}
			processedFiles[filePath] = true
		}
	}

	log.Printf("[loader] total deduplicated problems loaded: %d", len(dedupedProblems))
	return dedupedProblems, nil
}
