package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// entry holds the raw serialized byte payload and expiration timestamp for a cached item.
type entry struct {
	value     []byte
	expiresAt time.Time
}

// MemoryCache is a thread-safe in-memory cache implementation supporting TTL expiration
// and periodic background garbage collection of stale keys.
type MemoryCache struct {
	mu          sync.RWMutex
	items       map[string]entry
	stopCleanup chan struct{}
}

// NewMemoryCache instantiates a new MemoryCache and starts a background goroutine
// that sweeps expired entries every 5 minutes.
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		items:       make(map[string]entry),
		stopCleanup: make(chan struct{}),
	}

	go mc.cleanupLoop(5 * time.Minute)

	return mc
}

// Get retrieves the serialized data for key, verifies TTL expiration,
// and deserializes JSON into dest. Returns ErrCacheMiss if not found or expired.
func (mc *MemoryCache) Get(_ context.Context, key string, dest interface{}) error {
	mc.mu.RLock()
	item, found := mc.items[key]
	mc.mu.RUnlock()

	if !found {
		return ErrCacheMiss
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		// Lazily clean up the expired entry
		mc.mu.Lock()
		delete(mc.items, key)
		mc.mu.Unlock()
		return ErrCacheMiss
	}

	if err := json.Unmarshal(item.value, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached JSON for key %q: %w", key, err)
	}

	return nil
}

// Set serializes val to JSON and saves it in the in-memory map under key with the given TTL.
// If ttl <= 0, the item will not expire automatically.
func (mc *MemoryCache) Set(_ context.Context, key string, val interface{}, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal value for cache key %q: %w", key, err)
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	mc.mu.Lock()
	mc.items[key] = entry{
		value:     data,
		expiresAt: expiresAt,
	}
	mc.mu.Unlock()

	return nil
}

// Del immediately removes key from the in-memory cache.
func (mc *MemoryCache) Del(_ context.Context, key string) error {
	mc.mu.Lock()
	delete(mc.items, key)
	mc.mu.Unlock()
	return nil
}

// Close terminates the background cleanup goroutine.
func (mc *MemoryCache) Close() {
	select {
	case <-mc.stopCleanup:
		// already closed
	default:
		close(mc.stopCleanup)
	}
}

// cleanupLoop runs on a ticker interval to purge expired cache entries.
func (mc *MemoryCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.stopCleanup:
			return
		case now := <-ticker.C:
			mc.mu.Lock()
			for k, v := range mc.items {
				if !v.expiresAt.IsZero() && now.After(v.expiresAt) {
					delete(mc.items, k)
				}
			}
			mc.mu.Unlock()
		}
	}
}

// NoopCache implements Cache interface doing nothing (always returns ErrCacheMiss).
type NoopCache struct{}

// NewNoopCache constructs a no-op cache instance.
func NewNoopCache() *NoopCache {
	return &NoopCache{}
}

// Get always returns ErrCacheMiss.
func (n *NoopCache) Get(_ context.Context, _ string, _ interface{}) error {
	return ErrCacheMiss
}

// Set performs a no-op and returns nil.
func (n *NoopCache) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) error {
	return nil
}

// Del performs a no-op and returns nil.
func (n *NoopCache) Del(_ context.Context, _ string) error {
	return nil
}
