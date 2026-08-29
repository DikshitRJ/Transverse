// Package embedding provides dense vector embedding interfaces, BERT/WordPiece tokenization, and ONNX Runtime inference.
package embedding

import "context"

// Provider generates dense vector embeddings for text.
type Provider interface {
	// Embed generates a normalized 384-dim embedding for a single text.
	Embed(ctx context.Context, text string) ([]float32, error)
	// EmbedBatch generates embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	// Dims returns the embedding dimension (384 for bge-small-en-v1.5).
	Dims() int
	// Close releases model resources.
	Close() error
}
