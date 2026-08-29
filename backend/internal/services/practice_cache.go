// Package services contains domain business logic, psychometric models, and heuristic problem recommendation algorithms.
package services

import "fmt"

// CacheKeyProblem returns the cache key for a specific problem entity.
func CacheKeyProblem(id string) string {
	return fmt.Sprintf("problem:%s", id)
}

// CacheKeySeenIDs returns the cache key for a user's seen problem attempts map.
func CacheKeySeenIDs(userID string) string {
	return fmt.Sprintf("seen:%s", userID)
}

// CacheKeyDNA returns the cache key for a user's learning DNA profile.
func CacheKeyDNA(userID string) string {
	return fmt.Sprintf("dna:%s", userID)
}

// CacheKeyTopicStats returns the cache key for a user's topic mastery statistics.
func CacheKeyTopicStats(userID string) string {
	return fmt.Sprintf("topic_stats:%s", userID)
}
