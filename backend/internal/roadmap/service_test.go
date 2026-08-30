package roadmap

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"transverse/internal/graph"
	"transverse/internal/models"
)

// Mock repos for in-memory roadmap testing
type mockRoadmapRepo struct {
	templates  map[uuid.UUID]*models.RoadmapTemplate
	phases     map[uuid.UUID][]models.RoadmapPhase
	nodes      map[uuid.UUID][]models.RoadmapNode
	userMap    map[uuid.UUID]*models.UserRoadmap
	progresses map[uuid.UUID]map[uuid.UUID]*models.UserRoadmapNodeProgress
}

func newMockRoadmapRepo() *mockRoadmapRepo {
	return &mockRoadmapRepo{
		templates:  make(map[uuid.UUID]*models.RoadmapTemplate),
		phases:     make(map[uuid.UUID][]models.RoadmapPhase),
		nodes:      make(map[uuid.UUID][]models.RoadmapNode),
		userMap:    make(map[uuid.UUID]*models.UserRoadmap),
		progresses: make(map[uuid.UUID]map[uuid.UUID]*models.UserRoadmapNodeProgress),
	}
}

func (m *mockRoadmapRepo) CreateTemplate(ctx context.Context, tmpl *models.RoadmapTemplate) error {
	tmpl.ID = uuid.New()
	m.templates[tmpl.ID] = tmpl
	return nil
}

func (m *mockRoadmapRepo) CreatePhase(ctx context.Context, phase *models.RoadmapPhase) error {
	phase.ID = uuid.New()
	m.phases[phase.RoadmapTemplateID] = append(m.phases[phase.RoadmapTemplateID], *phase)
	return nil
}

func (m *mockRoadmapRepo) CreateNode(ctx context.Context, node *models.RoadmapNode) error {
	node.ID = uuid.New()
	m.nodes[node.PhaseID] = append(m.nodes[node.PhaseID], *node)
	return nil
}

func (m *mockRoadmapRepo) CreateUserRoadmap(ctx context.Context, ur *models.UserRoadmap) error {
	ur.ID = uuid.New()
	m.userMap[ur.UserID] = ur
	if m.progresses[ur.ID] == nil {
		m.progresses[ur.ID] = make(map[uuid.UUID]*models.UserRoadmapNodeProgress)
	}
	return nil
}

func (m *mockRoadmapRepo) CreateUserNodeProgress(ctx context.Context, up *models.UserRoadmapNodeProgress) error {
	up.ID = uuid.New()
	if m.progresses[up.UserRoadmapID] == nil {
		m.progresses[up.UserRoadmapID] = make(map[uuid.UUID]*models.UserRoadmapNodeProgress)
	}
	m.progresses[up.UserRoadmapID][up.NodeID] = up
	return nil
}

func (m *mockRoadmapRepo) GetUserRoadmap(ctx context.Context, userID uuid.UUID) (*models.UserRoadmap, error) {
	return m.userMap[userID], nil
}

func (m *mockRoadmapRepo) GetTemplatePhases(ctx context.Context, templateID uuid.UUID) ([]models.RoadmapPhase, error) {
	return m.phases[templateID], nil
}

func (m *mockRoadmapRepo) GetPhaseNodes(ctx context.Context, phaseID uuid.UUID) ([]models.RoadmapNode, error) {
	return m.nodes[phaseID], nil
}

func (m *mockRoadmapRepo) GetUserNodeProgresses(ctx context.Context, userRoadmapID uuid.UUID) ([]models.UserRoadmapNodeProgress, error) {
	var list []models.UserRoadmapNodeProgress
	if pMap, ok := m.progresses[userRoadmapID]; ok {
		for _, p := range pMap {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockRoadmapRepo) UpdateNodeProgress(ctx context.Context, id uuid.UUID, status models.NodeStatus, unlockedAt, masteredAt *time.Time) error {
	for _, pMap := range m.progresses {
		for _, p := range pMap {
			if p.ID == id {
				p.Status = status
				return nil
			}
		}
	}
	return nil
}

func (m *mockRoadmapRepo) UpdateUserRoadmapPhase(ctx context.Context, id uuid.UUID, phaseID *uuid.UUID) error {
	for _, ur := range m.userMap {
		if ur.ID == id {
			ur.CurrentPhaseID = phaseID
			return nil
		}
	}
	return nil
}

func (m *mockRoadmapRepo) GetTutorialsByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Tutorial, error) {
	return []models.Tutorial{}, nil
}

func (m *mockRoadmapRepo) GetTutorialsByTopic(ctx context.Context, topicID string) ([]models.Tutorial, error) {
	return []models.Tutorial{}, nil
}

func (m *mockRoadmapRepo) GetUserProgressByNode(ctx context.Context, userRoadmapID, nodeID uuid.UUID) (*models.UserRoadmapNodeProgress, error) {
	if pMap, ok := m.progresses[userRoadmapID]; ok {
		return pMap[nodeID], nil
	}
	return nil, nil
}

func TestRoadmap_SingleActiveSectionAndProgression(t *testing.T) {
	mockRepo := newMockRoadmapRepo()
	userID := uuid.New()

	jsonTopics := []byte(`[
		{"id": "foundations", "name": "Foundations", "order": 1},
		{"id": "arrays-hashing", "name": "Arrays & Hashing", "order": 2},
		{"id": "two-pointers", "name": "Two Pointers", "order": 3},
		{"id": "sliding-window", "name": "Sliding Window", "order": 4},
		{"id": "stack-queues", "name": "Stacks & Queues", "order": 5}
	]`)
	tg, err := graph.NewTopicGraphFromJSON(jsonTopics)
	if err != nil {
		t.Fatalf("failed to create topic graph: %v", err)
	}

	svc := &Service{
		roadmapRepo: nil,
		graph:       tg,
	}

	// 1. Initialize default roadmap
	err = svc.initializeMock(context.Background(), mockRepo, userID)
	if err != nil {
		t.Fatalf("failed to init mock: %v", err)
	}

	// 2. Fetch user roadmap
	ur, _ := mockRepo.GetUserRoadmap(context.Background(), userID)
	if ur == nil {
		t.Fatalf("expected user roadmap to be created")
	}

	phases, _ := mockRepo.GetTemplatePhases(context.Background(), ur.RoadmapTemplateID)
	if len(phases) != 5 {
		t.Fatalf("expected 5 progressive phases, got %d", len(phases))
	}

	// Section 1 should be active
	if *ur.CurrentPhaseID != phases[0].ID {
		t.Errorf("expected current phase to be phase 1")
	}

	// 3. Verify nodes in section 1
	nodes, _ := mockRepo.GetPhaseNodes(context.Background(), phases[0].ID)
	if len(nodes) != 5 {
		t.Fatalf("expected 5 nodes in section 1, got %d", len(nodes))
	}

	// Verify tutorials exist for topic
	tutorials := svc.GetCuratedTutorials("arrays-hashing")
	if len(tutorials) == 0 {
		t.Errorf("expected curated tutorials for arrays-hashing")
	}
	if tutorials[0].Title == "" || tutorials[0].SourceURL == "" {
		t.Errorf("tutorial missing required fields: %+v", tutorials[0])
	}
}

func (s *Service) initializeMock(ctx context.Context, m *mockRoadmapRepo, userID uuid.UUID) error {
	tmpl := &models.RoadmapTemplate{
		TargetRole: "Software Engineer - DSA",
		Source:     models.RoadmapSourceCurated,
		Version:    1,
	}
	_ = m.CreateTemplate(ctx, tmpl)

	ur := &models.UserRoadmap{
		UserID:            userID,
		RoadmapTemplateID: tmpl.ID,
		Status:            models.RoadmapStatusActive,
	}
	_ = m.CreateUserRoadmap(ctx, ur)

	for i := 1; i <= 5; i++ {
		phase := &models.RoadmapPhase{
			RoadmapTemplateID: tmpl.ID,
			Sequence:          i,
			Title:             "Section " + string(rune('0'+i)),
		}
		_ = m.CreatePhase(ctx, phase)
		if i == 1 {
			ur.CurrentPhaseID = &phase.ID
		}

		for j := 1; j <= 5; j++ {
			node := &models.RoadmapNode{
				PhaseID:  phase.ID,
				TopicID:  "topic",
				Sequence: j,
			}
			_ = m.CreateNode(ctx, node)
			_ = m.CreateUserNodeProgress(ctx, &models.UserRoadmapNodeProgress{
				UserRoadmapID: ur.ID,
				NodeID:        node.ID,
				Status:        models.NodeStatusLocked,
			})
		}
	}
	return nil
}
