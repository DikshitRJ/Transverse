package models

import (
	"time"
)

type EvidenceKind string

const (
	EvidenceKindGithub     EvidenceKind = "github"
	EvidenceKindLeetcode   EvidenceKind = "leetcode"
	EvidenceKindCodeforces EvidenceKind = "codeforces"
	EvidenceKindResume     EvidenceKind = "resume"
	EvidenceKindCodebase   EvidenceKind = "codebase"
)

type EvidenceStatus string

const (
	EvidenceStatusPending    EvidenceStatus = "pending"
	EvidenceStatusFetching   EvidenceStatus = "fetching"
	EvidenceStatusProcessing EvidenceStatus = "processing"
	EvidenceStatusDone       EvidenceStatus = "done"
	EvidenceStatusFailed     EvidenceStatus = "failed"
	EvidenceStatusPurged     EvidenceStatus = "purged"
)

type EvidenceSource struct {
	ID           string
	UserID       string
	Kind         EvidenceKind
	ExternalRef  *string // Nullable for file uploads
	ObjectKey    *string // Nullable after purge
	Status       EvidenceStatus
	ErrorMessage *string
	CreatedAt    time.Time
	ProcessedAt  *time.Time
	PurgeAt      *time.Time
}

type EvidenceExtract struct {
	ID               string
	EvidenceSourceID string
	ExtractedJSON    map[string]interface{}
	Confidence       float64
	CreatedAt        time.Time
}
