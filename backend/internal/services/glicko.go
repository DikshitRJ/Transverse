package services

import "math"

const (
	glicko2Scale     = 173.7178 // 400 / ln(10) — converts Glicko-1 <-> Glicko-2
	glickoOpponentRD = 50.0     // residual uncertainty for a session
)

// GlickoSessionInput contains session summary metrics needed for Glicko-2 rating updates.
type GlickoSessionInput struct {
	PlayerRating      float64 // current Glicko-1 rating
	PlayerRD          float64 // current rating deviation
	PlayerVol         float64 // volatility
	AvgOpponentRating float64 // avg glicko_rating of problems served
	Score             float64 // 0.0-1.0 (session accuracy)
}

// GlickoOutput holds updated psychometric parameters after Glicko-2 evaluation.
type GlickoOutput struct {
	NewRating float64
	NewRD     float64
	NewVol    float64
}

// UpdateGlickoFromSession applies full Glicko-2 algorithm.
// Treats the entire session as one game against the average problem difficulty.
func UpdateGlickoFromSession(in GlickoSessionInput) GlickoOutput {
	const (
		tau     = 0.5
		initVol = 0.06
	)

	// Step 1 & 2: Convert inputs to Glicko-2 scale
	r := in.PlayerRating
	rd := math.Max(in.PlayerRD, 30.0)
	vol := math.Max(in.PlayerVol, initVol)

	mu := (r - 1500.0) / glicko2Scale
	phi := rd / glicko2Scale

	// Opponent on Glicko-2 scale
	muJ := (in.AvgOpponentRating - 1500.0) / glicko2Scale
	phiJ := glickoOpponentRD / glicko2Scale

	s := in.Score

	// Step 3: g(φ_j), E(μ, μ_j, φ_j)
	gJ := glicko2G(phiJ)
	e := 1.0 / (1.0 + math.Exp(-gJ*(mu-muJ)))

	// Step 4: v (variance of the outcome)
	v := 1.0 / (gJ * gJ * e * (1.0 - e))

	// Step 5: Δ (rating change)
	delta := v * gJ * (s - e)

	// Step 6: new volatility σ' via Illinois algorithm
	newVol := glicko2Vol(vol, delta, phi, v, tau)

	// Step 7: pre-rating RD φ* = sqrt(φ² + σ'²)
	phiStar := math.Sqrt(phi*phi + newVol*newVol)

	// Step 8: new RD φ' = 1 / sqrt(1/φ*² + 1/v)
	newPhi := 1.0 / math.Sqrt(1.0/(phiStar*phiStar)+1.0/v)

	// Step 9: new rating μ' = μ + φ'² · g(φ_j) · (s - E)
	newMu := mu + newPhi*newPhi*gJ*(s-e)

	// Convert back to Glicko-1 scale
	newRating := newMu*glicko2Scale + 1500.0
	newRD := newPhi * glicko2Scale

	newRating = clamp(newRating, 800.0, 3500.0)

	return GlickoOutput{
		NewRating: newRating,
		NewRD:     newRD,
		NewVol:    newVol,
	}
}

// glicko2G computes g(φ) = 1 / sqrt(1 + 3φ²/π²) on Glicko-2 scale.
func glicko2G(phi float64) float64 {
	return 1.0 / math.Sqrt(1.0+3.0*phi*phi/(math.Pi*math.Pi))
}

// glicko2Vol computes new volatility σ' using the Illinois algorithm (secant root finder).
func glicko2Vol(vol, delta, phi, v, tau float64) float64 {
	a := math.Log(vol * vol)
	epsilon := 1e-6
	A := a

	// f(x) = (e^x·(Δ² − v − e^x)) / (2·(φ² + v + e^x)²) − (x − a) / τ²
	f := func(x float64) float64 {
		ex := math.Exp(x)
		d := phi*phi + v + ex
		top := ex * (delta*delta - v - ex)
		bottom := 2.0 * d * d
		return (top/bottom - (x-a)/(tau*tau))
	}

	for i := 0; i < 100; i++ {
		B := f(A)
		if math.Abs(B) < epsilon {
			break
		}
		A = A - B/(f(A+epsilon)-B)*epsilon
	}

	return math.Exp(A / 2.0)
}
