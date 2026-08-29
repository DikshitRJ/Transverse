package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
)

type Question struct {
	ID             string
	Type           string
	QuestionText   string
	OptionsRaw     json.RawMessage
	ImagesRaw      json.RawMessage
	Correct        string
	Subject        string
	Chapter        string
	ChapterGroup   string
	Difficulty     string
	ShiftDate      string
	Source         string
	ExamType       string
	GlickoRating   float32
	GlickoRD       float32
	TimespentAvgMs int32
	PercentCorrect int
	AttemptCount   int
	Embedding      pgvector.Vector
}

func (q *Question) CorrectOptions() []string {
	if q.Correct == "" {
		return nil
	}
	parts := strings.Split(q.Correct, ",")
	result := make([]string, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result
}

type QuestionOption struct {
	Key    string   `json:"key"`
	Text   string   `json:"text"`
	Images []string `json:"images"`
}

func (q *Question) Options() ([]QuestionOption, error) {
	if q.OptionsRaw == nil {
		return nil, fmt.Errorf("options not set")
	}
	var opts []QuestionOption
	if err := json.Unmarshal(q.OptionsRaw, &opts); err != nil {
		return nil, err
	}
	if opts == nil {
		opts = []QuestionOption{}
	}
	return opts, nil
}

func (q *Question) Images() ([]string, error) {
	if q.ImagesRaw == nil {
		return nil, fmt.Errorf("images not set")
	}
	var imgs []string
	if err := json.Unmarshal(q.ImagesRaw, &imgs); err != nil {
		return nil, err
	}
	return imgs, nil
}

type LearnSession struct {
	ID                   string
	UserID               string
	Mode                 string
	Chapter              string
	ScopeRaw             json.RawMessage
	ThetaStart           float32
	ThetaCurrent         float32
	QuestionCount        int
	CurrentQuestionID    *string
	ResponsesRaw         json.RawMessage
	Status               string
	QuestionLimit        int
	BiometricEnabled     bool
	BiometricLogsRaw     json.RawMessage
	BiometricBaselineRaw json.RawMessage
	QuestionOrdering     string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type SessionScope struct {
	Chapters      []string `json:"chapters"`
	ChapterGroups []string `json:"chapter_groups"`
	Subjects      []string `json:"subjects"`
	Years         []string `json:"years,omitempty"`
	ExamTypes     []string `json:"exam_types,omitempty"`
}

func (s *LearnSession) Scope() (SessionScope, error) {
	if s.ScopeRaw == nil || string(s.ScopeRaw) == "null" {
		return SessionScope{Chapters: []string{s.Chapter}}, nil
	}
	var scope SessionScope
	if err := json.Unmarshal(s.ScopeRaw, &scope); err != nil {
		return SessionScope{Chapters: []string{s.Chapter}}, nil
	}
	if len(scope.Chapters) == 0 {
		scope.Chapters = []string{s.Chapter}
	}
	return scope, nil
}

type SessionResponse struct {
	QuestionID          string    `json:"question_id"`
	SelectedOptions     []string  `json:"selected_options,omitempty"`
	IsCorrect           bool      `json:"is_correct"`
	Skipped             bool      `json:"skipped"`
	ThetaBefore         float64   `json:"theta_before"`
	ThetaAfter          float64   `json:"theta_after"`
	QuestionCount       int       `json:"question_count"`
	TimeTakenMs         int64     `json:"time_taken_ms"`
	SubmittedAt         time.Time `json:"submitted_at"`
	ScScore             float64   `json:"sc_score,omitempty"`
	DifficultyFit       float64   `json:"difficulty_fit,omitempty"`
	VectorSimilarity    float64   `json:"vector_similarity,omitempty"`
	TimeMatch           float64   `json:"time_match,omitempty"`
	NoveltyFactor       float64   `json:"novelty_factor,omitempty"`
	ImmediateReinforce  float64   `json:"immediate_reinforce,omitempty"`
	CarelessnessPenalty float64   `json:"carelessness_penalty,omitempty"`
	ThetaEffective      float64   `json:"theta_effective,omitempty"`
	Momentum            float64   `json:"momentum,omitempty"`
}

func (s *LearnSession) Responses() ([]SessionResponse, error) {
	if s.ResponsesRaw == nil {
		return nil, fmt.Errorf("responses not set")
	}
	var resp []SessionResponse
	if err := json.Unmarshal(s.ResponsesRaw, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		resp = []SessionResponse{}
	}
	return resp, nil
}

func (s *LearnSession) SeenQuestionIDs() (map[string]bool, error) {
	responses, err := s.Responses()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	for _, r := range responses {
		seen[r.QuestionID] = true
	}
	return seen, nil
}

type User struct {
	ID              string
	Username        string
	Email           string
	LearnRating     float32
	LearnRD         float32
	LearnVol        float32
	ContestRating   float32
	ContestRD       float32
	ContestVol      float32
	MockRating      float32
	MockRD          float32
	MockVol         float32
	LearningDNARaw  json.RawMessage
	BiometricDNARaw json.RawMessage
	SettingsRaw     json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type LearningDNA struct {
	AvgAccuracy          float64            `json:"avg_accuracy"`
	AvgTimeTakenMs       int64              `json:"avg_time_taken_ms"`
	AvgSolveVelocity     float64            `json:"avg_solve_velocity"`
	AvgFatigueTolerance  float64            `json:"avg_fatigue_tolerance"`
	CarelessnessIndex    float64            `json:"carelessness_index"`
	PeakPerformanceHour  int                `json:"peak_performance_hour"`
	AvgSessionLength     float64            `json:"avg_session_length"`
	TotalSessions        int                `json:"total_sessions"`
	TotalQuestionsSolved int                `json:"total_questions_solved"`
	SubjectBias          map[string]float64 `json:"subject_bias"`
}

func (u *User) DNA() (LearningDNA, error) {
	if u.LearningDNARaw == nil {
		return LearningDNA{}, nil
	}
	var dna LearningDNA
	if err := json.Unmarshal(u.LearningDNARaw, &dna); err != nil {
		return LearningDNA{}, err
	}
	return dna, nil
}

type BiometricDNA struct {
	Fatigue       WelfordState `json:"fatigue"`
	Distraction   WelfordState `json:"distraction"`
	CognitiveLoad WelfordState `json:"cognitive_load"`
}

func (u *User) BiometricDNA() (BiometricDNA, error) {
	if u.BiometricDNARaw == nil {
		return BiometricDNA{}, nil
	}
	var dna BiometricDNA
	if err := json.Unmarshal(u.BiometricDNARaw, &dna); err != nil {
		return BiometricDNA{}, err
	}
	return dna, nil
}

type WelfordState struct {
	Mean     float64 `json:"mean"`
	Variance float64 `json:"variance"`
	N        int     `json:"n"`
}

type GazeZone string

const (
	GazeZoneTask    GazeZone = "task"
	GazeZoneOffTask GazeZone = "off_task"
	GazeZoneWrite   GazeZone = "write"
	GazeZoneAbsent  GazeZone = "absent"
)

type BiometricSnapshot struct {
	TS                 int64    `json:"ts"`
	EARAvg             float64  `json:"ear_avg"`
	PERCLOS            float64  `json:"perclos"`
	BlinkRate          float64  `json:"blink_rate"`
	MAR                float64  `json:"mar"`
	HeadPitch          float64  `json:"head_pitch"`
	HeadRoll           float64  `json:"head_roll"`
	HeadMoveVariance   float64  `json:"head_move_variance"`
	BrowFurrow         float64  `json:"brow_furrow"`
	FatigueScore       float64  `json:"fatigue_score"`
	DistractionScore   float64  `json:"distraction_score"`
	CognitiveLoadScore float64  `json:"cognitive_load_score"`
	GazeZone           GazeZone `json:"gaze_zone"`
	InFlowState        bool     `json:"in_flow_state"`
	FaceAbsent         bool     `json:"face_absent"`
}

type BiometricBaseline struct {
	FatigueScore       float64 `json:"fatigue_score"`
	DistractionScore   float64 `json:"distraction_score"`
	CognitiveLoadScore float64 `json:"cognitive_load_score"`
}

type BiometricSummary struct {
	AvgFatigue       float64
	AvgDistraction   float64
	AvgCognitiveLoad float64
	PeakFatigue      float64
	TimeOffTaskPct   float64
	WritingPct       float64
	FaceAbsentPct    float64
	DurationS        float64
	SampleCount      int
}

func (s *LearnSession) BiometricLogs() ([]BiometricSnapshot, error) {
	if s.BiometricLogsRaw == nil {
		return nil, fmt.Errorf("biometric logs not set")
	}
	var logs []BiometricSnapshot
	if err := json.Unmarshal(s.BiometricLogsRaw, &logs); err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []BiometricSnapshot{}
	}
	return logs, nil
}

func (s *LearnSession) BiometricBaseline() (*BiometricBaseline, error) {
	if s.BiometricBaselineRaw == nil {
		return nil, nil
	}
	var b BiometricBaseline
	if err := json.Unmarshal(s.BiometricBaselineRaw, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

type UserSettings struct {
	DefaultExamType   string                `json:"default_exam_type"`
	Profile           ProfileSettings       `json:"profile"`
	Notifications     NotificationsSettings `json:"notifications"`
	Beta              BetaSettings          `json:"beta"`
	SocialConnections SocialConnections     `json:"social_connections"`
}

func DefaultUserSettings() UserSettings {
	return UserSettings{
		DefaultExamType: "JEE_MAIN",
		Profile: ProfileSettings{
			DisplayName: "",
			Bio:         "",
		},
		Notifications: NotificationsSettings{
			EmailNotifications: true,
			PushNotifications:  true,
			SessionReminders:   true,
			WeeklyReport:       true,
		},
		Beta: BetaSettings{
			CameraBasedAnalysis: false,
		},
		SocialConnections: SocialConnections{},
	}
}

func (u *User) Settings() (UserSettings, error) {
	defaults := DefaultUserSettings()
	if u.SettingsRaw == nil || string(u.SettingsRaw) == "null" {
		return defaults, nil
	}
	var s UserSettings
	if err := json.Unmarshal(u.SettingsRaw, &s); err != nil {
		return defaults, err
	}
	if s.Notifications == (NotificationsSettings{}) {
		s.Notifications = defaults.Notifications
	}
	if s.Profile == (ProfileSettings{}) {
		s.Profile = defaults.Profile
	}
	if s.DefaultExamType == "" {
		s.DefaultExamType = defaults.DefaultExamType
	}
	if s.SocialConnections == nil {
		s.SocialConnections = SocialConnections{}
	}
	return s, nil
}

type NotificationsSettings struct {
	EmailNotifications bool `json:"email_notifications"`
	PushNotifications  bool `json:"push_notifications"`
	SessionReminders   bool `json:"session_reminders"`
	WeeklyReport       bool `json:"weekly_report"`
}

type BetaSettings struct {
	CameraBasedAnalysis bool `json:"camera_based_analysis"`
}

type ProfileSettings struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
}

type SocialConnections map[string]string

type ChapterInfo struct {
	Chapter        string
	ChapterGroup   string
	Subject        string
	TotalQuestions int
	VeryEasyCount  int
	EasyCount      int
	MediumCount    int
	HardCount      int
	VeryHardCount  int
	ExamTypes      []string // which exam types have questions for this chapter
}

type ChapterStats struct {
	Theta           float64
	GlickoRating    float64
	GlickoRD        float64
	GlickoVol       float64
	MasteryScore    float64
	TotalAttempts   int
	CorrectAttempts int
	AvgTimeMs       int64
	SessionsCount   int
	LastPracticedAt time.Time
}

type LearningStats struct {
	UserID      string
	ChaptersRaw json.RawMessage
	LastSeenAt  *time.Time
	UpdatedAt   time.Time
}

func (ls *LearningStats) Chapters() (map[string]ChapterStats, error) {
	if ls.ChaptersRaw == nil {
		return map[string]ChapterStats{}, nil
	}
	var chapters map[string]ChapterStats
	if err := json.Unmarshal(ls.ChaptersRaw, &chapters); err != nil {
		return map[string]ChapterStats{}, nil
	}
	if chapters == nil {
		chapters = map[string]ChapterStats{}
	}
	return chapters, nil
}
