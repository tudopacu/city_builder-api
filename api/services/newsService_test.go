package services

import (
	"API/api/dto"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"
)

func newNewsServiceTestDB(t *testing.T, count int) *gorm.DB {
	t.Helper()
	db := newServiceTestDB(t, `CREATE TABLE news (
		id INTEGER PRIMARY KEY, title TEXT, content TEXT, image_url TEXT,
		created_at DATETIME, updated_at DATETIME)`)
	for id := 1; id <= count; id++ {
		created := time.Date(2025, 1, id, 0, 0, 0, 0, time.UTC)
		if err := db.Exec(`INSERT INTO news (id, title, content, image_url, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`, id, fmt.Sprintf("News %d", id), "Announcement", "/news.png", created, created.Add(time.Hour)).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func wantServiceNews(ids ...uint) []dto.News {
	news := make([]dto.News, 0, len(ids))
	for _, id := range ids {
		created := time.Date(2025, 1, int(id), 0, 0, 0, 0, time.UTC)
		news = append(news, dto.News{
			ID: id, Title: fmt.Sprintf("News %d", id), Content: "Announcement", ImageUrl: "/news.png",
			CreatedAt: created, UpdatedAt: created.Add(time.Hour),
		})
	}
	return news
}

func TestGetNewsPaginatesNewestFirstAndCachesEachPageAndSize(t *testing.T) {
	cache := newServiceTestCache(t)
	db := newNewsServiceTestDB(t, 5)
	cases := []struct {
		page, size int
		key        string
		want       []dto.News
	}{
		{1, 2, "news:page:1:size:2", wantServiceNews(5, 4)},
		{2, 2, "news:page:2:size:2", wantServiceNews(3, 2)},
		{1, 3, "news:page:1:size:3", wantServiceNews(5, 4, 3)},
		{3, 2, "news:page:3:size:2", wantServiceNews(1)},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			got, count, err := GetNews(tc.page, tc.size)
			if err != nil || count != 5 || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("news = %+v, count = %d, error = %v; want %+v and total 5", got, count, err, tc.want)
			}
			if cache.ttls[tc.key] != 5*time.Minute {
				t.Fatalf("cache TTL = %s, want 5m", cache.ttls[tc.key])
			}
			var stored newsCacheEntry
			if err := json.Unmarshal([]byte(cache.values[tc.key]), &stored); err != nil {
				t.Fatal(err)
			}
			if stored.TotalCount != 5 || !reflect.DeepEqual(stored.News, tc.want) {
				t.Fatalf("cached page = %+v; want news and unpaginated total", stored)
			}
		})
	}
	// All pages must survive a database outage using their own cached content.
	execServiceTestSQL(t, db, `DROP TABLE news`)
	for _, tc := range cases {
		got, count, err := GetNews(tc.page, tc.size)
		if err != nil || count != 5 || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("cached %s = %+v, count = %d, error = %v", tc.key, got, count, err)
		}
	}
	if len(cache.sets) != len(cases) {
		t.Fatalf("cache hits rewrote entries: %v", cache.sets)
	}
}

func TestGetNewsHandlesEmptyPages(t *testing.T) {
	for _, tc := range []struct {
		name       string
		rows, page int
	}{
		{"empty database", 0, 1},
		{"beyond last page", 5, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			newServiceTestCache(t)
			newNewsServiceTestDB(t, tc.rows)
			news, count, err := GetNews(tc.page, 2)
			if err != nil || len(news) != 0 || count != int64(tc.rows) {
				t.Fatalf("news = %+v, count = %d, error = %v; want empty page, total %d", news, count, err, tc.rows)
			}
		})
	}
}

func TestGetNewsFallsBackOnCacheProblems(t *testing.T) {
	for _, failure := range []string{"corrupt entry", "read failure", "write failure"} {
		t.Run(failure, func(t *testing.T) {
			cache := newServiceTestCache(t)
			newNewsServiceTestDB(t, 3)
			const key = "news:page:1:size:2"
			switch failure {
			case "corrupt entry":
				cache.Set(key, `{invalid`)
			case "read failure":
				cache.getErr = errors.New("cache read unavailable")
			case "write failure":
				cache.setErr = errors.New("cache write unavailable")
			}
			news, count, err := GetNews(1, 2)
			if err != nil || count != 3 || !reflect.DeepEqual(news, wantServiceNews(3, 2)) {
				t.Fatalf("news = %+v, count = %d, error = %v", news, count, err)
			}
			if !reflect.DeepEqual(cache.gets, []string{key}) || !reflect.DeepEqual(cache.sets, []string{key}) {
				t.Fatalf("cache gets = %v, sets = %v", cache.gets, cache.sets)
			}
			if failure != "write failure" && !json.Valid([]byte(cache.values[key])) {
				t.Fatalf("cache was not refreshed with JSON: %q", cache.values[key])
			}
		})
	}
}

func TestGetNewsReturnsDatabaseFailuresWithoutCaching(t *testing.T) {
	for _, tc := range []struct {
		name, schema, want string
	}{
		{"count failure", "", "failed to count news"},
		{"fetch failure", `CREATE TABLE news (id INTEGER PRIMARY KEY)`, "failed to fetch news"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache := newServiceTestCache(t)
			db := newServiceTestDB(t)
			if tc.schema != "" {
				execServiceTestSQL(t, db, tc.schema)
			}
			news, count, err := GetNews(1, 2)
			if news != nil || count != 0 || err == nil || err.Error() != tc.want {
				t.Fatalf("news = %+v, count = %d, error = %v; want nil, 0, %q", news, count, err, tc.want)
			}
			if len(cache.sets) != 0 {
				t.Fatalf("failed query wrote cache: %v", cache.sets)
			}
		})
	}
}
