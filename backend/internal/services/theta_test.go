package services

import (
	"math"
	"testing"
)

func TestComputeThetaUpdate_Correct(t *testing.T) {
	thetaBefore := 1500.0
	glickoRating := 1500.0
	isCorrect := true
	timeTakenMs := int64(60000)
	expectedTimeMs := int64(60000)

	thetaAfter := ComputeThetaUpdate(thetaBefore, glickoRating, isCorrect, timeTakenMs, expectedTimeMs)

	// Since thetaBefore == glickoRating, thetaIRT == bIRT == 0
	// P(correct) = 1 / (1 + e^0) = 0.5
	// actual = 1.0, diff = 0.5
	// timeFactor = 1.0
	// thetaDelta = 30.0 * 1.0 * 0.5 = +15.0
	// thetaAfter = 1515.0
	expected := 1515.0
	if math.Abs(thetaAfter-expected) > 1e-4 {
		t.Errorf("ComputeThetaUpdate() = %f; want %f", thetaAfter, expected)
	}
}

func TestComputeThetaUpdate_Wrong(t *testing.T) {
	thetaBefore := 1500.0
	glickoRating := 1500.0
	isCorrect := false
	timeTakenMs := int64(60000)
	expectedTimeMs := int64(60000)

	thetaAfter := ComputeThetaUpdate(thetaBefore, glickoRating, isCorrect, timeTakenMs, expectedTimeMs)

	// P(correct) = 0.5, actual = 0.0, diff = -0.5
	// thetaDelta = 30.0 * 1.0 * (-0.5) = -15.0
	// thetaAfter = 1485.0
	expected := 1485.0
	if math.Abs(thetaAfter-expected) > 1e-4 {
		t.Errorf("ComputeThetaUpdate() = %f; want %f", thetaAfter, expected)
	}
}

func TestComputeThetaUpdate_TimeFactorScaling(t *testing.T) {
	thetaBefore := 1500.0
	glickoRating := 1500.0
	isCorrect := true

	// Fast answer (2x faster than expected)
	fastTheta := ComputeThetaUpdate(thetaBefore, glickoRating, isCorrect, 30000, 60000)
	// timeFactor = 2.0 -> delta = 30 * 2.0 * 0.5 = +30.0 -> 1530.0
	if math.Abs(fastTheta-1530.0) > 1e-4 {
		t.Errorf("Fast theta = %f; want 1530.0", fastTheta)
	}

	// Slow answer (2x slower than expected)
	slowTheta := ComputeThetaUpdate(thetaBefore, glickoRating, isCorrect, 120000, 60000)
	// timeFactor = 0.5 -> delta = 30 * 0.5 * 0.5 = +7.5 -> 1507.5
	if math.Abs(slowTheta-1507.5) > 1e-4 {
		t.Errorf("Slow theta = %f; want 1507.5", slowTheta)
	}
}

func TestComputeThetaUpdate_Clamping(t *testing.T) {
	// Floor clamp
	lowTheta := ComputeThetaUpdate(805.0, 3000.0, false, 30000, 60000)
	if lowTheta < thetaFloor {
		t.Errorf("Theta dropped below floor: got %f, floor %f", lowTheta, thetaFloor)
	}

	// Ceiling clamp
	highTheta := ComputeThetaUpdate(3495.0, 1000.0, true, 30000, 60000)
	if highTheta > thetaCeiling {
		t.Errorf("Theta exceeded ceiling: got %f, ceiling %f", highTheta, thetaCeiling)
	}
}

func TestComputeMasteryScore(t *testing.T) {
	tests := []struct {
		name     string
		theta    float64
		expected float64
	}{
		{"floor / cold-start baseline", 1300.0, 0.0},
		{"below baseline", 1100.0, 0.0},
		{"midpoint", 2050.0, 50.0}, // (2050-1300)/1500 * 100 = 750/1500 * 100 = 50.0
		{"ceiling", 2800.0, 100.0},
		{"above ceiling", 3200.0, 100.0},
		{"typical intermediate", 1600.0, 20.0}, // (300/1500)*100 = 20.0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeMasteryScore(tt.theta)
			if math.Abs(got-tt.expected) > 1e-1 {
				t.Errorf("ComputeMasteryScore(%f) = %f; want %f", tt.theta, got, tt.expected)
			}
		})
	}
}
