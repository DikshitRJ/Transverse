package evidence

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"transverse/internal/config"
	"transverse/internal/connectors"
	"transverse/internal/models"
)

// MockRepo implements EvidenceRepository
type MockRepo struct {
	mu       sync.Mutex
	sources  map[string]*models.EvidenceSource
	extracts map[string]*models.EvidenceExtract
}

func NewMockRepo() *MockRepo {
	return &MockRepo{
		sources:  make(map[string]*models.EvidenceSource),
		extracts: make(map[string]*models.EvidenceExtract),
	}
}

func (m *MockRepo) CreateSource(ctx context.Context, source *models.EvidenceSource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sources[source.ID] = source
	return nil
}

func (m *MockRepo) UpdateSourceStatus(ctx context.Context, id string, status models.EvidenceStatus, errMsg *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if src, ok := m.sources[id]; ok {
		src.Status = status
		src.ErrorMessage = errMsg
	}
	return nil
}

func (m *MockRepo) ClearObjectKey(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if src, ok := m.sources[id]; ok {
		src.ObjectKey = nil
	}
	return nil
}

func (m *MockRepo) CreateExtract(ctx context.Context, extract *models.EvidenceExtract) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.extracts[extract.ID] = extract
	return nil
}

func (m *MockRepo) GetSource(ctx context.Context, id string) (*models.EvidenceSource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if src, ok := m.sources[id]; ok {
		return src, nil
	}
	return nil, fmt.Errorf("not found")
}

// MockObjectStore implements ObjectStore
type MockObjectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func NewMockObjectStore() *MockObjectStore {
	return &MockObjectStore{
		objects: make(map[string][]byte),
	}
}

func (m *MockObjectStore) PresignPut(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
	// Simulate an upload URL
	return "http://minio/mock-url/" + objectKey, nil
}

func (m *MockObjectStore) Delete(ctx context.Context, objectKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, objectKey)
	return nil
}

func (m *MockObjectStore) Get(ctx context.Context, objectKey string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if data, ok := m.objects[objectKey]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("object not found")
}

func (m *MockObjectStore) SimulateUpload(objectKey string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[objectKey] = data
}

// MockJobQueue implements JobQueue
type MockJobQueue struct{}

func (m *MockJobQueue) EnqueueHypothesisGeneration(ctx context.Context, userID string) (string, error) {
	return "job-123", nil
}

func TestEvidencePipeline(t *testing.T) {
	cfg := &config.Config{
		GithubAPIBase:            "https://api.github.com",
		LeetcodeGraphQLURL:       "https://leetcode.com/graphql",
		CodeforcesAPIBase:        "https://codeforces.com/api",
		ConnectorTimeoutSeconds:  1,
		ConnectorMaxReposScanned: 1,
	}

	repo := NewMockRepo()
	store := NewMockObjectStore()
	queue := &MockJobQueue{}

	gh := connectors.NewGithubConnector(cfg)
	lc := connectors.NewLeetcodeConnector(cfg)
	cf := connectors.NewCodeforcesConnector(cfg)

	svc := NewService(repo, store, queue, gh, lc, cf)

	ctx := context.Background()
	userID := "user-123"

	// 1. Test Resume Upload
	srcID, _, err := svc.GeneratePresignedUpload(ctx, userID, models.EvidenceKindResume, "resume.pdf")
	if err != nil {
		t.Fatalf("GeneratePresignedUpload failed: %v", err)
	}
	
	// Simulate client upload
	src, _ := repo.GetSource(ctx, srcID)
	store.SimulateUpload(*src.ObjectKey, []byte("mock pdf content"))

	// Confirm upload
	err = svc.ConfirmUpload(ctx, srcID)
	if err != nil {
		t.Fatalf("ConfirmUpload failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // Wait for async process

	// Verify resume object deleted and extract created
	src, _ = repo.GetSource(ctx, srcID)
	if src.ObjectKey != nil {
		t.Errorf("Resume object key not cleared from DB")
	}
	if len(store.objects) != 0 {
		t.Errorf("Resume MinIO object not deleted")
	}
	if src.Status != models.EvidenceStatusDone {
		t.Errorf("Resume source status not done")
	}

	// 2. Test Codebase Upload
	srcID2, _, err := svc.GeneratePresignedUpload(ctx, userID, models.EvidenceKindCodebase, "code.zip")
	if err != nil {
		t.Fatalf("GeneratePresignedUpload failed: %v", err)
	}
	
	src2, _ := repo.GetSource(ctx, srcID2)
	store.SimulateUpload(*src2.ObjectKey, []byte("mock zip content"))
	
	_ = svc.ConfirmUpload(ctx, srcID2)
	time.Sleep(100 * time.Millisecond)

	src2, _ = repo.GetSource(ctx, srcID2)
	if src2.ObjectKey != nil || len(store.objects) != 0 {
		t.Errorf("Codebase object not cleaned up")
	}

	// For connectors, we can't easily test without hitting the real API or mocking the HTTP client.
	// But we can verify that unsupported kind returns error.
	_, err = svc.StartConnectorSource(ctx, userID, "unknown", "ref")
	if err == nil {
		t.Errorf("Expected error for unknown connector kind")
	}
}
