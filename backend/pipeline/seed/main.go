// Package main is the CLI entrypoint for the Transverse DSA/CP embedding pipeline and database seeder.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"transverse/internal/database"
	"transverse/internal/graph"
)

func main() {
	dataDir := flag.String("data-dir", "./data/generated", "Path to problem dataset JSON directory (required)")
	modelPath := flag.String("model", "./models/bge-small-en-v1.5.onnx", "Path to ONNX embedding model (required)")
	tokenizerDir := flag.String("tokenizer", "", "Path to tokenizer directory (defaults to model directory)")
	topicsPath := flag.String("topics", "./data/topics.json", "Path to topics.json knowledge graph file")
	dbURL := flag.String("db", "", "Database URL connection string (or read from DATABASE_URL env)")
	sqlDir := flag.String("sql-dir", "./backend/sql", "Path to directory containing SQL migration scripts")
	batchSize := flag.Int("batch-size", 32, "ONNX inference batch size per worker forward pass")
	workers := flag.Int("workers", 4, "Number of parallel ONNX embedding workers")
	migrate := flag.Bool("migrate", false, "Execute SQL schema migrations in alphabetical order before seeding")
	dryRun := flag.Bool("dry-run", false, "Load, normalize, and evaluate without inserting into the database")
	skipEmbed := flag.Bool("skip-embed", false, "Skip ONNX vector embedding generation (seeds metadata only)")
	limit := flag.Int("limit", 0, "Optional cap on number of problems to process (0 for unlimited)")
	flag.Parse()

	log.Println("=================================================================")
	log.Println("🚀 Transverse DSA/CP Adaptive Engine Seeding Pipeline")
	log.Println("=================================================================")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	overallStart := time.Now()

	// 1. Resolve and load topic graph
	resolvedTopicsPath := resolveFilePath(*topicsPath, []string{
		"./data/topics.json",
		"../data/topics.json",
		filepath.Join(*dataDir, "topics.json"),
		filepath.Join(*dataDir, "../topics.json"),
	})

	log.Printf("[main] loading knowledge graph from %q...", resolvedTopicsPath)
	topicGraph, err := graph.NewTopicGraph(resolvedTopicsPath)
	if err != nil {
		log.Fatalf("[main] failed to load topic graph: %v", err)
	}
	allTopics := topicGraph.GetAllTopics()
	log.Printf("[main] topic graph loaded successfully with %d canonical topics", len(allTopics))

	// 2. Database connection & optional migrations
	var pool *pgxpool.Pool
	if !*dryRun {
		connectionString := *dbURL
		if strings.TrimSpace(connectionString) == "" {
			connectionString = os.Getenv("DATABASE_URL")
		}

		if strings.TrimSpace(connectionString) == "" {
			log.Fatalf("[main] error: DATABASE_URL must be specified via --db flag or environment variable (or run with --dry-run)")
		}

		log.Printf("[main] connecting to PostgreSQL database...")
		pool, err = database.NewPoolFromURL(ctx, connectionString, 2, 10)
		if err != nil {
			log.Fatalf("[main] database connection failed: %v", err)
		}
		defer pool.Close()

		if *migrate {
			resolvedSQLDir := resolveDirPath(*sqlDir, []string{
				"./backend/sql",
				"./sql",
				"../sql",
			})
			log.Printf("[main] running database migrations from %q...", resolvedSQLDir)
			if err := runMigrations(ctx, pool, resolvedSQLDir); err != nil {
				log.Fatalf("[main] migration failure: %v", err)
			}
			log.Printf("[main] all migrations applied successfully")
		}
	} else {
		log.Println("[main] running in DRY-RUN mode: database modifications will be skipped")
	}

	// 3. Load and deduplicate problems from data directory
	resolvedDataDir := resolveDirPath(*dataDir, []string{
		"./data/generated",
		"./data",
		"../data/generated",
		"../data",
	})

	log.Printf("[main] reading problem bank from %q...", resolvedDataDir)
	rawProblems, err := LoadProblems(resolvedDataDir)
	if err != nil {
		log.Fatalf("[main] failed loading problems: %v", err)
	}

	if *limit > 0 && len(rawProblems) > *limit {
		log.Printf("[main] applying limit: truncating from %d to %d problems", len(rawProblems), *limit)
		rawProblems = rawProblems[:*limit]
	}

	// 4. Normalize all problems
	log.Printf("[main] normalizing %d problems against topic knowledge graph...", len(rawProblems))
	normalized := make([]NormalizedProblem, len(rawProblems))
	topicCounts := make(map[string]int)
	difficultyCounts := make(map[string]int)
	sourceCounts := make(map[string]int)

	for i, raw := range rawProblems {
		np := Normalize(raw, topicGraph)
		normalized[i] = np
		topicCounts[np.Topic]++
		difficultyCounts[np.DifficultyLabel]++
		sourceCounts[np.Source]++
	}

	log.Println("-----------------------------------------------------------------")
	log.Printf("📊 Platform Sources: %v", sourceCounts)
	log.Printf("📊 Difficulty Distribution: %v", difficultyCounts)
	log.Printf("📊 Top Topics: %v", getTopEntries(topicCounts, 6))
	log.Println("-----------------------------------------------------------------")

	// 5. Generate embeddings via ONNX Runtime
	embeddings := make(map[string][]float32)
	if !*skipEmbed {
		resolvedModelPath := *modelPath
		resolvedTokDir := *tokenizerDir
		if resolvedTokDir == "" {
			resolvedTokDir = filepath.Dir(resolvedModelPath)
		}

		if _, err := os.Stat(resolvedModelPath); err != nil {
			if *dryRun {
				log.Printf("[main] notice: ONNX model %q not found, continuing dry-run without embeddings", resolvedModelPath)
			} else {
				log.Fatalf("[main] error: ONNX model file not found at %q: %v", resolvedModelPath, err)
			}
		} else {
			log.Printf("[main] initiating embedding generation (model=%s, workers=%d, batch_size=%d)...",
				resolvedModelPath, *workers, *batchSize)
			embMap, err := EmbedAll(ctx, normalized, resolvedModelPath, resolvedTokDir, *workers, *batchSize)
			if err != nil {
				log.Fatalf("[main] embedding generation failed: %v", err)
			}
			embeddings = embMap
		}
	} else {
		log.Println("[main] skipping embedding generation as requested by --skip-embed flag")
	}

	// 6. Seed to database
	if !*dryRun && pool != nil {
		log.Printf("[main] seeding %d normalized problems into PostgreSQL...", len(normalized))
		if err := SeedProblems(ctx, pool, normalized, embeddings, 250); err != nil {
			log.Fatalf("[main] seeding failed: %v", err)
		}
	}

	// 7. Final Summary
	elapsed := time.Since(overallStart)
	log.Println("=================================================================")
	log.Println("✅ Transverse Seeding Pipeline Completed Successfully")
	log.Printf("   • Total Problems Processed : %d", len(normalized))
	log.Printf("   • Embeddings Generated     : %d", len(embeddings))
	if !*dryRun {
		log.Printf("   • Problems Upserted to DB  : %d", len(normalized))
	} else {
		log.Printf("   • DB Upsert Status         : Skipped (Dry Run)")
	}
	log.Printf("   • Total Pipeline Duration  : %s", elapsed.Round(time.Millisecond))
	log.Println("=================================================================")
}

// runMigrations discovers and executes all .sql scripts in alphabetical order within the directory.
func runMigrations(ctx context.Context, pool *pgxpool.Pool, sqlDir string) error {
	entries, err := os.ReadDir(sqlDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %q: %w", sqlDir, err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, filepath.Join(sqlDir, entry.Name()))
		}
	}

	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %q: %w", file, err)
		}

		log.Printf("[migrate] applying %s...", filepath.Base(file))
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("failed executing migration %q: %w", filepath.Base(file), err)
		}
	}

	return nil
}

// resolveFilePath checks if targetPath exists, otherwise checks alternate search locations.
func resolveFilePath(targetPath string, alternatives []string) string {
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath
	}
	for _, alt := range alternatives {
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}
	return targetPath
}

// resolveDirPath checks if targetDir exists, otherwise checks alternate directory locations.
func resolveDirPath(targetDir string, alternatives []string) string {
	if fi, err := os.Stat(targetDir); err == nil && fi.IsDir() {
		return targetDir
	}
	for _, alt := range alternatives {
		if fi, err := os.Stat(alt); err == nil && fi.IsDir() {
			return alt
		}
	}
	return targetDir
}

// getTopEntries returns the top N entries from a frequency counter map.
func getTopEntries(m map[string]int, n int) map[string]int {
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range m {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	res := make(map[string]int)
	for i := 0; i < len(ss) && i < n; i++ {
		res[ss[i].Key] = ss[i].Value
	}
	return res
}
