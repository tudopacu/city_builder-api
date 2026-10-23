package controllers

import (
	"API/api/dto"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"
)

func newNewsControllerTestDB(t *testing.T, count int) *gorm.DB {
	t.Helper()
	db := newControllerTestDB(t, `CREATE TABLE news (
		id INTEGER PRIMARY KEY, title TEXT, content TEXT, image_url TEXT,
		created_at DATETIME, updated_at DATETIME)`)
	for id := 1; id <= count; id++ {
		created := time.Date(2025, 1, 1, 0, 0, id, 0, time.UTC)
		if err := db.Exec(`INSERT INTO news (id, title, content, image_url, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`, id, fmt.Sprintf("News %d", id), "Announcement", "/news.png", created, created).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestGetNewsPagination(t *testing.T) {
	for _, tc := range []struct {
		name, query                       string
		count, page, pageSize, totalPages int
		wantIDs                           []uint
	}{
		{"defaults", "", 12, 1, 10, 2, []uint{12, 11, 10, 9, 8, 7, 6, 5, 4, 3}},
		{"explicit page and size", "?page=2&pageSize=2", 5, 2, 2, 3, []uint{3, 2}},
		{"partial last page", "?page=3&pageSize=2", 5, 3, 2, 3, []uint{1}},
		{"size capped at 100", "?pageSize=101", 105, 1, 100, 2, nil},
		{"page beyond results", "?page=4&pageSize=2", 5, 4, 2, 3, []uint{}},
		{"empty database", "", 0, 1, 10, 0, []uint{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			newControllerTestCache(t)
			newNewsControllerTestDB(t, tc.count)
			response := performControllerRequest(http.MethodGet, "/news", "/news"+tc.query, "", GetNews)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var payload struct {
				News       []dto.News `json:"news"`
				Page       int        `json:"page"`
				PageSize   int        `json:"pageSize"`
				TotalCount int        `json:"totalCount"`
				TotalPages int        `json:"totalPages"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Page != tc.page || payload.PageSize != tc.pageSize || payload.TotalCount != tc.count || payload.TotalPages != tc.totalPages {
				t.Fatalf("unexpected pagination metadata: %s", response.Body.String())
			}
			wantIDs := tc.wantIDs
			if wantIDs == nil {
				for id := uint(105); id > 5; id-- {
					wantIDs = append(wantIDs, id)
				}
			}
			gotIDs := make([]uint, 0, len(payload.News))
			for _, news := range payload.News {
				gotIDs = append(gotIDs, news.ID)
				created := time.Date(2025, 1, 1, 0, 0, int(news.ID), 0, time.UTC)
				if news.Title != fmt.Sprintf("News %d", news.ID) || news.Content != "Announcement" || news.ImageUrl != "/news.png" || !news.CreatedAt.Equal(created) || !news.UpdatedAt.Equal(created) {
					t.Fatalf("unexpected news details: %+v", news)
				}
			}
			if !reflect.DeepEqual(gotIDs, wantIDs) {
				t.Fatalf("news IDs = %v, want %v", gotIDs, wantIDs)
			}
		})
	}
}

func TestGetNewsRejectsInvalidPagination(t *testing.T) {
	for _, parameter := range []string{"page", "pageSize"} {
		for _, value := range []string{"abc", "0", "-1", "1.5", "999999999999999999999999999999"} {
			t.Run(parameter+"/"+value, func(t *testing.T) {
				response := performControllerRequest(http.MethodGet, "/news", "/news?"+parameter+"="+value, "", GetNews)
				assertControllerError(t, response, http.StatusBadRequest, "invalid "+parameter+" parameter")
			})
		}
	}
}

func TestGetNewsReturnsDatabaseError(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t)
	response := performControllerRequest(http.MethodGet, "/news", "/news", "", GetNews)
	assertControllerError(t, response, http.StatusInternalServerError, "failed to count news")
}

func TestGetNewsReusesCachedPage(t *testing.T) {
	newControllerTestCache(t)
	db := newNewsControllerTestDB(t, 3)
	first := performControllerRequest(http.MethodGet, "/news", "/news?page=2&pageSize=1", "", GetNews)
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", first.Code, first.Body.String())
	}
	// The same page remains available from cache, while other pages still
	// need the database. This also checks that cached metadata survives.
	execControllerSQL(t, db, `DROP TABLE news`)
	cached := performControllerRequest(http.MethodGet, "/news", "/news?page=2&pageSize=1", "", GetNews)
	assertControllerJSON(t, cached, http.StatusOK, first.Body.String())
	other := performControllerRequest(http.MethodGet, "/news", "/news?page=1&pageSize=1", "", GetNews)
	assertControllerError(t, other, http.StatusInternalServerError, "failed to count news")
}
