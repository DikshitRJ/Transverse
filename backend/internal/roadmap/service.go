package roadmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"transverse/internal/graph"
	"transverse/internal/llm"
	"transverse/internal/models"
	"transverse/internal/repository"
)

type Service struct {
	roadmapRepo *repository.RoadmapRepo
	userRepo    *repository.UserRepo
	problemRepo *repository.ProblemRepo
	statsRepo   *repository.StatsRepo
	graph       graph.TopicGraph
	llmClient   llm.Client
	rdb         *redis.Client
	tmpl        *template.Template
}

func NewService(
	rr *repository.RoadmapRepo,
	ur *repository.UserRepo,
	pr *repository.ProblemRepo,
	sr *repository.StatsRepo,
	g graph.TopicGraph,
	llmClient llm.Client,
	rdb *redis.Client,
	templatePath string,
) (*Service, error) {
	if templatePath == "" {
		templatePath = "internal/llm/prompts/roadmap.tmpl"
	}
	var tmpl *template.Template
	var err error
	if templatePath != "" {
		tmpl, err = template.ParseFiles(templatePath)
		if err != nil {
			slog.Warn("roadmap template not found, LLM roadmap generation will be unavailable", "path", templatePath, "error", err)
		}
	}

	return &Service{
		roadmapRepo: rr,
		userRepo:    ur,
		problemRepo: pr,
		statsRepo:   sr,
		graph:       g,
		llmClient:   llmClient,
		rdb:         rdb,
		tmpl:        tmpl,
	}, nil
}

type GenerateRequest struct {
	UserID              uuid.UUID
	TargetRole          string
	ConfirmedHypotheses []string
	DebunkedHypotheses  []string
}

type llmRoadmapPhase struct {
	Title string           `json:"title"`
	Nodes []llmRoadmapNode `json:"nodes"`
}

type llmRoadmapNode struct {
	TopicID    string          `json:"topic_id"`
	UnlockRule json.RawMessage `json:"unlock_rule"`
}

type llmRoadmapResponse struct {
	Phases []llmRoadmapPhase `json:"phases"`
}

// GetCurrentRoadmap is the primary endpoint called by the frontend on load.
// Dynamically evaluates progression and presents only ONE active section with full details,
// generating/unlocking subsequent sections after the current one completes.
func (s *Service) GetCurrentRoadmap(ctx context.Context, userID uuid.UUID) (*models.RoadmapCurrentResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found: %s", userID.String())
	}

	userRating := user.Theta
	if userRating <= 0 {
		userRating = 1300.0
	}

	// 1. Fetch user roadmap
	ur, err := s.roadmapRepo.GetUserRoadmap(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user roadmap: %w", err)
	}

	// 2. Initialize default roadmap if none exists
	if ur == nil {
		if initErr := s.initializeDefaultRoadmap(ctx, userID, "Software Engineer - DSA & Problem Solving"); initErr != nil {
			return nil, fmt.Errorf("failed to initialize roadmap: %w", initErr)
		}
		ur, err = s.roadmapRepo.GetUserRoadmap(ctx, userID)
		if err != nil || ur == nil {
			return nil, fmt.Errorf("failed to load newly initialized roadmap: %w", err)
		}
	}

	// 3. Load phases of this template
	phases, err := s.roadmapRepo.GetTemplatePhases(ctx, ur.RoadmapTemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to load template phases: %w", err)
	}
	if len(phases) == 0 {
		return nil, fmt.Errorf("no phases found in roadmap template")
	}

	// 4. Determine current active phase
	var currentPhase *models.RoadmapPhase
	if ur.CurrentPhaseID != nil {
		for i := range phases {
			if phases[i].ID == *ur.CurrentPhaseID {
				currentPhase = &phases[i]
				break
			}
		}
	}
	if currentPhase == nil {
		currentPhase = &phases[0]
		_ = s.roadmapRepo.UpdateUserRoadmapPhase(ctx, ur.ID, &currentPhase.ID)
	}

	// 5. Evaluate node progresses for the current phase
	progresses, err := s.roadmapRepo.GetUserNodeProgresses(ctx, ur.ID)
	if err != nil {
		progresses = []models.UserRoadmapNodeProgress{}
	}
	progressMap := make(map[uuid.UUID]models.UserRoadmapNodeProgress)
	for _, p := range progresses {
		progressMap[p.NodeID] = p
	}

	nodes, err := s.roadmapRepo.GetPhaseNodes(ctx, currentPhase.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load phase nodes: %w", err)
	}

	// Check if all nodes in current phase are mastered
	allMastered := len(nodes) > 0
	for _, node := range nodes {
		p, exists := progressMap[node.ID]
		if !exists || (p.Status != models.NodeStatusMastered && p.Status != models.NodeStatusTestedOut) {
			allMastered = false
			break
		}
	}

	// 6. If all nodes in current phase are completed, advance to next section
	if allMastered {
		nextPhaseIdx := -1
		for i := range phases {
			if phases[i].ID == currentPhase.ID {
				nextPhaseIdx = i + 1
				break
			}
		}

		if nextPhaseIdx >= 0 && nextPhaseIdx < len(phases) {
			// Advance to next section
			nextPhase := &phases[nextPhaseIdx]
			_ = s.roadmapRepo.UpdateUserRoadmapPhase(ctx, ur.ID, &nextPhase.ID)
			currentPhase = nextPhase

			// Load nodes for new section
			nodes, _ = s.roadmapRepo.GetPhaseNodes(ctx, currentPhase.ID)

			// Unlock initial node of next phase
			for j, n := range nodes {
				prog, exists := progressMap[n.ID]
				if !exists || prog.Status == models.NodeStatusLocked {
					if j == 0 || isNoPrereq(n.UnlockRule) {
						now := time.Now()
						_ = s.roadmapRepo.CreateUserNodeProgress(ctx, &models.UserRoadmapNodeProgress{
							UserRoadmapID: ur.ID,
							NodeID:        n.ID,
							Status:        models.NodeStatusUnlocked,
							UnlockedAt:    &now,
						})
					}
				}
			}

			// Reload progresses
			progresses, _ = s.roadmapRepo.GetUserNodeProgresses(ctx, ur.ID)
			progressMap = make(map[uuid.UUID]models.UserRoadmapNodeProgress)
			for _, p := range progresses {
				progressMap[p.NodeID] = p
			}

			s.publishEvent(ctx, userID, "roadmap.section_unlocked", map[string]interface{}{
				"section_sequence": currentPhase.Sequence,
				"section_title":    currentPhase.Title,
			})
		}
	}

	// 7. Build rich subsections for the single active section
	var subsections []models.RoadmapSubsection
	masteredCount := 0

	for i, node := range nodes {
		prog, exists := progressMap[node.ID]
		status := models.NodeStatusLocked
		if exists {
			status = prog.Status
		} else if i == 0 {
			status = models.NodeStatusUnlocked
		}

		if status == models.NodeStatusMastered || status == models.NodeStatusTestedOut {
			masteredCount++
		}

		// Load topic stats for this user
		var topicTheta float64 = userRating
		var masteryScore float64 = 0.0
		if s.statsRepo != nil {
			stat, sErr := s.statsRepo.GetByUserAndTopic(ctx, userID.String(), node.TopicID)
			if sErr == nil && stat != nil {
				if stat.Theta > 0 {
					topicTheta = stat.Theta
				}
				masteryScore = stat.MasteryScore
			}
		}

		targetRating := 1200.0 + float64(currentPhase.Sequence*200)

		// Tutorials: fetch from repository or fallback
		tutorials, _ := s.roadmapRepo.GetTutorialsByIDs(ctx, node.TutorialIDs)
		if len(tutorials) == 0 {
			tutorials, _ = s.roadmapRepo.GetTutorialsByTopic(ctx, node.TopicID)
		}
		if len(tutorials) == 0 {
			tutorials = s.GetCuratedTutorials(node.TopicID)
		}

		// Questions: fetch problems matching this topic
		var questions []models.ProblemPayload
		if s.problemRepo != nil {
			probs, _ := s.problemRepo.GetByScope(ctx, models.SessionScope{
				Topics: []string{node.TopicID},
				DifficultyRange: [2]int{
					int(targetRating - 250),
					int(targetRating + 250),
				},
			})
			if len(probs) == 0 {
				probs, _ = s.problemRepo.GetByTopic(ctx, node.TopicID)
			}
			if len(probs) > 4 {
				probs = probs[:4]
			}
			questions = models.ToProblemPayloads(probs)
		}

		subsections = append(subsections, models.RoadmapSubsection{
			NodeID:       node.ID,
			TopicID:      node.TopicID,
			Title:        s.formatTopicTitle(node.TopicID),
			Sequence:     node.Sequence,
			Status:       status,
			UserRating:   topicTheta,
			TargetRating: targetRating,
			MasteryScore: masteryScore,
			Tutorials:    tutorials,
			Questions:    questions,
		})
	}

	progressPct := 0.0
	if len(nodes) > 0 {
		progressPct = float64(masteredCount) / float64(len(nodes)) * 100.0
	}

	activeSection := &models.RoadmapSection{
		PhaseID:            currentPhase.ID,
		Sequence:           currentPhase.Sequence,
		Title:              currentPhase.Title,
		Status:             "ACTIVE",
		ProgressPercentage: progressPct,
		Subsections:        subsections,
	}

	// 8. Build minimal upcoming sections preview (unrevealed questions)
	var upcomingPreviews []models.UpcomingSectionPreview
	for _, p := range phases {
		if p.Sequence > currentPhase.Sequence {
			upcomingPreviews = append(upcomingPreviews, models.UpcomingSectionPreview{
				Sequence: p.Sequence,
				Title:    p.Title,
				Status:   "LOCKED",
			})
		}
	}

	return &models.RoadmapCurrentResponse{
		RoadmapID:        ur.ID,
		UserID:           userID.String(),
		UserRating:       userRating,
		TargetRole:       "Software Engineer - DSA & Problem Solving",
		Status:           ur.Status,
		TotalSections:    len(phases),
		CurrentSection:   activeSection,
		UpcomingSections: upcomingPreviews,
	}, nil
}

// CompleteNode marks a roadmap node as mastered and unlocks subsequent dependent nodes.
func (s *Service) CompleteNode(ctx context.Context, userID, nodeID uuid.UUID) error {
	ur, err := s.roadmapRepo.GetUserRoadmap(ctx, userID)
	if err != nil || ur == nil {
		return fmt.Errorf("user roadmap not found")
	}

	prog, err := s.roadmapRepo.GetUserProgressByNode(ctx, ur.ID, nodeID)
	now := time.Now()
	if prog == nil {
		prog = &models.UserRoadmapNodeProgress{
			UserRoadmapID: ur.ID,
			NodeID:        nodeID,
			Status:        models.NodeStatusMastered,
			UnlockedAt:    &now,
			MasteredAt:    &now,
		}
		if err := s.roadmapRepo.CreateUserNodeProgress(ctx, prog); err != nil {
			return err
		}
	} else {
		if err := s.roadmapRepo.UpdateNodeProgress(ctx, prog.ID, models.NodeStatusMastered, prog.UnlockedAt, &now); err != nil {
			return err
		}
	}

	_ = s.Unlock(ctx, ur.ID)
	s.publishEvent(ctx, userID, "node.completed", map[string]interface{}{"node_id": nodeID})
	return nil
}

// TestOut allows skipping a node by testing out on a placement challenge.
func (s *Service) TestOut(ctx context.Context, userID, nodeID uuid.UUID) error {
	ur, err := s.roadmapRepo.GetUserRoadmap(ctx, userID)
	if err != nil || ur == nil {
		return fmt.Errorf("user roadmap not found")
	}

	prog, err := s.roadmapRepo.GetUserProgressByNode(ctx, ur.ID, nodeID)
	now := time.Now()
	if prog == nil {
		prog = &models.UserRoadmapNodeProgress{
			UserRoadmapID: ur.ID,
			NodeID:        nodeID,
			Status:        models.NodeStatusTestedOut,
			UnlockedAt:    &now,
			MasteredAt:    &now,
		}
		if err := s.roadmapRepo.CreateUserNodeProgress(ctx, prog); err != nil {
			return err
		}
	} else {
		if err := s.roadmapRepo.UpdateNodeProgress(ctx, prog.ID, models.NodeStatusTestedOut, prog.UnlockedAt, &now); err != nil {
			return err
		}
	}

	_ = s.Unlock(ctx, ur.ID)
	s.publishEvent(ctx, userID, "node.tested_out", map[string]interface{}{"node_id": nodeID})
	return nil
}

// initializeDefaultRoadmap builds a curated progressive 5-section roadmap.
func (s *Service) initializeDefaultRoadmap(ctx context.Context, userID uuid.UUID, targetRole string) error {
	tmpl := &models.RoadmapTemplate{
		TargetRole: targetRole,
		Source:     models.RoadmapSourceCurated,
		Version:    1,
	}
	if err := s.roadmapRepo.CreateTemplate(ctx, tmpl); err != nil {
		return err
	}

	ur := &models.UserRoadmap{
		UserID:            userID,
		RoadmapTemplateID: tmpl.ID,
		Status:            models.RoadmapStatusActive,
	}
	if err := s.roadmapRepo.CreateUserRoadmap(ctx, ur); err != nil {
		return err
	}

	sectionDefs := []struct {
		title  string
		topics []string
	}{
		{
			title:  "Section 1: Foundations & Linear Data Structures",
			topics: []string{"foundations", "arrays-hashing", "two-pointers", "sliding-window", "stack-queues"},
		},
		{
			title:  "Section 2: Searching, Sorting & Non-Linear Basics",
			topics: []string{"binary-search", "sorting-searching", "linked-list", "trees", "heaps-priority-queues"},
		},
		{
			title:  "Section 3: Graph Traversal & Structural Algorithms",
			topics: []string{"backtracking", "graphs", "disjoint-set-union", "topological-sort", "shortest-paths"},
		},
		{
			title:  "Section 4: Dynamic Programming & Optimization",
			topics: []string{"greedy", "bit-manipulation", "dynamic-programming", "minimum-spanning-tree"},
		},
		{
			title:  "Section 5: Advanced Algorithms & Mastery",
			topics: []string{"advanced-dp", "network-flows"},
		},
	}

	var firstPhaseID uuid.UUID

	for i, sDef := range sectionDefs {
		phase := &models.RoadmapPhase{
			RoadmapTemplateID: tmpl.ID,
			Sequence:          i + 1,
			Title:             sDef.title,
			UnlockRule:        json.RawMessage(`{"type":"no_prerequisite"}`),
		}
		if err := s.roadmapRepo.CreatePhase(ctx, phase); err != nil {
			return err
		}

		if i == 0 {
			firstPhaseID = phase.ID
		}

		for j, topicID := range sDef.topics {
			node := &models.RoadmapNode{
				PhaseID:          phase.ID,
				TopicID:          topicID,
				Sequence:         j + 1,
				UnlockRule:       json.RawMessage(`{"type":"no_prerequisite"}`),
				TutorialIDs:      []uuid.UUID{},
				PracticeTopicIDs: []string{topicID},
			}
			if err := s.roadmapRepo.CreateNode(ctx, node); err != nil {
				return err
			}

			status := models.NodeStatusLocked
			if i == 0 && j == 0 {
				status = models.NodeStatusUnlocked
			}

			var unlockedAt *time.Time
			if status == models.NodeStatusUnlocked {
				now := time.Now()
				unlockedAt = &now
			}

			_ = s.roadmapRepo.CreateUserNodeProgress(ctx, &models.UserRoadmapNodeProgress{
				UserRoadmapID: ur.ID,
				NodeID:        node.ID,
				Status:        status,
				UnlockedAt:    unlockedAt,
			})
		}
	}

	return s.roadmapRepo.UpdateUserRoadmapPhase(ctx, ur.ID, &firstPhaseID)
}

func (s *Service) formatTopicTitle(topicID string) string {
	parts := map[string]string{
		"foundations":           "Foundations & Implementation",
		"arrays-hashing":        "Arrays & Hashing",
		"two-pointers":          "Two Pointers",
		"sliding-window":        "Sliding Window",
		"stack-queues":          "Stacks & Queues",
		"binary-search":         "Binary Search",
		"sorting-searching":     "Sorting & Searching",
		"linked-list":           "Linked Lists",
		"trees":                 "Trees & Binary Trees",
		"heaps-priority-queues": "Heaps & Priority Queues",
		"backtracking":          "Backtracking & Recursion",
		"graphs":                "Graphs & Traversals",
		"shortest-paths":        "Shortest Paths (Dijkstra)",
		"disjoint-set-union":    "Disjoint Set Union (DSU)",
		"topological-sort":      "Topological Sort & DAGs",
		"minimum-spanning-tree": "Minimum Spanning Tree (MST)",
		"dynamic-programming":   "Dynamic Programming",
		"advanced-dp":           "Advanced & Bitmask DP",
		"greedy":                "Greedy Algorithms",
		"bit-manipulation":      "Bit Manipulation",
		"network-flows":         "Network Flows & Matchings",
	}
	if name, ok := parts[topicID]; ok {
		return name
	}
	return topicID
}

func (s *Service) GetCuratedTutorials(topicID string) []models.Tutorial {
	curated := map[string][]models.Tutorial{
		"arrays-hashing": {
			{
				ID:               uuid.New(),
				Source:           "Transverse Core",
				SourceURL:        "https://neetcode.io/courses/dsa-for-beginners/0",
				Title:            "Arrays & Dynamic Arrays Deep Dive",
				TopicTags:        []string{"arrays-hashing"},
				Type:             "article",
				Difficulty:       "beginner",
				EstimatedMinutes: 10,
				Summary:          "Understand memory layout, cache locality, amortized dynamic array resizing, and hash table collisions.",
				Status:           "UNREAD",
			},
			{
				ID:               uuid.New(),
				Source:           "NeetCode",
				SourceURL:        "https://www.youtube.com/watch?v=KLlXCFG5TnA",
				Title:            "Two Sum & Hash Table Pattern",
				TopicTags:        []string{"arrays-hashing"},
				Type:             "video",
				Difficulty:       "beginner",
				EstimatedMinutes: 12,
				Summary:          "Video breakdown of using hash maps for O(n) complement lookups instead of O(n^2) brute force.",
				Status:           "UNREAD",
			},
		},
		"binary-search": {
			{
				ID:               uuid.New(),
				Source:           "Transverse Core",
				SourceURL:        "https://cp-algorithms.com/num_methods/binary_search.html",
				Title:            "Binary Search on Monotonic Predicates",
				TopicTags:        []string{"binary-search"},
				Type:             "article",
				Difficulty:       "intermediate",
				EstimatedMinutes: 15,
				Summary:          "Generalizing binary search beyond sorted arrays to monotonic boolean functions (binary search on answer).",
				Status:           "UNREAD",
			},
		},
		"two-pointers": {
			{
				ID:               uuid.New(),
				Source:           "Transverse Core",
				SourceURL:        "https://leetcode.com/explore/learn/card/array-and-string/",
				Title:            "Two-Pointer Convergence & Partitioning",
				TopicTags:        []string{"two-pointers"},
				Type:             "article",
				Difficulty:       "beginner",
				EstimatedMinutes: 10,
				Summary:          "Patterns for opposite-end convergence, fast-and-slow runner pointers, and in-place partitioning.",
				Status:           "UNREAD",
			},
		},
		"dynamic-programming": {
			{
				ID:               uuid.New(),
				Source:           "Transverse Core",
				SourceURL:        "https://cp-algorithms.com/dynamic_programming/intro-to-dp.html",
				Title:            "DP State Design & Transition Formulations",
				TopicTags:        []string{"dynamic-programming"},
				Type:             "article",
				Difficulty:       "intermediate",
				EstimatedMinutes: 20,
				Summary:          "Step-by-step framework for defining optimal substructure, overlapping subproblems, state dimensions, and base cases.",
				Status:           "UNREAD",
			},
		},
	}

	if list, ok := curated[topicID]; ok {
		return list
	}

	return []models.Tutorial{
		{
			ID:               uuid.New(),
			Source:           "Transverse Core",
			SourceURL:        fmt.Sprintf("https://cp-algorithms.com/%s", topicID),
			Title:            fmt.Sprintf("Comprehensive Guide to %s", s.formatTopicTitle(topicID)),
			TopicTags:        []string{topicID},
			Type:             "article",
			Difficulty:       "intermediate",
			EstimatedMinutes: 15,
			Summary:          fmt.Sprintf("Essential theoretical foundations, time/space complexities, and implementation idioms for %s.", s.formatTopicTitle(topicID)),
			Status:           "UNREAD",
		},
	}
}

// Generate creates an LLM-driven personalized roadmap based on confirmed/debunked skill hypotheses.
func (s *Service) Generate(ctx context.Context, req GenerateRequest) error {
	user, err := s.userRepo.GetByID(ctx, req.UserID.String())
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	skillProfile := make(map[string]float64)
	if user.DNARaw != nil {
		var dna models.LearningDNA
		if err := json.Unmarshal(user.DNARaw, &dna); err == nil {
			for topic, bias := range dna.TopicBias {
				skillProfile[topic] = bias
			}
		}
	}

	masteryJSON, _ := json.Marshal(skillProfile)
	confirmedTopicsJSON, _ := json.Marshal(req.ConfirmedHypotheses)
	debunkedTopicsJSON, _ := json.Marshal(req.DebunkedHypotheses)
	topicDAGJSON, _ := json.Marshal(s.graph.GetAllTopics())

	data := map[string]interface{}{
		"TargetRole":      req.TargetRole,
		"MasteryJSON":     string(masteryJSON),
		"ConfirmedTopics": string(confirmedTopicsJSON),
		"DebunkedTopics":  string(debunkedTopicsJSON),
		"TopicDAG":        string(topicDAGJSON),
	}

	if s.tmpl == nil {
		return s.initializeDefaultRoadmap(ctx, req.UserID, req.TargetRole)
	}

	var buf bytes.Buffer
	if err := s.tmpl.Execute(&buf, data); err != nil {
		return s.initializeDefaultRoadmap(ctx, req.UserID, req.TargetRole)
	}

	llmReq := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are an expert technical curriculum designer. Return strict JSON."},
			{Role: "user", Content: buf.String()},
		},
		MaxTokens:   1200,
		Temperature: 0.2,
	}

	llmRes, err := s.llmClient.Complete(ctx, llmReq, true)
	if err != nil {
		return s.initializeDefaultRoadmap(ctx, req.UserID, req.TargetRole)
	}

	var resp llmRoadmapResponse
	if err := json.Unmarshal([]byte(llmRes), &resp); err != nil || len(resp.Phases) == 0 {
		return s.initializeDefaultRoadmap(ctx, req.UserID, req.TargetRole)
	}

	tmpl := &models.RoadmapTemplate{
		TargetRole: req.TargetRole,
		Source:     models.RoadmapSourceLLMGenerated,
		Version:    1,
	}
	if err := s.roadmapRepo.CreateTemplate(ctx, tmpl); err != nil {
		return err
	}

	ur := &models.UserRoadmap{
		UserID:            req.UserID,
		RoadmapTemplateID: tmpl.ID,
		Status:            models.RoadmapStatusActive,
	}
	if err := s.roadmapRepo.CreateUserRoadmap(ctx, ur); err != nil {
		return err
	}

	var firstPhaseID uuid.UUID

	for i, p := range resp.Phases {
		phase := &models.RoadmapPhase{
			RoadmapTemplateID: tmpl.ID,
			Sequence:          i + 1,
			Title:             p.Title,
			UnlockRule:        json.RawMessage(`{"type":"no_prerequisite"}`),
		}
		if err := s.roadmapRepo.CreatePhase(ctx, phase); err != nil {
			return err
		}

		if i == 0 {
			firstPhaseID = phase.ID
		}

		for j, n := range p.Nodes {
			if !s.graph.IsValidTopic(n.TopicID) {
				continue
			}

			node := &models.RoadmapNode{
				PhaseID:          phase.ID,
				TopicID:          n.TopicID,
				Sequence:         j + 1,
				UnlockRule:       n.UnlockRule,
				TutorialIDs:      []uuid.UUID{},
				PracticeTopicIDs: []string{n.TopicID},
			}
			if err := s.roadmapRepo.CreateNode(ctx, node); err != nil {
				return err
			}

			status := models.NodeStatusLocked
			if i == 0 && (j == 0 || isNoPrereq(n.UnlockRule)) {
				status = models.NodeStatusUnlocked
			}

			var unlockedAt *time.Time
			if status == models.NodeStatusUnlocked {
				now := time.Now()
				unlockedAt = &now
			}

			_ = s.roadmapRepo.CreateUserNodeProgress(ctx, &models.UserRoadmapNodeProgress{
				UserRoadmapID: ur.ID,
				NodeID:        node.ID,
				Status:        status,
				UnlockedAt:    unlockedAt,
			})
		}
	}

	return s.roadmapRepo.UpdateUserRoadmapPhase(ctx, ur.ID, &firstPhaseID)
}

func isNoPrereq(rule json.RawMessage) bool {
	if len(rule) == 0 {
		return true
	}
	var r models.UnlockRule
	if err := json.Unmarshal(rule, &r); err == nil {
		return r.Type == "no_prerequisite" || r.Type == ""
	}
	return true
}

func (s *Service) Unlock(ctx context.Context, userRoadmapID uuid.UUID) error {
	ur, err := s.roadmapRepo.GetUserRoadmap(ctx, userRoadmapID)
	if err != nil || ur == nil {
		return nil
	}

	progresses, err := s.roadmapRepo.GetUserNodeProgresses(ctx, userRoadmapID)
	if err != nil {
		return err
	}
	progressMap := make(map[uuid.UUID]models.UserRoadmapNodeProgress)
	for _, p := range progresses {
		progressMap[p.NodeID] = p
	}

	phases, err := s.roadmapRepo.GetTemplatePhases(ctx, ur.RoadmapTemplateID)
	if err != nil {
		return err
	}

	for _, phase := range phases {
		nodes, err := s.roadmapRepo.GetPhaseNodes(ctx, phase.ID)
		if err != nil {
			continue
		}

		for _, node := range nodes {
			prog, ok := progressMap[node.ID]
			if ok && prog.Status == models.NodeStatusLocked {
				now := time.Now()
				_ = s.roadmapRepo.UpdateNodeProgress(ctx, prog.ID, models.NodeStatusUnlocked, &now, prog.MasteredAt)
				s.publishEvent(ctx, ur.UserID, "node.unlocked", map[string]interface{}{"node_id": node.ID})
			}
		}
	}
	return nil
}

func (s *Service) Regenerate(ctx context.Context, userRoadmapID uuid.UUID) error {
	ur, err := s.roadmapRepo.GetUserRoadmap(ctx, userRoadmapID)
	if err != nil || ur == nil {
		return err
	}
	s.publishEvent(ctx, ur.UserID, "roadmap.updated", map[string]interface{}{"user_roadmap_id": userRoadmapID})
	return nil
}

func (s *Service) publishEvent(ctx context.Context, userID uuid.UUID, eventType string, data map[string]interface{}) {
	if s.rdb == nil {
		return
	}
	channel := fmt.Sprintf("user:%s:events", userID.String())
	payload := map[string]interface{}{
		"type": eventType,
		"data": data,
	}
	b, _ := json.Marshal(payload)
	s.rdb.Publish(ctx, channel, string(b))
}
