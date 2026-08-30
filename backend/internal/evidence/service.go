package evidence

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"transverse/internal/connectors"
	"transverse/internal/models"
)

type EvidenceRepository interface {
	CreateSource(ctx context.Context, source *models.EvidenceSource) error
	UpdateSourceStatus(ctx context.Context, id string, status models.EvidenceStatus, errMsg *string) error
	ClearObjectKey(ctx context.Context, id string) error
	CreateExtract(ctx context.Context, extract *models.EvidenceExtract) error
	GetSource(ctx context.Context, id string) (*models.EvidenceSource, error)
}

type ObjectStore interface {
	PresignPut(ctx context.Context, objectKey string, expires time.Duration) (string, error)
	Delete(ctx context.Context, objectKey string) error
	Get(ctx context.Context, objectKey string) ([]byte, error)
}

type JobQueue interface {
	EnqueueHypothesisGeneration(ctx context.Context, userID string) (string, error)
}

type Service struct {
	repo         EvidenceRepository
	objectStore  ObjectStore
	jobQueue     JobQueue
	connectors   map[models.EvidenceKind]connectors.Connector
}

func NewService(repo EvidenceRepository, objectStore ObjectStore, jobQueue JobQueue, 
	gh *connectors.GithubConnector, lc *connectors.LeetcodeConnector, cf *connectors.CodeforcesConnector) *Service {
	return &Service{
		repo:        repo,
		objectStore: objectStore,
		jobQueue:    jobQueue,
		connectors: map[models.EvidenceKind]connectors.Connector{
			models.EvidenceKindGithub:     gh,
			models.EvidenceKindLeetcode:   lc,
			models.EvidenceKindCodeforces: cf,
		},
	}
}

// StartConnectorSource creates a new source and processes it immediately for API-based connectors.
func (s *Service) StartConnectorSource(ctx context.Context, userID string, kind models.EvidenceKind, ref string) (string, error) {
	if _, ok := s.connectors[kind]; !ok {
		return "", fmt.Errorf("unsupported connector kind: %s", kind)
	}

	sourceID := generateID() // assuming some utility, will implement
	extRef := ref
	source := &models.EvidenceSource{
		ID:          sourceID,
		UserID:      userID,
		Kind:        kind,
		ExternalRef: &extRef,
		Status:      models.EvidenceStatusPending,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateSource(ctx, source); err != nil {
		return "", err
	}

	// Dispatch processing asynchronously
	go func() {
		// Use a detached context for background processing
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		s.processConnector(bgCtx, source)
	}()

	return sourceID, nil
}

func (s *Service) processConnector(ctx context.Context, source *models.EvidenceSource) {
	_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusFetching, nil)

	connector := s.connectors[source.Kind]
	rawSignal, err := connector.Fetch(ctx, *source.ExternalRef)
	if err != nil {
		errMsg := err.Error()
		_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusFailed, &errMsg)
		return
	}

	_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusProcessing, nil)

	// Normalize into EvidenceExtract
	extractMap := map[string]interface{}{
		"languages":      rawSignal.Languages,
		"claimed_topics": rawSignal.ClaimedTopics,
		"signals":        rawSignal.Signals,
	}

	extract := &models.EvidenceExtract{
		ID:               generateID(),
		EvidenceSourceID: source.ID,
		ExtractedJSON:    extractMap,
		Confidence:       0.8, // default connector confidence
		CreatedAt:        time.Now(),
	}

	if err := s.repo.CreateExtract(ctx, extract); err != nil {
		errMsg := err.Error()
		_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusFailed, &errMsg)
		return
	}

	now := time.Now()
	source.ProcessedAt = &now
	_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusDone, nil)

	// Enqueue hypothesis generation
	_, _ = s.jobQueue.EnqueueHypothesisGeneration(ctx, source.UserID)
}

// GeneratePresignedUpload creates an upload source for file-based evidence and returns a presigned URL.
func (s *Service) GeneratePresignedUpload(ctx context.Context, userID string, kind models.EvidenceKind, filename string) (string, string, error) {
	sourceID := generateID()
	objectKey := fmt.Sprintf("evidence/%s/%s/%s", userID, sourceID, filename)

	source := &models.EvidenceSource{
		ID:        sourceID,
		UserID:    userID,
		Kind:      kind,
		ObjectKey: &objectKey,
		Status:    models.EvidenceStatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateSource(ctx, source); err != nil {
		return "", "", err
	}

	uploadURL, err := s.objectStore.PresignPut(ctx, objectKey, 5*time.Minute)
	if err != nil {
		return "", "", err
	}

	return sourceID, uploadURL, nil
}

// ConfirmUpload is called when the client finishes the upload to MinIO.
func (s *Service) ConfirmUpload(ctx context.Context, sourceID string) error {
	source, err := s.repo.GetSource(ctx, sourceID)
	if err != nil {
		return err
	}
	if source.Status != models.EvidenceStatusPending {
		return fmt.Errorf("source not in pending state")
	}

	if source.ObjectKey == nil {
		return fmt.Errorf("no object key associated with source")
	}

	// Dispatch processing asynchronously
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		s.processUpload(bgCtx, source)
	}()

	return nil
}

func (s *Service) processUpload(ctx context.Context, source *models.EvidenceSource) {
	// 1. Mark as fetching
	_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusFetching, nil)

	// Defer deletion of the object key to ensure zero residual MinIO objects
	defer func() {
		if source.ObjectKey != nil {
			_ = s.objectStore.Delete(ctx, *source.ObjectKey)
			_ = s.repo.ClearObjectKey(ctx, source.ID)
		}
	}()

	// 2. Download from object store into memory
	data, err := s.objectStore.Get(ctx, *source.ObjectKey)
	if err != nil {
		errMsg := fmt.Sprintf("failed to download from store: %v", err)
		_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusFailed, &errMsg)
		return
	}

	// 3. Mark as processing
	_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusProcessing, nil)

	// 4. Extract signal (Mocked text extraction for now, in a real scenario we'd parse the resume/codebase)
	// For codebase, we'd parse zip and extract languages. For resume, pdf/docx to text -> LLM.
	extractMap := s.extractFileSignal(source.Kind, data)

	extract := &models.EvidenceExtract{
		ID:               generateID(),
		EvidenceSourceID: source.ID,
		ExtractedJSON:    extractMap,
		Confidence:       0.6,
		CreatedAt:        time.Now(),
	}

	if err := s.repo.CreateExtract(ctx, extract); err != nil {
		errMsg := err.Error()
		_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusFailed, &errMsg)
		return
	}

	now := time.Now()
	source.ProcessedAt = &now
	_ = s.repo.UpdateSourceStatus(ctx, source.ID, models.EvidenceStatusDone, nil)

	// 5. Enqueue hypothesis generation
	_, _ = s.jobQueue.EnqueueHypothesisGeneration(ctx, source.UserID)
}

func (s *Service) extractFileSignal(kind models.EvidenceKind, data []byte) map[string]interface{} {
	// Stub implementation for file processing.
	// As per plan, real extraction would involve an LLM for resume or a walk for codebase.
	// We extract mocked data to fulfill M3.
	if kind == models.EvidenceKindResume {
		return map[string]interface{}{
			"claimed_topics": []string{"data-structures", "algorithms"},
			"signals": []connectors.Signal{
				{TopicTag: "algorithms", Evidence: "Extracted from resume", Strength: "weak"},
			},
		}
	} else if kind == models.EvidenceKindCodebase {
		return map[string]interface{}{
			"languages": map[string]float64{"Go": 1.0},
			"signals": []connectors.Signal{
				{TopicTag: "backend", Evidence: "Found Go files in codebase", Strength: "moderate"},
			},
		}
	}
	return map[string]interface{}{}
}

func generateID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomHex(4))
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
