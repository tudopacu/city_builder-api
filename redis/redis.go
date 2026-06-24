package redis

import (
	"API/configuration"
	"context"
	"log"
	"strconv"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	rdb  *redis.Client
	once sync.Once
)

var RDB *redis.Client

func InitRedis() {
	once.Do(func() {
		db, err := strconv.Atoi(configuration.MustGetEnv("REDIS_DATABASE"))
		if err != nil {
			log.Fatalf("failed to get redis db: %v", err)
		}

		client := redis.NewClient(&redis.Options{
			Addr:     configuration.MustGetEnv("REDIS_HOST") + ":" + configuration.MustGetEnv("REDIS_PORT"),
			Password: configuration.MustGetEnv("REDIS_PASSWORD"),
			DB:       db,
		})

		ctx := context.Background()
		_, err = client.Ping(ctx).Result()
		if err != nil {
			log.Fatalf("failed to connect to redis: %v", err)
		}

		rdb = client
	})

	RDB = rdb
}
