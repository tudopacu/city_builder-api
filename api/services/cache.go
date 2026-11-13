package services

import (
	redisClient "API/redis"
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	// cacheKeyVersion prefixes every cache key. Bump it whenever a cached DTO
	// changes shape, so entries written by the previous release are ignored
	// instead of being decoded with missing fields left at their zero values.
	cacheKeyVersion = "v2"

	// cacheOpTimeout bounds each Redis call so a slow or unreachable cache
	// degrades to a database read instead of stalling the request.
	cacheOpTimeout = 500 * time.Millisecond
)

var cacheLoads singleflight.Group

func versionedCacheKey(key string) string {
	return cacheKeyVersion + ":" + key
}

// cached returns the value stored under key, or calls load, stores the result
// for ttl, and returns it. Concurrent misses for the same key share one load.
// Cache failures are logged and never fail the request; load errors are
// returned and not cached.
func cached[T any](ctx context.Context, key string, ttl time.Duration, load func() (T, error)) (T, error) {
	key = versionedCacheKey(key)

	if value, ok := readCache[T](ctx, key); ok {
		return value, nil
	}

	// The shared load outlives any single caller, so it must not be cancelled
	// when the request that started it goes away.
	loadCtx := context.WithoutCancel(ctx)
	result, err, _ := cacheLoads.Do(key, func() (any, error) {
		value, err := load()
		if err != nil {
			return value, err
		}
		writeCache(loadCtx, key, value, ttl)
		return value, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return result.(T), nil
}

func readCache[T any](ctx context.Context, key string) (T, bool) {
	var value T

	ctx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	raw, err := redisClient.RDB.Get(ctx, key).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Default().Printf("cache read failed for %s: %v", key, err)
		}
		return value, false
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		log.Default().Printf("cache entry for %s is invalid: %v", key, err)
		return value, false
	}
	return value, true
}

func writeCache(ctx context.Context, key string, value any, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Default().Printf("failed to encode cache entry for %s: %v", key, err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	if err := redisClient.RDB.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Default().Printf("cache write failed for %s: %v", key, err)
	}
}
