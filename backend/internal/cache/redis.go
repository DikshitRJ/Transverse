package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return ErrCacheMiss
	} else if err != nil {
		return err
	}
	return json.Unmarshal(val, dest)
}

func (r *RedisCache) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, bytes, ttl).Err()
}

func (r *RedisCache) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Client() *redis.Client {
	return r.client
}
