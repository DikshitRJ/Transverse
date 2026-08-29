package models

import "time"

// ProblemPayload represents sanitized problem information safe for external client consumption,
// excluding internal psychometric parameters and embedding vectors.
type ProblemPayload struct {
	ID              string   `json:"id"`
	Source          string   `json:"source"`
	Name            string   `json:"name"`
	URL             string   `json:"url"`
	Slug            string   `json:"slug"`
	ContestID       string   `json:"contest_id,omitempty"`
	Tags            []string `json:"tags"`
	Topic           string   `json:"topic"`
	Subtopic        string   `json:"subtopic"`
	DifficultyLabel string   `json:"difficulty_label"`
	SolveRate       float64  `json:"solve_rate"`
	AvgTimeMs       int      `json:"avg_time_ms"`
}

// ToProblemPayload converts a database Problem entity into a sanitized ProblemPayload.
func ToProblemPayload(p *Problem) ProblemPayload {
	if p == nil {
		return ProblemPayload{}
	}
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return ProblemPayload{
		ID:              p.ID,
		Source:          p.Source,
		Name:            p.Name,
		URL:             p.URL,
		Slug:            p.Slug,
		ContestID:       p.ContestID,
		Tags:            tags,
		Topic:           p.Topic,
		Subtopic:        p.Subtopic,
		DifficultyLabel: p.DifficultyLabel,
		SolveRate:       p.SolveRate,
		AvgTimeMs:       p.AvgTimeMs,
	}
}

// ToProblemPayloads converts a slice of Problem entities into sanitized ProblemPayload structs.
func ToProblemPayloads(problems []Problem) []ProblemPayload {
	if problems == nil {
		return []ProblemPayload{}
	}
	payloads := make([]ProblemPayload, len(problems))
	for i := range problems {
		payloads[i] = ToProblemPayload(&problems[i])
	}
	return payloads
}

// StartSessionRequest defines the payload for creating or resuming a practice session.
type StartSessionRequest struct {
	Mode  string       `json:"mode"` // "ADAPTIVE" | "REGULAR"
	Scope SessionScope `json:"scope"`
}

// StartSessionResponse returns the initialized practice session and its first problem.
type StartSessionResponse struct {
	SessionID      string          `json:"session_id"`
	Mode           string          `json:"mode"`
	Theta          float64         `json:"theta"`
	CurrentProblem *ProblemPayload `json:"current_problem,omitempty"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

// VerdictDetail holds execution outcome details from Judge0.
type VerdictDetail struct {
	StatusID      int    `json:"status_id"`
	StatusDesc    string `json:"status_desc"`
	TimeMs        int    `json:"time_ms"`
	MemoryKB      int    `json:"memory_kb"`
	Stderr        string `json:"stderr,omitempty"`
	CompileOutput string `json:"compile_output,omitempty"`
	Message       string `json:"message,omitempty"`
}

// SubmitRequest submits a Judge0 execution token for server-side evaluation in the session.
type SubmitRequest struct {
	Judge0Token string `json:"judge0_token"`
	TimeTakenMs int64  `json:"time_taken_ms"`
}

// SubmitResponse returns the evaluation verdict, updated ability estimates, and next problem.
type SubmitResponse struct {
	IsCorrect           bool            `json:"is_correct"`
	Verdict             VerdictDetail   `json:"verdict"`
	ThetaBefore         float64         `json:"theta_before"`
	ThetaAfter          float64         `json:"theta_after"`
	NextProblem         *ProblemPayload `json:"next_problem,omitempty"`
	SessionStatus       string          `json:"session_status"`
	QuestionCount       int             `json:"question_count"`
	CarelessnessPenalty float64         `json:"carelessness_penalty,omitempty"`
}

// SkipRequest records a skipped problem during a practice session.
type SkipRequest struct {
	TimeTakenMs int64 `json:"time_taken_ms"`
}

// SkipResponse returns the post-skip psychometric adjustments and next problem candidate.
type SkipResponse struct {
	Skipped       bool            `json:"skipped"`
	ThetaBefore   float64         `json:"theta_before"`
	ThetaAfter    float64         `json:"theta_after"`
	NextProblem   *ProblemPayload `json:"next_problem,omitempty"`
	QuestionCount int             `json:"question_count"`
}

// CloseSessionResponse returns the final session summary and updated topic masteries.
type CloseSessionResponse struct {
	SessionID         string                   `json:"session_id"`
	Status            string                   `json:"status"`
	ThetaStart        float64                  `json:"theta_start"`
	ThetaFinal        float64                  `json:"theta_final"`
	MasteryScore      float64                  `json:"mastery_score"`
	Accuracy          float64                  `json:"accuracy"`
	TotalQuestions    int                      `json:"total_questions"`
	TotalSolved       int                      `json:"total_solved"`
	PerTopicBreakdown map[string]TopicProgress `json:"per_topic_breakdown"`
}

// GetSessionResponse returns the complete state of an active or past practice session.
type GetSessionResponse struct {
	SessionID      string            `json:"session_id"`
	UserID         string            `json:"user_id"`
	Mode           string            `json:"mode"`
	Status         string            `json:"status"`
	Scope          SessionScope      `json:"scope"`
	ThetaStart     float64           `json:"theta_start"`
	ThetaCurrent   float64           `json:"theta_current"`
	QuestionCount  int               `json:"question_count"`
	CurrentProblem *ProblemPayload   `json:"current_problem,omitempty"`
	Responses      []SessionResponse `json:"responses"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// TopicProgress summarizes a user's mastery and problem volume in a topic.
type TopicProgress struct {
	Topic        string  `json:"topic"`
	MasteryScore float64 `json:"mastery_score"`
	Theta        float64 `json:"theta"`
	GlickoRating float64 `json:"glicko_rating"`
	AttemptCount int     `json:"attempt_count"`
	CorrectCount int     `json:"correct_count"`
}

// TopicsResponse contains progress records across all known curriculum topics.
type TopicsResponse struct {
	Topics []TopicProgress `json:"topics"`
}

// UserProfileResponse returns the public and analytical profile for a learner.
type UserProfileResponse struct {
	ID           string      `json:"id"`
	Username     string      `json:"username"`
	Email        string      `json:"email"`
	Theta        float64     `json:"theta"`
	GlickoRating float64     `json:"glicko_rating"`
	GlickoRD     float64     `json:"glicko_rd"`
	DNA          LearningDNA `json:"dna"`
	CreatedAt    time.Time   `json:"created_at"`
}

// SimilarProblemsRequest requests semantically similar problems based on vector embeddings.
type SimilarProblemsRequest struct {
	ProblemID        string   `json:"problem_id"`
	Limit            int      `json:"limit,omitempty"`
	TargetDifficulty *float64 `json:"target_difficulty,omitempty"`
}

// SimilarProblemsResponse contains nearest-neighbor problem recommendations.
type SimilarProblemsResponse struct {
	ProblemID       string           `json:"problem_id"`
	SimilarProblems []ProblemPayload `json:"similar_problems"`
}

// ExecuteRequest represents a code execution submission to Judge0.
type ExecuteRequest struct {
	ProblemID   string `json:"problem_id"`
	LanguageID  int    `json:"language_id"`
	SourceCode  string `json:"source_code"`
	CustomStdin string `json:"custom_stdin,omitempty"`
}

// ExecuteResponse returns the tracking token for an asynchronous Judge0 run.
type ExecuteResponse struct {
	Judge0Token string `json:"judge0_token"`
}

// VerdictPollResponse contains the polling result for a Judge0 execution submission.
type VerdictPollResponse struct {
	Token         string `json:"token"`
	StatusID      int    `json:"status_id"`
	StatusDesc    string `json:"status_desc"`
	TimeMs        int    `json:"time_ms"`
	MemoryKB      int    `json:"memory_kb"`
	Stdout        string `json:"stdout,omitempty"`
	Stderr        string `json:"stderr,omitempty"`
	CompileOutput string `json:"compile_output,omitempty"`
	Message       string `json:"message,omitempty"`
	IsDone        bool   `json:"is_done"`
}

// ProblemSearchRequest defines filtering parameters for searching the problem repository.
type ProblemSearchRequest struct {
	Query           string `json:"query"`
	Topic           string `json:"topic,omitempty"`
	Source          string `json:"source,omitempty"`
	DifficultyLabel string `json:"difficulty_label,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	Offset          int    `json:"offset,omitempty"`
}

// ProblemSearchResponse returns paginated problem search results.
type ProblemSearchResponse struct {
	Total    int              `json:"total"`
	Problems []ProblemPayload `json:"problems"`
}
