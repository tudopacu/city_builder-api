package controllers

import (
	"API/authentication"
	"API/database"
	redisClient "API/redis"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Tests that replace the package-level database or cache must run serially.
func newControllerTestDB(t *testing.T, statements ...string) *gorm.DB {
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
	execControllerSQL(t, db, statements...)
	return db
}

func execControllerSQL(t *testing.T, db *gorm.DB, statements ...string) {
	t.Helper()
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("fixture SQL failed: %v\n%s", err, statement)
		}
	}
}

func performControllerRequest(method, route, target, body string, handler gin.HandlerFunc, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	router := gin.New()
	router.Handle(method, route, handler)
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func useAuthenticationTestSecret(t *testing.T) {
	t.Helper()
	previous := authentication.JwtSecret
	authentication.JwtSecret = []byte("controller-test-secret-0123456789abcdef")
	t.Cleanup(func() { authentication.JwtSecret = previous })
}

// performPlayerRequest sends the request as playerID through the real
// AuthMiddleware, because player-scoped handlers take the player from there.
func performPlayerRequest(t *testing.T, playerID uint, method, route, target, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	useAuthenticationTestSecret(t)
	cookie := authenticationTestCookie(t, strconv.FormatUint(uint64(playerID), 10), time.Now().Add(time.Hour), authentication.JwtSecret)
	router := gin.New()
	router.Handle(method, route, (&authentication.Server{}).AuthMiddleware(), handler)
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func assertControllerJSON(t *testing.T, response *httptest.ResponseRecorder, status int, want string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	var gotJSON, wantJSON any
	if err := json.Unmarshal(response.Body.Bytes(), &gotJSON); err != nil {
		t.Fatalf("invalid response JSON: %v; body = %s", err, response.Body.String())
	}
	if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Fatalf("invalid expected JSON: %v", err)
	}
	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Fatalf("body = %s, want %s", response.Body.String(), want)
	}
}

func assertControllerError(t *testing.T, response *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	want, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		t.Fatal(err)
	}
	assertControllerJSON(t, response, status, string(want))
}

// The cache hook handles only the GET/SET operations used by these services.
// It bypasses the network, keeping controller tests independent of Redis.
// Expiration is deliberately outside the scope of these HTTP handler tests.
type controllerTestCache struct {
	values map[string]string
}

func newControllerTestCache(t *testing.T) *controllerTestCache {
	t.Helper()
	cache := &controllerTestCache{values: make(map[string]string)}
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

func (c *controllerTestCache) Set(key, value string) {
	c.values[key] = value
}

func (c *controllerTestCache) DialHook(redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, fmt.Errorf("controller cache must not open network connections")
	}
}

func (c *controllerTestCache) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		switch cmd.Name() {
		case "get":
			value, ok := c.values[cmd.Args()[1].(string)]
			if !ok {
				cmd.SetErr(redis.Nil)
				return redis.Nil
			}
			cmd.(*redis.StringCmd).SetVal(value)
			return nil
		case "set":
			var value string
			switch raw := cmd.Args()[2].(type) {
			case string:
				value = raw
			case []byte:
				value = string(raw)
			default:
				err := fmt.Errorf("unsupported controller cache value: %T", raw)
				cmd.SetErr(err)
				return err
			}
			c.Set(cmd.Args()[1].(string), value)
			cmd.(*redis.StatusCmd).SetVal("OK")
			return nil
		default:
			err := fmt.Errorf("unsupported controller cache command: %s", cmd.Name())
			cmd.SetErr(err)
			return err
		}
	}
}

func (c *controllerTestCache) ProcessPipelineHook(redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error {
		return fmt.Errorf("controller cache does not support pipelines")
	}
}
