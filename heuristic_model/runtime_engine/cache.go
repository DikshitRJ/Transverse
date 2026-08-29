package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

type memoryEntry struct {
	data   []byte
	expiry time.Time
}

type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]memoryEntry
	cleanup time.Duration
}

func NewMemory(cleanup time.Duration) *MemoryCache {
	c := &MemoryCache{
		entries: make(map[string]memoryEntry),
		cleanup: cleanup,
	}
	go c.evictLoop()
	return c
}

func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return fmt.Errorf("cache: key %q not found", key)
	}

	if !entry.expiry.IsZero() && time.Now().After(entry.expiry) {
		c.Del(ctx, key)
		return fmt.Errorf("cache: key %q expired", key)
	}

	return json.Unmarshal(entry.data, dest)
}

func (c *MemoryCache) Set(_ context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}

	var expiry time.Time
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}

	c.mu.Lock()
	c.entries[key] = memoryEntry{data: data, expiry: expiry}
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Del(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) evictLoop() {
	ticker := time.NewTicker(c.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, v := range c.entries {
			if !v.expiry.IsZero() && now.After(v.expiry) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

type NoopCache struct{}

func (NoopCache) Get(_ context.Context, _ string, _ interface{}) error {
	return fmt.Errorf("cache: disabled")
}

func (NoopCache) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) error {
	return nil
}

func (NoopCache) Del(_ context.Context, _ string) error {
	return nil
}
