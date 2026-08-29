package services

import (
	"math"
	"strings"
	"testing"
)

func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		v        float64
		lo       float64
		hi       float64
		expected float64
	}{
		{"within bounds", 15.0, 10.0, 20.0, 15.0},
		{"below lo", 5.0, 10.0, 20.0, 10.0},
		{"above hi", 25.0, 10.0, 20.0, 20.0},
		{"exact lo", 10.0, 10.0, 20.0, 10.0},
		{"exact hi", 20.0, 10.0, 20.0, 20.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clamp(tt.v, tt.lo, tt.hi)
			if got != tt.expected {
				t.Errorf("clamp(%f, %f, %f) = %f; want %f", tt.v, tt.lo, tt.hi, got, tt.expected)
			}
		})
	}
}

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{"equal 3D vectors", []float32{1, 2, 3}, []float32{4, 5, 6}, 32.0}, // 1*4 + 2*5 + 3*6 = 4+10+18 = 32
		{"orthogonal vectors", []float32{1, 0}, []float32{0, 1}, 0.0},
		{"mismatched length", []float32{1, 2}, []float32{1, 2, 3}, 0.0},
		{"empty vectors", []float32{}, []float32{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dotProduct(tt.a, tt.b)
			if math.Abs(got-tt.expected) > 1e-6 {
				t.Errorf("dotProduct() = %f; want %f", got, tt.expected)
			}
		})
	}
}

func TestL2Norm(t *testing.T) {
	tests := []struct {
		name     string
		v        []float32
		expected float64
	}{
		{"3-4-5 triangle", []float32{3, 4}, 5.0},
		{"1D unit vector", []float32{1}, 1.0},
		{"zero vector", []float32{0, 0, 0}, 0.0},
		{"empty slice", []float32{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l2Norm(tt.v)
			if math.Abs(got-tt.expected) > 1e-6 {
				t.Errorf("l2Norm() = %f; want %f", got, tt.expected)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{"identical vectors", []float32{1, 2, 3}, []float32{1, 2, 3}, 1.0},
		{"opposite vectors", []float32{1, 0}, []float32{-1, 0}, -1.0},
		{"orthogonal vectors", []float32{1, 0}, []float32{0, 1}, 0.0},
		{"zero vector a", []float32{0, 0}, []float32{1, 1}, 0.0},
		{"zero vector b", []float32{1, 1}, []float32{0, 0}, 0.0},
		{"mismatched length", []float32{1, 2}, []float32{1, 2, 3}, 0.0},
		{"empty vectors", []float32{}, []float32{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.expected) > 1e-5 {
				t.Errorf("cosineSimilarity() = %f; want %f", got, tt.expected)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID("sess")
	id2 := generateID("sess")

	if !strings.HasPrefix(id1, "sess_") {
		t.Errorf("expected id to have prefix sess_, got %s", id1)
	}
	if id1 == id2 {
		t.Errorf("expected random distinct IDs, got duplicate %s", id1)
	}

	rawID := generateID("")
	if strings.Contains(rawID, "_") {
		t.Errorf("expected no prefix delimiter for empty prefix, got %s", rawID)
	}
}
