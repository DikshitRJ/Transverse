package services

import (
	"math"
	"sort"

	"velocity/internal/models"
)

// ── State ───────────────────────────────────────────────────────────────────

// ScState captures everything needed to pick the next question.
// Built once per Submit/Skip from session state + user DNA.
type ScState struct {
	ThetaCurrent          float64
	Subject               string
	SubjectBias           float64
	PeakPerformanceHour   int
	CurrentHour           int
	ConsecutiveCorrect    int
	ConsecutiveWrong      int
	QuestionCount         int
	AvgSessionLength      float64
	AvgTimeTakenMs        float64
	ChapterAvgTimeTakenMs float64
	CarelessnessIndex     float64        // from DNA [0, 1], 0=low
	AttemptCounts         map[string]int // questionID → lifetime attempts (from seenIDs)
}

// ── WeightSet ───────────────────────────────────────────────────────────────

// WeightSet defines the 6-factor blend for a selection context.
// After tuning, all six weights sum to 1.0. CarelessnessPenalty is then
// subtracted in the total (see formulae in v0.2.0 SNAPSHOT §3.3).
type WeightSet struct {
	DifficultyFit       float64
	VectorSimilarity    float64
	TimeMatch           float64
	NoveltyFactor       float64
	ImmediateReinforce  float64
	CarelessnessPenalty float64
}

// Default weight sets per context.
//
// Derived from v0.2.0 SNAPSHOT §3.3:
//
//	Default: DF=35%, VS=15%, TM=10%, NF=10%, IR=20%, CP=10%
//	→ Context-tuned to bias the right factors when data is unavailable.
var (
	// Cold start — no prior question, VS/IR can't be computed.
	coldStartWeights = WeightSet{
		DifficultyFit:       0.70,
		VectorSimilarity:    0.00,
		TimeMatch:           0.10,
		NoveltyFactor:       0.20,
		ImmediateReinforce:  0.00,
		CarelessnessPenalty: 0.00,
	}

	// After correct — user is performing well, focus on progression.
	correctWeights = WeightSet{
		DifficultyFit:       0.50,
		VectorSimilarity:    0.15,
		TimeMatch:           0.10,
		NoveltyFactor:       0.10,
		ImmediateReinforce:  0.05,
		CarelessnessPenalty: 0.10,
	}

	// After wrong / skip — reinforce similar material.
	wrongWeights = WeightSet{
		DifficultyFit:       0.15,
		VectorSimilarity:    0.25,
		TimeMatch:           0.10,
		NoveltyFactor:       0.05,
		ImmediateReinforce:  0.35,
		CarelessnessPenalty: 0.10,
	}
)

// ScoreComponents records the individual factor scores for debug logging.
// These are stored in SessionResponse.ScScore and exposed in the debug panel.
type ScoreComponents struct {
	DifficultyFit       float64 `json:"difficulty_fit"`
	VectorSimilarity    float64 `json:"vector_similarity"`
	TimeMatch           float64 `json:"time_match"`
	NoveltyFactor       float64 `json:"novelty_factor"`
	ImmediateReinforce  float64 `json:"immediate_reinforce"`
	CarelessnessPenalty float64 `json:"carelessness_penalty"`
	Total               float64 `json:"total"`
}

type PickResult struct {
	Question *models.Question
	Scores   ScoreComponents
	ThetaEff float64
	Momentum float64
}

type scoredCandidate struct {
	q     models.Question
	score float64
	comp  ScoreComponents
}

// ── JEE Question Type Bonuses ───────────────────────────────────────────────
//
// Different JEE question types have fundamentally different difficulty
// profiles. The bonus is a Glicko-point shift applied to difficulty fit.
//
//	Type              | Bonus  | Rationale
//	──────────────────|────────|──────────────────────────────────────────
//	MCQ               |    0   | 4 options, -1/+4 marking
//	MULTI_CORRECT/MSQ |  -50   | Partial marking, must identify ALL correct
//	NUMERICAL/INTEGER |  +30   | No negative marking, computation not elimination

func questionTypeBonus(qtype string) float64 {
	switch qtype {
	case "MULTI_CORRECT", "MSQ":
		return -50.0
	case "NUMERICAL", "INTEGER":
		return 30.0
	default:
		return 0.0
	}
}

// ── Effective Theta ─────────────────────────────────────────────────────────
//
// Before picking a question, raw theta is adjusted by several modifiers
// to produce a difficulty target (thetaEff). The picker then finds a
// question whose Glicko rating is closest to this target.
//
// Modifiers applied in order:
//  1. SubjectBias × 200  (clamped to [-0.5, 0.5] × 200 = ±100 max)
//  2. Momentum           correct×15 − wrong×20, clamped ±60
//  3. Circadian          peak hour = +25, off hour = −15
//  4. SessionPhase       warm-up (<20%) = −30, cooldown (≥70%) = −20

func ComputeEffectiveTheta(s *ScState) (thetaEff, momentum float64) {
	thetaEff = s.ThetaCurrent

	// Subject bias
	bias := clamp(s.SubjectBias, -0.5, 0.5)
	thetaEff += bias * 200.0

	// Momentum
	rawMomentum := float64(s.ConsecutiveCorrect)*15.0 - float64(s.ConsecutiveWrong)*20.0
	momentum = clamp(rawMomentum, -60.0, 60.0)
	thetaEff += momentum

	// Circadian
	if s.CurrentHour == s.PeakPerformanceHour {
		thetaEff += 25.0
	} else {
		thetaEff -= 15.0
	}

	// Session phase
	if s.AvgSessionLength > 0 {
		pct := float64(s.QuestionCount) / s.AvgSessionLength
		if pct < 0.2 {
			thetaEff -= 30.0 // warm-up
		} else if pct >= 0.7 {
			thetaEff -= 20.0 // cool-down
		}
	}

	return thetaEff, momentum
}

// ── Weight Tuning ───────────────────────────────────────────────────────────

// tuneWeights applies dynamic weight adjustments based on user state.
//
// Carelessness tuning (v0.2.0 SNAPSHOT §3.3):
//
//	w_difficulty += carelessness_index × 0.15
//	w_penalty    -= carelessness_index × 0.15
//	→ all six weights normalized to sum = 1.0
func tuneWeights(w WeightSet, carelessnessIndex float64) WeightSet {
	if carelessnessIndex <= 0 {
		return w
	}

	shift := carelessnessIndex * 0.15
	w.DifficultyFit += shift
	w.CarelessnessPenalty -= shift
	if w.CarelessnessPenalty < 0 {
		w.CarelessnessPenalty = 0
	}

	// Normalise all six weights to sum = 1.0
	sum := w.DifficultyFit + w.VectorSimilarity + w.TimeMatch + w.NoveltyFactor + w.ImmediateReinforce + w.CarelessnessPenalty
	if sum > 0 {
		inv := 1.0 / sum
		w.DifficultyFit *= inv
		w.VectorSimilarity *= inv
		w.TimeMatch *= inv
		w.NoveltyFactor *= inv
		w.ImmediateReinforce *= inv
		w.CarelessnessPenalty *= inv
	}
	return w
}

// ── 6-Factor Scorer ─────────────────────────────────────────────────────────

// ScoreCandidate computes all six score components for a single candidate
// question and returns them with the weighted total.
//
// Each factor is clamped to [0, 1] independently. The total is:
//
//	total = Σ(weight_i × factor_i) - CP_weight × CP_value
//
// where CP (CarelessnessPenalty) is subtracted per the spec.
func ScoreCandidate(q models.Question, current *models.Question, thetaEff float64, state *ScState, weights WeightSet) ScoreComponents {
	// ── 1. DifficultyFit ──────────────────────────────────────────────────
	// How well the question's Glicko rating matches the effective theta.
	diff := math.Abs(thetaEff - float64(q.GlickoRating))
	df := 1.0 - diff/1500.0
	df += questionTypeBonus(q.Type) / 1500.0
	df = clamp(df, 0.0, 1.0)

	// ── 2. VectorSimilarity ──────────────────────────────────────────────
	// Cosine similarity between current and candidate embeddings, mapped
	// from [-1, 1] to [0, 1]. Falls back to 0 when embedding is unavailable
	// or there is no current question (cold start).
	var vs float64
	if current != nil {
		curEmb := current.Embedding.Slice()
		qEmb := q.Embedding.Slice()
		if len(curEmb) > 0 && len(qEmb) > 0 {
			sim := cosineSimilarity(qEmb, curEmb)
			vs = clamp((sim+1)/2, 0.0, 1.0)
		}
	}

	// ── 3. TimeMatch ─────────────────────────────────────────────────────
	// Measures how well the question's expected solve time matches the
	// user's average pace. Returns 0.5 (neutral) when data is unavailable.
	var tm float64
	if state.AvgTimeTakenMs > 0 && q.TimespentAvgMs > 0 {
		avgT := state.AvgTimeTakenMs
		qT := float64(q.TimespentAvgMs)
		maxT := math.Max(avgT, qT)
		tm = 1.0 - math.Abs(avgT-qT)/maxT
		tm = clamp(tm, 0.0, 1.0)
	} else {
		tm = 0.5
	}

	// ── 4. NoveltyFactor ─────────────────────────────────────────────────
	// Penalises questions the user has already seen many times.
	//   new (0 attempts)     → 1.0
	//   5+ attempts          → 0.5
	// No attempt data       → 1.0
	var nf float64
	if state.AttemptCounts != nil {
		count := state.AttemptCounts[q.ID]
		nf = 1.0 - math.Min(float64(count)/5.0, 0.5)
	} else {
		nf = 1.0
	}

	// ── 5. ImmediateReinforce ────────────────────────────────────────────
	// After wrong answers, boosts questions with similar embeddings to the
	// one the user got wrong. Computationally identical to VectorSimilarity;
	// the weight set assigns it meaningful weight only in wrong/skip contexts.
	ir := vs

	// ── 6. CarelessnessPenalty ───────────────────────────────────────────
	// Penalises easy questions (high difficulty fit) for users with a high
	// carelessness index. The penalty scales linearly with carelessness.
	cp := state.CarelessnessIndex * (1.0 - df)
	cp = clamp(cp, 0.0, 1.0)

	// ── Weighted Total ───────────────────────────────────────────────────
	// CP is subtracted per the v0.2.0 SNAPSHOT §3.3 formula.
	total := weights.DifficultyFit*df +
		weights.VectorSimilarity*vs +
		weights.TimeMatch*tm +
		weights.NoveltyFactor*nf +
		weights.ImmediateReinforce*ir -
		weights.CarelessnessPenalty*cp
	total = clamp(total, 0.0, 1.0)

	return ScoreComponents{
		DifficultyFit:       df,
		VectorSimilarity:    vs,
		TimeMatch:           tm,
		NoveltyFactor:       nf,
		ImmediateReinforce:  ir,
		CarelessnessPenalty: cp,
		Total:               total,
	}
}

// ── Pre-Filters ─────────────────────────────────────────────────────────────
//
// Before scoring, the candidate pool is filtered by context to bias toward
// pedagogically useful questions.

// afterCorrectFilter limits to questions strictly harder than the current one.
// Falls back to all candidates (minus current) when none are harder.
func afterCorrectFilter(candidates []models.Question, current *models.Question) []models.Question {
	currentRating := float64(current.GlickoRating)
	var harder []models.Question
	for i := range candidates {
		if float64(candidates[i].GlickoRating) > currentRating && candidates[i].ID != current.ID {
			harder = append(harder, candidates[i])
		}
	}
	if len(harder) > 0 {
		return harder
	}
	// Fallback: all except current
	var all []models.Question
	for i := range candidates {
		if candidates[i].ID != current.ID {
			all = append(all, candidates[i])
		}
	}
	return all
}

// afterWrongFilter limits to the same chapter as the current question.
// Falls back to all candidates (minus current) when none exist in the chapter.
func afterWrongFilter(candidates []models.Question, current *models.Question) []models.Question {
	currentChapter := current.Chapter
	var sameChapter []models.Question
	for i := range candidates {
		if candidates[i].Chapter == currentChapter && candidates[i].ID != current.ID {
			sameChapter = append(sameChapter, candidates[i])
		}
	}
	if len(sameChapter) > 0 {
		return sameChapter
	}
	// Fallback: all except current
	var all []models.Question
	for i := range candidates {
		if candidates[i].ID != current.ID {
			all = append(all, candidates[i])
		}
	}
	return all
}

// ── PickBestQuestion ────────────────────────────────────────────────────────
//
// Unified question selection using the 6-factor weighted scorer.
//
// Context determines the weight set and pre-filter:
//  1. Cold start (currentQuestion == nil) — no pre-filter, cold-start weights
//  2. After correct answer — harder-rating pre-filter, correct weights
//  3. After wrong answer — same-chapter pre-filter, wrong weights
//  4. After skip — same as wrong (streaks reset by PickAfterSkip)

func PickBestQuestion(candidates []models.Question, state *ScState, currentQuestion *models.Question, wasCorrect bool) *PickResult {
	if len(candidates) == 0 {
		return nil
	}

	thetaEff, momentum := ComputeEffectiveTheta(state)

	// Select weight set and pre-filter based on context
	var weights WeightSet
	var pool []models.Question

	if currentQuestion == nil {
		// Cold start
		weights = tuneWeights(coldStartWeights, state.CarelessnessIndex)
		pool = candidates
	} else if wasCorrect {
		weights = tuneWeights(correctWeights, state.CarelessnessIndex)
		pool = afterCorrectFilter(candidates, currentQuestion)
	} else {
		weights = tuneWeights(wrongWeights, state.CarelessnessIndex)
		pool = afterWrongFilter(candidates, currentQuestion)
	}

	if len(pool) == 0 {
		pool = candidates
	}

	// Score every candidate with the 6-factor weighted scorer
	scored := make([]scoredCandidate, 0, len(pool))
	for i := range pool {
		comp := ScoreCandidate(pool[i], currentQuestion, thetaEff, state, weights)
		scored = append(scored, scoredCandidate{
			q:     pool[i],
			score: comp.Total,
			comp:  comp,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	best := scored[0]
	return &PickResult{
		Question: &best.q,
		Scores:   best.comp,
		ThetaEff: thetaEff,
		Momentum: momentum,
	}
}

// ── PickAfterSkip ───────────────────────────────────────────────────────────
//
// After a skip, treat it as PickBestQuestion with wasCorrect=false but
// reset streaks to 0 (skips break streaks).

func PickAfterSkip(candidates []models.Question, state *ScState, skippedQuestion *models.Question) *PickResult {
	skipState := *state
	skipState.ConsecutiveCorrect = 0
	skipState.ConsecutiveWrong = 0
	return PickBestQuestion(candidates, &skipState, skippedQuestion, false)
}
