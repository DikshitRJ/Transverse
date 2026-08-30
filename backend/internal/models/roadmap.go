package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RoadmapSource string

const (
	RoadmapSourceLLMGenerated RoadmapSource = "llm_generated"
	RoadmapSourceCurated      RoadmapSource = "curated"
)

type RoadmapStatus string

const (
	RoadmapStatusActive    RoadmapStatus = "active"
	RoadmapStatusCompleted RoadmapStatus = "completed"
	RoadmapStatusAbandoned RoadmapStatus = "abandoned"
)

type NodeStatus string

const (
	NodeStatusLocked     NodeStatus = "locked"
	NodeStatusUnlocked   NodeStatus = "unlocked"
	NodeStatusInProgress NodeStatus = "in_progress"
	NodeStatusMastered   NodeStatus = "mastered"
	NodeStatusTestedOut  NodeStatus = "tested_out"
)

type UnlockRule struct {
	Type      string  `json:"type"` // "no_prerequisite", "mastery_threshold", "phase_complete", "quiz_pass"
	TopicID   string  `json:"topic_id,omitempty"`
	MinRating float64 `json:"min_rating,omitempty"`
	PhaseID   string  `json:"phase_id,omitempty"`
}

type RoadmapTemplate struct {
	ID         uuid.UUID     `json:"id"`
	TargetRole string        `json:"target_role"`
	Source     RoadmapSource `json:"source"`
	Version    int           `json:"version"`
	CreatedAt  time.Time     `json:"created_at"`
}

type RoadmapPhase struct {
	ID                uuid.UUID       `json:"id"`
	RoadmapTemplateID uuid.UUID       `json:"roadmap_template_id"`
	Sequence          int             `json:"sequence"`
	Title             string          `json:"title"`
	UnlockRule        json.RawMessage `json:"unlock_rule"`
	CreatedAt         time.Time       `json:"created_at"`
}

type RoadmapNode struct {
	ID               uuid.UUID       `json:"id"`
	PhaseID          uuid.UUID       `json:"phase_id"`
	TopicID          string          `json:"topic_id"`
	Sequence         int             `json:"sequence"`
	UnlockRule       json.RawMessage `json:"unlock_rule"`
	TutorialIDs      []uuid.UUID     `json:"tutorial_ids"`
	PracticeTopicIDs []string        `json:"practice_topic_ids"`
	CreatedAt        time.Time       `json:"created_at"`
}

type UserRoadmap struct {
	ID                uuid.UUID     `json:"id"`
	UserID            uuid.UUID     `json:"user_id"`
	RoadmapTemplateID uuid.UUID     `json:"roadmap_template_id"`
	Status            RoadmapStatus `json:"status"`
	CurrentPhaseID    *uuid.UUID    `json:"current_phase_id"`
	CreatedAt         time.Time     `json:"created_at"`
}

type UserRoadmapNodeProgress struct {
	ID            uuid.UUID  `json:"id"`
	UserRoadmapID uuid.UUID  `json:"user_roadmap_id"`
	NodeID        uuid.UUID  `json:"node_id"`
	Status        NodeStatus `json:"status"`
	UnlockedAt    *time.Time `json:"unlocked_at,omitempty"`
	MasteredAt    *time.Time `json:"mastered_at,omitempty"`
}

// Tutorial represents a learning resource (article, video, interactive lesson).
type Tutorial struct {
	ID               uuid.UUID `json:"id"`
	Source           string    `json:"source"`
	SourceURL        string    `json:"source_url"`
	Title            string    `json:"title"`
	TopicID          *string   `json:"topic_id,omitempty"`
	TopicTags        []string  `json:"topic_tags"`
	Type             string    `json:"type"`       // "article" | "video" | "interactive" | "playlist"
	Difficulty       string    `json:"difficulty"` // "beginner" | "intermediate" | "advanced"
	EstimatedMinutes int       `json:"estimated_minutes"`
	Summary          string    `json:"summary"`
	LicenseNote      string    `json:"license_note,omitempty"`
	ThumbnailURL     string    `json:"thumbnail_url,omitempty"`
	Status           string    `json:"status"` // "COMPLETED" | "UNREAD"
}

// RoadmapSubsection represents a single topic / node inside a roadmap section.
type RoadmapSubsection struct {
	NodeID       uuid.UUID        `json:"node_id"`
	TopicID      string           `json:"topic_id"`
	Title        string           `json:"title"`
	Sequence     int              `json:"sequence"`
	Status       NodeStatus       `json:"status"` // "locked", "unlocked", "in_progress", "mastered", "tested_out"
	UserRating   float64          `json:"user_rating"`
	TargetRating float64          `json:"target_rating"`
	MasteryScore float64          `json:"mastery_score"`
	Tutorials    []Tutorial       `json:"tutorials"`
	Questions    []ProblemPayload `json:"questions"`
}

// RoadmapSection represents a single progressive section (phase) visible to the user.
type RoadmapSection struct {
	PhaseID            uuid.UUID           `json:"phase_id"`
	Sequence           int                 `json:"sequence"`
	Title              string              `json:"title"`
	Status             string              `json:"status"` // "ACTIVE", "COMPLETED", "LOCKED"
	ProgressPercentage float64             `json:"progress_percentage"`
	Subsections        []RoadmapSubsection `json:"subsections"`
}

// UpcomingSectionPreview represents minimal metadata for locked upcoming sections.
type UpcomingSectionPreview struct {
	Sequence int    `json:"sequence"`
	Title    string `json:"title"`
	Status   string `json:"status"` // "LOCKED"
}

// RoadmapCurrentResponse is the primary dynamic roadmap payload fetched by the frontend.
// Only the current active section is populated with full tutorials and questions;
// subsequent sections are locked until the current section is mastered.
type RoadmapCurrentResponse struct {
	RoadmapID        uuid.UUID                `json:"roadmap_id"`
	UserID           string                   `json:"user_id"`
	UserRating       float64                  `json:"user_rating"`
	TargetRole       string                   `json:"target_role"`
	Status           RoadmapStatus            `json:"status"`
	TotalSections    int                      `json:"total_sections"`
	CurrentSection   *RoadmapSection          `json:"current_section"`
	UpcomingSections []UpcomingSectionPreview `json:"upcoming_sections"`
}

type RoadmapFull struct {
	Template RoadmapTemplate    `json:"template"`
	Phases   []RoadmapPhaseFull `json:"phases"`
}

type RoadmapPhaseFull struct {
	Phase RoadmapPhase      `json:"phase"`
	Nodes []RoadmapNodeFull `json:"nodes"`
}

type RoadmapNodeFull struct {
	Node     RoadmapNode              `json:"node"`
	Progress *UserRoadmapNodeProgress `json:"progress,omitempty"`
}

type CompleteNodeRequest struct {
	NodeID uuid.UUID `json:"node_id"`
}

type TestOutRequest struct {
	NodeID uuid.UUID `json:"node_id"`
}
