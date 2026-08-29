package models

import "time"

// dto.go — Data Transfer Objects
//
// These are the ONLY structs that ever get serialised to JSON and sent to
// the client. Nothing from db_models.go ever crosses this boundary directly.
//
// The handlers layer converts db_models → DTOs before writing the response.
// The service layer returns DTOs to the handler — it never returns raw db models.
//
// LEARN MODE:   answers are revealed after submission (student is studying).
// CONTEST/MOCK: answers stay hidden until the entire test is submitted.
//
// The structural guarantee: `correct_options` and `explanation` only appear
// on LearnSubmitResponse. They are absent by design from every other type.

// ═══════════════════════════════════════════════════════════════════════════
// SHARED — used across learn, contest, and mock modes
// ═══════════════════════════════════════════════════════════════════════════

// QuestionOptionDTO is the public representation of one answer choice.
// Identical shape to the internal QuestionOption — but defined separately
// so a future change to one doesn't silently affect the other.
type QuestionOptionDTO struct {
	Key    string   `json:"key"`    // "A", "B", "C", "D"
	Text   string   `json:"text"`   // LaTeX / Markdown content
	Images []string `json:"images"` // signed asset URLs, empty slice if none
}

// QuestionPayload is the safe public representation of a question.
// This is what the frontend renders. Notice what is absent:
//   - No `correct` field
//   - No `embedding`
//   - Raw Glicko internals are NOT exposed — only a derived RatingTag
//
// Used inside StartLearnSessionResponse, LearnSubmitResponse.NextQuestion,
// and contest question delivery. Always answer-free on first serve.
type QuestionPayload struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`          // "MCQ" | "MULTI_CORRECT" | "NUMERICAL"
	QuestionText string              `json:"question_text"` // Markdown/LaTeX
	Images       []string            `json:"images"`        // asset URLs, empty slice if none
	Options      []QuestionOptionDTO `json:"options"`       // empty slice for NUMERICAL type
	Subject      string              `json:"subject"`
	Chapter      string              `json:"chapter"`
	ChapterGroup string              `json:"chapter_group"`
	ShiftDate    string              `json:"shift_date"`    // e.g. "2026_2401S1" = 24 Jan 2026 Shift 1
	Source       string              `json:"source"`        // "JEE Main 2026" — useful for PYQ display
	ExamType     string              `json:"exam_type"`     // "JEE_MAIN" | "JEE_ADV"
	RatingTag    string              `json:"rating_tag"`    // dynamic difficulty tag: "very easy" | "easy" | "medium" | "hard" | "very hard"
	AttemptCount int                 `json:"attempt_count"` // number of times this user has attempted this question (0 = never)
}

// ═══════════════════════════════════════════════════════════════════════════
// LEARN MODE
// ═══════════════════════════════════════════════════════════════════════════

// StartLearnSessionRequest is the payload sent by the client to begin a session.
// At least one of Chapters, ChapterGroups, or Subjects must be provided.
// If multiple are provided, the union of all matching questions is used.
type StartLearnSessionRequest struct {
	Mode             string   `json:"mode"`    // "ADAPTIVE" | "REGULAR"
	Chapter          string   `json:"chapter"` // DEPRECATED: kept for backward compat, single chapter
	Chapters         []string `json:"chapters,omitempty"`
	ChapterGroups    []string `json:"chapter_groups,omitempty"`
	Subjects         []string `json:"subjects,omitempty"`   // "physics" | "chemistry" | "maths"
	Years            []string `json:"years,omitempty"`      // e.g. ["2024", "2026"] — filter by exam year
	ExamTypes        []string `json:"exam_types,omitempty"` // e.g. ["JEE_MAIN", "JEE_ADV"] — filter by exam type
	QuestionLimit    int      `json:"question_limit"`       // 0 = no limit, session runs forever
	BiometricEnabled bool     `json:"biometric_enabled"`    // enable camera tracking (beta)
	QuestionOrdering string   `json:"question_ordering"`    // "latest_first" | "oldest_first" | "random" — only used in REGULAR mode
}

// StartLearnSessionResponse is returned immediately after session creation.
// Includes the first question and the student's starting theta so the UI
// can display their entry difficulty level.
type StartLearnSessionResponse struct {
	SessionID           string            `json:"session_id"`
	Mode                string            `json:"mode"`
	Chapter             string            `json:"chapter"`
	ThetaStart          float64           `json:"theta_start"`            // where the session ladder begins
	FirstQuestion       QuestionPayload   `json:"first_question"`         // answer-free
	QuestionMap         []QuestionMapItem `json:"question_map,omitempty"` // all scope questions for REGULAR mode map
	QuestionOrdering    string            `json:"question_ordering"`      // only relevant for REGULAR mode
	BiometricEnabled    bool              `json:"biometric_enabled"`
	TotalScopeQuestions int               `json:"total_scope_questions"` // total questions matching scope
}

// QuestionMapItem is a lightweight reference for the question map in REGULAR mode.
type QuestionMapItem struct {
	ID   string `json:"id"`
	Year string `json:"year"` // extracted from shift_date, e.g. "2026"
	Seen bool   `json:"seen"` // whether user has attempted this question in any previous session
}

// LearnSubmitRequest is sent when the student submits an answer.
// TimeTakenMs is measured client-side from when the question was displayed.
type LearnSubmitRequest struct {
	QuestionID      string   `json:"question_id"`
	SelectedOptions []string `json:"selected_options"` // ["A"] or ["A","C"] or ["105.5"]
	TimeTakenMs     int64    `json:"time_taken_ms"`
}

// LearnSkipRequest is sent when the student skips a question without answering.
type LearnSkipRequest struct {
	QuestionID  string `json:"question_id"`
	TimeTakenMs int64  `json:"time_taken_ms"`
}

// LearnSkipResponse is returned after a skip.
type LearnSkipResponse struct {
	Skipped      bool             `json:"skipped"`
	ThetaBefore  float64          `json:"theta_before"`
	ThetaAfter   float64          `json:"theta_after"` // same as theta_before (no change)
	NextQuestion *QuestionPayload `json:"next_question"`
}

// LearnSubmitResponse is the richest response in the system.
// Because this is learn mode, the student sees the answer immediately.
//
// CorrectOptions and Explanation are ONLY here — in no other response type.
// NextQuestion is nil when the session has no more unseen questions.
type LearnSubmitResponse struct {
	// Correctness feedback — shown immediately
	IsCorrect      bool     `json:"is_correct"`
	CorrectOptions []string `json:"correct_options"` // e.g. ["A", "C"] or ["105.4TO105.6"]
	Explanation    string   `json:"explanation"`     // full solution text, Markdown/LaTeX

	// Theta movement — lets the UI animate the difficulty ladder
	ThetaBefore float64 `json:"theta_before"`
	ThetaAfter  float64 `json:"theta_after"`

	// The next question to display. Nil signals session completion.
	// Answer-free — same QuestionPayload type as everywhere else.
	NextQuestion *QuestionPayload `json:"next_question"`
}

// CloseLearnSessionResponse is returned when the student explicitly ends a session.
// Summary stats for the end-of-session screen.
type CloseLearnSessionResponse struct {
	SessionID      string  `json:"session_id"`
	Chapter        string  `json:"chapter"`
	ThetaStart     float64 `json:"theta_start"`
	ThetaFinal     float64 `json:"theta_final"`
	MasteryScore   float64 `json:"mastery_score"` // normalised 0-100 from theta_final
	TotalQuestions int     `json:"total_questions"`
	CorrectCount   int     `json:"correct_count"`
	Accuracy       float64 `json:"accuracy"` // 0.0 to 1.0
	AvgTimeTakenMs int64   `json:"avg_time_taken_ms"`
}

// GetSessionResponse is returned when the client resumes an active session
// after a page refresh. Contains the current question, session metadata,
// and the full history of answered questions so accuracy can be restored.
type GetSessionResponse struct {
	SessionID           string                `json:"session_id"`
	Mode                string                `json:"mode"`
	Chapter             string                `json:"chapter"`
	ThetaStart          float64               `json:"theta_start"`
	ThetaCurrent        float64               `json:"theta_current"`
	QuestionCount       int                   `json:"question_count"`
	CurrentQuestion     *QuestionPayload      `json:"current_question"`
	Responses           []ResponseHistoryItem `json:"responses"`              // past answered questions, newest last
	QuestionMap         []QuestionMapItem     `json:"question_map,omitempty"` // all scope questions for REGULAR mode map
	QuestionOrdering    string                `json:"question_ordering"`
	QuestionLimit       int                   `json:"question_limit"`        // max questions for this session
	BiometricEnabled    bool                  `json:"biometric_enabled"`
	TotalScopeQuestions int                   `json:"total_scope_questions"` // total questions matching scope
}

// ResponseHistoryItem reconstructs one answered question for the frontend
// so it can display review history after a page refresh.
type ResponseHistoryItem struct {
	Question        QuestionPayload `json:"question"`
	IsCorrect       bool            `json:"is_correct"`
	Skipped         bool            `json:"skipped"`
	SelectedOptions []string        `json:"selected_options"`
	CorrectOptions  []string        `json:"correct_options"`
}

// ═══════════════════════════════════════════════════════════════════════════
// GRAPH API
// ═══════════════════════════════════════════════════════════════════════════

// GraphNode is one chapter node in the knowledge graph response.
// Visual properties are computed server-side; the frontend renders directly.
type GraphNode struct {
	ID             string  `json:"id"`              // slug: "electrostatics"
	Chapter        string  `json:"chapter"`         // display: "Electrostatics"
	Subject        string  `json:"subject"`         // "physics" | "chemistry" | "maths"
	Group          string  `json:"group"`           // "electricity"
	MasteryScore   float64 `json:"mastery_score"`   // 0-100 → drives node size
	GlickoRD       float64 `json:"glicko_rd"`       // → drives node opacity/glow
	Theta          float64 `json:"theta"`           // raw IRT value
	LastSeen       string  `json:"last_seen"`       // ISO8601 or empty string
	TotalQuestions int     `json:"total_questions"` // total questions in chapter
	VeryEasyCount  int     `json:"very_easy_count"` // glicko_rating < 1350
	EasyCount      int     `json:"easy_count"`      // 1350-1449
	MediumCount    int     `json:"medium_count"`    // 1450-1549
	HardCount      int     `json:"hard_count"`      // 1550-1699
	VeryHardCount  int     `json:"very_hard_count"` // >= 1700
}

// GraphEdge is one prerequisite relationship between two chapters.
type GraphEdge struct {
	From           string  `json:"from"`            // "physics/electrostatics"
	To             string  `json:"to"`              // "physics/current-electricity"
	StrainIndex    float64 `json:"strain_index"`    // prerequisite.mastery - child.mastery → edge length
	CrossSubject   bool    `json:"cross_subject"`   // true if edge spans subjects
	IsPrerequisite bool    `json:"is_prerequisite"` // true → rendered as dotted line in frontend
}

// GraphPayload is the full response for GET /api/v1/graph.
type GraphPayload struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// ═══════════════════════════════════════════════════════════════════════════
// AUTH / OIDC
// ═══════════════════════════════════════════════════════════════════════════

// AuthUserInfo is the user profile returned by the Alpha-Auth userinfo endpoint.
// Used by the callback handler to upsert users into the local database.
type AuthUserInfo struct {
	AlphaID  string `json:"alpha_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Profile  struct {
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
		Dob         string `json:"dob"`
	} `json:"profile"`
}

// TokenResponse is the OAuth2 token exchange response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// ═══════════════════════════════════════════════════════════════════════════
// LEARN PAGE
// ═══════════════════════════════════════════════════════════════════════════

// LearnChapterNode is a chapter entry for the /learn page.
// Extends the graph node with attempt stats needed for the chapter list.
type LearnChapterNode struct {
	ID              string   `json:"id"`
	Chapter         string   `json:"chapter"`
	Subject         string   `json:"subject"`
	Group           string   `json:"group"`
	MasteryScore    float64  `json:"mastery_score"`
	GlickoRD        float64  `json:"glicko_rd"`
	Theta           float64  `json:"theta"`
	LastPracticedAt string   `json:"last_practiced_at"` // ISO8601 or ""
	TotalAttempts   int      `json:"total_attempts"`
	CorrectAttempts int      `json:"correct_attempts"`
	TotalQuestions  int      `json:"total_questions"`
	VeryEasyCount   int      `json:"very_easy_count"`
	EasyCount       int      `json:"easy_count"`
	MediumCount     int      `json:"medium_count"`
	HardCount       int      `json:"hard_count"`
	VeryHardCount   int      `json:"very_hard_count"`
	ExamTypes       []string `json:"exam_types"` // exam types this chapter has questions for
}

// LastSessionInfo is the summary of the most recent completed session, used for "Resume Learning".
type LastSessionInfo struct {
	SessionID      string  `json:"session_id"`
	Chapter        string  `json:"chapter"`
	Mode           string  `json:"mode"`
	TotalQuestions int     `json:"total_questions"`
	CorrectCount   int     `json:"correct_count"`
	Accuracy       float64 `json:"accuracy"`
	CompletedAt    string  `json:"completed_at"` // ISO8601
}

// LearnPageResponse is the response for GET /api/v1/learn/page.
type LearnPageResponse struct {
	Chapters           []LearnChapterNode  `json:"chapters"`
	ActiveSession      *LastSessionInfo    `json:"active_session"` // non-null if there's an active (resumable) session
	LastSession        *LastSessionInfo    `json:"last_session"`   // most recent completed session, null if none
	DefaultExamType    string              `json:"default_exam_type"`
	AvailableExamTypes []string            `json:"available_exam_types"`
	SubjectsPerExam    map[string][]string `json:"subjects_per_exam"` // exam_type → subjects that have questions for it
}

// ═══════════════════════════════════════════════════════════════════════════
// DASHBOARD
// ═══════════════════════════════════════════════════════════════════════════

// DailyTrendEntry is one day's aggregated stats for the performance trend chart.
type DailyTrendEntry struct {
	Date    string `json:"date"`    // "2026-05-24"
	Correct int    `json:"correct"` // questions answered correctly
	Wrong   int    `json:"wrong"`   // questions answered incorrectly
	Total   int    `json:"total"`   // total questions attempted
}

// RecentActivityEntry is a compact summary of one completed session.
type RecentActivityEntry struct {
	SessionID      string  `json:"session_id"`
	Chapter        string  `json:"chapter"`
	Mode           string  `json:"mode"`
	TotalQuestions int     `json:"total_questions"`
	CorrectCount   int     `json:"correct_count"`
	Accuracy       float64 `json:"accuracy"`
	DurationMs     int64   `json:"duration_ms"`
	CompletedAt    string  `json:"completed_at"` // ISO8601
}

// TrendChange shows how correct count changed vs the previous period.
// All values are in percentage points (e.g. +12.5 means 12.5% more correct).
type TrendChange struct {
	CorrectChange   float64 `json:"correct_change"`   // % change in correct count
	AccuracyChange  float64 `json:"accuracy_change"`  // percentage point change in accuracy
	CorrectThis     int     `json:"correct_this"`     // raw correct count this period
	CorrectPrevious int     `json:"correct_previous"` // raw correct count previous period
	TotalThis       int     `json:"total_this"`
	TotalPrevious   int     `json:"total_previous"`
}

// DashboardResponse is the single response for GET /api/v1/dashboard.
type DashboardResponse struct {
	QuestionsSolved  int                   `json:"questions_solved"`
	AverageAccuracy  float64               `json:"average_accuracy"`
	LearningRating   float64               `json:"learning_rating"`
	Rank             int                   `json:"rank"`
	TotalUsers       int                   `json:"total_users"`
	CurrentStreak    int                   `json:"current_streak"`
	PerformanceTrend []DailyTrendEntry     `json:"performance_trend"`
	TrendChange      *TrendChange          `json:"trend_change,omitempty"`
	RecentActivity   []RecentActivityEntry `json:"recent_activity"`
}

// ═══════════════════════════════════════════════════════════════════════════
// DEBUG / SESSION REPLAY
// ═══════════════════════════════════════════════════════════════════════════

type DebugSessionInfo struct {
	ID                string            `json:"id"`
	UserID            string            `json:"user_id"`
	UserEmail         string            `json:"user_email"`
	Mode              string            `json:"mode"`
	Chapter           string            `json:"chapter"`
	ThetaStart        float64           `json:"theta_start"`
	ThetaCurrent      float64           `json:"theta_current"`
	QuestionCount     int               `json:"question_count"`
	CurrentQuestionID *string           `json:"current_question_id"`
	Responses         []SessionResponse `json:"responses"`
	Status            string            `json:"status"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════════
// SESSION ANALYSIS / HISTORY
// ═══════════════════════════════════════════════════════════════════════════

// SessionQuestionAnalysis is one question's result within a completed session.
type SessionQuestionAnalysis struct {
	QuestionID      string          `json:"question_id"`
	Question        QuestionPayload `json:"question"`
	SelectedOptions []string        `json:"selected_options"`
	CorrectOptions  []string        `json:"correct_options"`
	IsCorrect       bool            `json:"is_correct"`
	Skipped         bool            `json:"skipped"`
	TimeTakenMs     int64           `json:"time_taken_ms"`
	ThetaBefore     float64         `json:"theta_before"`
	ThetaAfter      float64         `json:"theta_after"`
	SubmittedAt     time.Time       `json:"submitted_at"`
}

// SessionAnalysisResponse is the full breakdown of a completed session.
// Returned by GET /api/v1/learn/session/analysis?session_id=X
type SessionAnalysisResponse struct {
	SessionID        string    `json:"session_id"`
	Mode             string    `json:"mode"`
	Chapter          string    `json:"chapter"`
	Status           string    `json:"status"`
	ThetaStart       float64   `json:"theta_start"`
	ThetaFinal       float64   `json:"theta_final"`
	MasteryScore     float64   `json:"mastery_score"`
	CreatedAt        time.Time `json:"created_at"`
	CompletedAt      time.Time `json:"completed_at"`
	DurationMs       int64     `json:"duration_ms"` // wall-clock time from created_at to completed_at
	BiometricEnabled bool      `json:"biometric_enabled"`

	TotalQuestions int     `json:"total_questions"`
	CorrectCount   int     `json:"correct_count"`
	WrongCount     int     `json:"wrong_count"`
	SkippedCount   int     `json:"skipped_count"`
	Accuracy       float64 `json:"accuracy"` // excluding skips

	AvgTimeTakenMs int64 `json:"avg_time_taken_ms"`
	TotalTimeMs    int64 `json:"total_time_ms"`
	FastestTimeMs  int64 `json:"fastest_time_ms"`
	SlowestTimeMs  int64 `json:"slowest_time_ms"`

	Questions []SessionQuestionAnalysis `json:"questions"`
}

// SessionHistoryItem is one session entry in the history list.
type SessionHistoryItem struct {
	SessionID        string    `json:"session_id"`
	Chapter          string    `json:"chapter"`
	Mode             string    `json:"mode"`
	Status           string    `json:"status"`
	TotalQuestions   int       `json:"total_questions"`
	CorrectCount     int       `json:"correct_count"`
	Accuracy         float64   `json:"accuracy"`
	MasteryScore     float64   `json:"mastery_score"`
	DurationMs       int64     `json:"duration_ms"`
	CreatedAt        time.Time `json:"created_at"`
	CompletedAt      time.Time `json:"completed_at"`
	BiometricEnabled bool      `json:"biometric_enabled"`
}

// SessionHistoryResponse is the list of past sessions.
// Returned by GET /api/v1/learn/history
type SessionHistoryResponse struct {
	Sessions []SessionHistoryItem `json:"sessions"`
}

// ═══════════════════════════════════════════════════════════════════════════
// PUBLIC USER PROFILE
// ═══════════════════════════════════════════════════════════════════════════

// RDFromFloat32 converts a float32 RD to a confidence string.
func RDFromFloat32(rd float32) string {
	if rd < 60 {
		return "low"
	} else if rd < 100 {
		return "medium"
	}
	return "high"
}

// DisplayRating is a safe public view of a user's Glicko rating.
type DisplayRating struct {
	Rating    float64 `json:"rating"`
	RD        float64 `json:"rd"`
	Deviation string  `json:"deviation"` // "low" | "medium" | "high" — confidence based on RD
	Rank      int     `json:"rank"`      // leaderboard rank, 0 = unranked
}

// PublicProfileResponse is the public profile visible at /u/{id}.
type PublicProfileResponse struct {
	ID                string            `json:"id"`
	Username          string            `json:"username"`
	MemberSince       string            `json:"member_since"` // ISO8601
	Bio               string            `json:"bio"`
	LearnRating       DisplayRating     `json:"learn_rating"`
	ContestRating     DisplayRating     `json:"contest_rating"`
	MockRating        DisplayRating     `json:"mock_rating"`
	Accuracy          float64           `json:"accuracy"` // 0-1
	QuestionsSolved   int               `json:"questions_solved"`
	TotalSessions     int               `json:"total_sessions"`
	SolveVelocity     float64           `json:"solve_velocity"`     // questions/hr
	Carelessness      float64           `json:"carelessness"`       // 0-1
	SocialConnections map[string]string `json:"social_connections"` // non-empty connections only
}

// ═══════════════════════════════════════════════════════════════════════════
// COMMON ERROR RESPONSE
// ═══════════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════════
// USER SETTINGS
// ═══════════════════════════════════════════════════════════════════════════

// UserSettingsDTO is the public settings object sent to/from the frontend.
type UserSettingsDTO struct {
	Username          string                `json:"username"`
	Email             string                `json:"email"`
	DefaultExamType   string                `json:"default_exam_type"`
	Profile           ProfileSettingsDTO    `json:"profile"`
	Notifications     NotificationsSettings `json:"notifications"`
	Beta              BetaSettings          `json:"beta"`
	SocialConnections SocialConnections     `json:"social_connections"`
}

type ProfileSettingsDTO struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
}

// ErrorResponse is the standard JSON error envelope used across all endpoints.
// The Code field is a machine-readable string for the frontend to act on.
type ErrorResponse struct {
	Code    string `json:"code"`    // e.g. "SESSION_NOT_FOUND", "INVALID_QUESTION"
	Message string `json:"message"` // human-readable, shown to user
}

// ═══════════════════════════════════════════════════════════════════════════
// BIOMETRIC TELEMETRY DTOs
// ═══════════════════════════════════════════════════════════════════════════

// BiometricSyncRequest is a single telemetry frame sent from the client.
type BiometricSyncRequest struct {
	SessionID          string  `json:"session_id"`
	TS                 int64   `json:"ts"` // ms since session start
	EARAvg             float64 `json:"ear_avg"`
	PERCLOS            float64 `json:"perclos"`
	BlinkRate          float64 `json:"blink_rate"`
	MAR                float64 `json:"mar"`
	HeadPitch          float64 `json:"head_pitch"`
	HeadRoll           float64 `json:"head_roll"`
	HeadMoveVariance   float64 `json:"head_move_variance"`
	BrowFurrow         float64 `json:"brow_furrow"`
	FatigueScore       float64 `json:"fatigue_score"`
	DistractionScore   float64 `json:"distraction_score"`
	CognitiveLoadScore float64 `json:"cognitive_load_score"`
	GazeZone           string  `json:"gaze_zone"`
	InFlowState        bool    `json:"in_flow_state"`
	FaceAbsent         bool    `json:"face_absent"`
}

// BiometricSyncBatchRequest allows the client to flush multiple frames at once.
type BiometricSyncBatchRequest struct {
	SessionID string                 `json:"session_id"`
	Snapshots []BiometricSyncRequest `json:"snapshots"`
}

// BiometricCloseResponse is returned after closing a biometric session.
type BiometricCloseResponse struct {
	Summary    BiometricSummaryDTO `json:"summary"`
	DNAUpdated bool                `json:"dna_updated"`
}

// BiometricSummaryDTO is the public summary of a biometric session.
type BiometricSummaryDTO struct {
	AvgFatigue       float64 `json:"avg_fatigue"`
	AvgDistraction   float64 `json:"avg_distraction"`
	AvgCognitiveLoad float64 `json:"avg_cognitive_load"`
	PeakFatigue      float64 `json:"peak_fatigue"`
	TimeOffTaskPct   float64 `json:"time_off_task_pct"`
	WritingPct       float64 `json:"writing_pct"`
	FaceAbsentPct    float64 `json:"face_absent_pct"`
	DurationS        float64 `json:"duration_s"`
	SampleCount      int     `json:"sample_count"`
}

// BiometricSessionDTO is returned by GET /biometrics/{sessionId}.
type BiometricSessionDTO struct {
	Enabled  bool                   `json:"enabled"`
	Baseline *BiometricBaselineDTO  `json:"baseline,omitempty"`
	Logs     []BiometricSyncRequest `json:"logs,omitempty"`
	Summary  *BiometricSummaryDTO   `json:"summary,omitempty"`
}

// BiometricBaselineDTO is the public view of the calibration baseline.
// Type alias — identical to the internal BiometricBaseline (same JSON tags).
type BiometricBaselineDTO = BiometricBaseline

// ═══════════════════════════════════════════════════════════════════════════
// DPP PREVIEW / DISPLAY
// ═══════════════════════════════════════════════════════════════════════════

// DPPQuestionPreviewDTO is the question data returned by the DPP preview endpoint.
// Unlike QuestionPayload, this includes glicko_rating for difficulty tagging
// and options in parseable form (not stripped to DTO).
type DPPQuestionPreviewDTO struct {
	ID             string              `json:"id"`
	Type           string              `json:"type"`
	QuestionText   string              `json:"question_text"`
	Subject        string              `json:"subject"`
	Chapter        string              `json:"chapter"`
	ChapterGroup   string              `json:"chapter_group"`
	Difficulty     string              `json:"difficulty"`
	GlickoRating   float32             `json:"glicko_rating"`
	Options        []QuestionOptionDTO `json:"options"`
	Images         []string            `json:"images"`
	Source         string              `json:"source"`
	ShiftDate      string              `json:"shift_date"`
	ExamType       string              `json:"exam_type"`
	RatingTag      string              `json:"rating_tag"`
	AttemptCount   int                 `json:"attempt_count"`
	PercentCorrect int                 `json:"percent_correct"`
}
