// Package models contains core domain entities, database models, and data transfer objects.
package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
)

// Problem represents a DSA/CP problem from any competitive programming platform
// stored in the central problem repository.
type Problem struct {
	ID               string          `json:"id" db:"id"`
	Source           string          `json:"source" db:"source"` // "codeforces" | "leetcode" | "atcoder" | "cses"
	Name             string          `json:"name" db:"name"`
	URL              string          `json:"url" db:"url"`
	Slug             string          `json:"slug" db:"slug"`
	ContestID        string          `json:"contest_id" db:"contest_id"`
	Tags             []string        `json:"tags" db:"tags"`
	Topic            string          `json:"topic" db:"topic"` // Normalized primary topic
	Subtopic         string          `json:"subtopic" db:"subtopic"`
	DifficultyLabel  string          `json:"difficulty_label" db:"difficulty_label"` // "easy" | "medium" | "hard" | "expert"
	GlickoRating     float64         `json:"glicko_rating" db:"glicko_rating"`
	GlickoRD         float64         `json:"glicko_rd" db:"glicko_rd"`
	GlickoVolatility float64         `json:"glicko_volatility" db:"glicko_volatility"`
	AttemptCount     int             `json:"attempt_count" db:"attempt_count"`
	SolveRate        float64         `json:"solve_rate" db:"solve_rate"`
	AvgTimeMs        int             `json:"avg_time_ms" db:"avg_time_ms"`
	Embedding        pgvector.Vector `json:"embedding" db:"embedding"` // 384-dimensional dense vector
	EmbedText        string          `json:"embed_text" db:"embed_text"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// User represents a Transverse learner with latent ability (IRT Theta) and Glicko psychometrics.
type User struct {
	ID           string          `json:"id" db:"id"`
	Username     string          `json:"username" db:"username"`
	Email        string          `json:"email" db:"email"`
	Theta        float64         `json:"theta" db:"theta"`
	GlickoRating float64         `json:"glicko_rating" db:"glicko_rating"`
	GlickoRD     float64         `json:"glicko_rd" db:"glicko_rd"`
	GlickoVol    float64         `json:"glicko_vol" db:"glicko_vol"`
	DNARaw       json.RawMessage `json:"dna" db:"dna"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// LearningDNA holds rolling behavioral analytics and psychometric telemetry for a learner.
type LearningDNA struct {
	AvgAccuracy         float64            `json:"avg_accuracy"`
	AvgTimeTakenMs      int64              `json:"avg_time_taken_ms"`
	AvgSolveVelocity    float64            `json:"avg_solve_velocity"`
	CarelessnessIndex   float64            `json:"carelessness_index"`
	PeakPerformanceHour int                `json:"peak_performance_hour"`
	AvgSessionLength    float64            `json:"avg_session_length"`
	TotalSessions       int                `json:"total_sessions"`
	TotalProblemsSolved int                `json:"total_problems_solved"`
	TopicBias           map[string]float64 `json:"topic_bias"`
	PreferredPlatform   string             `json:"preferred_platform"`
	StreakRecord        int                `json:"streak_record"`
}

// DNA deserializes the JSONB DNARaw field into a structured LearningDNA model,
// returning default values if the raw payload is empty or uninitialized.
func (u *User) DNA() (LearningDNA, error) {
	if len(u.DNARaw) == 0 || string(u.DNARaw) == "null" || string(u.DNARaw) == "{}" {
		return DefaultDNA(), nil
	}

	var dna LearningDNA
	if err := json.Unmarshal(u.DNARaw, &dna); err != nil {
		return DefaultDNA(), fmt.Errorf("failed to unmarshal user learning dna: %w", err)
	}

	if dna.TopicBias == nil {
		dna.TopicBias = make(map[string]float64)
	}

	return dna, nil
}

// DefaultDNA returns a baseline LearningDNA struct populated with default values.
func DefaultDNA() LearningDNA {
	return LearningDNA{
		AvgAccuracy:         0.0,
		AvgTimeTakenMs:      0,
		AvgSolveVelocity:    0.0,
		CarelessnessIndex:   0.0,
		PeakPerformanceHour: 18,
		AvgSessionLength:    0.0,
		TotalSessions:       0,
		TotalProblemsSolved: 0,
		TopicBias:           make(map[string]float64),
		PreferredPlatform:   "leetcode",
		StreakRecord:        0,
	}
}

// PracticeSession represents a single active, completed, or abandoned practice session.
type PracticeSession struct {
	ID               string          `json:"id" db:"id"`
	UserID           string          `json:"user_id" db:"user_id"`
	Mode             string          `json:"mode" db:"mode"` // "ADAPTIVE" | "REGULAR"
	ScopeRaw         json.RawMessage `json:"scope" db:"scope"`
	ThetaStart       float64         `json:"theta_start" db:"theta_start"`
	ThetaCurrent     float64         `json:"theta_current" db:"theta_current"`
	CurrentProblemID *string         `json:"current_problem_id" db:"current_problem_id"`
	ResponsesRaw     json.RawMessage `json:"responses" db:"responses"`
	QuestionCount    int             `json:"question_count" db:"question_count"`
	Status           string          `json:"status" db:"status"` // "ACTIVE" | "COMPLETED" | "ABANDONED"
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// SessionScope defines the boundary criteria for candidate problem retrieval in a session.
type SessionScope struct {
	Topics          []string `json:"topics"`
	Subtopics       []string `json:"subtopics,omitempty"`
	Sources         []string `json:"sources,omitempty"`
	DifficultyRange [2]int   `json:"difficulty_range,omitempty"` // [min, max] Glicko rating
}

// Scope deserializes the JSONB ScopeRaw field into a SessionScope struct.
func (s *PracticeSession) Scope() (SessionScope, error) {
	if len(s.ScopeRaw) == 0 || string(s.ScopeRaw) == "null" || string(s.ScopeRaw) == "{}" {
		return SessionScope{Topics: []string{}}, nil
	}

	var scope SessionScope
	if err := json.Unmarshal(s.ScopeRaw, &scope); err != nil {
		return SessionScope{Topics: []string{}}, fmt.Errorf("failed to unmarshal session scope: %w", err)
	}

	if scope.Topics == nil {
		scope.Topics = []string{}
	}

	return scope, nil
}

// SessionResponse records the outcome, psychometrics, and heuristic telemetry for a single problem submission.
type SessionResponse struct {
	ProblemID           string    `json:"problem_id"`
	IsCorrect           bool      `json:"is_correct"`
	Skipped             bool      `json:"skipped"`
	Judge0StatusID      int       `json:"judge0_status_id"`
	Judge0StatusDesc    string    `json:"judge0_status_desc"`
	ExecutionTimeMs     int       `json:"execution_time_ms"`
	MemoryKB            int       `json:"memory_kb"`
	TimeTakenMs         int64     `json:"time_taken_ms"`
	ThetaBefore         float64   `json:"theta_before"`
	ThetaAfter          float64   `json:"theta_after"`
	QuestionCount       int       `json:"question_count"`
	ScScore             float64   `json:"sc_score"`
	DifficultyFit       float64   `json:"difficulty_fit"`
	ConceptSimilarity   float64   `json:"concept_similarity"`
	TopicProgression    float64   `json:"topic_progression"`
	NoveltyFactor       float64   `json:"novelty_factor"`
	ImmediateReinforce  float64   `json:"immediate_reinforce"`
	PlatformDiversity   float64   `json:"platform_diversity"`
	CarelessnessPenalty float64   `json:"carelessness_penalty"`
	ThetaEffective      float64   `json:"theta_effective"`
	Momentum            float64   `json:"momentum"`
	SubmittedAt         time.Time `json:"submitted_at"`
}

// Responses deserializes the JSONB ResponsesRaw slice into []SessionResponse.
func (s *PracticeSession) Responses() ([]SessionResponse, error) {
	if len(s.ResponsesRaw) == 0 || string(s.ResponsesRaw) == "null" || string(s.ResponsesRaw) == "[]" {
		return []SessionResponse{}, nil
	}

	var responses []SessionResponse
	if err := json.Unmarshal(s.ResponsesRaw, &responses); err != nil {
		return []SessionResponse{}, fmt.Errorf("failed to unmarshal session responses: %w", err)
	}

	return responses, nil
}

// UserProblemStats tracks a learner's historical attempts and solve record on a specific problem.
type UserProblemStats struct {
	UserID        string     `json:"user_id" db:"user_id"`
	ProblemID     string     `json:"problem_id" db:"problem_id"`
	AttemptCount  int        `json:"attempt_count" db:"attempt_count"`
	CorrectCount  int        `json:"correct_count" db:"correct_count"`
	TotalTimeMs   int64      `json:"total_time_ms" db:"total_time_ms"`
	LastAttempted *time.Time `json:"last_attempted,omitempty" db:"last_attempted"`
}

// TopicStats tracks per-user per-topic mastery, rating, and engagement counters.
type TopicStats struct {
	UserID       string  `json:"user_id" db:"user_id"`
	Topic        string  `json:"topic" db:"topic"`
	Theta        float64 `json:"theta" db:"theta"`
	MasteryScore float64 `json:"mastery_score" db:"mastery_score"`
	GlickoRating float64 `json:"glicko_rating" db:"glicko_rating"`
	AttemptCount int     `json:"attempt_count" db:"attempt_count"`
	CorrectCount int     `json:"correct_count" db:"correct_count"`
}
