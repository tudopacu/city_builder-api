package redis

import (
	"API/configuration"
	"context"
	"log"
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
		client := redis.NewClient(&redis.Options{
			Addr:     configuration.RedisAddr,
			Password: configuration.RedisPassword,
			DB:       configuration.RedisDB,
		})

		ctx := context.Background()
		_, err := client.Ping(ctx).Result()
		if err != nil {
			log.Fatalf("failed to connect to redis: %v", err)
		}

		rdb = client
	})

	RDB = rdb
}
