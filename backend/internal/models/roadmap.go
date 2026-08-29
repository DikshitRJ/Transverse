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
	Type      string  `json:"type"` // e.g. "no_prerequisite", "mastery_threshold", "phase_complete", "quiz_pass"
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
	ID               uuid.UUID   `json:"id"`
	PhaseID          uuid.UUID   `json:"phase_id"`
	TopicID          string          `json:"topic_id"`
	Sequence         int             `json:"sequence"`
	UnlockRule       json.RawMessage `json:"unlock_rule"`
	TutorialIDs      []uuid.UUID     `json:"tutorial_ids"`
	PracticeTopicIDs []string    `json:"practice_topic_ids"`
	CreatedAt        time.Time   `json:"created_at"`
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

type RoadmapFull struct {
	Template RoadmapTemplate     `json:"template"`
	Phases   []RoadmapPhaseFull  `json:"phases"`
}

type RoadmapPhaseFull struct {
	Phase RoadmapPhase      `json:"phase"`
	Nodes []RoadmapNodeFull `json:"nodes"`
}

type RoadmapNodeFull struct {
	Node     RoadmapNode              `json:"node"`
	Progress *UserRoadmapNodeProgress `json:"progress,omitempty"` // For user specific views
}
