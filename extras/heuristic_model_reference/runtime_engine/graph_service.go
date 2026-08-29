package services

import (
	"context"
	"fmt"
	"strings"

	"velocity/internal/graph"
	"velocity/internal/models"
	"velocity/internal/repository"
)

type GraphService struct {
	userRepo *repository.UserRepo
	graphSvc *LearnService
}

func NewGraphService(userRepo *repository.UserRepo, learnSvc *LearnService) *GraphService {
	return &GraphService{userRepo: userRepo, graphSvc: learnSvc}
}

func (gs *GraphService) HydrateForUserID(ctx context.Context, userID string) (*models.GraphPayload, error) {
	nodes, err := gs.graphSvc.GetChapters(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("graph: get chapters: %w", err)
	}

	nodeMap := make(map[string]models.GraphNode)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	syllabus := gs.graphSvc.SyllabusGraph()
	edges := gs.buildEdges(syllabus, nodeMap)

	return &models.GraphPayload{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func (gs *GraphService) buildEdges(syllabus graph.SyllabusGraph, nodeMap map[string]models.GraphNode) []models.GraphEdge {
	var edges []models.GraphEdge
	for subject, chapters := range syllabus {
		for id, node := range chapters {
			for _, prereq := range node.Prerequisites {
				var fromID, prereqSlug string
				crossSubject := false

				if strings.Contains(prereq, "/") {
					// Cross-subject reference: "maths/vector-algebra"
					parts := strings.SplitN(prereq, "/", 2)
					prereqSlug = parts[1]
					fromID = prereq // already fully qualified
					crossSubject = true
				} else {
					prereqSlug = prereq
					fromID = subject + "/" + prereq
				}

				toID := subject + "/" + id

				strainIndex := 0.0
				if from, ok := nodeMap[prereqSlug]; ok {
					if to, ok := nodeMap[id]; ok {
						strainIndex = from.MasteryScore - to.MasteryScore
					}
				}

				edges = append(edges, models.GraphEdge{
					From:           fromID,
					To:             toID,
					StrainIndex:    strainIndex,
					CrossSubject:   crossSubject,
					IsPrerequisite: true,
				})
			}
		}
	}
	if edges == nil {
		edges = []models.GraphEdge{}
	}
	return edges
}
