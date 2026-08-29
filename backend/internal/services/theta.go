package services

import "math"

const (
	irtDiscrimination = 1.0 / 27.0 // a: 27 Glicko pts ≈ 1% P(correct) shift
	irtLearningRate   = 30.0       // K: max theta change per problem
	thetaFloor        = 800.0      // absolute minimum
	thetaCeiling      = 3500.0     // absolute maximum
	thetaDefault      = 1300.0     // cold-start default
)

// ComputeThetaUpdate runs one IRT step with time-aware scaling.
// isCorrect: true for Accepted (Judge0 status 3)
// timeTakenMs: wall-clock time user spent (from client)
// expectedTimeMs: avg_time_ms from the problem row (0 if unknown)
func ComputeThetaUpdate(thetaBefore, glickoRating float64, isCorrect bool, timeTakenMs, expectedTimeMs int64) float64 {
	// Convert to shared IRT scale: centre on 1500, divide by 100
	thetaIRT := (thetaBefore - 1500.0) / 100.0
	bIRT := (glickoRating - 1500.0) / 100.0

	pCorrect := 1.0 / (1.0 + math.Exp(-irtDiscrimination*(thetaIRT-bIRT)))

	actual := 0.0
	if isCorrect {
		actual = 1.0
	}

	// Symmetric time factor: ratio = expected / actual
	//   ratio > 1 -> answered faster than expected (strong ability signal)
	//   ratio < 1 -> answered slower than expected (struggled / weak signal)
	//   Floor at 0.3, ceiling at 2.0. Defaults to 1.0 if either time is 0.
	timeFactor := 1.0
	if expectedTimeMs > 0 && timeTakenMs > 0 {
		ratio := float64(expectedTimeMs) / float64(timeTakenMs)
		timeFactor = clamp(ratio, 0.3, 2.0)
	}

	thetaNew := thetaBefore + irtLearningRate*timeFactor*(actual-pCorrect)
	return clamp(thetaNew, thetaFloor, thetaCeiling)
}

// ComputeMasteryScore maps theta to 0-100 mastery.
// mastery = round((theta - 1300) / (2800 - 1300) * 100, 1 decimal)
// clamped to [0, 100]
func ComputeMasteryScore(theta float64) float64 {
	if theta <= 1300.0 {
		return 0.0
	}
	if theta >= 2800.0 {
		return 100.0
	}
	raw := (theta - 1300.0) / (2800.0 - 1300.0) * 100.0
	rounded := math.Round(raw*10.0) / 10.0
	return clamp(rounded, 0.0, 100.0)
}
