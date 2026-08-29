package services

import (
	"math"
	"testing"
)

func TestUpdateGlickoFromSession_Win(t *testing.T) {
	in := GlickoSessionInput{
		PlayerRating:      1500.0,
		PlayerRD:          200.0,
		PlayerVol:         0.06,
		AvgOpponentRating: 1500.0,
		Score:             1.0, // 100% solve rate in session
	}

	out := UpdateGlickoFromSession(in)

	if out.NewRating <= 1500.0 {
		t.Errorf("expected rating to increase on 100%% solve rate, got %f", out.NewRating)
	}
	if out.NewRD >= 200.0 {
		t.Errorf("expected RD to decrease after observation, got %f", out.NewRD)
	}
	if out.NewVol <= 0 {
		t.Errorf("expected positive volatility, got %f", out.NewVol)
	}
}

func TestUpdateGlickoFromSession_Loss(t *testing.T) {
	in := GlickoSessionInput{
		PlayerRating:      1500.0,
		PlayerRD:          200.0,
		PlayerVol:         0.06,
		AvgOpponentRating: 1500.0,
		Score:             0.0, // 0% solve rate
	}

	out := UpdateGlickoFromSession(in)

	if out.NewRating >= 1500.0 {
		t.Errorf("expected rating to decrease on 0%% solve rate, got %f", out.NewRating)
	}
	if out.NewRD >= 200.0 {
		t.Errorf("expected RD to decrease after session, got %f", out.NewRD)
	}
}

func TestUpdateGlickoFromSession_Clamping(t *testing.T) {
	in := GlickoSessionInput{
		PlayerRating:      3480.0,
		PlayerRD:          300.0,
		PlayerVol:         0.1,
		AvgOpponentRating: 3000.0,
		Score:             1.0,
	}

	out := UpdateGlickoFromSession(in)
	if out.NewRating > 3500.0 {
		t.Errorf("rating exceeded 3500.0 max: got %f", out.NewRating)
	}
}

func TestGlicko2G(t *testing.T) {
	// g(0) = 1 / sqrt(1 + 0) = 1.0
	g0 := glicko2G(0.0)
	if math.Abs(g0-1.0) > 1e-6 {
		t.Errorf("glicko2G(0) = %f; want 1.0", g0)
	}

	// g(phi) should strictly decrease as phi increases
	g1 := glicko2G(1.0)
	g2 := glicko2G(2.0)
	if g1 >= g0 || g2 >= g1 {
		t.Errorf("glicko2G is not monotonically decreasing: g0=%f, g1=%f, g2=%f", g0, g1, g2)
	}
}
