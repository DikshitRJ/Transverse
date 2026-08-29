package models

type TutorialIngestRecord struct {
	Source           string   `json:"source"`
	SourceURL        string   `json:"source_url"`
	Title            string   `json:"title"`
	TopicTags        []string `json:"topic_tags"`
	Type             string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	Summary          string   `json:"summary"`
	Author           string   `json:"author,omitempty"`
	ThumbnailURL     string   `json:"thumbnail_url,omitempty"`
	LicenseNote      string   `json:"license_note,omitempty"`
}

type RoadmapTemplateIngestRecord struct {
	RoadmapName string                  `json:"roadmap_name"`
	TargetRole  string                  `json:"target_role"`
	Phases      []RoadmapPhaseIngestDTO `json:"phases"`
}

type RoadmapPhaseIngestDTO struct {
	Title      string                 `json:"title"`
	Sequence   int                    `json:"sequence"`
	UnlockRule map[string]interface{} `json:"unlock_rule"`
	Nodes      []RoadmapNodeIngestDTO `json:"nodes"`
}

type RoadmapNodeIngestDTO struct {
	TopicTag           string   `json:"topic_tag"`
	Sequence           int      `json:"sequence"`
	TutorialSourceURLs []string `json:"tutorial_source_urls"`
	PracticeTopicTags  []string `json:"practice_topic_tags"`
}
