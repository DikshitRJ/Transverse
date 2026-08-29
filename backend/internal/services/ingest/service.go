package ingest

import (
	"context"
	"fmt"

	"transverse/internal/models"
	"transverse/internal/repository"
	"transverse/internal/services"
)

type Service struct {
	repo         *repository.IngestRepo
	graphService *services.GraphService
}

func NewService(repo *repository.IngestRepo, graphService *services.GraphService) *Service {
	return &Service{
		repo:         repo,
		graphService: graphService,
	}
}

// IngestTutorials processes an array of tutorials. Returns a list of errors for the ones that failed.
func (s *Service) IngestTutorials(ctx context.Context, tutorials []models.TutorialIngestRecord) []error {
	var errors []error
	for i, t := range tutorials {
		// Resolve the first topic tag as the primary topic ID
		var topicID string
		if len(t.TopicTags) > 0 {
			resolved, err := s.graphService.ResolveScope(t.TopicTags)
			if err == nil && len(resolved) > 0 {
				topicID = resolved[0]
			}
		}

		_, err := s.repo.UpsertTutorial(ctx, &t, topicID)
		if err != nil {
			errors = append(errors, fmt.Errorf("record %d (%s): %w", i, t.SourceURL, err))
		}
	}
	return errors
}

// IngestRoadmapTemplates processes curated roadmaps.
func (s *Service) IngestRoadmapTemplates(ctx context.Context, templates []models.RoadmapTemplateIngestRecord) []error {
	var errors []error
	for i, t := range templates {
		// Prepare resolution closures
		resolveTopic := func(tag string) string {
			res, err := s.graphService.ResolveScope([]string{tag})
			if err == nil && len(res) > 0 {
				return res[0]
			}
			return ""
		}

		getTutorials := func(urls []string) []string {
			urlMap, err := s.repo.GetTutorialIDsByURLs(ctx, urls)
			if err != nil {
				return []string{}
			}
			var ids []string
			for _, url := range urls {
				if id, ok := urlMap[url]; ok {
					ids = append(ids, id)
				}
			}
			return ids
		}

		_, err := s.repo.CreateRoadmapTemplate(ctx, &t, resolveTopic, getTutorials)
		if err != nil {
			errors = append(errors, fmt.Errorf("record %d (%s): %w", i, t.RoadmapName, err))
		}
	}
	return errors
}
