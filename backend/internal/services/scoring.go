package services

import (
	"math"
	"sort"

	"transverse/internal/models"
)

// ScState captures everything needed to pick the next problem.
// Built once per Submit/Skip from session state + user DNA.
type ScState struct {
	ThetaCurrent        float64
	Topic               string         // current active topic from scope
	TopicBias           float64        // from DNA.TopicBias[topic]
	ConsecutiveCorrect  int
	ConsecutiveWrong    int
	QuestionCount       int
	AvgSessionLength    float64        // from DNA
	AvgTimeTakenMs      float64        // from DNA
	CarelessnessIndex   float64        // from DNA [0,1]
	AttemptCounts       map[string]int // problemID -> lifetime attempts
	ActiveSources       []string       // platforms in current session scope
	SessionSourceCounts map[string]int // source -> count in current session (for PlatformDiversity)
}

// WeightSet defines the 7-factor blend for a selection context.
type WeightSet struct {
	DifficultyFit       float64
	ConceptSimilarity   float64
	TopicProgression    float64
	NoveltyFactor       float64
	ImmediateReinforce  float64
	PlatformDiversity   float64
	CarelessnessPenalty float64
}

// Default weight sets (all weights within a set sum to ~1.0 before normalization)
var (
	// Cold start: no current problem, VS/IR can't be computed
	coldStartWeights = WeightSet{
		DifficultyFit:    0.70,
		TopicProgression: 0.10,
		NoveltyFactor:    0.20,
	}

	// After correct: push toward harder problems
	correctWeights = WeightSet{
		DifficultyFit:       0.50,
		ConceptSimilarity:   0.15,
		TopicProgression:    0.10,
		NoveltyFactor:       0.10,
		ImmediateReinforce:  0.05,
		PlatformDiversity:   0.05,
		CarelessnessPenalty: 0.05,
	}

	// After wrong: reinforce similar concept at lower difficulty
	wrongWeights = WeightSet{
		DifficultyFit:       0.15,
		ConceptSimilarity:   0.25,
		TopicProgression:    0.05,
		NoveltyFactor:       0.05,
		ImmediateReinforce:  0.35,
		PlatformDiversity:   0.05,
		CarelessnessPenalty: 0.10,
	}
)

// ScoreComponents records all factor scores for debug logging
type ScoreComponents struct {
	DifficultyFit       float64 `json:"difficulty_fit"`
	ConceptSimilarity   float64 `json:"concept_similarity"`
	TopicProgression    float64 `json:"topic_progression"`
	NoveltyFactor       float64 `json:"novelty_factor"`
	ImmediateReinforce  float64 `json:"immediate_reinforce"`
	PlatformDiversity   float64 `json:"platform_diversity"`
	CarelessnessPenalty float64 `json:"carelessness_penalty"`
	Total               float64 `json:"total"`
}

// PickResult contains the chosen problem recommendation along with its heuristic breakdown.
type PickResult struct {
	Problem  *models.Problem
	Scores   ScoreComponents
	ThetaEff float64
	Momentum float64
}

type scoredProblemCandidate struct {
	problem models.Problem
	score   float64
	comp    ScoreComponents
}

// ComputeEffectiveTheta computes the difficulty target with DSA/CP modifiers:
//
//	thetaEff = ThetaCurrent
//	         + TopicBias * 200          (clamped +-100)
//	         + Momentum                  (correct*15 - wrong*20, clamped +-60)
//	         + SessionPhase              (warm-up <20%: -30, cool-down >=70%: -20)
func ComputeEffectiveTheta(s *ScState) (thetaEff, momentum float64) {
	if s == nil {
		return thetaDefault, 0.0
	}
	thetaEff = s.ThetaCurrent

	// Topic bias (clamped to [-100, 100])
	biasShift := clamp(s.TopicBias*200.0, -100.0, 100.0)
	thetaEff += biasShift

	// Momentum (correct*15 - wrong*20, clamped [-60, 60])
	rawMomentum := float64(s.ConsecutiveCorrect)*15.0 - float64(s.ConsecutiveWrong)*20.0
	momentum = clamp(rawMomentum, -60.0, 60.0)
	thetaEff += momentum

	// Session phase
	if s.AvgSessionLength > 0 {
		pct := float64(s.QuestionCount) / s.AvgSessionLength
		if pct < 0.2 {
			thetaEff -= 30.0 // warm-up
		} else if pct >= 0.7 {
			thetaEff -= 20.0 // cool-down
		}
	}

	thetaEff = clamp(thetaEff, thetaFloor, thetaCeiling)
	return thetaEff, momentum
}

// tuneWeights applies carelessness adjustment:
//
//	w_difficulty += carelessnessIndex * 0.15
//	w_penalty    -= carelessnessIndex * 0.15
//
// Then renormalizes all weights to sum = 1.0
func tuneWeights(w WeightSet, carelessnessIndex float64) WeightSet {
	if carelessnessIndex > 0 {
		shift := carelessnessIndex * 0.15
		w.DifficultyFit += shift
		w.CarelessnessPenalty -= shift
		if w.CarelessnessPenalty < 0 {
			w.CarelessnessPenalty = 0
		}
	}

	sum := w.DifficultyFit + w.ConceptSimilarity + w.TopicProgression + w.NoveltyFactor +
		w.ImmediateReinforce + w.PlatformDiversity + w.CarelessnessPenalty
	if sum > 0 {
		inv := 1.0 / sum
		w.DifficultyFit *= inv
		w.ConceptSimilarity *= inv
		w.TopicProgression *= inv
		w.NoveltyFactor *= inv
		w.ImmediateReinforce *= inv
		w.PlatformDiversity *= inv
		w.CarelessnessPenalty *= inv
	}
	return w
}

// ScoreCandidate computes all 7 score components for a single candidate problem.
//
// Factor formulas:
// 1. DifficultyFit:      1 - |thetaEff - glickoRating| / 1500, clamped [0,1]
// 2. ConceptSimilarity:  cosineSimilarity(current.Embedding, candidate.Embedding)
//                        mapped from [-1,1] to [0,1]: (sim+1)/2
//                        0.0 if no current problem or no embedding
// 3. TopicProgression:   0.3 if candidate.Topic == state.Topic, else 0.0
//                        (future: +0.2 for next prerequisite topic)
// 4. NoveltyFactor:      1.0 - min(attempts/5, 0.5)
//                        1.0 for new, 0.5 for 5+ attempts
// 5. ImmediateReinforce: same as ConceptSimilarity (higher weight in wrong context)
// 6. PlatformDiversity:  1.0 - (sessionSourceCount[source] / max(totalSessionCount,1))
//                        rewards underrepresented platforms
// 7. CarelessnessPenalty: carelessnessIndex * (1 - difficultyFit), clamped [0,1]
//
// Total = Σ(weight_i * factor_i) - w_cp * cp  (CP is subtracted, not added)
// Total clamped to [0, 1]
func ScoreCandidate(p models.Problem, current *models.Problem, thetaEff float64, state *ScState, weights WeightSet) ScoreComponents {
	// 1. DifficultyFit
	diff := math.Abs(thetaEff - p.GlickoRating)
	df := clamp(1.0-diff/1500.0, 0.0, 1.0)

	// 2. ConceptSimilarity
	var cs float64
	if current != nil {
		curEmb := current.Embedding.Slice()
		pEmb := p.Embedding.Slice()
		if len(curEmb) > 0 && len(pEmb) > 0 {
			sim := cosineSimilarity(curEmb, pEmb)
			cs = clamp((sim+1.0)/2.0, 0.0, 1.0)
		}
	}

	// 3. TopicProgression
	var tp float64
	if state != nil && state.Topic != "" && p.Topic == state.Topic {
		tp = 0.3
	}

	// 4. NoveltyFactor
	var nf float64 = 1.0
	if state != nil && state.AttemptCounts != nil {
		count := state.AttemptCounts[p.ID]
		nf = 1.0 - math.Min(float64(count)/5.0, 0.5)
	}

	// 5. ImmediateReinforce
	ir := cs

	// 6. PlatformDiversity
	var pd float64 = 1.0
	if state != nil && state.SessionSourceCounts != nil {
		srcCount := float64(state.SessionSourceCounts[p.Source])
		totalSessionCount := 0
		for _, count := range state.SessionSourceCounts {
			totalSessionCount += count
		}
		denom := math.Max(float64(totalSessionCount), 1.0)
		pd = clamp(1.0-(srcCount/denom), 0.0, 1.0)
	}

	// 7. CarelessnessPenalty
	var cp float64
	if state != nil {
		cp = clamp(state.CarelessnessIndex*(1.0-df), 0.0, 1.0)
	}

	// Weighted Total (CarelessnessPenalty is subtracted per formula)
	total := weights.DifficultyFit*df +
		weights.ConceptSimilarity*cs +
		weights.TopicProgression*tp +
		weights.NoveltyFactor*nf +
		weights.ImmediateReinforce*ir +
		weights.PlatformDiversity*pd -
		weights.CarelessnessPenalty*cp
	total = clamp(total, 0.0, 1.0)

	return ScoreComponents{
		DifficultyFit:       df,
		ConceptSimilarity:   cs,
		TopicProgression:    tp,
		NoveltyFactor:       nf,
		ImmediateReinforce:  ir,
		PlatformDiversity:   pd,
		CarelessnessPenalty: cp,
		Total:               total,
	}
}

// afterCorrectFilter returns only candidates strictly harder than current.
// Falls back to all candidates (minus current) if none are harder.
func afterCorrectFilter(candidates []models.Problem, current *models.Problem) []models.Problem {
	if current == nil {
		return candidates
	}
	currentRating := current.GlickoRating
	var harder []models.Problem
	for i := range candidates {
		if candidates[i].ID != current.ID && candidates[i].GlickoRating > currentRating {
			harder = append(harder, candidates[i])
		}
	}
	if len(harder) > 0 {
		return harder
	}
	// Fallback: all except current
	var fallback []models.Problem
	for i := range candidates {
		if candidates[i].ID != current.ID {
			fallback = append(fallback, candidates[i])
		}
	}
	if len(fallback) > 0 {
		return fallback
	}
	return candidates
}

// afterWrongFilter returns only candidates in the same topic as current.
// Falls back to all candidates (minus current) if none found.
func afterWrongFilter(candidates []models.Problem, current *models.Problem) []models.Problem {
	if current == nil {
		return candidates
	}
	currentTopic := current.Topic
	var sameTopic []models.Problem
	for i := range candidates {
		if candidates[i].ID != current.ID && candidates[i].Topic == currentTopic {
			sameTopic = append(sameTopic, candidates[i])
		}
	}
	if len(sameTopic) > 0 {
		return sameTopic
	}
	// Fallback: all except current
	var fallback []models.Problem
	for i := range candidates {
		if candidates[i].ID != current.ID {
			fallback = append(fallback, candidates[i])
		}
	}
	if len(fallback) > 0 {
		return fallback
	}
	return candidates
}

// PickBestProblem is the unified problem selection function.
// Context:
//
//	currentProblem==nil         -> cold start (no pre-filter, cold-start weights)
//	wasCorrect==true            -> after correct (harder filter, correct weights)
//	wasCorrect==false           -> after wrong (same-topic filter, wrong weights)
func PickBestProblem(candidates []models.Problem, state *ScState, currentProblem *models.Problem, wasCorrect bool) *PickResult {
	if len(candidates) == 0 {
		return nil
	}

	thetaEff, momentum := ComputeEffectiveTheta(state)

	var weights WeightSet
	var pool []models.Problem

	carelessness := 0.0
	if state != nil {
		carelessness = state.CarelessnessIndex
	}

	if currentProblem == nil {
		// Cold start
		weights = tuneWeights(coldStartWeights, carelessness)
		pool = candidates
	} else if wasCorrect {
		weights = tuneWeights(correctWeights, carelessness)
		pool = afterCorrectFilter(candidates, currentProblem)
	} else {
		weights = tuneWeights(wrongWeights, carelessness)
		pool = afterWrongFilter(candidates, currentProblem)
	}

	if len(pool) == 0 {
		pool = candidates
	}

	scored := make([]scoredProblemCandidate, 0, len(pool))
	for i := range pool {
		comp := ScoreCandidate(pool[i], currentProblem, thetaEff, state, weights)
		scored = append(scored, scoredProblemCandidate{
			problem: pool[i],
			score:   comp.Total,
			comp:    comp,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	best := scored[0]
	bestProblem := best.problem
	return &PickResult{
		Problem:  &bestProblem,
		Scores:   best.comp,
		ThetaEff: thetaEff,
		Momentum: momentum,
	}
}

// PickAfterSkip treats skip as wrong but resets streaks to 0 before calling PickBestProblem.
func PickAfterSkip(candidates []models.Problem, state *ScState, skippedProblem *models.Problem) *PickResult {
	var skipState ScState
	if state != nil {
		skipState = *state
	}
	skipState.ConsecutiveCorrect = 0
	skipState.ConsecutiveWrong = 0
	return PickBestProblem(candidates, &skipState, skippedProblem, false)
}

// computeStreaks scans session responses backwards to count consecutive correct/wrong.
func computeStreaks(responses []models.SessionResponse, latestCorrect bool) (consecutiveCorrect, consecutiveWrong int) {
	if latestCorrect {
		consecutiveCorrect = 1
		for i := len(responses) - 1; i >= 0; i-- {
			if responses[i].IsCorrect && !responses[i].Skipped {
				consecutiveCorrect++
			} else {
				break
			}
		}
		return consecutiveCorrect, 0
	}

	consecutiveWrong = 1
	for i := len(responses) - 1; i >= 0; i-- {
		if !responses[i].IsCorrect && !responses[i].Skipped {
			consecutiveWrong++
		} else {
			break
		}
	}
	return 0, consecutiveWrong
}

// ComputeStreaks is the exported helper for computing consecutive streaks from response history.
func ComputeStreaks(responses []models.SessionResponse, latestCorrect bool) (consecutiveCorrect, consecutiveWrong int) {
	return computeStreaks(responses, latestCorrect)
}
