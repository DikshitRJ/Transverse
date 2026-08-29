// Package services contains domain business logic, psychometric models, and heuristic problem recommendation algorithms.
package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
)

// clamp clamps v to [lo, hi].
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// dotProduct computes the dot product of two float32 slices.
func dotProduct(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}

// l2Norm computes the L2 Euclidean norm of a float32 slice.
func l2Norm(v []float32) float64 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	return math.Sqrt(sum)
}

// cosineSimilarity computes cosine similarity between two equal-length float32 slices.
// Returns value in [-1, 1]. Handles zero vectors gracefully (returns 0).
func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	normA := l2Norm(a)
	normB := l2Norm(b)
	if normA == 0 || normB == 0 {
		return 0
	}
	dot := dotProduct(a, b)
	sim := dot / (normA * normB)
	return clamp(sim, -1.0, 1.0)
}

// generateID generates a random hex ID with the given prefix.
// e.g. generateID("sess") -> "sess_a1b2c3d4e5f6"
func generateID(prefix string) string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s_%x", prefix, bytes)
	}
	if prefix == "" {
		return hex.EncodeToString(bytes)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(bytes))
}
