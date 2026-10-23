package services

import (
	"API/api/dto"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"
)

func seedBuildingsServiceCatalog(t *testing.T) (*gorm.DB, []dto.BuildingWithDetails) {
	t.Helper()
	db := newServiceTestDB(t,
		`CREATE TABLE buildings (
			id INTEGER PRIMARY KEY, name TEXT, image_url TEXT, description TEXT,
			width INTEGER, length INTEGER, building_category_id INTEGER)`,
		`CREATE TABLE building_categories (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE building_construction_costs (
			id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER, quantity INTEGER)`,
		`CREATE TABLE building_productions (
			id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER,
			quantity INTEGER, production_time_seconds INTEGER)`,
		`INSERT INTO building_categories (id, name) VALUES (2, 'Production'), (3, 'Storage')`,
		`INSERT INTO buildings (id, name, image_url, description, width, length, building_category_id) VALUES
			(5, 'Sawmill', '/images/sawmill.png', 'Cuts timber', 3, 4, 2),
			(6, 'Warehouse', NULL, 'Stores goods', 2, 3, 3)`,
		`INSERT INTO items (id, name) VALUES (7, 'Stone'), (8, 'Wood')`,
		`INSERT INTO building_construction_costs (id, building_id, item_id, quantity) VALUES
			(11, 5, 7, 20), (12, 6, 8, 10)`,
		`INSERT INTO building_productions (id, building_id, item_id, quantity, production_time_seconds)
			VALUES (13, 5, 8, 6, 120)`,
	)
	imageURL := "/images/sawmill.png"
	want := []dto.BuildingWithDetails{
		{
			ID: 5, Name: "Sawmill", ImageURL: &imageURL, Description: "Cuts timber", Width: 3, Length: 4,
			BuildingCategory: "Production",
			Costs:            []dto.BuildingCost{{ItemID: 7, ItemName: "Stone", Quantity: 20}},
			Productions: []dto.BuildingProductionItem{{
				BuildingProductionID: 13, ItemID: 8, ItemName: "Wood", Quantity: 6, ProductionTimeSeconds: 120,
			}},
		},
		{
			ID: 6, Name: "Warehouse", Description: "Stores goods", Width: 2, Length: 3,
			BuildingCategory: "Storage", Costs: []dto.BuildingCost{{ItemID: 8, ItemName: "Wood", Quantity: 10}},
		},
	}
	return db, want
}

func TestGetAllBuildingsLoadsAssociationsAndCachesResult(t *testing.T) {
	for _, mode := range []string{"cache miss", "corrupt cache", "cache read error"} {
		t.Run(mode, func(t *testing.T) {
			cache := newServiceTestCache(t)
			_, want := seedBuildingsServiceCatalog(t)
			switch mode {
			case "corrupt cache":
				cache.Set("buildings:all", `{"invalid":`)
			case "cache read error":
				cache.Set("buildings:all", `[]`)
				cache.getErr = errors.New("cache unavailable")
			}

			got, err := GetAllBuildings()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("buildings = %#v, want %#v", got, want)
			}
			if !reflect.DeepEqual(cache.gets, []string{"buildings:all"}) || !reflect.DeepEqual(cache.sets, []string{"buildings:all"}) {
				t.Fatalf("cache calls: GET %v, SET %v", cache.gets, cache.sets)
			}
			if ttl := cache.ttls["buildings:all"]; ttl != 10*time.Minute {
				t.Fatalf("cache TTL = %s, want 10m", ttl)
			}
			var cached []dto.BuildingWithDetails
			if err := json.Unmarshal([]byte(cache.values["buildings:all"]), &cached); err != nil {
				t.Fatalf("invalid cached JSON: %v", err)
			}
			if !reflect.DeepEqual(cached, want) {
				t.Fatalf("cached buildings = %#v, want %#v", cached, want)
			}
		})
	}
}

func TestGetAllBuildingsReturnsCachedCatalogWithoutDatabase(t *testing.T) {
	cache := newServiceTestCache(t)
	cache.Set("buildings:all", `[{
		"id":5,"name":"Sawmill","image_url":"/images/sawmill.png","description":"Cuts timber",
		"width":3,"length":4,"building_category":"Production",
		"costs":[{"item_id":7,"item_name":"Stone","quantity":20}],
		"productions":[{"building_production_id":13,"item_id":8,"item_name":"Wood","quantity":6,"production_time_seconds":120}]
	}]`)
	newServiceTestDB(t)
	imageURL := "/images/sawmill.png"
	want := []dto.BuildingWithDetails{{
		ID: 5, Name: "Sawmill", ImageURL: &imageURL, Description: "Cuts timber", Width: 3, Length: 4,
		BuildingCategory: "Production", Costs: []dto.BuildingCost{{ItemID: 7, ItemName: "Stone", Quantity: 20}},
		Productions: []dto.BuildingProductionItem{{
			BuildingProductionID: 13, ItemID: 8, ItemName: "Wood", Quantity: 6, ProductionTimeSeconds: 120,
		}},
	}}

	got, err := GetAllBuildings()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("buildings = %#v, error = %v, want %#v", got, err, want)
	}
	if !reflect.DeepEqual(cache.gets, []string{"buildings:all"}) || len(cache.sets) != 0 {
		t.Fatalf("cache hit should only read: GET %v, SET %v", cache.gets, cache.sets)
	}
}

func TestGetAllBuildingsReturnsEmptyCatalog(t *testing.T) {
	cache := newServiceTestCache(t)
	newServiceTestDB(t, `CREATE TABLE buildings (id INTEGER PRIMARY KEY)`)

	got, err := GetAllBuildings()
	if err != nil || len(got) != 0 {
		t.Fatalf("buildings = %#v, error = %v, want an empty catalog", got, err)
	}
	var cached []dto.BuildingWithDetails
	if err := json.Unmarshal([]byte(cache.values["buildings:all"]), &cached); err != nil || len(cached) != 0 {
		t.Fatalf("cached empty catalog = %q, error = %v", cache.values["buildings:all"], err)
	}
}

func TestGetAllBuildingsIgnoresCacheWriteFailure(t *testing.T) {
	cache := newServiceTestCache(t)
	cache.setErr = errors.New("cache is read-only")
	_, want := seedBuildingsServiceCatalog(t)

	got, err := GetAllBuildings()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("buildings = %#v, error = %v, want %#v despite cache error", got, err, want)
	}
	if !reflect.DeepEqual(cache.sets, []string{"buildings:all"}) {
		t.Fatalf("cache writes = %v, want one attempted catalog write", cache.sets)
	}
}

func TestGetAllBuildingsReturnsDatabaseErrorsWithoutCaching(t *testing.T) {
	for _, table := range []string{"buildings", "building_categories", "building_construction_costs", "building_productions", "items"} {
		t.Run(table, func(t *testing.T) {
			cache := newServiceTestCache(t)
			db, _ := seedBuildingsServiceCatalog(t)
			execServiceTestSQL(t, db, "DROP TABLE "+table)

			got, err := GetAllBuildings()
			if err == nil || err.Error() != "failed to fetch buildings" || got != nil {
				t.Fatalf("buildings = %#v, error = %v, want failed to fetch buildings", got, err)
			}
			if len(cache.sets) != 0 {
				t.Fatalf("failed database query populated cache: %v", cache.sets)
			}
		})
	}
}
