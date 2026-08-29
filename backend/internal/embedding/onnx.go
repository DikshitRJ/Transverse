// Package embedding provides ONNX Runtime integration for running BAAI/bge-small-en-v1.5 embedding models.
package embedding

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

var (
	ortInitOnce sync.Once
	ortInitErr  error
)

// ensureORTEnvironment guarantees that the ONNX Runtime environment is initialized once per process.
func ensureORTEnvironment() error {
	if ort.IsInitialized() {
		return nil
	}

	ortInitOnce.Do(func() {
		// Attempt standard library discovery or use default environment initialization
		ortInitErr = ort.InitializeEnvironment()
	})

	return ortInitErr
}

// ONNXEmbedder runs BAAI/bge-small-en-v1.5 via ONNX Runtime to generate 384-dimensional dense embeddings.
type ONNXEmbedder struct {
	session   *ort.DynamicAdvancedSession
	tokenizer *Tokenizer
	dims      int
}

// NewONNXEmbedder initializes a new ONNX embedder session from the model path and tokenizer directory.
func NewONNXEmbedder(modelPath string, tokenizerDir string) (*ONNXEmbedder, error) {
	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("onnx model file not found at %q: %w", modelPath, err)
	}

	if tokenizerDir == "" {
		tokenizerDir = filepath.Dir(modelPath)
	}

	tokenizer, err := NewTokenizer(tokenizerDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tokenizer from %q: %w", tokenizerDir, err)
	}

	if err := ensureORTEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize onnxruntime environment: %w", err)
	}

	inputNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputNames := []string{"last_hidden_state"}

	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		// Try alternative output tensor name fallback common in different ONNX exporters
		outputNames = []string{"output_0"}
		session, err = ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
		if err != nil {
			outputNames = []string{"sentence_embedding"}
			session, err = ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create onnx dynamic session for model %q: %w", modelPath, err)
			}
		}
	}

	return &ONNXEmbedder{
		session:   session,
		tokenizer: tokenizer,
		dims:      384,
	}, nil
}

// Embed generates an L2-normalized 384-dimensional dense embedding for a single text string.
func (e *ONNXEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	batchResults, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(batchResults) == 0 {
		return nil, fmt.Errorf("embed failed: empty output vector slice")
	}
	return batchResults[0], nil
}

// EmbedBatch generates normalized 384-dim embeddings for a batch of text strings in a single ONNX forward pass.
func (e *ONNXEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	batchSize := len(texts)
	tokenizedBatch, maxSeqLen := e.tokenizer.TokenizeBatch(texts)
	if maxSeqLen == 0 {
		maxSeqLen = 1
	}

	// Flatten batch tensors for ONNX Runtime CGO interface
	flatInputIDs := make([]int64, batchSize*maxSeqLen)
	flatAttentionMask := make([]int64, batchSize*maxSeqLen)
	flatTokenTypeIDs := make([]int64, batchSize*maxSeqLen)

	for b := 0; b < batchSize; b++ {
		offset := b * maxSeqLen
		for i := 0; i < maxSeqLen; i++ {
			if i < len(tokenizedBatch[b].InputIDs) {
				flatInputIDs[offset+i] = tokenizedBatch[b].InputIDs[i]
				flatAttentionMask[offset+i] = tokenizedBatch[b].AttentionMask[i]
				flatTokenTypeIDs[offset+i] = tokenizedBatch[b].TokenTypeIDs[i]
			} else {
				flatInputIDs[offset+i] = e.tokenizer.padID
				flatAttentionMask[offset+i] = 0
				flatTokenTypeIDs[offset+i] = 0
			}
		}
	}

	inputShape := ort.NewShape(int64(batchSize), int64(maxSeqLen))

	inputIDsTensor, err := ort.NewTensor(inputShape, flatInputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attentionMaskTensor, err := ort.NewTensor(inputShape, flatAttentionMask)
	if err != nil {
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer attentionMaskTensor.Destroy()

	tokenTypeIDsTensor, err := ort.NewTensor(inputShape, flatTokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}
	defer tokenTypeIDsTensor.Destroy()

	outputShape := ort.NewShape(int64(batchSize), int64(maxSeqLen), int64(e.dims))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		// Fallback: try 2D pooled output shape [batchSize, dims]
		outputShape2D := ort.NewShape(int64(batchSize), int64(e.dims))
		outputTensor, err = ort.NewEmptyTensor[float32](outputShape2D)
		if err != nil {
			return nil, fmt.Errorf("failed to create output tensor: %w", err)
		}
	}
	defer outputTensor.Destroy()

	inputTensors := []ort.ArbitraryTensor{inputIDsTensor, attentionMaskTensor, tokenTypeIDsTensor}
	outputTensors := []ort.ArbitraryTensor{outputTensor}

	if err := e.session.Run(inputTensors, outputTensors); err != nil {
		return nil, fmt.Errorf("onnx forward pass execution failed: %w", err)
	}

	rawOutputData := outputTensor.GetData()
	results := make([][]float32, batchSize)

	// Determine if output is 3D [batch, seq_len, dims] or 2D [batch, dims]
	is3DOutput := len(rawOutputData) == batchSize*maxSeqLen*e.dims

	for b := 0; b < batchSize; b++ {
		vec := make([]float32, e.dims)
		if is3DOutput {
			// Extract [CLS] token embedding (index 0 of each sequence)
			clsOffset := b * maxSeqLen * e.dims
			copy(vec, rawOutputData[clsOffset:clsOffset+e.dims])
		} else {
			// 2D pooled output
			offset := b * e.dims
			copy(vec, rawOutputData[offset:offset+e.dims])
		}

		results[b] = L2Normalize(vec)
	}

	return results, nil
}

// Dims returns the embedding vector dimension (384 for bge-small-en-v1.5).
func (e *ONNXEmbedder) Dims() int {
	return e.dims
}

// Close releases the underlying ONNX Runtime session resources.
func (e *ONNXEmbedder) Close() error {
	if e.session != nil {
		err := e.session.Destroy()
		e.session = nil
		if err != nil {
			return fmt.Errorf("failed to destroy onnx session: %w", err)
		}
	}
	return nil
}

// L2Normalize computes the Euclidean L2-normalized vector.
func L2Normalize(vec []float32) []float32 {
	if len(vec) == 0 {
		return vec
	}

	var sumSq float64
	for _, val := range vec {
		sumSq += float64(val) * float64(val)
	}

	norm := float32(math.Sqrt(sumSq))
	if norm == 0 {
		return vec
	}

	normalized := make([]float32, len(vec))
	for i, val := range vec {
		normalized[i] = val / norm
	}

	return normalized
}
