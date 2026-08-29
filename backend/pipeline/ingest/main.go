package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	
	"transverse/internal/config"
	"transverse/internal/graph"
	"transverse/internal/models"
	"transverse/internal/repository"
	"transverse/internal/services"
	"transverse/internal/services/ingest"
)

func main() {
	mode := flag.String("mode", "", "tutorial or roadmap")
	file := flag.String("file", "", "path to json/ndjson file")
	flag.Parse()

	if *mode == "" || *file == "" {
		log.Fatalf("Usage: ingest -mode [tutorial|roadmap] -file <path>")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	topicGraph, err := graph.NewTopicGraph(cfg.TopicsGraphPath)
	if err != nil {
		log.Fatalf("Failed to load topic graph: %v", err)
	}
	
	gs := services.NewGraphService(topicGraph)
	repo := repository.NewIngestRepo(pool)
	svc := ingest.NewService(repo, gs)

	fileBytes, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(string(fileBytes), "\n")
	var isNDJSON bool
	if len(lines) > 1 && len(lines[0]) > 0 && lines[0][0] == '{' && lines[1][0] == '{' {
		isNDJSON = true
	} else if strings.HasSuffix(*file, ".ndjson") {
		isNDJSON = true
	}

	if *mode == "tutorial" {
		var records []models.TutorialIngestRecord
		if isNDJSON {
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				var rec models.TutorialIngestRecord
				if err := json.Unmarshal([]byte(line), &rec); err != nil {
					log.Printf("Failed to decode NDJSON line: %v", err)
				} else {
					records = append(records, rec)
				}
			}
		} else {
			if err := json.Unmarshal(fileBytes, &records); err != nil {
				log.Fatalf("Failed to parse JSON array: %v", err)
			}
		}

		errs := svc.IngestTutorials(ctx, records)
		for _, e := range errs {
			log.Printf("Ingest error: %v", e)
		}
		fmt.Printf("Ingested %d tutorials, %d errors\n", len(records), len(errs))

	} else if *mode == "roadmap" {
		var records []models.RoadmapTemplateIngestRecord
		if isNDJSON {
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				var rec models.RoadmapTemplateIngestRecord
				if err := json.Unmarshal([]byte(line), &rec); err != nil {
					log.Printf("Failed to decode NDJSON line: %v", err)
				} else {
					records = append(records, rec)
				}
			}
		} else {
			if err := json.Unmarshal(fileBytes, &records); err != nil {
				log.Fatalf("Failed to parse JSON array: %v", err)
			}
		}

		errs := svc.IngestRoadmapTemplates(ctx, records)
		for _, e := range errs {
			log.Printf("Ingest error: %v", e)
		}
		fmt.Printf("Ingested %d roadmaps, %d errors\n", len(records), len(errs))
	} else {
		log.Fatalf("Unknown mode: %s", *mode)
	}
}
