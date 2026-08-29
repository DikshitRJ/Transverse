package embedding_test

import (
	"math"
	"testing"
	"transverse/internal/embedding"
)

func TestL2Normalize(t *testing.T) {
	vec := []float32{3.0, 4.0}
	norm := embedding.L2Normalize(vec)

	if len(norm) != 2 {
		t.Fatalf("expected length 2, got %d", len(norm))
	}

	// 3/5 = 0.6, 4/5 = 0.8
	if math.Abs(float64(norm[0]-0.6)) > 1e-5 || math.Abs(float64(norm[1]-0.8)) > 1e-5 {
		t.Errorf("expected [0.6, 0.8], got %v", norm)
	}

	// Sum of squares of normalized vector should be 1.0
	var sumSq float64
	for _, v := range norm {
		sumSq += float64(v) * float64(v)
	}
	if math.Abs(sumSq-1.0) > 1e-5 {
		t.Errorf("normalized vector magnitude != 1.0 (got %f)", sumSq)
	}

	// Zero vector test
	zeroVec := []float32{0.0, 0.0, 0.0}
	normZero := embedding.L2Normalize(zeroVec)
	for _, v := range normZero {
		if v != 0 {
			t.Errorf("expected 0 for zero vector normalization, got %f", v)
		}
	}
}
