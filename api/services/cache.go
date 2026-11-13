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
	// TestCachedDTOShapesMatchKeyVersion fails until it is bumped.
	cacheKeyVersion = "v2"

	// cacheOpTimeout bounds each Redis call so a slow or unreachable cache
	// degrades to a database read instead of stalling the request.
	cacheOpTimeout = 500 * time.Millisecond

	// cacheLoadTimeout bounds a shared load. It is detached from the request
	// that started it, so without a limit a hung query would also hang every
	// later request for the same key.
	cacheLoadTimeout = 10 * time.Second
)

var cacheLoads singleflight.Group

// startCacheWrite runs a Redis write off the response path. Tests replace it
// to make writes synchronous.
var startCacheWrite = func(write func()) { go write() }

func versionedCacheKey(key string) string {
	return cacheKeyVersion + ":" + key
}

// cached returns the value stored under key, or calls load, stores the result
// for ttl, and returns it. Concurrent misses for the same key share one load,
// but every caller gets its own decoded copy, so callers may modify the result.
// Cache failures are logged and never fail the request; load errors are
// returned and not cached.
func cached[T any](ctx context.Context, key string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	var zero T
	key = versionedCacheKey(key)

	if value, ok := readCache[T](ctx, key); ok {
		return value, nil
	}

	// The shared load outlives any single caller, so it must not be cancelled
	// when the request that started it goes away.
	result, err, _ := cacheLoads.Do(key, func() (any, error) {
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheLoadTimeout)
		defer cancel()

		value, err := load(loadCtx)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		writeCtx, rdb := context.WithoutCancel(ctx), redisClient.RDB
		startCacheWrite(func() { writeCache(writeCtx, rdb, key, data, ttl) })
		return data, nil
	})
	if err != nil {
		return zero, err
	}

	var value T
	if err := json.Unmarshal(result.([]byte), &value); err != nil {
		return zero, err
	}
	return value, nil
}

func readCache[T any](ctx context.Context, key string) (T, bool) {
	var value T

	opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	raw, err := redisClient.RDB.Get(opCtx, key).Bytes()
	if err != nil {
		// A cancelled request is the client leaving, not a cache problem.
		if !errors.Is(err, redis.Nil) && ctx.Err() == nil {
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

func writeCache(ctx context.Context, rdb *redis.Client, key string, data []byte, ttl time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	if err := rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Default().Printf("cache write failed for %s: %v", key, err)
	}
}
