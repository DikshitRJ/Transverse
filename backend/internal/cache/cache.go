// Package cache provides caching interfaces and implementations for in-memory and distributed backends.
package cache

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned when a requested key does not exist in the cache or has expired.
var ErrCacheMiss = errors.New("cache miss")

// Cache defines the generic interface for key-value caching operations.
type Cache interface {
	// Get retrieves the value for a key and unmarshals it into dest.
	// Returns ErrCacheMiss if key does not exist or is expired.
	Get(ctx context.Context, key string, dest interface{}) error

	// Set marshals val and stores it under key with the given time-to-live duration.
	Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error

	// Del removes the key from the cache.
	Del(ctx context.Context, key string) error
}
