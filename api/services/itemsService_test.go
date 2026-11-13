package services

import (
	"API/api/dto"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func seedItemsServiceCatalog(t *testing.T) []dto.Item {
	t.Helper()
	// Keep row insertion order distinct from ID order so the ordering assertion
	// also catches removal of the service's explicit ORDER BY.
	newServiceTestDB(t,
		`CREATE TABLE items (
			id INTEGER UNIQUE, name TEXT, description TEXT, type TEXT,
			icon_url TEXT, is_tradeable BOOLEAN)`,
		`INSERT INTO items (id, name, description, type, icon_url, is_tradeable) VALUES
			(8, 'Stone', NULL, 'resource', NULL, 0),
			(3, 'Wood', 'Building material', 'resource', '/images/wood.png', 1)`,
	)
	description, iconURL := "Building material", "/images/wood.png"
	return []dto.Item{
		{ID: 3, Name: "Wood", Description: &description, Type: "resource", IconURL: &iconURL, IsTradeable: true},
		{ID: 8, Name: "Stone", Type: "resource", IsTradeable: false},
	}
}

func TestGetAllItemsLoadsOrderedCatalogAndCachesResult(t *testing.T) {
	for _, mode := range []string{"cache miss", "corrupt cache", "cache read error"} {
		t.Run(mode, func(t *testing.T) {
			cache := newServiceTestCache(t)
			want := seedItemsServiceCatalog(t)
			switch mode {
			case "corrupt cache":
				cache.Set("v2:items:all", `{"invalid":`)
			case "cache read error":
				cache.Set("v2:items:all", `[]`)
				cache.getErr = errors.New("cache unavailable")
			}

			got, err := GetAllItems(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("items = %#v, want %#v", got, want)
			}
			if !reflect.DeepEqual(cache.gets, []string{"v2:items:all"}) || !reflect.DeepEqual(cache.sets, []string{"v2:items:all"}) {
				t.Fatalf("cache calls: GET %v, SET %v", cache.gets, cache.sets)
			}
			if ttl := cache.ttls["v2:items:all"]; ttl != 10*time.Minute {
				t.Fatalf("cache TTL = %s, want 10m", ttl)
			}
			var cached []dto.Item
			if err := json.Unmarshal([]byte(cache.values["v2:items:all"]), &cached); err != nil {
				t.Fatalf("invalid cached JSON: %v", err)
			}
			if !reflect.DeepEqual(cached, want) {
				t.Fatalf("cached items = %#v, want %#v", cached, want)
			}
		})
	}
}

func TestGetAllItemsReturnsCachedCatalogWithoutDatabase(t *testing.T) {
	cache := newServiceTestCache(t)
	cache.Set("v2:items:all", `[{
		"id":7,"name":"Iron","description":"For tools","type":"resource",
		"icon_url":"/images/iron.png","is_tradeable":true
	}]`)
	newServiceTestDB(t)
	description, iconURL := "For tools", "/images/iron.png"
	want := []dto.Item{{ID: 7, Name: "Iron", Description: &description, Type: "resource", IconURL: &iconURL, IsTradeable: true}}

	got, err := GetAllItems(context.Background())
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("items = %#v, error = %v, want %#v", got, err, want)
	}
	if !reflect.DeepEqual(cache.gets, []string{"v2:items:all"}) || len(cache.sets) != 0 {
		t.Fatalf("cache hit should only read: GET %v, SET %v", cache.gets, cache.sets)
	}
}

func TestGetAllItemsReturnsEmptyNonNilCatalog(t *testing.T) {
	cache := newServiceTestCache(t)
	newServiceTestDB(t, `CREATE TABLE items (id INTEGER PRIMARY KEY)`)

	got, err := GetAllItems(context.Background())
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("items = %#v, error = %v, want a non-nil empty catalog", got, err)
	}
	if cache.values["v2:items:all"] != `[]` {
		t.Fatalf("cached empty catalog = %q, want []", cache.values["v2:items:all"])
	}
}

func TestGetAllItemsIgnoresCacheWriteFailure(t *testing.T) {
	cache := newServiceTestCache(t)
	cache.setErr = errors.New("cache is read-only")
	want := seedItemsServiceCatalog(t)

	got, err := GetAllItems(context.Background())
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("items = %#v, error = %v, want %#v despite cache error", got, err, want)
	}
	if !reflect.DeepEqual(cache.sets, []string{"v2:items:all"}) {
		t.Fatalf("cache writes = %v, want one attempted catalog write", cache.sets)
	}
}

func TestGetAllItemsReturnsDatabaseErrorWithoutCaching(t *testing.T) {
	cache := newServiceTestCache(t)
	newServiceTestDB(t)

	got, err := GetAllItems(context.Background())
	if err == nil || err.Error() != "failed to fetch items" || got != nil {
		t.Fatalf("items = %#v, error = %v, want failed to fetch items", got, err)
	}
	if len(cache.sets) != 0 {
		t.Fatalf("failed database query populated cache: %v", cache.sets)
	}
}
