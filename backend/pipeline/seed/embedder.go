// Package main provides parallel ONNX embedding workers for problem statement vectorization.
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
	"transverse/internal/embedding"
)

// problemBatch represents a chunk of normalized problems submitted to an ONNX worker.
type problemBatch struct {
	BatchID  int
	Problems []NormalizedProblem
}

// batchResult contains the generated embeddings or error produced by a worker.
type batchResult struct {
	BatchID int
	Items   map[string][]float32
	Err     error
}

// EmbedAll generates embeddings for all problems using numWorkers parallel ONNX sessions.
// Each worker goroutine instantiates its own isolated ONNXEmbedder session.
// batchSize controls how many problem texts are evaluated in a single ONNX forward pass.
func EmbedAll(ctx context.Context, problems []NormalizedProblem, modelPath string, tokDir string, numWorkers int, batchSize int) (map[string][]float32, error) {
	totalProblems := len(problems)
	if totalProblems == 0 {
		return make(map[string][]float32), nil
	}

	if numWorkers <= 0 {
		numWorkers = 4
	}
	if batchSize <= 0 {
		batchSize = 32
	}

	// Partition problems into batches
	var batches []problemBatch
	for i := 0; i < totalProblems; i += batchSize {
		end := i + batchSize
		if end > totalProblems {
			end = totalProblems
		}
		batches = append(batches, problemBatch{
			BatchID:  len(batches),
			Problems: problems[i:end],
		})
	}

	totalBatches := len(batches)
	if numWorkers > totalBatches {
		numWorkers = totalBatches
	}

	log.Printf("[embedder] starting parallel embedding: %d problems across %d batches with %d workers (batch_size=%d)",
		totalProblems, totalBatches, numWorkers, batchSize)

	jobs := make(chan problemBatch, totalBatches)
	results := make(chan batchResult, totalBatches)

	for _, b := range batches {
		jobs <- b
	}
	close(jobs)

	var wg sync.WaitGroup
	var processedCounter int64
	startTime := time.Now()

	// Launch worker pool
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			embedder, err := embedding.NewONNXEmbedder(modelPath, tokDir)
			if err != nil {
				results <- batchResult{
					Err: fmt.Errorf("worker %d failed to initialize ONNX embedder: %w", workerID, err),
				}
				return
			}
			defer embedder.Close()

			for {
				select {
				case <-ctx.Done():
					results <- batchResult{Err: ctx.Err()}
					return
				case batch, ok := <-jobs:
					if !ok {
						return
					}

					texts := make([]string, len(batch.Problems))
					for i, p := range batch.Problems {
						texts[i] = p.EmbedText
					}

					vecs, err := embedder.EmbedBatch(ctx, texts)
					if err != nil {
						results <- batchResult{
							BatchID: batch.BatchID,
							Err:     fmt.Errorf("worker %d inference failed for batch %d: %w", workerID, batch.BatchID, err),
						}
						continue
					}

					itemMap := make(map[string][]float32, len(batch.Problems))
					for i, p := range batch.Problems {
						if i < len(vecs) {
							itemMap[p.ID] = vecs[i]
						}
					}

					results <- batchResult{
						BatchID: batch.BatchID,
						Items:   itemMap,
					}

					currTotal := atomic.AddInt64(&processedCounter, int64(len(batch.Problems)))
					if currTotal%500 == 0 || currTotal == int64(totalProblems) {
						elapsed := time.Since(startTime).Seconds()
						rate := 0.0
						if elapsed > 0 {
							rate = float64(currTotal) / elapsed
						}
						pct := (float64(currTotal) / float64(totalProblems)) * 100.0
						log.Printf("[embedder] progress: %d/%d problems (%.1f%%) - %.1f items/sec",
							currTotal, totalProblems, pct, rate)
					}
				}
			}
		}(w + 1)
	}

	// Closer goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	finalEmbeddings := make(map[string][]float32, totalProblems)
	var firstError error

	for res := range results {
		if res.Err != nil {
			if firstError == nil {
				firstError = res.Err
			}
			log.Printf("[embedder] warning: batch failed: %v", res.Err)
			continue
		}

		for id, vec := range res.Items {
			finalEmbeddings[id] = vec
		}
	}

	totalDuration := time.Since(startTime)
	log.Printf("[embedder] embedding completed in %s: %d embeddings generated",
		totalDuration.Round(time.Millisecond), len(finalEmbeddings))

	if len(finalEmbeddings) == 0 && firstError != nil {
		return nil, fmt.Errorf("all embedding batches failed, first error: %w", firstError)
	}

	return finalEmbeddings, nil
}
