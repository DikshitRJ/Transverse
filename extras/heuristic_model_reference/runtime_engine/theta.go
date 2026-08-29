package services

import "math"

// ── IRT Theta Update (the "ladder") ─────────────────────────────────────────
//
// Uses a 1PL (Rasch) approximation:
//
//	P_correct = 1 / (1 + e^(-a·(θ_irt - b_irt)))
//	θ_new = θ + K · timeFactor · (actual - P_correct)
//
// Where:
//   - a = discrimination (fixed at 1.0 / 27.0 per JEE scale)
//   - b_irt = (question Glicko rating − 1500) / 100  (IRT scale)
//   - θ_irt = (user theta − 1500) / 100               (IRT scale)
//   - K = learning rate (how fast theta moves per question)
//   - timeFactor = scales the update by how quickly the student answered
//     relative to the question's average solve time.
//     Quick answers → full update (strong ability signal).
//     Slow answers → reduced update (struggled / unsure).
//   - θ = user theta on JEE scale (1300-2800)
//
// Both θ and question difficulty are centred on 1500 (midpoint of JEE range)
// and divided by 100 to map to a shared IRT scale before computing P(correct).
// This keeps the logistic curve smooth and the theta movement gradual.
//
// Theta is maintained on the JEE score scale (~1300–2800), NOT the standard
// IRT N(0,1) scale. This keeps it human-readable and directly mappable to
// mastery scores and question Glicko ratings.

const (
	irtDiscrimination = 1.0 / 27.0 // a: one JEE scale point ≈ 27 rating points
	irtLearningRate   = 30.0       // K: max theta change per question

	thetaFloor   = 800.0  // absolute minimum (empty guesser)
	thetaCeiling = 3500.0 // absolute maximum (perfect performer)
)

// ComputeThetaUpdate runs one IRT step with symmetric time-aware scaling.
// timeTakenMs is the student's actual time; expectedTimeMs is the question's
// average solve time (timespent_avg_ms). The update magnitude is scaled
// by a time factor:
//   - Answered at expected time → 1.0 (neutral)
//   - Answered 2x faster → 2.0 (double the signal — clearly knew/didn't know)
//   - Answered 2x slower → 0.5 (half signal — struggled/guessed)
//   - Floor at 0.3, ceiling at 2.0
//
// Unlike the previous asymmetric version (cap at 1.0), this rewards fast
// correct answers with a larger theta gain and penalises fast wrong answers
// with a larger theta drop — because response time is a strong signal of
// ability certainty.
func ComputeThetaUpdate(thetaBefore float64, questionGlicko float64, isCorrect bool, timeTakenMs, expectedTimeMs int64) float64 {
	// Convert to shared IRT scale: centre on 1500, divide by 100
	thetaIRT := (thetaBefore - 1500.0) / 100.0
	bIRT := (questionGlicko - 1500.0) / 100.0

	pCorrect := 1.0 / (1.0 + math.Exp(-irtDiscrimination*(thetaIRT-bIRT)))

	actual := 0.0
	if isCorrect {
		actual = 1.0
	}

	// Symmetric time factor: ratio = expected / actual
	//   ratio > 1 → answered faster than expected (strong signal)
	//   ratio < 1 → answered slower than expected (weak signal)
	timeFactor := 1.0
	if expectedTimeMs > 0 && timeTakenMs > 0 {
		ratio := float64(expectedTimeMs) / float64(timeTakenMs)
		timeFactor = clamp(ratio, 0.3, 2.0)
	}

	thetaNew := thetaBefore + irtLearningRate*timeFactor*(actual-pCorrect)
	return clamp(thetaNew, thetaFloor, thetaCeiling)
}
