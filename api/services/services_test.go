package services

import (
	"API/database"
	redisClient "API/redis"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These fixtures replace package globals; callers must not use t.Parallel.
func newServiceTestDB(t *testing.T, statements ...string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previousDB
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	execServiceTestSQL(t, db, statements...)
	return db
}

func execServiceTestSQL(t *testing.T, db *gorm.DB, statements ...string) {
	t.Helper()
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("fixture SQL failed: %v\n%s", err, statement)
		}
	}
}

// This hook substitutes only the Redis boundary. Tests run the real services
// and database queries, while controlling cache hits, misses, and failures.
// TTLs record requested expiration; the fixture does not emulate Redis clocks.
type serviceTestCache struct {
	mu     sync.Mutex
	values map[string]string
	ttls   map[string]time.Duration
	getErr error
	setErr error
	gets   []string
	sets   []string
}

func newServiceTestCache(t *testing.T) *serviceTestCache {
	t.Helper()
	cache := &serviceTestCache{
		values: make(map[string]string),
		ttls:   make(map[string]time.Duration),
	}
	client := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	client.AddHook(cache)
	previousClient := redisClient.RDB
	redisClient.RDB = client
	t.Cleanup(func() {
		redisClient.RDB = previousClient
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	})
	return cache
}

func (c *serviceTestCache) Set(key, value string) {
	c.values[key] = value
}

func (c *serviceTestCache) DialHook(redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, fmt.Errorf("service cache must not open network connections")
	}
}

func (c *serviceTestCache) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		args := cmd.Args()
		switch cmd.Name() {
		case "get":
			key := args[1].(string)
			c.gets = append(c.gets, key)
			if c.getErr != nil {
				cmd.SetErr(c.getErr)
				return c.getErr
			}
			value, ok := c.values[key]
			if !ok {
				cmd.SetErr(redis.Nil)
				return redis.Nil
			}
			cmd.(*redis.StringCmd).SetVal(value)
			return nil
		case "set":
			key := args[1].(string)
			c.sets = append(c.sets, key)
			if c.setErr != nil {
				cmd.SetErr(c.setErr)
				return c.setErr
			}
			var value string
			switch raw := args[2].(type) {
			case string:
				value = raw
			case []byte:
				value = string(raw)
			default:
				err := fmt.Errorf("unsupported service cache value: %T", raw)
				cmd.SetErr(err)
				return err
			}
			for i := 3; i+1 < len(args); i += 2 {
				amount, err := strconv.ParseInt(fmt.Sprint(args[i+1]), 10, 64)
				if err != nil {
					cmd.SetErr(err)
					return err
				}
				switch strings.ToLower(fmt.Sprint(args[i])) {
				case "ex":
					c.ttls[key] = time.Duration(amount) * time.Second
				case "px":
					c.ttls[key] = time.Duration(amount) * time.Millisecond
				default:
					err := fmt.Errorf("unsupported service cache expiration: %v", args[i])
					cmd.SetErr(err)
					return err
				}
			}
			c.Set(key, value)
			cmd.(*redis.StatusCmd).SetVal("OK")
			return nil
		default:
			err := fmt.Errorf("unsupported service cache command: %s", cmd.Name())
			cmd.SetErr(err)
			return err
		}
	}
}

func (c *serviceTestCache) ProcessPipelineHook(redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error {
		return fmt.Errorf("service cache does not support pipelines")
	}
}
